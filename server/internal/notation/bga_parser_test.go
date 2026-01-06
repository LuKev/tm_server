package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

func TestBGAParser(t *testing.T) {
	// Inline log content for test
	content := `Game board: Base Game
Mini-expansions: On
deragned selected the faction Halflings on plains to play in position #1
kezilu selected the faction Auren on forest to play in position #2
Redrame selected the faction Chaos Magicians on wasteland to play in position #3
Locky_91 selected the faction Alchemists on swamp to play in position #4
deragned is playing the Halflings Faction (with 32 VP Starting VPs)
Redrame is playing the Auren Faction (with 40 VP Starting VPs)
kezilu is playing the Chaos Magicians Faction (with 40 VP Starting VPs)
Locky_91 is playing the Alchemists Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
deragned places a Dwelling [F5]
Redrame places a Dwelling [F4]
Locky_91 places a Dwelling [E5]
Locky_91 places a Dwelling [B5]
Redrame places a Dwelling [C3]
deragned places a Dwelling [E6]
kezilu places a Dwelling [D4]
~ Action phase ~
`

	parser := NewBGAParser(string(content))
	actions, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	if len(actions) == 0 {
		t.Errorf("Expected actions, got 0")
	}

	// Print first few actions for inspection
	t.Logf("Parsed %d actions", len(actions))
	for i, action := range actions {
		if i >= 20 {
			break
		}
		t.Logf("Action %d: %T %+v", i, action, action)
	}

	// Verify specific known actions
	// Move 54: felipebart places a Dwelling [F3] -> SetupDwellingAction
	foundSetup := false
	for _, item := range actions {
		if actionItem, ok := item.(ActionItem); ok {
			if _, ok := actionItem.Action.(*game.SetupDwellingAction); ok {
				foundSetup = true
				break
			}
		}
	}
	if !foundSetup {
		t.Errorf("Did not find SetupDwellingAction")
	}

	// Check Round 1 TurnOrder
	foundRound1 := false
	for _, item := range actions {
		if rs, ok := item.(RoundStartItem); ok && rs.Round == 1 {
			foundRound1 = true
			t.Logf("Round 1 TurnOrder: %v", rs.TurnOrder)
			// Check for duplicates
			seen := make(map[string]bool)
			for _, p := range rs.TurnOrder {
				if seen[p] {
					t.Errorf("Duplicate player in TurnOrder: %s", p)
				}
				seen[p] = true
			}
			if len(rs.TurnOrder) != 4 {
				t.Errorf("Expected 4 players in TurnOrder, got %d", len(rs.TurnOrder))
			}
		}
	}
	if !foundRound1 {
		t.Errorf("Did not find Round 1 Start")
	}
}

func TestBGAParserSettings(t *testing.T) {
	content := `Game board: Base Game
Round 1 scoring: SCORE2, TOWN >> 5
Round 2 scoring: SCORE9, TE >> 4
Round 3 scoring: SCORE4, SH/SA >> 5
Round 4 scoring: SCORE1, SPADE >> 2
Round 5 scoring: SCORE6, TP >> 3
Round 6 scoring: SCORE7, SH/SA >> 5
Removing tile BON9
Removing tile BON2
Removing tile BON5
deragned is playing the Halflings Faction
~ Every player has chosen a Faction ~
`
	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var settings *GameSettingsItem
	for _, item := range items {
		if s, ok := item.(GameSettingsItem); ok {
			settings = &s
			break
		}
	}

	if settings == nil {
		t.Fatal("GameSettingsItem not found")
	}

	// Check ScoringTiles
	expectedTiles := "SCORE2,SCORE9,SCORE4,SCORE1,SCORE6,SCORE7"
	if settings.Settings["ScoringTiles"] != expectedTiles {
		t.Errorf("Expected ScoringTiles %q, got %q", expectedTiles, settings.Settings["ScoringTiles"])
	}

	// Check BonusCards
	// Removed: BON9 (BON-DW), BON2 (BON-4C), BON5 (BON-WP)
	// All: SPD, 4C, 6C, SHIP, WP, BB, TP, P, DW, SHIP-VP
	// Expected: SPD, 6C, SHIP, BB, TP, P, SHIP-VP
	// (Order depends on allBonusCodes order in parser)
	// allBonusCodes: SPD, 4C, 6C, SHIP, WP, BB, TP, P, DW, SHIP-VP
	// Removed: 4C, WP, DW
	// Remaining: SPD, 6C, SHIP, BB, TP, P, SHIP-VP
	expectedBonus := "BON-SPD,BON-6C,BON-SHIP,BON-BB,BON-TP,BON-P,BON-SHIP-VP"
	if settings.Settings["BonusCards"] != expectedBonus {
		t.Errorf("Expected BonusCards %q, got %q", expectedBonus, settings.Settings["BonusCards"])
	}
}
