package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pulse/api/internal/config"
)

// SendVerificationEmail sends an email with a verification link.
func SendVerificationEmail(toEmail, toName, token string) error {
	link := fmt.Sprintf("%s/verify-email/%s", config.App.ClientURL, token)
	subject := "Verify your Pulse account"
	body := verificationEmailHTML(toName, link)
	return sendEmail(toEmail, subject, body)
}

// SendPasswordResetEmail sends a password reset link.
func SendPasswordResetEmail(toEmail, toName, token string) error {
	link := fmt.Sprintf("%s/reset-password/%s", config.App.ClientURL, token)
	subject := "Reset your Pulse password"
	body := passwordResetEmailHTML(toName, link)
	return sendEmail(toEmail, subject, body)
}

func sendEmail(to, subject, htmlBody string) error {
	switch {
	case config.App.BrevoAPIKey != "":
		return sendViaBrevo(to, subject, htmlBody)
	case config.App.ResendAPIKey != "":
		return sendViaResend(to, subject, htmlBody)
	default:
		fmt.Printf("\n[DEV EMAIL] To: %s | Subject: %s\n", to, subject)
		return nil
	}
}

func sendViaBrevo(to, subject, htmlBody string) error {
	from := config.App.SMTPFrom
	if from == "" {
		from = "noreply@pulse.app"
	}

	payload, _ := json.Marshal(map[string]any{
		"sender":      map[string]string{"name": "Pulse", "email": from},
		"to":          []map[string]string{{"email": to}},
		"subject":     subject,
		"htmlContent": htmlBody,
	})

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", config.App.BrevoAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func sendViaResend(to, subject, htmlBody string) error {
	from := config.App.SMTPFrom
	if from == "" {
		from = "noreply@pulse.app"
	}

	payload, _ := json.Marshal(map[string]any{
		"from":    fmt.Sprintf("Pulse <%s>", from),
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	})

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.App.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── Email templates ──────────────────────────────────────────

func verificationEmailHTML(name, link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Inter,sans-serif;background:#0f172a;color:#f8fafc;padding:40px;">
  <div style="max-width:480px;margin:0 auto;background:#1e293b;border-radius:12px;padding:40px;">
    <h1 style="color:#2563eb;margin-bottom:8px;">Pulse</h1>
    <h2 style="font-size:20px;margin-bottom:16px;">Verify your email</h2>
    <p style="color:#94a3b8;">Hi %s,</p>
    <p style="color:#94a3b8;">Click the button below to verify your Pulse account.</p>
    <a href="%s" style="display:inline-block;margin:24px 0;padding:12px 28px;background:#2563eb;color:#fff;border-radius:8px;text-decoration:none;font-weight:600;">
      Verify Email
    </a>
    <p style="color:#64748b;font-size:13px;">This link expires in 24 hours. If you didn't create an account, ignore this email.</p>
  </div>
</body>
</html>`, name, link)
}

func passwordResetEmailHTML(name, link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Inter,sans-serif;background:#0f172a;color:#f8fafc;padding:40px;">
  <div style="max-width:480px;margin:0 auto;background:#1e293b;border-radius:12px;padding:40px;">
    <h1 style="color:#2563eb;margin-bottom:8px;">Pulse</h1>
    <h2 style="font-size:20px;margin-bottom:16px;">Reset your password</h2>
    <p style="color:#94a3b8;">Hi %s,</p>
    <p style="color:#94a3b8;">Click the button below to reset your password. This link expires in 1 hour.</p>
    <a href="%s" style="display:inline-block;margin:24px 0;padding:12px 28px;background:#2563eb;color:#fff;border-radius:8px;text-decoration:none;font-weight:600;">
      Reset Password
    </a>
    <p style="color:#64748b;font-size:13px;">If you didn't request a password reset, ignore this email.</p>
  </div>
</body>
</html>`, name, link)
}
