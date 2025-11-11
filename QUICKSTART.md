# Quick Start Guide

Get notify-core up and running in 5 minutes!

## Prerequisites

- Go 1.21+ installed
- SMTP credentials (Gmail, SendGrid, AWS SES, etc.)
- Docker (optional)

## Option 1: Run Locally

### 1. Clone and Setup

```bash
git clone https://github.com/huzaifabhutta/notify-core
cd notify-core
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment

Create a `.env` file:

```bash
cp .env.example .env
```

Edit `.env` with your SMTP credentials:

```env
PORT=8080
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=noreply@yourapp.com
```

### 4. Run the Server

```bash
go run cmd/server/main.go
```

Or use Make:

```bash
make run
```

The server will start at `http://localhost:8080`

## Option 2: Run with Docker

### 1. Configure Environment

```bash
cp .env.example .env
# Edit .env with your credentials
```

### 2. Start Containers

```bash
docker-compose up -d
```

### 3. Check Status

```bash
docker-compose logs -f notify-core
```

## Testing the API

### 1. Check Health

```bash
curl http://localhost:8080/health
```

### 2. Send a Test Email

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "recipient@example.com",
    "channel": "email",
    "template": "welcome",
    "subject": "Welcome to Our Platform!",
    "data": {
      "CustomerName": "John Doe",
      "AccountID": "ACC-12345",
      "ActionURL": "https://yourapp.com/get-started"
    }
  }'
```

### 3. Expected Response

```json
{
  "success": true,
  "message": "Notification sent successfully"
}
```

## Using Different Templates

### Welcome Email

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "channel": "email",
    "template": "welcome",
    "subject": "Welcome!",
    "data": {
      "CustomerName": "Jane Smith",
      "ActionURL": "https://app.com/start"
    }
  }'
```

### Order Confirmation

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "customer@example.com",
    "channel": "email",
    "template": "order-confirmation",
    "subject": "Your Order is Confirmed!",
    "data": {
      "CustomerName": "John Doe",
      "OrderID": "ORD-12345",
      "Total": "99.99",
      "TrackingURL": "https://track.com/ORD-12345"
    }
  }'
```

### Password Reset

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "channel": "email",
    "template": "password-reset",
    "subject": "Reset Your Password",
    "data": {
      "CustomerName": "Jane Smith",
      "ResetURL": "https://app.com/reset?token=abc123",
      "ExpiresIn": "24 hours"
    }
  }'
```

## Using with Gmail

### Enable App Passwords

1. Go to [Google Account Settings](https://myaccount.google.com/)
2. Navigate to Security → 2-Step Verification
3. Scroll down to "App passwords"
4. Generate a new app password for "Mail"
5. Use this password in your `.env` file

### Gmail Configuration

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password  # 16-character app password
SMTP_FROM=your-email@gmail.com
```

## Development Commands

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Build binary
make build

# Format code
make fmt

# Clean artifacts
make clean

# Docker build
make docker-build

# Start Docker containers
make docker-up

# Stop Docker containers
make docker-down
```

## Troubleshooting

### SMTP Connection Failed

- Check your SMTP credentials
- Verify SMTP host and port
- For Gmail, ensure app passwords are enabled
- Check firewall settings

### Template Not Found

- Ensure template file exists in `./templates/`
- Template name should match filename without `.html`
- Check file permissions

### Port Already in Use

Change the port in `.env`:

```env
PORT=9090
```

## Next Steps

1. **Create Custom Templates**: Add new HTML templates in `./templates/`
2. **Add WhatsApp Support**: Coming in Week 2
3. **Setup Multi-tenancy**: Coming in Week 3
4. **Use the SDK**: Coming in Week 3

## Support

- Documentation: See [README.md](./README.md)
- Templates: See [templates/README.md](./templates/README.md)
- Issues: [GitHub Issues](https://github.com/huzaifabhutta/notify-core/issues)

---

**Ready to build something awesome!** 🚀
