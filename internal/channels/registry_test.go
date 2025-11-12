package channels_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/channels"
)

// mockChannel is a test channel implementation
type mockChannel struct {
	name     string
	chanType channels.Type
	adapters []string
	features []channels.Feature
}

func (m *mockChannel) Name() string {
	return m.name
}

func (m *mockChannel) Type() channels.Type {
	return m.chanType
}

func (m *mockChannel) Validate(ctx context.Context, req *channels.SendRequest) error {
	if req.To == "" {
		return errors.New("recipient required")
	}
	return nil
}

func (m *mockChannel) Send(ctx context.Context, req *channels.SendRequest) (*channels.SendResponse, error) {
	return &channels.SendResponse{
		MessageID: "mock-id-123",
		Channel:   m.name,
		Adapter:   m.adapters[0],
	}, nil
}

func (m *mockChannel) GetAdapters() []string {
	return m.adapters
}

func (m *mockChannel) SupportsFeature(feature channels.Feature) bool {
	for _, f := range m.features {
		if f == feature {
			return true
		}
	}
	return false
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		metadata  channels.Metadata
		factory   channels.Factory
		wantError bool
		errorType error
	}{
		{
			name: "successful registration",
			metadata: channels.Metadata{
				Name:        "test-email",
				Type:        channels.TypeMessaging,
				Description: "Test email channel",
				Version:     "1.0.0",
			},
			factory: func(config interface{}) (channels.Channel, error) {
				return &mockChannel{name: "test-email"}, nil
			},
			wantError: false,
		},
		{
			name: "empty name",
			metadata: channels.Metadata{
				Name: "",
				Type: channels.TypeMessaging,
			},
			factory: func(config interface{}) (channels.Channel, error) {
				return &mockChannel{}, nil
			},
			wantError: true,
			errorType: channels.ErrInvalidChannel,
		},
		{
			name: "nil factory",
			metadata: channels.Metadata{
				Name: "test-nil",
				Type: channels.TypeMessaging,
			},
			factory:   nil,
			wantError: true,
			errorType: channels.ErrInvalidChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := channels.NewRegistry()
			err := registry.Register(tt.metadata, tt.factory)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				}
			}
		})
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	registry := channels.NewRegistry()

	meta := channels.Metadata{
		Name: "duplicate-test",
		Type: channels.TypeMessaging,
	}
	factory := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "duplicate-test"}, nil
	}

	// First registration should succeed
	err := registry.Register(meta, factory)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration should fail
	err = registry.Register(meta, factory)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
	if !errors.Is(err, channels.ErrChannelAlreadyRegistered) {
		t.Errorf("Expected ErrChannelAlreadyRegistered, got %v", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := channels.NewRegistry()

	// Register a test channel
	meta := channels.Metadata{
		Name: "test-get",
		Type: channels.TypeMessaging,
	}
	factory := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "test-get"}, nil
	}

	err := registry.Register(meta, factory)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Get the channel
	channel, err := registry.Get("test-get", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if channel == nil {
		t.Fatal("Expected channel but got nil")
	}

	if channel.Name() != "test-get" {
		t.Errorf("Expected channel name 'test-get', got %s", channel.Name())
	}

	// Get non-existent channel
	_, err = registry.Get("non-existent", nil)
	if err == nil {
		t.Error("Expected error for non-existent channel")
	}
	if !errors.Is(err, channels.ErrChannelNotFound) {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := channels.NewRegistry()

	// Register multiple channels
	channelNames := []string{"email", "sms", "push"}
	for _, name := range channelNames {
		meta := channels.Metadata{
			Name: name,
			Type: channels.TypeMessaging,
		}
		factory := func(n string) channels.Factory {
			return func(config interface{}) (channels.Channel, error) {
				return &mockChannel{name: n}, nil
			}
		}(name)

		err := registry.Register(meta, factory)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", name, err)
		}
	}

	// List all channels
	list := registry.List()
	if len(list) != len(channelNames) {
		t.Errorf("Expected %d channels, got %d", len(channelNames), len(list))
	}

	// Verify all channels are in the list
	for _, name := range channelNames {
		found := false
		for _, listName := range list {
			if listName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find channel %s in list", name)
		}
	}
}

func TestRegistry_ListByType(t *testing.T) {
	registry := channels.NewRegistry()

	// Register channels of different types
	channelTypes := []struct {
		name     string
		chanType channels.Type
	}{
		{"email", channels.TypeMessaging},
		{"sms", channels.TypeMessaging},
		{"push", channels.TypePush},
		{"slack", channels.TypeChatApp},
	}

	for _, ch := range channelTypes {
		meta := channels.Metadata{
			Name: ch.name,
			Type: ch.chanType,
		}
		factory := func(n string, ct channels.Type) channels.Factory {
			return func(config interface{}) (channels.Channel, error) {
				return &mockChannel{name: n, chanType: ct}, nil
			}
		}(ch.name, ch.chanType)

		err := registry.Register(meta, factory)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", ch.name, err)
		}
	}

	// List messaging channels
	messaging := registry.ListByType(channels.TypeMessaging)
	if len(messaging) != 2 {
		t.Errorf("Expected 2 messaging channels, got %d", len(messaging))
	}

	// List push channels
	push := registry.ListByType(channels.TypePush)
	if len(push) != 1 {
		t.Errorf("Expected 1 push channel, got %d", len(push))
	}

	// List chat channels
	chat := registry.ListByType(channels.TypeChatApp)
	if len(chat) != 1 {
		t.Errorf("Expected 1 chat channel, got %d", len(chat))
	}
}

func TestRegistry_GetMetadata(t *testing.T) {
	registry := channels.NewRegistry()

	meta := channels.Metadata{
		Name:        "test-metadata",
		Type:        channels.TypeMessaging,
		Description: "Test channel",
		Version:     "1.0.0",
		Adapters:    []string{"adapter1", "adapter2"},
		Features:    []channels.Feature{channels.FeatureAttachments},
	}

	factory := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "test-metadata"}, nil
	}

	err := registry.Register(meta, factory)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Get metadata
	retrievedMeta := registry.GetMetadata("test-metadata")
	if retrievedMeta == nil {
		t.Fatal("Expected metadata but got nil")
	}

	if retrievedMeta.Name != meta.Name {
		t.Errorf("Expected name %s, got %s", meta.Name, retrievedMeta.Name)
	}

	if retrievedMeta.Description != meta.Description {
		t.Errorf("Expected description %s, got %s", meta.Description, retrievedMeta.Description)
	}

	if len(retrievedMeta.Adapters) != len(meta.Adapters) {
		t.Errorf("Expected %d adapters, got %d", len(meta.Adapters), len(retrievedMeta.Adapters))
	}

	// Get metadata for non-existent channel
	nonExistent := registry.GetMetadata("non-existent")
	if nonExistent != nil {
		t.Error("Expected nil metadata for non-existent channel")
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := channels.NewRegistry()

	meta := channels.Metadata{
		Name: "test-has",
		Type: channels.TypeMessaging,
	}
	factory := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "test-has"}, nil
	}

	err := registry.Register(meta, factory)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Check existing channel
	if !registry.Has("test-has") {
		t.Error("Expected channel to exist")
	}

	// Check non-existent channel
	if registry.Has("non-existent") {
		t.Error("Expected channel to not exist")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := channels.NewRegistry()

	// Number of goroutines
	numGoroutines := 100
	numOperations := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				meta := channels.Metadata{
					Name: "channel-" + string(rune(id)) + "-" + string(rune(j)),
					Type: channels.TypeMessaging,
				}
				factory := func(config interface{}) (channels.Channel, error) {
					return &mockChannel{name: meta.Name}, nil
				}
				_ = registry.Register(meta, factory)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = registry.List()
				_ = registry.ListByType(channels.TypeMessaging)
			}
		}()
	}

	// Concurrent Has checks
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = registry.Has("test-channel")
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Verify registry is still functional
	list := registry.List()
	if len(list) == 0 {
		t.Error("Expected some channels after concurrent access")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := channels.NewRegistry()

	meta := channels.Metadata{
		Name: "test-unregister",
		Type: channels.TypeMessaging,
	}
	factory := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "test-unregister"}, nil
	}

	err := registry.Register(meta, factory)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Verify channel exists
	if !registry.Has("test-unregister") {
		t.Fatal("Expected channel to exist before unregister")
	}

	// Unregister
	err = registry.Unregister("test-unregister")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	// Verify channel no longer exists
	if registry.Has("test-unregister") {
		t.Error("Expected channel to not exist after unregister")
	}

	// Unregister non-existent channel
	err = registry.Unregister("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent channel")
	}
	if !errors.Is(err, channels.ErrChannelNotFound) {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Clean up global registry after test
	defer func() {
		// Unregister test channels
		_ = channels.Unregister("global-test-1")
		_ = channels.Unregister("global-test-2")
	}()

	// Register channels in global registry
	meta1 := channels.Metadata{
		Name: "global-test-1",
		Type: channels.TypeMessaging,
	}
	factory1 := func(config interface{}) (channels.Channel, error) {
		return &mockChannel{name: "global-test-1"}, nil
	}

	err := channels.Register(meta1, factory1)
	if err != nil && !errors.Is(err, channels.ErrChannelAlreadyRegistered) {
		t.Fatalf("Global registration failed: %v", err)
	}

	// Verify channel exists
	if !channels.Has("global-test-1") {
		t.Error("Expected channel in global registry")
	}

	// Get channel
	channel, err := channels.Get("global-test-1", nil)
	if err != nil {
		t.Fatalf("Failed to get channel: %v", err)
	}

	if channel.Name() != "global-test-1" {
		t.Errorf("Expected channel name 'global-test-1', got %s", channel.Name())
	}
}

func TestMustRegister(t *testing.T) {
	// Clean up after test
	defer func() {
		_ = channels.Unregister("must-register-test")
	}()

	// Test successful registration (should not panic)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustRegister panicked unexpectedly: %v", r)
			}
		}()

		channels.MustRegister(
			channels.Metadata{
				Name: "must-register-test",
				Type: channels.TypeMessaging,
			},
			func(config interface{}) (channels.Channel, error) {
				return &mockChannel{name: "must-register-test"}, nil
			},
		)
	}()

	// Verify channel was registered
	if !channels.Has("must-register-test") {
		t.Error("Expected channel to be registered via MustRegister")
	}

	// Test duplicate registration (should NOT panic - should ignore)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustRegister panicked on duplicate: %v", r)
			}
		}()

		channels.MustRegister(
			channels.Metadata{
				Name: "must-register-test",
				Type: channels.TypeMessaging,
			},
			func(config interface{}) (channels.Channel, error) {
				return &mockChannel{name: "must-register-test"}, nil
			},
		)
	}()
}
