package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Test the specific Round 6 compound actions that are causing validation errors
func TestCompoundAction_Round6Patterns(t *testing.T) {
	tests := []struct {
		name   string
		action string
	}{
		{
			name:   "Line 364 - send priest then convert",
			action: "send p to FIRE. convert 1PW to 1C",
		},
		{
			name:   "Line 369 - convert and advance shipping",
			action: "convert 2PW to 2C. advance ship",
		},
		{
			name:   "Line 383 - burn, multiple converts, and advance",
			action: "burn 1. convert 1PW to 1C. convert 3W to 3C. advance ship",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := game.NewGameState()
			gs.AddPlayer("engineers", &factions.Engineers{})

			entry := &LogEntry{
				Faction: models.FactionEngineers,
				Action:  tt.action,
			}

			compound, err := ParseCompoundAction(entry.Action, entry, gs)
			if err != nil {
				t.Fatalf("ParseCompoundAction() error = %v", err)
			}

			if len(compound.Components) == 0 {
				t.Errorf("expected components, got 0")
			}

			t.Logf("Action: %s", tt.action)
			t.Logf("Parsed into %d components:", len(compound.Components))
			for i, comp := range compound.Components {
				t.Logf("  %d. %s", i+1, comp.String())
			}
		})
	}
}
