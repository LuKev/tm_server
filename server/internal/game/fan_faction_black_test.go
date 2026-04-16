package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestGoblinsStartWithSingleTreasureToken(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewGoblins()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if got := gs.GetPlayer("p1").GoblinTreasureTokens; got != 1 {
		t.Fatalf("goblin treasure tokens = %d, want 1", got)
	}
}

func TestGoblinsTempleAndSanctuaryGrantTreasure(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewGoblins()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	hex := board.NewHex(0, 0)
	gs.Map.TransformTerrain(hex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(hex, testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))

	if err := NewUpgradeBuildingAction("p1", hex, models.BuildingTemple).Execute(gs); err != nil {
		t.Fatalf("upgrade to temple failed: %v", err)
	}
	if got := player.GoblinTreasureTokens; got != 2 {
		t.Fatalf("goblin treasure after temple = %d, want 2", got)
	}

	if err := NewUpgradeBuildingAction("p1", hex, models.BuildingSanctuary).Execute(gs); err != nil {
		t.Fatalf("upgrade to sanctuary failed: %v", err)
	}
	if got := player.GoblinTreasureTokens; got != 3 {
		t.Fatalf("goblin treasure after sanctuary = %d, want 3", got)
	}
}

func TestGoblinsTownWithStrongholdGrantsTreasure(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewGoblins()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true

	gs.ApplyFactionTownBonus("p1")

	if got := player.GoblinTreasureTokens; got != 2 {
		t.Fatalf("goblin treasure after town = %d, want 2", got)
	}
}

func TestGoblinsTreasureRewardsAndCultResolution(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewGoblins()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}

	player := gs.GetPlayer("p1")
	player.GoblinTreasureTokens = 4
	player.Resources.Coins = 0
	player.Resources.Workers = 0
	player.Resources.Power = NewPowerSystem(2, 0, 0)

	structures := map[board.Hex]models.BuildingType{
		board.NewHex(0, 0): models.BuildingDwelling,
		board.NewHex(1, 0): models.BuildingDwelling,
		board.NewHex(2, 0): models.BuildingTradingHouse,
		board.NewHex(3, 0): models.BuildingTemple,
		board.NewHex(4, 0): models.BuildingStronghold,
		board.NewHex(5, 0): models.BuildingSanctuary,
	}
	for hex, buildingType := range structures {
		gs.Map.TransformTerrain(hex, player.Faction.GetHomeTerrain())
		gs.Map.PlaceBuilding(hex, testBuilding("p1", player.Faction.GetType(), buildingType))
	}

	if err := NewUseGoblinsTreasureAction("p1", GoblinsTreasureDwellings).Execute(gs); err != nil {
		t.Fatalf("dwelling treasure reward failed: %v", err)
	}
	if got := player.Resources.Power.Bowl2; got != 2 {
		t.Fatalf("bowl II after dwelling reward = %d, want 2", got)
	}

	if err := NewUseGoblinsTreasureAction("p1", GoblinsTreasureTradingPosts).Execute(gs); err != nil {
		t.Fatalf("trading-post treasure reward failed: %v", err)
	}
	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins after trading-post reward = %d, want 2", got)
	}

	if err := NewUseGoblinsTreasureAction("p1", GoblinsTreasureTemples).Execute(gs); err != nil {
		t.Fatalf("temple treasure reward failed: %v", err)
	}
	if got := player.Resources.Workers; got != 1 {
		t.Fatalf("workers after temple reward = %d, want 1", got)
	}

	if err := NewUseGoblinsTreasureAction("p1", GoblinsTreasureBigStructures).Execute(gs); err != nil {
		t.Fatalf("big-structure treasure reward failed: %v", err)
	}
	if gs.PendingGoblinsCultSteps == nil || gs.PendingGoblinsCultSteps.StepsRemaining != 3 {
		t.Fatalf("pending goblins cult steps = %+v, want 3 remaining", gs.PendingGoblinsCultSteps)
	}

	for _, track := range []CultTrack{CultFire, CultWater, CultAir} {
		if err := NewSelectGoblinsCultTrackAction("p1", track).Execute(gs); err != nil {
			t.Fatalf("select goblins cult track %v failed: %v", track, err)
		}
	}

	if gs.PendingGoblinsCultSteps != nil {
		t.Fatalf("expected goblins cult steps to be fully resolved")
	}
	if got := player.CultPositions[CultFire]; got != 1 {
		t.Fatalf("fire cult = %d, want 1", got)
	}
	if got := player.CultPositions[CultWater]; got != 1 {
		t.Fatalf("water cult = %d, want 1", got)
	}
	if got := player.CultPositions[CultAir]; got != 2 {
		t.Fatalf("air cult = %d, want 2", got)
	}
	if got := player.CultPositions[CultEarth]; got != 1 {
		t.Fatalf("earth cult = %d, want 1", got)
	}
}

func TestChildrenOfTheWyrmStartWithTwelveCoins(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if got := gs.GetPlayer("p1").Resources.Coins; got != 12 {
		t.Fatalf("coins = %d, want 12", got)
	}
}

func TestChildrenOfTheWyrmBurnPowerUsesThreeForTwo(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Power = NewPowerSystem(0, 3, 0)

	action := &BurnPowerAction{
		BaseAction: BaseAction{Type: ActionBurnPower, PlayerID: "p1"},
		Amount:     1,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("BurnPowerAction.Execute failed: %v", err)
	}

	if player.Resources.Power.Bowl1 != 0 || player.Resources.Power.Bowl2 != 0 || player.Resources.Power.Bowl3 != 2 {
		t.Fatalf("unexpected children power state after burn: %d/%d/%d", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
	}
}

func TestChildrenOfTheWyrmStrongholdRestoresRemovedTokens(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Power = NewPowerSystem(0, 0, 9)

	action := &UpgradeBuildingAction{
		BaseAction: BaseAction{Type: ActionUpgradeBuilding, PlayerID: "p1"},
	}
	action.handleStrongholdBonuses(gs, player)

	if got := player.Resources.Power.Bowl1; got != 3 {
		t.Fatalf("bowl I after stronghold bonus = %d, want 3", got)
	}
}

func TestChildrenOfTheWyrmPowerTokensNeedBowl3Confirmation(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(0, 0, 1)

	sourceHex := board.NewHex(0, -1)
	riverHex := board.NewHex(0, 0)
	gs.Map.Hexes[sourceHex] = &board.MapHex{Coord: sourceHex, Terrain: player.Faction.GetHomeTerrain()}
	gs.Map.Hexes[riverHex] = &board.MapHex{Coord: riverHex, Terrain: models.TerrainRiver}
	gs.Map.RiverHexes[riverHex] = true
	gs.Map.PlaceBuilding(sourceHex, testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))

	withoutConfirm := NewChildrenPlacePowerTokensAction("p1", []board.Hex{riverHex}, false)
	if err := withoutConfirm.Validate(gs); err == nil {
		t.Fatalf("expected Children power-token action to require Bowl III confirmation")
	}

	withConfirm := NewChildrenPlacePowerTokensAction("p1", []board.Hex{riverHex}, true)
	if err := withConfirm.Execute(gs); err != nil {
		t.Fatalf("Children power-token action with confirmation failed: %v", err)
	}
	if got := gs.Map.GetHex(riverHex).PowerTokenOwnerPlayerID; got != "p1" {
		t.Fatalf("river token owner = %q, want p1", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl III after confirmed placement = %d, want 0", got)
	}
}

func TestChildrenOfTheWyrmRiverNetworkAffectsAdjacencyDiscountsAndTownConnectivity(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}
	gs.TurnOrder = []string{"p1", "p2"}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(2, 0, 0)

	sourceHex := board.NewHex(0, -1)
	riverHex1 := board.NewHex(0, 0)
	riverHex2 := board.NewHex(1, 0)
	targetHex := board.NewHex(2, -1)
	upgradeHex := board.NewHex(1, 1)
	opponentHex := board.NewHex(2, 0)

	customHexes := map[board.Hex]models.TerrainType{
		sourceHex:           player.Faction.GetHomeTerrain(),
		targetHex:           player.Faction.GetHomeTerrain(),
		upgradeHex:          player.Faction.GetHomeTerrain(),
		opponentHex:         models.TerrainForest,
		riverHex1:           models.TerrainRiver,
		riverHex2:           models.TerrainRiver,
		board.NewHex(1, -1): player.Faction.GetHomeTerrain(),
		board.NewHex(0, 1):  player.Faction.GetHomeTerrain(),
		board.NewHex(2, -2): player.Faction.GetHomeTerrain(),
	}
	for hex, terrain := range customHexes {
		gs.Map.Hexes[hex] = &board.MapHex{Coord: hex, Terrain: terrain}
		if terrain == models.TerrainRiver {
			gs.Map.RiverHexes[hex] = true
		}
	}

	gs.Map.PlaceBuilding(sourceHex, testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))
	gs.Map.PlaceBuilding(upgradeHex, testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))
	gs.Map.PlaceBuilding(opponentHex, testBuilding("p2", models.FactionWitches, models.BuildingDwelling))

	if gs.IsAdjacentToPlayerBuilding(targetHex, "p1") {
		t.Fatalf("expected target hex to be non-adjacent before placing river tokens")
	}

	if err := NewChildrenPlacePowerTokensAction("p1", []board.Hex{riverHex1, riverHex2}, false).Execute(gs); err != nil {
		t.Fatalf("Children power-token placement failed: %v", err)
	}

	if !gs.IsAdjacentToPlayerBuilding(targetHex, "p1") {
		t.Fatalf("expected target hex to become adjacent through the Children river network")
	}

	if got := getUpgradeCost(gs, player, gs.Map.GetHex(upgradeHex), models.BuildingStronghold).Coins; got != 5 {
		t.Fatalf("children stronghold upgrade coins = %d, want 5", got)
	}

	connectedHexes := []board.Hex{
		sourceHex,
		board.NewHex(1, -1),
		board.NewHex(0, 1),
		upgradeHex,
		board.NewHex(2, -2),
	}
	gs.Map.PlaceBuilding(board.NewHex(1, -1), testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))
	gs.Map.PlaceBuilding(board.NewHex(0, 1), testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))
	gs.Map.PlaceBuilding(board.NewHex(2, -2), testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))

	connected := gs.getConnectedBuildingsForPlayer("p1", sourceHex)
	if got := len(connected); got != len(connectedHexes) {
		t.Fatalf("connected Children buildings = %d, want %d", got, len(connectedHexes))
	}
	if !gs.CanFormTown("p1", connected) {
		t.Fatalf("expected Children river-network component to satisfy town requirements")
	}
}
