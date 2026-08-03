package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		command Command
		want    bool
	}{
		{
			name:    "valid command",
			command: Command{DeviceID: 1, Type: TypePositionSingle},
			want:    true,
		},
		{
			name:    "missing device id",
			command: Command{DeviceID: 0, Type: TypePositionSingle},
			want:    false,
		},
		{
			name:    "missing type",
			command: Command{DeviceID: 1, Type: ""},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.command.IsValid())
		})
	}
}

func TestCommandTypeConstants(t *testing.T) {
	assert.Equal(t, "positionSingle", TypePositionSingle)
	assert.Equal(t, "engineStop", TypeEngineStop)
	assert.Equal(t, "engineResume", TypeEngineResume)
	assert.Equal(t, "rebootDevice", TypeRebootDevice)
	assert.Equal(t, "custom", TypeCustom)
}

func TestCommandStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "sent", StatusSent)
	assert.Equal(t, "failed", StatusFailed)
}
