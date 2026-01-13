package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

func TestBGAParser_DuplicateLeech(t *testing.T) {
	content := `
Game board: Base
Player1 is playing the Swarmlings Faction
Player2 is playing the Nomads Faction
Every player has chosen a Faction
Player1 places a Dwelling [A1]
Player2 places a Dwelling [B1]
~ The Factions auction is over ~
~ Action phase ~
Move 1 :
Player2 upgrades a Dwelling to a Trading house for 2 workers 3 coins [B1]
Player1 declines getting Power via Structures [A1]
Player1 declines getting Power via Structures [A1]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Count leech actions
	leechCount := 0
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if _, ok := actionItem.Action.(*game.DeclinePowerLeechAction); ok {
				leechCount++
			}
		}
	}

	if leechCount != 1 {
		for i, item := range items {
			t.Logf("Item %d: %T", i, item)
			if actionItem, ok := item.(ActionItem); ok {
				t.Logf("  Action: %T", actionItem.Action)
			}
		}
		t.Errorf("Expected 1 leech action, got %d", leechCount)
	} else {
		// Debug anyway to be sure
		for i, item := range items {
			t.Logf("Item %d: %T", i, item)
			if actionItem, ok := item.(ActionItem); ok {
				t.Logf("  Action: %T", actionItem.Action)
			}
		}
	}
}
