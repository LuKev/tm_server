package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Test VP scoring from Dwelling VP bonus card when passing
func TestBonusCard_DwellingVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardDwellingVP, BonusCardPriest})

	// Place 4 dwellings on the map
	for i := 0; i < 4; i++ {
		hex := board.NewHex(i, 0)
		gs.Map.GetHex(hex).Terrain = models.TerrainForest
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		}
	}

	// First pass: take the dwelling VP card (no VP yet)
	bonusCard := BonusCardDwellingVP
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the dwelling VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (4 dwellings × 1 VP = 4 VP)
	expectedVP := initialVP + 4
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (4 dwellings), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test VP scoring from Trading House VP bonus card when passing
func TestBonusCard_TradingHouseVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardTradingHouseVP, BonusCardPriest})

	// Place 3 trading houses on the map
	for i := 0; i < 3; i++ {
		hex := board.NewHex(i, 0)
		gs.Map.GetHex(hex).Terrain = models.TerrainForest
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		}
	}

	// First pass: take the trading house VP card (no VP yet)
	bonusCard := BonusCardTradingHouseVP
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the trading house VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (3 trading houses × 2 VP = 6 VP)
	expectedVP := initialVP + 6
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (3 trading houses), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test VP scoring from Stronghold/Sanctuary VP bonus card when passing
func TestBonusCard_StrongholdSanctuaryVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardStrongholdSanctuary, BonusCardPriest})

	// Place a stronghold
	strongholdHex := board.NewHex(0, 0)
	gs.Map.GetHex(strongholdHex).Terrain = models.TerrainForest
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Place a sanctuary
	sanctuaryHex := board.NewHex(1, 0)
	gs.Map.GetHex(sanctuaryHex).Terrain = models.TerrainForest
	gs.Map.GetHex(sanctuaryHex).Building = &models.Building{
		Type:       models.BuildingSanctuary,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// First pass: take the stronghold/sanctuary VP card (no VP yet)
	bonusCard := BonusCardStrongholdSanctuary
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the stronghold/sanctuary VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (4 VP for stronghold + 4 VP for sanctuary = 8 VP)
	expectedVP := initialVP + 8
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (stronghold + sanctuary), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test VP scoring from Stronghold/Sanctuary VP bonus card with only stronghold
func TestBonusCard_StrongholdOnlyVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardStrongholdSanctuary, BonusCardPriest})

	// Place only a stronghold
	strongholdHex := board.NewHex(0, 0)
	gs.Map.GetHex(strongholdHex).Terrain = models.TerrainForest
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// First pass: take the stronghold/sanctuary VP card (no VP yet)
	bonusCard := BonusCardStrongholdSanctuary
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the stronghold/sanctuary VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (4 VP for stronghold only)
	expectedVP := initialVP + 4
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (stronghold only), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test VP scoring from Shipping VP bonus card when passing
func TestBonusCard_ShippingVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShippingVP, BonusCardPriest})

	// Set shipping level to 3
	player.ShippingLevel = 3

	// First pass: take the shipping VP card (no VP yet)
	bonusCard := BonusCardShippingVP
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the shipping VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (3 shipping × 3 VP = 9 VP)
	expectedVP := initialVP + 9
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (shipping level 3), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test that Dwarves don't benefit from Shipping VP bonus card
func TestBonusCard_ShippingVP_Dwarves(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShippingVP, BonusCardPriest})

	// Set shipping level to 3
	player.ShippingLevel = 3

	// First pass: take the shipping VP card (no VP yet)
	bonusCard := BonusCardShippingVP
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the shipping VP card
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify NO VP was awarded (Dwarves don't benefit from shipping)
	if player.VictoryPoints != initialVP {
		t.Errorf("expected %d VP (Dwarves don't benefit), got %d", initialVP, player.VictoryPoints)
	}
}

// Test that Fakirs don't benefit from Shipping VP bonus card
func TestBonusCard_ShippingVP_Fakirs(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShippingVP, BonusCardPriest})

	// Set shipping level to 3
	player.ShippingLevel = 3

	// First pass: take the shipping VP card (no VP yet)
	bonusCard := BonusCardShippingVP
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset for next round (simulate cleanup and new round)
	player.HasPassed = false
	delete(gs.BonusCards.PlayerHasCard, "player1")

	// Second pass: return the shipping VP card and get VP
	initialVP := player.VictoryPoints
	bonusCard2 := BonusCardPriest
	action2 := NewPassAction("player1", &bonusCard2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Fakirs don't get VP (should still be at initial VP)
	if player.VictoryPoints != initialVP {
		t.Errorf("Fakirs should not gain VP from shipping card, expected %d, got %d", initialVP, player.VictoryPoints)
	}
}

// Test bonus card with no VP bonus (e.g., 6 Coins card)
func TestBonusCard_NoVPBonus(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins})

	initialVP := player.VictoryPoints
	bonusCard := BonusCard6Coins
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no VP was awarded
	if player.VictoryPoints != initialVP {
		t.Errorf("expected %d VP (no bonus), got %d", initialVP, player.VictoryPoints)
	}
}

// Test bonus card coin accumulation
func TestBonusCard_CoinAccumulation(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins, BonusCardPriest})

	// Manually add 3 coins to the 6 Coins card (simulating 3 rounds of accumulation)
	gs.BonusCards.Available[BonusCard6Coins] = 3

	initialCoins := player.Resources.Coins
	bonusCard := BonusCard6Coins
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify player received the accumulated coins
	expectedCoins := initialCoins + 3
	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins (3 accumulated), got %d", expectedCoins, player.Resources.Coins)
	}
}

// Test AddCoinsToLeftoverCards functionality
func TestBonusCard_AddCoinsToLeftover(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())

	// Set up 3 bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCard6Coins, BonusCardPriest, BonusCardWorkerPower,
	})

	// All cards start with 0 coins
	if gs.BonusCards.Available[BonusCard6Coins] != 0 {
		t.Errorf("expected 0 coins on 6 Coins card initially")
	}

	// Add coins to leftover cards
	gs.BonusCards.AddCoinsToLeftoverCards()

	// All 3 cards should now have 1 coin
	if gs.BonusCards.Available[BonusCard6Coins] != 1 {
		t.Errorf("expected 1 coin on 6 Coins card after adding")
	}
	if gs.BonusCards.Available[BonusCardPriest] != 1 {
		t.Errorf("expected 1 coin on Priest card after adding")
	}
	if gs.BonusCards.Available[BonusCardWorkerPower] != 1 {
		t.Errorf("expected 1 coin on Worker/Power card after adding")
	}

	// Add coins again
	gs.BonusCards.AddCoinsToLeftoverCards()

	// All 3 cards should now have 2 coins
	if gs.BonusCards.Available[BonusCard6Coins] != 2 {
		t.Errorf("expected 2 coins on 6 Coins card after second adding")
	}
}
