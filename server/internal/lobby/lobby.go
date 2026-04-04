package lobby

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/game/board"
)

var (
	ErrGameNotFound       = errors.New("game not found")
	ErrGameAlreadyStarted = errors.New("game already started")
	ErrGameFull           = errors.New("game full")
	ErrAlreadyInOpenGame  = errors.New("player already seated in another open game")
	ErrPlayerNotInGame    = errors.New("player not seated in this game")
	ErrInvalidMap         = errors.New("invalid map")
)

type GameMeta struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Host       string    `json:"host"`
	MapID      string    `json:"mapId"`
	CustomMap  *board.CustomMapDefinition `json:"customMap,omitempty"`
	Players    []string  `json:"players"`
	MaxPlayers int       `json:"maxPlayers"`
	Started    bool      `json:"started"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Manager maintains a list of open games for joining
// This is separate from the game.Manager which holds full game state

type Manager struct {
	mu             sync.RWMutex
	games          map[string]*GameMeta
	openGameByUser map[string]string
	nextID         int
}

func NewManager() *Manager {
	return &Manager{
		games:          make(map[string]*GameMeta),
		openGameByUser: make(map[string]string),
		nextID:         1,
	}
}

func cloneGameMeta(in *GameMeta) *GameMeta {
	if in == nil {
		return nil
	}
	out := *in
	out.Players = append([]string(nil), in.Players...)
	out.CustomMap = board.CloneCustomMapDefinition(in.CustomMap)
	return &out
}

func (m *Manager) CreateGame(name string, maxPlayers int, host string, mapID string, customMap *board.CustomMapDefinition) (*GameMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	host = strings.TrimSpace(host)
	if existingID, ok := m.openGameByUser[host]; host != "" && ok {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyInOpenGame, existingID)
	}

	id := strconv.Itoa(m.nextID)
	m.nextID++
	normalizedMapID := board.NormalizeMapID(mapID)
	if normalizedMapID == board.MapCustom {
		if _, err := board.NewTerraMysticaMapForCustom(customMap); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMap, err)
		}
	} else {
		if customMap != nil {
			return nil, fmt.Errorf("%w: custom map payload requires mapId=%s", ErrInvalidMap, board.MapCustom)
		}
		if _, ok := board.MapInfoByID(normalizedMapID); !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMap, strings.TrimSpace(mapID))
		}
	}
	if normalizedMapID == "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMap, strings.TrimSpace(mapID))
	}
	g := &GameMeta{
		ID:         id,
		Name:       name,
		Host:       host,
		MapID:      string(normalizedMapID),
		CustomMap:  board.CloneCustomMapDefinition(customMap),
		MaxPlayers: maxPlayers,
		CreatedAt:  time.Now(),
		Players:    make([]string, 0, maxPlayers),
	}
	m.games[id] = g
	if host != "" {
		g.Players = append(g.Players, host)
		m.openGameByUser[host] = id
	}
	return cloneGameMeta(g), nil
}

func (m *Manager) GetGame(id string) (*GameMeta, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.games[id]
	if !ok {
		return nil, false
	}
	return cloneGameMeta(g), true
}

func (m *Manager) JoinGame(id string, playerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	playerName = strings.TrimSpace(playerName)
	g, ok := m.games[id]
	if !ok {
		return ErrGameNotFound
	}
	if g.Started {
		return ErrGameAlreadyStarted
	}
	if existingID, alreadySeated := m.openGameByUser[playerName]; alreadySeated {
		if existingID == id {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrAlreadyInOpenGame, existingID)
	}
	if len(g.Players) >= g.MaxPlayers {
		return ErrGameFull
	}
	g.Players = append(g.Players, playerName)
	m.openGameByUser[playerName] = id
	return nil
}

func (m *Manager) LeaveGame(id string, playerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	playerName = strings.TrimSpace(playerName)
	g, ok := m.games[id]
	if !ok {
		return ErrGameNotFound
	}
	if g.Started {
		return ErrGameAlreadyStarted
	}
	newPlayers := make([]string, 0, len(g.Players))
	found := false
	for _, p := range g.Players {
		if p == playerName {
			found = true
			continue
		}
		newPlayers = append(newPlayers, p)
	}
	if !found {
		return ErrPlayerNotInGame
	}
	g.Players = newPlayers
	delete(m.openGameByUser, playerName)
	if len(g.Players) == 0 {
		delete(m.games, id)
		return nil
	}
	if g.Host == playerName {
		g.Host = g.Players[0]
	}
	return nil
}

func (m *Manager) StartGame(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.games[id]
	if !ok {
		return ErrGameNotFound
	}
	if g.Started {
		return nil
	}
	g.Started = true
	for _, playerID := range g.Players {
		delete(m.openGameByUser, playerID)
	}
	return nil
}

func (m *Manager) ListGames() []*GameMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*GameMeta, 0, len(m.games))
	for _, g := range m.games {
		if g.Started {
			continue
		}
		out = append(out, cloneGameMeta(g))
	}
	return out
}
