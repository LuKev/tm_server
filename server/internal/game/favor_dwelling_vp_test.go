package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TestFavorEarth1_BuildDwelling tests that the Earth+1 favor tile
// awards +2 VP when building a dwelling (Bug #31 regression test)
func TestFavorEarth1_BuildDwelling(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Give player resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10

	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build a dwelling on home terrain (no transformation needed)
	targetHex := NewHex(1, 0)
	gs.Map.GetHex(targetHex).Terrain = faction.GetHomeTerrain()

	initialVP := player.VictoryPoints

	// Execute build action
	action := NewTransformAndBuildAction("player1", targetHex, true)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to build dwelling: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}

	// Should get +2 VP from Earth+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected +2 VP from Earth+1 favor tile, got %d VP", vpGained)
	}
}

// TestFavorEarth1_PowerAction tests that Earth+1 bonus is applied
// when building dwelling via power action (Bug #31 regression test)
func TestFavorEarth1_PowerAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Give player resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	player.Resources.Power.Bowl3 = 10

	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build a dwelling via ACT5 (1 free spade power action)
	targetHex := NewHex(1, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainLake // 1 spade away from Plains

	initialVP := player.VictoryPoints

	// Execute power action
	action := NewPowerAction("player1", PowerActionSpade1)
	action.TargetHex = &targetHex
	action.BuildDwelling = true

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute power action: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}

	// Should get +2 VP from Earth+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected +2 VP from Earth+1 favor tile, got %d VP", vpGained)
	}
}

// TestFavorEarth1_WitchesRide tests that Earth+1 bonus is applied
// when building dwelling via Witches special action (Bug #31 regression test)
func TestFavorEarth1_WitchesRide(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build stronghold to unlock Witches' Ride ability
	player.HasStrongholdAbility = true

	// Use Witches' Ride to build on any Forest hex (no adjacency required)
	targetHex := NewHex(5, 5)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints

	// Execute Witches' Ride
	action := NewSpecialAction("player1", SpecialActionWitchesRide)
	action.TargetHex = &targetHex

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Witches' Ride: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}

	// Should get +2 VP from Earth+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected +2 VP from Earth+1 favor tile, got %d VP", vpGained)
	}
}

// TestFavorEarth1_BonusCardSpade tests that Earth+1 bonus is applied
// when building dwelling via bonus card spade action (Bug #31 regression test)
func TestFavorEarth1_BonusCardSpade(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Give player BON1 bonus card (provides 1 free spade)
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true

	// Give player resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10

	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build a dwelling using bonus card spade
	targetHex := NewHex(1, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainLake // 1 spade away from Plains

	initialVP := player.VictoryPoints

	// Execute bonus card spade action
	action := NewSpecialAction("player1", SpecialActionBonusCardSpade)
	action.TargetHex = &targetHex
	action.BuildDwelling = true

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}

	// Should get +2 VP from Earth+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected +2 VP from Earth+1 favor tile, got %d VP", vpGained)
	}
}

// TestFavorEarth1_GiantsTransform tests that Earth+1 bonus is applied
// when building dwelling via Giants special action (Bug #31 regression test)
func TestFavorEarth1_GiantsTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player Earth+1 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth1)

	// Give player resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10

	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 0)
	gs.Map.GetHex(initialHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build stronghold to unlock Giants special action (2 free spades)
	player.HasStrongholdAbility = true

	// Use Giants special action to transform and build
	targetHex := NewHex(1, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainLake // 2 spades away from Wasteland

	initialVP := player.VictoryPoints

	// Execute Giants transform action
	action := NewSpecialAction("player1", SpecialActionGiantsTransform)
	action.TargetHex = &targetHex
	action.BuildDwelling = true

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Giants transform: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}

	// Should get +2 VP from Earth+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected +2 VP from Earth+1 favor tile, got %d VP", vpGained)
	}
}
