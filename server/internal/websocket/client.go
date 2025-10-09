package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512 KB
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Client ID for identification
	id string

	// Access to other server subsystems
	deps ServerDeps
}

// inbound message envelope from client
type inboundMsg struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload,omitempty"`
}

// lobby messages
type createGamePayload struct {
    Name       string `json:"name"`
    MaxPlayers int    `json:"maxPlayers"`
    Creator    string `json:"creator"`
}

type joinGamePayload struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// outbound message helpers
type lobbyStateMsg struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// readPump pumps messages from the websocket connection to the hub and other subsystems.
// There must be only one reader on a connection.
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break
        }
        message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

        // Try to parse as JSON envelope for commands
        var env inboundMsg
        if err := json.Unmarshal(message, &env); err != nil {
            // Not JSON - ignore for now
            log.Printf("Received non-JSON message from %s: %s", c.id, string(message))
            continue
        }

        switch env.Type {
        case "list_games":
            games := c.deps.Lobby.ListGames()
            out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
            c.send <- out

        case "create_game":
            var p createGamePayload
            if err := json.Unmarshal(env.Payload, &p); err != nil {
                log.Printf("create_game payload error: %v", err)
                continue
            }
            if p.MaxPlayers <= 0 {
                p.MaxPlayers = 5
            }
            meta := c.deps.Lobby.CreateGame(p.Name, p.MaxPlayers)
            if p.Creator != "" {
                // Auto-join creator; ignore join failure silently
                _ = c.deps.Lobby.JoinGame(meta.ID, p.Creator)
            }
            games := c.deps.Lobby.ListGames()
            out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
            // broadcast updated lobby
            c.hub.broadcast <- out

        case "join_game":
            var p joinGamePayload
            if err := json.Unmarshal(env.Payload, &p); err != nil {
                log.Printf("join_game payload error: %v", err)
                continue
            }
            ok := c.deps.Lobby.JoinGame(p.ID, p.Name)
            if !ok {
                // send failure
                out, _ := json.Marshal(map[string]any{"type": "error", "payload": "join_failed"})
                c.send <- out
                continue
            }
            games := c.deps.Lobby.ListGames()
            out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
            c.hub.broadcast <- out

        default:
            log.Printf("Unknown message type: %s", env.Type)
        }
    }
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
