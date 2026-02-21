package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// SelectTownTileAction represents selecting a town tile from pending town formations.
type SelectTownTileAction struct {
	BaseAction
	TileType models.TownTileType
}

// GetType returns the action type.
func (a *SelectTownTileAction) GetType() ActionType {
	return ActionSelectTownTile
}

// Validate checks if the town tile selection is valid.
func (a *SelectTownTileAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	pendingTowns, ok := gs.PendingTownFormations[a.PlayerID]
	if !ok || len(pendingTowns) == 0 {
		return fmt.Errorf("no pending town formation for player %s", a.PlayerID)
	}

	if !gs.TownTiles.IsAvailable(a.TileType) {
		return fmt.Errorf("town tile %v is not available", a.TileType)
	}

	return nil
}

// Execute performs the town tile selection.
func (a *SelectTownTileAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	if err := gs.SelectTownTile(a.PlayerID, a.TileType); err != nil {
		return err
	}

	if pendingTowns, ok := gs.PendingTownFormations[a.PlayerID]; !ok || len(pendingTowns) == 0 {
		gs.NextTurn()
	}

	return nil
}
