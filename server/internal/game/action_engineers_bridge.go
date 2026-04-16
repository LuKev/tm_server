package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

const engineersBridgeWorkerCost = 2
const architectsBridgePriestCost = 1

// EngineersBridgeAction represents the reusable bridge action used by
// Engineers, Atlanteans, and Architects.
type EngineersBridgeAction struct {
	BaseAction
	BridgeHex1 board.Hex
	BridgeHex2 board.Hex
}

// NewEngineersBridgeAction creates a new Engineers bridge action.
func NewEngineersBridgeAction(playerID string, hex1, hex2 board.Hex) *EngineersBridgeAction {
	return &EngineersBridgeAction{
		BaseAction: BaseAction{
			Type:     ActionEngineersBridge,
			PlayerID: playerID,
		},
		BridgeHex1: hex1,
		BridgeHex2: hex2,
	}
}

// GetType returns the action type.
func (a *EngineersBridgeAction) GetType() ActionType {
	return ActionEngineersBridge
}

// Validate checks whether the bridge action is legal.
func (a *EngineersBridgeAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	if player.Faction == nil || (player.Faction.GetType() != models.FactionEngineers && player.Faction.GetType() != models.FactionAtlanteans && player.Faction.GetType() != models.FactionArchitects) {
		return fmt.Errorf("only Engineers, Atlanteans, or Architects can use this action")
	}
	if player.Faction.GetType() == models.FactionArchitects {
		if player.Resources.Priests < architectsBridgePriestCost {
			return fmt.Errorf("not enough priests: need %d, have %d", architectsBridgePriestCost, player.Resources.Priests)
		}
	} else if player.Resources.Workers < engineersBridgeWorkerCost {
		return fmt.Errorf("not enough workers: need %d, have %d", engineersBridgeWorkerCost, player.Resources.Workers)
	}
	if player.BridgesBuilt >= 3 {
		return fmt.Errorf("player has already built 3 bridges (maximum)")
	}

	h1, err := gs.ValidateHex(a.BridgeHex1)
	if err != nil {
		return err
	}
	h2, err := gs.ValidateHex(a.BridgeHex2)
	if err != nil {
		return err
	}
	if h1.Terrain == models.TerrainRiver || h2.Terrain == models.TerrainRiver {
		return fmt.Errorf("bridge endpoints must be land hexes")
	}
	if gs.Map.HasBridge(a.BridgeHex1, a.BridgeHex2) {
		return fmt.Errorf("bridge already exists between these hexes")
	}

	endpoint1Owned := h1.Building != nil && h1.Building.PlayerID == a.PlayerID
	endpoint2Owned := h2.Building != nil && h2.Building.PlayerID == a.PlayerID
	if !endpoint1Owned && !endpoint2Owned {
		return fmt.Errorf("bridge must connect to at least one of your structures")
	}

	return nil
}

// Execute performs the bridge action.
func (a *EngineersBridgeAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	if player.Faction.GetType() == models.FactionArchitects {
		player.Resources.Priests -= architectsBridgePriestCost
	} else {
		player.Resources.Workers -= engineersBridgeWorkerCost
	}

	if err := gs.Map.BuildBridge(a.BridgeHex1, a.BridgeHex2, a.PlayerID); err != nil {
		if player.Faction.GetType() == models.FactionArchitects {
			player.Resources.Priests += architectsBridgePriestCost
		} else {
			player.Resources.Workers += engineersBridgeWorkerCost
		}
		return fmt.Errorf("failed to build bridge: %w", err)
	}

	player.BridgesBuilt++
	gs.CheckForTownFormation(a.PlayerID, a.BridgeHex1)
	gs.CheckForTownFormation(a.PlayerID, a.BridgeHex2)
	gs.updateAtlanteansStrongholdTown(a.PlayerID)
	gs.NextTurn()
	return nil
}
