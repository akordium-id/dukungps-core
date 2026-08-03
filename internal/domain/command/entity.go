package command

import (
	"time"
)

const (
	TypePositionSingle = "positionSingle"
	TypeEngineStop     = "engineStop"
	TypeEngineResume   = "engineResume"
	TypeRebootDevice   = "rebootDevice"
	TypeCustom         = "custom"
)

const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
)

type Command struct {
	ID        int64
	DeviceID  int64
	Type      string
	Data      string
	Status    string
	SentAt    time.Time
	CreatedAt time.Time
}

func (c *Command) IsValid() bool {
	return c.DeviceID > 0 && c.Type != ""
}
