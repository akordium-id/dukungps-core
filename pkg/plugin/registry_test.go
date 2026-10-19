package plugin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
)

type mockCore struct {
	plugin.Core
	hooks []plugin.PositionHook
}

func (m *mockCore) OnPosition(hook plugin.PositionHook) {
	m.hooks = append(m.hooks, hook)
}

type testPlugin struct {
	name        string
	version     string
	initCalled  bool
	closeCalled bool
}

func (p *testPlugin) Name() string    { return p.name }
func (p *testPlugin) Version() string { return p.version }
func (p *testPlugin) Init(core plugin.Core) error {
	p.initCalled = true
	core.OnPosition(func(ctx context.Context, p *position.Position) error {
		return nil
	})
	return nil
}
func (p *testPlugin) Close() error {
	p.closeCalled = true
	return nil
}

func TestPluginRegistryLifecycle(t *testing.T) {
	core := &mockCore{}
	reg := plugin.NewRegistry(core)

	p1 := &testPlugin{name: "test-plugin-1", version: "1.0.0"}
	err := reg.Register(p1)
	require.NoError(t, err)

	assert.Equal(t, []string{"test-plugin-1"}, reg.Plugins())

	// Duplicate registration error
	err = reg.Register(p1)
	assert.Error(t, err)

	// Init
	err = reg.InitAll()
	require.NoError(t, err)
	assert.True(t, p1.initCalled)

	// Close
	reg.CloseAll()
	assert.True(t, p1.closeCalled)
}
