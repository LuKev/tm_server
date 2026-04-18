package replay

import (
	"errors"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/notation"
)

func TestGameSimulator_ReportsMissingInitialBonusCardsBeforeRoundOneAction(t *testing.T) {
	items := []notation.LogItem{
		notation.GameSettingsItem{
			Settings: map[string]string{
				"Player:alice":         "Cultists",
				"Player:bob":           "Engineers",
				"BonusCards":           "BON-SPD,BON-4C,BON-6C,BON-SHIP,BON-WP",
				"ScoringTiles":         "SCORE1,SCORE2,SCORE3,SCORE4,SCORE5,SCORE6",
				"StartingVP:Cultists":  "20",
				"StartingVP:Engineers": "20",
			},
		},
		notation.RoundStartItem{
			Round:     1,
			TurnOrder: []string{"Cultists", "Engineers"},
		},
		notation.ActionItem{
			Action: game.NewAdvanceShippingAction("Cultists"),
		},
	}

	initialState := createInitialState(items)
	sim := NewGameSimulator(initialState, items)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("settings step failed: %v", err)
	}
	if err := sim.StepForward(); err != nil {
		t.Fatalf("round start step failed: %v", err)
	}

	err := sim.StepForward()
	if err == nil {
		t.Fatal("expected missing initial bonus card error")
	}

	var missingErr *game.MissingInfoError
	if !errors.As(err, &missingErr) {
		t.Fatalf("error type = %T, want *game.MissingInfoError", err)
	}
	if missingErr.Type != "initial_bonus_card" {
		t.Fatalf("MissingInfoError.Type = %q, want %q", missingErr.Type, "initial_bonus_card")
	}
	if len(missingErr.Players) != 2 {
		t.Fatalf("MissingInfoError.Players len = %d, want 2 (%v)", len(missingErr.Players), missingErr.Players)
	}
}

func TestGameSimulator_ValidatesFinalScoringBlock(t *testing.T) {
	gs := game.NewGameState()
	gs.Phase = game.PhaseEnd
	gs.FinalScoring = map[string]*game.PlayerFinalScore{
		"Witches": {
			PlayerID:           "Witches",
			CultVP:             8,
			AreaVP:             18,
			ResourceVP:         3,
			LargestAreaSize:    15,
			TotalResourceValue: 10,
		},
	}

	items := []notation.LogItem{
		notation.FinalScoringValidationItem{
			Scores: map[string]*notation.FinalScoringExpectation{
				"Witches": {
					PlayerID:           "Witches",
					CultVP:             8,
					AreaVP:             18,
					ResourceVP:         3,
					LargestAreaSize:    15,
					TotalResourceValue: 10,
					HasAreaScore:       true,
					HasResourceScore:   true,
				},
			},
		},
	}

	sim := NewGameSimulator(gs, items)
	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() failed: %v", err)
	}
}

func TestGameSimulator_FinalScoringValidationMismatchFails(t *testing.T) {
	gs := game.NewGameState()
	gs.Phase = game.PhaseEnd
	gs.FinalScoring = map[string]*game.PlayerFinalScore{
		"Witches": {
			PlayerID: "Witches",
			CultVP:   8,
		},
	}

	items := []notation.LogItem{
		notation.FinalScoringValidationItem{
			Scores: map[string]*notation.FinalScoringExpectation{
				"Witches": {
					PlayerID: "Witches",
					CultVP:   10,
				},
			},
		},
	}

	sim := NewGameSimulator(gs, items)
	err := sim.StepForward()
	if err == nil {
		t.Fatal("expected final scoring validation error")
	}
	if !strings.Contains(err.Error(), "cult VP mismatch") {
		t.Fatalf("error = %v, want cult VP mismatch", err)
	}
}

func TestGameSimulator_FinalScoringValidationTriggersRoundSixCleanup(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("Witches", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer() failed: %v", err)
	}
	gs.Round = 6
	gs.Phase = game.PhaseAction
	gs.Players["Witches"].HasPassed = true

	expected := gs.CalculateFinalScoring()["Witches"]
	items := []notation.LogItem{
		notation.FinalScoringValidationItem{
			Scores: map[string]*notation.FinalScoringExpectation{
				"Witches": {
					PlayerID:           "Witches",
					CultVP:             expected.CultVP,
					AreaVP:             expected.AreaVP,
					ResourceVP:         expected.ResourceVP,
					LargestAreaSize:    expected.LargestAreaSize,
					TotalResourceValue: expected.TotalResourceValue,
					HasAreaScore:       true,
					HasResourceScore:   true,
				},
			},
		},
	}

	sim := NewGameSimulator(gs, items)
	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() failed: %v", err)
	}
	if sim.GetState().Phase != game.PhaseEnd {
		t.Fatalf("phase after final scoring validation = %v, want %v", sim.GetState().Phase, game.PhaseEnd)
	}
	if sim.GetState().FinalScoring == nil {
		t.Fatal("FinalScoring should be populated after round 6 cleanup")
	}
}
