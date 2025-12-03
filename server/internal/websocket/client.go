package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
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

		case "get_game_state":
			var p struct {
				GameID string `json:"gameID"`
			}
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				log.Printf("error parsing get_game_state payload: %v", err)
				continue
			}
			gameState := c.deps.Games.SerializeGameState(p.GameID)
			if gameState != nil {
				gameStateMsg, _ := json.Marshal(map[string]any{
					"type":    "game_state_update",
					"payload": gameState,
				})
				c.send <- gameStateMsg
			}

		case "start_game":
			var p struct {
				GameID string `json:"gameID"`
			}
			if err := json.Unmarshal(env.Payload, &p); err != nil {
				log.Printf("error parsing start_game payload: %v", err)
				continue
			}

			// Get game info from lobby
			meta, ok := c.deps.Lobby.GetGame(p.GameID)
			if !ok {
				log.Printf("game not found in lobby: %s", p.GameID)
				errorMsg, _ := json.Marshal(map[string]any{
					"type":    "error",
					"payload": "game_not_found",
				})
				c.send <- errorMsg
				continue
			}

			// Validate all slots are filled
			if len(meta.Players) < meta.MaxPlayers {
				log.Printf("game %s not full: %d/%d players", p.GameID, len(meta.Players), meta.MaxPlayers)
				errorMsg, _ := json.Marshal(map[string]any{
					"type":    "error",
					"payload": map[string]any{
						"error":       "game_not_full",
						"playerCount": len(meta.Players),
						"maxPlayers":  meta.MaxPlayers,
					},
				})
				c.send <- errorMsg
				continue
			}

			// Initialize game state
			err := c.deps.Games.CreateGame(p.GameID, meta.Players)
			if err != nil {
				log.Printf("error creating game: %v", err)
				// If game already exists, we might just want to broadcast the state anyway
				// continue
			}

			// Broadcast initial state
			gameState := c.deps.Games.SerializeGameState(p.GameID)
			gameStateMsg, _ := json.Marshal(map[string]any{
				"type":    "game_state_update",
				"payload": gameState,
			})
			c.hub.broadcast <- gameStateMsg
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
				// Send game_created message to creator
				createdMsg, _ := json.Marshal(map[string]any{
					"type": "game_created",
					"payload": map[string]string{"gameId": meta.ID, "playerId": p.Creator},
				})
				c.send <- createdMsg
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
			// Send success to the joining client
			successMsg, _ := json.Marshal(map[string]any{
				"type": "game_joined", 
				"payload": map[string]string{"gameId": p.ID, "playerId": p.Name},
			})
			c.send <- successMsg

			games := c.deps.Lobby.ListGames()
			out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
			c.hub.broadcast <- out

		case "perform_action":
			// Handle game actions (setup dwelling, transform & build, etc.)
			log.Printf("Received perform_action from client %s: %s", c.id, string(env.Payload))
			
			// Parse action payload
			var actionPayload struct {
				Type     string `json:"type"`
				PlayerID string `json:"playerID"`
				Faction  string `json:"faction,omitempty"`
				Hex      *struct {
					Q int `json:"q"`
					R int `json:"r"`
				} `json:"hex,omitempty"`
				GameID string `json:"gameID"` // Added GameID to payload
			}
			if err := json.Unmarshal(env.Payload, &actionPayload); err != nil {
				log.Printf("perform_action payload error: %v", err)
				errorMsg, _ := json.Marshal(map[string]any{
					"type": "error",
					"payload": "invalid_action_payload",
				})
				c.send <- errorMsg
				continue
			}
			
			// Use GameID from payload if present, otherwise default (for backward compatibility/testing)
			gameID := actionPayload.GameID
			if gameID == "" {
				gameID = "2" // Fallback for now
			}
			
			// Create appropriate action based on type
			var action game.Action
			switch actionPayload.Type {
			case "select_faction":
				if actionPayload.Faction == "" {
					log.Printf("select_faction missing faction")
					errorMsg, _ := json.Marshal(map[string]any{
						"type": "error",
						"payload": "missing_faction",
					})
					c.send <- errorMsg
					continue
				}
				action = &game.SelectFactionAction{
					PlayerID:    actionPayload.PlayerID,
					FactionType: models.FactionTypeFromString(actionPayload.Faction),
				}
			case "setup_dwelling":
				if actionPayload.Hex == nil {
					log.Printf("setup_dwelling missing hex")
					errorMsg, _ := json.Marshal(map[string]any{
						"type": "error",
						"payload": "missing_hex",
					})
					c.send <- errorMsg
					continue
				}
				hex := board.NewHex(actionPayload.Hex.Q, actionPayload.Hex.R)
				action = game.NewSetupDwellingAction(actionPayload.PlayerID, hex)
			default:
				log.Printf("unknown action type: %s", actionPayload.Type)
				errorMsg, _ := json.Marshal(map[string]any{
					"type": "error",
					"payload": "unknown_action_type",
				})
				c.send <- errorMsg
				continue
			}
			
			// Execute action via game manager
			err := c.deps.Games.ExecuteAction(gameID, action)
			if err != nil {
				log.Printf("action execution failed: %v", err)
				errorMsg, _ := json.Marshal(map[string]any{
					"type": "action_rejected",
					"payload": map[string]string{
						"error": err.Error(),
						"action": actionPayload.Type,
					},
				})
				c.send <- errorMsg
				continue
			}
			
			// Action succeeded - broadcast updated game state to all clients
			log.Printf("Action executed successfully: %s", actionPayload.Type)
			gameState := c.deps.Games.SerializeGameState(gameID)
			stateMsg, _ := json.Marshal(map[string]any{
				"type": "game_state_update",
				"payload": gameState,
			})
			c.hub.broadcast <- stateMsg

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
