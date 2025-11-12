package channels

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrChannelNotFound is returned when a channel is not found in the registry
	ErrChannelNotFound = fmt.Errorf("channel not found")

	// ErrChannelAlreadyRegistered is returned when attempting to register a duplicate channel
	ErrChannelAlreadyRegistered = fmt.Errorf("channel already registered")

	// ErrInvalidChannel is returned when a channel configuration is invalid
	ErrInvalidChannel = fmt.Errorf("invalid channel")
)

// Factory creates a channel instance with the given configuration
type Factory func(config interface{}) (Channel, error)

// Registry manages channel registration and instantiation
// It is thread-safe and can be used concurrently
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
	metadata  map[string]Metadata
}

// NewRegistry creates a new channel registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
		metadata:  make(map[string]Metadata),
	}
}

// Register registers a channel with its factory
func (r *Registry) Register(meta Metadata, factory Factory) error {
	if meta.Name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidChannel)
	}
	if factory == nil {
		return fmt.Errorf("%w: factory cannot be nil", ErrInvalidChannel)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[meta.Name]; exists {
		return fmt.Errorf("%w: %s", ErrChannelAlreadyRegistered, meta.Name)
	}

	r.factories[meta.Name] = factory
	r.metadata[meta.Name] = meta
	return nil
}

// Get creates a channel instance with the given configuration
func (r *Registry) Get(name string, config interface{}) (Channel, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrChannelNotFound, name)
	}

	channel, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel %s: %w", name, err)
	}
	return channel, nil
}

// List returns all registered channel names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// ListByType returns channels of a specific type
func (r *Registry) ListByType(channelType Type) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, meta := range r.metadata {
		if meta.Type == channelType {
			names = append(names, name)
		}
	}
	return names
}

// GetMetadata returns metadata for a channel
func (r *Registry) GetMetadata(name string) *Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, exists := r.metadata[name]; exists {
		metaCopy := meta // Create a copy to avoid race conditions
		return &metaCopy
	}
	return nil
}

// Has checks if a channel is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// Unregister removes a channel from the registry (useful for testing)
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; !exists {
		return fmt.Errorf("%w: %s", ErrChannelNotFound, name)
	}

	delete(r.factories, name)
	delete(r.metadata, name)
	return nil
}

// Global registry instance
var global = NewRegistry()

// Register registers a channel in the global registry
func Register(meta Metadata, factory Factory) error {
	return global.Register(meta, factory)
}

// MustRegister registers a channel or panics (useful in init() functions)
func MustRegister(meta Metadata, factory Factory) {
	if err := Register(meta, factory); err != nil {
		// If already registered, ignore (allows multiple imports)
		if errors.Is(err, ErrChannelAlreadyRegistered) {
			return
		}
		panic(fmt.Sprintf("failed to register channel %s: %v", meta.Name, err))
	}
}

// Get gets a channel from the global registry
func Get(name string, config interface{}) (Channel, error) {
	return global.Get(name, config)
}

// List lists all channels in the global registry
func List() []string {
	return global.List()
}

// ListByType lists channels by type in the global registry
func ListByType(channelType Type) []string {
	return global.ListByType(channelType)
}

// GetMetadata gets metadata from the global registry
func GetMetadata(name string) *Metadata {
	return global.GetMetadata(name)
}

// Has checks if a channel exists in the global registry
func Has(name string) bool {
	return global.Has(name)
}

// Unregister removes a channel from the global registry
func Unregister(name string) error {
	return global.Unregister(name)
}
