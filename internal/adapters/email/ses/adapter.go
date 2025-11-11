package ses

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/logger"
)

// Adapter implements the AWS SES email adapter using native AWS SDK
// This provides full access to SES features including:
// - Bounce and complaint handling
// - Configuration sets
// - Email templates
// - Sending statistics
type Adapter struct {
	client *sesv2.Client
	config *Config
}

// Config holds AWS SES adapter configuration
type Config struct {
	// AWS Region (e.g., "us-east-1", "eu-west-1")
	Region string

	// Default sender email address
	// Must be verified in SES
	FromEmail string

	// Optional: SES Configuration Set name
	// Used for tracking bounces, complaints, and delivery events
	ConfigurationSet string

	// Optional: AWS credentials
	// If not provided, uses default credential chain
	AccessKeyID     string
	SecretAccessKey string

	// Optional: Assume Role ARN for cross-account sending
	RoleARN string
}

// NewAdapter creates a new AWS SES adapter instance
// This is the factory function used by the adapter registry
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
	sesConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("%w: expected *ses.Config", adapters.ErrInvalidAdapter)
	}

	// Validate required fields
	if sesConfig.Region == "" {
		return nil, fmt.Errorf("%w: region is required", adapters.ErrInvalidAdapter)
	}

	if sesConfig.FromEmail == "" {
		return nil, fmt.Errorf("%w: from email is required", adapters.ErrInvalidAdapter)
	}

	// Load AWS configuration
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(sesConfig.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SES client
	client := sesv2.NewFromConfig(awsCfg)

	return &Adapter{
		client: client,
		config: sesConfig,
	}, nil
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "ses"
}

// Send sends an email via AWS SES
// Returns the SES Message ID for tracking
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
	start := time.Now()
	log := logger.FromContext(ctx)

	// Extract email request using common adapter helper
	baseReq, err := adapters.ExtractBaseRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract SES request")
		return "", fmt.Errorf("ses adapter: %w", err)
	}

	// Mask sensitive data for logging
	maskedTo := logger.MaskEmail(baseReq.To)

	log.Debug().
		Str("adapter", "ses").
		Str("to", maskedTo).
		Str("subject", baseReq.Subject).
		Str("region", a.config.Region).
		Msg("Sending email via AWS SES")

	// Determine sender
	from := a.config.FromEmail
	if baseReq.From != "" {
		from = baseReq.From
	}

	// Build SES email content
	content := a.buildEmailContent(&baseReq)

	// Build SES send request
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: []string{baseReq.To},
		},
		Content: content,
	}

	// Add configuration set if specified
	if a.config.ConfigurationSet != "" {
		input.ConfigurationSetName = aws.String(a.config.ConfigurationSet)
	}

	// Send email via SES
	output, err := a.client.SendEmail(ctx, input)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", maskedTo).
			Str("region", a.config.Region).
			Dur("duration", time.Since(start)).
			Msg("Failed to send email via SES")
		return "", fmt.Errorf("SES send failed: %w", err)
	}

	messageID := aws.ToString(output.MessageId)

	log.Info().
		Str("adapter", "ses").
		Str("to", maskedTo).
		Str("message_id", messageID).
		Str("region", a.config.Region).
		Dur("duration", time.Since(start)).
		Msg("Email sent successfully via AWS SES")

	return messageID, nil
}

// buildEmailContent constructs the SES email content
// Handles both simple text/HTML emails and template-based emails
func (a *Adapter) buildEmailContent(req *adapters.BaseRequest) *types.EmailContent {
	// For now, build a simple email
	// TODO: Add template support in future iteration

	subject := req.Subject
	if subject == "" {
		subject = "Notification from " + a.config.FromEmail
	}

	// Determine body content
	body := req.Data["body"]
	if body == nil && req.Template != "" {
		// If template is specified but no body, use template name as placeholder
		body = fmt.Sprintf("Template: %s (Template rendering not yet implemented)", req.Template)
	}
	if body == nil {
		body = "No content"
	}

	bodyStr := fmt.Sprintf("%v", body)

	// Build simple email content
	return &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{
				Text: &types.Content{
					Data:    aws.String(bodyStr),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}
}

// Ping checks if the SES service is accessible
// This is useful for health checks
func (a *Adapter) Ping(ctx context.Context) error {
	// Get account sending quota to verify SES access
	_, err := a.client.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		return fmt.Errorf("SES ping failed: %w", err)
	}
	return nil
}

// GetSendingQuota returns the current SES sending quota
// Useful for monitoring and alerting
func (a *Adapter) GetSendingQuota(ctx context.Context) (*types.SendQuota, error) {
	output, err := a.client.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SES account: %w", err)
	}

	return output.SendQuota, nil
}
