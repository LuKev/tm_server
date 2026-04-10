package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestSetupFlow_AtlanteansPlacementOrderRelativeToChaos(t *testing.T) {
	tests := []struct {
		name      string
		turnOrder []string
		want      []string
	}{
		{
			name:      "atlanteans_before_chaos",
			turnOrder: []string{"atl", "chaos", "witches"},
			want:      []string{"witches", "witches", "atl", "chaos"},
		},
		{
			name:      "atlanteans_after_chaos",
			turnOrder: []string{"chaos", "atl", "witches"},
			want:      []string{"witches", "witches", "chaos", "atl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGameState()
			if err := gs.AddPlayer("atl", factions.NewAtlanteans()); err != nil {
				t.Fatalf("add atl: %v", err)
			}
			if err := gs.AddPlayer("chaos", factions.NewChaosMagicians()); err != nil {
				t.Fatalf("add chaos: %v", err)
			}
			if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
				t.Fatalf("add witches: %v", err)
			}

			gs.TurnOrder = tt.turnOrder
			gs.InitializeSetupSequence()

			if len(gs.SetupDwellingOrder) != len(tt.want) {
				t.Fatalf("order length = %d, want %d", len(gs.SetupDwellingOrder), len(tt.want))
			}
			for i, playerID := range tt.want {
				if gs.SetupDwellingOrder[i] != playerID {
					t.Fatalf("order[%d] = %s, want %s", i, gs.SetupDwellingOrder[i], playerID)
				}
			}
		})
	}
}

func TestAtlanteansSetupPlacesStrongholdAndPendingTownTile(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("atl", factions.NewAtlanteans()); err != nil {
		t.Fatalf("add atl: %v", err)
	}
	if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
		t.Fatalf("add witches: %v", err)
	}

	gs.TurnOrder = []string{"atl", "witches"}
	gs.InitializeSetupSequence()

	witch1 := board.NewHex(0, 0)
	witch2 := board.NewHex(1, 0)
	atlHex := board.NewHex(0, 1)
	gs.Map.TransformTerrain(witch1, models.TerrainForest)
	gs.Map.TransformTerrain(witch2, models.TerrainForest)
	gs.Map.TransformTerrain(atlHex, models.TerrainLake)

	if err := NewSetupDwellingAction("witches", witch1).Execute(gs); err != nil {
		t.Fatalf("first witches setup failed: %v", err)
	}
	if err := NewSetupDwellingAction("witches", witch2).Execute(gs); err != nil {
		t.Fatalf("second witches setup failed: %v", err)
	}
	if err := NewSetupDwellingAction("atl", atlHex).Execute(gs); err != nil {
		t.Fatalf("atlanteans setup failed: %v", err)
	}

	mapHex := gs.Map.GetHex(atlHex)
	if mapHex == nil || mapHex.Building == nil || mapHex.Building.Type != models.BuildingStronghold {
		t.Fatalf("expected Atlanteans setup structure to be a stronghold")
	}
	if pending := gs.PendingTownFormations["atl"]; len(pending) != 1 {
		t.Fatalf("expected one pending town formation, got %d", len(pending))
	}

	selectTown := &SelectTownTileAction{
		BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "atl"},
		TileType:   models.TownTile11Points,
	}
	if err := selectTown.Execute(gs); err != nil {
		t.Fatalf("select town tile failed: %v", err)
	}

	player := gs.GetPlayer("atl")
	if !gs.Map.GetHex(atlHex).PartOfTown {
		t.Fatalf("expected starting stronghold to be marked as part of town")
	}
	if got := player.TownsFormed; got != 1 {
		t.Fatalf("towns formed = %d, want 1", got)
	}
	if got := len(player.TownTiles); got != 1 || player.TownTiles[0] != models.TownTile11Points {
		t.Fatalf("unexpected town tiles: %v", player.TownTiles)
	}
	if got := len(player.AtlanteansTownHexes); got != 1 || player.AtlanteansTownHexes[0] != atlHex {
		t.Fatalf("unexpected Atlanteans town hexes: %v", player.AtlanteansTownHexes)
	}
}

func TestAtlanteansStrongholdTownRewardsTriggerOnce(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("atl", factions.NewAtlanteans()); err != nil {
		t.Fatalf("add atl: %v", err)
	}

	player := gs.GetPlayer("atl")
	player.VictoryPoints = 0
	player.AtlanteansTownRewards = make(map[int]bool)

	hexes := []board.Hex{
		board.NewHex(0, 0),
		board.NewHex(1, 0),
		board.NewHex(2, 0),
		board.NewHex(3, 0),
		board.NewHex(4, 0),
		board.NewHex(5, 0),
		board.NewHex(6, 0),
	}
	buildings := []models.BuildingType{
		models.BuildingStronghold,
		models.BuildingSanctuary,
		models.BuildingTemple,
		models.BuildingTradingHouse,
		models.BuildingTradingHouse,
		models.BuildingTradingHouse,
		models.BuildingTradingHouse,
	}

	for i, hex := range hexes {
		gs.Map.TransformTerrain(hex, player.Faction.GetHomeTerrain())
		powerValue := GetPowerValue(buildings[i])
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       buildings[i],
			Faction:    player.Faction.GetType(),
			PlayerID:   "atl",
			PowerValue: powerValue,
		})
	}
	gs.Map.GetHex(hexes[0]).PartOfTown = true
	player.AtlanteansTownHexes = []board.Hex{hexes[0]}

	gs.updateAtlanteansStrongholdTown("atl")

	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("shipping = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 22 {
		t.Fatalf("victory points = %d, want 22", got)
	}
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		if got := player.CultPositions[track]; got != 2 {
			t.Fatalf("cult %v = %d, want 2", track, got)
		}
	}

	gs.updateAtlanteansStrongholdTown("atl")
	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("shipping after second update = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 22 {
		t.Fatalf("victory points after second update = %d, want 22", got)
	}
}

func TestAtlanteansCanUseBridgeAction(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("atl", factions.NewAtlanteans()); err != nil {
		t.Fatalf("add atl: %v", err)
	}

	player := gs.GetPlayer("atl")
	player.Resources.Workers = 2

	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(1, -2)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: player.Faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: player.Faction.GetHomeTerrain()}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	gs.Map.PlaceBuilding(hex1, testBuilding("atl", player.Faction.GetType(), models.BuildingStronghold))

	action := NewEngineersBridgeAction("atl", hex1, hex2)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("bridge action failed: %v", err)
	}
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Fatalf("expected bridge to be built")
	}
}

func TestWispsTradingHouseCreatesAdjacentSingleSpadeFollowup(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWisps()); err != nil {
		t.Fatalf("add wisps: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20

	sourceHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	farHex := board.NewHex(3, 0)
	gs.Map.TransformTerrain(sourceHex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(sourceHex, testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))

	var oneSpadeTerrain models.TerrainType
	found := false
	for _, terrain := range []models.TerrainType{
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
		models.TerrainDesert,
	} {
		if gs.Map.GetTerrainDistance(terrain, player.Faction.GetHomeTerrain()) == 1 {
			oneSpadeTerrain = terrain
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("failed to find a one-spade terrain for Wisps")
	}
	gs.Map.TransformTerrain(targetHex, oneSpadeTerrain)
	gs.Map.TransformTerrain(farHex, oneSpadeTerrain)

	if err := NewUpgradeBuildingAction("p1", sourceHex, models.BuildingTradingHouse).Execute(gs); err != nil {
		t.Fatalf("upgrade to trading house failed: %v", err)
	}
	if got := gs.PendingSpades["p1"]; got != 1 {
		t.Fatalf("pending spades = %d, want 1", got)
	}
	if allowed := gs.PendingSpadeBuildAllowed["p1"]; allowed {
		t.Fatalf("expected dwelling builds to be disallowed on Wisps pending spade")
	}

	invalid := NewTransformAndBuildAction("p1", farHex, false, models.TerrainTypeUnknown)
	if err := invalid.Validate(gs); err == nil {
		t.Fatalf("expected non-adjacent Wisps spade target to be rejected")
	}

	valid := NewTransformAndBuildAction("p1", targetHex, false, models.TerrainTypeUnknown)
	if err := valid.Execute(gs); err != nil {
		t.Fatalf("valid Wisps spade follow-up failed: %v", err)
	}
	if got := gs.Map.GetHex(targetHex).Terrain; got != player.Faction.GetHomeTerrain() {
		t.Fatalf("target terrain = %v, want %v", got, player.Faction.GetHomeTerrain())
	}
	if got := gs.PendingSpades["p1"]; got != 0 {
		t.Fatalf("pending spades after use = %d, want 0", got)
	}
}

func TestWispsStrongholdGivesVPAndFreeLakeDwelling(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWisps()); err != nil {
		t.Fatalf("add wisps: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.VictoryPoints = 0
	lakeHex := board.NewHex(0, 0)
	gs.Map.TransformTerrain(lakeHex, models.TerrainLake)

	action := &UpgradeBuildingAction{
		BaseAction:      BaseAction{Type: ActionUpgradeBuilding, PlayerID: "p1"},
		NewBuildingType: models.BuildingStronghold,
	}
	action.handleStrongholdBonuses(gs, player)

	if got := player.VictoryPoints; got != 7 {
		t.Fatalf("victory points = %d, want 7", got)
	}
	if gs.PendingWispsStrongholdDwelling == nil {
		t.Fatalf("expected pending Wisps stronghold dwelling")
	}

	build := NewBuildWispsStrongholdDwellingAction("p1", lakeHex)
	if err := build.Execute(gs); err != nil {
		t.Fatalf("build free Wisps dwelling failed: %v", err)
	}
	if gs.PendingWispsStrongholdDwelling != nil {
		t.Fatalf("expected pending Wisps dwelling to clear")
	}
	if got := gs.Map.GetHex(lakeHex).Building; got == nil || got.Type != models.BuildingDwelling {
		t.Fatalf("expected free Wisps dwelling on lake")
	}
}
