package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/lobby"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict this in production
		return true
	},
}

// ServerDeps contains references to other subsystems used by websocket clients.
type ServerDeps struct {
	Lobby *lobby.Manager
	Games *game.Manager
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, deps ServerDeps, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Generate a simple client ID (in production, use UUID or similar)
	clientID := r.RemoteAddr

	client := &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		id:          clientID,
		deps:        deps,
		seatsByGame: make(map[string]string),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
