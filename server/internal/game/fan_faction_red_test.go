package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func newRedFactionTestMap() *board.TerraMysticaMap {
	hexes := map[board.Hex]*board.MapHex{}
	rivers := map[board.Hex]bool{}
	setHex := func(hex board.Hex, terrain models.TerrainType) {
		hexes[hex] = &board.MapHex{Coord: hex, Terrain: terrain}
		if terrain == models.TerrainRiver {
			rivers[hex] = true
		}
	}

	for _, land := range []board.Hex{
		board.NewHex(-1, 0),
		board.NewHex(0, 0),
		board.NewHex(1, -2),
		board.NewHex(2, -2),
		board.NewHex(2, 0),
		board.NewHex(1, 1),
	} {
		setHex(land, models.TerrainPlains)
	}
	for _, river := range []board.Hex{
		board.NewHex(0, -1),
		board.NewHex(1, -1),
		board.NewHex(1, 0),
		board.NewHex(0, 1),
	} {
		setHex(river, models.TerrainRiver)
	}

	return &board.TerraMysticaMap{
		Hexes:      hexes,
		Bridges:    make(map[board.BridgeKey]string),
		RiverHexes: rivers,
	}
}

func TestArchitectsBridgeActionCostsPriest(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 1
	player.Resources.Workers = 0
	gs.Map.GetHex(board.NewHex(0, 0)).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(board.NewHex(1, -2)).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(board.NewHex(0, 0)).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(board.NewHex(1, -2)).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)

	if err := NewEngineersBridgeAction("p1", board.NewHex(0, 0), board.NewHex(1, -2)).Execute(gs); err != nil {
		t.Fatalf("bridge action failed: %v", err)
	}

	if got := player.Resources.Priests; got != 0 {
		t.Fatalf("priests after bridge action = %d, want 0", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after bridge action = %d, want 0", got)
	}
	if !gs.Map.HasBridge(board.NewHex(0, 0), board.NewHex(1, -2)) {
		t.Fatalf("expected bridge to be placed")
	}
}

func TestArchitectsBridgeCountsAsTownPower(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	hexA := board.NewHex(-1, 0)
	hexB := board.NewHex(0, 0)
	hexC := board.NewHex(1, -2)
	hexD := board.NewHex(2, -2)
	for _, hex := range []board.Hex{hexA, hexB, hexC, hexD} {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	}
	gs.Map.GetHex(hexA).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(hexB).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse)
	gs.Map.GetHex(hexC).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse)
	gs.Map.GetHex(hexD).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	if err := gs.Map.BuildBridge(hexB, hexC, "p1"); err != nil {
		t.Fatalf("BuildBridge failed: %v", err)
	}

	if !gs.CanFormTown("p1", []board.Hex{hexA, hexB, hexC, hexD}) {
		t.Fatalf("expected Architects bridge to contribute the seventh town power")
	}
}

func TestArchitectsTransformBuildUsesBridgeDiscount(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	origin := board.NewHex(0, 0)
	target := board.NewHex(1, -2)
	gs.Map.GetHex(origin).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(origin).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(target).Terrain = models.TerrainDesert
	if err := gs.Map.BuildBridge(origin, target, "p1"); err != nil {
		t.Fatalf("BuildBridge failed: %v", err)
	}

	startWorkers := player.Resources.Workers
	startCoins := player.Resources.Coins
	if err := NewTransformAndBuildAction("p1", target, true, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("TransformAndBuildAction.Execute failed: %v", err)
	}

	if got := player.Resources.Workers; got != startWorkers-1 {
		t.Fatalf("workers after bridge-discount transform/build = %d, want %d", got, startWorkers-1)
	}
	if got := player.Resources.Coins; got != startCoins-2 {
		t.Fatalf("coins after bridge-discount transform/build = %d, want %d", got, startCoins-2)
	}
	if gs.Map.GetHex(target).Terrain != player.Faction.GetHomeTerrain() {
		t.Fatalf("expected target terrain to transform to home terrain")
	}
}

func TestArchitectsStrongholdMoveBridgeGrantsVP(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.VictoryPoints = 0
	for _, hex := range []board.Hex{board.NewHex(0, 0), board.NewHex(1, -2)} {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
		gs.Map.GetHex(hex).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	}
	if err := gs.Map.BuildBridge(board.NewHex(0, 0), board.NewHex(1, -2), "p1"); err != nil {
		t.Fatalf("BuildBridge failed: %v", err)
	}

	action := NewArchitectsMoveBridgeAction("p1", board.NewHex(0, 0), board.NewHex(1, -2), board.NewHex(0, 0), board.NewHex(1, 1))
	if err := action.Execute(gs); err != nil {
		t.Fatalf("Architects move bridge failed: %v", err)
	}

	if got := player.VictoryPoints; got != 3 {
		t.Fatalf("victory points after Architects bridge move = %d, want 3", got)
	}
	if gs.Map.HasBridge(board.NewHex(0, 0), board.NewHex(1, -2)) {
		t.Fatalf("old bridge should have been removed")
	}
	if !gs.Map.HasBridge(board.NewHex(0, 0), board.NewHex(1, 1)) {
		t.Fatalf("new bridge should have been placed")
	}
}

func TestArchitectsStrongholdMovedBridgeEnablesAdjacentBuild(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Coins = 10
	player.Resources.Workers = 10

	source := board.NewHex(0, 0)
	oldEndpoint := board.NewHex(1, -2)
	target := board.NewHex(1, 1)
	for _, hex := range []board.Hex{source, oldEndpoint, target} {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	}
	gs.Map.GetHex(source).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(oldEndpoint).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(target).Terrain = models.TerrainDesert
	if err := gs.Map.BuildBridge(source, oldEndpoint, "p1"); err != nil {
		t.Fatalf("BuildBridge failed: %v", err)
	}

	move := NewArchitectsMoveBridgeAction("p1", source, oldEndpoint, source, target)
	if err := move.Execute(gs); err != nil {
		t.Fatalf("Architects move bridge failed: %v", err)
	}
	if !gs.IsAdjacentToPlayerBuilding(target, "p1") {
		t.Fatalf("moved bridge should make empty endpoint adjacent to Architects structure")
	}
	if err := NewTransformAndBuildAction("p1", target, true, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("TransformAndBuildAction.Execute failed: %v", err)
	}
	if got := gs.Map.GetHex(target).Building; got == nil || got.PlayerID != "p1" {
		t.Fatalf("expected Architects dwelling on moved bridge endpoint")
	}
}

func TestArchitectsStrongholdMoveBridgePreservesExistingTownMarker(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	if err := gs.AddPlayer("p1", factions.NewArchitects()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	hexA := board.NewHex(-1, 0)
	hexB := board.NewHex(0, 0)
	hexC := board.NewHex(1, -2)
	hexD := board.NewHex(2, -2)
	for _, hex := range []board.Hex{hexA, hexB, hexC, hexD, board.NewHex(1, 1)} {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	}
	gs.Map.GetHex(hexA).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(hexB).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse)
	gs.Map.GetHex(hexC).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse)
	gs.Map.GetHex(hexD).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	if err := gs.Map.BuildBridge(hexB, hexC, "p1"); err != nil {
		t.Fatalf("BuildBridge failed: %v", err)
	}
	if err := gs.FormTownWithAnchor("p1", []board.Hex{hexA, hexB, hexC, hexD}, models.TownTile5Points, nil, &hexB); err != nil {
		t.Fatalf("FormTownWithAnchor failed: %v", err)
	}
	if !gs.Map.GetHex(hexB).HasTownTile {
		t.Fatalf("expected formed town to record its chosen anchor hex")
	}

	action := NewArchitectsMoveBridgeAction("p1", hexB, hexC, hexB, board.NewHex(1, 1))
	if err := action.Execute(gs); err != nil {
		t.Fatalf("Architects bridge move should not invalidate permanent town marker: %v", err)
	}
	if !gs.Map.GetHex(hexB).HasTownTile {
		t.Fatalf("town marker should remain after bridge move")
	}
	if !gs.Map.GetHex(hexC).PartOfTown {
		t.Fatalf("existing town membership should remain after bridge move")
	}
}

func TestTreasurersIncomeCanBeBankedAndTreasuryDoublesNextRound(t *testing.T) {
	gs := NewGameState()
	gs.Map = newRedFactionTestMap()
	gs.Phase = PhaseIncome
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.TreasuryCoins = 1
	player.TreasuryWorkers = 1
	player.TreasuryPriests = 1
	homeHex := board.NewHex(0, 0)
	templeHex := board.NewHex(-1, 0)
	gs.Map.GetHex(homeHex).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(templeHex).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(homeHex).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)
	gs.Map.GetHex(templeHex).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingTemple)

	gs.GrantIncome()

	if got := player.Resources.Coins; got != 17 {
		t.Fatalf("coins after treasury release and income = %d, want 17", got)
	}
	if got := player.Resources.Workers; got != 7 {
		t.Fatalf("workers after treasury release and income = %d, want 7", got)
	}
	if got := player.Resources.Priests; got != 3 {
		t.Fatalf("priests after treasury release and income = %d, want 3", got)
	}
	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers income deposit")
	}
	if got := gs.PendingTreasurersDeposit.AvailableCoins; got != 2 {
		t.Fatalf("available coin income to bank = %d, want 2", got)
	}
	if got := gs.PendingTreasurersDeposit.AvailableWorkers; got != 3 {
		t.Fatalf("available worker income to bank = %d, want 3", got)
	}
	if got := gs.PendingTreasurersDeposit.AvailablePriests; got != 3 {
		t.Fatalf("available priest income to bank = %d, want 3", got)
	}

	if err := NewSelectTreasurersDepositAction("p1", 0, 1, 1).Execute(gs); err != nil {
		t.Fatalf("SelectTreasurersDepositAction.Execute failed: %v", err)
	}

	if got := player.TreasuryWorkers; got != 1 {
		t.Fatalf("treasury workers after banking income = %d, want 1", got)
	}
	if got := player.TreasuryPriests; got != 1 {
		t.Fatalf("treasury priests after banking income = %d, want 1", got)
	}
	if gs.Phase != PhaseAction {
		t.Fatalf("phase after resolving income deposit = %v, want action phase", gs.Phase)
	}
}

func TestTreasurersStrongholdPromptsForActionResourceDeposit(t *testing.T) {
	manager := NewManager()
	gs := NewGameState()
	gs.TurnOrder = []string{"p1"}
	gs.Phase = PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	manager.CreateGameWithState("g1", gs)

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(0, 8, 0)

	if _, err := manager.ExecuteActionWithMeta("g1", NewPowerAction("p1", PowerActionWorkers), ActionMeta{ExpectedRevision: -1}); err != nil {
		t.Fatalf("ExecuteActionWithMeta failed: %v", err)
	}

	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers action deposit")
	}
	if got := gs.PendingTreasurersDeposit.AvailableWorkers; got != 2 {
		t.Fatalf("available workers to bank = %d, want 2", got)
	}
}

func TestTreasurersStrongholdConversionPromptsForTreasuryDeposit(t *testing.T) {
	manager := NewManager()
	gs := NewGameState()
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	manager.CreateGameWithState("g1", gs)

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Priests = 1
	player.Resources.Workers = 0

	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "p1"},
		ConversionType: ConversionPriestToWorker,
		Amount:         1,
	}
	if _, err := manager.ExecuteActionWithMeta("g1", action, ActionMeta{ExpectedRevision: -1}); err != nil {
		t.Fatalf("ExecuteActionWithMeta failed: %v", err)
	}

	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers conversion deposit")
	}
	if got := gs.PendingTreasurersDeposit.AvailableWorkers; got != 1 {
		t.Fatalf("available workers to bank after conversion = %d, want 1", got)
	}
	if got := gs.PendingTreasurersDeposit.Reason; got != "conversion" {
		t.Fatalf("deposit reason = %q, want conversion", got)
	}
}

func TestTreasurersStrongholdDoublesNonPowerCultRewards(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Workers = 0
	player.Resources.Coins = 0
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 6
	player.CultPositions[CultFire] = 6

	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringStrongholdFire, CultTrack: CultFire, CultThreshold: 2, CultRewardType: CultRewardWorker, CultRewardAmount: 1},
			{Type: ScoringTemplePriest, CultRewardAmount: 2},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
		},
		PriestsSent: map[string]int{"p1": 2},
	}

	gs.AwardCultRewardsForRound(1)
	if got := player.Resources.Workers; got != 6 {
		t.Fatalf("workers after doubled cult rewards = %d, want 6", got)
	}

	gs.AwardCultRewardsForRound(2)
	if got := player.Resources.Coins; got != 8 {
		t.Fatalf("coins after doubled temple-priest cult rewards = %d, want 8", got)
	}
}

func TestTreasurersCultRewardQueuesIncomeDeposit(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true

	gs.grantCultReward("p1", player, CultRewardWorker, 2)

	if gs.PendingTreasurersDeposit == nil {
		t.Fatalf("expected pending Treasurers deposit after cult reward")
	}
	if got := gs.PendingTreasurersDeposit.AvailableWorkers; got != 4 {
		t.Fatalf("available workers to bank after doubled cult reward = %d, want 4", got)
	}
	if got := gs.PendingTreasurersDeposit.Reason; got != "cult_reward" {
		t.Fatalf("deposit reason = %q, want cult_reward", got)
	}
}

func TestTreasurersCultRewardWithoutStrongholdDoesNotQueueDeposit(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")

	gs.grantCultReward("p1", player, CultRewardCoin, 4)

	if gs.PendingTreasurersDeposit != nil {
		t.Fatalf("did not expect pending Treasurers deposit without stronghold ability")
	}
	if got := player.Resources.Coins; got != 19 {
		t.Fatalf("coins after cult reward = %d, want 19", got)
	}
}

func TestTreasurersStrongholdDoesNotDoublePowerOrSpadeCultRewards(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(0, 8, 0)
	player.Resources.Priests = 0
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 4
	player.CultPositions[CultFire] = 4
	gs.CultTracks.PlayerPositions["p1"][CultWater] = 4
	player.CultPositions[CultWater] = 4

	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringDwellingFire, CultTrack: CultFire, CultThreshold: 4, CultRewardType: CultRewardPower, CultRewardAmount: 4},
			{Type: ScoringTradingHouseWater, CultTrack: CultWater, CultThreshold: 4, CultRewardType: CultRewardSpade, CultRewardAmount: 1},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
		},
		PriestsSent: map[string]int{},
	}

	gs.AwardCultRewardsForRound(1)
	if got := player.Resources.Power.Bowl3; got != 4 {
		t.Fatalf("power after cult rewards = %d, want 4", got)
	}

	gs.AwardCultRewardsForRound(2)
	if got := gs.PendingCultRewardSpades["p1"]; got != 1 {
		t.Fatalf("pending cult reward spades = %d, want 1", got)
	}
}

func TestTreasurersPriestsInTreasuryBlockPriestPowerActionAtCap(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Power.Bowl3 = 10
	player.Resources.Priests = 2
	player.TreasuryPriests = 1
	gs.CultTracks.InitializePlayer("p1")
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultWater] = 2

	if got := gs.GetTotalOwnedPriests("p1"); got != 7 {
		t.Fatalf("total owned priests = %d, want 7", got)
	}

	if err := NewPowerAction("p1", PowerActionPriest).Validate(gs); err == nil {
		t.Fatalf("expected priest power action to be blocked at total priest cap")
	}
}

func TestTreasurersPriestsInTreasuryBlockPriestTownTileAtCap(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTreasurers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 2
	player.TreasuryPriests = 1
	gs.CultTracks.InitializePlayer("p1")
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultWater] = 2

	hexes := setupConnectedBuildings(gs, "p1", player.Faction, 4, 7)
	gs.PendingTownFormations["p1"] = []*PendingTownFormation{{
		PlayerID: "p1",
		Hexes:    hexes,
	}}

	action := &SelectTownTileAction{
		BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "p1"},
		TileType:   models.TownTile9Points,
	}
	if err := action.Validate(gs); err == nil {
		t.Fatalf("expected priest town tile to be blocked at total priest cap")
	}
}
