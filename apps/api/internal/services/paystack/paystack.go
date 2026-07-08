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

// ── Payouts (transfers) ──────────────────────────────────────

type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type ResolvedAccount struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
}

type RecipientRequest struct {
	Type          string `json:"type"` // "nuban"
	Name          string `json:"name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Currency      string `json:"currency"`
}

type RecipientData struct {
	RecipientCode string `json:"recipient_code"`
}

type TransferRequest struct {
	Source    string `json:"source"` // "balance"
	Amount    int64  `json:"amount"` // smallest unit
	Recipient string `json:"recipient"`
	Reason    string `json:"reason,omitempty"`
	Reference string `json:"reference,omitempty"`
}

type TransferData struct {
	TransferCode string `json:"transfer_code"`
	Reference    string `json:"reference"`
	Status       string `json:"status"` // "success", "pending", "otp", "failed"
}

func doJSON[T any](method, path, secretKey string, body any) (*T, error) {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}

	httpReq, err := http.NewRequest(method, baseURL+path, buf)
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

	var result paystackResp[T]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack: %s", result.Message)
	}
	return &result.Data, nil
}

// ListBanks returns the banks Paystack supports for the given currency.
func ListBanks(secretKey, currency string) ([]Bank, error) {
	data, err := doJSON[[]Bank]("GET", "/bank?currency="+currency+"&perPage=100", secretKey, nil)
	if err != nil {
		return nil, err
	}
	return *data, nil
}

// ResolveAccount verifies a bank account number is real and returns the
// account holder's name as held by the bank — used so users can't type in
// an arbitrary account and have payouts silently misdirect.
func ResolveAccount(secretKey, accountNumber, bankCode string) (*ResolvedAccount, error) {
	path := fmt.Sprintf("/bank/resolve?account_number=%s&bank_code=%s", accountNumber, bankCode)
	return doJSON[ResolvedAccount]("GET", path, secretKey, nil)
}

// CreateTransferRecipient registers a payout destination with Paystack and
// returns a recipient code that can be reused across future transfers.
func CreateTransferRecipient(secretKey string, req RecipientRequest) (*RecipientData, error) {
	return doJSON[RecipientData]("POST", "/transferrecipient", secretKey, req)
}

// InitiateTransfer sends money from the platform's Paystack balance to a
// previously created recipient. The returned status may be "success",
// "pending" (async — completion arrives via webhook), or "otp" (the
// merchant account requires manual OTP confirmation and cannot be completed
// by this API call).
func InitiateTransfer(secretKey string, req TransferRequest) (*TransferData, error) {
	return doJSON[TransferData]("POST", "/transfer", secretKey, req)
}
