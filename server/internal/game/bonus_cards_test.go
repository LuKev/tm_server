package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

// Regression test for Bug #5: Bonus cards returned too early
// The bug was that bonus cards were being returned to the pool at round transitions
// BEFORE income was calculated, causing players to receive no bonus card income.
// Bonus cards should be kept across rounds and only returned when selecting a new card.
func TestBonusCards_RetainedForNextRoundIncome(t *testing.T) {
	gs := NewGameState()

	// Add players
	gs.AddPlayer("player1", factions.NewWitches())
	gs.AddPlayer("player2", factions.NewEngineers())

	// Give players bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCard6Coins,        // +6 coins
		BonusCardWorkerPower,   // +1 worker, +3 power
	})

	// Simulate players passing and selecting bonus cards in Round 1
	gs.BonusCards.TakeBonusCard("player1", BonusCard6Coins)
	gs.BonusCards.TakeBonusCard("player2", BonusCardWorkerPower)

	// Verify players have their cards
	card1, ok1 := gs.BonusCards.GetPlayerCard("player1")
	if !ok1 || card1 != BonusCard6Coins {
		t.Error("player1 should have 6 coins bonus card")
	}

	card2, ok2 := gs.BonusCards.GetPlayerCard("player2")
	if !ok2 || card2 != BonusCardWorkerPower {
		t.Error("player2 should have worker+power bonus card")
	}

	// Start Round 2 (this should NOT return bonus cards)
	gs.StartNewRound()

	// Verify players STILL have their cards after round transition
	card1, ok1 = gs.BonusCards.GetPlayerCard("player1")
	if !ok1 || card1 != BonusCard6Coins {
		t.Error("player1 should still have 6 coins bonus card after round transition")
	}

	card2, ok2 = gs.BonusCards.GetPlayerCard("player2")
	if !ok2 || card2 != BonusCardWorkerPower {
		t.Error("player2 should still have worker+power bonus card after round transition")
	}

	// Calculate income (bonus cards should provide income)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	initialCoins1 := player1.Resources.Coins
	initialWorkers2 := player2.Resources.Workers

	gs.GrantIncome()

	// Verify player1 got +6 coins from bonus card
	expectedCoins := initialCoins1 + 6 // 6 from bonus card
	if player1.Resources.Coins < expectedCoins {
		t.Errorf("player1 should have gained at least 6 coins from bonus card, got %d coins (started with %d)",
			player1.Resources.Coins, initialCoins1)
	}

	// Verify player2 got +1 worker from bonus card
	expectedWorkers := initialWorkers2 + 1 // 1 from bonus card
	if player2.Resources.Workers < expectedWorkers {
		t.Errorf("player2 should have gained at least 1 worker from bonus card, got %d workers (started with %d)",
			player2.Resources.Workers, initialWorkers2)
	}
}

// Test that bonus cards ARE returned when selecting a new one
func TestBonusCards_ReturnedWhenSelectingNew(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewWitches())

	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCard6Coins,
		BonusCardWorkerPower,
		BonusCardPriest,
	})

	// Take first bonus card
	_, err := gs.BonusCards.TakeBonusCard("player1", BonusCard6Coins)
	if err != nil {
		t.Fatalf("failed to take bonus card: %v", err)
	}

	// Verify card is no longer available
	if gs.BonusCards.IsAvailable(BonusCard6Coins) {
		t.Error("bonus card should not be available after being taken")
	}

	// Simulate next round - player passes and selects new bonus card
	gs.BonusCards.PlayerHasCard["player1"] = false // Reset for new round

	// Take second bonus card (should return first one)
	_, err = gs.BonusCards.TakeBonusCard("player1", BonusCardWorkerPower)
	if err != nil {
		t.Fatalf("failed to take second bonus card: %v", err)
	}

	// Verify first card was returned and is now available
	if !gs.BonusCards.IsAvailable(BonusCard6Coins) {
		t.Error("first bonus card should be returned when taking a new one")
	}

	// Verify player has new card
	card, ok := gs.BonusCards.GetPlayerCard("player1")
	if !ok || card != BonusCardWorkerPower {
		t.Error("player should have new bonus card")
	}
}
