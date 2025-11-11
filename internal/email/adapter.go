package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/logger"
)

// validTemplateNameRegex ensures template names contain only safe characters
var validTemplateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

// SendRequest represents the expected structure for email adapter
// This mirrors notify.SendRequest to avoid import cycles
type SendRequest struct {
	To       string
	Template string
	Subject  string
	Data     map[string]interface{}
	From     string
}

// Send sends an email notification
// Returns a generated message ID and error
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
	start := time.Now()
	log := logger.FromContext(ctx)

	// Generate message ID upfront
	messageID := uuid.New().String()

	// Extract fields using common adapter logic
	baseReq, err := adapters.ExtractBaseRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract email request")
		return "", fmt.Errorf("email adapter: %w", err)
	}

	maskedTo := logger.MaskEmail(baseReq.To)
	log.Debug().
		Str("channel", "email").
		Str("to", maskedTo).
		Str("template", baseReq.Template).
		Str("message_id", messageID).
		Msg("Processing email notification")

	// Render template
	body, err := a.renderTemplate(baseReq.Template, baseReq.Data)
	if err != nil {
		log.Error().
			Err(err).
			Str("template", baseReq.Template).
			Str("message_id", messageID).
			Msg("Failed to render email template")
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	// Determine sender
	from := a.config.From
	if baseReq.From != "" {
		from = baseReq.From
	}

	// Determine subject
	subject := baseReq.Subject
	if subject == "" {
		subject = "Notification from " + from
	}

	// Send email
	if err := a.sendEmail(from, baseReq.To, subject, body); err != nil {
		log.Error().
			Err(err).
			Str("to", maskedTo).
			Str("smtp_host", a.config.Host).
			Str("message_id", messageID).
			Dur("duration", time.Since(start)).
			Msg("Failed to send email via SMTP")
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	log.Info().
		Str("channel", "email").
		Str("to", maskedTo).
		Str("template", baseReq.Template).
		Str("message_id", messageID).
		Dur("duration", time.Since(start)).
		Msg("Email sent successfully")

	return messageID, nil
}

// renderTemplate renders an HTML template with the given data
func (a *Adapter) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
	// SECURITY: Validate template name to prevent path traversal
	if !validTemplateNameRegex.MatchString(templateName) {
		return "", fmt.Errorf("invalid template name: must contain only letters, numbers, hyphens, and underscores")
	}

	templatePath := filepath.Join(a.templates.Dir, templateName+".html")

	// SECURITY: Verify resolved path is within templates directory
	absTemplatePath, err := filepath.Abs(templatePath)
	if err != nil {
		return "", fmt.Errorf("invalid template path: %w", err)
	}

	absTemplateDir, err := filepath.Abs(a.templates.Dir)
	if err != nil {
		return "", fmt.Errorf("invalid template directory: %w", err)
	}

	// Check if the resolved path starts with the templates directory
	if !strings.HasPrefix(absTemplatePath, absTemplateDir+string(filepath.Separator)) &&
		absTemplatePath != absTemplateDir {
		return "", errors.New("invalid template: path traversal detected")
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed")
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
