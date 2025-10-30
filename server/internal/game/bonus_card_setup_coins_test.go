package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

// TestBonusCards_CoinsAfterSetup tests that leftover bonus cards accumulate
// 1 coin after the setup phase (Bug #32 regression test)
func TestBonusCards_CoinsAfterSetup(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewEngineers())
	gs.AddPlayer("player2", factions.NewDarklings())
	gs.AddPlayer("player3", factions.NewCultists())
	gs.AddPlayer("player4", factions.NewWitches())

	// Set up 7 bonus cards for a 4-player game (4 players + 3 = 7 cards)
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardSpade,              // BON1
		BonusCardPriest,             // BON2
		BonusCardWorkerPower,        // BON3
		BonusCardDwellingVP,         // BON5
		BonusCardTradingHouseVP,     // BON6
		BonusCardShipping,           // BON7
		BonusCardShippingVP,         // BON10
	})

	// During setup, players pass and take 4 cards (BON1, 5, 6, 10)
	// leaving BON2, BON3, BON7 with 0 coins
	gs.BonusCards.TakeBonusCard("player1", BonusCardSpade)             // BON1
	gs.BonusCards.TakeBonusCard("player2", BonusCardDwellingVP)        // BON5
	gs.BonusCards.TakeBonusCard("player3", BonusCardTradingHouseVP)    // BON6
	gs.BonusCards.TakeBonusCard("player4", BonusCardShippingVP)        // BON10

	// Verify leftover cards have 0 coins initially
	if gs.BonusCards.Available[BonusCardPriest] != 0 {
		t.Errorf("BON2 should have 0 coins after setup, got %d", gs.BonusCards.Available[BonusCardPriest])
	}
	if gs.BonusCards.Available[BonusCardWorkerPower] != 0 {
		t.Errorf("BON3 should have 0 coins after setup, got %d", gs.BonusCards.Available[BonusCardWorkerPower])
	}
	if gs.BonusCards.Available[BonusCardShipping] != 0 {
		t.Errorf("BON7 should have 0 coins after setup, got %d", gs.BonusCards.Available[BonusCardShipping])
	}

	// Simulate starting Round 1 (transition from PhaseSetup to PhaseIncome)
	gs.Phase = PhaseSetup
	gs.PassOrder = []string{"player1", "player2", "player3", "player4"}
	gs.StartNewRound()

	// Verify leftover cards now have 1 coin each
	if gs.BonusCards.Available[BonusCardPriest] != 1 {
		t.Errorf("BON2 should have 1 coin after setup phase, got %d", gs.BonusCards.Available[BonusCardPriest])
	}
	if gs.BonusCards.Available[BonusCardWorkerPower] != 1 {
		t.Errorf("BON3 should have 1 coin after setup phase, got %d", gs.BonusCards.Available[BonusCardWorkerPower])
	}
	if gs.BonusCards.Available[BonusCardShipping] != 1 {
		t.Errorf("BON7 should have 1 coin after setup phase, got %d", gs.BonusCards.Available[BonusCardShipping])
	}
}

// TestBonusCards_GetCoinsWhenPassingInRound1 tests that players receive
// the accumulated coins when they take a leftover bonus card in Round 1
// (Bug #32 regression test)
func TestBonusCards_GetCoinsWhenPassingInRound1(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewEngineers())
	gs.AddPlayer("player2", factions.NewDarklings())

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardPriest,        // BON2
		BonusCardWorkerPower,   // BON3
	})

	// During setup, player1 takes BON3, leaving BON2
	gs.BonusCards.TakeBonusCard("player1", BonusCardWorkerPower)

	// BON2 has 0 coins initially
	if gs.BonusCards.Available[BonusCardPriest] != 0 {
		t.Errorf("BON2 should have 0 coins initially, got %d", gs.BonusCards.Available[BonusCardPriest])
	}

	// Simulate starting Round 1
	gs.Phase = PhaseSetup
	gs.PassOrder = []string{"player1"}
	gs.StartNewRound()

	// BON2 should now have 1 coin
	if gs.BonusCards.Available[BonusCardPriest] != 1 {
		t.Errorf("BON2 should have 1 coin after setup, got %d", gs.BonusCards.Available[BonusCardPriest])
	}

	// In Round 1, player2 passes and takes BON2 (with 1 coin)
	player2 := gs.GetPlayer("player2")
	initialCoins := player2.Resources.Coins

	coins, err := gs.BonusCards.TakeBonusCard("player2", BonusCardPriest)
	if err != nil {
		t.Fatalf("failed to take bonus card: %v", err)
	}

	// Should receive 1 coin from the bonus card
	if coins != 1 {
		t.Errorf("expected to receive 1 coin from BON2, got %d", coins)
	}

	// Player should receive the coin
	player2.Resources.Coins += coins
	if player2.Resources.Coins != initialCoins+1 {
		t.Errorf("expected player to have %d coins (initial + 1), got %d", initialCoins+1, player2.Resources.Coins)
	}
}

// TestBonusCards_NoCoinsAfterNonSetupRounds tests that coins are not
// double-added when transitioning between non-setup rounds
func TestBonusCards_NoCoinsAfterNonSetupRounds(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewEngineers())

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardPriest,
		BonusCardWorkerPower,
	})

	// Player takes one card
	gs.BonusCards.TakeBonusCard("player1", BonusCardPriest)

	// BON3 has 0 coins initially
	if gs.BonusCards.Available[BonusCardWorkerPower] != 0 {
		t.Errorf("BON3 should have 0 coins initially, got %d", gs.BonusCards.Available[BonusCardWorkerPower])
	}

	// Simulate transitioning from PhaseCleanup (not PhaseSetup) to Round 2
	gs.Phase = PhaseCleanup // NOT PhaseSetup
	gs.Round = 1
	gs.PassOrder = []string{"player1"}
	gs.StartNewRound()

	// BON3 should still have 0 coins (not added by StartNewRound from cleanup phase)
	// Coins are added during cleanup phase, not during StartNewRound from cleanup
	if gs.BonusCards.Available[BonusCardWorkerPower] != 0 {
		t.Errorf("BON3 should have 0 coins after non-setup round transition, got %d", gs.BonusCards.Available[BonusCardWorkerPower])
	}
}
