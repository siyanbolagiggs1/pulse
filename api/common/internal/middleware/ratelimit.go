package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pulse/api/internal/database"
	"github.com/pulse/api/internal/utils"
	"github.com/redis/go-redis/v9"
)

// slidingWindowScript enforces a sliding-window rate limit atomically in
// Redis, keyed on a sorted set of request timestamps.
//
// A fixed window (INCR + EXPIRE on an hourly bucket) lets a client burst up
// to 2x the limit right at the boundary — e.g. 3 requests at 11:59:59 and 3
// more at 12:00:00 is 6 requests in two seconds, even though the limit is
// "3 per hour". The sorted set holds one member per request, scored by its
// timestamp; every check first evicts members older than the window, then
// only admits the request if what's left is under the limit. That keeps the
// limit accurate no matter when in the window a request lands.
//
// Running the whole check as one Lua script makes "evict old, count, admit"
// atomic — without it, two concurrent requests could both read a count of
// (limit-1) and both be admitted, overshooting the limit under load.
//
// KEYS[1] = rate limit key
// ARGV[1] = now (unix seconds, float)
// ARGV[2] = window size in seconds
// ARGV[3] = max requests allowed in the window
// ARGV[4] = unique member ID for this request (avoids collisions when two
//           requests land in the same millisecond)
//
// Returns {allowed (1/0), remaining-if-allowed or retry-after-seconds-if-not}
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)

if count < limit then
  redis.call('ZADD', key, now, member)
  redis.call('EXPIRE', key, window)
  return {1, limit - count - 1}
end

local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local retryAfter = window
if oldest[2] then
  retryAfter = math.ceil(tonumber(oldest[2]) + window - now)
end
return {0, retryAfter}
`)

// KeyFunc extracts the identity a rate limit is scoped to — e.g. the
// authenticated user, or the caller's IP for routes with no auth yet.
type KeyFunc func(c *gin.Context) string

// PerUser scopes a rate limit to the authenticated user. Must run after
// RequireAuth — reads the same context value it sets.
func PerUser(c *gin.Context) string {
	id, _ := c.Get(ContextUserID)
	s, _ := id.(string)
	return s
}

// PerIP scopes a rate limit to the caller's IP — for routes without auth
// (login, register, forgot-password) where there's no user ID yet to key on.
func PerIP(c *gin.Context) string {
	return c.ClientIP()
}

// RateLimit returns a Gin middleware that allows at most max requests per
// window for a given identity (as resolved by keyFunc), scoped under
// keyPrefix so different routes don't share a counter.
//
// Fails open if Redis is unavailable or errors — consistent with the rest of
// the app's stance that an infra hiccup shouldn't block legitimate traffic —
// but every fail-open is logged so it never fails silently in production.
func RateLimit(max int, window time.Duration, keyPrefix string, keyFunc KeyFunc) gin.HandlerFunc {
	windowSeconds := int64(window.Seconds())

	return func(c *gin.Context) {
		if database.Redis == nil {
			c.Next()
			return
		}
		identity := keyFunc(c)
		if identity == "" {
			// No identity to scope on (e.g. auth middleware didn't run) —
			// nothing sane to key by, so don't block the request over it.
			c.Next()
			return
		}

		member, err := utils.GenerateSecureToken(8)
		if err != nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, identity)
		now := float64(time.Now().UnixNano()) / 1e9

		res, err := slidingWindowScript.Run(c.Request.Context(), database.Redis,
			[]string{key}, now, windowSeconds, max, member).Result()
		if err != nil {
			log.Printf("WARNING: rate limiter Redis error on %s (%v) — failing open", key, err)
			c.Next()
			return
		}

		vals, ok := res.([]interface{})
		if !ok || len(vals) != 2 {
			c.Next()
			return
		}
		allowed, _ := vals[0].(int64)
		second, _ := vals[1].(int64)

		c.Header("X-RateLimit-Limit", strconv.Itoa(max))
		if allowed == 1 {
			c.Header("X-RateLimit-Remaining", strconv.FormatInt(second, 10))
			c.Next()
			return
		}

		c.Header("X-RateLimit-Remaining", "0")
		c.Header("Retry-After", strconv.FormatInt(second, 10))
		utils.Fail(c, http.StatusTooManyRequests,
			fmt.Sprintf("Rate limit exceeded — try again in %d seconds", second))
		c.Abort()
	}
}
