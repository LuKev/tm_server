package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/notation"
)

func TestDetectMissingInfo(t *testing.T) {
	// Create mock items
	items := []notation.LogItem{
		notation.GameSettingsItem{
			Settings: map[string]string{
				"BonusCards":   "BON1,BON2",
				"ScoringTiles": "SCORE1,SCORE2",
			},
		},
		notation.RoundStartItem{Round: 1},
		notation.ActionItem{
			Action: game.NewPassAction("player1", nil), // Missing bonus card
		},
		notation.ActionItem{
			Action: game.NewPassAction("player2", nil), // Missing bonus card
		},
		notation.RoundStartItem{Round: 6},
		notation.ActionItem{
			Action: game.NewPassAction("player1", nil), // Round 6 pass (should be ignored)
		},
	}

	missing := detectMissingInfo(items)

	// Since we moved bonus card checks to runtime, detectMissingInfo should return nil
	// if global settings are present.
	if missing != nil {
		t.Error("detectMissingInfo should return nil when global settings are present")
	}

	// The rest of the test checking for specific missing bonus cards is now obsolete
	// for this function, as it only checks global info.
}

func TestDetectMissingGlobalInfo(t *testing.T) {
	// Create mock items with missing settings
	items := []notation.LogItem{
		notation.GameSettingsItem{
			Settings: map[string]string{},
		},
	}

	missing := detectMissingInfo(items)

	if !missing.GlobalBonusCards {
		t.Error("GlobalBonusCards should be true")
	}
	if !missing.GlobalScoringTiles {
		t.Error("GlobalScoringTiles should be true")
	}
}
