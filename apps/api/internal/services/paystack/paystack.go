package paystack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://api.paystack.co"

type InitRequest struct {
	Email       string         `json:"email"`
	Amount      int64          `json:"amount"` // smallest unit (kobo for NGN, cents for USD)
	Reference   string         `json:"reference,omitempty"`
	CallbackURL string         `json:"callback_url,omitempty"`
	Currency    string         `json:"currency,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type InitData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

type VerifyData struct {
	Status    string         `json:"status"` // "success", "failed", "abandoned"
	Reference string         `json:"reference"`
	Amount    int64          `json:"amount"`
	Currency  string         `json:"currency"`
	Metadata  map[string]any `json:"metadata"`
}

type paystackResp[T any] struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func Initialize(secretKey string, req InitRequest) (*InitData, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", baseURL+"/transaction/initialize", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result paystackResp[InitData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack: %s", result.Message)
	}
	return &result.Data, nil
}

func Verify(secretKey, reference string) (*VerifyData, error) {
	httpReq, err := http.NewRequest("GET", baseURL+"/transaction/verify/"+reference, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+secretKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result paystackResp[VerifyData]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack: %s", result.Message)
	}
	return &result.Data, nil
}

// ValidateWebhookSignature verifies Paystack's HMAC-SHA512 webhook signature.
func ValidateWebhookSignature(secretKey, signature string, body []byte) bool {
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
