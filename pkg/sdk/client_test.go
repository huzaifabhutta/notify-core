package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080", "test-api-key")

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL 'http://localhost:8080', got %s", client.baseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey 'test-api-key', got %s", client.apiKey)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_WithTimeout(t *testing.T) {
	client := NewClient("http://localhost:8080", "test-api-key").
		WithTimeout(10 * time.Second)

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.httpClient.Timeout)
	}
}

func TestNotification_Validate(t *testing.T) {
	tests := []struct {
		name    string
		notif   Notification
		wantErr bool
	}{
		{
			name: "valid email notification",
			notif: Notification{
				To:       "user@example.com",
				Channel:  "email",
				Template: "welcome",
			},
			wantErr: false,
		},
		{
			name: "valid whatsapp notification",
			notif: Notification{
				To:       "+1234567890",
				Channel:  "whatsapp",
				Template: "welcome",
			},
			wantErr: false,
		},
		{
			name: "missing recipient",
			notif: Notification{
				Channel:  "email",
				Template: "welcome",
			},
			wantErr: true,
		},
		{
			name: "missing channel",
			notif: Notification{
				To:       "user@example.com",
				Template: "welcome",
			},
			wantErr: true,
		},
		{
			name: "missing template",
			notif: Notification{
				To:      "user@example.com",
				Channel: "email",
			},
			wantErr: true,
		},
		{
			name: "unsupported channel",
			notif: Notification{
				To:       "user@example.com",
				Channel:  "telegram",
				Template: "welcome",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.notif.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Send_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/send" {
			t.Errorf("Expected path '/send', got %s", r.URL.Path)
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			t.Errorf("Expected X-API-Key 'test-api-key', got %s", apiKey)
		}

		// Parse request body
		var notif Notification
		if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if notif.To != "user@example.com" {
			t.Errorf("Expected to 'user@example.com', got %s", notif.To)
		}

		// Return success response (new API format)
		resp := map[string]interface{}{
			"status":     "success",
			"message":    "Notification sent successfully",
			"message_id": "msg-123",
			"channel":    "email",
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-api-key")

	// Send notification
	notif := Notification{
		To:       "user@example.com",
		Channel:  "email",
		Template: "welcome",
		Subject:  "Welcome!",
		Data: map[string]interface{}{
			"name": "John Doe",
		},
	}

	resp, err := client.Send(notif)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !resp.IsSuccess() {
		t.Errorf("Expected status 'success', got %s", resp.Status)
	}

	if resp.Message != "Notification sent successfully" {
		t.Errorf("Expected message 'Notification sent successfully', got %s", resp.Message)
	}

	if resp.MessageID != "msg-123" {
		t.Errorf("Expected message_id 'msg-123', got %s", resp.MessageID)
	}

	if resp.Channel != "email" {
		t.Errorf("Expected channel 'email', got %s", resp.Channel)
	}
}

func TestClient_Send_Error(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errResp := map[string]interface{}{
			"status":    "error",
			"error":     "INVALID_REQUEST",
			"message":   "Invalid email address",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-api-key")

	// Send notification
	notif := Notification{
		To:       "invalid-email",
		Channel:  "email",
		Template: "welcome",
	}

	_, err := client.Send(notif)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err != nil && err.Error() != "API error (INVALID_REQUEST): Invalid email address" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestClient_SendEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var notif Notification
		json.NewDecoder(r.Body).Decode(&notif)

		if notif.Channel != "email" {
			t.Errorf("Expected channel 'email', got %s", notif.Channel)
		}

		resp := map[string]interface{}{
			"status":  "success",
			"message": "Sent",
			"channel": "email",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	_, err := client.SendEmail("user@example.com", "Welcome", "welcome", map[string]interface{}{
		"name": "John",
	})

	if err != nil {
		t.Errorf("SendEmail() error = %v", err)
	}
}

func TestClient_SendWhatsApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var notif Notification
		json.NewDecoder(r.Body).Decode(&notif)

		if notif.Channel != "whatsapp" {
			t.Errorf("Expected channel 'whatsapp', got %s", notif.Channel)
		}

		resp := Response{Success: true, Message: "Sent"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	_, err := client.SendWhatsApp("+1234567890", "welcome", map[string]interface{}{
		"name": "John",
	})

	if err != nil {
		t.Errorf("SendWhatsApp() error = %v", err)
	}
}

func TestClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path '/health', got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	err := client.Ping()
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestClient_PingWithContext_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.PingWithContext(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
