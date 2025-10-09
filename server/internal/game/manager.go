package game

import (
	"sync"

	"github.com/lukev/tm_server/internal/models"
)

// Manager owns and guards access to game state instances in-memory.
// This will later be backed by persistent storage.

type Manager struct {
	mu    sync.RWMutex
	games map[string]*models.GameState
}

func NewManager() *Manager {
	return &Manager{
		games: make(map[string]*models.GameState),
	}
}

func (m *Manager) CreateGame(id string, initial models.GameState) *models.GameState {
	m.mu.Lock()
	defer m.mu.Unlock()
	g := initial
	g.ID = id
	m.games[id] = &g
	return &g
}

func (m *Manager) GetGame(id string) (*models.GameState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.games[id]
	return g, ok
}

func (m *Manager) ListGames() []*models.GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.GameState, 0, len(m.games))
	for _, g := range m.games {
		out = append(out, g)
	}
	return out
}
