package replay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

// ConvertLogEntryToAction converts a parsed log entry to a game.Action
func ConvertLogEntryToAction(entry *LogEntry, gs *game.GameState) (game.Action, error) {
	if entry.IsComment {
		return nil, nil // Skip comment entries
	}

	if entry.Action == "" {
		return nil, nil // Skip entries with no action
	}

	// Parse the action string
	actionType, params, err := ParseAction(entry.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action: %v", err)
	}

	// Get player ID from faction (using faction name as player ID)
	playerID := entry.Faction.String()

	// Check if we're in setup phase
	isSetup := (gs != nil && gs.Phase == game.PhaseSetup)

	// Convert based on action type
	switch actionType {
	case ActionBuild:
		return convertBuildAction(playerID, params, isSetup)

	case ActionUpgrade:
		return convertUpgradeAction(playerID, params, gs)

	case ActionTransformAndBuild:
		return convertTransformAndBuildAction(playerID, params, gs)

	case ActionPass:
		return convertPassAction(playerID, params)

	case ActionAdvanceShipping:
		return game.NewAdvanceShippingAction(playerID), nil

	case ActionAdvanceDigging:
		return game.NewAdvanceDiggingAction(playerID), nil

	case ActionSendPriest:
		return convertSendPriestAction(playerID, params)

	case ActionPowerAction:
		// Power actions can be standalone or combined with build/transform
		// e.g., "action ACT5. build F3" or "action ACT6. transform F2 to gray. build D4"
		// Also handles bonus card actions: "action BON1. build C5"
		actionTypeStr, ok := params["action_type"]
		if !ok {
			return nil, fmt.Errorf("power action missing action_type")
		}

		// Check if this is a bonus card action (BON1-10) vs power action (ACT1-6)
		if strings.HasPrefix(actionTypeStr, "BON") {
			// Bonus card actions like BON1 (spade bonus)
			// These work similarly to power actions but come from bonus cards
			// For now, treat BON1 like a spade power action
			coordStr, hasCoord := params["coord"]
			if hasCoord {
				hex, err := ConvertLogCoordToAxial(coordStr)
				if err != nil {
					return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
				}
				// BON1 provides a free spade transform
				// Use TransformAndBuild action
				return game.NewTransformAndBuildAction(playerID, hex, true), nil
			}
			// Standalone bonus card action (not combined with build)
			// Skip for now - these are typically informational
			return nil, nil
		}

		powerActionType, err := ParsePowerActionType(actionTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid power action type: %v", err)
		}

		// Check if this is combined with build/transform
		coordStr, hasCoord := params["coord"]
		if hasCoord {
			// Power action with build or transform+build
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
			}

			// Check if there's a transform
			_, hasTransform := params["transform_coord"]
			if hasTransform || powerActionType == game.PowerActionSpade1 || powerActionType == game.PowerActionSpade2 {
				// Power action provides free spades for transform+build
				return game.NewPowerActionWithTransform(playerID, powerActionType, hex, true), nil
			}

			// Just a build (e.g., with bridge power action)
			// For now, treat as error - need to implement this case
			return nil, fmt.Errorf("power action with build but no transform not yet implemented")
		}

		// Standalone power action (ACT1=bridge, ACT2=priest, ACT3=workers, ACT4=coins)
		return game.NewPowerAction(playerID, powerActionType), nil

	case ActionBurnPower:
		// Burning power is usually part of a compound action
		return nil, nil

	case ActionLeech:
		return convertLeechAction(playerID, params, gs)

	case ActionSetup, ActionIncome, ActionConvert, ActionCultAdvance, ActionWait:
		// These are state changes, not player actions
		return nil, nil

	case ActionUnknown:
		// Unknown actions - might be valid state changes we don't need to execute
		return nil, nil

	default:
		return nil, fmt.Errorf("unhandled action type: %v", actionType)
	}
}

func convertBuildAction(playerID string, params map[string]string, isSetup bool) (game.Action, error) {
	coordStr, ok := params["coord"]
	if !ok {
		return nil, fmt.Errorf("build action missing coord")
	}

	hex, err := ConvertLogCoordToAxial(coordStr)
	if err != nil {
		return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
	}

	// During setup, use setup dwelling action (no cost, no adjacency)
	if isSetup {
		return game.NewSetupDwellingAction(playerID, hex), nil
	}

	// During normal gameplay, building a dwelling on home terrain (no transformation needed)
	return game.NewTransformAndBuildAction(playerID, hex, true), nil
}

func convertUpgradeAction(playerID string, params map[string]string, gs *game.GameState) (game.Action, error) {
	coordStr, ok := params["coord"]
	if !ok {
		return nil, fmt.Errorf("upgrade action missing coord")
	}

	buildingStr, ok := params["building"]
	if !ok {
		return nil, fmt.Errorf("upgrade action missing building type")
	}

	hex, err := ConvertLogCoordToAxial(coordStr)
	if err != nil {
		return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
	}

	buildingType, err := ParseBuildingType(buildingStr)
	if err != nil {
		return nil, fmt.Errorf("invalid building type %s: %v", buildingStr, err)
	}

	upgradeAction := game.NewUpgradeBuildingAction(playerID, hex, buildingType)

	// If there's a favor tile specified, this is a compound action:
	// upgrade + select favor tile. Execute both immediately.
	if favorTileStr, hasFavorTile := params["favor_tile"]; hasFavorTile {
		// Execute upgrade first
		if err := upgradeAction.Validate(gs); err != nil {
			return nil, fmt.Errorf("upgrade validation failed: %v", err)
		}
		if err := upgradeAction.Execute(gs); err != nil {
			return nil, fmt.Errorf("upgrade execution failed: %v", err)
		}

		// Now create and execute favor tile selection
		favorTileType, err := ParseFavorTile(favorTileStr)
		if err != nil {
			return nil, fmt.Errorf("invalid favor tile %s: %v", favorTileStr, err)
		}

		favorAction := &game.SelectFavorTileAction{
			BaseAction: game.BaseAction{
				Type:     game.ActionSelectFavorTile,
				PlayerID: playerID,
			},
			TileType: favorTileType,
		}

		if err := favorAction.Validate(gs); err != nil {
			return nil, fmt.Errorf("favor tile validation failed: %v", err)
		}
		if err := favorAction.Execute(gs); err != nil {
			return nil, fmt.Errorf("favor tile execution failed: %v", err)
		}

		// Both actions executed, return nil to skip normal execution
		return nil, nil
	}

	return upgradeAction, nil
}

func convertTransformAndBuildAction(playerID string, params map[string]string, gs *game.GameState) (game.Action, error) {
	coordStr, ok := params["coord"]
	if !ok {
		return nil, fmt.Errorf("transform and build action missing coord")
	}

	hex, err := ConvertLogCoordToAxial(coordStr)
	if err != nil {
		return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
	}

	// Handle burning power if present
	if burnStr, hasBurn := params["burn"]; hasBurn {
		burnAmount, err := strconv.Atoi(burnStr)
		if err != nil {
			return nil, fmt.Errorf("invalid burn amount %s: %v", burnStr, err)
		}
		player := gs.GetPlayer(playerID)
		if player == nil {
			return nil, fmt.Errorf("player not found: %s", playerID)
		}
		// Debug: print power state before burning
		fmt.Printf("DEBUG: Player %s attempting to burn %d power. Current bowls: %d/%d/%d\n",
			playerID, burnAmount, player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
		// Burn power before the main action
		if err := player.Resources.BurnPower(burnAmount); err != nil {
			return nil, fmt.Errorf("failed to burn power: %v", err)
		}
	}

	// Handle power action if present (e.g., ACT6 for 2 free spades)
	if powerActionStr, hasPowerAction := params["power_action"]; hasPowerAction {
		powerActionType, err := ParsePowerActionType(powerActionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid power action %s: %v", powerActionStr, err)
		}

		// For spade power actions, create a PowerAction with transform
		if powerActionType == game.PowerActionSpade1 || powerActionType == game.PowerActionSpade2 {
			return game.NewPowerActionWithTransform(playerID, powerActionType, hex, true), nil
		}

		// Other power actions - not expected in transform-and-build context
		return nil, fmt.Errorf("unexpected power action %s in transform-and-build", powerActionStr)
	}

	// Check if we need to transform (has transform_coord or spades)
	_, hasTransform := params["transform_coord"]
	_, hasSpades := params["spades"]

	if !hasTransform && !hasSpades {
		// No transformation, just building
		return game.NewTransformAndBuildAction(playerID, hex, true), nil
	}

	// Has transformation - always build dwelling after transform
	return game.NewTransformAndBuildAction(playerID, hex, true), nil
}

func convertPassAction(playerID string, params map[string]string) (game.Action, error) {
	bonusStr, ok := params["bonus"]
	if !ok {
		// Pass without selecting bonus (might happen in some phases)
		return game.NewPassAction(playerID, nil), nil
	}

	// Parse bonus tile - convert string to BonusCardType
	bonusCard, err := ParseBonusCard(bonusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid bonus card %s: %v", bonusStr, err)
	}

	return game.NewPassAction(playerID, &bonusCard), nil
}

func convertLeechAction(playerID string, params map[string]string, gs *game.GameState) (game.Action, error) {
	// Leech actions in the log are informational - they show power being gained
	// The log entry's power bowls already reflect the final state after leeching
	// We don't need to execute the leech action; we just need to update the power state
	// to match what the log says it should be.
	//
	// The validator will check that the final power state matches the log entry
	// after this function returns nil (no action to execute).
	//
	// However, we DO need to manually adjust the power to match the log, because
	// the leech offers were created by previous builds, but we're not tracking them
	// across entries in the replay.
	//
	// For now, skip executing leech actions and let state validation catch discrepancies
	return nil, nil
}

func convertSendPriestAction(playerID string, params map[string]string) (game.Action, error) {
	cultStr, ok := params["cult"]
	if !ok {
		return nil, fmt.Errorf("send priest action missing cult track")
	}

	cultTrack, err := ParseCultTrack(cultStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cult track %s: %v", cultStr, err)
	}

	// Convert models.CultType to game.CultTrack
	var gameCultTrack game.CultTrack
	switch cultTrack {
	case models.CultFire:
		gameCultTrack = game.CultFire
	case models.CultWater:
		gameCultTrack = game.CultWater
	case models.CultEarth:
		gameCultTrack = game.CultEarth
	case models.CultAir:
		gameCultTrack = game.CultAir
	}

	// Default to advancing 1 space (standard priest send)
	return &game.SendPriestToCultAction{
		BaseAction: game.BaseAction{
			Type:     game.ActionSendPriestToCult,
			PlayerID: playerID,
		},
		Track:         gameCultTrack,
		SpacesToClimb: 1,
	}, nil
}
