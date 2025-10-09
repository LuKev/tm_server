package lobby

import (
    "strconv"
	"sync"
	"time"
)

type GameMeta struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Players    []string  `json:"players"`
	MaxPlayers int       `json:"maxPlayers"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Manager maintains a list of open games for joining
// This is separate from the game.Manager which holds full game state

type Manager struct {
	mu    sync.RWMutex
	games map[string]*GameMeta
    nextID int
}
func NewManager() *Manager {
    return &Manager{games: make(map[string]*GameMeta), nextID: 1}
}

func (m *Manager) CreateGame(name string, maxPlayers int) *GameMeta {
    m.mu.Lock()
    defer m.mu.Unlock()
    id := strconv.Itoa(m.nextID)
    m.nextID++
    g := &GameMeta{ID: id, Name: name, MaxPlayers: maxPlayers, CreatedAt: time.Now(), Players: make([]string, 0, maxPlayers)}
    m.games[id] = g
    return g
}

func (m *Manager) JoinGame(id string, playerName string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    g, ok := m.games[id]
    if !ok {
        return false
    }
    if len(g.Players) >= g.MaxPlayers {
        return false
    }
    // prevent duplicate seats for the same player
    for _, p := range g.Players {
        if p == playerName {
            return false
        }
    }
    g.Players = append(g.Players, playerName)
    return true
}

func (m *Manager) LeaveGame(id string, playerName string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    g, ok := m.games[id]
    if !ok {
        return false
    }
    newPlayers := make([]string, 0, len(g.Players))
    for _, p := range g.Players {
        if p != playerName {
            newPlayers = append(newPlayers, p)
        }
    }
    g.Players = newPlayers
    return true
}

func (m *Manager) ListGames() []*GameMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*GameMeta, 0, len(m.games))
	for _, g := range m.games {
		out = append(out, g)
	}
	return out
}
