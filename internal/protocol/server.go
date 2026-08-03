package protocol

import (
	"errors"
	"io"
	"log"
	"net"
)

var (
	errNotFramed = errors.New("protocol does not implement FrameDecoder")
)

const maxBufferSize = 64 * 1024

// PositionHandler receives decoded positions associated with a device.
type PositionHandler interface {
	HandlePositions(deviceID string, res *DecodeResult)
}

type Server struct {
	registry  *Registry
	ports     map[string]int
	handlers  map[string]PositionHandler
	listeners map[string]net.Listener
}

func NewServer(registry *Registry) *Server {
	return &Server{
		registry:  registry,
		ports:     make(map[string]int),
		handlers:  make(map[string]PositionHandler),
		listeners: make(map[string]net.Listener),
	}
}

// RegisterPort associates a protocol name with a TCP port.
func (s *Server) RegisterPort(name string, port int) {
	s.ports[name] = port
}

// SetHandler sets the position handler for a protocol.
func (s *Server) SetHandler(name string, handler PositionHandler) {
	s.handlers[name] = handler
}

func (s *Server) Start() error {
	for name, port := range s.ports {
		proto, err := s.registry.Get(name)
		if err != nil {
			continue
		}
		ln, err := net.Listen("tcp", addrFor(port))
		if err != nil {
			return err
		}
		s.listeners[name] = ln
		go s.acceptLoop(name, ln, proto)
	}
	return nil
}

// Addr returns the actual bound address for a protocol listener (useful when
// the port is 0, i.e. OS-assigned).
func (s *Server) Addr(name string) net.Addr {
	if ln, ok := s.listeners[name]; ok {
		return ln.Addr()
	}
	return nil
}

// Stop closes all listeners.
func (s *Server) Stop() {
	for _, ln := range s.listeners {
		_ = ln.Close()
	}
}

func addrFor(port int) string {
	return ":" + itoa(port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 5)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func (s *Server) acceptLoop(name string, ln net.Listener, proto Protocol) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn, name, proto)
	}
}

func (s *Server) handleConnection(conn net.Conn, name string, proto Protocol) {
	defer conn.Close()

	decoder := proto.Decoder()
	framer, ok := decoder.(FrameDecoder)
	if !ok {
		log.Printf("protocol %s: %v", name, errNotFramed)
		return
	}

	buf := make([]byte, 0, 4096)
	readBuf := make([]byte, 4096)
	var deviceID string

	for {
		n, err := conn.Read(readBuf)
		if err != nil {
			if err != io.EOF {
				log.Printf("protocol %s: read error: %v", name, err)
			}
			return
		}
		buf = append(buf, readBuf[:n]...)

		for {
			frameLen, err := framer.FrameLen(buf)
			if err != nil {
				if len(buf) > maxBufferSize {
					return
				}
				break
			}
			frame := buf[:frameLen]
			buf = buf[frameLen:]

			res, err := decoder.Decode(frame)
			if err != nil {
				continue
			}

			if res.DeviceID != "" {
				deviceID = res.DeviceID
			}

			if len(res.Response) > 0 {
				if _, err := conn.Write(res.Response); err != nil {
					return
				}
			}

			if handler, ok := s.handlers[name]; ok && handler != nil {
				handler.HandlePositions(deviceID, res)
			}
		}
	}
}
