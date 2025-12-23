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

	if missing.GlobalBonusCards {
		t.Error("GlobalBonusCards should be false")
	}
	if missing.GlobalScoringTiles {
		t.Error("GlobalScoringTiles should be false")
	}

	// Check Round 1
	if missing.BonusCardSelections[1] == nil {
		t.Fatal("Round 1 bonus card selections should be missing")
	}
	if !missing.BonusCardSelections[1]["player1"] {
		t.Error("player1 should be missing bonus card in Round 1")
	}
	if !missing.BonusCardSelections[1]["player2"] {
		t.Error("player2 should be missing bonus card in Round 1")
	}

	// Check Round 6 (should be nil or empty)
	if missing.BonusCardSelections[6] != nil {
		t.Error("Round 6 bonus card selections should not be tracked")
	}
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
