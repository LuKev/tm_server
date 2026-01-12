package notation

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

func TestBGAParser_CultistsAbilityMerge(t *testing.T) {
	// Example from user:
	// WoodMarco transforms a Terrain space swamp → plains for 1 spade(s) (Bonus card action) [E5]
	// WoodMarco builds a Dwelling for 1 workers 2 coins [E5]
	// WoodMarco declines doing Conversions
	// ~ You have enabled automatic acceptance of Power via Structures up to 4 power for 3 VP ~
	// Mellus gets 1 power via Structures [E5]
	// Arivor gets 1 power via Structures [E5]
	// ~ You have enabled automatic acceptance of Power via Structures up to 3 power for 2 VP ~
	// WoodMarco gains 1 on the Cult of Fire track (Cultists ability)
	// WoodMarco declines doing Conversions

	// Expected: The build action should include the cult advancement.
	// ACTS-E5.E5.CULT-F

	logContent := `
Game board: Base
Player1 is playing the Cultists Faction
Player2 is playing the Nomads Faction
Player3 is playing the Mermaids Faction
Every player has chosen a Faction
~ Action phase ~
Move 1 :
Player1 transforms a Terrain space swamp → plains for 1 spade(s) (Bonus card action) [E5]
Player1 builds a Dwelling for 1 workers 2 coins [E5]
Player1 declines doing Conversions
Player3 gets 1 power via Structures [E5]
Player2 gets 1 power via Structures [E5]
Player1 gains 1 on the Cult of Fire track (Cultists ability)
Player1 declines doing Conversions
`

	// Scenario 2: Normal Build
	// Player1 builds a Dwelling for 1 workers 2 coins [E6]
	// Player3 gets 1 power via Structures [E6]
	// Player1 gains 1 on the Cult of Water track (Cultists ability)

	// Scenario 3: Upgrade
	// Player1 upgrades a Dwelling to a Trading house for 2 workers 3 coins [E6]
	// Player3 gets 2 power via Structures [E6]
	// Player1 gains 1 on the Cult of Earth track (Cultists ability)

	logContent += `
Player1 builds a Dwelling for 1 workers 2 coins [E6]
Player3 gets 1 power via Structures [E6]
Player1 gains 1 on the Cult of Water track (Cultists ability)
Player1 upgrades a Dwelling to a Trading house for 2 workers 3 coins [E6]
Player3 gets 2 power via Structures [E6]
Player1 gains 1 on the Cult of Earth track (Cultists ability)
`

	parser := NewBGAParser(logContent)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect:
	// 1. Round Start
	// 2. Player1 Action: ACTS-E5.e5 + CULT-F (Compound)
	// 3. Player3 Leech
	// 4. Player2 Leech
	// 5. Player1 Action: Build Dwelling + Cult Water
	// 6. Player3 Leech
	// 7. Player1 Action: Upgrade + Cult Earth
	// 8. Player3 Leech

	var bonusCardAction *LogCompoundAction
	var buildAction *LogCompoundAction
	var upgradeAction *LogCompoundAction

	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if ca, ok := actionItem.Action.(*LogCompoundAction); ok {
				if len(ca.Actions) > 0 {
					firstAction := ca.Actions[0]
					if sa, ok := firstAction.(*LogSpecialAction); ok {
						if sa.PlayerID == "Cultists" && (sa.ActionCode == "ACTS-E5.e5" || sa.ActionCode == "ACTS-E5") {
							bonusCardAction = ca
						}
					} else if _, ok := firstAction.(*game.TransformAndBuildAction); ok {
						buildAction = ca
					} else if _, ok := firstAction.(*game.UpgradeBuildingAction); ok {
						upgradeAction = ca
					}
				}
			}
		}
	}

	if bonusCardAction == nil {
		t.Fatal("Should find compound bonus card action")
	}
	if len(bonusCardAction.Actions) != 2 {
		t.Errorf("Bonus card action should have 2 sub-actions, got %d", len(bonusCardAction.Actions))
	}
	if sa, ok := bonusCardAction.Actions[0].(*LogSpecialAction); !ok {
		t.Error("First action in bonus card compound should be LogSpecialAction")
	} else {
		if sa.ActionCode != "ACTS-E5.e5" {
			t.Errorf("Action code should be ACTS-E5.e5, got %s", sa.ActionCode)
		}
	}
	if _, ok := bonusCardAction.Actions[1].(*LogCultistAdvanceAction); !ok {
		t.Error("Second action in bonus card compound should be LogCultistAdvanceAction")
	} else {
		cultAction := bonusCardAction.Actions[1].(*LogCultistAdvanceAction)
		if cultAction.Track != game.CultFire {
			t.Errorf("Bonus card cult action should be Fire, got %v", cultAction.Track)
		}
	}

	if buildAction == nil {
		t.Error("Should find compound build action")
	} else {
		if len(buildAction.Actions) != 2 {
			t.Errorf("Build action should have 2 sub-actions, got %d", len(buildAction.Actions))
		}
		if _, ok := buildAction.Actions[1].(*LogCultistAdvanceAction); !ok {
			t.Error("Second action in build compound should be LogCultistAdvanceAction")
		} else {
			cultAction := buildAction.Actions[1].(*LogCultistAdvanceAction)
			if cultAction.Track != game.CultWater {
				t.Errorf("Build cult action should be Water, got %v", cultAction.Track)
			}
		}
	}

	if upgradeAction == nil {
		t.Error("Should find compound upgrade action")
	} else {
		if len(upgradeAction.Actions) != 2 {
			t.Errorf("Upgrade action should have 2 sub-actions, got %d", len(upgradeAction.Actions))
		}
		if _, ok := upgradeAction.Actions[1].(*LogCultistAdvanceAction); !ok {
			t.Error("Second action in upgrade compound should be LogCultistAdvanceAction")
		} else {
			cultAction := upgradeAction.Actions[1].(*LogCultistAdvanceAction)
			if cultAction.Track != game.CultEarth {
				t.Errorf("Upgrade cult action should be Earth, got %v", cultAction.Track)
			}
		}
	}

	// Verify string generation
	conciseLog, _ := GenerateConciseLog(items)

	// Check for expected strings in the log
	foundBonusCard := false
	foundBuild := false
	foundUpgrade := false

	for _, line := range conciseLog {
		if strings.Contains(line, "ACTS-E5.e5.CULT-F") {
			foundBonusCard = true
		}
		if strings.Contains(line, "E6.CULT-W") {
			foundBuild = true
		}
		if strings.Contains(line, "UP-TH-E6.CULT-E") {
			foundUpgrade = true
		}
	}

	if !foundBonusCard {
		t.Error("Did not find expected string 'ACTS-E5.e5.CULT-F' in concise log")
	}
	if !foundBuild {
		t.Error("Did not find expected string 'E6.CULT-W' in concise log")
	}
	if !foundUpgrade {
		t.Error("Did not find expected string 'UP-TH-E6.CULT-E' in concise log")
	}
	// Scenario 4: Upgrade + Cultist + Favor
	// WoodMarco upgrades a Trading house to a Temple for 2 workers 5 coins [E6]
	// ... leeches ...
	// WoodMarco gains 1 on the Cult of Fire track (Cultists ability)
	// WoodMarco takes a Favor tile
	// WoodMarco gains 1 on the Cult of Earth track (Favor tile)

	logContent += `
Player1 upgrades a Trading house to a Temple for 2 workers 5 coins [E6]
Player1 gets 4 VP (Scoring tile bonus)
Player1 declines doing Conversions
Player2 pays 1 VP and gets 2 power via Structures [E6]
Player3 pays 1 VP and gets 2 power via Structures [E6]
Player1 gains 1 on the Cult of Fire track (Cultists ability)
Player1 takes a Favor tile
Player1 gains 1 on the Cult of Earth track (Favor tile)
Player1 declines doing Conversions
`

	parser = NewBGAParser(logContent)
	items, err = parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Scenario 4
	foundScenario4 := false
	conciseLog, _ = GenerateConciseLog(items)
	for _, line := range conciseLog {
		if strings.Contains(line, "UP-TE-E6.CULT-F.FAV-E1") {
			foundScenario4 = true
		}
	}
	if !foundScenario4 {
		t.Error("Did not find expected string 'UP-TE-E6.CULT-F.FAV-E1' in concise log")
		for _, line := range conciseLog {
			t.Log(line)
		}
	}
}
