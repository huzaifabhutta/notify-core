package adapters

import (
	"context"
	"fmt"
	"reflect"
)

// BaseRequest defines the common fields all notification requests should have
type BaseRequest struct {
	To       string
	Template string
	Subject  string
	From     string
	Data     map[string]interface{}
}

// Adapter is the base interface that all channel adapters must implement
type Adapter interface {
	Send(ctx context.Context, req interface{}) error
	Name() string
}

// ExtractBaseRequest extracts common fields from any request struct
// This eliminates duplicate extraction logic across adapters
func ExtractBaseRequest(req interface{}) (BaseRequest, error) {
	// Use reflection to extract fields
	val := reflect.ValueOf(req)

	// Handle pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return BaseRequest{}, fmt.Errorf("nil request pointer")
		}
		val = val.Elem()
	}

	// Must be a struct
	if val.Kind() != reflect.Struct {
		return BaseRequest{}, fmt.Errorf("invalid request type: expected struct, got %T", req)
	}

	// Extract fields
	result := BaseRequest{}
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
				if d, ok := fieldVal.Interface().(map[string]interface{}); ok {
					result.Data = d
				}
			}
		}
	}

	// Validate required fields
	if result.To == "" {
		return BaseRequest{}, fmt.Errorf("missing required field: To")
	}
	if result.Template == "" {
		return BaseRequest{}, fmt.Errorf("missing required field: Template")
	}

	return result, nil
}

// AdapterConfig holds common adapter configuration
type AdapterConfig struct {
	Timeout    int    // Timeout in seconds
	MaxRetries int    // Maximum retry attempts
	Name       string // Adapter name for logging
}

// DefaultConfig returns default adapter configuration
func DefaultConfig(name string) AdapterConfig {
	return AdapterConfig{
		Timeout:    30,
		MaxRetries: 3,
		Name:       name,
	}
}

// WithTimeout is a functional option for setting timeout
func WithTimeout(seconds int) func(*AdapterConfig) {
	return func(c *AdapterConfig) {
		c.Timeout = seconds
	}
}

// WithMaxRetries is a functional option for setting max retries
func WithMaxRetries(retries int) func(*AdapterConfig) {
	return func(c *AdapterConfig) {
		c.MaxRetries = retries
	}
}

// ApplyOptions applies functional options to config
func (c *AdapterConfig) ApplyOptions(opts ...func(*AdapterConfig)) {
	for _, opt := range opts {
		opt(c)
	}
}
