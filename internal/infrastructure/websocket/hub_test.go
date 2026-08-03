package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/dukungps/internal/domain/position"
)

func newTestClient() *Client {
	return &Client{send: make(chan []byte, 10)}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := newTestClient()
	hub.Register(client)

	update := &PositionUpdate{
		DeviceID: 1,
		Position: &position.Position{Latitude: -7.24, Longitude: 112.73},
	}
	hub.Broadcast(update)

	select {
	case data := <-client.send:
		assert.Contains(t, string(data), "-7.24")
	case <-time.After(1 * time.Second):
		t.Fatal("client did not receive broadcast")
	}
}

func TestHubUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := newTestClient()
	hub.Register(client)
	hub.Unregister(client)

	update := &PositionUpdate{DeviceID: 1}
	hub.Broadcast(update)

	select {
	case _, ok := <-client.send:
		assert.False(t, ok, "unregistered client channel should be closed, not receiving data")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("unregistered client send channel was not closed")
	}
}

func TestHubMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	c1 := newTestClient()
	c2 := newTestClient()
	hub.Register(c1)
	hub.Register(c2)

	hub.Broadcast(&PositionUpdate{DeviceID: 5})

	for i, c := range []*Client{c1, c2} {
		select {
		case <-c.send:
		case <-time.After(1 * time.Second):
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}
	require.Len(t, hub.clients, 2)
}

func TestHubClientClosedSendChannel(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	client := newTestClient()
	client.close()
	hub.Register(client)

	hub.Broadcast(&PositionUpdate{DeviceID: 9})
	time.Sleep(50 * time.Millisecond)

	// no panic, client should be dropped
	assert.Len(t, hub.clients, 0)
}
