package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/huzaifabhutta/notify-core/internal/config"
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
func (a *Adapter) Send(ctx context.Context, req interface{}) error {
	// Extract fields from request using duck typing
	// This works with any struct that has these fields (like notify.SendRequest)
	sendReq, err := extractSendRequest(req)
	if err != nil {
		return err
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

// extractSendRequest extracts SendRequest fields from any compatible struct using reflection
// This allows the email adapter to work with notify.SendRequest without creating an import cycle
func extractSendRequest(req interface{}) (SendRequest, error) {
	// Try direct cast first (for testing with email.SendRequest)
	if r, ok := req.(*SendRequest); ok {
		return *r, nil
	}
	if r, ok := req.(SendRequest); ok {
		return r, nil
	}

	// Use reflection to extract fields from any struct with matching fields
	val := reflect.ValueOf(req)

	// Handle pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return SendRequest{}, fmt.Errorf("nil request pointer")
		}
		val = val.Elem()
	}

	// Must be a struct
	if val.Kind() != reflect.Struct {
		return SendRequest{}, fmt.Errorf("invalid request type: expected struct, got %T", req)
	}

	// Extract fields
	result := SendRequest{}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		switch field.Name {
		case "To":
			if field.Type.Kind() == reflect.String {
				result.To = fieldVal.String()
			}
		case "Template":
			if field.Type.Kind() == reflect.String {
				result.Template = fieldVal.String()
			}
		case "Subject":
			if field.Type.Kind() == reflect.String {
				result.Subject = fieldVal.String()
			}
		case "From":
			if field.Type.Kind() == reflect.String {
				result.From = fieldVal.String()
			}
		case "Data":
			if fieldVal.Type().Kind() == reflect.Map {
				if data, ok := fieldVal.Interface().(map[string]interface{}); ok {
					result.Data = data
				}
			}
		}
	}

	// Validate required fields were found
	if result.To == "" || result.Template == "" {
		return SendRequest{}, fmt.Errorf("invalid request: missing required fields (To or Template)")
	}

	return result, nil
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
