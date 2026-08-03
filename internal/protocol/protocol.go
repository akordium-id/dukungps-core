package protocol

import (
	"fmt"

	"github.com/akordium-id/dukungps/internal/domain/command"
	"github.com/akordium-id/dukungps/internal/domain/position"
)

const (
	Teltonika = "teltonika"
	GT06      = "gt06"
	Huabao    = "huabao"
)

type DecodeResult struct {
	DeviceID  string
	Positions []*position.Position
	Response  []byte
}

type Protocol interface {
	Name() string
	Decoder() Decoder
	Encoder() Encoder
}

type Decoder interface {
	Decode(data []byte) (*DecodeResult, error)
}

type Encoder interface {
	Encode(cmd command.Command) ([]byte, error)
}

type Registry struct {
	protocols map[string]Protocol
}

func NewRegistry() *Registry {
	return &Registry{
		protocols: make(map[string]Protocol),
	}
}

func (r *Registry) Register(p Protocol) {
	r.protocols[p.Name()] = p
}

func (r *Registry) Get(name string) (Protocol, error) {
	p, ok := r.protocols[name]
	if !ok {
		return nil, fmt.Errorf("protocol %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.protocols))
	for name := range r.protocols {
		names = append(names, name)
	}
	return names
}
