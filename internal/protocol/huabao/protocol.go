package huabao

import "github.com/akordium-id/dukungps/internal/protocol"

type Protocol struct{}

func (p *Protocol) Name() string          { return protocol.Huabao }
func (p *Protocol) Decoder() protocol.Decoder { return NewDecoder() }
func (p *Protocol) Encoder() protocol.Encoder { return nil }

var _ protocol.Protocol = (*Protocol)(nil)
