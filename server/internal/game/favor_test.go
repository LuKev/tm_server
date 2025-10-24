package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestFavorTileState_Initialize(t *testing.T) {
	fts := NewFavorTileState()

	// Verify all tiles are available in correct quantities
	if fts.Available[FavorFire3] != 1 {
		t.Errorf("expected 1 Fire+3 tile, got %d", fts.Available[FavorFire3])
	}
	if fts.Available[FavorWater2] != 3 {
		t.Errorf("expected 3 Water+2 tiles, got %d", fts.Available[FavorWater2])
	}
	if fts.Available[FavorEarth1] != 3 {
		t.Errorf("expected 3 Earth+1 tiles, got %d", fts.Available[FavorEarth1])
	}
}

func TestFavorTileState_TakeTile(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())

	// Take a favor tile
	err := gs.FavorTiles.TakeFavorTile("player1", FavorFire3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tile was taken
	if gs.FavorTiles.Available[FavorFire3] != 0 {
		t.Errorf("expected 0 Fire+3 tiles remaining, got %d", gs.FavorTiles.Available[FavorFire3])
	}

	// Verify player has the tile
	if !gs.FavorTiles.HasTileType("player1", FavorFire3) {
		t.Error("player should have Fire+3 tile")
	}

	tiles := gs.FavorTiles.GetPlayerTiles("player1")
	if len(tiles) != 1 {
		t.Errorf("expected player to have 1 tile, got %d", len(tiles))
	}
}

func TestFavorTileState_CannotTakeSameType(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())

	// Take first Water+2 tile
	err := gs.FavorTiles.TakeFavorTile("player1", FavorWater2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to take another Water+2 tile (should fail - only one type per player)
	err = gs.FavorTiles.TakeFavorTile("player1", FavorWater2)
	if err == nil {
		t.Error("expected error when taking duplicate tile type")
	}
}

func TestFavorTileState_CannotTakeUnavailable(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewWitches())

	// Player 1 takes the only Fire+3 tile
	gs.FavorTiles.TakeFavorTile("player1", FavorFire3)

	// Player 2 tries to take Fire+3 (should fail - none available)
	err := gs.FavorTiles.TakeFavorTile("player2", FavorFire3)
	if err == nil {
		t.Error("expected error when taking unavailable tile")
	}
}

func TestApplyFavorTileImmediate_CultAdvancement(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Take and apply Fire+3 favor tile
	gs.FavorTiles.TakeFavorTile("player1", FavorFire3)
	err := ApplyFavorTileImmediate(gs, "player1", FavorFire3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cult track advancement (3 spaces on Fire)
	if gs.CultTracks.GetPosition("player1", CultFire) != 3 {
		t.Errorf("expected Fire position 3, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}

	// Verify power gained (1 bonus at position 3)
	if player.Resources.Power.Bowl2 != 1 {
		t.Errorf("expected 1 power in Bowl2, got %d", player.Resources.Power.Bowl2)
	}
}

func TestGetFavorTileIncomeBonus(t *testing.T) {
	// Test Fire+1 (3 coins)
	tiles := []FavorTileType{FavorFire1}
	coins, workers, power := GetFavorTileIncomeBonus(tiles)
	if coins != 3 || workers != 0 || power != 0 {
		t.Errorf("Fire+1: expected 3 coins, got %d coins, %d workers, %d power", coins, workers, power)
	}

	// Test Earth+2 (1 worker, 1 power)
	tiles = []FavorTileType{FavorEarth2}
	coins, workers, power = GetFavorTileIncomeBonus(tiles)
	if coins != 0 || workers != 1 || power != 1 {
		t.Errorf("Earth+2: expected 1 worker + 1 power, got %d coins, %d workers, %d power", coins, workers, power)
	}

	// Test Air+2 (4 power)
	tiles = []FavorTileType{FavorAir2}
	coins, workers, power = GetFavorTileIncomeBonus(tiles)
	if coins != 0 || workers != 0 || power != 4 {
		t.Errorf("Air+2: expected 4 power, got %d coins, %d workers, %d power", coins, workers, power)
	}

	// Test multiple tiles
	tiles = []FavorTileType{FavorFire1, FavorEarth2, FavorAir2}
	coins, workers, power = GetFavorTileIncomeBonus(tiles)
	if coins != 3 || workers != 1 || power != 5 {
		t.Errorf("Multiple: expected 3 coins + 1 worker + 5 power, got %d coins, %d workers, %d power", coins, workers, power)
	}
}

func TestGetTownPowerRequirement(t *testing.T) {
	// Without Fire+2 tile
	tiles := []FavorTileType{FavorFire1}
	req := GetTownPowerRequirement(tiles)
	if req != 7 {
		t.Errorf("expected power requirement 7, got %d", req)
	}

	// With Fire+2 tile
	tiles = []FavorTileType{FavorFire2}
	req = GetTownPowerRequirement(tiles)
	if req != 6 {
		t.Errorf("expected power requirement 6 (with Fire+2), got %d", req)
	}
}

func TestGetAir1PassVP(t *testing.T) {
	// Without Air+1 tile
	tiles := []FavorTileType{FavorFire1}
	vp := GetAir1PassVP(tiles, 3)
	if vp != 0 {
		t.Errorf("expected 0 VP without Air+1, got %d", vp)
	}

	// With Air+1 tile
	tiles = []FavorTileType{FavorAir1}

	testCases := []struct {
		tradingHouses int
		expectedVP    int
	}{
		{0, 0},
		{1, 2},
		{2, 3},
		{3, 3},
		{4, 4},
		{5, 4}, // Max 4 trading houses
	}

	for _, tc := range testCases {
		vp := GetAir1PassVP(tiles, tc.tradingHouses)
		if vp != tc.expectedVP {
			t.Errorf("expected %d VP for %d trading houses, got %d", tc.expectedVP, tc.tradingHouses, vp)
		}
	}
}

func TestHasFavorTile(t *testing.T) {
	tiles := []FavorTileType{FavorFire1, FavorWater2, FavorEarth3}

	if !HasFavorTile(tiles, FavorFire1) {
		t.Error("should have Fire+1")
	}
	if !HasFavorTile(tiles, FavorWater2) {
		t.Error("should have Water+2")
	}
	if HasFavorTile(tiles, FavorAir3) {
		t.Error("should not have Air+3")
	}
}

func TestFavorTileQuantities(t *testing.T) {
	allTiles := GetAllFavorTiles()

	// Verify +3 tiles have quantity 1
	plus3Tiles := []FavorTileType{FavorFire3, FavorWater3, FavorEarth3, FavorAir3}
	for _, tileType := range plus3Tiles {
		tile := allTiles[tileType]
		if tile.AvailableQty != 1 {
			t.Errorf("%s should have quantity 1, got %d", tile.Name, tile.AvailableQty)
		}
		if tile.CultAdvance != 3 {
			t.Errorf("%s should advance 3 spaces, got %d", tile.Name, tile.CultAdvance)
		}
	}

	// Verify +2 and +1 tiles have quantity 3
	otherTiles := []FavorTileType{
		FavorFire2, FavorWater2, FavorEarth2, FavorAir2,
		FavorFire1, FavorWater1, FavorEarth1, FavorAir1,
	}
	for _, tileType := range otherTiles {
		tile := allTiles[tileType]
		if tile.AvailableQty != 3 {
			t.Errorf("%s should have quantity 3, got %d", tile.Name, tile.AvailableQty)
		}
		if !tile.HasAbility {
			t.Errorf("%s should have an ability", tile.Name)
		}
	}
}
