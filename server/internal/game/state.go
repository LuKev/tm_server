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
	Map              *TerraMysticaMap
	Players          map[string]*Player
	Round            int
	Phase            GamePhase
	PendingLeechOffers map[string][]*PowerLeechOffer // Key: playerID who can accept
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

// NewGameState creates a new game state with an initialized map
func NewGameState() *GameState {
	return &GameState{
		Map:                NewTerraMysticaMap(),
		Players:            make(map[string]*Player),
		Round:              1,
		Phase:              PhaseSetup,
		PendingLeechOffers: make(map[string][]*PowerLeechOffer),
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
// According to Terra Mystica rules, each adjacent player receives ONE offer equal to
// the sum of power values from ALL their buildings adjacent to the new building
func (gs *GameState) TriggerPowerLeech(buildingHex Hex, buildingPlayerID string, powerValue int) {
	if powerValue <= 0 {
		return
	}

	// Find all adjacent players and calculate total power from their adjacent buildings
	neighbors := buildingHex.Neighbors()
	adjacentPlayerPower := make(map[string]int) // playerID -> total power from their adjacent buildings

	for _, neighbor := range neighbors {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil {
			neighborPlayerID := mapHex.Building.PlayerID
			if neighborPlayerID != buildingPlayerID {
				// Add this building's power value to the player's total
				adjacentPlayerPower[neighborPlayerID] += mapHex.Building.PowerValue
			}
		}
	}

	// Create ONE power leech offer per adjacent player based on their total adjacent power
	for neighborPlayerID, totalPower := range adjacentPlayerPower {
		neighborPlayer := gs.GetPlayer(neighborPlayerID)
		if neighborPlayer == nil {
			continue
		}

		// Create offer based on TOTAL power from all adjacent buildings
		offer := NewPowerLeechOffer(totalPower, buildingPlayerID, neighborPlayer.Resources.Power)
		if offer != nil {
			// Store offer for player to accept/decline
			if gs.PendingLeechOffers[neighborPlayerID] == nil {
				gs.PendingLeechOffers[neighborPlayerID] = []*PowerLeechOffer{}
			}
			gs.PendingLeechOffers[neighborPlayerID] = append(gs.PendingLeechOffers[neighborPlayerID], offer)
		}
	}
}

// GetPendingLeechOffers returns all pending leech offers for a player
func (gs *GameState) GetPendingLeechOffers(playerID string) []*PowerLeechOffer {
	return gs.PendingLeechOffers[playerID]
}

// HasPendingLeechOffers checks if any player has pending leech offers
func (gs *GameState) HasPendingLeechOffers() bool {
	for _, offers := range gs.PendingLeechOffers {
		if len(offers) > 0 {
			return true
		}
	}
	return false
}

// AcceptLeechOffer allows a player to accept a power leech offer
func (gs *GameState) AcceptLeechOffer(playerID string, offerIndex int) error {
	offers := gs.PendingLeechOffers[playerID]
	if offerIndex < 0 || offerIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", offerIndex)
	}

	offer := offers[offerIndex]
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Gain power
	player.Resources.Power.GainPower(offer.Amount)

	// Lose VP
	player.VictoryPoints -= offer.VPCost

	// Remove the offer
	gs.PendingLeechOffers[playerID] = append(offers[:offerIndex], offers[offerIndex+1:]...)

	return nil
}

// DeclineLeechOffer allows a player to decline a power leech offer
func (gs *GameState) DeclineLeechOffer(playerID string, offerIndex int) error {
	offers := gs.PendingLeechOffers[playerID]
	if offerIndex < 0 || offerIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", offerIndex)
	}

	// Simply remove the offer without gaining power or losing VP
	gs.PendingLeechOffers[playerID] = append(offers[:offerIndex], offers[offerIndex+1:]...)

	return nil
}

// ClearPendingLeechOffers clears all pending leech offers for a player
func (gs *GameState) ClearPendingLeechOffers(playerID string) {
	delete(gs.PendingLeechOffers, playerID)
}
