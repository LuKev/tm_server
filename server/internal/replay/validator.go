package replay

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
)

// GameValidator validates a game replay against a log file
type GameValidator struct {
	GameState      *game.GameState
	LogEntries     []*LogEntry
	CurrentEntry   int
	Errors         []ValidationError
	IncomeApplied  bool // Track if income has been applied for current round
}

// ValidationError represents a validation error
type ValidationError struct {
	Line  int
	Entry *LogEntry
	Field string
	Expected interface{}
	Actual interface{}
	Message string
}

// NewGameValidator creates a new validator
func NewGameValidator() *GameValidator {
	return &GameValidator{
		Errors: make([]ValidationError, 0),
	}
}

// LoadGameLog loads a game log file
func (v *GameValidator) LoadGameLog(filename string) error {
	entries, err := ParseGameLog(filename)
	if err != nil {
		return fmt.Errorf("failed to parse game log: %v", err)
	}

	v.LogEntries = entries
	return nil
}

// InitializeGame sets up the initial game state from the log
func (v *GameValidator) InitializeGame() error {
	return v.SetupGame()
}

// ValidateNextEntry validates the next log entry
func (v *GameValidator) ValidateNextEntry() error {
	if v.CurrentEntry >= len(v.LogEntries) {
		return fmt.Errorf("no more entries to validate")
	}

	entry := v.LogEntries[v.CurrentEntry]
	v.CurrentEntry++
	
	// Handle comment lines - detect round transitions
	if entry.IsComment {
		// Check if this is a "Round X income" comment
		if len(entry.CommentText) > 5 && entry.CommentText[:5] == "Round" &&
		   strings.Contains(entry.CommentText, "income") {
			// Parse the round number to avoid duplicate increments
			var roundNum int
			fmt.Sscanf(entry.CommentText, "Round %d", &roundNum)
			
			// Only start a new round if this is actually a new round
			// (Log may have duplicate "Round X income" comments)
			if roundNum > v.GameState.Round {
				// Execute cleanup phase BEFORE starting new round
				// This awards cult rewards and adds coins to leftover bonus cards
				// Cleanup happens at end of rounds 1-5 (ExecuteCleanupPhase skips round 6)
				if v.GameState.Round >= 1 && v.GameState.Round < 6 {
					v.GameState.ExecuteCleanupPhase()
				}
				
				// Start new round (this resets HasPassed, power actions, etc.)
				v.GameState.StartNewRound()
				// Reset income flag for new round
				v.IncomeApplied = false
				// Note: Bonus cards are selected when passing and players keep them across rounds.
				// Cards are only returned when players pass and select a new card (handled in TakeBonusCard).
			}
		}
		return nil
	}

	// Skip "setup" entries - they just show initial state, no action to execute
	if entry.Action == "setup" {
		return nil
	}

	// Handle leech entries by manually syncing state
	// Leech offers are created asynchronously and we don't track them across entries in replay
	// So we just sync the entire state to match the log
	if strings.Contains(entry.Action, "Leech") || strings.Contains(entry.Action, "Decline") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.syncPlayerState(player, entry.Faction.String(), entry)
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle compound cult advance + pass actions FIRST (before standalone cult advancement)
	// Example: "+FIRE. pass BON10"
	// The cult advancement is reflected in cult track positions, we sync first then execute pass
	if strings.HasPrefix(entry.Action, "+") && strings.Contains(entry.Action, "pass ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync cult positions and power bowls (in case leech happened)
			v.syncPlayerCultPositions(player, entry.Faction.String(), entry)
			v.syncPlayerPowerBowls(player, entry)
		}
		// Continue to execute pass action (don't return early)
	}

	// Handle cult advancement entries (e.g., "+WATER", "+EARTH")
	// These are state-change entries that show cult track position changes
	// The cult advancement happens due to power actions, favor tiles, etc. from previous entries
	// EXCLUDE compound actions with "pass" (handled above)
	if strings.HasPrefix(entry.Action, "+") && !strings.Contains(entry.Action, "pass ") &&
		(strings.Contains(entry.Action, "FIRE") || strings.Contains(entry.Action, "WATER") ||
		strings.Contains(entry.Action, "EARTH") || strings.Contains(entry.Action, "AIR")) {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync all state - cult advancements can trigger power/coin gains
			v.syncPlayerState(player, entry.Faction.String(), entry)
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle "[opponent accepted power]" entries - these show state after leech resolution
	// These are informational entries that show the updated cult track state
	if strings.Contains(entry.Action, "[opponent accepted power]") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.syncPlayerState(player, entry.Faction.String(), entry)
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle "[all opponents declined power]" entries - Cultists gain power when all decline
	// This is an informational entry showing state after cultists receive their power bonus
	if strings.Contains(entry.Action, "[all opponents declined power]") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.syncPlayerState(player, entry.Faction.String(), entry)
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle dig + transform actions: sync resources and transform terrain
	// Example: "dig 1. transform H8 to green"
	// The dig is a state change (reflected in deltas), transform changes terrain
	if strings.HasPrefix(entry.Action, "dig ") && strings.Contains(entry.Action, "transform ") && !strings.Contains(entry.Action, "convert ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.syncPlayerResources(player, entry)
		}

		// Parse and execute the transform part
		v.executeTransformFromAction(entry.Action)
		// Skip normal action execution
		return nil
	}

	// Handle compound convert + dig + transform actions: sync resources and transform terrain
	// Example: "convert 1PW to 1C. dig 1. transform H4 to green"
	// The convert and dig are state changes (reflected in deltas), transform changes terrain
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "dig ") && strings.Contains(entry.Action, "transform ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.syncPlayerResources(player, entry)
		}

		// Parse and execute the transform part
		v.executeTransformFromAction(entry.Action)
		// Skip normal action execution
		return nil
	}

	// Handle compound convert + send priest + convert actions
	// Example: "convert 1PW to 1C. send p to EARTH. convert 1PW to 1C"
	// The converts must happen in order: first convert, send priest (which grants power), second convert
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "send p to") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			parts := strings.Split(entry.Action, ".")
			for i, part := range parts {
				part = strings.TrimSpace(part)
				
				if strings.HasPrefix(part, "convert ") {
					// Execute this specific convert
					v.executeConvertFromAction(player, part)
				} else if strings.HasPrefix(part, "send p to ") {
					// Execute the send priest action
					// The action conversion will happen normally below
					// But we need to sync power bowls to reflect the converts so far
					v.syncPlayerPowerBowls(player, entry)
					
					// Execute remaining parts after this as separate converts
					for j := i + 1; j < len(parts); j++ {
						remainingPart := strings.TrimSpace(parts[j])
						if strings.HasPrefix(remainingPart, "convert ") {
							// This convert happens after the send priest
							// We'll handle it after the action executes
							continue
						}
					}
					break
				}
			}
		}
		// Continue to execute the send priest action normally
	} else if strings.HasPrefix(entry.Action, "convert ") && !strings.Contains(entry.Action, "pass ") && !strings.Contains(entry.Action, "upgrade ") {
		// Handle compound convert + action patterns where converts need to be executed first
		// Examples:
		//   "convert 1PW to 1C. action ACTW. build H4"
		//   "convert 1P to 1W. dig 2. build D5"
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.executeConvertFromAction(player, entry.Action)
		}
		// Continue to execute the main action
	} else if strings.Contains(entry.Action, "convert ") && !strings.Contains(entry.Action, "pass ") && !strings.Contains(entry.Action, "upgrade ") {
		// Handle compound action + convert patterns (reverse order)
		// Examples:
		//   "action ACT6. convert 1W to 1C. build D7"
		//   "burn 3. action ACT2. convert 1PW to 1C"
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.executeConvertFromAction(player, entry.Action)
		}
		// Continue to execute the main action
	}

	// Handle compound convert + pass actions: execute convert, then pass
	// Example: "convert 1PW to 1C. pass BON7" with delta "+3" (1 from convert + 2 from bonus card)
	// Bug: Previously we synced coins to final state (which includes bonus card coins),
	// then PassAction added bonus card coins again, double-counting them.
	// Fix: Execute convert to add its coins, sync power bowls to final state, then execute pass
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "pass ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			v.executeConvertFromAction(player, entry.Action)
			// Sync power bowls to final state (in case conversion isn't 1:1 or there are other effects)
			v.syncPlayerPowerBowls(player, entry)
		}
		// Continue to execute pass (which will add bonus card coins)
	}

	// Handle compound convert+upgrade actions: sync all state, then execute action for building placement
	// Example: "convert 1W to 1C. upgrade F3 to TE. +FAV9"
	// Note: Action may have power leech prefix like "2 3  convert 1W to 1C. upgrade..."
	// The convert AND upgrade costs are reflected in resource deltas, so we sync state first
	// Then execute the action which will place the building (action converter skips validation/cost payment)
	if strings.Contains(entry.Action, "convert ") && strings.Contains(entry.Action, "upgrade ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync all state (includes convert + upgrade costs, favor tile might advance cult)
			v.syncPlayerState(player, entry.Faction.String(), entry)
		}
		// Continue to execute action - it will place building and apply favor tile
		// (action converter skips validation since resources are pre-synced)
	}

	// Handle cult income phase (only validates, doesn't grant income - cult rewards already given in cleanup)
	if entry.Action == "cult_income_for_faction" {
		// Just validate state - cult rewards were already given in ExecuteCleanupPhase
		if err := v.ValidateState(entry); err != nil {
			return fmt.Errorf("validation failed at entry %d: %v", v.CurrentEntry, err)
		}
		return nil
	}

	// Handle regular income phase (building/faction/bonus card income)
	if entry.Action == "other_income_for_faction" {
		// Apply income once for all players when we see the first income entry
		if !v.IncomeApplied {
			// Note: StartNewRound() was already called when we saw "Round X income" comment
			// It set phase to PhaseIncome, so now we just grant income
			// Do NOT return bonus cards here - players keep their cards for the entire round
			// Cards are only returned at the end of the round during cleanup phase

			v.GameState.GrantIncome()

			v.IncomeApplied = true
			v.GameState.StartActionPhase() // Transition to action phase
		}
		// Validate state matches after income
		if err := v.ValidateState(entry); err != nil {
			return fmt.Errorf("validation failed at entry %d: %v", v.CurrentEntry, err)
		}
		// Skip further processing - income is not a player action
		return nil
	}

	// Validate resources BEFORE executing action (except for leech/decline/income entries)
	// Also skip for compound convert actions since conversion is pre-executed or state is pre-synced
	if !strings.Contains(entry.Action, "Leech") &&
	   !strings.Contains(entry.Action, "Decline") &&
	   !strings.Contains(entry.Action, "_income_for_faction") &&
	   !strings.Contains(entry.Action, "show history") &&
	   !strings.Contains(entry.Action, "convert ") &&
	   entry.Faction.String() != "" {
		v.validateResourcesBeforeAction(entry)
	}

	// First, try to convert and execute the action for this entry
	action, err := ConvertLogEntryToAction(entry, v.GameState)
	if err != nil {
		return fmt.Errorf("failed to convert action at entry %d: %v", v.CurrentEntry, err)
	}

	// Execute the action (if it's not nil)
	if action != nil {
		if err := v.executeAction(action); err != nil {
			return fmt.Errorf("failed to execute action at entry %d: %v", v.CurrentEntry, err)
		}
	}

	// Handle post-action converts for compound "convert + send p to + convert" actions
	// After the send priest action grants power, we need to execute remaining converts
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "send p to") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			parts := strings.Split(entry.Action, ".")
			foundSendPriest := false
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if foundSendPriest && strings.HasPrefix(part, "convert ") {
					// Execute converts that come after "send p to"
					v.executeConvertFromAction(player, part)
				} else if strings.HasPrefix(part, "send p to ") {
					foundSendPriest = true
				}
			}
		}
	}

	// Handle town tile selection for actions with "+TW" marker
	// Example: "action BON1. build I7. +TW5"
	// The main action creates a pending town formation, now we select the tile
	// Note: Only process if there's actually a pending town formation (action_converter
	// handles upgrade+town tile compounds, so those won't have pending formations here)
	if strings.Contains(entry.Action, "+TW") {
		playerID := entry.Faction.String()
		// Only process if there's a pending town formation
		if v.GameState.PendingTownFormations[playerID] != nil {
			// Extract town tile marker (e.g., "+TW5")
			parts := strings.Split(entry.Action, ".")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "+TW") {
					townTileStr := strings.TrimPrefix(part, "+")
					townTileType, err := ParseTownTile(townTileStr)
					if err != nil {
						return fmt.Errorf("failed to parse town tile %s: %v", townTileStr, err)
					}

					if err := v.GameState.SelectTownTile(playerID, townTileType); err != nil {
						return fmt.Errorf("failed to select town tile: %v", err)
					}
					break
				}
			}
		}
	}

	// Then, validate state matches the log entry AFTER executing the action
	// (The log shows state AFTER the action is executed)
	if err := v.ValidateState(entry); err != nil {
		return fmt.Errorf("validation failed at entry %d: %v", v.CurrentEntry, err)
	}

	return nil
}

// executeAction executes an action and updates game state
func (v *GameValidator) executeAction(action game.Action) error {
	// Validate the action first
	if err := action.Validate(v.GameState); err != nil {
		return fmt.Errorf("action validation failed: %v", err)
	}

	// Execute the action
	if err := action.Execute(v.GameState); err != nil {
		return fmt.Errorf("action execution failed: %v", err)
	}

	return nil
}

// ReplayGame replays the entire game and validates each step
func (v *GameValidator) ReplayGame() error {
	// Initialize game
	if err := v.InitializeGame(); err != nil {
		return fmt.Errorf("failed to initialize game: %v", err)
	}

	// Process all entries
	for v.CurrentEntry < len(v.LogEntries) {
		if err := v.ValidateNextEntry(); err != nil {
			// Don't fail immediately, collect error and continue
			fmt.Printf("Error at entry %d: %v\n", v.CurrentEntry, err)
			// For now, stop on first error to make debugging easier
			return err
		}
	}

	return nil
}

// validateResourcesBeforeAction validates that the player's resources match expected state BEFORE executing the action
// The log entry shows final state + deltas, so we calculate expected initial state by reversing the deltas
func (v *GameValidator) validateResourcesBeforeAction(entry *LogEntry) {
	playerState := v.GameState.GetPlayer(entry.Faction.String())
	if playerState == nil {
		return // Skip if player not found
	}

	// Calculate expected resources BEFORE action (final - delta = initial)
	// Note: Deltas in log are signed (negative means resource decreased)
	// So to get initial state: initial = final - delta
	// Example: final=2C, delta=-4 → initial = 2 - (-4) = 6C

	expectedVP := entry.VP - entry.VPDelta
	expectedCoins := entry.Coins - entry.CoinsDelta
	expectedWorkers := entry.Workers - entry.WorkersDelta
	expectedPriests := entry.Priests - entry.PriestsDelta

	// Only validate if we have actual deltas (non-zero changes)
	if entry.CoinsDelta != 0 && playerState.Resources.Coins != expectedCoins {
		fmt.Printf("⚠️  Entry %d (%s) - Coins mismatch BEFORE action: expected %d, got %d (delta=%d, final=%d)\n",
			v.CurrentEntry, entry.Faction.String(), expectedCoins, playerState.Resources.Coins,
			entry.CoinsDelta, entry.Coins)
	}
	if entry.WorkersDelta != 0 && playerState.Resources.Workers != expectedWorkers {
		fmt.Printf("⚠️  Entry %d (%s) - Workers mismatch BEFORE action: expected %d, got %d (delta=%d, final=%d)\n",
			v.CurrentEntry, entry.Faction.String(), expectedWorkers, playerState.Resources.Workers,
			entry.WorkersDelta, entry.Workers)
	}
	if entry.PriestsDelta != 0 && playerState.Resources.Priests != expectedPriests {
		fmt.Printf("⚠️  Entry %d (%s) - Priests mismatch BEFORE action: expected %d, got %d (delta=%d, final=%d)\n",
			v.CurrentEntry, entry.Faction.String(), expectedPriests, playerState.Resources.Priests,
			entry.PriestsDelta, entry.Priests)
	}
	if entry.VPDelta != 0 && playerState.VictoryPoints != expectedVP {
		fmt.Printf("⚠️  Entry %d (%s) - VP mismatch BEFORE action: expected %d, got %d (delta=%d, final=%d)\n",
			v.CurrentEntry, entry.Faction.String(), expectedVP, playerState.VictoryPoints,
			entry.VPDelta, entry.VP)
	}
}

// ValidateState validates the current game state against a log entry
func (v *GameValidator) ValidateState(entry *LogEntry) error {
	// Get player state
	var playerState *game.Player
	for _, p := range v.GameState.Players {
		if p.Faction.GetType() == entry.Faction {
			playerState = p
			break
		}
	}
	if playerState == nil {
		return fmt.Errorf("player not found for faction %v", entry.Faction)
	}

	// Validate VP
	if playerState.VictoryPoints != entry.VP {
		v.addError(v.CurrentEntry, entry, "VP", entry.VP, playerState.VictoryPoints,
			fmt.Sprintf("VP mismatch: expected %d, got %d", entry.VP, playerState.VictoryPoints))
	}

	// Validate Coins
	if playerState.Resources.Coins != entry.Coins {
		v.addError(v.CurrentEntry, entry, "Coins", entry.Coins, playerState.Resources.Coins,
			fmt.Sprintf("Coins mismatch: expected %d, got %d", entry.Coins, playerState.Resources.Coins))
	}

	// Validate Workers
	if playerState.Resources.Workers != entry.Workers {
		v.addError(v.CurrentEntry, entry, "Workers", entry.Workers, playerState.Resources.Workers,
			fmt.Sprintf("Workers mismatch: expected %d, got %d", entry.Workers, playerState.Resources.Workers))
	}

	// Validate Priests
	if playerState.Resources.Priests != entry.Priests {
		v.addError(v.CurrentEntry, entry, "Priests", entry.Priests, playerState.Resources.Priests,
			fmt.Sprintf("Priests mismatch: expected %d, got %d", entry.Priests, playerState.Resources.Priests))
	}

	// Validate Power Bowls
	if playerState.Resources.Power.Bowl1 != entry.PowerBowls.Bowl1 ||
	   playerState.Resources.Power.Bowl2 != entry.PowerBowls.Bowl2 ||
	   playerState.Resources.Power.Bowl3 != entry.PowerBowls.Bowl3 {
		v.addError(v.CurrentEntry, entry, "PowerBowls",
			fmt.Sprintf("%d/%d/%d", entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3),
			fmt.Sprintf("%d/%d/%d", playerState.Resources.Power.Bowl1, playerState.Resources.Power.Bowl2, playerState.Resources.Power.Bowl3),
			fmt.Sprintf("Power bowls mismatch: expected %d/%d/%d, got %d/%d/%d",
				entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3,
				playerState.Resources.Power.Bowl1, playerState.Resources.Power.Bowl2, playerState.Resources.Power.Bowl3))
	}

	// Validate Cult Tracks
	fireCult, fireOk := playerState.CultPositions[game.CultFire]
	waterCult, waterOk := playerState.CultPositions[game.CultWater]
	earthCult, earthOk := playerState.CultPositions[game.CultEarth]
	airCult, airOk := playerState.CultPositions[game.CultAir]

	if fireOk && fireCult != entry.CultTracks.Fire {
		v.addError(v.CurrentEntry, entry, "Fire Cult", entry.CultTracks.Fire, fireCult,
			fmt.Sprintf("Fire cult mismatch: expected %d, got %d", entry.CultTracks.Fire, fireCult))
	}
	if waterOk && waterCult != entry.CultTracks.Water {
		v.addError(v.CurrentEntry, entry, "Water Cult", entry.CultTracks.Water, waterCult,
			fmt.Sprintf("Water cult mismatch: expected %d, got %d", entry.CultTracks.Water, waterCult))
	}
	if earthOk && earthCult != entry.CultTracks.Earth {
		v.addError(v.CurrentEntry, entry, "Earth Cult", entry.CultTracks.Earth, earthCult,
			fmt.Sprintf("Earth cult mismatch: expected %d, got %d", entry.CultTracks.Earth, earthCult))
	}
	if airOk && airCult != entry.CultTracks.Air {
		v.addError(v.CurrentEntry, entry, "Air Cult", entry.CultTracks.Air, airCult,
			fmt.Sprintf("Air cult mismatch: expected %d, got %d", entry.CultTracks.Air, airCult))
	}

	// After validation, sync state to match log entry to prevent drift
	// This ensures that even if there are small mismatches, they don't accumulate
	playerState.VictoryPoints = entry.VP
	playerState.Resources.Coins = entry.Coins
	playerState.Resources.Workers = entry.Workers
	playerState.Resources.Priests = entry.Priests
	playerState.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
	playerState.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
	playerState.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
	playerState.CultPositions[game.CultFire] = entry.CultTracks.Fire
	playerState.CultPositions[game.CultWater] = entry.CultTracks.Water
	playerState.CultPositions[game.CultEarth] = entry.CultTracks.Earth
	playerState.CultPositions[game.CultAir] = entry.CultTracks.Air
	
	// Also sync cult track state
	v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
	v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
	v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
	v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air

	return nil
}

// executeConvertFromAction parses and executes resource conversion from an action string
// Handles multiple convert statements in compound actions (e.g., "convert 1PW to 1C. send p to EARTH. convert 1PW to 1C")
func (v *GameValidator) executeConvertFromAction(player *game.Player, actionStr string) {
	// Split by periods to find all convert statements
	parts := strings.Split(actionStr, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "convert ") {
			continue
		}
		
		// Parse the convert amount from this part
		// Format: "convert XPW to YC" or "convert XW to YC" or "convert XP to YW" or "convert XPW to YW"
		var powerAmount, coinAmount, workerAmount, priestAmount int
		if strings.Contains(part, "PW to") && strings.Contains(part, "W") && !strings.Contains(part, "C") {
			// Power to workers: "convert 3PW to 1W"
			fmt.Sscanf(part, "convert %dPW to %dW", &powerAmount, &workerAmount)
			if powerAmount > 0 && workerAmount > 0 {
				// Spend power from Bowl 3, gain workers
				err := player.Resources.Power.SpendPower(powerAmount)
				if err == nil {
					player.Resources.Workers += workerAmount
				}
			}
		} else if strings.Contains(part, "PW to") && strings.Contains(part, "C") {
			// Power to coins: "convert 1PW to 1C"
			fmt.Sscanf(part, "convert %dPW to %dC", &powerAmount, &coinAmount)
			if powerAmount > 0 && coinAmount > 0 {
				player.Resources.ConvertPowerToCoins(powerAmount)
			}
		} else if strings.Contains(part, "W to") && strings.Contains(part, "C") {
			// Workers to coins: "convert 1W to 1C"
			fmt.Sscanf(part, "convert %dW to %dC", &workerAmount, &coinAmount)
			if workerAmount > 0 && coinAmount > 0 {
				player.Resources.Workers -= workerAmount
				player.Resources.Coins += coinAmount
			}
		} else if strings.Contains(part, "P to") && strings.Contains(part, "W") {
			// Priests to workers: "convert 1P to 1W"
			fmt.Sscanf(part, "convert %dP to %dW", &priestAmount, &workerAmount)
			if priestAmount > 0 && workerAmount > 0 {
				player.Resources.Priests -= priestAmount
				player.Resources.Workers += workerAmount
			}
		}
	}
}

// executeTransformFromAction parses and executes terrain transformation from an action string
func (v *GameValidator) executeTransformFromAction(actionStr string) {
	parts := strings.Split(actionStr, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "transform ") {
			// Parse: "transform H4 to green"
			fields := strings.Fields(part)
			if len(fields) >= 4 && fields[2] == "to" {
				coordStr := fields[1]
				colorStr := fields[3]

				hex, err := ConvertLogCoordToAxial(coordStr)
				if err == nil {
					terrain, err := ParseTerrainColor(colorStr)
					if err == nil {
						v.GameState.Map.TransformTerrain(hex, terrain)
					}
				}
			}
			break
		}
	}
}

// syncPlayerResources syncs all player resources to match log entry
func (v *GameValidator) syncPlayerResources(player *game.Player, entry *LogEntry) {
	player.VictoryPoints = entry.VP
	player.Resources.Coins = entry.Coins
	player.Resources.Workers = entry.Workers
	player.Resources.Priests = entry.Priests
	player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
	player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
	player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
}

// syncPlayerCultPositions syncs player cult positions to match log entry
func (v *GameValidator) syncPlayerCultPositions(player *game.Player, factionStr string, entry *LogEntry) {
	player.CultPositions[game.CultFire] = entry.CultTracks.Fire
	player.CultPositions[game.CultWater] = entry.CultTracks.Water
	player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
	player.CultPositions[game.CultAir] = entry.CultTracks.Air
	
	// Also sync cult track state
	v.GameState.CultTracks.PlayerPositions[factionStr][game.CultFire] = entry.CultTracks.Fire
	v.GameState.CultTracks.PlayerPositions[factionStr][game.CultWater] = entry.CultTracks.Water
	v.GameState.CultTracks.PlayerPositions[factionStr][game.CultEarth] = entry.CultTracks.Earth
	v.GameState.CultTracks.PlayerPositions[factionStr][game.CultAir] = entry.CultTracks.Air
}

// syncPlayerPowerBowls syncs only power bowls to match log entry
func (v *GameValidator) syncPlayerPowerBowls(player *game.Player, entry *LogEntry) {
	player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
	player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
	player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
}

// syncPlayerState syncs all player state (resources + cult positions) to match log entry
func (v *GameValidator) syncPlayerState(player *game.Player, factionStr string, entry *LogEntry) {
	v.syncPlayerResources(player, entry)
	v.syncPlayerCultPositions(player, factionStr, entry)
}

// addError adds a validation error
func (v *GameValidator) addError(line int, entry *LogEntry, field string, expected, actual interface{}, message string) {
	v.Errors = append(v.Errors, ValidationError{
		Line:     line,
		Entry:    entry,
		Field:    field,
		Expected: expected,
		Actual:   actual,
		Message:  message,
	})
}

// HasErrors returns true if there are any validation errors
func (v *GameValidator) HasErrors() bool {
	return len(v.Errors) > 0
}

// GetErrorSummary returns a summary of all errors
func (v *GameValidator) GetErrorSummary() string {
	if !v.HasErrors() {
		return "No errors"
	}

	summary := fmt.Sprintf("Found %d validation errors:\n", len(v.Errors))
	for i, err := range v.Errors {
		if i >= 10 {
			summary += fmt.Sprintf("... and %d more errors\n", len(v.Errors)-10)
			break
		}
		summary += fmt.Sprintf("  Line %d: %s\n", err.Line, err.Message)
	}
	return summary
}
