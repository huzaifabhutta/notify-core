package sdk_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/huzaifabhutta/notify-core/pkg/sdk"
)

// Example demonstrates basic SDK usage
func Example() {
	// Create client
	client := sdk.NewClient("http://notify-core:8080", "your-api-key")

	// Send email
	resp, err := client.SendEmail(
		"user@example.com",
		"Welcome!",
		"welcome",
		map[string]interface{}{
			"CustomerName": "John Doe",
		},
	)

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("Sent: %s\n", resp.Message)
}

// Example_sendWhatsApp demonstrates WhatsApp messaging
func Example_sendWhatsApp() {
	client := sdk.NewClient("http://notify-core:8080", "your-api-key")

	resp, err := client.SendWhatsApp(
		"+1234567890",
		"order_update",
		map[string]interface{}{
			"order_id": "12345",
			"status":   "shipped",
		},
	)

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("WhatsApp sent: %s\n", resp.Message)
}

// Example_withContext demonstrates using context for timeout
func Example_withContext() {
	client := sdk.NewClient("http://notify-core:8080", "your-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := sdk.Notification{
		To:       "user@example.com",
		Channel:  "email",
		Template: "welcome",
		Subject:  "Welcome!",
		Data: map[string]interface{}{
			"CustomerName": "Jane",
		},
	}

	resp, err := client.SendWithContext(ctx, notification)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Printf("Sent: %s\n", resp.Message)
}

// Example_errorHandling demonstrates proper error handling
func Example_errorHandling() {
	client := sdk.NewClient("http://notify-core:8080", "your-api-key")

	notification := sdk.Notification{
		To:       "user@example.com",
		Channel:  "email",
		Template: "welcome",
	}

	resp, err := client.Send(notification)
	if err != nil {
		// Handle different error types
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Success: %s (ID: %s)\n", resp.Message, resp.MessageID)
}

// Example_healthCheck demonstrates health checking
func Example_healthCheck() {
	client := sdk.NewClient("http://notify-core:8080", "your-api-key")

	if err := client.Ping(); err != nil {
		log.Printf("Service is down: %v", err)
		return
	}

	fmt.Println("Service is healthy")
}

// Example_orderConfirmation demonstrates real-world usage
func Example_orderConfirmation() {
	client := sdk.NewClient("http://notify-core:8080", "mrqz-key")

	// Send order confirmation email
	_, err := client.SendEmail(
		"customer@example.com",
		"Your Order Confirmation #12345",
		"order-confirmation",
		map[string]interface{}{
			"CustomerName": "John Doe",
			"OrderID":      "12345",
			"Total":        "$99.99",
			"Items": []map[string]interface{}{
				{"Name": "Product A", "Price": "$49.99"},
				{"Name": "Product B", "Price": "$50.00"},
			},
			"TrackingURL": "https://track.example.com/12345",
		},
	)

	if err != nil {
		log.Printf("Failed to send order confirmation: %v", err)
		return
	}

	// Also send WhatsApp notification
	_, err = client.SendWhatsApp(
		"+1234567890",
		"order_confirmed",
		map[string]interface{}{
			"order_id": "12345",
		},
	)

	if err != nil {
		log.Printf("Failed to send WhatsApp: %v", err)
	}

	fmt.Println("Order notifications sent")
}
