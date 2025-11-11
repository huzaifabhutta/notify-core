package adapters

// Type represents the category of adapter (email, whatsapp, sms)
type Type string

const (
	TypeEmail    Type = "email"
	TypeWhatsApp Type = "whatsapp"
	TypeSMS      Type = "sms"
)

// Metadata contains information about an adapter
type Metadata struct {
	Name        string // Unique identifier (e.g., "smtp", "ses", "cloud-api")
	Type        Type   // Category (email, whatsapp, sms)
	Description string // Human-readable description
	Version     string // Adapter version
}
