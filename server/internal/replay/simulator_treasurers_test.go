package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/notation"
)

func TestStepForward_QueuesTreasurersDepositAfterActionResourceGain(t *testing.T) {
	gs := game.NewGameState()
	gs.TurnOrder = []string{"p1"}
	gs.Phase = game.PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = game.NewPowerSystem(0, 8, 0)

	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewPowerAction("p1", game.PowerActionWorkers)},
		notation.ActionItem{Action: game.NewSelectTreasurersDepositAction("p1", 0, 2, 0)},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("first StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers deposit after action resource gain")
	}
	if got := gs.PendingTreasurersDeposit.AvailableWorkers; got != 2 {
		t.Fatalf("available workers to bank = %d, want 2", got)
	}

	if err := sim.StepForward(); err != nil {
		t.Fatalf("second StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit != nil {
		t.Fatalf("expected pending Treasurers deposit to be cleared")
	}
	if got := player.TreasuryWorkers; got != 2 {
		t.Fatalf("treasury workers after replay deposit = %d, want 2", got)
	}
}

func TestFinalScoringValidation_RejectsBGADragonlordsResourceMismatch(t *testing.T) {
	gs := game.NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true, "__bga__": true}
	gs.Phase = game.PhaseEnd
	if err := gs.AddPlayer("Dragonlords", factions.NewDragonlords()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("Dragonlords")
	player.Resources.Coins = 3

	sim := NewGameSimulator(gs, nil)
	err := sim.validateFinalScoring(notation.FinalScoringValidationItem{
		Scores: map[string]*notation.FinalScoringExpectation{
			"Dragonlords": {
				PlayerID:           "Dragonlords",
				ResourceVP:         0,
				TotalResourceValue: 2,
				HasResourceScore:   true,
			},
		},
	})
	if err == nil {
		t.Fatal("expected Dragonlords resource mismatch to fail validation")
	}
}

func TestStepForward_HandlesRoundOneTreasurersIncomeDepositBeforeActionPhaseMarker(t *testing.T) {
	gs := game.NewGameState()
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	actions := []notation.LogItem{
		notation.ActionItem{Action: &notation.LogPostIncomeAction{
			Action: game.NewSelectTreasurersDepositAction("p1", 0, 4, 0),
		}},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v", err)
	}
	if got := gs.Round; got != 1 {
		t.Fatalf("round after initial income deposit = %d, want 1", got)
	}
	if got := gs.GetPlayer("p1").TreasuryWorkers; got != 4 {
		t.Fatalf("treasury workers after initial income deposit = %d, want 4", got)
	}
}

func TestStepForward_QueuesTreasurersDepositAfterPassBonusCardCoins(t *testing.T) {
	gs := game.NewGameState()
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = game.PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer(p1) failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewHalflings()); err != nil {
		t.Fatalf("AddPlayer(p2) failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true

	gs.BonusCards.SetAvailableBonusCards([]game.BonusCardType{
		game.BonusCardStrongholdSanctuary,
		game.BonusCardSpade,
	})
	gs.BonusCards.PlayerCards["p1"] = game.BonusCardStrongholdSanctuary
	gs.BonusCards.PlayerHasCard["p1"] = true
	gs.BonusCards.Available[game.BonusCardSpade] = 1

	card := game.BonusCardSpade
	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewPassAction("p1", &card)},
		notation.ActionItem{Action: game.NewSelectTreasurersDepositAction("p1", 1, 0, 0)},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("first StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers deposit after pass bonus-card coins")
	}
	if got := gs.PendingTreasurersDeposit.AvailableCoins; got != 1 {
		t.Fatalf("available coins to bank after pass = %d, want 1", got)
	}

	if err := sim.StepForward(); err != nil {
		t.Fatalf("second StepForward() error = %v", err)
	}
	if got := player.TreasuryCoins; got != 1 {
		t.Fatalf("treasury coins after pass deposit = %d, want 1", got)
	}
}

func TestStepForward_DelaysCleanupUntilFinalPassTreasurersDepositResolves(t *testing.T) {
	gs := game.NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true}
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = game.PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer(p1) failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewHalflings()); err != nil {
		t.Fatalf("AddPlayer(p2) failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	gs.GetPlayer("p2").HasPassed = true

	gs.BonusCards.SetAvailableBonusCards([]game.BonusCardType{
		game.BonusCardStrongholdSanctuary,
		game.BonusCardSpade,
	})
	gs.BonusCards.PlayerCards["p1"] = game.BonusCardStrongholdSanctuary
	gs.BonusCards.PlayerHasCard["p1"] = true
	gs.BonusCards.Available[game.BonusCardSpade] = 1

	card := game.BonusCardSpade
	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewPassAction("p1", &card)},
		notation.ActionItem{Action: game.NewSelectTreasurersDepositAction("p1", 1, 0, 0)},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("first StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers deposit after final pass")
	}
	if got := gs.Phase; got != game.PhaseAction {
		t.Fatalf("phase after final pass = %v, want action until deposit resolves", got)
	}

	if err := sim.StepForward(); err != nil {
		t.Fatalf("second StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit != nil {
		t.Fatalf("expected final-pass Treasurers deposit to be cleared")
	}
	if got := player.TreasuryCoins; got != 1 {
		t.Fatalf("treasury coins after final-pass deposit = %d, want 1", got)
	}
}

func TestStepForward_AutoResolvesImplicitZeroTreasurersDepositBeforeNextAction(t *testing.T) {
	gs := game.NewGameState()
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 1
	gs.Phase = game.PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer(p1) failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewHalflings()); err != nil {
		t.Fatalf("AddPlayer(p2) failed: %v", err)
	}

	game.QueueTreasurersDeposit(gs, "p1", 1, 2, 0, "income")

	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewPassAction("p2", nil)},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v", err)
	}
	if gs.PendingTreasurersDeposit != nil {
		t.Fatalf("expected implicit zero Treasurers deposit to be cleared before next action")
	}
}

func TestStepForward_ReconcilesHiddenRoundSixTreasurersIncomeDepositFromFinalScoring(t *testing.T) {
	gs := game.NewGameState()
	gs.Round = 6
	gs.Phase = game.PhaseEnd
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 33
	player.Resources.Workers = 6
	player.Resources.Priests = 0
	player.Resources.Power = game.NewPowerSystem(1, 3, 0)

	actions := []notation.LogItem{
		notation.FinalScoringValidationItem{
			Scores: map[string]*notation.FinalScoringExpectation{
				"p1": {
					PlayerID:                          "p1",
					ResourceVP:                        11,
					TotalResourceValue:                35,
					HasResourceScore:                  true,
					FinalCoinsBeforeResourceScoring:   33,
					FinalWorkersBeforeResourceScoring: 1,
					FinalPriestsBeforeResourceScoring: 0,
					FinalPowerCoinsConverted:          1,
					HasExactResourceBreakdown:         true,
				},
			},
		},
	}
	sim := NewGameSimulator(gs, actions)
	sim.lastTreasurersIncomeOffers["p1"] = &game.PendingTreasurersDeposit{
		PlayerID:         "p1",
		AvailableCoins:   47,
		AvailableWorkers: 12,
		AvailablePriests: 3,
		Reason:           "income",
	}

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v", err)
	}
	if got := player.Resources.Workers; got != 1 {
		t.Fatalf("workers after final-scoring reconciliation = %d, want 1", got)
	}
	if got := player.TreasuryWorkers; got != 5 {
		t.Fatalf("treasury workers after final-scoring reconciliation = %d, want 5", got)
	}
}

func TestStepForward_AutoResolvesEarlierNonIncomeTreasurersDepositBeforePostIncomeDeposit(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	game.QueueTreasurersDeposit(gs, "p1", 4, 0, 0, "cult_reward")
	game.QueueTreasurersDeposit(gs, "p1", 1, 2, 0, "income")

	actions := []notation.LogItem{
		notation.ActionItem{Action: &notation.LogPostIncomeAction{
			Action: game.NewSelectTreasurersDepositAction("p1", 1, 2, 0),
		}},
	}
	sim := NewGameSimulator(gs, actions)

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 5
	player.Resources.Workers = 2

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v", err)
	}
	if got := player.TreasuryCoins; got != 1 {
		t.Fatalf("treasury coins after post-income deposit = %d, want 1", got)
	}
	if got := player.TreasuryWorkers; got != 2 {
		t.Fatalf("treasury workers after post-income deposit = %d, want 2", got)
	}
}
