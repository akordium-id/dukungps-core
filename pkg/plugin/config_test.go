package plugin_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/pkg/plugin"
)

type configMockCore struct {
	*mockCore
	env map[string]string
}

func (c *configMockCore) Config(key string, fallback string) string {
	if val, ok := c.env[key]; ok {
		return val
	}
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func TestPluginConfig_ScopedKeys(t *testing.T) {
	core := &configMockCore{
		mockCore: &mockCore{},
		env: map[string]string{
			"PLUGIN_GEOFENCE_ENABLED":   "true",
			"PLUGIN_GEOFENCE_THRESHOLD": "50",
			"PLUGIN_GEOFENCE_WEBHOOK":   "https://example.com/hook",
		},
	}

	cfg := plugin.NewPluginConfig("geofence", core)

	assert.True(t, cfg.GetBool("ENABLED", false))
	assert.Equal(t, 50, cfg.GetInt("THRESHOLD", 10))
	assert.Equal(t, "https://example.com/hook", cfg.GetString("WEBHOOK", ""))
	assert.Equal(t, "default_val", cfg.GetString("NON_EXISTENT", "default_val"))
}

func TestManagedRegistry_FeatureFlagDisable(t *testing.T) {
	core := &configMockCore{
		mockCore: &mockCore{},
		env: map[string]string{
			"PLUGIN_DISABLED_PLUGIN_ENABLED": "false",
		},
	}

	reg := plugin.NewManagedRegistry(core)

	enabledP := &testPlugin{name: "active-plugin", version: "1.0.0"}
	disabledP := &testPlugin{name: "disabled-plugin", version: "1.0.0"}

	require.NoError(t, reg.Register(enabledP))
	require.NoError(t, reg.Register(disabledP))

	require.NoError(t, reg.InitAll())

	list := reg.List()
	require.Len(t, list, 2)

	assert.Equal(t, plugin.StatusInitialized, list[0].Status)
	assert.True(t, enabledP.initCalled)

	assert.Equal(t, plugin.StatusDisabled, list[1].Status)
	assert.False(t, disabledP.initCalled)

	reg.CloseAll()
	assert.True(t, enabledP.closeCalled)
	assert.False(t, disabledP.closeCalled)
}
