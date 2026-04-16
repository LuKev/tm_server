package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// SelectTownTileAction represents selecting a town tile from pending town formations.
type SelectTownTileAction struct {
	BaseAction
	TileType  models.TownTileType
	AnchorHex *board.Hex
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
	if a.TileType == models.TownTile9Points && gs.RemainingPriestCapacity(a.PlayerID) < 1 {
		return fmt.Errorf("cannot take priest town tile at the 7-priest limit")
	}
	pending := pendingTowns[0]
	if pending.SkippedRiverHex != nil {
		if a.AnchorHex != nil && *a.AnchorHex != *pending.SkippedRiverHex {
			return fmt.Errorf("mermaids river town must use the skipped river hex as the town tile anchor")
		}
		return nil
	}
	if a.AnchorHex == nil {
		return fmt.Errorf("town tile anchor hex is required")
	}

	isValidAnchor := false
	for _, hex := range pending.Hexes {
		if hex != *a.AnchorHex {
			continue
		}
		mapHex := gs.Map.GetHex(hex)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == a.PlayerID {
			isValidAnchor = true
		}
		break
	}
	if !isValidAnchor {
		return fmt.Errorf("anchor hex must be one of the pending town's building hexes")
	}

	return nil
}

// Execute performs the town tile selection.
func (a *SelectTownTileAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	if err := gs.SelectTownTile(a.PlayerID, a.TileType, a.AnchorHex); err != nil {
		return err
	}
	gs.updateAtlanteansStrongholdTown(a.PlayerID)

	if pendingTowns, ok := gs.PendingTownFormations[a.PlayerID]; !ok || len(pendingTowns) == 0 {
		if current := gs.GetCurrentPlayer(); current != nil && current.ID == a.PlayerID {
			gs.NextTurn()
		}
	}

	return nil
}
