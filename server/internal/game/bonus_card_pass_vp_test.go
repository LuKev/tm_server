package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// TestBonusCard_PassVP_AwardedOnReturn tests that pass VP from bonus cards
// is awarded when RETURNING the card, not when taking it (Bug #33 regression test)
func TestBonusCard_PassVP_AwardedOnReturn(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player 1 shipping level
	player.ShippingLevel = 1

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardShippingVP, // BON10: +3 Power income, +3 VP per shipping level on pass
		BonusCardPriest,     // BON2
	})

	initialVP := player.VictoryPoints

	// Player passes and takes BON10 (Shipping VP card) during setup
	// Should NOT get VP yet (no card to return)
	bonusCard := BonusCardShippingVP
	passAction := NewPassAction("player1", &bonusCard)
	err := passAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass: %v", err)
	}

	// Should NOT have gained VP (no old card to return)
	if player.VictoryPoints != initialVP {
		t.Errorf("should not gain VP when taking first bonus card, expected %d VP, got %d VP", 
			initialVP, player.VictoryPoints)
	}

	// Advance to next round, player passes and takes BON2
	// NOW should get VP from returning BON10
	initialVP = player.VictoryPoints
	bonusCard2 := BonusCardPriest
	passAction2 := NewPassAction("player1", &bonusCard2)
	err = passAction2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass second time: %v", err)
	}

	// Should have gained 3 VP (1 shipping level * 3 VP per level)
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP from returning BON10 (1 shipping * 3 VP), got %d VP", 
			expectedVP, player.VictoryPoints)
	}
}

// TestBonusCard_PassVP_DwellingCard tests that dwelling VP bonus card
// awards VP when returned (Bug #33 regression test)
func TestBonusCard_PassVP_DwellingCard(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player resources and build 3 dwellings
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(1, 0)
	hex3 := board.NewHex(2, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex2).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex3).Terrain = faction.GetHomeTerrain()

	// Build 3 dwellings
	action1 := NewTransformAndBuildAction("player1", hex1, true)
	action1.Execute(gs)
	action2 := NewTransformAndBuildAction("player1", hex2, true)
	action2.Execute(gs)
	action3 := NewTransformAndBuildAction("player1", hex3, true)
	action3.Execute(gs)

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardDwellingVP, // BON5: +1 VP per dwelling on pass
		BonusCardPriest,     // BON2
	})

	// Pass and take dwelling VP card (first time, no return)
	initialVP := player.VictoryPoints
	bonusCard := BonusCardDwellingVP
	passAction := NewPassAction("player1", &bonusCard)
	err := passAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass: %v", err)
	}

	// Should NOT gain VP when taking the card
	if player.VictoryPoints != initialVP {
		t.Errorf("should not gain VP when taking first bonus card, expected %d VP, got %d VP", 
			initialVP, player.VictoryPoints)
	}

	// Pass and take a different card (return dwelling VP card)
	initialVP = player.VictoryPoints
	bonusCard2 := BonusCardPriest
	passAction2 := NewPassAction("player1", &bonusCard2)
	err = passAction2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass second time: %v", err)
	}

	// Should gain 3 VP (3 dwellings * 1 VP per dwelling)
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP from returning dwelling card (3 dwellings * 1 VP), got %d VP", 
			expectedVP, player.VictoryPoints)
	}
}

// TestBonusCard_PassVP_TradingHouseCard tests that trading house VP bonus card
// awards VP when returned (Bug #33 regression test)
func TestBonusCard_PassVP_TradingHouseCard(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player resources and build 1 dwelling, then upgrade to trading house
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	hex1 := board.NewHex(0, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()

	// Build dwelling
	action1 := NewTransformAndBuildAction("player1", hex1, true)
	action1.Execute(gs)

	// Upgrade to trading house
	upgradeAction := NewUpgradeBuildingAction("player1", hex1, models.BuildingTradingHouse)
	upgradeAction.Execute(gs)

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardTradingHouseVP, // BON6: +2 VP per trading house on pass
		BonusCardPriest,         // BON2
	})

	// Pass and take trading house VP card (first time, no return)
	initialVP := player.VictoryPoints
	bonusCard := BonusCardTradingHouseVP
	passAction := NewPassAction("player1", &bonusCard)
	err := passAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass: %v", err)
	}

	// Should NOT gain VP when taking the card
	if player.VictoryPoints != initialVP {
		t.Errorf("should not gain VP when taking first bonus card, expected %d VP, got %d VP", 
			initialVP, player.VictoryPoints)
	}

	// Pass and take a different card (return trading house VP card)
	initialVP = player.VictoryPoints
	bonusCard2 := BonusCardPriest
	passAction2 := NewPassAction("player1", &bonusCard2)
	err = passAction2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to pass second time: %v", err)
	}

	// Should gain 2 VP (1 trading house * 2 VP per trading house)
	expectedVP := initialVP + 2
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP from returning trading house card (1 TH * 2 VP), got %d VP", 
			expectedVP, player.VictoryPoints)
	}
}
