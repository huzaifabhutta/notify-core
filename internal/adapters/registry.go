package adapters

import (
	"fmt"
	"sync"
)

var (
	// ErrAdapterNotFound is returned when an adapter is not registered
	ErrAdapterNotFound = fmt.Errorf("adapter not found")

	// ErrAdapterAlreadyRegistered is returned when registering a duplicate adapter
	ErrAdapterAlreadyRegistered = fmt.Errorf("adapter already registered")

	// ErrInvalidAdapter is returned when adapter validation fails
	ErrInvalidAdapter = fmt.Errorf("invalid adapter")
)

// Factory creates an adapter instance with the given configuration
// Configuration type is interface{} to allow adapter-specific config structs
type Factory func(config interface{}) (Adapter, error)

// Registry manages adapter registration and instantiation
// It is thread-safe and can be used concurrently
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
	metadata  map[string]Metadata
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
		metadata:  make(map[string]Metadata),
	}
}

// Register registers an adapter factory with metadata
// Returns ErrAdapterAlreadyRegistered if the adapter name is already taken
func (r *Registry) Register(meta Metadata, factory Factory) error {
	if meta.Name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidAdapter)
	}
	if factory == nil {
		return fmt.Errorf("%w: factory cannot be nil", ErrInvalidAdapter)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[meta.Name]; exists {
		return fmt.Errorf("%w: %s", ErrAdapterAlreadyRegistered, meta.Name)
	}

	r.factories[meta.Name] = factory
	r.metadata[meta.Name] = meta

	return nil
}

// MustRegister registers an adapter and panics on error
// Use this in init() functions for built-in adapters
func (r *Registry) MustRegister(meta Metadata, factory Factory) {
	if err := r.Register(meta, factory); err != nil {
		panic(fmt.Sprintf("failed to register adapter %s: %v", meta.Name, err))
	}
}

// Get retrieves an adapter by name and creates an instance with the given config
// Returns ErrAdapterNotFound if the adapter is not registered
func (r *Registry) Get(name string, config interface{}) (Adapter, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAdapterNotFound, name)
	}

	adapter, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter %s: %w", name, err)
	}

	return adapter, nil
}

// Has checks if an adapter is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[name]
	return exists
}

// List returns all registered adapter names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// ListByType returns all adapter names of a specific type
func (r *Registry) ListByType(adapterType Type) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0)
	for name, meta := range r.metadata {
		if meta.Type == adapterType {
			names = append(names, name)
		}
	}
	return names
}

// GetMetadata returns metadata for a registered adapter
// Returns nil if the adapter is not found
func (r *Registry) GetMetadata(name string) *Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, exists := r.metadata[name]; exists {
		// Return a copy to prevent external modification
		metaCopy := meta
		return &metaCopy
	}
	return nil
}

// Global registry instance for convenience
// Applications can use this or create their own registry
var global = NewRegistry()

// Register registers an adapter in the global registry
func Register(meta Metadata, factory Factory) error {
	return global.Register(meta, factory)
}

// MustRegister registers an adapter in the global registry and panics on error
func MustRegister(meta Metadata, factory Factory) {
	global.MustRegister(meta, factory)
}

// Get retrieves an adapter from the global registry
func Get(name string, config interface{}) (Adapter, error) {
	return global.Get(name, config)
}

// Has checks if an adapter exists in the global registry
func Has(name string) bool {
	return global.Has(name)
}

// List returns all adapters in the global registry
func List() []string {
	return global.List()
}

// ListByType returns adapters of a specific type from the global registry
func ListByType(adapterType Type) []string {
	return global.ListByType(adapterType)
}

// GetMetadata returns adapter metadata from the global registry
func GetMetadata(name string) *Metadata {
	return global.GetMetadata(name)
}
