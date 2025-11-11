# Email Templates

This directory contains HTML email templates with Go template syntax.

## Available Templates

### welcome.html
Welcome email for new users.

**Variables:**
- `CustomerName` (string) - Name of the customer
- `AccountID` (string, optional) - Account ID
- `ActionURL` (string, optional) - Call-to-action URL

**Example:**
```json
{
  "CustomerName": "John Doe",
  "AccountID": "ACC-12345",
  "ActionURL": "https://app.example.com/onboarding"
}
```

### order-confirmation.html
Order confirmation email.

**Variables:**
- `CustomerName` (string) - Name of the customer
- `OrderID` (string) - Order ID
- `OrderDate` (string, optional) - Order date
- `Items` (array, optional) - Array of items with Name and Price
- `Total` (string, optional) - Total amount
- `ShippingAddress` (string, optional) - Shipping address
- `TrackingURL` (string, optional) - Tracking URL

**Example:**
```json
{
  "CustomerName": "Jane Smith",
  "OrderID": "ORD-67890",
  "OrderDate": "2024-11-11",
  "Items": [
    {"Name": "Product A", "Price": "29.99"},
    {"Name": "Product B", "Price": "49.99"}
  ],
  "Total": "79.98",
  "ShippingAddress": "123 Main St, City, State 12345",
  "TrackingURL": "https://track.example.com/ORD-67890"
}
```

### password-reset.html
Password reset email.

**Variables:**
- `CustomerName` (string) - Name of the customer
- `ResetURL` (string, optional) - Password reset URL
- `ResetCode` (string, optional) - Reset code
- `ExpiresIn` (string, optional) - Expiration time (e.g., "24 hours")

**Example:**
```json
{
  "CustomerName": "John Doe",
  "ResetURL": "https://app.example.com/reset?token=abc123",
  "ExpiresIn": "24 hours"
}
```

## Creating Custom Templates

1. Create a new `.html` file in this directory
2. Use Go template syntax for variables: `{{.VariableName}}`
3. Use conditionals: `{{if .Variable}}...{{end}}`
4. Use loops: `{{range .Items}}...{{end}}`

### Example Template

```html
<!DOCTYPE html>
<html>
<body>
    <h1>Hello {{.Name}}!</h1>
    {{if .Message}}
    <p>{{.Message}}</p>
    {{end}}
</body>
</html>
```

## Template Syntax Reference

- `{{.Variable}}` - Insert variable
- `{{if .Variable}}...{{end}}` - Conditional
- `{{if .Variable}}...{{else}}...{{end}}` - If-else
- `{{range .Array}}...{{end}}` - Loop over array
- `{{.}}` - Current item in range
