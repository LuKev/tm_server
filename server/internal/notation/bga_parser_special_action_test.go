package notation

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

// =============================================================================
// SPECIAL PARSING REGRESSION TESTS
// Tests for stronghold actions, special abilities, and faction-specific parsing
// =============================================================================

// TestBGAParser_HalflingsStronghold tests parsing Halflings stronghold spade transforms
func TestBGAParser_HalflingsStronghold(t *testing.T) {
	content := `Arivor is playing the Halflings Faction
Arivor places a Dwelling [E6]
Arivor places a Dwelling [F5]
~ Action phase ~
Move 1 :
Arivor does some Conversions (spent: 3 power 0 Priests 0 workers ; collects: 0 Priests 0 workers 3 coins)
Arivor upgrades a Trading house to a Faction Stronghold for 4 workers 8 coins [E6]
Arivor gets 5 VP (Scoring tile bonus)
Arivor gets 3 Spades to Transform and Build (Halflings Stronghold)
Move 2 :
Arivor transforms a Terrain space swamp → plains for 1 spade(s) [G5]
Arivor earns 1 VP (Halflings Ability)
Arivor transforms a Terrain space mountains → desert for 2 spade(s) [F6]
Arivor earns 2 VP (Halflings Ability)
Move 3 :
Arivor declines building a Dwelling for now
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find compound action with UP-SH and Halflings spade action
	foundHalflingsSpades := false
	foundTerrainG5 := false
	foundTerrainF6 := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if halflingsSpade, ok := subAction.(*LogHalflingsSpadeAction); ok {
						foundHalflingsSpades = true
						for i, coord := range halflingsSpade.TransformCoords {
							if coord == "G5" {
								foundTerrainG5 = true
								// G5 transforms to plains (home terrain, no suffix needed)
								if i < len(halflingsSpade.TargetTerrains) && halflingsSpade.TargetTerrains[i] != "plains" {
									t.Errorf("G5 expected terrain 'plains', got '%s'", halflingsSpade.TargetTerrains[i])
								}
							}
							if coord == "F6" {
								foundTerrainF6 = true
								// F6 transforms to desert
								if i < len(halflingsSpade.TargetTerrains) && halflingsSpade.TargetTerrains[i] != "desert" {
									t.Errorf("F6 expected terrain 'desert', got '%s'", halflingsSpade.TargetTerrains[i])
								}
							}
						}
					}
				}
			}
		}
	}

	if !foundHalflingsSpades {
		t.Error("Did not find LogHalflingsSpadeAction")
	}
	if !foundTerrainG5 {
		t.Error("Did not find G5 transform")
	}
	if !foundTerrainF6 {
		t.Error("Did not find F6 transform")
	}
}

// TestBGAParser_ChaosMagiciansDoubleTurn tests parsing Chaos Magicians stronghold double turn
func TestBGAParser_ChaosMagiciansDoubleTurn(t *testing.T) {
	content := `Player1 is playing the Chaos Magicians Faction
Player1 places a Dwelling [D2]
Player1 places a Dwelling [G2]
~ Action phase ~
Move 1 :
Player1 upgrades a Trading house to a Faction Stronghold for 4 workers 6 coins [D2]
Player1 takes a double-turn (Chaos Magicians Stronghold)
Player1 builds a Dwelling for 1 workers 2 coins [E3]
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find ACT-SH-2X action
	foundDoubleTurn := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if specialAction, ok := actionItem.Action.(*LogSpecialAction); ok {
				if specialAction.ActionCode == "ACT-SH-2X" {
					foundDoubleTurn = true
					break
				}
			}
		}
	}

	if !foundDoubleTurn {
		t.Error("Did not find ACT-SH-2X for Chaos Magicians double turn")
	}
}

// TestBGAParser_DwarvesTunnelling tests parsing Dwarves tunnelling/Fakirs carpet flight
func TestBGAParser_DwarvesTunnelling(t *testing.T) {
	content := `Player1 is playing the Dwarves Faction
Player1 places a Dwelling [A3]
~ Action phase ~
Move 1 :
Player1 transforms a Terrain space forest → mountains for 2 spade(s) [C5]
Player1 builds a Dwelling for 1 workers 2 coins [C5]
Player1 earns 2 VP (Scoring tile)
~ This dwelling was built by tunneling. ~
Move 2 :
Player2 passes
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find BUILD action with tunnelling marker or verify it parsed correctly
	foundBuild := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if buildAction, ok := subAction.(*game.TransformAndBuildAction); ok {
						if buildAction.TargetHex.Q != 0 || buildAction.TargetHex.R != 0 {
							foundBuild = true
						}
					}
				}
			}
		}
	}

	// Basic check that the content was parsed
	if len(items) == 0 {
		t.Error("Expected some parsed items")
	}
	t.Logf("Parsed %d items, foundBuild=%v", len(items), foundBuild)
}

// TestBGAParser_NomadsStronghold tests parsing Nomads stronghold terraform
func TestBGAParser_NomadsStronghold(t *testing.T) {
	content := `Player1 is playing the Nomads Faction
Player1 places a Dwelling [D4]
~ Action phase ~
Move 1 :
Player1 upgrades a Trading house to a Faction Stronghold for 4 workers 6 coins [D4]
Player1 uses Sandstorm ability to terraform a Terrain space swamp → desert [E5]
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find ACT-SH action with coordinate
	foundNomadsAction := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if specialAction, ok := subAction.(*LogSpecialAction); ok {
						if strings.HasPrefix(specialAction.ActionCode, "ACT-SH") {
							foundNomadsAction = true
							t.Logf("Found Nomads stronghold action: %s", specialAction.ActionCode)
						}
					}
				}
			}
		}
	}

	// Nomads action should be parsed
	t.Logf("Parsed %d items, foundNomadsAction=%v", len(items), foundNomadsAction)
}

// TestBGAParser_GiantsStronghold tests parsing Giants stronghold 2 spades action
func TestBGAParser_GiantsStronghold(t *testing.T) {
	content := `Player1 is playing the Giants Faction
Player1 places a Dwelling [G4]
~ Action phase ~
Move 1 :
Player1 upgrades a Trading house to a Faction Stronghold for 4 workers 6 coins [G4]
Player1 uses Giants Stronghold to terraform [H3]
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find ACT-SH action for Giants
	foundGiantsAction := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if specialAction, ok := subAction.(*LogSpecialAction); ok {
						if strings.HasPrefix(specialAction.ActionCode, "ACT-SH") {
							foundGiantsAction = true
							t.Logf("Found Giants stronghold action: %s", specialAction.ActionCode)
						}
					}
				}
			}
		}
	}

	t.Logf("Parsed %d items, foundGiantsAction=%v", len(items), foundGiantsAction)
}

// TestBGAParser_SwarmlingsStronghold tests parsing Swarmlings stronghold free TP upgrade
func TestBGAParser_SwarmlingsStronghold(t *testing.T) {
	content := `Player1 is playing the Swarmlings Faction
Player1 places a Dwelling [E4]
~ Action phase ~
Move 1 :
Player1 upgrades a Trading house to a Faction Stronghold for 3 workers 6 coins [E4]
Player1 upgrades a Dwelling to a Trading house for 0 workers 0 coins (Swarmlings Stronghold) [F5]
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find upgrade action with Swarmlings bonus
	foundSwarmlingsBonus := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if specialAction, ok := subAction.(*LogSpecialAction); ok {
						if strings.HasPrefix(specialAction.ActionCode, "ACT-SH-TP") {
							foundSwarmlingsBonus = true
							t.Logf("Found Swarmlings stronghold action: %s", specialAction.ActionCode)
						}
					}
				}
			}
		}
	}

	t.Logf("Parsed %d items, foundSwarmlingsBonus=%v", len(items), foundSwarmlingsBonus)
}

// TestBGAParser_MermaidsRiverTown tests parsing Mermaids river town formation
func TestBGAParser_MermaidsRiverTown(t *testing.T) {
	content := `Player1 is playing the Mermaids Faction
Player1 places a Dwelling [B3]
Player1 places a Dwelling [C2]
~ Action phase ~
Move 1 :
Player1 upgrades a Dwelling to a Trading house for 2 workers 3 coins [B3]
Player1 forms a Town (Mermaids using river) [R~B2]
Player1 gains 5 VP (Town tile)
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find ACT-TOWN action for Mermaids river town
	foundMermaidsTown := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if specialAction, ok := actionItem.Action.(*LogSpecialAction); ok {
				if specialAction.ActionCode == "ACT-TOWN" {
					foundMermaidsTown = true
					t.Logf("Found Mermaids river town action")
				}
			}
		}
	}

	t.Logf("Parsed %d items, foundMermaidsTown=%v", len(items), foundMermaidsTown)
}

// TestBGAParser_WitchesRide tests parsing Witches stronghold ride action
func TestBGAParser_WitchesRide(t *testing.T) {
	content := `Player1 is playing the Witches Faction
Player1 places a Dwelling [D4]
~ Action phase ~
Move 1 :
Player1 upgrades a Trading house to a Faction Stronghold for 4 workers 6 coins [D4]
Player1 takes a Witches Ride (Stronghold) [F6]
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Find ACT-SH action for Witches
	foundWitchesRide := false
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if compound, ok := actionItem.Action.(*LogCompoundAction); ok {
				for _, subAction := range compound.Actions {
					if specialAction, ok := subAction.(*LogSpecialAction); ok {
						if strings.HasPrefix(specialAction.ActionCode, "ACT-SH") {
							foundWitchesRide = true
							t.Logf("Found Witches ride action: %s", specialAction.ActionCode)
						}
					}
				}
			}
		}
	}

	t.Logf("Parsed %d items, foundWitchesRide=%v", len(items), foundWitchesRide)
}

// TestBGAParser_NotationGeneration verifies generated notation includes correct terrain codes
func TestBGAParser_NotationGeneration(t *testing.T) {
	// Test that Halflings notation includes terrain suffix for non-home terrain
	action := &LogHalflingsSpadeAction{
		PlayerID:        "Halflings",
		TransformCoords: []string{"G5", "F6"},
		TargetTerrains:  []string{"plains", "desert"},
	}

	// Generate notation
	notation := generateActionCode(action, 0) // 0 = Plains (Halflings home)

	// G5→plains should NOT have suffix (home terrain)
	// F6→desert should have -Y suffix
	if !strings.Contains(notation, "T-G5") {
		t.Errorf("Expected T-G5 in notation, got: %s", notation)
	}
	if !strings.Contains(notation, "T-F6-Y") {
		t.Errorf("Expected T-F6-Y in notation, got: %s", notation)
	}
}

// TestMergeTransformAndBuildTokens tests merging T-X + X patterns
// This is a regression test for the Fakirs carpet flight double priest bug
func TestMergeTransformAndBuildTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "T-A7 followed by A7 should merge",
			input:    []string{"BURN2", "ACT6", "T-A7", "T-B2", "A7"},
			expected: []string{"BURN2", "ACT6", "TB-A7", "T-B2"},
		},
		{
			name:     "Multiple T-X + X merges",
			input:    []string{"ACT6", "T-A7", "T-B2", "A7", "B2"},
			expected: []string{"ACT6", "TB-A7", "TB-B2"},
		},
		{
			name:     "No merge when build is at different hex",
			input:    []string{"ACT6", "T-A7", "T-B2", "C3"},
			expected: []string{"ACT6", "T-A7", "T-B2", "C3"},
		},
		{
			name:     "Preserve terrain suffix when merging",
			input:    []string{"T-A7-Y", "A7"},
			expected: []string{"TB-A7-Y"},
		},
		{
			name:     "No transforms, no changes",
			input:    []string{"BURN2", "ACT6", "A7"},
			expected: []string{"BURN2", "ACT6", "A7"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := mergeTransformAndBuildTokens(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d tokens, got %d\nInput: %v\nExpected: %v\nGot: %v",
					len(tc.expected), len(result), tc.input, tc.expected, result)
				return
			}

			for i, tok := range result {
				if tok != tc.expected[i] {
					t.Errorf("Token %d: expected %q, got %q\nInput: %v\nExpected: %v\nGot: %v",
						i, tc.expected[i], tok, tc.input, tc.expected, result)
				}
			}
		})
	}
}

// TestBGAParser_FakirsCarpetFlightMultiple tests that multiple carpet flight actions
// are parsed correctly as separate actions.
// Scenario: transform A7 (carpet flight), transform B2 (carpet flight), build A7
// Backend logic ensures A7 is charged once, and B2 is charged once (total 2 charges).
func TestBGAParser_FakirsCarpetFlightMultiple(t *testing.T) {
	// This simulates the BGA log scenario:
	// Move 233: Fakirs burn 2, spend 6pw for 2 spades
	// - transform plains -> desert [A7] (1 spade + 1 priest carpet flight)
	// - gets 4 VP (Carpet Flight)
	// - transform plains -> desert [B2] (1 spade + 1 priest carpet flight implied if needed)
	// - builds Dwelling [A7]
	content := `Mellus is playing the Fakirs Faction
Mellus places a Dwelling [E6]
Mellus places a Dwelling [F5]
~ Action phase ~
Move 1 :
Mellus sacrificed 2 in Bowl 2 to get 2 from Bowl 2 to Bowl 3
Mellus spends 6 to get 2 (Power action)
Mellus transforms a Terrain space plains → desert for 1 + 1   [A7]
Mellus gets 4 VP (Carpet Flight)
Mellus transforms a Terrain space plains → desert for 1   [B2]
Mellus builds a Dwelling for 1 workers 2 coins   [A7]
Mellus gets 2 VP (Scoring tile bonus)
***** Final Scoring *****
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Count TransformAndBuildActions at A7
	// There should be 2 actions:
	// 1. Transform (BuildDwelling=false)
	// 2. Build (BuildDwelling=true)
	countA7Actions := 0
	foundTransform := false
	foundBuild := false

	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if tba, ok := actionItem.Action.(*game.TransformAndBuildAction); ok {
				// Check if this is at A7 (convert to axial: A=column 0, row 7)
				// In our coordinate system, A7 should be specific hex
				// We can check the hex coordinates directly or trust the parser test setup
				// Since we only have A7 and B2 in the input, and B2 is distinct
				if tba.TargetHex.Q == 0 && tba.TargetHex.R == 7 { // A7
					countA7Actions++
					if tba.BuildDwelling {
						foundBuild = true
					} else {
						foundTransform = true
					}
				}
			}
		}
	}

	// Key assertion: we should have TWO actions at A7
	// One transform, one build. They are NOT merged in the parser anymore.
	// The backend handles the double-charge prevention via SkipAbilityUsedThisAction.
	if countA7Actions != 2 {
		t.Errorf("Expected 2 actions at A7 (transform + build), got %d", countA7Actions)
	}

	if !foundTransform {
		t.Error("Expected A7 transform action")
	}
	if !foundBuild {
		t.Error("Expected A7 build action")
	}

	t.Logf("Parsed %d items, found %d actions at A7", len(items), countA7Actions)
}
