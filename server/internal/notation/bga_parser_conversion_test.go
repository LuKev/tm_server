package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestBGAParser_ConversionSimplification(t *testing.T) {
	// Example log line from user report: "Player1 converts 3 Power 2 Workers to 3 Workers 2 Coins"
	// This should result in: Cost: 3 Power, Reward: 1 Worker, 2 Coins
	// (2 Workers subtracted from both sides)

	logContent := `
Game board: Base
Player1 is playing the Engineers Faction
Player2 is playing the Nomads Faction
Every player has chosen a Faction
~ Action phase ~
Move 1 :
Player1 does some Conversions (spent: 3 Power 2 Workers ; collects: 3 Workers 2 Coins)
`

	parser := NewBGAParser(logContent)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the conversion action
	var conversionAction *LogConversionAction
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			if conv, ok := actionItem.Action.(*LogConversionAction); ok {
				conversionAction = conv
				break
			}
		}
	}

	if conversionAction == nil {
		t.Fatal("Should find a conversion action")
	}

	t.Logf("Conversion Action: %+v", conversionAction)
	t.Logf("Cost: %+v", conversionAction.Cost)
	t.Logf("Reward: %+v", conversionAction.Reward)

	// Check Cost
	if val := conversionAction.Cost[models.ResourcePower]; val != 3 {
		t.Errorf("Cost should have 3 Power, got %d", val)
	}
	if val := conversionAction.Cost[models.ResourceWorker]; val != 0 {
		t.Errorf("Cost should have 0 Workers (subtracted), got %d", val)
	}

	// Check Reward
	if val := conversionAction.Reward[models.ResourceWorker]; val != 1 {
		t.Errorf("Reward should have 1 Worker (3 - 2), got %d", val)
	}
	if val := conversionAction.Reward[models.ResourceCoin]; val != 2 {
		t.Errorf("Reward should have 2 Coins, got %d", val)
	}
}
