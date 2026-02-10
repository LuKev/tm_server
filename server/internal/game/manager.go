package game

import (
	"fmt"
	"sync"

	"github.com/lukev/tm_server/internal/models"
)

// Manager owns and guards access to game state instances in-memory.
// This will later be backed by persistent storage.

// Manager handles multiple game instances
type Manager struct {
	mu    sync.RWMutex
	games map[string]*GameState // Changed from models.GameState to game.GameState
}

// NewManager creates a new game manager
func NewManager() *Manager {
	return &Manager{
		games: make(map[string]*GameState),
	}
}

// CreateGameWithState creates a game with an existing GameState
func (m *Manager) CreateGameWithState(id string, gs *GameState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[id] = gs
}

// GetGame retrieves a game by ID
func (m *Manager) GetGame(id string) (*GameState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.games[id]
	return g, ok
}

// ListGames returns all active games
func (m *Manager) ListGames() []*GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*GameState, 0, len(m.games))
	for _, g := range m.games {
		out = append(out, g)
	}
	return out
}

// ExecuteAction executes an action in the specified game
func (m *Manager) ExecuteAction(gameID string, action Action) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	gs := m.games[gameID]
	if gs == nil {
		return fmt.Errorf("game %s not found", gameID)
	}

	// Validate the action
	if err := action.Validate(gs); err != nil {
		return fmt.Errorf("action validation failed: %w", err)
	}

	// Execute the action
	if err := action.Execute(gs); err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	return nil
}

// CreateGame initializes a new game state with the given ID and players.
// It assigns factions to players in a fixed order for now.
func (m *Manager) CreateGame(id string, playerIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.games[id]; exists {
		return fmt.Errorf("game already exists")
	}

	gs := NewGameState()
	if err := gs.ScoringTiles.InitializeForGame(); err != nil {
		return fmt.Errorf("failed to initialize scoring tiles: %w", err)
	}

	// Initialize bonus cards
	gs.BonusCards.SelectRandomBonusCards(len(playerIDs))

	// Add players without factions initially
	for _, pid := range playerIDs {
		gs.AddPlayer(pid, nil)
	}

	gs.Phase = PhaseFactionSelection

	// Set turn order
	gs.TurnOrder = make([]string, len(playerIDs))
	copy(gs.TurnOrder, playerIDs)
	gs.CurrentPlayerIndex = 0

	m.games[id] = gs
	return nil
}

// SerializeGameState converts GameState to a JSON-friendly format for the frontend
func (m *Manager) SerializeGameState(gameID string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gs := m.games[gameID]
	if gs == nil {
		return nil
	}
	return SerializeState(gs, gameID)
}

// SerializeState converts the game state to a map for JSON response
// This ensures consistent field naming (e.g. currentTurn vs CurrentPlayerIndex)
func SerializeState(gs *GameState, gameID string) map[string]interface{} {
	// Build players map
	players := make(map[string]interface{})
	for playerID, player := range gs.Players {

		var factionType models.FactionType
		if player.Faction != nil {
			factionType = player.Faction.GetType()
		}

		players[playerID] = map[string]interface{}{
			"id":      playerID,
			"name":    playerID, // TODO: Get actual player name
			"faction": factionType,
			"resources": map[string]interface{}{
				"coins":   player.Resources.Coins,
				"workers": player.Resources.Workers,
				"priests": player.Resources.Priests,
				"power": map[string]interface{}{
					"powerI":   player.Resources.Power.Bowl1,
					"powerII":  player.Resources.Power.Bowl2,
					"powerIII": player.Resources.Power.Bowl3,
				},
			},
			"shipping":      player.ShippingLevel,
			"digging":       player.DiggingLevel,
			"victoryPoints": player.VictoryPoints,
			"keys":          player.Keys,
			"townsFormed":   player.TownsFormed,
			"townTiles":     player.TownTiles,
			"cults": map[string]interface{}{
				"0": player.CultPositions[CultFire],
				"1": player.CultPositions[CultWater],
				"2": player.CultPositions[CultEarth],
				"3": player.CultPositions[CultAir],
			},
			"specialActionsUsed": player.SpecialActionsUsed,
		}
	}

	// Build map hexes
	hexes := make(map[string]interface{})
	for _, mapHex := range gs.Map.Hexes {
		key := fmt.Sprintf("%d,%d", mapHex.Coord.Q, mapHex.Coord.R)
		hexData := map[string]interface{}{
			"coord": map[string]int{
				"q": mapHex.Coord.Q,
				"r": mapHex.Coord.R,
			},
			"terrain": mapHex.Terrain,
		}

		if mapHex.Building != nil {
			hexData["building"] = map[string]interface{}{
				"ownerPlayerId": mapHex.Building.PlayerID,
				"faction":       mapHex.Building.Faction,
				"type":          mapHex.Building.Type,
			}
		}

		hexes[key] = hexData
	}

	// Build bridges
	bridges := make([]map[string]interface{}, 0)
	for bridgeKey, ownerID := range gs.Map.Bridges {
		var factionType models.FactionType
		if p, ok := gs.Players[ownerID]; ok && p.Faction != nil {
			factionType = p.Faction.GetType()
		}

		bridges = append(bridges, map[string]interface{}{
			"ownerPlayerId": ownerID,
			"faction":       factionType,
			"fromCoord": map[string]int{
				"q": bridgeKey.H1.Q,
				"r": bridgeKey.H1.R,
			},
			"toCoord": map[string]int{
				"q": bridgeKey.H2.Q,
				"r": bridgeKey.H2.R,
			},
		})
	}

	return map[string]interface{}{
		"id":          gameID,
		"phase":       gs.Phase,
		"currentTurn": gs.CurrentPlayerIndex,
		"players":     players,
		"map": map[string]interface{}{
			"hexes":   hexes,
			"bridges": bridges,
		},
		"turnOrder": gs.TurnOrder,
		"passOrder": gs.PassOrder,
		"round": map[string]interface{}{
			"round": gs.Round,
		},
		"started":  gs.Phase != PhaseSetup,
		"finished": gs.Phase == PhaseEnd,
		"scoringTiles": func() interface{} {
			if gs.ScoringTiles == nil {
				return nil
			}
			return gs.ScoringTiles
		}(),
		"bonusCards": func() interface{} {
			if gs.BonusCards == nil {
				return nil
			}
			return gs.BonusCards
		}(),
		"townTiles": func() interface{} {
			if gs.TownTiles == nil {
				return nil
			}
			return gs.TownTiles
		}(),
		"favorTiles": func() interface{} {
			if gs.FavorTiles == nil {
				return nil
			}
			// Return available tiles and player tiles
			return map[string]interface{}{
				"available":   gs.FavorTiles.Available,
				"playerTiles": gs.FavorTiles.PlayerTiles,
			}
		}(),
		"powerActions": func() interface{} {
			if gs.PowerActions == nil {
				return nil
			}
			return gs.PowerActions
		}(),
		"cultTracks": func() interface{} {
			if gs.CultTracks == nil {
				return nil
			}
			return gs.CultTracks
		}(),
		"finalScoring": func() interface{} {
			if gs.FinalScoring == nil {
				return nil
			}
			return gs.FinalScoring
		}(),
	}
}
