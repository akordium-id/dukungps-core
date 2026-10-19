package plugin

import (
	"fmt"
	"strings"
	"sync"
)

// ConfigProvider allows accessing hierarchical or flat configuration for plugins.
type ConfigProvider interface {
	GetString(key string, fallback string) string
	GetBool(key string, fallback bool) bool
	GetInt(key string, fallback int) int
}

// PluginConfig holds config values specifically scoped for a plugin (e.g. PLUGIN_<NAME>_<KEY>).
type PluginConfig struct {
	pluginName string
	core       Core
}

func NewPluginConfig(pluginName string, core Core) *PluginConfig {
	return &PluginConfig{
		pluginName: pluginName,
		core:       core,
	}
}

// ScopedKey formats key into uppercase env style: PLUGIN_<PLUGINNAME>_<KEY>
func (c *PluginConfig) ScopedKey(key string) string {
	cleanName := strings.ToUpper(strings.ReplaceAll(c.pluginName, "-", "_"))
	cleanKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
	return fmt.Sprintf("PLUGIN_%s_%s", cleanName, cleanKey)
}

func (c *PluginConfig) GetString(key string, fallback string) string {
	scoped := c.ScopedKey(key)
	// Try scoped first
	if val := c.core.Config(scoped, ""); val != "" {
		return val
	}
	// Fallback to direct key or fallback default
	return c.core.Config(key, fallback)
}

func (c *PluginConfig) GetBool(key string, fallback bool) bool {
	str := strings.ToLower(strings.TrimSpace(c.GetString(key, "")))
	switch str {
	case "1", "true", "yes", "on", "enable", "enabled":
		return true
	case "0", "false", "no", "off", "disable", "disabled":
		return false
	default:
		return fallback
	}
}

func (c *PluginConfig) GetInt(key string, fallback int) int {
	str := strings.TrimSpace(c.GetString(key, ""))
	if str == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(str, "%d", &n); err == nil {
		return n
	}
	return fallback
}

// PluginStatus tracks the operational state of a plugin.
type PluginStatus string

const (
	StatusRegistered  PluginStatus = "registered"
	StatusInitialized PluginStatus = "initialized"
	StatusDisabled    PluginStatus = "disabled"
	StatusFailed      PluginStatus = "failed"
	StatusClosed      PluginStatus = "closed"
)

// PluginInfo contains status and metadata for a registered plugin.
type PluginInfo struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description,omitempty"`
	Status      PluginStatus `json:"status"`
	Error       string       `json:"error,omitempty"`
}

// ManagedRegistry extends Registry with feature flags, scoped configs, and status introspection.
type ManagedRegistry struct {
	mu       sync.RWMutex
	core     Core
	plugins  map[string]Plugin
	statuses map[string]*PluginInfo
	ordered  []string
}

func NewManagedRegistry(core Core) *ManagedRegistry {
	return &ManagedRegistry{
		core:     core,
		plugins:  make(map[string]Plugin),
		statuses: make(map[string]*PluginInfo),
		ordered:  make([]string, 0),
	}
}

// Register adds a plugin with initial 'registered' status.
func (r *ManagedRegistry) Register(p Plugin) error {
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
	r.statuses[name] = &PluginInfo{
		Name:    name,
		Version: p.Version(),
		Status:  StatusRegistered,
	}
	r.ordered = append(r.ordered, name)
	return nil
}

// IsEnabled checks if a plugin is enabled via env/config (default true unless PLUGIN_<NAME>_ENABLED=false).
func (r *ManagedRegistry) IsEnabled(name string) bool {
	cfg := NewPluginConfig(name, r.core)
	return cfg.GetBool("ENABLED", true)
}

// InitAll initializes all registered and enabled plugins.
func (r *ManagedRegistry) InitAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range r.ordered {
		p := r.plugins[name]
		info := r.statuses[name]

		// Check if disabled by feature flag
		cfg := NewPluginConfig(name, r.core)
		if !cfg.GetBool("ENABLED", true) {
			info.Status = StatusDisabled
			continue
		}

		if err := p.Init(r.core); err != nil {
			info.Status = StatusFailed
			info.Error = err.Error()
			return fmt.Errorf("failed to initialize plugin %q: %w", name, err)
		}
		info.Status = StatusInitialized
	}
	return nil
}

// CloseAll shuts down active plugins in reverse order.
func (r *ManagedRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.ordered) - 1; i >= 0; i-- {
		name := r.ordered[i]
		info := r.statuses[name]
		if info.Status != StatusInitialized {
			continue
		}

		p := r.plugins[name]
		if err := p.Close(); err != nil {
			info.Error = err.Error()
		}
		info.Status = StatusClosed
	}
}

// List returns details of all registered plugins.
func (r *ManagedRegistry) List() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginInfo, 0, len(r.ordered))
	for _, name := range r.ordered {
		if info, ok := r.statuses[name]; ok {
			result = append(result, *info)
		}
	}
	return result
}
