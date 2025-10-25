package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestSelectFavorTile_AfterTemple(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to temple
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainForest)

	// Upgrade to temple
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingTemple)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to temple: %v", err)
	}

	// Verify pending favor tile selection was created
	if gs.PendingFavorTileSelection == nil {
		t.Fatal("expected pending favor tile selection after building temple")
	}

	if gs.PendingFavorTileSelection.PlayerID != "player1" {
		t.Errorf("expected player1, got %s", gs.PendingFavorTileSelection.PlayerID)
	}

	if gs.PendingFavorTileSelection.Count != 1 {
		t.Errorf("expected 1 favor tile to select, got %d", gs.PendingFavorTileSelection.Count)
	}

	// Select a favor tile
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorFire3,
	}

	err = selectAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select favor tile: %v", err)
	}

	// Verify player has the tile
	if !gs.FavorTiles.HasTileType("player1", FavorFire3) {
		t.Error("player should have Fire+3 favor tile")
	}

	// Verify pending selection was cleared
	if gs.PendingFavorTileSelection != nil {
		t.Error("pending favor tile selection should be cleared after selecting all tiles")
	}

	// Verify tile was taken from available pool
	if gs.FavorTiles.Available[FavorFire3] != 0 {
		t.Errorf("expected 0 Fire+3 tiles remaining, got %d", gs.FavorTiles.Available[FavorFire3])
	}
}

func TestSelectFavorTile_AfterSanctuary(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a temple to upgrade to sanctuary
	templeHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(templeHex, &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(templeHex, models.TerrainForest)

	// Upgrade to sanctuary
	action := NewUpgradeBuildingAction("player1", templeHex, models.BuildingSanctuary)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to sanctuary: %v", err)
	}

	// Verify pending favor tile selection was created
	if gs.PendingFavorTileSelection == nil {
		t.Fatal("expected pending favor tile selection after building sanctuary")
	}

	// Select a favor tile
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorWater2,
	}

	err = selectAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select favor tile: %v", err)
	}

	// Verify player has the tile
	if !gs.FavorTiles.HasTileType("player1", FavorWater2) {
		t.Error("player should have Water+2 favor tile")
	}
}

func TestSelectFavorTile_ChaosMagiciansGetTwoTiles(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to temple
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainSwamp)

	// Upgrade to temple
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingTemple)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to temple: %v", err)
	}

	// Verify pending favor tile selection was created with count=2
	if gs.PendingFavorTileSelection == nil {
		t.Fatal("expected pending favor tile selection after building temple")
	}

	if gs.PendingFavorTileSelection.Count != 2 {
		t.Errorf("expected Chaos Magicians to get 2 favor tiles, got %d", gs.PendingFavorTileSelection.Count)
	}

	// Select first favor tile
	selectAction1 := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorFire2,
	}

	err = selectAction1.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select first favor tile: %v", err)
	}

	// Verify pending selection still exists (need to select one more)
	if gs.PendingFavorTileSelection == nil {
		t.Error("pending favor tile selection should still exist after selecting 1 of 2 tiles")
	}

	// Select second favor tile
	selectAction2 := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorWater1,
	}

	err = selectAction2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select second favor tile: %v", err)
	}

	// Verify player has both tiles
	if !gs.FavorTiles.HasTileType("player1", FavorFire2) {
		t.Error("player should have Fire+2 favor tile")
	}
	if !gs.FavorTiles.HasTileType("player1", FavorWater1) {
		t.Error("player should have Water+1 favor tile")
	}

	// Verify pending selection was cleared after selecting both
	if gs.PendingFavorTileSelection != nil {
		t.Error("pending favor tile selection should be cleared after selecting all 2 tiles")
	}
}

func TestSelectFavorTile_AurenStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainForest)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify pending favor tile selection was created (Auren bonus)
	if gs.PendingFavorTileSelection == nil {
		t.Fatal("expected pending favor tile selection after Auren builds stronghold")
	}

	if gs.PendingFavorTileSelection.Count != 1 {
		t.Errorf("expected 1 favor tile for Auren stronghold, got %d", gs.PendingFavorTileSelection.Count)
	}

	// Select a favor tile
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorEarth3,
	}

	err = selectAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select favor tile: %v", err)
	}

	// Verify player has the tile
	if !gs.FavorTiles.HasTileType("player1", FavorEarth3) {
		t.Error("player should have Earth+3 favor tile")
	}

	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("Auren should have stronghold ability")
	}
}

func TestSelectFavorTile_CannotSelectUnavailable(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", factions.NewWitches())

	// Player 2 takes the only Fire+3 tile
	gs.FavorTiles.TakeFavorTile("player2", FavorFire3)

	// Create pending selection for player1
	gs.PendingFavorTileSelection = &PendingFavorTileSelection{
		PlayerID:      "player1",
		Count:         1,
		SelectedTiles: []FavorTileType{},
	}

	// Try to select Fire+3 (already taken)
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorFire3,
	}

	err := selectAction.Validate(gs)
	if err == nil {
		t.Error("expected error when selecting unavailable tile")
	}
}

func TestSelectFavorTile_CannotSelectDuplicateType(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)

	// Player already has Water+2
	gs.FavorTiles.TakeFavorTile("player1", FavorWater2)

	// Create pending selection for 2 more tiles
	gs.PendingFavorTileSelection = &PendingFavorTileSelection{
		PlayerID:      "player1",
		Count:         2,
		SelectedTiles: []FavorTileType{},
	}

	// Try to select Water+2 again
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorWater2,
	}

	err := selectAction.Validate(gs)
	if err == nil {
		t.Error("expected error when selecting duplicate tile type")
	}
}

func TestSelectFavorTile_Fire3AppliesImmediately(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	_ = gs.GetPlayer("player1")

	// Initialize cult tracks
	gs.CultTracks.InitializePlayer("player1")

	// Create pending selection
	gs.PendingFavorTileSelection = &PendingFavorTileSelection{
		PlayerID:      "player1",
		Count:         1,
		SelectedTiles: []FavorTileType{},
	}

	initialFirePosition := gs.CultTracks.GetPosition("player1", CultFire)

	// Select Fire+3 tile (should immediately advance +3 on Fire cult)
	selectAction := &SelectFavorTileAction{
		BaseAction: BaseAction{
			Type:     ActionSelectFavorTile,
			PlayerID: "player1",
		},
		TileType: FavorFire3,
	}

	err := selectAction.Execute(gs)
	if err != nil {
		t.Fatalf("failed to select favor tile: %v", err)
	}

	// Verify cult advancement
	newFirePosition := gs.CultTracks.GetPosition("player1", CultFire)
	if newFirePosition != initialFirePosition+3 {
		t.Errorf("expected Fire position to increase by 3, got %d → %d", initialFirePosition, newFirePosition)
	}
}
