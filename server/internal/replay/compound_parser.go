package replay

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
)

// ParseCompoundAction parses a compound action string into structured components
func ParseCompoundAction(actionStr string, entry *LogEntry, gs *game.GameState) (*CompoundAction, error) {
	// Split by periods to get tokens
	tokens := splitIntoTokens(actionStr)

	compound := &CompoundAction{
		Components: []ActionComponent{},
	}

	var burnAmount int
	var powerModifier *PowerActionModifier
	var mainActionFound bool
	var auxiliaries []*AuxiliaryComponent // Collect these to add after main action
	var digAmount int // Track "dig X" for transform-and-build actions

	for i := 0; i < len(tokens); i++ {
		token := strings.TrimSpace(tokens[i])
		if token == "" {
			continue
		}

		// Parse burn
		if strings.HasPrefix(token, "burn ") {
			var amount int
			fmt.Sscanf(token, "burn %d", &amount)
			// Store burn amount for potential power action modifier
			// We'll decide whether to add it as a component or attach to modifier later
			burnAmount = amount
			continue
		}

		// Parse conversion (check for Darklings W->P first)
		if darklingsAction, ok := parseDarklingsWorkerToPriest(token, entry); ok {
			// Flush any collected auxiliaries BEFORE the conversion to preserve order
			for _, aux := range auxiliaries {
				compound.Components = append(compound.Components, aux)
			}
			auxiliaries = nil

			// This is Darklings priest ordination: "convert 3W to 3P"
			compound.Components = append(compound.Components, darklingsAction)
			continue
		}

		if conv, ok := parseConversion(token); ok {
			// Flush any collected auxiliaries BEFORE the conversion to preserve order
			// This ensures: upgrade. +FAV. convert => upgrade, +FAV, convert (not upgrade, convert, +FAV)
			for _, aux := range auxiliaries {
				compound.Components = append(compound.Components, aux)
			}
			auxiliaries = nil

			// If we have a pending burn that hasn't been used, add it first
			if burnAmount > 0 {
				compound.Components = append(compound.Components, &ConversionComponent{
					Type:   ConvBurn,
					Amount: burnAmount,
				})
				burnAmount = 0
			}
			compound.Components = append(compound.Components, conv)
			continue
		}

		// Parse auxiliary - collect these to add after main action
		if aux, ok := parseAuxiliary(token); ok {
			auxiliaries = append(auxiliaries, aux)
			continue
		}

		// Parse action (which might be a power action modifier, bonus card action, or special faction action)
		tokenLower := strings.ToLower(token)
		if strings.HasPrefix(tokenLower, "action ") {
			actionType := strings.ToUpper(strings.Fields(token)[1])

			// Check if this is a bonus card action (BON1-BON12)
			// Some bonus cards provide special actions with benefits
			if strings.HasPrefix(actionType, "BON") {
				// BON1: Provides 1 free spade for building
				// Grant free spade via PendingSpades before processing the next action
				if actionType == "BON1" {
					playerID := entry.GetPlayerID()
					if gs.PendingSpades == nil {
						gs.PendingSpades = make(map[string]int)
					}
					gs.PendingSpades[playerID] += 1
				}
				// Skip bonus card marker - the following token will be the actual action
				continue
			}

			// Check if this is a favor tile special action (FAV1-FAV12)
			// FAV6 (Water 2): Allows advancing 1 step on any cult track
			if strings.HasPrefix(actionType, "FAV") {
				if actionType == "FAV6" {
					// Pattern: "action FAV6. +FIRE" or "action FAV6. +WATER"
					// This is similar to ACTA but advances by 1 step instead of 2
					playerID := entry.GetPlayerID()
					if i+1 < len(tokens) {
						nextToken := strings.TrimSpace(tokens[i+1])
						if strings.HasPrefix(nextToken, "+") {
							cultStr := strings.TrimPrefix(nextToken, "+")
							// Strip any leading digit and optional space (e.g., "2AIR" -> "AIR")
							cultStr = strings.TrimLeft(cultStr, "0123456789 ")
							cultType, err := ParseCultTrack(cultStr)
							if err != nil {
								return nil, fmt.Errorf("invalid cult track %s: %w", cultStr, err)
							}
							// Create a Water+2 favor tile cult advance action for 1 step (FAV6)
							action := game.NewWater2CultAdvanceAction(playerID, game.CultTrack(cultType))
							mainAction := &MainActionComponent{Action: action}
							compound.Components = append(compound.Components, mainAction)
							mainActionFound = true

							// Add any collected auxiliaries after the main action
							for _, aux := range auxiliaries {
								compound.Components = append(compound.Components, aux)
							}
							auxiliaries = nil

							// Skip the cult track token
							i += 1
							continue
						}
					}
				}
				// Other favor tile actions might be handled here in the future
				// For now, skip favor tile marker
				continue
			}

			// Check if this is a special faction action (ACTW, ACTA, etc.)
			// These require special handling - create SpecialAction based on type
			// Special actions: ACTW (Witches' Ride), ACTA (Auren cult advance), etc.
			if strings.HasPrefix(actionType, "ACT") && !strings.HasPrefix(actionType, "ACT1") &&
			   !strings.HasPrefix(actionType, "ACT2") && !strings.HasPrefix(actionType, "ACT3") &&
			   !strings.HasPrefix(actionType, "ACT4") && !strings.HasPrefix(actionType, "ACT5") &&
			   !strings.HasPrefix(actionType, "ACT6") {
				// Handle special faction actions
				playerID := entry.GetPlayerID()
				action, tokensConsumed, err := convertSpecialFactionAction(actionType, tokens, i, playerID, entry, gs)
				if err != nil {
					return nil, fmt.Errorf("failed to convert special faction action %s: %w", actionType, err)
				}
				if action != nil {
					mainAction := &MainActionComponent{Action: action}
					compound.Components = append(compound.Components, mainAction)
					mainActionFound = true

					// Add any collected auxiliaries after the main action
					for _, aux := range auxiliaries {
						compound.Components = append(compound.Components, aux)
					}
					auxiliaries = nil // Clear for next potential main action

					// Skip the consumed tokens
					i += tokensConsumed
				}
				continue
			}

			// Check if next token is a main action (build, upgrade, transform, etc.)
			if i+1 < len(tokens) {
				nextToken := strings.TrimSpace(tokens[i+1])
				if isMainActionToken(nextToken) {
					// Parse the power action type
					powerActionType, err := parsePowerActionType(actionType)
					if err == nil {
						// For spade actions (ACT5, ACT6), execute as standalone action first
						// to grant free spades BEFORE any transforms happen
						if powerActionType == game.PowerActionSpade1 || powerActionType == game.PowerActionSpade2 {
							// Create standalone power action that grants free spades
							action := &PowerActionFreeSpades{
								PowerActionType: powerActionType,
								Burned:          burnAmount,
							}
							compound.Components = append(compound.Components, action)
							burnAmount = 0
							continue
						}

						// For other power actions, treat as modifier
						powerModifier = &PowerActionModifier{
							PowerActionType: powerActionType,
							Burned:          burnAmount,
						}
						burnAmount = 0 // Reset after using
						continue        // Don't create main action component yet
					}
				}
			}

			// Standalone power action (no build/transform after)
			// For spade actions (ACT5/ACT6), create PowerActionFreeSpades component
			powerActionType, err := parsePowerActionType(actionType)
			if err == nil && (powerActionType == game.PowerActionSpade1 || powerActionType == game.PowerActionSpade2) {
				// Handle burn first if needed
				if burnAmount > 0 {
					compound.Components = append(compound.Components, &ConversionComponent{
						Type:   ConvBurn,
						Amount: burnAmount,
					})
					burnAmount = 0
				}

				// Create standalone power action that grants free spades
				action := &PowerActionFreeSpades{
					PowerActionType: powerActionType,
					Burned:          0, // Already handled above
				}
				compound.Components = append(compound.Components, action)
				mainActionFound = true

				// Add any collected auxiliaries after the main action
				for _, aux := range auxiliaries {
					compound.Components = append(compound.Components, aux)
				}
				auxiliaries = nil // Clear for next potential main action

				continue
			}

			// For other power actions, convert to game action
			action, err := convertPowerActionToGameAction(actionType, entry, gs)
			if err != nil {
				return nil, fmt.Errorf("failed to convert power action: %w", err)
			}

			mainAction := &MainActionComponent{
				Action:    action,
				Modifiers: []ActionModifier{},
			}
			if burnAmount > 0 {
				// Burn was done before this standalone power action
				// We need to execute the burn as a conversion first
				compound.Components = append(compound.Components, &ConversionComponent{
					Type:   ConvBurn,
					Amount: burnAmount,
				})
				burnAmount = 0
			}
			compound.Components = append(compound.Components, mainAction)
			mainActionFound = true

			// Add any collected auxiliaries after the main action
			for _, aux := range auxiliaries {
				compound.Components = append(compound.Components, aux)
			}
			auxiliaries = nil // Clear for next potential main action

			continue
		}

		// Parse cult advancement (from bonus cards or other sources)
		// Pattern: "+WATER", "+FIRE", "+EARTH", "+AIR"
		// BUT: For Cultists, these are ALWAYS informational (side effect of power leech acceptance)
		// and should be skipped entirely
		if strings.HasPrefix(token, "+") && !strings.HasPrefix(token, "+FAV") && !strings.HasPrefix(token, "+TW") {
			cultTrack := strings.TrimPrefix(token, "+")
			_, err := ParseCultTrack(cultTrack)
			if err == nil {
				// This is a valid cult track token
				// For Cultists, these are ALWAYS informational side effects
				// They happen automatically when opponents accept power leeches
				// For all factions, when cult advancement appears alongside other actions,
				// it's informational (the actual cult advancement already happened)
				// Skip parsing these as actions
				continue
			}
		}

		// Parse main action (build, upgrade, transform, send priest, pass, advance, etc.)
		if main, ok := parseMainActionToken(token); ok {
			// If this is a build action and we have a dig amount, convert to transform-and-build
			if main.Type == ActionBuild && digAmount > 0 {
				main.Type = ActionTransformAndBuild
				main.Params["dig"] = fmt.Sprintf("%d", digAmount)
				digAmount = 0 // Clear after use
			}

			action, err := convertMainActionPartToGameAction(main, entry, gs)
			if err != nil {
				return nil, fmt.Errorf("failed to convert main action: %w", err)
			}

			mainAction := &MainActionComponent{
				Action:    action,
				Modifiers: []ActionModifier{},
			}

			// Attach power modifier if we parsed one earlier
			if powerModifier != nil {
				mainAction.Modifiers = append(mainAction.Modifiers, powerModifier)
				powerModifier = nil
			} else if burnAmount > 0 {
				// We have a burn but no power modifier - this means standalone burn before main action
				// Add burn as a separate conversion component
				compound.Components = append(compound.Components, &ConversionComponent{
					Type:   ConvBurn,
					Amount: burnAmount,
				})
				burnAmount = 0
			}

			compound.Components = append(compound.Components, mainAction)
			mainActionFound = true

			// Add any collected auxiliaries after the main action
			for _, aux := range auxiliaries {
				compound.Components = append(compound.Components, aux)
			}
			auxiliaries = nil // Clear for next potential main action

			continue
		}

		// Special handling for "dig X" or "Dig X" - this is just notation
		// indicating the transformation uses X spades at current digging level cost
		// We don't grant pending spades for this - the TransformAndBuildAction
		// will calculate the terrain distance and charge workers accordingly
		tokenLower = strings.ToLower(token)
		if strings.HasPrefix(tokenLower, "dig ") {
			// Parse the dig amount (might be used for validation later)
			fmt.Sscanf(tokenLower, "dig %d", &digAmount)
			// Just skip this token - it's informational only
			continue
		}

		// Special handling for "transform X to Y" or "transform X" (implied home terrain)
		tokenLower = strings.ToLower(token)
		if strings.HasPrefix(tokenLower, "transform ") {
			// Use original token for field splitting to preserve coordinate case
			fields := strings.Fields(token)

			// Case 1: "transform X to Y" (explicit target terrain)
			if len(fields) >= 4 && strings.ToLower(fields[2]) == "to" {
				transformHexStr := fields[1]

				// Check if there's a build at the SAME hex (transform-and-build pattern)
				// If there's a build at a DIFFERENT hex, we need to process the transform separately
				if hasFollowingBuildAtSameHex(tokens, i, transformHexStr) {
					// Transform and build at same hex - let the build action handle both
					continue
				}

				// Either no build following, or build at different hex - process transform separately
				hex, err := ConvertLogCoordToAxial(transformHexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid coordinate in transform: %w", err)
				}

				// Parse target terrain from log (use original case for terrain color)
				targetTerrainStr := fields[3]
				targetTerrain, err := ParseTerrainColor(targetTerrainStr)
				if err != nil {
					return nil, fmt.Errorf("invalid target terrain %s: %w", targetTerrainStr, err)
				}

				playerID := entry.GetPlayerID()

				// Check if player has pending cult reward spades
				if gs.PendingSpades != nil && gs.PendingSpades[playerID] > 0 {
					// Use cult spade action (FREE, transforms to home terrain)
					action := game.NewUseCultSpadeAction(playerID, hex)
					mainAction := &MainActionComponent{Action: action}
					compound.Components = append(compound.Components, mainAction)
				} else {
					// Use TransformTerrainComponent for transform-only to specified terrain
					transformComp := &TransformTerrainComponent{
						TargetHex:     hex,
						TargetTerrain: targetTerrain,
					}
					compound.Components = append(compound.Components, transformComp)
				}

				mainActionFound = true

				// Add any collected auxiliaries after the main action
				for _, aux := range auxiliaries {
					compound.Components = append(compound.Components, aux)
				}
				auxiliaries = nil // Clear for next potential main action

				continue
			}

			// Case 2: "transform X" (implied home terrain)
			if len(fields) == 2 {
				transformHexStr := fields[1]
				hex, err := ConvertLogCoordToAxial(transformHexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid coordinate in transform: %w", err)
				}

				playerID := entry.GetPlayerID()

				// Check if player has pending cult reward spades
				if gs.PendingSpades != nil && gs.PendingSpades[playerID] > 0 {
					// Use cult spade action (FREE, transforms BY 1 spade towards home)
					action := game.NewUseCultSpadeAction(playerID, hex)
					mainAction := &MainActionComponent{Action: action}
					compound.Components = append(compound.Components, mainAction)
				} else {
					// Transform to home terrain using paid spades
					player := gs.GetPlayer(playerID)
					if player != nil {
						targetTerrain := player.Faction.GetHomeTerrain()
						transformComp := &TransformTerrainComponent{
							TargetHex:     hex,
							TargetTerrain: targetTerrain,
						}
						compound.Components = append(compound.Components, transformComp)
					}
				}

				mainActionFound = true

				// Add any collected auxiliaries after the main action
				for _, aux := range auxiliaries {
					compound.Components = append(compound.Components, aux)
				}
				auxiliaries = nil // Clear for next potential main action

				continue
			}
		}
	}

	// Add any remaining auxiliaries at the end
	// (These would be auxiliaries that came after the last main action)
	for _, aux := range auxiliaries {
		compound.Components = append(compound.Components, aux)
	}

	// In Terra Mystica, conversions can happen without a main action
	// So we allow compound actions that are just conversions or auxiliaries
	if !mainActionFound && len(compound.Components) == 0 {
		return nil, fmt.Errorf("no components found in: %s", actionStr)
	}

	return compound, nil
}

// splitIntoTokens splits an action string by periods, handling edge cases
func splitIntoTokens(actionStr string) []string {
	// Remove power leech prefix if present (e.g., "2 3  convert..." or "1 1  upgrade...")
	// Power leech format: "<num> <num>  <action>" with two spaces after the numbers
	actionStr = strings.TrimSpace(actionStr)

	// Check for power leech pattern: starts with digit(s), space, digit(s), double-space
	// Example: "2 3  convert..." or "1 1  upgrade..." or "3 5 3  upgrade..." (Cultists)
	// Cultists have 3 numbers: leech_from leech_to cultist_gain
	parts := strings.Split(actionStr, "  ") // Split on double-space
	if len(parts) >= 2 {
		// Check if first part looks like power leech (e.g., "2 3" or "1 1" or "3 5 3")
		firstPart := strings.TrimSpace(parts[0])
		fields := strings.Fields(firstPart)
		if len(fields) >= 1 && len(fields) <= 3 {
			// Check if all fields are numbers
			allNumbers := true
			for _, field := range fields {
				if _, err := fmt.Sscanf(field, "%d", new(int)); err != nil {
					allNumbers = false
					break
				}
			}
			if allNumbers {
				// This is a power leech prefix, remove it
				actionStr = strings.Join(parts[1:], "  ")
			}
		}
	}

	// Split by period
	return strings.Split(actionStr, ".")
}

// parseConversion attempts to parse a token as a conversion
func parseConversion(token string) (*ConversionComponent, bool) {
	token = strings.TrimSpace(token)

	// convert XPW to YC
	// convert XP to YW
	// convert XW to YC
	tokenLower := strings.ToLower(token)
	if strings.HasPrefix(tokenLower, "convert ") && strings.Contains(tokenLower, " to ") {
		parts := strings.Split(strings.TrimPrefix(tokenLower, "convert "), " to ")
		if len(parts) != 2 {
			return nil, false
		}

		fromPart := strings.TrimSpace(parts[0])
		toPart := strings.TrimSpace(parts[1])

		// Parse amounts and resources
		var fromAmount, toAmount int
		var fromRes, toRes string

		// Parse "from" part (e.g., "1pw", "3p", "5w") - already lowercase from tokenLower
		if strings.Contains(fromPart, "pw") {
			fmt.Sscanf(fromPart, "%dpw", &fromAmount)
			fromRes = "PW"
		} else if strings.Contains(fromPart, "p") {
			fmt.Sscanf(fromPart, "%dp", &fromAmount)
			fromRes = "P"
		} else if strings.Contains(fromPart, "w") {
			fmt.Sscanf(fromPart, "%dw", &fromAmount)
			fromRes = "W"
		} else {
			return nil, false
		}

		// Parse "to" part (e.g., "1c", "2w", "1p") - already lowercase from tokenLower
		if strings.Contains(toPart, "c") {
			fmt.Sscanf(toPart, "%dc", &toAmount)
			toRes = "C"
		} else if strings.Contains(toPart, "w") {
			fmt.Sscanf(toPart, "%dw", &toAmount)
			toRes = "W"
		} else if strings.Contains(toPart, "p") {
			fmt.Sscanf(toPart, "%dp", &toAmount)
			toRes = "P"
		} else {
			return nil, false
		}

		// Determine conversion type
		convType := determineConversionType(fromRes, toRes)
		if convType == -1 {
			return nil, false
		}

		return &ConversionComponent{
			Type:   ConversionType(convType),
			From:   fromRes,
			To:     toRes,
			Amount: toAmount, // Use target amount
		}, true
	}

	return nil, false
}

// determineConversionType determines the conversion type from resources
func determineConversionType(from, to string) int {
	if from == "PW" && to == "C" {
		return int(ConvPowerToCoins)
	} else if from == "PW" && to == "W" {
		return int(ConvPowerToWorkers)
	} else if from == "PW" && to == "P" {
		return int(ConvPowerToPriests)
	} else if from == "P" && to == "W" {
		return int(ConvPriestToWorker)
	} else if from == "W" && to == "C" {
		return int(ConvWorkerToCoin)
	}
	return -1
}

// parseDarklingsWorkerToPriest checks if this is a Darklings priest ordination conversion
func parseDarklingsWorkerToPriest(token string, entry *LogEntry) (*DarklingsPriestOrdinationComponent, bool) {
	token = strings.TrimSpace(token)

	// Pattern: "convert 3W to 3P"
	if strings.HasPrefix(token, "convert ") && strings.Contains(token, " to ") {
		parts := strings.Split(strings.TrimPrefix(token, "convert "), " to ")
		if len(parts) != 2 {
			return nil, false
		}

		fromPart := strings.TrimSpace(parts[0])
		toPart := strings.TrimSpace(parts[1])

		// Check if this is W to P conversion
		if strings.Contains(fromPart, "W") && strings.Contains(toPart, "P") {
			var workersToConvert int
			fmt.Sscanf(fromPart, "%dW", &workersToConvert)

			// Verify this is Darklings (case-insensitive)
			playerID := strings.ToLower(entry.GetPlayerID())
			if playerID != "darklings" {
				// Only Darklings can do W->P conversion
				return nil, false
			}

			return &DarklingsPriestOrdinationComponent{
				WorkersToConvert: workersToConvert,
			}, true
		}
	}

	return nil, false
}

// parseAuxiliary attempts to parse a token as an auxiliary action
func parseAuxiliary(token string) (*AuxiliaryComponent, bool) {
	token = strings.TrimSpace(token)
	tokenUpper := strings.ToUpper(token)

	// +FAV5 or +fav5
	if strings.HasPrefix(tokenUpper, "+FAV") {
		return &AuxiliaryComponent{
			Type:   AuxFavorTile,
			Params: map[string]string{"tile": tokenUpper},
		}, true
	}

	// +TW3 or +tw3
	if strings.HasPrefix(tokenUpper, "+TW") {
		return &AuxiliaryComponent{
			Type:   AuxTownTile,
			Params: map[string]string{"tile": tokenUpper},
		}, true
	}

	return nil, false
}

// MainActionPart represents a parsed main action token
type MainActionPart struct {
	Type   ActionType
	Params map[string]string
}

// parseMainActionToken attempts to parse a token as a main action
func parseMainActionToken(token string) (*MainActionPart, bool) {
	token = strings.TrimSpace(token)
	tokenLower := strings.ToLower(token)

	// build E7 (case-insensitive)
	if strings.HasPrefix(tokenLower, "build ") {
		coord := strings.TrimPrefix(tokenLower, "build ")
		coord = strings.ToUpper(strings.Fields(coord)[0])
		return &MainActionPart{
			Type:   ActionBuild,
			Params: map[string]string{"coord": coord},
		}, true
	}

	// upgrade E5 to TP (case-insensitive)
	if strings.HasPrefix(tokenLower, "upgrade ") {
		parts := strings.Fields(tokenLower)
		if len(parts) >= 4 && parts[2] == "to" {
			return &MainActionPart{
				Type: ActionUpgrade,
				Params: map[string]string{
					"coord":    strings.ToUpper(parts[1]),
					"building": strings.ToUpper(parts[3]),
				},
			}, true
		}
	}

	// send p to WATER (case-insensitive)
	if strings.HasPrefix(tokenLower, "send p to ") {
		cult := strings.TrimPrefix(tokenLower, "send p to ")
		cultName := strings.Fields(cult)[0]
		return &MainActionPart{
			Type:   ActionSendPriest,
			Params: map[string]string{"cult": cultName},
		}, true
	}

	// advance ship (case-insensitive)
	if strings.HasPrefix(tokenLower, "advance ship") {
		return &MainActionPart{Type: ActionAdvanceShipping}, true
	}

	// advance dig (case-insensitive)
	if strings.HasPrefix(tokenLower, "advance dig") {
		return &MainActionPart{Type: ActionAdvanceDigging}, true
	}

	// pass BON1 or just pass
	if strings.HasPrefix(token, "pass") || strings.HasPrefix(token, "Pass") {
		fields := strings.Fields(token)
		if len(fields) >= 2 {
			// Pass with bonus card
			return &MainActionPart{
				Type:   ActionPass,
				Params: map[string]string{"bonus": strings.ToUpper(fields[1])},
			}, true
		} else if len(fields) == 1 {
			// Pass without bonus card (end of game)
			return &MainActionPart{
				Type:   ActionPass,
				Params: map[string]string{},
			}, true
		}
	}

	return nil, false
}

// isMainActionToken checks if a token looks like a main action
func isMainActionToken(token string) bool {
	token = strings.TrimSpace(token)
	tokenLower := strings.ToLower(token)
	return strings.HasPrefix(tokenLower, "build ") ||
		strings.HasPrefix(tokenLower, "upgrade ") ||
		strings.HasPrefix(tokenLower, "transform ") ||
		strings.HasPrefix(tokenLower, "dig ") ||
		strings.HasPrefix(tokenLower, "pass") ||
		strings.HasPrefix(tokenLower, "send p to") ||
		strings.HasPrefix(tokenLower, "advance ship") ||
		strings.HasPrefix(tokenLower, "advance dig")
}

// hasFollowingBuild checks if there's a "build" token after the current index
func hasFollowingBuild(tokens []string, currentIndex int) bool {
	for i := currentIndex + 1; i < len(tokens); i++ {
		token := strings.TrimSpace(tokens[i])
		if strings.HasPrefix(token, "build ") {
			return true
		}
	}
	return false
}

// hasFollowingBuildAtSameHex checks if there's a "build" token at the same hex as the transform
func hasFollowingBuildAtSameHex(tokens []string, currentIndex int, transformHex string) bool {
	for i := currentIndex + 1; i < len(tokens); i++ {
		token := strings.TrimSpace(tokens[i])
		if strings.HasPrefix(token, "build ") {
			// Extract build hex
			buildHex := strings.TrimPrefix(token, "build ")
			buildHex = strings.Fields(buildHex)[0]
			return buildHex == transformHex
		}
	}
	return false
}

// parsePowerActionType parses a power action type string
func parsePowerActionType(actionType string) (game.PowerActionType, error) {
	switch actionType {
	case "ACT1":
		return game.PowerActionBridge, nil
	case "ACT2":
		return game.PowerActionPriest, nil
	case "ACT3":
		return game.PowerActionWorkers, nil
	case "ACT4":
		return game.PowerActionCoins, nil
	case "ACT5":
		return game.PowerActionSpade1, nil
	case "ACT6":
		return game.PowerActionSpade2, nil
	default:
		return 0, fmt.Errorf("unknown power action type: %s", actionType)
	}
}

// Helper functions to convert parsed parts to game actions
// These will delegate to the existing action_converter.go functions

func convertPowerActionToGameAction(actionType string, entry *LogEntry, gs *game.GameState) (game.Action, error) {
	// Parse the power action type
	powerActionType, err := parsePowerActionType(actionType)
	if err != nil {
		return nil, err
	}

	// Check if this is a bridge action (ACT1) with coordinates
	// Entry.Action might be "action ACT1. Bridge F4:G3"
	if powerActionType == game.PowerActionBridge && strings.Contains(entry.Action, "Bridge ") {
		// Extract bridge coordinates
		parts := strings.Split(entry.Action, "Bridge ")
		if len(parts) >= 2 {
			bridgeStr := strings.TrimSpace(parts[1])
			// Split on colon to get hex coordinates
			coords := strings.Split(bridgeStr, ":")
			if len(coords) == 2 {
				hex1Str := strings.TrimSpace(coords[0])
				hex2Str := strings.TrimSpace(coords[1])

				hex1, err := ConvertLogCoordToAxial(hex1Str)
				if err != nil {
					return nil, fmt.Errorf("invalid bridge hex1 %s: %w", hex1Str, err)
				}

				hex2, err := ConvertLogCoordToAxial(hex2Str)
				if err != nil {
					return nil, fmt.Errorf("invalid bridge hex2 %s: %w", hex2Str, err)
				}

				return game.NewPowerActionWithBridge(entry.GetPlayerID(), hex1, hex2), nil
			}
		}
	}

	// Create a regular power action
	return game.NewPowerAction(entry.GetPlayerID(), powerActionType), nil
}

func convertMainActionPartToGameAction(part *MainActionPart, entry *LogEntry, gs *game.GameState) (game.Action, error) {
	// This will delegate to existing action converter functions
	playerID := entry.GetPlayerID()

	switch part.Type {
	case ActionBuild:
		// Check if we're in setup phase
		isSetup := (gs.Phase == game.PhaseSetup)
		return convertBuildAction(playerID, part.Params, isSetup)
	case ActionTransformAndBuild:
		return convertTransformAndBuildAction(playerID, part.Params, gs)
	case ActionUpgrade:
		return convertUpgradeAction(playerID, part.Params, entry, gs)
	case ActionSendPriest:
		return convertSendPriestAction(playerID, part.Params, entry, gs)
	case ActionAdvanceShipping:
		return game.NewAdvanceShippingAction(playerID), nil
	case ActionAdvanceDigging:
		return game.NewAdvanceDiggingAction(playerID), nil
	case ActionPass:
		return convertPassAction(playerID, part.Params)
	default:
		return nil, fmt.Errorf("unsupported action type: %v", part.Type)
	}
}

// parseTransformOnly is no longer used - transform-only actions are now handled
// inline in ParseCompoundAction to properly use TransformTerrainComponent

// convertSpecialFactionAction converts special faction actions (ACTW, ACTA, etc.) to game actions
// Returns: (action, tokensConsumed, error)
// tokensConsumed indicates how many additional tokens were consumed (beyond the action type itself)
func convertSpecialFactionAction(actionType string, tokens []string, currentIndex int, playerID string, entry *LogEntry, gs *game.GameState) (game.Action, int, error) {
	// Look for the next token to determine what the special action does
	var nextToken string
	if currentIndex+1 < len(tokens) {
		// Trim the token (it may have leading/trailing spaces from period splitting)
		nextToken = strings.TrimSpace(tokens[currentIndex+1])
	}

	switch actionType {
	case "ACTW":
		// Witches' Ride: Build dwelling on any Forest hex (flying)
		// Pattern: "action ACTW. build I11"
		fields := strings.Fields(nextToken)
		if len(fields) >= 2 && fields[0] == "build" {
			coordStr := fields[1]
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid coordinate %s: %w", coordStr, err)
			}
			return game.NewWitchesRideAction(playerID, hex), 1, nil // Consumed 1 token (the "build" token)
		}
		return nil, 0, fmt.Errorf("ACTW requires a build action, got: %s", nextToken)

	case "ACTA":
		// Auren: Advance 2 spaces on cult track
		// Pattern: "action ACTA. +WATER" or "action ACTA. +2AIR"
		if strings.HasPrefix(nextToken, "+") {
			cultStr := strings.TrimPrefix(nextToken, "+")
			// Strip any leading digit and optional space (e.g., "2AIR" -> "AIR")
			cultStr = strings.TrimLeft(cultStr, "0123456789 ")
			cultType, err := ParseCultTrack(cultStr)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid cult track %s: %w", cultStr, err)
			}
			return game.NewAurenCultAdvanceAction(playerID, game.CultTrack(cultType)), 1, nil // Consumed 1 token (the cult track token)
		}
		return nil, 0, fmt.Errorf("ACTA requires a cult track advancement, got: %s", nextToken)

	case "ACTN":
		// Nomads Sandstorm: Transform hex to home terrain (yellow/desert) and/or build
		// Pattern 1: "action ACTN. transform E5 to yellow" (transform only)
		// Pattern 2: "action ACTN. transform E5 to yellow. build E5" (transform + build)
		// Pattern 3: "action ACTN. build E4" (build only - terrain already home)
		fields := strings.Fields(nextToken)

		// Check for build-only pattern
		if len(fields) >= 2 && fields[0] == "build" {
			coordStr := fields[1]
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid coordinate %s: %w", coordStr, err)
			}
			return game.NewNomadsSandstormAction(playerID, hex, true), 1, nil
		}

		// Check for transform pattern
		if len(fields) >= 4 && fields[0] == "transform" {
			coordStr := fields[1]
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid coordinate %s: %w", coordStr, err)
			}

			// Check if there's a build action after the transform (in the next token)
			buildDwelling := false
			tokensConsumed := 1
			if currentIndex+2 < len(tokens) {
				buildToken := strings.TrimSpace(tokens[currentIndex+2])
				if strings.HasPrefix(buildToken, "build ") {
					buildDwelling = true
					tokensConsumed = 2
				}
			}

			return game.NewNomadsSandstormAction(playerID, hex, buildDwelling), tokensConsumed, nil
		}
		return nil, 0, fmt.Errorf("ACTN requires a transform or build action, got: %s", nextToken)

	case "ACTS":
		// Swarmlings Stronghold: Upgrade Dwelling to Trading House for free
		// Pattern: "action ACTS. Upgrade D5 to TP"
		fields := strings.Fields(nextToken)
		if len(fields) >= 4 && strings.ToLower(fields[0]) == "upgrade" {
			coordStr := fields[1]
			hex, err := ConvertLogCoordToAxial(coordStr)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid coordinate %s: %w", coordStr, err)
			}
			return game.NewSwarmlingsUpgradeAction(playerID, hex), 1, nil
		}
		return nil, 0, fmt.Errorf("ACTS requires an upgrade action, got: %s", nextToken)

	default:
		// For other special actions, return nil (not yet implemented)
		return nil, 0, fmt.Errorf("special action %s not yet implemented", actionType)
	}
}
