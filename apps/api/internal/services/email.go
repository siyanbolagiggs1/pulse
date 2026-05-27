package services

import (
	"fmt"

	"github.com/pulse/api/internal/config"
	"gopkg.in/gomail.v2"
)

// SendVerificationEmail sends an email with a verification link.
func SendVerificationEmail(toEmail, toName, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", config.App.ClientURL, token)
	subject := "Verify your Pulse account"
	body := verificationEmailHTML(toName, link)
	return sendEmail(toEmail, subject, body)
}

// SendPasswordResetEmail sends a password reset link.
func SendPasswordResetEmail(toEmail, toName, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", config.App.ClientURL, token)
	subject := "Reset your Pulse password"
	body := passwordResetEmailHTML(toName, link)
	return sendEmail(toEmail, subject, body)
}

func sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("Pulse <%s>", config.App.SMTPFrom))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(
		config.App.SMTPHost,
		config.App.SMTPPort,
		config.App.SMTPUser,
		config.App.SMTPPass,
	)

	return d.DialAndSend(m)
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
