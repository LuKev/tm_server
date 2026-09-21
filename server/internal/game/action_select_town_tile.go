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

	if !gs.TownTiles.IsAvailable(a.TileType) {
		return fmt.Errorf("town tile %v is not available", a.TileType)
	}
	choices := gs.TownFormationChoices(a.PlayerID)
	index, err := gs.townFormationIndex(a.PlayerID, choices, a.AnchorHex)
	if err != nil {
		return err
	}
	pending := choices[index]
	if a.AnchorHex == nil && pending.SkippedRiverHex == nil {
		return fmt.Errorf("town tile anchor hex is required")
	}
	if pending.CanBeDelayed {
		current := gs.GetCurrentPlayer()
		ownsTurn := current != nil && current.ID == a.PlayerID || gs.PendingFreeActionsPlayerID == a.PlayerID
		if gs.Phase != PhaseAction || !ownsTurn || gs.HasBlockingPendingLeechOffers() {
			return fmt.Errorf("delayed Mermaid town requires the owner's action turn, after leech resolves")
		}
	}

	return nil
}

// An anchor identifies which simultaneously founded town the player wants to
// reward first. River anchors additionally identify distinct Mermaid choices.
func (gs *GameState) townFormationIndex(playerID string, choices []*PendingTownFormation, anchor *board.Hex) (int, error) {
	mandatory := hasImmediatePendingTownSelection(choices)
	for i, pending := range choices {
		if pending == nil || mandatory && pending.CanBeDelayed {
			continue
		}
		if anchor == nil {
			return i, nil
		} // Legacy direct-call default.
		if pending.SkippedRiverHex != nil {
			if *pending.SkippedRiverHex == *anchor {
				return i, nil
			}
			continue
		}
		for _, hex := range pending.Hexes {
			mapHex := gs.Map.GetHex(hex)
			if hex == *anchor && mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("anchor does not identify an eligible pending town")
}

// Execute performs the town tile selection.
func (a *SelectTownTileAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	formations := gs.TownFormationChoices(a.PlayerID)
	index, _ := gs.townFormationIndex(a.PlayerID, formations, a.AnchorHex)
	delayedBeforeMain := formations[index].CanBeDelayed && gs.PendingFreeActionsPlayerID == ""

	if err := gs.SelectTownTile(a.PlayerID, a.TileType, a.AnchorHex); err != nil {
		return err
	}
	if delayedBeforeMain && gs.PendingTownCultTopChoice != nil {
		gs.PendingTownCultTopChoice.ContinueMainAction = true
	}
	gs.updateAtlanteansStrongholdTown(a.PlayerID)

	// NextTurn itself waits for mandatory rewards. An unrelated delayed town
	// must not keep the completed main action open for another main action.
	if current := gs.GetCurrentPlayer(); current != nil && current.ID == a.PlayerID && !delayedBeforeMain {
		gs.NextTurn()
	}

	return nil
}
