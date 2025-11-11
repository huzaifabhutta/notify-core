package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"strings"

	"github.com/huzaifabhutta/notify-core/internal/config"
)

// Adapter handles email notifications via SMTP
type Adapter struct {
	config    *config.SMTPConfig
	templates *config.TemplatesConfig
}

// NewAdapter creates a new email adapter
func NewAdapter(smtpCfg *config.SMTPConfig, templatesCfg *config.TemplatesConfig) *Adapter {
	return &Adapter{
		config:    smtpCfg,
		templates: templatesCfg,
	}
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "email"
}

// SendRequest represents an email send request (local type to avoid import cycle)
type SendRequest struct {
	To       string
	Channel  interface{}
	Template string
	Subject  string
	Data     map[string]interface{}
	From     string
}

// Send sends an email notification
func (a *Adapter) Send(ctx context.Context, req interface{}) error {
	// Type assertion
	sendReq, ok := req.(*SendRequest)
	if !ok {
		return fmt.Errorf("invalid request type for email adapter")
	}

	// Render template
	body, err := a.renderTemplate(sendReq.Template, sendReq.Data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Determine sender
	from := a.config.From
	if sendReq.From != "" {
		from = sendReq.From
	}

	// Determine subject
	subject := sendReq.Subject
	if subject == "" {
		subject = "Notification from " + from
	}

	// Send email
	if err := a.sendEmail(from, sendReq.To, subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// renderTemplate renders an HTML template with the given data
func (a *Adapter) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
	templatePath := filepath.Join(a.templates.Dir, templateName+".html")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		// If template file doesn't exist, use a simple default
		return a.renderSimpleTemplate(data)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderSimpleTemplate creates a simple HTML template when no template file exists
func (a *Adapter) renderSimpleTemplate(data map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("<html><body>")

	for key, value := range data {
		buf.WriteString(fmt.Sprintf("<p><strong>%s:</strong> %v</p>", key, value))
	}

	buf.WriteString("</body></html>")
	return buf.String(), nil
}

// sendEmail sends an email via SMTP
func (a *Adapter) sendEmail(from, to, subject, body string) error {
	// SMTP authentication
	auth := smtp.PlainAuth("", a.config.User, a.config.Password, a.config.Host)

	// Compose email
	msg := a.composeEmail(from, to, subject, body)

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)

	// Send email
	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("smtp error: %w", err)
	}

	return nil
}

// composeEmail composes an email message with headers
func (a *Adapter) composeEmail(from, to, subject, body string) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}
