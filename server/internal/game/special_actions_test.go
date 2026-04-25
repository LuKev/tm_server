package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Helper function to build a stronghold for a player
func buildStrongholdForPlayer(gs *GameState, playerID string, hex board.Hex) {
	player := gs.GetPlayer(playerID)
	mapHex := gs.Map.GetHex(hex)

	mapHex.Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    player.Faction.GetType(),
		PlayerID:   playerID,
		PowerValue: 3,
	}

	player.HasStrongholdAbility = true
}

func TestAurenCultAdvance_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold to enable special ability
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set initial cult position
	player.CultPositions[CultFire] = 3

	// Use Auren cult advance special action
	action := NewAurenCultAdvanceAction("player1", CultFire)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Auren cult advance to succeed, got error: %v", err)
	}

	// Verify cult position advanced by 2
	if player.CultPositions[CultFire] != 5 {
		t.Errorf("expected Fire cult position to be 5, got %d", player.CultPositions[CultFire])
	}

	// Verify ability is marked as used
	if !player.SpecialActionsUsed[SpecialActionAurenCultAdvance] {
		t.Error("expected Auren cult advance to be marked as used")
	}
}

func TestAurenCultAdvance_MaxPosition(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set cult position near max
	player.CultPositions[CultWater] = 9

	// Give player a key to reach position 10
	player.Keys = 1

	// Use Auren cult advance special action
	action := NewAurenCultAdvanceAction("player1", CultWater)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Auren cult advance to succeed, got error: %v", err)
	}

	// Verify cult position capped at 10
	if player.CultPositions[CultWater] != 10 {
		t.Errorf("expected Water cult position to be capped at 10, got %d", player.CultPositions[CultWater])
	}
}

func TestAurenCultAdvance_AlreadyAtMax(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set cult position at max
	player.CultPositions[CultEarth] = 10

	// Try to use Auren cult advance special action
	action := NewAurenCultAdvanceAction("player1", CultEarth)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when already at max cult position")
	}
}

func TestAurenCultAdvance_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set cult positions
	player.CultPositions[CultFire] = 3
	player.CultPositions[CultWater] = 3

	// Use Auren cult advance on Fire
	action1 := NewAurenCultAdvanceAction("player1", CultFire)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected first cult advance to succeed, got error: %v", err)
	}

	// Try to use it again on Water - should fail
	action2 := NewAurenCultAdvanceAction("player1", CultWater)
	err = action2.Execute(gs)
	if err == nil {
		t.Fatal("expected error when using stronghold ability twice in one round")
	}
}

func TestAurenCultAdvance_ResetBetweenRounds(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set cult positions
	player.CultPositions[CultFire] = 3
	player.CultPositions[CultWater] = 5

	// Use ability in round 1
	action1 := NewAurenCultAdvanceAction("player1", CultFire)
	action1.Execute(gs)

	// Start new round
	gs.StartNewRound()

	// Should be able to use ability again
	action2 := NewAurenCultAdvanceAction("player1", CultWater)
	err := action2.Execute(gs)
	if err != nil {
		t.Fatalf("expected cult advance to succeed in new round, got error: %v", err)
	}

	// Verify Water position advanced
	if player.CultPositions[CultWater] != 7 {
		t.Errorf("expected Water cult position to be 7, got %d", player.CultPositions[CultWater])
	}
}

func TestAurenCultAdvance_WithoutStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.CultPositions[CultFire] = 3

	// Try to use ability without stronghold
	action := NewAurenCultAdvanceAction("player1", CultFire)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when using ability without stronghold")
	}
}

func TestWitchesRide_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Set target hex to Forest
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	// Use Witches' Ride
	action := NewWitchesRideAction("player1", targetHex)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Witches' Ride to succeed, got error: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PlayerID != "player1" {
		t.Errorf("expected building to belong to player1, got %s", mapHex.Building.PlayerID)
	}

	// Verify ability is marked as used
	if !player.SpecialActionsUsed[SpecialActionWitchesRide] {
		t.Error("expected Witches' Ride to be marked as used")
	}
}

func TestWitchesRide_IgnoresAdjacency(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	// Build stronghold at one location
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Target hex far away from any player buildings
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	// Witches' Ride should succeed despite no adjacency
	action := NewWitchesRideAction("player1", targetHex)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Witches' Ride to succeed (ignoring adjacency), got error: %v", err)
	}

	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
}

func TestWitchesRide_OnlyOnForest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Try to use Witches' Ride on non-Forest hex
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainPlains)

	action := NewWitchesRideAction("player1", targetHex)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when using Witches' Ride on non-Forest hex")
	}
}

func TestWitchesRide_BuildingLimitEnforced(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Place 8 dwellings (the limit)
	for i := 0; i < 8; i++ {
		hex := board.NewHex(i, 2)
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		}
	}

	// Try to use Witches' Ride to build 9th dwelling
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	action := NewWitchesRideAction("player1", targetHex)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when building limit reached")
	}
}

func TestWitchesRide_PowerLeech(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewWitches()
	faction2 := factions.NewCultists()
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)

	// Build stronghold for Witches
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Place player2's dwelling adjacent to target
	player2Hex := board.NewHex(5, 4)
	gs.Map.GetHex(player2Hex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	}

	// Use Witches' Ride on adjacent Forest hex
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	action := NewWitchesRideAction("player1", targetHex)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Witches' Ride to succeed, got error: %v", err)
	}

	// Verify player2 has a pending leech offer
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) == 0 {
		t.Fatal("expected player2 to have a pending leech offer")
	}

	offer := offers[0]
	if offer.Amount != 1 {
		t.Errorf("expected offer amount of 1, got %d", offer.Amount)
	}
}

func TestWitchesRide_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Use Witches' Ride once
	targetHex1 := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex1, models.TerrainForest)

	action1 := NewWitchesRideAction("player1", targetHex1)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected first Witches' Ride to succeed, got error: %v", err)
	}

	// Try to use it again - should fail
	targetHex2 := board.NewHex(6, 6)
	gs.Map.TransformTerrain(targetHex2, models.TerrainForest)

	action2 := NewWitchesRideAction("player1", targetHex2)
	err = action2.Execute(gs)
	if err == nil {
		t.Fatal("expected error when using Witches' Ride twice in one round")
	}
}

// SWARMLINGS TESTS

func TestSwarmlingsUpgrade_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Build stronghold to enable special ability
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Place a dwelling
	dwellingHex := board.NewHex(1, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Use Swarmlings upgrade special action
	action := NewSwarmlingsUpgradeAction("player1", dwellingHex)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Swarmlings upgrade to succeed, got error: %v", err)
	}

	// Verify dwelling was upgraded to trading house
	mapHex := gs.Map.GetHex(dwellingHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 2 {
		t.Errorf("expected power value 2, got %d", mapHex.Building.PowerValue)
	}

	// Verify ability is marked as used
	if !player.SpecialActionsUsed[SpecialActionSwarmlingsUpgrade] {
		t.Error("expected Swarmlings upgrade to be marked as used")
	}
}

func TestSwarmlingsUpgrade_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Place two dwellings
	dwelling1 := board.NewHex(1, 1)
	gs.Map.GetHex(dwelling1).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	dwelling2 := board.NewHex(2, 1)
	gs.Map.GetHex(dwelling2).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Upgrade first dwelling
	action1 := NewSwarmlingsUpgradeAction("player1", dwelling1)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected first upgrade to succeed, got error: %v", err)
	}

	// Try to upgrade second dwelling - should fail
	action2 := NewSwarmlingsUpgradeAction("player1", dwelling2)
	err = action2.Execute(gs)
	if err == nil {
		t.Fatal("expected error when using ability twice in one round")
	}
}

func TestSwarmlingsUpgrade_BuildingLimitEnforced(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Place 4 trading houses (the limit)
	for i := 0; i < 4; i++ {
		hex := board.NewHex(i, 2)
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		}
	}

	// Try to upgrade a 5th dwelling
	dwellingHex := board.NewHex(5, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	action := NewSwarmlingsUpgradeAction("player1", dwellingHex)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when trading house limit reached")
	}
}

// GIANTS TESTS

func TestGiantsTransform_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Target hex adjacent to stronghold
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)

	// Use Giants transform special action
	action := NewGiantsTransformAction("player1", targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Giants transform to succeed, got error: %v", err)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainWasteland {
		t.Errorf("expected Wasteland terrain, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestGiantsTransform_TransformOnly(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Target hex adjacent to stronghold
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	// Use Giants transform without building
	action := NewGiantsTransformAction("player1", targetHex, false)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Giants transform to succeed, got error: %v", err)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainWasteland {
		t.Errorf("expected Wasteland terrain, got %v", mapHex.Terrain)
	}

	// Verify no building was built
	if mapHex.Building != nil {
		t.Error("expected no building to be built")
	}
}

// NOMADS TESTS

func TestNomadsSandstorm_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewNomads()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Target hex directly adjacent to stronghold
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainSwamp)

	// Use Nomads sandstorm special action
	action := NewNomadsSandstormAction("player1", targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Nomads sandstorm to succeed, got error: %v", err)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainDesert {
		t.Errorf("expected Desert terrain, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestNomadsSandstorm_RequiresDirectAdjacency(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewNomads()
	gs.AddPlayer("player1", faction)

	// Build stronghold
	strongholdHex := board.NewHex(0, 1)
	buildStrongholdForPlayer(gs, "player1", strongholdHex)

	// Target hex NOT adjacent to stronghold
	targetHex := board.NewHex(5, 5)
	gs.Map.TransformTerrain(targetHex, models.TerrainSwamp)

	// Try to use sandstorm - should fail
	action := NewNomadsSandstormAction("player1", targetHex, true)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when target is not directly adjacent")
	}
}

func TestSelkiesStronghold_AllowsShippingPlusOne(t *testing.T) {
	gs, err := NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("new game state: %v", err)
	}
	faction := factions.NewSelkies()
	if err := gs.AddPlayer("player1", faction); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	player.HasStrongholdAbility = true
	player.StartingTerrain = models.TerrainPlains
	player.HasStartingTerrain = true

	origin, ok := board.HexForDisplayCoordinate(board.MapFjords, "E6")
	if !ok {
		t.Fatal("expected E6 to exist on Fjords")
	}
	startDwelling, ok := board.HexForDisplayCoordinate(board.MapFjords, "B2")
	if !ok {
		t.Fatal("expected B2 to exist on Fjords")
	}
	targetHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "A3")
	if !ok {
		t.Fatal("expected A3 to exist on Fjords")
	}

	gs.Map.GetHex(origin).Terrain = models.TerrainIce
	gs.Map.GetHex(startDwelling).Terrain = models.TerrainIce
	gs.Map.PlaceBuilding(startDwelling, &models.Building{Type: models.BuildingDwelling, PlayerID: "player1", Faction: faction.GetType(), PowerValue: 1})
	gs.Map.PlaceBuilding(origin, &models.Building{Type: models.BuildingStronghold, PlayerID: "player1", Faction: faction.GetType(), PowerValue: 3})

	action := NewSelkiesStrongholdAction("player1", targetHex, true, models.TerrainTypeUnknown)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("expected Selkies stronghold to succeed, got error: %v", err)
	}

	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.PlayerID != "player1" {
		t.Fatalf("expected dwelling to be built, got %+v", mapHex.Building)
	}
}

func TestSelkiesStronghold_TransformOnlyUsesProvidedTerrainAndFreeSpade(t *testing.T) {
	gs, err := NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("new game state: %v", err)
	}
	faction := factions.NewSelkies()
	if err := gs.AddPlayer("player1", faction); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 0
	player.Resources.Priests = 1
	player.HasStrongholdAbility = true
	player.StartingTerrain = models.TerrainPlains
	player.HasStartingTerrain = true

	targetHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "D6")
	if !ok {
		t.Fatal("expected D6 to exist on Fjords")
	}
	var supportHex board.Hex
	foundSupport := false
	for _, neighbor := range gs.Map.GetDirectNeighbors(targetHex) {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain == models.TerrainRiver {
			continue
		}
		supportHex = neighbor
		foundSupport = true
		break
	}
	if !foundSupport {
		t.Fatal("expected a non-river support hex adjacent to D6")
	}

	gs.Map.GetHex(supportHex).Terrain = models.TerrainIce
	gs.Map.PlaceBuilding(supportHex, &models.Building{Type: models.BuildingStronghold, PlayerID: "player1", Faction: faction.GetType(), PowerValue: 3})

	action := NewSelkiesStrongholdAction("player1", targetHex, false, models.TerrainDesert)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("expected Selkies transform-only stronghold action to succeed, got error: %v", err)
	}

	if got := gs.Map.GetHex(targetHex).Terrain; got != models.TerrainDesert {
		t.Fatalf("target terrain = %v, want %v", got, models.TerrainDesert)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers = %d, want 0 after free one-spade transform", got)
	}
	if got := player.Resources.Coins; got != 10 {
		t.Fatalf("coins = %d, want 10 for transform-only stronghold action", got)
	}
}

func TestUpgradeToStronghold_GrantsAbility(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house
	tradingHouseHex := board.NewHex(0, 1)
	gs.Map.GetHex(tradingHouseHex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}

	// Verify player doesn't have stronghold ability yet
	if player.HasStrongholdAbility {
		t.Error("expected player to not have stronghold ability before building")
	}

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}

	// Verify player now has stronghold ability
	if !player.HasStrongholdAbility {
		t.Error("expected player to have stronghold ability after building stronghold")
	}

	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}
}

// Regression test for Bug #4: BON1 bonus card spade action
// The bug was that BON1 was being converted to a regular TransformAndBuildAction
// instead of a SpecialAction, causing it to consume all workers instead of providing
// 1 free spade.
func TestBonusCardSpade_ProvidesFreeSpade(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade

	// Set up a hex that needs 1 spade to transform (distance 1)
	targetHex := board.NewHex(0, 1)
	mapHex := gs.Map.GetHex(targetHex)
	mapHex.Terrain = models.TerrainMountain // Witches are Forest, Mountain->Forest = 1 step

	// Place a dwelling nearby for adjacency
	adjacentHex := board.NewHex(0, 0)
	adjacentMapHex := gs.Map.GetHex(adjacentHex)
	adjacentMapHex.Terrain = models.TerrainForest
	adjacentMapHex.Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionWitches,
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Player starts with 3 workers (Witches starting resources)
	initialWorkers := player.Resources.Workers

	// Use bonus card spade action to transform and build
	action := NewBonusCardSpadeAction("player1", targetHex, true, models.TerrainTypeUnknown)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected bonus card spade action to succeed, got error: %v", err)
	}

	// Verify terrain was transformed
	if mapHex.Terrain != models.TerrainForest {
		t.Errorf("expected terrain to be Forest, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("expected dwelling to be built")
	}

	// Verify workers consumed = only dwelling cost (1 worker)
	// Mountain->Forest is 1 step (1 spade needed)
	// BON1 provides 1 free spade, so transformation is free
	// Only dwelling cost (1 worker) should be charged
	expectedWorkers := initialWorkers - 1
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers remaining (spent 1 for dwelling), got %d workers (spent %d)",
			expectedWorkers, player.Resources.Workers, initialWorkers-player.Resources.Workers)
	}
}
