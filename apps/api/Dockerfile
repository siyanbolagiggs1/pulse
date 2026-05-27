# ── Build stage ──────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git (needed for some go modules)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pulse-api ./cmd/server

# ── Run stage ────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

# Install CA certs for HTTPS calls (Stripe, etc.) and wget for health checks
RUN apk add --no-cache ca-certificates tzdata wget

# Non-root user
RUN addgroup -S pulse && adduser -S pulse -G pulse

COPY --from=builder /pulse-api .

# Create uploads directory with correct ownership
RUN mkdir -p /app/uploads && chown -R pulse:pulse /app

USER pulse

EXPOSE 5000

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:5000/health || exit 1

CMD ["./pulse-api"]
