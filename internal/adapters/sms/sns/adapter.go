package sns

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/logger"
)

// Adapter implements the AWS SNS SMS adapter using native AWS SDK
// This provides access to SNS features including:
// - SMS sending
// - Delivery status tracking
// - Sender ID customization
// - Transactional vs promotional messaging
type Adapter struct {
	client *sns.Client
	config *Config
}

// Config holds AWS SNS adapter configuration
type Config struct {
	// AWS Region (e.g., "us-east-1", "eu-west-1")
	Region string

	// Optional: Default Sender ID (appears as sender on recipient's device)
	// Note: Not supported in all regions/countries
	SenderID string

	// Optional: SMS Type - "Transactional" or "Promotional"
	// Transactional: Higher priority, higher cost
	// Promotional: Lower priority, lower cost (default)
	SMSType string

	// Optional: AWS credentials
	// If not provided, uses default credential chain
	AccessKeyID     string
	SecretAccessKey string

	// Optional: Topic ARN for publishing (alternative to direct SMS)
	TopicARN string
}

// NewAdapter creates a new AWS SNS adapter instance
// This is the factory function used by the adapter registry
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
	snsConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("%w: expected *sns.Config", adapters.ErrInvalidAdapter)
	}

	// Validate required fields
	if snsConfig.Region == "" {
		return nil, fmt.Errorf("%w: region is required", adapters.ErrInvalidAdapter)
	}

	// Default to Promotional if not specified
	if snsConfig.SMSType == "" {
		snsConfig.SMSType = "Promotional"
	}

	// Validate SMS type
	if snsConfig.SMSType != "Transactional" && snsConfig.SMSType != "Promotional" {
		return nil, fmt.Errorf("%w: sms_type must be 'Transactional' or 'Promotional'", adapters.ErrInvalidAdapter)
	}

	// Load AWS configuration
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(snsConfig.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SNS client
	client := sns.NewFromConfig(awsCfg)

	return &Adapter{
		client: client,
		config: snsConfig,
	}, nil
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "sns"
}

// Send sends an SMS via AWS SNS
// Returns the SNS Message ID for tracking
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
	start := time.Now()
	log := logger.FromContext(ctx)

	// Extract SMS request
	smsReq, err := a.extractSMSRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract SNS request")
		return "", fmt.Errorf("sns adapter: %w", err)
	}

	// Validate phone number
	if smsReq.To == "" {
		return "", fmt.Errorf("recipient phone number is required")
	}

	if smsReq.Body == "" {
		return "", fmt.Errorf("message body is required")
	}

	// Mask phone number for logging
	maskedPhone := logger.MaskPhone(smsReq.To)

	log.Debug().
		Str("adapter", "sns").
		Str("to", maskedPhone).
		Str("sms_type", a.config.SMSType).
		Str("region", a.config.Region).
		Msg("Sending SMS via AWS SNS")

	// Build SNS publish input
	input := &sns.PublishInput{
		PhoneNumber: aws.String(smsReq.To),
		Message:     aws.String(smsReq.Body),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"AWS.SNS.SMS.SMSType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(a.config.SMSType),
			},
		},
	}

	// Add sender ID if configured
	if a.config.SenderID != "" {
		input.MessageAttributes["AWS.SNS.SMS.SenderID"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(a.config.SenderID),
		}
	}

	// Send SMS via SNS
	output, err := a.client.Publish(ctx, input)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", maskedPhone).
			Str("region", a.config.Region).
			Dur("duration", time.Since(start)).
			Msg("Failed to send SMS via SNS")
		return "", fmt.Errorf("SNS publish failed: %w", err)
	}

	messageID := aws.ToString(output.MessageId)

	log.Info().
		Str("adapter", "sns").
		Str("to", maskedPhone).
		Str("message_id", messageID).
		Str("region", a.config.Region).
		Dur("duration", time.Since(start)).
		Msg("SMS sent successfully via AWS SNS")

	return messageID, nil
}

// extractSMSRequest extracts SMS request from interface{}
func (a *Adapter) extractSMSRequest(req interface{}) (*SMSRequest, error) {
	// Try direct type assertion first
	if smsReq, ok := req.(*SMSRequest); ok {
		return smsReq, nil
	}

	// Try map[string]interface{}
	if mapReq, ok := req.(map[string]interface{}); ok {
		to, _ := mapReq["to"].(string)
		body, _ := mapReq["body"].(string)

		return &SMSRequest{
			To:   to,
			Body: body,
		}, nil
	}

	// Try using BaseRequest extraction
	baseReq, err := adapters.ExtractBaseRequest(req)
	if err != nil {
		return nil, err
	}

	// Extract body from Data if not in main fields
	body := ""
	if bodyVal, ok := baseReq.Data["body"]; ok {
		body = fmt.Sprintf("%v", bodyVal)
	}

	return &SMSRequest{
		To:   baseReq.To,
		Body: body,
	}, nil
}

// SMSRequest represents an SMS send request
type SMSRequest struct {
	To   string // Phone number in E.164 format (e.g., +1234567890)
	Body string // Message content (max 160 chars for single SMS)
}

// Ping checks if the SNS service is accessible
// This is useful for health checks
func (a *Adapter) Ping(ctx context.Context) error {
	// Try to list SMS sandbox phone numbers to verify SNS access
	_, err := a.client.GetSMSSandboxAccountStatus(ctx, &sns.GetSMSSandboxAccountStatusInput{})
	if err != nil {
		// If sandbox API fails, try a simpler call
		_, err = a.client.GetSMSAttributes(ctx, &sns.GetSMSAttributesInput{})
		if err != nil {
			return fmt.Errorf("SNS ping failed: %w", err)
		}
	}
	return nil
}

// GetSMSAttributes returns current SNS SMS attributes
// Useful for monitoring spend limits and default settings
func (a *Adapter) GetSMSAttributes(ctx context.Context) (map[string]string, error) {
	output, err := a.client.GetSMSAttributes(ctx, &sns.GetSMSAttributesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SMS attributes: %w", err)
	}

	return output.Attributes, nil
}
