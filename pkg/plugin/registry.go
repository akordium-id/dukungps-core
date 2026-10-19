package plugin

import (
	"fmt"
	"log"
	"sync"
)

// Registry manages the lifecycle of registered plugins.
type Registry struct {
	mu      sync.RWMutex
	core    Core
	plugins map[string]Plugin
	ordered []string
}

func NewRegistry(core Core) *Registry {
	return &Registry{
		core:    core,
		plugins: make(map[string]Plugin),
		ordered: make([]string, 0),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	r.ordered = append(r.ordered, name)
	return nil
}

// InitAll initializes all registered plugins with Core in registration order.
func (r *Registry) InitAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range r.ordered {
		p := r.plugins[name]
		log.Printf("[plugin] Initializing %s v%s...", p.Name(), p.Version())
		if err := p.Init(r.core); err != nil {
			return fmt.Errorf("failed to init plugin %q: %w", name, err)
		}
		log.Printf("[plugin] Initialized %s successfully", p.Name())
	}
	return nil
}

// CloseAll closes all initialized plugins in reverse order.
func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.ordered) - 1; i >= 0; i-- {
		name := r.ordered[i]
		p := r.plugins[name]
		if err := p.Close(); err != nil {
			log.Printf("[plugin] Error closing %s: %v", name, err)
		}
	}
}

// Plugins returns the list of registered plugin names.
func (r *Registry) Plugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]string, len(r.ordered))
	copy(res, r.ordered)
	return res
}
