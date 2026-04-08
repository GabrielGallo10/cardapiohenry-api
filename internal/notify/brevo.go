package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// TransactionalEmail — envio transacional (Brevo) ou modo log em desenvolvimento.
type TransactionalEmail struct {
	Provider    string // brevo | log
	APIKey      string
	SenderName  string
	SenderEmail string
}

// SendPasswordResetEmail envia o código de 6 dígitos por e-mail (Brevo API v3).
func SendPasswordResetEmail(cfg TransactionalEmail, toEmail, code string) error {
	subject := "HenryBebidas — código para redefinir senha"
	textBody := fmt.Sprintf(`Olá,

Use o código abaixo para redefinir a sua senha na HenryBebidas:

%s

O código expira em 10 minutos. Se não pediu isto, ignore este e-mail.

— Equipe HenryBebidas`, code)
	htmlBody := fmt.Sprintf(
		`<p>Olá,</p><p>Use o código abaixo para redefinir a sua senha na <strong>HenryBebidas</strong>:</p><p style="font-size:24px;font-weight:bold;letter-spacing:0.25em;font-family:ui-monospace,monospace;">%s</p><p>O código expira em 10 minutos. Se não pediu isto, ignore este e-mail.</p><p>— Equipe HenryBebidas</p>`,
		code,
	)

	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "log", "":
		log.Printf("[email/log] To=%s subject=%s código=%s", toEmail, subject, code)
		return nil
	case "brevo":
		if cfg.APIKey == "" || cfg.SenderEmail == "" {
			return fmt.Errorf("brevo: defina BREVO_API_KEY e BREVO_SENDER_EMAIL (remetente verificado na Brevo)")
		}
		return sendBrevoSMTP(cfg, toEmail, subject, textBody, htmlBody)
	default:
		return fmt.Errorf("EMAIL_PROVIDER inválido: use brevo ou log")
	}
}

type brevoPayload struct {
	Sender      brevoSender `json:"sender"`
	To          []brevoTo   `json:"to"`
	Subject     string      `json:"subject"`
	TextContent string      `json:"textContent"`
	HTMLContent string      `json:"htmlContent"`
}

type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type brevoTo struct {
	Email string `json:"email"`
}

func sendBrevoSMTP(cfg TransactionalEmail, to, subject, text, html string) error {
	name := strings.TrimSpace(cfg.SenderName)
	if name == "" {
		name = "HenryBebidas"
	}
	body, err := json.Marshal(brevoPayload{
		Sender:      brevoSender{Name: name, Email: cfg.SenderEmail},
		To:          []brevoTo{{Email: to}},
		Subject:     subject,
		TextContent: text,
		HTMLContent: html,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.brevo.com/v3/smtp/email",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("brevo HTTP %d: %s", res.StatusCode, string(b))
	}
	return nil
}
