package notify

import (
	"bytes"
	"context"
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
func SendPasswordResetEmail(ctx context.Context, cfg TransactionalEmail, toEmail, code string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	toEmail = strings.TrimSpace(toEmail)
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
		if toEmail == "" {
			return fmt.Errorf("brevo: destinatário em falta")
		}
		return sendBrevoSMTP(ctx, cfg, toEmail, subject, textBody, htmlBody)
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

func sendBrevoSMTP(ctx context.Context, cfg TransactionalEmail, to, subject, text, html string) error {
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
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.brevo.com/v3/smtp/email",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("api-key", cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("[brevo] pedido falhou: %v", err)
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Printf("[brevo] HTTP %d para=%s corpo=%s", res.StatusCode, to, truncateForLog(b, 500))
		return fmt.Errorf("brevo HTTP %d: %s", res.StatusCode, string(b))
	}
	log.Printf("[brevo] e-mail transacional aceite para=%s", to)
	return nil
}

func truncateForLog(b []byte, max int) string {
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
