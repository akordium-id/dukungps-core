package teltonika

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/akordium-id/dukungps/internal/domain/command"
)

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(cmd command.Command) ([]byte, error) {
	switch cmd.Type {
	case command.TypeEngineStop:
		return e.encodeContent([]byte("setdigout 1"))
	case command.TypeEngineResume:
		return e.encodeContent([]byte("setdigout 0"))
	case command.TypeCustom:
		if isHexString(cmd.Data) {
			content, err := hex.DecodeString(cmd.Data)
			if err != nil {
				return nil, err
			}
			return e.encodeContent(content)
		}
		return e.encodeContent([]byte(cmd.Data))
	default:
		return nil, errors.New("teltonika: unsupported command type")
	}
}

func (e *Encoder) encodeContent(content []byte) ([]byte, error) {
	buf := make([]byte, 0, 16+len(content))

	buf = binary.BigEndian.AppendUint32(buf, 0)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(content)+8))
	buf = append(buf, Codec12)
	buf = append(buf, 1)
	buf = append(buf, 5)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(content)))
	buf = append(buf, content...)
	buf = append(buf, 1)

	crc := crc16IBM(buf[8:])
	buf = binary.BigEndian.AppendUint32(buf, uint32(crc))

	return buf, nil
}

func isHexString(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

func crc16IBM(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
