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
		return convertSendPriestAction(playerID, params, entry, gs)

	case ActionPowerAction:
		// Power actions can be standalone or combined with build/transform
		// e.g., "action ACT5. build F3" or "action ACT6. transform F2 to gray. build D4"
		// Also handles bonus card actions: "action BON1. build C5"
		actionTypeStr, ok := params["action_type"]
		if !ok {
			return nil, fmt.Errorf("power action missing action_type")
		}

		// Handle burning power if present (e.g., "burn 3. action ACT2")
		if burnStr, hasBurn := params["burn"]; hasBurn {
			burnAmount, err := strconv.Atoi(burnStr)
			if err != nil {
				return nil, fmt.Errorf("invalid burn amount %s: %v", burnStr, err)
			}
			player := gs.GetPlayer(playerID)
			if player == nil {
				return nil, fmt.Errorf("player not found: %s", playerID)
			}
			// Burn power before the main action
			if err := player.Resources.BurnPower(burnAmount); err != nil {
				return nil, fmt.Errorf("failed to burn power: %v", err)
			}
		}

		// Check if this is a bonus card action (BON1-10) vs power action (ACT1-6) vs stronghold action (ACTW, ACTA, etc.)
		if strings.HasPrefix(actionTypeStr, "BON") {
			// BON1 provides 1 free spade for transformation
			if actionTypeStr == "BON1" {
				coordStr, hasCoord := params["coord"]
				if hasCoord {
					hex, err := ConvertLogCoordToAxial(coordStr)
					if err != nil {
						return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
					}
					// BON1 provides a free spade transform + build
					return game.NewBonusCardSpadeAction(playerID, hex, true), nil
				}
				// Standalone BON1 action (not combined with build) - skip
				return nil, nil
			}
			// Other bonus card actions (BON2-BON10) - skip for now
			return nil, nil
		}

		// Check if this is a stronghold special action (ACTW=Witches' Ride, ACTA=Auren, etc.)
		if actionTypeStr == "ACTW" {
			// Witches' Ride: Build dwelling on any Forest hex
			coordStr, hasCoord := params["coord"]
			if !hasCoord {
				return nil, fmt.Errorf("Witches' Ride action missing coord")
			}
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, fmt.Errorf("invalid coordinate %s: %v", coordStr, err)
			}
			return game.NewWitchesRideAction(playerID, hex), nil
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
		fmt.Printf("DEBUG: Creating standalone power action %v for %s\n", powerActionType, playerID)
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

	// Check if validation should be skipped (for compound convert+upgrade actions)
	// Resources are already synced by the validator, so we just place the building
	skipValidation := params["skip_validation"] == "true"
	
	// If there's a favor tile specified, this is a compound action:
	// upgrade + select favor tile. Execute both immediately.
	if favorTileStr, hasFavorTile := params["favor_tile"]; hasFavorTile {
		fmt.Printf("DEBUG: Compound upgrade+favor action for %s: upgrade to %s, then take %s\n",
			playerID, buildingStr, favorTileStr)

		// Execute upgrade first
		if skipValidation {
			fmt.Printf("DEBUG: Manually placing building %s at %v for %s\n", buildingStr, hex, playerID)
			// Resources already synced - just place the building manually
			mapHex := gs.Map.GetHex(hex)
			if mapHex == nil {
				return nil, fmt.Errorf("hex does not exist: %v", hex)
			}
			player := gs.GetPlayer(playerID)
			if player == nil {
				return nil, fmt.Errorf("player not found: %s", playerID)
			}
			fmt.Printf("DEBUG: mapHex found, current building: %v\n", mapHex.Building)

			// Get new power value
			var newPowerValue int
			switch buildingType {
			case models.BuildingTradingHouse:
				newPowerValue = 2
			case models.BuildingTemple:
				newPowerValue = 2
			case models.BuildingSanctuary:
				newPowerValue = 3
			case models.BuildingStronghold:
				newPowerValue = 3
			}

			// Place building
			mapHex.Building = &models.Building{
				Type:       buildingType,
				Faction:    player.Faction.GetType(),
				PlayerID:   playerID,
				PowerValue: newPowerValue,
			}
			fmt.Printf("DEBUG: Building placed: %v\n", mapHex.Building)

			// Set stronghold ability if upgrading to stronghold
			if buildingType == models.BuildingStronghold {
				player.HasStrongholdAbility = true
			}

			// Set pending favor tile selection (if Temple or Sanctuary)
			if buildingType == models.BuildingTemple || buildingType == models.BuildingSanctuary {
				tileCount := game.GetFavorTileCount(player.Faction.GetType())
				gs.PendingFavorTileSelection = &game.PendingFavorTileSelection{
					PlayerID: playerID,
					Count:    tileCount,
				}
			}
			fmt.Printf("DEBUG: Building placed manually, PendingFavorTileSelection: %v\n", gs.PendingFavorTileSelection != nil)
		} else {
			// Normal execution
			if err := upgradeAction.Validate(gs); err != nil {
				return nil, fmt.Errorf("upgrade validation failed: %v", err)
			}
			if err := upgradeAction.Execute(gs); err != nil {
				return nil, fmt.Errorf("upgrade execution failed: %v", err)
			}
			fmt.Printf("DEBUG: Upgrade executed, PendingFavorTileSelection: %v\n", gs.PendingFavorTileSelection != nil)
		}

		// Now create and execute favor tile selection
		favorTileType, err := ParseFavorTile(favorTileStr)
		if err != nil {
			return nil, fmt.Errorf("invalid favor tile %s: %v", favorTileStr, err)
		}
		fmt.Printf("DEBUG: Parsed favor tile %s as type %v\n", favorTileStr, favorTileType)

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
		fmt.Printf("DEBUG: Favor tile action validated, now executing\n")
		if err := favorAction.Execute(gs); err != nil {
			return nil, fmt.Errorf("favor tile execution failed: %v", err)
		}
		fmt.Printf("DEBUG: Favor tile action executed successfully\n")

		// Both actions executed, return nil to skip normal execution
		return nil, nil
	}

	// If skip_validation is set but there's no favor tile, manually place the building
	if skipValidation {
		fmt.Printf("DEBUG: Manually placing building %s at %v for %s (no favor tile)\n", buildingStr, hex, playerID)
		mapHex := gs.Map.GetHex(hex)
		if mapHex == nil {
			return nil, fmt.Errorf("hex does not exist: %v", hex)
		}
		player := gs.GetPlayer(playerID)
		if player == nil {
			return nil, fmt.Errorf("player not found: %s", playerID)
		}

		// Get new power value
		var newPowerValue int
		switch buildingType {
		case models.BuildingTradingHouse:
			newPowerValue = 2
		case models.BuildingTemple:
			newPowerValue = 2
		case models.BuildingSanctuary:
			newPowerValue = 3
		case models.BuildingStronghold:
			newPowerValue = 3
		}

		// Place building
		mapHex.Building = &models.Building{
			Type:       buildingType,
			Faction:    player.Faction.GetType(),
			PlayerID:   playerID,
			PowerValue: newPowerValue,
		}

		// Set stronghold ability if upgrading to stronghold
		if buildingType == models.BuildingStronghold {
			player.HasStrongholdAbility = true
		}

		// Trigger power leech for adjacent players
		gs.TriggerPowerLeech(hex, playerID)

		// Return nil to indicate action was executed inline
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
	if powerActionStr, hasPowerAction := params["action_type"]; hasPowerAction {
		powerActionType, err := ParsePowerActionType(powerActionStr)
		if err != nil {
			return nil, fmt.Errorf("invalid power action %s: %v", powerActionStr, err)
		}

		// For spade power actions, check if transform and build are on different hexes
		// Example: "action ACT6. transform F2 to gray. build D4"
		// This means: transform F2 (using 2 free spades), then build dwelling on D4
		if powerActionType == game.PowerActionSpade1 || powerActionType == game.PowerActionSpade2 {
			if transformCoordStr, hasTransformCoord := params["transform_coord"]; hasTransformCoord {
				transformHex, err := ConvertLogCoordToAxial(transformCoordStr)
				if err != nil {
					return nil, fmt.Errorf("invalid transform coordinate %s: %v", transformCoordStr, err)
				}

				// If transform and build are on different hexes, manually transform first
				if transformHex.Q != hex.Q || transformHex.R != hex.R {
					// Manually transform the hex without using actions
					// The power action provides free spades, so just transform the terrain
					player := gs.GetPlayer(playerID)
					if player == nil {
						return nil, fmt.Errorf("player not found: %s", playerID)
					}

					// Get target terrain from params (e.g., "gray", "green", etc.)
					targetTerrainStr, hasTargetTerrain := params["transform_color"]
					if !hasTargetTerrain {
						return nil, fmt.Errorf("transform action missing target terrain color")
					}
					targetTerrain, err := ParseTerrainColor(targetTerrainStr)
					if err != nil {
						return nil, fmt.Errorf("invalid target terrain %s: %v", targetTerrainStr, err)
					}

					buildHexTerrain := gs.Map.GetHex(hex).Terrain
					homeTerrain := player.Faction.GetHomeTerrain()
					fmt.Printf("DEBUG: Player %s ACT%d - Transforming %v (from %v to %v), then building on %v (terrain: %v, home: %v)\n",
						playerID, powerActionType+4,
						transformHex, gs.Map.GetHex(transformHex).Terrain, targetTerrain, hex,
						buildHexTerrain, homeTerrain)

					// Transform the transform_coord hex to target terrain (using free spades from power action)
					if err := gs.Map.TransformTerrain(transformHex, targetTerrain); err != nil {
						return nil, fmt.Errorf("failed to transform terrain: %w", err)
					}

					// If build hex also needs transformation, transform it to home terrain (also using free spades)
					if buildHexTerrain != homeTerrain {
						if err := gs.Map.TransformTerrain(hex, homeTerrain); err != nil {
							return nil, fmt.Errorf("failed to transform build hex: %w", err)
						}
					}

					// Spend the power from bowl 3 (it was already burned from bowl 2 to bowl 3 earlier)
					powerCost := game.GetPowerCost(powerActionType)
					if err := player.Resources.Power.SpendPower(powerCost); err != nil {
						return nil, fmt.Errorf("failed to spend power for power action: %w", err)
					}

					// Mark power action as used
					gs.PowerActions.MarkUsed(powerActionType)

					// Now just build the dwelling (no transformation needed since we already did it)
					// Use TransformAndBuildAction but the terrain is already correct
					return game.NewTransformAndBuildAction(playerID, hex, true), nil
				}
			}

			// Transform and build on same hex
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

func convertSendPriestAction(playerID string, params map[string]string, entry *LogEntry, gs *game.GameState) (game.Action, error) {
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

	// Calculate spaces to climb from current to target position
	// Use game state to get current position, and entry to get target position
	spacesToClimb := 1 // Default
	if gs != nil && entry != nil {
		currentPos := gs.CultTracks.GetPosition(playerID, gameCultTrack)
		var targetPos int
		switch gameCultTrack {
		case game.CultFire:
			targetPos = entry.CultTracks.Fire
		case game.CultWater:
			targetPos = entry.CultTracks.Water
		case game.CultEarth:
			targetPos = entry.CultTracks.Earth
		case game.CultAir:
			targetPos = entry.CultTracks.Air
		}
		spacesToClimb = targetPos - currentPos
		if spacesToClimb < 1 {
			spacesToClimb = 1 // Minimum 1 space
		}
		if spacesToClimb > 3 {
			spacesToClimb = 3 // Maximum 3 spaces per priest
		}
	}

	return &game.SendPriestToCultAction{
		BaseAction: game.BaseAction{
			Type:     game.ActionSendPriestToCult,
			PlayerID: playerID,
		},
		Track:         gameCultTrack,
		SpacesToClimb: spacesToClimb,
	}, nil
}
