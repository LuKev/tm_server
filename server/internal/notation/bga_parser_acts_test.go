package notation

import (
	"strings"
	"testing"
)

func TestBGAParser_BonusCardSpadeTerrain(t *testing.T) {
	// Example from user:
	// WoodMarco transforms a Terrain space lakes → swamp for 1 spade(s) (Bonus card action) [G6]
	// WoodMarco declines doing Conversions

	// Expected: ACTS-G6-Bk (Swamp is Black/Bk)

	logContent := `
Game board: Base
Player1 is playing the Cultists Faction
Every player has chosen a Faction
~ Action phase ~
Player1 transforms a Terrain space lakes → swamp for 1 spade(s) (Bonus card action) [G6]
Player1 declines doing Conversions
`

	parser := NewBGAParser(logContent)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect:
	// 1. Round Start (implicit/skipped if not full log) or just the action
	// 2. Player1 Action: ACTS-G6-Bk

	var actsAction *LogSpecialAction

	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if sa, ok := actionItem.Action.(*LogSpecialAction); ok {
				if strings.HasPrefix(sa.ActionCode, "ACTS-") {
					actsAction = sa
					break
				}
			}
		}
	}

	if actsAction == nil {
		t.Fatal("Should find ACTS action")
	}

	// The user reported it was ACTS-G6, but wants ACTS-G6-Bk (or similar)
	// Let's check what we get
	if actsAction.ActionCode == "ACTS-G6" {
		t.Error("Action code is missing terrain suffix: ACTS-G6")
	} else if actsAction.ActionCode != "ACTS-G6-Bk" {
		t.Errorf("Expected ACTS-G6-Bk, got %s", actsAction.ActionCode)
	}
}
