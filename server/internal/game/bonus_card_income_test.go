package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestIncome_BonusCard_Priest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Priest bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardPriest})
	gs.BonusCards.TakeBonusCard("player1", BonusCardPriest)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+1 priest)
	expectedPriests := 2 + 1
	if player.Resources.Priests != expectedPriests {
		t.Errorf("expected %d priests, got %d", expectedPriests, player.Resources.Priests)
	}
}

func TestIncome_BonusCard_WorkerPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Worker/Power bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardWorkerPower})
	gs.BonusCards.TakeBonusCard("player1", BonusCardWorkerPower)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+1 worker, +3 power)
	expectedWorkers := 5 + 1 + 1 // Base 1 + bonus 1
	expectedBowl2 := 3 // Bonus card power
	
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in Bowl2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}

func TestIncome_BonusCard_6Coins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player 6 Coins bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins})
	gs.BonusCards.TakeBonusCard("player1", BonusCard6Coins)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+6 coins)
	expectedCoins := 10 + 6
	expectedWorkers := 5 + 1 // Base worker
	
	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_DwellingVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Dwelling VP bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardDwellingVP})
	gs.BonusCards.TakeBonusCard("player1", BonusCardDwellingVP)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+2 coins)
	expectedCoins := 10 + 2
	expectedWorkers := 5 + 1 // Base worker
	
	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_TradingHouseVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Trading House VP bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardTradingHouseVP})
	gs.BonusCards.TakeBonusCard("player1", BonusCardTradingHouseVP)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+1 worker)
	expectedWorkers := 5 + 1 + 1 // Base 1 + bonus 1
	
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_Spade(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Spade bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardSpade})
	gs.BonusCards.TakeBonusCard("player1", BonusCardSpade)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+2 coins)
	expectedCoins := 10 + 2
	expectedWorkers := 5 + 1 // Base worker
	
	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_CultAdvance(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Cult Advance bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardCultAdvance})
	gs.BonusCards.TakeBonusCard("player1", BonusCardCultAdvance)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+4 coins)
	expectedCoins := 10 + 4
	expectedWorkers := 5 + 1 // Base worker
	
	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_StrongholdSanctuary(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Stronghold/Sanctuary bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardStrongholdSanctuary})
	gs.BonusCards.TakeBonusCard("player1", BonusCardStrongholdSanctuary)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+2 workers)
	expectedWorkers := 5 + 1 + 2 // Base 1 + bonus 2
	
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestIncome_BonusCard_Shipping(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Shipping bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShipping})
	gs.BonusCards.TakeBonusCard("player1", BonusCardShipping)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+3 power)
	expectedWorkers := 5 + 1 // Base worker
	expectedBowl2 := 3 // Bonus card power
	
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in Bowl2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}

func TestIncome_BonusCard_ShippingVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player Shipping VP bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShippingVP})
	gs.BonusCards.TakeBonusCard("player1", BonusCardShippingVP)

	// Grant income
	gs.GrantIncome()

	// Expected income: Base (Auren: 1 worker) + Bonus card (+3 power)
	expectedWorkers := 5 + 1 // Base worker
	expectedBowl2 := 3 // Bonus card power
	
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in Bowl2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}
