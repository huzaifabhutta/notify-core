package errors

import "fmt"

// ErrorCode represents a machine-readable error code
type ErrorCode string

const (
	// Client errors (4xx)
	ErrInvalidRequest     ErrorCode = "INVALID_REQUEST"
	ErrAuthRequired       ErrorCode = "AUTH_REQUIRED"
	ErrInvalidAPIKey      ErrorCode = "INVALID_API_KEY"
	ErrUnsupportedChannel ErrorCode = "UNSUPPORTED_CHANNEL"
	ErrTemplateNotFound   ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrInvalidRecipient   ErrorCode = "INVALID_RECIPIENT"
	ErrInvalidTemplate    ErrorCode = "INVALID_TEMPLATE"
	ErrPayloadTooLarge    ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Server errors (5xx)
	ErrSendFailed      ErrorCode = "SEND_FAILED"
	ErrInternalError   ErrorCode = "INTERNAL_ERROR"
	ErrConfigError     ErrorCode = "CONFIG_ERROR"
	ErrTemplateError   ErrorCode = "TEMPLATE_ERROR"
	ErrAdapterError    ErrorCode = "ADAPTER_ERROR"
)

// AppError represents an application error with safe public message
type AppError struct {
	Code        ErrorCode `json:"code"`
	Message     string    `json:"message"`
	InternalErr error     `json:"-"` // Never sent to client
	StatusCode  int       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.InternalErr != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Code, e.Message, e.InternalErr)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the internal error for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.InternalErr
}

// New creates a new AppError
func New(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an internal error with a safe public message
func Wrap(code ErrorCode, message string, statusCode int, internalErr error) *AppError {
	return &AppError{
		Code:        code,
		Message:     message,
		InternalErr: internalErr,
		StatusCode:  statusCode,
	}
}

// Common error constructors

// InvalidRequest creates an invalid request error
func InvalidRequest(message string, internalErr error) *AppError {
	return Wrap(ErrInvalidRequest, message, 400, internalErr)
}

// UnsupportedChannel creates an unsupported channel error
func UnsupportedChannel(channel string) *AppError {
	return New(ErrUnsupportedChannel, fmt.Sprintf("Unsupported channel: %s", channel), 400)
}

// TemplateNotFound creates a template not found error
func TemplateNotFound(template string) *AppError {
	return New(ErrTemplateNotFound, fmt.Sprintf("Template not found: %s", template), 404)
}

// SendFailed creates a send failed error
func SendFailed(channel string, internalErr error) *AppError {
	return Wrap(ErrSendFailed, fmt.Sprintf("Failed to send notification via %s", channel), 500, internalErr)
}

// InternalError creates an internal error
func InternalError(message string, internalErr error) *AppError {
	return Wrap(ErrInternalError, message, 500, internalErr)
}

// ConfigError creates a configuration error
func ConfigError(message string, internalErr error) *AppError {
	return Wrap(ErrConfigError, message, 500, internalErr)
}
