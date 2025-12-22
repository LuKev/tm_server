package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestScoringTileInitialization(t *testing.T) {
	sts := NewScoringTileState()

	err := sts.InitializeForGame()
	if err != nil {
		t.Fatalf("failed to initialize scoring tiles: %v", err)
	}

	if len(sts.Tiles) != 6 {
		t.Errorf("expected 6 tiles, got %d", len(sts.Tiles))
	}

	// Check that spades tile is not in rounds 5 or 6
	for i := 4; i < 6; i++ {
		if sts.Tiles[i].Type == ScoringSpades {
			t.Errorf("spades tile found in round %d (should not be in rounds 5 or 6)", i+1)
		}
	}
}

func TestScoringTile_DwellingVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile for round 1: Dwelling + Water (2 VP per dwelling)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringDwellingWater,
			ActionType: ScoringActionDwelling,
			ActionVP:   2,
		},
	}

	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	initialVP := player.VictoryPoints

	// Build a dwelling
	action := NewTransformAndBuildAction("player1", board.NewHex(0, 0), true, models.TerrainTypeUnknown)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to build dwelling: %v", err)
	}

	// Should get 2 VP from scoring tile
	if player.VictoryPoints != initialVP+2 {
		t.Errorf("expected %d VP (+2 from scoring tile), got %d", initialVP+2, player.VictoryPoints)
	}
}

func TestScoringTile_TradingHouseVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile for round 1: Trading House + Water (3 VP per TH)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringTradingHouseWater,
			ActionType: ScoringActionTradingHouse,
			ActionVP:   3,
		},
	}

	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100
	player.Resources.Priests = 5

	// Place a dwelling first
	gs.Map.PlaceBuilding(board.NewHex(0, 0), &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.GetHex(board.NewHex(0, 0)).Terrain = faction.GetHomeTerrain()

	initialVP := player.VictoryPoints

	// Upgrade to trading house
	action := NewUpgradeBuildingAction("player1", board.NewHex(0, 0), models.BuildingTradingHouse)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to trading house: %v", err)
	}

	// Should get 3 VP from scoring tile
	if player.VictoryPoints != initialVP+3 {
		t.Errorf("expected %d VP (+3 from scoring tile), got %d", initialVP+3, player.VictoryPoints)
	}
}

func TestScoringTile_StrongholdVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile for round 1: Stronghold + Fire (5 VP per SH/SA)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringStrongholdFire,
			ActionType: ScoringActionStronghold,
			ActionVP:   5,
		},
	}

	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100
	player.Resources.Priests = 5

	// Place a trading house first
	gs.Map.PlaceBuilding(board.NewHex(0, 0), &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.GetHex(board.NewHex(0, 0)).Terrain = faction.GetHomeTerrain()

	initialVP := player.VictoryPoints

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", board.NewHex(0, 0), models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Should get 5 VP from scoring tile
	if player.VictoryPoints != initialVP+5 {
		t.Errorf("expected %d VP (+5 from scoring tile), got %d", initialVP+5, player.VictoryPoints)
	}
}

func TestScoringTile_SpadesVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants() // Giants always use 2 spades
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile for round 1: Spades + Earth (2 VP per spade)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringSpades,
			ActionType: ScoringActionSpades,
			ActionVP:   2,
		},
	}

	// Verify scoring tile is set up correctly
	tile := gs.ScoringTiles.GetTileForRound(1)
	if tile == nil {
		t.Fatal("scoring tile not found for round 1")
	}
	if tile.ActionVP != 2 {
		t.Errorf("expected ActionVP=2, got %d", tile.ActionVP)
	}

	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	initialVP := player.VictoryPoints
	t.Logf("Initial VP: %d", initialVP)

	// Check terrain at (0,0)
	hex := board.NewHex(0, 0)
	mapHex := gs.Map.GetHex(hex)
	t.Logf("Hex (0,0) terrain: %v, Giants home: %v", mapHex.Terrain, faction.GetHomeTerrain())

	// Calculate expected spades
	distance := gs.Map.GetTerrainDistance(mapHex.Terrain, faction.GetHomeTerrain())
	workersNeeded := faction.GetTerraformCost(distance)
	t.Logf("Distance: %d, Workers needed: %d", distance, workersNeeded)

	// Giants always use exactly 2 spades regardless of distance
	expectedSpades := 2
	t.Logf("Expected spades for Giants: %d", expectedSpades)

	// Transform terrain (Giants always use 2 spades regardless of distance)
	action := NewTransformAndBuildAction("player1", hex, false, models.TerrainTypeUnknown)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to transform terrain: %v", err)
	}

	t.Logf("Final VP: %d", player.VictoryPoints)

	// Should get 2 VP per spade (Giants use 2 spades = 4 VP)
	vpGained := player.VictoryPoints - initialVP
	expectedVP := expectedSpades * 2
	if vpGained != expectedVP {
		t.Errorf("expected %d VP (%d spades * 2 VP), got %d", expectedVP, expectedSpades, vpGained)
	}
}

func TestScoringTile_TownVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile for round 1: Town + Earth (5 VP per town)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringTown,
			ActionType: ScoringActionTown,
			ActionVP:   5,
		},
	}

	// Set up 4 connected buildings with power = 7
	hexes := []board.Hex{
		board.NewHex(0, 0),
		board.NewHex(1, 0),
		board.NewHex(2, 0),
		board.NewHex(3, 0),
	}

	for i, h := range hexes {
		powerValue := 1
		buildingType := models.BuildingDwelling
		if i < 3 {
			powerValue = 2
			buildingType = models.BuildingTradingHouse
		}

		gs.Map.PlaceBuilding(h, &models.Building{
			Type:       buildingType,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: powerValue,
		})
		gs.Map.GetHex(h).Terrain = faction.GetHomeTerrain()
	}

	initialVP := player.VictoryPoints

	// Form town
	err := gs.FormTown("player1", hexes, models.TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}

	// Should get 5 VP from town tile + 5 VP from scoring tile = 10 VP total
	expectedVP := initialVP + 5 + 5
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP (+5 from tile, +5 from scoring), got %d", expectedVP, player.VictoryPoints)
	}
}

func TestScoringTile_PriestTracking(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player priests
	player.Resources.Priests = 5

	// Send 2 priests to cult tracks (place on action spaces, not sacrifice)
	// For SCORE5, only priests placed on action spaces (2-3 steps) count
	action1 := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultFire,
		SpacesToClimb: 2, // Place on 2-step action space
	}
	action1.Execute(gs)

	action2 := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultWater,
		SpacesToClimb: 3, // Place on 3-step action space
	}
	action2.Execute(gs)

	// Check that 2 priests were recorded
	if gs.ScoringTiles.GetPriestsSent("player1") != 2 {
		t.Errorf("expected 2 priests sent, got %d", gs.ScoringTiles.GetPriestsSent("player1"))
	}

	// Reset and check
	gs.ScoringTiles.ResetPriestsSent()
	if gs.ScoringTiles.GetPriestsSent("player1") != 0 {
		t.Errorf("expected 0 priests after reset, got %d", gs.ScoringTiles.GetPriestsSent("player1"))
	}
}

func TestAwardCultRewards_CultThreshold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile: 4 steps on Water = 1 priest
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringDwellingWater,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardPriest,
			CultRewardAmount: 1,
		},
	}

	// Advance player to position 4 on Water
	gs.CultTracks.AdvancePlayer("player1", CultWater, 4, player, gs)

	initialPriests := player.Resources.Priests

	// Award cult rewards
	gs.AwardCultRewards()

	// Should get 1 priest
	if player.Resources.Priests != initialPriests+1 {
		t.Errorf("expected %d priests (+1 from cult reward), got %d", initialPriests+1, player.Resources.Priests)
	}
}

func TestAwardCultRewards_PriestCoins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up scoring tile: 2 coins per priest sent to cult (Temple + Priest)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringTemplePriest,
			CultRewardType:   CultRewardCoin,
			CultRewardAmount: 2,
		},
	}

	// Record 3 priests sent
	gs.ScoringTiles.RecordPriestSent("player1")
	gs.ScoringTiles.RecordPriestSent("player1")
	gs.ScoringTiles.RecordPriestSent("player1")

	initialCoins := player.Resources.Coins

	// Award cult rewards
	gs.AwardCultRewards()

	// Should get 6 coins (3 priests * 2 coins)
	if player.Resources.Coins != initialCoins+6 {
		t.Errorf("expected %d coins (+6 from 3 priests), got %d", initialCoins+6, player.Resources.Coins)
	}

	// Priest count should be reset
	if gs.ScoringTiles.GetPriestsSent("player1") != 0 {
		t.Errorf("expected priest count to be reset, got %d", gs.ScoringTiles.GetPriestsSent("player1"))
	}
}
