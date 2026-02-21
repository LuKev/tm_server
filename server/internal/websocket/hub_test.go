package websocket

import (
	"testing"
	"time"
)

func TestHubBroadcastToGame_IsRoomScoped(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{hub: hub, send: make(chan []byte, 8), seatsByGame: make(map[string]string)}
	c2 := &Client{hub: hub, send: make(chan []byte, 8), seatsByGame: make(map[string]string)}

	hub.register <- c1
	hub.register <- c2
	hub.JoinGame(c1, "g1")
	hub.JoinGame(c2, "g2")

	msg := []byte(`{"type":"game_state_update","payload":{"id":"g1"}}`)
	hub.BroadcastToGame("g1", msg)

	select {
	case got := <-c1.send:
		if string(got) != string(msg) {
			t.Fatalf("unexpected message for c1: %s", string(got))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for c1 room message")
	}

	select {
	case got := <-c2.send:
		t.Fatalf("c2 should not receive room-scoped message, got: %s", string(got))
	case <-time.After(150 * time.Millisecond):
		// expected
	}

	hub.unregister <- c1
	hub.unregister <- c2
}
