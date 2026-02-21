package websocket

import (
	"log"
	"sync"
)

type gameBroadcastMessage struct {
	GameID  string
	Message []byte
}

// Hub maintains connected websocket clients and room subscriptions.
type Hub struct {
	clients map[*Client]bool

	broadcast     chan []byte
	gameBroadcast chan gameBroadcastMessage
	register      chan *Client
	unregister    chan *Client

	mu sync.RWMutex

	gameSubscribers map[string]map[*Client]bool
	clientGames     map[*Client]map[string]bool
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan []byte),
		gameBroadcast:   make(chan gameBroadcastMessage),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		gameSubscribers: make(map[string]map[*Client]bool),
		clientGames:     make(map[*Client]map[string]bool),
	}
}

// Run starts the hub loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			h.unregisterClientLocked(client)
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				h.sendToClientLocked(client, message)
			}
			h.mu.RUnlock()

		case msg := <-h.gameBroadcast:
			h.mu.RLock()
			for client := range h.gameSubscribers[msg.GameID] {
				h.sendToClientLocked(client, msg.Message)
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) unregisterClientLocked(client *Client) {
	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	if games := h.clientGames[client]; games != nil {
		for gameID := range games {
			if subscribers := h.gameSubscribers[gameID]; subscribers != nil {
				delete(subscribers, client)
				if len(subscribers) == 0 {
					delete(h.gameSubscribers, gameID)
				}
			}
		}
		delete(h.clientGames, client)
	}

	close(client.send)
	log.Printf("Client disconnected. Total clients: %d", len(h.clients))
}

func (h *Hub) sendToClientLocked(client *Client, message []byte) {
	select {
	case client.send <- message:
	default:
		close(client.send)
		delete(h.clients, client)
		if games := h.clientGames[client]; games != nil {
			for gameID := range games {
				if subscribers := h.gameSubscribers[gameID]; subscribers != nil {
					delete(subscribers, client)
					if len(subscribers) == 0 {
						delete(h.gameSubscribers, gameID)
					}
				}
			}
			delete(h.clientGames, client)
		}
	}
}

// BroadcastMessage sends a message to all connected clients.
func (h *Hub) BroadcastMessage(message []byte) {
	h.broadcast <- message
}

// BroadcastToGame sends a message to subscribers of a single game room.
func (h *Hub) BroadcastToGame(gameID string, message []byte) {
	h.gameBroadcast <- gameBroadcastMessage{GameID: gameID, Message: message}
}

// JoinGame subscribes a client to a game room.
func (h *Hub) JoinGame(client *Client, gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client]; !exists {
		return
	}

	if h.gameSubscribers[gameID] == nil {
		h.gameSubscribers[gameID] = make(map[*Client]bool)
	}
	h.gameSubscribers[gameID][client] = true

	if h.clientGames[client] == nil {
		h.clientGames[client] = make(map[string]bool)
	}
	h.clientGames[client][gameID] = true
}

// LeaveGame unsubscribes a client from a game room.
func (h *Hub) LeaveGame(client *Client, gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers := h.gameSubscribers[gameID]; subscribers != nil {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(h.gameSubscribers, gameID)
		}
	}

	if games := h.clientGames[client]; games != nil {
		delete(games, gameID)
		if len(games) == 0 {
			delete(h.clientGames, client)
		}
	}
}

// GetClientCount returns connected clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
