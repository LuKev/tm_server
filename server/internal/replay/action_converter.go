package replay

import (
	"fmt"

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
		return convertUpgradeAction(playerID, params)

	case ActionTransformAndBuild:
		return convertTransformAndBuildAction(playerID, params)

	case ActionPass:
		return convertPassAction(playerID, params)

	case ActionAdvanceShipping:
		return game.NewAdvanceShippingAction(playerID), nil

	case ActionAdvanceDigging:
		return game.NewAdvanceDiggingAction(playerID), nil

	case ActionSendPriest:
		return convertSendPriestAction(playerID, params)

	case ActionPowerAction:
		// Power actions are complex - they modify state but don't have a dedicated Action type
		// They're usually combined with other actions (e.g., "action ACT6. transform ... build")
		// For now, return nil and let the compound action handle it
		return nil, nil

	case ActionBurnPower:
		// Burning power is usually part of a compound action
		return nil, nil

	case ActionSetup, ActionIncome, ActionLeech, ActionConvert, ActionCultAdvance, ActionWait:
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

func convertUpgradeAction(playerID string, params map[string]string) (game.Action, error) {
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

	return game.NewUpgradeBuildingAction(playerID, hex, buildingType), nil
}

func convertTransformAndBuildAction(playerID string, params map[string]string) (game.Action, error) {
	coordStr, ok := params["coord"]
	if !ok {
		return nil, fmt.Errorf("transform and build action missing coord")
	}

	hex, err := ConvertLogCoordToAxial(coordStr)
	if err != nil {
		return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
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
