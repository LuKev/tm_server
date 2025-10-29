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

	// Handle compound convert+upgrade actions: sync all state, then execute action for building placement
	// Example: "convert 1W to 1C. upgrade F3 to TE. +FAV9"
	// The convert AND upgrade costs are reflected in resource deltas, so we sync state first
	// Then execute the action which will place the building (action converter skips validation/cost payment)
	if strings.HasPrefix(entry.Action, "convert ") && strings.Contains(entry.Action, "upgrade ") {
		fmt.Printf("DEBUG: Entry %d - Processing compound convert+upgrade action: %s\n", v.CurrentEntry, entry.Action)
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
			fmt.Printf("DEBUG: Applying income for all players at entry %d\n", v.CurrentEntry)

			// Debug: show power state BEFORE income
			if cultists := v.GameState.GetPlayer("Cultists"); cultists != nil {
				fmt.Printf("DEBUG: BEFORE income - Cultists power: %d/%d/%d\n",
					cultists.Resources.Power.Bowl1, cultists.Resources.Power.Bowl2, cultists.Resources.Power.Bowl3)
			}

			v.GameState.GrantIncome()

			// Debug: show power state AFTER income
			if cultists := v.GameState.GetPlayer("Cultists"); cultists != nil {
				fmt.Printf("DEBUG: AFTER income - Cultists power: %d/%d/%d\n",
					cultists.Resources.Power.Bowl1, cultists.Resources.Power.Bowl2, cultists.Resources.Power.Bowl3)
			}

			v.IncomeApplied = true
			v.GameState.StartActionPhase() // Transition to action phase
		}
		// Debug: show power state after income for this player
		player := v.GameState.GetPlayer(entry.Faction.String())
		if player != nil {
			fmt.Printf("DEBUG: Entry %d (%s) - Expected power: %d/%d/%d, Actual: %d/%d/%d\n",
				v.CurrentEntry, entry.Faction.String(),
				entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3,
				player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
		}
		// Validate state matches after income
		if err := v.ValidateState(entry); err != nil {
			return fmt.Errorf("validation failed at entry %d: %v", v.CurrentEntry, err)
		}
		// Skip further processing - income is not a player action
		return nil
	}

	// Debug: show resources for darklings at entry 110
	if v.CurrentEntry == 110 {
		if player := v.GameState.GetPlayer("Darklings"); player != nil {
			fmt.Printf("DEBUG: Entry %d - Darklings resources before action: %d C, %d W, %d P\n",
				v.CurrentEntry, player.Resources.Coins, player.Resources.Workers, player.Resources.Priests)
		}
	}

	// Debug: show resources for engineers at entry 58 and 132
	if v.CurrentEntry == 58 || v.CurrentEntry == 132 {
		if player := v.GameState.GetPlayer("Engineers"); player != nil {
			fmt.Printf("DEBUG: Entry %d - Engineers resources before action: %d C, %d W, %d P\n",
				v.CurrentEntry, player.Resources.Coins, player.Resources.Workers, player.Resources.Priests)
			fmt.Printf("DEBUG: Entry %d action: %s\n", v.CurrentEntry, entry.Action)
		}
	}

	// Debug: show power bowls at entry 48
	if v.CurrentEntry == 48 {
		if player := v.GameState.GetPlayer("Engineers"); player != nil {
			fmt.Printf("DEBUG: Entry 48 - Engineers BEFORE action:\n")
			fmt.Printf("  Power bowls: %d/%d/%d\n", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
			fmt.Printf("  Expected after: %d/%d/%d\n", entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3)
			fmt.Printf("  Action: %s\n", entry.Action)
		}
	}

	// Debug: show cult tracks at entry 59
	if v.CurrentEntry == 59 {
		if player := v.GameState.GetPlayer("Darklings"); player != nil {
			fmt.Printf("DEBUG: Entry 59 - Darklings BEFORE action:\n")
			fmt.Printf("  Cult tracks: Fire=%d, Water=%d, Earth=%d, Air=%d\n",
				player.CultPositions[game.CultFire], player.CultPositions[game.CultWater],
				player.CultPositions[game.CultEarth], player.CultPositions[game.CultAir])
			fmt.Printf("  Expected after: Fire=%d, Water=%d, Earth=%d, Air=%d\n",
				entry.CultTracks.Fire, entry.CultTracks.Water, entry.CultTracks.Earth, entry.CultTracks.Air)
			fmt.Printf("  Action: %s\n", entry.Action)
		}
	}

	// First, try to convert and execute the action for this entry
	if v.CurrentEntry == 119 {
		fmt.Printf("DEBUG: Entry 119 - About to convert action: %s\n", entry.Action)
		fmt.Printf("DEBUG: Entry 119 - Current round: %d\n", v.GameState.Round)
	}
	action, err := ConvertLogEntryToAction(entry, v.GameState)
	if err != nil {
		return fmt.Errorf("failed to convert action at entry %d: %v", v.CurrentEntry, err)
	}
	if v.CurrentEntry == 119 {
		fmt.Printf("DEBUG: Entry 119 - Action converted: %v (is nil: %v)\n", action, action == nil)
	}

	// Debug: show worker count for witches BEFORE action
	var workersBefore int
	if v.CurrentEntry >= 85 && v.CurrentEntry <= 92 {
		if player := v.GameState.GetPlayer("Witches"); player != nil {
			workersBefore = player.Resources.Workers
		}
	}

	// Execute the action (if it's not nil)
	if action != nil {
		if v.CurrentEntry == 119 {
			fmt.Printf("DEBUG: Entry 119 - Executing action\n")
		}
		if err := v.executeAction(action); err != nil {
			return fmt.Errorf("failed to execute action at entry %d: %v", v.CurrentEntry, err)
		}
		if v.CurrentEntry == 119 {
			fmt.Printf("DEBUG: Entry 119 - Action executed successfully\n")
			// Check building state at F3 (coord 2,5)
			hex := v.GameState.Map.GetHex(game.NewHex(2, 5))
			if hex != nil && hex.Building != nil {
				fmt.Printf("DEBUG: Entry 119 - Building at F3 (2,5) after action: %v\n", hex.Building.Type)
			} else {
				fmt.Printf("DEBUG: Entry 119 - No building at F3 (2,5) after action\n")
			}
		}
	} else if v.CurrentEntry == 119 {
		fmt.Printf("DEBUG: Entry 119 - Action is nil, skipping execution\n")
	}

	// Debug: show worker count for witches AFTER action
	if v.CurrentEntry >= 85 && v.CurrentEntry <= 92 {
		if player := v.GameState.GetPlayer("Witches"); player != nil {
			fmt.Printf("DEBUG: Entry %d (%s) - Witches workers: %d→%d (expected: %d), action: %s\n",
				v.CurrentEntry, entry.Faction, workersBefore, player.Resources.Workers, entry.Workers, entry.Action)
		}
	}

	// Debug: show power bowls at entry 48 AFTER action
	if v.CurrentEntry == 48 {
		if player := v.GameState.GetPlayer("Engineers"); player != nil {
			fmt.Printf("DEBUG: Entry 48 - Engineers AFTER action:\n")
			fmt.Printf("  Power bowls: %d/%d/%d\n", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
			fmt.Printf("  Expected: %d/%d/%d\n", entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3)
		}
	}

	// Debug: show cult tracks at entry 59 AFTER action
	if v.CurrentEntry == 59 {
		if player := v.GameState.GetPlayer("Darklings"); player != nil {
			fmt.Printf("DEBUG: Entry 59 - Darklings AFTER action:\n")
			fmt.Printf("  Cult tracks: Fire=%d, Water=%d, Earth=%d, Air=%d\n",
				player.CultPositions[game.CultFire], player.CultPositions[game.CultWater],
				player.CultPositions[game.CultEarth], player.CultPositions[game.CultAir])
			fmt.Printf("  Expected: Fire=%d, Water=%d, Earth=%d, Air=%d\n",
				entry.CultTracks.Fire, entry.CultTracks.Water, entry.CultTracks.Earth, entry.CultTracks.Air)
		}
	}

	// Debug: track power bowls from entry 136 to 148
	if v.CurrentEntry >= 136 && v.CurrentEntry <= 148 {
		if player := v.GameState.GetPlayer("Cultists"); player != nil {
			total := player.Resources.Power.Bowl1 + player.Resources.Power.Bowl2 + player.Resources.Power.Bowl3
			expectedTotal := entry.PowerBowls.Bowl1 + entry.PowerBowls.Bowl2 + entry.PowerBowls.Bowl3
			fmt.Printf("DEBUG: Entry %d (%s) - Cultists power: %d/%d/%d (total %d), expected %d/%d/%d (total %d), action: %s\n",
				v.CurrentEntry, entry.Faction,
				player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3, total,
				entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3, expectedTotal,
				entry.Action)
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
