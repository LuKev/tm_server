package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ============================================================================
// MERMAIDS TESTS - River Skipping for Town Formation
// ============================================================================

func TestMermaids_RiverSkippingTownFormation(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	_ = gs.GetPlayer("player1")
	
	// Set up: 4 buildings separated by a river
	// Buildings on both sides of the river
	// Layout:
	//   B1  B2
	//   |  /
	//   R (river)
	//  /  |
	// B3  B4
	
	// Buildings on one side (Lake terrain)
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	
	// River hex in the middle
	riverHex := NewHex(0, 1)
	
	// Buildings on the other side (Lake terrain)
	hex3 := NewHex(0, 2)
	hex4 := NewHex(1, 2)
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = models.TerrainLake
	gs.Map.GetHex(hex2).Terrain = models.TerrainLake
	gs.Map.GetHex(riverHex).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = models.TerrainLake
	gs.Map.GetHex(hex4).Terrain = models.TerrainLake
	
	// Place buildings (Trading Houses - 2 power each, total = 8 > 7)
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionMermaids,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Check that town can be formed (Mermaids skip river)
	connected := gs.CheckForTownFormation("player1", hex1)
	if connected == nil {
		t.Fatal("expected Mermaids to form town by skipping river")
	}
	
	// Verify all 4 buildings are connected
	if len(connected) != 4 {
		t.Errorf("expected 4 connected buildings, got %d", len(connected))
	}
	
	// Verify pending town formation was created
	pending := gs.PendingTownFormations["player1"]
	if pending == nil {
		t.Fatal("expected pending town formation")
	}
	
	// Verify the river hex was tracked
	if pending.SkippedRiverHex == nil {
		t.Error("expected skipped river hex to be tracked")
	} else if *pending.SkippedRiverHex != riverHex {
		t.Errorf("expected river hex %v, got %v", riverHex, *pending.SkippedRiverHex)
	}
	
	// Verify this town can be delayed (uses river skipping)
	if !pending.CanBeDelayed {
		t.Error("expected Mermaids town with river skipping to be delayable")
	}
}

func TestMermaids_LandOnlyTownMustClaimImmediately(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	_ = gs.GetPlayer("player1")
	
	// Set up: 4 buildings all adjacent on land (no river)
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	hex3 := NewHex(0, 1)
	hex4 := NewHex(1, 1)
	
	// Set up terrain (all Lake)
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.GetHex(hex).Terrain = models.TerrainLake
	}
	
	// Place buildings (Trading Houses - 2 power each, total = 8 > 7)
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionMermaids,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Check that town can be formed
	connected := gs.CheckForTownFormation("player1", hex1)
	if connected == nil {
		t.Fatal("expected town to be formable")
	}
	
	// Verify pending town formation was created
	pending := gs.PendingTownFormations["player1"]
	if pending == nil {
		t.Fatal("expected pending town formation")
	}
	
	// Verify NO river hex was skipped
	if pending.SkippedRiverHex != nil {
		t.Error("expected no skipped river hex for land-only town")
	}
	
	// Verify this town CANNOT be delayed (land tiles only)
	if pending.CanBeDelayed {
		t.Error("expected Mermaids land-only town to NOT be delayable (must claim immediately)")
	}
}

func TestMermaids_CannotSkipMultipleRivers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	
	// Set up: Buildings separated by TWO rivers
	// Should only skip one river, not both
	hex1 := NewHex(0, 0)
	river1 := NewHex(0, 1)
	hex2 := NewHex(0, 2)
	river2 := NewHex(0, 3)
	hex3 := NewHex(0, 4)
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = models.TerrainLake
	gs.Map.GetHex(river1).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex2).Terrain = models.TerrainLake
	gs.Map.GetHex(river2).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = models.TerrainLake
	
	// Place buildings at hex1, hex2, hex3
	for _, hex := range []Hex{hex1, hex2, hex3} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionMermaids,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Check connected buildings from hex1
	connected, skippedRiver := gs.Map.GetConnectedBuildingsForMermaids(hex1, "player1")
	
	// Should connect hex1 and hex2 (skipping river1), but NOT hex3 (second river)
	if len(connected) != 2 {
		t.Errorf("expected 2 connected buildings (hex1, hex2), got %d", len(connected))
	}
	
	// Verify only one river was skipped
	if skippedRiver == nil {
		t.Error("expected one river to be skipped")
	} else if *skippedRiver != river1 {
		t.Errorf("expected river1 to be skipped, got %v", *skippedRiver)
	}
}

func TestMermaids_TownTilePlacedOnSkippedRiver(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	_ = gs.GetPlayer("player1")
	
	// Set up: 4 buildings separated by a river
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	riverHex := NewHex(0, 1)
	hex3 := NewHex(0, 2)
	hex4 := NewHex(1, 2)
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = models.TerrainLake
	gs.Map.GetHex(hex2).Terrain = models.TerrainLake
	gs.Map.GetHex(riverHex).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = models.TerrainLake
	gs.Map.GetHex(hex4).Terrain = models.TerrainLake
	
	// Place buildings
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionMermaids,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Check for town formation
	gs.CheckForTownFormation("player1", hex1)
	
	// Get pending town
	pending := gs.PendingTownFormations["player1"]
	if pending == nil {
		t.Fatal("expected pending town formation")
	}
	
	// Form the town with a specific tile (passing skipped river hex)
	err := gs.FormTown("player1", pending.Hexes, TownTile5Points, pending.SkippedRiverHex)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify town tile was placed on the skipped river hex
	riverMapHex := gs.Map.GetHex(riverHex)
	if !riverMapHex.HasTownTile {
		t.Error("expected town tile to be placed on the skipped river hex")
	}
	if riverMapHex.TownTileType != TownTile5Points {
		t.Errorf("expected town tile type %v, got %v", TownTile5Points, riverMapHex.TownTileType)
	}
	
	// Verify buildings are marked as part of town
	for _, hex := range pending.Hexes {
		mapHex := gs.Map.GetHex(hex)
		if !mapHex.PartOfTown {
			t.Errorf("expected building at %v to be marked as part of town", hex)
		}
	}
}

func TestMermaids_NonMermaidsCannotSkipRiver(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren() // Not Mermaids
	gs.AddPlayer("player1", faction)
	
	// Set up: 4 buildings separated by a river
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	riverHex := NewHex(0, 1)
	hex3 := NewHex(0, 2)
	hex4 := NewHex(1, 2)
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = models.TerrainForest
	gs.Map.GetHex(hex2).Terrain = models.TerrainForest
	gs.Map.GetHex(riverHex).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = models.TerrainForest
	gs.Map.GetHex(hex4).Terrain = models.TerrainForest
	
	// Place buildings
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionAuren,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Check that town CANNOT be formed (non-Mermaids cannot skip river)
	connected := gs.CheckForTownFormation("player1", hex1)
	
	// Should only find 2 buildings (hex1, hex2) on one side of the river
	// Not enough for a town
	if connected != nil {
		// If connected is not nil, it should only have 2 buildings
		if len(connected) >= 4 {
			t.Error("non-Mermaids should not be able to skip river to form town")
		}
	}
}

func TestMermaids_RiverSkippingWithBridge(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	
	// Set up: Buildings with both a bridge and potential river skip
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	river1 := NewHex(0, 1)
	hex3 := NewHex(0, 2)
	river2 := NewHex(1, 1)
	hex4 := NewHex(1, 2)
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = models.TerrainLake
	gs.Map.GetHex(hex2).Terrain = models.TerrainLake
	gs.Map.GetHex(river1).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = models.TerrainLake
	gs.Map.GetHex(river2).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex4).Terrain = models.TerrainLake
	
	// Place buildings
	for _, hex := range []Hex{hex1, hex2, hex3, hex4} {
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    models.FactionMermaids,
			PlayerID:   "player1",
			PowerValue: 2,
		})
	}
	
	// Build a bridge between hex2 and hex4 (across river2)
	gs.Map.BuildBridge(hex2, hex4)
	
	// Check for town formation
	connected := gs.CheckForTownFormation("player1", hex1)
	if connected == nil {
		t.Fatal("expected town to be formable with bridge + river skip")
	}
	
	// Should connect all 4 buildings (bridge connects hex2-hex4, river skip connects hex1-hex3)
	if len(connected) != 4 {
		t.Errorf("expected 4 connected buildings, got %d", len(connected))
	}
	
	// Verify pending town
	pending := gs.PendingTownFormations["player1"]
	if pending == nil {
		t.Fatal("expected pending town formation")
	}
	
	// Should have skipped one river
	if pending.SkippedRiverHex == nil {
		t.Error("expected a river to be skipped")
	}
}

func TestMermaids_StrongholdGrantsFreeShipping(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Mermaids start at shipping level 1
	if faction.GetShippingLevel() != 1 {
		t.Errorf("expected starting shipping level 1, got %d", faction.GetShippingLevel())
	}
	
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
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainLake)
	
	// Upgrade to stronghold (this automatically calls faction.BuildStronghold() and increases shipping)
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}
	
	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("Mermaids should have stronghold ability")
	}
	
	// Verify shipping level was increased (1 → 2)
	if faction.GetShippingLevel() != 2 {
		t.Errorf("expected shipping level 2 after stronghold, got %d", faction.GetShippingLevel())
	}
	
	// Verify the shipping was only granted once (calling BuildStronghold again should return false)
	shouldGrantShipping := faction.BuildStronghold()
	if shouldGrantShipping {
		t.Error("should only grant free shipping once")
	}
}
