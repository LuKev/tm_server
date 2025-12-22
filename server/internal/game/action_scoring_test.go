package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Test VP scoring from Earth+1 favor tile when building Dwelling
func TestBuildDwelling_Earth1VP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 5

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Place initial dwelling to establish adjacency
	initialHex := board.NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = models.TerrainForest
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Build a new dwelling on adjacent hex
	targetHex := board.NewHex(0, 1)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints
	action := NewTransformAndBuildAction("player1", targetHex, true, models.TerrainTypeUnknown)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (+2 from Earth+1)
	expectedVP := initialVP + 2
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (Earth+1 bonus), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test no VP scoring when building Dwelling without Earth+1
func TestBuildDwelling_NoEarth1(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 5

	// No Earth+1 favor tile

	// Place initial dwelling to establish adjacency
	initialHex := board.NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = models.TerrainForest
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Build a new dwelling on adjacent hex
	targetHex := board.NewHex(0, 1)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints
	action := NewTransformAndBuildAction("player1", targetHex, true, models.TerrainTypeUnknown)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no VP was awarded
	if player.VictoryPoints != initialVP {
		t.Errorf("expected %d VP (no bonus), got %d", initialVP, player.VictoryPoints)
	}
}

// Test VP scoring from Water+1 favor tile when upgrading to Trading House
func TestUpgradeToTradingHouse_Water1VP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 5

	// Give player Water+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorWater1)

	// Place a dwelling
	dwellingHex := board.NewHex(0, 0)
	gs.Map.GetHex(dwellingHex).Terrain = models.TerrainForest
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	initialVP := player.VictoryPoints
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (+3 from Water+1)
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (Water+1 bonus), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test no VP scoring when upgrading to Trading House without Water+1
func TestUpgradeToTradingHouse_NoWater1(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 5

	// No Water+1 favor tile

	// Place a dwelling
	dwellingHex := board.NewHex(0, 0)
	gs.Map.GetHex(dwellingHex).Terrain = models.TerrainForest
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	initialVP := player.VictoryPoints
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no VP was awarded
	if player.VictoryPoints != initialVP {
		t.Errorf("expected %d VP (no bonus), got %d", initialVP, player.VictoryPoints)
	}
}

// Test VP scoring from Air+1 favor tile when passing
func TestPass_Air1VP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins})

	// Give player Air+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorAir1)

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

	initialVP := player.VictoryPoints
	bonusCard := BonusCard6Coins
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify VP was awarded (3 trading houses = 3 VP from Air+1)
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (Air+1: 3 TH), got %d", expectedVP, player.VictoryPoints)
	}
}

// Test Air+1 VP scaling with different Trading House counts
func TestPass_Air1VP_Scaling(t *testing.T) {
	testCases := []struct {
		tradingHouses int
		expectedVP    int
	}{
		{0, 0},
		{1, 2},
		{2, 3},
		{3, 3},
		{4, 4},
	}

	for _, tc := range testCases {
		gs := NewGameState()
		faction := factions.NewAuren()
		gs.AddPlayer("player1", faction)
		player := gs.GetPlayer("player1")

		// Set up bonus cards
		gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins})

		// Give player Air+1 favor tile
		gs.FavorTiles.TakeFavorTile("player1", FavorAir1)

		// Place trading houses
		for i := 0; i < tc.tradingHouses; i++ {
			hex := board.NewHex(i, 0)
			gs.Map.GetHex(hex).Terrain = models.TerrainForest
			gs.Map.GetHex(hex).Building = &models.Building{
				Type:       models.BuildingTradingHouse,
				Faction:    faction.GetType(),
				PlayerID:   "player1",
				PowerValue: 2,
			}
		}

		initialVP := player.VictoryPoints
		bonusCard := BonusCard6Coins
		action := NewPassAction("player1", &bonusCard)
		err := action.Execute(gs)
		if err != nil {
			t.Fatalf("unexpected error for %d TH: %v", tc.tradingHouses, err)
		}

		expectedVP := initialVP + tc.expectedVP
		if player.VictoryPoints != expectedVP {
			t.Errorf("%d TH: expected %d VP, got %d", tc.tradingHouses, expectedVP, player.VictoryPoints)
		}
	}
}

// Test no VP scoring when passing without Air+1
func TestPass_NoAir1(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCard6Coins})

	// No Air+1 favor tile

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

	initialVP := player.VictoryPoints
	bonusCard := BonusCard6Coins
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no VP was awarded from Air+1 (player doesn't have it)
	if player.VictoryPoints != initialVP {
		t.Errorf("expected %d VP (no bonus), got %d", initialVP, player.VictoryPoints)
	}
}
