package game

import (
	"testing"

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
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = models.TerrainForest
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Build a new dwelling on adjacent hex
	targetHex := NewHex(0, 1)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints
	action := NewTransformAndBuildAction("player1", targetHex, true)
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
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = models.TerrainForest
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Build a new dwelling on adjacent hex
	targetHex := NewHex(0, 1)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints
	action := NewTransformAndBuildAction("player1", targetHex, true)
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
	dwellingHex := NewHex(0, 0)
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
	dwellingHex := NewHex(0, 0)
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
		hex := NewHex(i, 0)
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
			hex := NewHex(i, 0)
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
		hex := NewHex(i, 0)
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

// Halflings Scoring Tests

func TestHalflings_RegularTransformScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Give player resources
	player.Resources.Workers = 20
	
	// Find a hex that needs transformation (not already home terrain)
	targetHex := NewHex(0, 0)
	// Make sure it's not already home terrain
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Plains
	
	initialVP := player.VictoryPoints
	
	// Transform (Halflings use 3 spades for distance 3)
	action := NewTransformAndBuildAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to transform: %v", err)
	}
	
	// Should get: 2 VP per spade (scoring tile) + 1 VP per spade (Halflings) = 3 VP per spade
	// Distance 3 = 3 spades, so 3 * 3 = 9 VP total
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3 * 3 // 3 spades * 3 VP per spade
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_BonusCardSpadeScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	player.Resources.Workers = 20
	
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Plains
	
	initialVP := player.VictoryPoints
	
	// Use bonus card spade (1 free spade)
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:    SpecialActionBonusCardSpade,
		TargetHex:     &targetHex,
		BuildDwelling: false,
	}
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use bonus card spade: %v", err)
	}
	
	// Distance 3 = 3 spades, should get 3 * 3 = 9 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3 * 3
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_CultSpadeScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Place a dwelling first to make hex adjacent
	startHex := NewHex(0, 0)
	gs.Map.GetHex(startHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(startHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Give player a pending spade from cult reward
	gs.PendingSpades = make(map[string]int)
	gs.PendingSpades["player1"] = 1
	
	targetHex := NewHex(1, 0) // Adjacent hex
	initialVP := player.VictoryPoints
	
	// Use cult spade
	action := NewUseCultSpadeAction("player1", targetHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use cult spade: %v", err)
	}
	
	// 1 spade: 2 VP (scoring tile) + 1 VP (Halflings) = 3 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_StrongholdSpadesScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold on the faction
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Check that Halflings can use stronghold spades
	if !faction.CanUseStrongholdSpades() {
		t.Fatal("Halflings should be able to use stronghold spades")
	}
	
	spades := faction.UseStrongholdSpades()
	if spades != 3 {
		t.Errorf("expected 3 spades from stronghold, got %d", spades)
	}
	
	// Note: The actual stronghold action implementation is TODO
	// This test just verifies the faction method works
}

// Swarmlings Scoring Tests

func TestSwarmlings_UpgradeWithScoringTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 3 VP per trading house
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringTradingHouseWater,
				ActionType: ScoringActionTradingHouse,
				ActionVP:   3,
			},
		},
	}
	
	// Place a dwelling first
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade (free D→TH)
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP from scoring tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 3 {
		t.Errorf("expected 3 VP from scoring tile, got %d", vpGained)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(upgradeHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Error("building should be upgraded to trading house")
	}
}

func TestSwarmlings_UpgradeWithWater1FavorTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Give player Water+1 favor tile
	gs.FavorTiles = NewFavorTileState()
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorWater1}
	
	// Place a dwelling
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP from Water+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 3 {
		t.Errorf("expected 3 VP from Water+1 favor tile, got %d", vpGained)
	}
}

func TestSwarmlings_UpgradeWithBothScoringAndFavor(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Set up scoring tile
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringTradingHouseWater,
				ActionType: ScoringActionTradingHouse,
				ActionVP:   3,
			},
		},
	}
	
	// Give player Water+1 favor tile
	gs.FavorTiles = NewFavorTileState()
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorWater1}
	
	// Place a dwelling
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP (scoring tile) + 3 VP (Water+1) = 6 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 6
	if vpGained != expectedVP {
		t.Errorf("expected %d VP (scoring + favor), got %d", expectedVP, vpGained)
	}
}
