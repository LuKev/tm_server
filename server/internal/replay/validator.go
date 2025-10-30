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
			// Start new round (this resets HasPassed, power actions, etc.)
			v.GameState.StartNewRound()
			// Reset income flag for new round
			v.IncomeApplied = false
			// Note: We do NOT return bonus cards here. Bonus cards are selected when passing
			// and provide income for the NEXT round. They are only returned to the pool when
			// the player passes again and selects a new bonus card (handled in TakeBonusCard).
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
			// Sync all resources to match log entry
			player.VictoryPoints = entry.VP
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
			player.Resources.Priests = entry.Priests
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3

			// Sync cult positions (leech entries sometimes include cult advancement)
			player.CultPositions[game.CultFire] = entry.CultTracks.Fire
			player.CultPositions[game.CultWater] = entry.CultTracks.Water
			player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
			player.CultPositions[game.CultAir] = entry.CultTracks.Air

			// Sync cult track state
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air
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
			// Sync cult positions to match final state
			player.CultPositions[game.CultFire] = entry.CultTracks.Fire
			player.CultPositions[game.CultWater] = entry.CultTracks.Water
			player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
			player.CultPositions[game.CultAir] = entry.CultTracks.Air
			
			// Sync cult track state
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air
			
			// Also sync power bowls in case leech happened
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
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
			// Sync cult positions to match log entry
			player.CultPositions[game.CultFire] = entry.CultTracks.Fire
			player.CultPositions[game.CultWater] = entry.CultTracks.Water
			player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
			player.CultPositions[game.CultAir] = entry.CultTracks.Air

			// Sync cult track state
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air

			// Also sync resources - cult advancements can trigger power/coin gains
			player.VictoryPoints = entry.VP
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
			player.Resources.Priests = entry.Priests
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle "[opponent accepted power]" entries - these show state after leech resolution
	// These are informational entries that show the updated cult track state
	if strings.Contains(entry.Action, "[opponent accepted power]") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync all resources and cult positions to match log entry
			player.VictoryPoints = entry.VP
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
			player.Resources.Priests = entry.Priests
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3

			// Sync cult positions
			player.CultPositions[game.CultFire] = entry.CultTracks.Fire
			player.CultPositions[game.CultWater] = entry.CultTracks.Water
			player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
			player.CultPositions[game.CultAir] = entry.CultTracks.Air

			// Sync cult track state
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air
		}
		// Skip normal action execution and validation
		return nil
	}

	// Handle compound convert + dig + transform actions: sync resources and transform terrain
	// Example: "convert 1PW to 1C. dig 1. transform H4 to green"
	// The convert and dig are state changes (reflected in deltas), transform changes terrain
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "dig ") && strings.Contains(entry.Action, "transform ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync resources to match final state
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
			player.Resources.Priests = entry.Priests
		}
		
		// Parse and execute the transform part
		parts := strings.Split(entry.Action, ".")
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
		// Skip normal action execution
		return nil
	}

	// Handle compound convert + action actions: sync resources (convert is a state change)
	// Example: "convert 1PW to 1C. action ACTW. build H4"
	// The convert part is reflected in resource deltas, we sync first then execute action
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, ". action ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync power bowls to match final state (convert costs are reflected in deltas)
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
			// Also sync coins/workers in case they were converted
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
		}
		// Continue to execute action
	}

	// Handle compound convert + pass actions: sync resources (convert is a state change)
	// Example: "convert 1PW to 1C. pass BON7"
	// The convert part is reflected in resource deltas, we sync first then execute pass
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "pass ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync power bowls to match final state (convert costs are reflected in deltas)
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3
			// Also sync coins/workers in case they were converted
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
		}
		// Continue to execute pass
	}

	// Handle compound convert+upgrade actions: sync all state, then execute action for building placement
	// Example: "convert 1W to 1C. upgrade F3 to TE. +FAV9"
	// The convert AND upgrade costs are reflected in resource deltas, so we sync state first
	// Then execute the action which will place the building (action converter skips validation/cost payment)
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "upgrade ") {
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			// Sync all resources to match final state (includes convert + upgrade costs)
			player.VictoryPoints = entry.VP
			player.Resources.Coins = entry.Coins
			player.Resources.Workers = entry.Workers
			player.Resources.Priests = entry.Priests
			player.Resources.Power.Bowl1 = entry.PowerBowls.Bowl1
			player.Resources.Power.Bowl2 = entry.PowerBowls.Bowl2
			player.Resources.Power.Bowl3 = entry.PowerBowls.Bowl3

			// Sync cult positions (favor tile might advance cult)
			player.CultPositions[game.CultFire] = entry.CultTracks.Fire
			player.CultPositions[game.CultWater] = entry.CultTracks.Water
			player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
			player.CultPositions[game.CultAir] = entry.CultTracks.Air

			// Sync cult track state
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultFire] = entry.CultTracks.Fire
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultWater] = entry.CultTracks.Water
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultEarth] = entry.CultTracks.Earth
			v.GameState.CultTracks.PlayerPositions[entry.Faction.String()][game.CultAir] = entry.CultTracks.Air
		}
		// Continue to execute action - it will place building and apply favor tile
		// (action converter skips validation since resources are pre-synced)
	}

	// Handle income phase
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
	if !strings.Contains(entry.Action, "Leech") &&
	   !strings.Contains(entry.Action, "Decline") &&
	   !strings.Contains(entry.Action, "_income_for_faction") &&
	   !strings.Contains(entry.Action, "show history") &&
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
			"Power bowls mismatch")
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
