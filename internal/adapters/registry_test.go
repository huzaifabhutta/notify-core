package adapters

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// mockAdapter is a simple adapter for testing
type mockAdapter struct {
	name string
}

func (m *mockAdapter) Send(ctx context.Context, req interface{}) (string, error) {
	return "mock-message-id", nil
}

func (m *mockAdapter) Name() string {
	return m.name
}

func mockFactory(config interface{}) (Adapter, error) {
	return &mockAdapter{name: "mock"}, nil
}

func failingFactory(config interface{}) (Adapter, error) {
	return nil, errors.New("factory error")
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.factories == nil {
		t.Error("factories map not initialized")
	}

	if registry.metadata == nil {
		t.Error("metadata map not initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name:        "test-adapter",
		Type:        TypeEmail,
		Description: "Test adapter",
		Version:     "1.0.0",
	}

	err := registry.Register(meta, mockFactory)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Verify registered
	if !registry.Has("test-adapter") {
		t.Error("Adapter not found after registration")
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "", // Empty name
		Type: TypeEmail,
	}

	err := registry.Register(meta, mockFactory)
	if err == nil {
		t.Error("Expected error for empty adapter name")
	}

	if !errors.Is(err, ErrInvalidAdapter) {
		t.Errorf("Expected ErrInvalidAdapter, got %v", err)
	}
}

func TestRegistry_Register_NilFactory(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "test",
		Type: TypeEmail,
	}

	err := registry.Register(meta, nil)
	if err == nil {
		t.Error("Expected error for nil factory")
	}

	if !errors.Is(err, ErrInvalidAdapter) {
		t.Errorf("Expected ErrInvalidAdapter, got %v", err)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "duplicate",
		Type: TypeEmail,
	}

	// First registration
	err := registry.Register(meta, mockFactory)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration (duplicate)
	err = registry.Register(meta, mockFactory)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}

	if !errors.Is(err, ErrAdapterAlreadyRegistered) {
		t.Errorf("Expected ErrAdapterAlreadyRegistered, got %v", err)
	}
}

func TestRegistry_MustRegister_Success(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "must-register-test",
		Type: TypeEmail,
	}

	// Should not panic
	registry.MustRegister(meta, mockFactory)

	if !registry.Has("must-register-test") {
		t.Error("Adapter not registered")
	}
}

func TestRegistry_MustRegister_Panic(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "", // Invalid
		Type: TypeEmail,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister() should panic on error")
		}
	}()

	registry.MustRegister(meta, mockFactory)
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "get-test",
		Type: TypeEmail,
	}

	registry.Register(meta, mockFactory)

	adapter, err := registry.Get("get-test", nil)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if adapter == nil {
		t.Error("Get() returned nil adapter")
	}

	// Test Send method
	ctx := context.Background()
	msgID, err := adapter.Send(ctx, nil)
	if err != nil {
		t.Errorf("Send() failed: %v", err)
	}

	if msgID != "mock-message-id" {
		t.Errorf("Expected 'mock-message-id', got %s", msgID)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	adapter, err := registry.Get("nonexistent", nil)
	if err == nil {
		t.Error("Expected error for nonexistent adapter")
	}

	if adapter != nil {
		t.Error("Expected nil adapter")
	}

	if !errors.Is(err, ErrAdapterNotFound) {
		t.Errorf("Expected ErrAdapterNotFound, got %v", err)
	}
}

func TestRegistry_Get_FactoryError(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name: "failing",
		Type: TypeEmail,
	}

	registry.Register(meta, failingFactory)

	adapter, err := registry.Get("failing", nil)
	if err == nil {
		t.Error("Expected error from failing factory")
	}

	if adapter != nil {
		t.Error("Expected nil adapter on factory error")
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()

	if registry.Has("nonexistent") {
		t.Error("Has() returned true for nonexistent adapter")
	}

	meta := Metadata{
		Name: "exists",
		Type: TypeEmail,
	}

	registry.Register(meta, mockFactory)

	if !registry.Has("exists") {
		t.Error("Has() returned false for registered adapter")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}

	// Register multiple adapters
	for i := 1; i <= 3; i++ {
		meta := Metadata{
			Name: fmt.Sprintf("adapter-%d", i),
			Type: TypeEmail,
		}
		registry.Register(meta, mockFactory)
	}

	list = registry.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 adapters, got %d", len(list))
	}
}

func TestRegistry_ListByType(t *testing.T) {
	registry := NewRegistry()

	// Register adapters of different types
	registry.Register(Metadata{Name: "email-1", Type: TypeEmail}, mockFactory)
	registry.Register(Metadata{Name: "email-2", Type: TypeEmail}, mockFactory)
	registry.Register(Metadata{Name: "whatsapp-1", Type: TypeWhatsApp}, mockFactory)
	registry.Register(Metadata{Name: "sms-1", Type: TypeSMS}, mockFactory)

	emailAdapters := registry.ListByType(TypeEmail)
	if len(emailAdapters) != 2 {
		t.Errorf("Expected 2 email adapters, got %d", len(emailAdapters))
	}

	whatsappAdapters := registry.ListByType(TypeWhatsApp)
	if len(whatsappAdapters) != 1 {
		t.Errorf("Expected 1 whatsapp adapter, got %d", len(whatsappAdapters))
	}

	smsAdapters := registry.ListByType(TypeSMS)
	if len(smsAdapters) != 1 {
		t.Errorf("Expected 1 SMS adapter, got %d", len(smsAdapters))
	}
}

func TestRegistry_GetMetadata(t *testing.T) {
	registry := NewRegistry()

	meta := Metadata{
		Name:        "metadata-test",
		Type:        TypeEmail,
		Description: "Test description",
		Version:     "1.0.0",
	}

	registry.Register(meta, mockFactory)

	retrieved := registry.GetMetadata("metadata-test")
	if retrieved == nil {
		t.Fatal("GetMetadata() returned nil")
	}

	if retrieved.Name != meta.Name {
		t.Errorf("Name mismatch: expected %s, got %s", meta.Name, retrieved.Name)
	}

	if retrieved.Type != meta.Type {
		t.Errorf("Type mismatch: expected %s, got %s", meta.Type, retrieved.Type)
	}

	if retrieved.Description != meta.Description {
		t.Errorf("Description mismatch: expected %s, got %s", meta.Description, retrieved.Description)
	}

	if retrieved.Version != meta.Version {
		t.Errorf("Version mismatch: expected %s, got %s", meta.Version, retrieved.Version)
	}

	// Test mutation doesn't affect stored metadata
	retrieved.Name = "modified"
	retrieved2 := registry.GetMetadata("metadata-test")
	if retrieved2.Name != meta.Name {
		t.Error("Metadata was modified externally (should be immutable)")
	}
}

func TestRegistry_GetMetadata_NotFound(t *testing.T) {
	registry := NewRegistry()

	meta := registry.GetMetadata("nonexistent")
	if meta != nil {
		t.Error("Expected nil for nonexistent adapter")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	var wg sync.WaitGroup
	const goroutines = 100

	// Concurrent registrations
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			meta := Metadata{
				Name: fmt.Sprintf("adapter-%d", id),
				Type: TypeEmail,
			}

			err := registry.Register(meta, mockFactory)
			if err != nil {
				t.Errorf("Concurrent registration failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all registered
	list := registry.List()
	if len(list) != goroutines {
		t.Errorf("Expected %d adapters, got %d", goroutines, len(list))
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			name := fmt.Sprintf("adapter-%d", id)
			if !registry.Has(name) {
				t.Errorf("Adapter %s not found", name)
			}

			meta := registry.GetMetadata(name)
			if meta == nil {
				t.Errorf("Metadata for %s not found", name)
			}
		}(i)
	}

	wg.Wait()
}

func TestGlobalRegistry(t *testing.T) {
	// Note: This test uses the global registry
	// It might interfere with other tests if run in parallel

	meta := Metadata{
		Name: "global-test",
		Type: TypeEmail,
	}

	err := Register(meta, mockFactory)
	if err != nil {
		t.Fatalf("Global Register() failed: %v", err)
	}

	if !Has("global-test") {
		t.Error("Adapter not found in global registry")
	}

	adapter, err := Get("global-test", nil)
	if err != nil {
		t.Fatalf("Global Get() failed: %v", err)
	}

	if adapter == nil {
		t.Error("Global Get() returned nil adapter")
	}

	list := List()
	found := false
	for _, name := range list {
		if name == "global-test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Adapter not in global List()")
	}

	metaRetrieved := GetMetadata("global-test")
	if metaRetrieved == nil {
		t.Error("GetMetadata() returned nil")
	}
}
