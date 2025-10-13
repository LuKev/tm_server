package game

import (
	"fmt"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// State is an alias to the authoritative models.GameState.
// Keeping the alias in this package lets us evolve engine-specific helpers
// without leaking them to external packages.
type State = models.GameState

// GameState represents the complete game state
type GameState struct {
	Map     *TerraMysticaMap
	Players map[string]*Player
	Round   int
	Phase   GamePhase
}

// GamePhase represents the current phase of the game
type GamePhase int

const (
	PhaseSetup GamePhase = iota
	PhaseAction
	PhaseIncome
	PhaseScoring
	PhaseEnd
)

// Player represents a player in the game
type Player struct {
	ID            string
	Faction       factions.Faction
	Resources     *ResourcePool
	ShippingLevel int
	DiggingLevel  int
	HasPassed     bool
	VictoryPoints int
}

// NewGameState creates a new game state
func NewGameState() *GameState {
	return &GameState{
		Map:     NewTerraMysticaMap(),
		Players: make(map[string]*Player),
		Round:   1,
		Phase:   PhaseSetup,
	}
}

// AddPlayer adds a player to the game
func (gs *GameState) AddPlayer(playerID string, faction factions.Faction) error {
	if _, exists := gs.Players[playerID]; exists {
		return fmt.Errorf("player already exists: %s", playerID)
	}

	player := &Player{
		ID:            playerID,
		Faction:       faction,
		Resources:     NewResourcePool(faction.GetStartingResources()),
		ShippingLevel: 0,
		DiggingLevel:  0,
		HasPassed:     false,
		VictoryPoints: 20, // Starting VP
	}

	gs.Players[playerID] = player
	return nil
}

// GetPlayer returns a player by ID
func (gs *GameState) GetPlayer(playerID string) *Player {
	return gs.Players[playerID]
}

// IsAdjacentToPlayerBuilding checks if a hex is adjacent to any of the player's buildings
// According to Terra Mystica rules, adjacency can be:
// 1. Direct adjacency (shared edge or connected via bridge)
// 2. Indirect adjacency (connected via river navigation with shipping)
func (gs *GameState) IsAdjacentToPlayerBuilding(targetHex Hex, playerID string) bool {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return false
	}

	// Check if player has any buildings at all (for first dwelling placement)
	hasAnyBuilding := false
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			hasAnyBuilding = true
			break
		}
	}
	
	// If player has no buildings yet, allow placement anywhere (first dwelling)
	if !hasAnyBuilding {
		return true
	}

	// Check adjacency to each of the player's buildings
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			buildingHex := mapHex.Coord
			
			// Check direct adjacency (includes bridges)
			if gs.Map.IsDirectlyAdjacent(targetHex, buildingHex) {
				return true
			}
			
			// Check indirect adjacency via shipping (river navigation)
			if player.ShippingLevel > 0 {
				if gs.Map.IsIndirectlyAdjacent(targetHex, buildingHex, player.ShippingLevel) {
					return true
				}
			}
		}
	}

	// TODO: Check for special abilities (Witches flying, Fakirs carpet, Dwarves tunneling)

	return false
}

// TriggerPowerLeech triggers power leech offers for all adjacent players
// This is called when a building is placed or upgraded
func (gs *GameState) TriggerPowerLeech(buildingHex Hex, buildingPlayerID string, powerValue int) {
	if powerValue <= 0 {
		return
	}

	// Find all adjacent players
	neighbors := buildingHex.Neighbors()
	adjacentPlayers := make(map[string]bool)

	for _, neighbor := range neighbors {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil {
			neighborPlayerID := mapHex.Building.PlayerID
			if neighborPlayerID != buildingPlayerID {
				adjacentPlayers[neighborPlayerID] = true
			}
		}
	}

	// Create power leech offers for each adjacent player
	for neighborPlayerID := range adjacentPlayers {
		neighborPlayer := gs.GetPlayer(neighborPlayerID)
		if neighborPlayer == nil {
			continue
		}

		// Create offer based on building value and player's power capacity
		offer := NewPowerLeechOffer(powerValue, buildingPlayerID, neighborPlayer.Resources.Power)
		if offer != nil {
			// TODO: Phase 6.1 - Store offer for player to accept/decline
			// For now, we'll just create the offer structure
			_ = offer
		}
	}
}
