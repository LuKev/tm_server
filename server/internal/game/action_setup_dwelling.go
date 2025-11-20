package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// SetupDwellingAction represents placing an initial dwelling during game setup
// Setup dwellings are placed without cost and without adjacency requirements
type SetupDwellingAction struct {
	BaseAction
	Hex board.Hex
}

// NewSetupDwellingAction creates a new setup dwelling action
func NewSetupDwellingAction(playerID string, hex board.Hex) *SetupDwellingAction {
	return &SetupDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionSetupDwelling,
			PlayerID: playerID,
		},
		Hex: hex,
	}
}

// Validate checks if the setup dwelling placement is valid
func (a *SetupDwellingAction) Validate(gs *GameState) error {
	// Must be in setup phase
	if gs.Phase != PhaseSetup {
		return fmt.Errorf("can only place setup dwellings during setup phase")
	}

	player, exists := gs.Players[a.PlayerID]
	if !exists {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if hex exists
	hexData := gs.Map.GetHex(a.Hex)
	if hexData == nil {
		return fmt.Errorf("hex does not exist: %v", a.Hex)
	}

	// Cannot be river
	if hexData.Terrain == models.TerrainRiver {
		return fmt.Errorf("cannot build on river (unless faction has water building ability)")
	}

	// Check if hex already has a building
	if hexData.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Must be player's home terrain
	if hexData.Terrain != player.Faction.GetHomeTerrain() {
		return fmt.Errorf("setup dwellings must be placed on home terrain")
	}

	return nil
}

// Execute places the dwelling on the map during setup
func (a *SetupDwellingAction) Execute(gs *GameState) error {
	player, exists := gs.Players[a.PlayerID]
	if !exists {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Place the dwelling on the map
	hexData := gs.Map.GetHex(a.Hex)
	hexData.Building = &models.Building{
		Type:       models.BuildingDwelling,
		PlayerID:   a.PlayerID,
		Faction:    player.Faction.GetType(),
		PowerValue: 1, // Dwellings have power value of 1
	}

	return nil
}
