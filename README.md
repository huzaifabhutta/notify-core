# Notify-Core

A self-hosted, multi-tenant, developer-friendly notification service supporting Email, WhatsApp, and SMS.

## Features

- 🚀 Simple SDK: `client.send()`
- 📧 Email support (SMTP)
- 📱 WhatsApp integration
- 💬 SMS support (coming soon)
- 🏢 Multi-tenant architecture
- 🎨 Template system with variables
- 🐳 Docker-ready
- 💰 Cost-free for small deployments

## Project Structure

```
notify-core/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── notify/          # Core notification logic
│   ├── email/           # Email channel adapter
│   ├── whatsapp/        # WhatsApp channel adapter
│   ├── sms/             # SMS channel adapter
│   └── config/          # Configuration management
├── pkg/
│   └── sdk/             # Client SDK for easy integration
├── templates/           # Email/notification templates
└── docker-compose.yml   # Docker setup
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose (optional)

### Configuration

Create a `.env` file:

```env
# SMTP Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# WhatsApp Configuration
WA_TOKEN=your-whatsapp-token
WA_PHONE_ID=your-phone-id

# Database
DB_URL=postgres://user:pass@localhost:5432/notify

# Server
PORT=8080
```

### Running Locally

```bash
go run cmd/server/main.go
```

### Using Docker

```bash
docker-compose up
```

## API Usage

### Send Notification

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "channel": "email",
    "template": "welcome",
    "data": {
      "CustomerName": "John Doe",
      "OrderID": "12345"
    }
  }'
```

## SDK Usage

```go
import "github.com/huzaifabhutta/notify-core/pkg/sdk"

client := sdk.NewClient("http://localhost:8080")
err := client.Send(sdk.Notification{
    To:       "user@example.com",
    Channel:  "email",
    Template: "welcome",
    Data: map[string]interface{}{
        "CustomerName": "John Doe",
        "OrderID":      "12345",
    },
})
```

## Development Roadmap

### Week 1 - Core Foundation ✅
- [x] Project scaffolding
- [ ] Environment config
- [ ] Base REST API
- [ ] Email channel (SMTP)
- [ ] Template system
- [ ] Docker setup

### Week 2 - WhatsApp Integration
- [ ] WhatsApp Business API setup
- [ ] Message formatting
- [ ] Media support

### Week 3 - Multi-tenancy & SDK
- [ ] Tenant management
- [ ] API keys & authentication
- [ ] Go SDK
- [ ] Rate limiting

### Week 4 - Production Ready
- [ ] Database integration
- [ ] Queue system
- [ ] Monitoring & logging
- [ ] Documentation

## License

MIT
