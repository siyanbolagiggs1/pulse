package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "https://api.twitter.com/2"

type UserData struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Username      string `json:"username"`
	PublicMetrics struct {
		FollowersCount int64 `json:"followers_count"`
		FollowingCount int64 `json:"following_count"`
		TweetCount     int64 `json:"tweet_count"`
	} `json:"public_metrics"`
	CreatedAt time.Time `json:"created_at"`
	Verified  bool      `json:"verified"`
}

type userResp struct {
	Data   *UserData `json:"data"`
	Errors []struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
	// Top-level error format (403 forbidden, auth errors)
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
	Reason string `json:"reason"`
}

func GetUserByUsername(bearerToken, username string) (*UserData, error) {
	url := fmt.Sprintf("%s/users/by/username/%s?user.fields=public_metrics,created_at,verified", baseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("[Twitter] status=%d body=%s", resp.StatusCode, string(bodyBytes))

	var result userResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, err
	}

	// Handle top-level errors (403 forbidden, auth errors)
	if resp.StatusCode == http.StatusForbidden || result.Reason == "client-not-enrolled" {
		return nil, fmt.Errorf("Twitter free tier does not support user lookup — upgrade to Basic plan at developer.twitter.com")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid Twitter Bearer Token")
	}

	// Handle errors array
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("%s", result.Errors[0].Detail)
	}

	if result.Data == nil {
		return nil, fmt.Errorf("user @%s not found on Twitter", username)
	}

	return result.Data, nil
}
