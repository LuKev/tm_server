package replay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ConvertLogEntryToAction converts a log entry to a game action
func ConvertLogEntryToAction(entry *LogEntry, gs *game.GameState) (game.Action, error) {
	if entry.IsComment {
		return nil, nil
	}

	playerID := entry.GetPlayerID()
	actionType, params, err := ParseAction(entry.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to parse action: %v", err)
	}

	// Check if we're in setup phase
	isSetup := (gs != nil && gs.Phase == game.PhaseSetup)

	// Convert based on action type
	switch actionType {
	case ActionBuild:
		return convertBuildAction(playerID, params, isSetup, entry, gs)

	case ActionUpgrade:
		if strings.Contains(entry.Action, "+FAV") && strings.Contains(entry.Action, "+TW") {
			fmt.Printf("DEBUG action_converter: ActionUpgrade with +FAV and +TW for %s\n", playerID)
		}
		return convertUpgradeAction(playerID, params, entry, gs)

	case ActionTransform:
		// Pure transform action (e.g., "dig 1. transform H8 to green")
		// The dig cost is already paid (reflected in deltas), we just need to transform the terrain
		// Return nil - the validator will handle the transform via executeTransformFromAction
		return nil, nil

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
			
			// BON2 provides cult advancement
			if actionTypeStr == "BON2" {
				cultTrackStr, hasCultTrack := params["cult_track"]
				if hasCultTrack {
					// Parse cult track
					cultTrack, err := ParseCultTrack(cultTrackStr)
					if err != nil {
						return nil, fmt.Errorf("invalid cult track %s: %v", cultTrackStr, err)
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
					
					// Create special action to advance cult track by 1
					return &game.SpecialAction{
						BaseAction: game.BaseAction{
							Type:     game.ActionSpecialAction,
							PlayerID: playerID,
						},
						ActionType: game.SpecialActionBonusCardCultAdvance,
						CultTrack:  &gameCultTrack,
					}, nil
				}
				// Standalone BON2 without cult track - skip
				return nil, nil
			}
			
			// Other bonus card actions (BON3-BON10) - skip for now
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

		// Check if this is a bridge action with coordinates
		bridgeFrom, hasBridgeFrom := params["bridge_from"]
		bridgeTo, hasBridgeTo := params["bridge_to"]
		if powerActionType == game.PowerActionBridge && hasBridgeFrom && hasBridgeTo {
			// Parse bridge endpoints
			hex1, err := ConvertLogCoordToAxial(bridgeFrom)
			if err != nil {
				return nil, fmt.Errorf("invalid bridge coordinate %s: %v", bridgeFrom, err)
			}
			hex2, err := ConvertLogCoordToAxial(bridgeTo)
			if err != nil {
				return nil, fmt.Errorf("invalid bridge coordinate %s: %v", bridgeTo, err)
			}

			// Execute the power action to pay the cost
			powerAction := game.NewPowerAction(playerID, powerActionType)
			if err := powerAction.Validate(gs); err != nil {
				return nil, fmt.Errorf("bridge power action validation failed: %v", err)
			}
			if err := powerAction.Execute(gs); err != nil {
				return nil, fmt.Errorf("bridge power action execution failed: %v", err)
			}

			// Build the bridge
			if err := gs.Map.BuildBridge(hex1, hex2); err != nil {
				return nil, fmt.Errorf("failed to build bridge: %v", err)
			}

			// Return nil to indicate action was executed inline
			return nil, nil
		}

		// Standalone power action (ACT2=priest, ACT3=workers, ACT4=coins, or ACT1=bridge without coords)
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

func convertBuildAction(playerID string, params map[string]string, isSetup bool, entry *LogEntry, gs *game.GameState) (game.Action, error) {
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

	// Check if this is a Dwarves tunneling or Fakirs carpet flight action
	// We check if the hex is adjacent - if not, Dwarves/Fakirs must be using their skip ability
	player := gs.GetPlayer(playerID)
	if player != nil {
		isAdjacent := gs.IsAdjacentToPlayerBuilding(hex, playerID)

		// If not adjacent, check if this faction can use skip ability
		if !isAdjacent {
			// Dwarves can use tunneling
			if _, ok := player.Faction.(*factions.Dwarves); ok {
				return game.NewTransformAndBuildActionWithSkip(playerID, hex, true), nil
			}

			// Fakirs can use carpet flight
			if _, ok := player.Faction.(*factions.Fakirs); ok {
				return game.NewTransformAndBuildActionWithSkip(playerID, hex, true), nil
			}
		}
	}

	// During normal gameplay, building a dwelling on home terrain (no transformation needed)
	return game.NewTransformAndBuildAction(playerID, hex, true), nil
}

func convertUpgradeAction(playerID string, params map[string]string, entry *LogEntry, gs *game.GameState) (game.Action, error) {
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
		// Execute upgrade first
		if skipValidation {
			// Resources already synced - just place the building manually
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

			// Award VP from scoring tile ONLY if there's also a town tile
			// (validator doesn't sync VP for actions with town tiles)
			// For favor-only actions, VP is already synced by validator, so skip VP awarding
			_, hasTownTile := params["town_tile"]
			if hasTownTile {
				var scoringAction game.ScoringActionType
				switch buildingType {
				case models.BuildingTradingHouse:
					scoringAction = game.ScoringActionTradingHouse
				case models.BuildingTemple:
					scoringAction = game.ScoringActionTemple
				case models.BuildingSanctuary, models.BuildingStronghold:
					scoringAction = game.ScoringActionStronghold
				}
				gs.AwardActionVP(playerID, scoringAction)
			}

			// Note: Don't set pending favor tile selection when skipValidation is true
			// The validator has already synced the final state, so we don't need to track
			// pending actions. The favor tile effects are already included in the synced state.
		} else {
			// Normal execution
			if err := upgradeAction.Validate(gs); err != nil {
				return nil, fmt.Errorf("upgrade validation failed: %v", err)
			}
			if err := upgradeAction.Execute(gs); err != nil {
				return nil, fmt.Errorf("upgrade execution failed: %v", err)
			}
		}

		// Now handle favor tile selection
		favorTileType, err := ParseFavorTile(favorTileStr)
		if err != nil {
			return nil, fmt.Errorf("invalid favor tile %s: %v", favorTileStr, err)
		}

		if skipValidation {
			// Bug #42: When skipValidation is true, the validator syncs cult positions but does NOT
			// add the favor tile to player's collection. We need to add it manually without applying
			// cult advancement (which was already synced by validator).
			// Just take the tile without executing the full action (to avoid double-applying cult advancement)
			if err := gs.FavorTiles.TakeFavorTile(playerID, favorTileType); err != nil {
				return nil, fmt.Errorf("failed to take favor tile: %v", err)
			}
		} else {
			// Normal execution: full favor tile action including cult advancement
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
		}

		// Check if there's also a town tile to process (compound action with both favor and town tiles)
		if townTileStr, hasTownTile := params["town_tile"]; hasTownTile {
			// This is a complex action with both favor tile and town tile
			// The favor tile may have provided a town key (from cult advancement), allowing a town to be formed

			// Check if there's already a pending town formation
			// If not, check all hexes with player buildings to find potential town formations
			if gs.PendingTownFormations[playerID] == nil {
				// Iterate through all hexes with player buildings and check for town formation
				// Map goes from row 0 to 8, q ranges from -1 to 12
				for q := -1; q <= 12; q++ {
					for r := 0; r <= 8; r++ {
						testHex := board.Hex{Q: q, R: r}
						mapHex := gs.Map.GetHex(testHex)
						if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
							gs.CheckForTownFormation(playerID, testHex)
							if gs.PendingTownFormations[playerID] != nil {
								break
							}
						}
					}
					if gs.PendingTownFormations[playerID] != nil {
						break
					}
				}
			}

			townTileType, err := ParseTownTile(townTileStr)
			if err != nil {
				return nil, fmt.Errorf("invalid town tile %s: %v", townTileStr, err)
			}

			// Select the town tile
			if err := gs.SelectTownTile(playerID, townTileType); err != nil {
				return nil, fmt.Errorf("town tile selection failed: %v", err)
			}
		}

		// All actions executed, return nil to skip normal execution
		return nil, nil
	}

	// If there's a town tile specified, this is a compound action:
	// upgrade + select town tile (e.g., upgrade to TP and form town). Execute both immediately.
	if townTileStr, hasTownTile := params["town_tile"]; hasTownTile {
		// If skipValidation is set, manually place the building (resources already synced)
		// Otherwise, execute the upgrade normally
		if skipValidation {
			// Manually place the upgraded building
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
			case models.BuildingTemple, models.BuildingStronghold, models.BuildingSanctuary:
				newPowerValue = 3
			default:
				newPowerValue = 1
			}

			// Upgrade the building
			mapHex.Building.Type = buildingType
			mapHex.Building.PowerValue = newPowerValue

			// Trigger power leech
			gs.TriggerPowerLeech(hex, playerID)

			// Apply faction-specific stronghold benefits
			if buildingType == models.BuildingStronghold {
				if cultists, ok := player.Faction.(*factions.Cultists); ok {
					strongholdVP := cultists.BuildStronghold()
					player.VictoryPoints += strongholdVP
				}
			}

			// Award VP from Water+1 favor tile if upgrading to Trading House
			if buildingType == models.BuildingTradingHouse {
				playerTiles := gs.FavorTiles.GetPlayerTiles(playerID)
				if game.HasFavorTile(playerTiles, game.FavorWater1) {
					player.VictoryPoints += 3
				}
			}

			// Award VP from scoring tile
			var scoringAction game.ScoringActionType
			switch buildingType {
			case models.BuildingTradingHouse:
				scoringAction = game.ScoringActionTradingHouse
			case models.BuildingTemple:
				scoringAction = game.ScoringActionTemple
			case models.BuildingSanctuary, models.BuildingStronghold:
				scoringAction = game.ScoringActionStronghold
			}
			gs.AwardActionVP(playerID, scoringAction)

			// Check for town formation - need to check all player buildings, not just this hex
			// A town can form anywhere in the cluster when we upgrade a building
			if gs.PendingTownFormations[playerID] == nil {
				// Iterate through all hexes with player buildings and check for town formation
				for q := -1; q <= 12; q++ {
					for r := 0; r <= 8; r++ {
						testHex := board.Hex{Q: q, R: r}
						mapHex := gs.Map.GetHex(testHex)
						if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
							gs.CheckForTownFormation(playerID, testHex)
							if gs.PendingTownFormations[playerID] != nil {
								break
							}
						}
					}
					if gs.PendingTownFormations[playerID] != nil {
						break
					}
				}
			}
		} else {
			// Normal execution: validate and execute upgrade
			if err := upgradeAction.Validate(gs); err != nil {
				return nil, fmt.Errorf("upgrade validation failed: %v", err)
			}
			if err := upgradeAction.Execute(gs); err != nil {
				return nil, fmt.Errorf("upgrade execution failed: %v", err)
			}
		}

		// Parse town tile
		townTileType, err := ParseTownTile(townTileStr)
		if err != nil {
			return nil, fmt.Errorf("invalid town tile %s: %v", townTileStr, err)
		}

		// In skipValidation mode (replay), the validator has already synced resources to final state
		// which includes town tile benefits. So we need to manually create a pending town formation
		// and skip applying the benefits (they're already in the synced state).
		if skipValidation && len(gs.PendingTownFormations[playerID]) == 0 {
			// Create a dummy pending town formation
			// The exact hexes don't matter since we're in replay mode and just need to allow tile selection
			gs.PendingTownFormations[playerID] = []*game.PendingTownFormation{
				{
					PlayerID: playerID,
					Hexes:    []board.Hex{hex}, // Use the upgraded hex as placeholder
				},
			}
			fmt.Printf("DEBUG: Created pending town formation for replay mode\n")
		}

		// Select the town tile (this will form the town and apply benefits)
		debugPlayer := gs.GetPlayer(playerID)
		if debugPlayer != nil {
			fmt.Printf("DEBUG: Before SelectTownTile - %s power bowls: %d/%d/%d\n",
				playerID, debugPlayer.Resources.Power.Bowl1, debugPlayer.Resources.Power.Bowl2, debugPlayer.Resources.Power.Bowl3)
		}
		if err := gs.SelectTownTile(playerID, townTileType); err != nil {
			return nil, fmt.Errorf("town tile selection failed: %v", err)
		}
		debugPlayer = gs.GetPlayer(playerID)
		if debugPlayer != nil {
			fmt.Printf("DEBUG: After SelectTownTile - %s power bowls: %d/%d/%d\n",
				playerID, debugPlayer.Resources.Power.Bowl1, debugPlayer.Resources.Power.Bowl2, debugPlayer.Resources.Power.Bowl3)
		}

		// Both actions executed, return nil to skip normal execution
		return nil, nil
	}

	// If skip_validation is set but there's no favor/town tile, manually place the building
	if skipValidation {
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

	// Note: "dig X" and "spades X" parameters are handled by the compound parser
	// which creates a DigAdvancementComponent that grants spades during execution
	// We just need to return the appropriate action here

	// Always return transform-and-build action - the action itself will check if transform is needed
	// If terrain is already home terrain, it will skip transform automatically
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
