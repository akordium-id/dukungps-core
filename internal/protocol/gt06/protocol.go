package gt06

import "github.com/akordium-id/dukungps/internal/protocol"

const DefaultPort = 5023

type Protocol struct{}

func (p *Protocol) Name() string              { return protocol.GT06 }
func (p *Protocol) Decoder() protocol.Decoder { return NewDecoder() }
func (p *Protocol) Encoder() protocol.Encoder { return nil }

var _ protocol.Protocol = (*Protocol)(nil)
