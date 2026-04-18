package notation

import "testing"

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
			if _, ok := actionItem.Action.(*LogDeclineLeechAction); ok {
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

func TestBGAParser_DeclineLeechUsesStructureCoordinateSource(t *testing.T) {
	content := `
Game board: Base
Nafghar is playing the Prospectors Faction
LANMEEE is playing the Archivists Faction
Xevoc is playing the Conspirators Faction
Every player has chosen a Faction
Nafghar places a Dwelling [A1]
LANMEEE places a Dwelling [B1]
Xevoc places a Dwelling [C1]
~ Action phase ~
Move 1 :
LANMEEE upgrades a Dwelling to a Trading house for 2 workers 3 coins [F3]
Xevoc gets 1 power via Structures [F3]
Nafghar pays 1 VP and gets 2 power via Structures [F3]
Xevoc upgrades a Dwelling to a Trading house for 2 workers 3 coins [F4]
LANMEEE pays 2 VP and gets 3 power via Structures [F4]
Nafghar declines getting Power via Structures [F4]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		decline, ok := actionItem.Action.(*LogDeclineLeechAction)
		if !ok {
			continue
		}
		if decline.PlayerID != "Prospectors" {
			continue
		}
		if decline.FromPlayerID != "Conspirators" {
			t.Fatalf("Prospectors decline source = %q, want Conspirators", decline.FromPlayerID)
		}
		return
	}

	t.Fatal("did not find Prospectors decline leech action")
}

func TestBGAParser_ZeroCapLeechResolvesLatestStructureSource(t *testing.T) {
	content := `
Game board: Base
Nafghar is playing the Prospectors Faction
Xevoc is playing the Conspirators Faction
Every player has chosen a Faction
Nafghar places a Dwelling [A1]
Xevoc places a Dwelling [B1]
~ Action phase ~
Move 1 :
Xevoc upgrades a Dwelling to a Trading house for 2 workers 3 coins [F4]
Nafghar Power gain via Structures is capped from 3 power to 0 power
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		decline, ok := actionItem.Action.(*LogDeclineLeechAction)
		if !ok {
			continue
		}
		if decline.PlayerID != "Prospectors" {
			continue
		}
		if decline.FromPlayerID != "Conspirators" {
			t.Fatalf("Prospectors zero-cap source = %q, want Conspirators", decline.FromPlayerID)
		}
		if decline.FromHex == nil || HexToShortString(*decline.FromHex) != "F4" {
			t.Fatalf("Prospectors zero-cap source hex = %+v, want F4", decline.FromHex)
		}
		return
	}

	t.Fatal("did not find Prospectors zero-cap decline action")
}

func TestBGAParser_CappedLeechMarksActualAmountExplicit(t *testing.T) {
	content := `
Game board: Base
Nafghar is playing the Prospectors Faction
Xevoc is playing the Conspirators Faction
Every player has chosen a Faction
Nafghar places a Dwelling [A1]
Xevoc places a Dwelling [B1]
~ Action phase ~
Move 1 :
Xevoc upgrades a Trading house to a Temple for 2 workers 5 coins [F6]
Nafghar Power gain via Structures is capped from 6 power to 2 power
Nafghar pays 1 VP and gets 2 power via Structures [F6]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		leech, ok := actionItem.Action.(*LogAcceptLeechAction)
		if !ok {
			continue
		}
		if leech.PlayerID != "Prospectors" {
			continue
		}
		if !leech.Explicit {
			t.Fatalf("Prospectors capped leech Explicit = false, want true")
		}
		if leech.PowerAmount != 2 || leech.VPCost != 1 {
			t.Fatalf("Prospectors capped leech = %+v, want power=2 vp=1", leech)
		}
		return
	}

	t.Fatal("did not find Prospectors capped leech action")
}
