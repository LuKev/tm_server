package game

import (
	"testing"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestPowerAction_Bridge(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 5
	initialBowl1 := player.Resources.Power.Bowl1
	
	action := NewPowerAction("player1", PowerActionBridge)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed, got error: %v", err)
	}
	
	// Verify power was moved from Bowl3 to Bowl1
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (5-3), got %d", player.Resources.Power.Bowl3)
	}
	if player.Resources.Power.Bowl1 != initialBowl1+3 {
		t.Errorf("expected Bowl1 to have %d power, got %d", initialBowl1+3, player.Resources.Power.Bowl1)
	}
	
	// Verify bridge count increased
	if player.BridgesBuilt != 1 {
		t.Errorf("expected 1 bridge built, got %d", player.BridgesBuilt)
	}
	
	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be marked as used")
	}
}

func TestPowerAction_BridgeLimit(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 20
	player.BridgesBuilt = 3 // Already at limit
	
	action := NewPowerAction("player1", PowerActionBridge)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when building 4th bridge")
	}
}

func TestPowerAction_Priest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 5
	initialPriests := player.Resources.Priests
	
	action := NewPowerAction("player1", PowerActionPriest)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected priest action to succeed, got error: %v", err)
	}
	
	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (5-3), got %d", player.Resources.Power.Bowl3)
	}
	
	// Verify priest was gained
	if player.Resources.Priests != initialPriests+1 {
		t.Errorf("expected %d priests, got %d", initialPriests+1, player.Resources.Priests)
	}
	
	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionPriest) {
		t.Error("expected priest action to be marked as used")
	}
}

func TestPowerAction_Workers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	initialWorkers := player.Resources.Workers
	
	action := NewPowerAction("player1", PowerActionWorkers)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected workers action to succeed, got error: %v", err)
	}
	
	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}
	
	// Verify 2 workers were gained
	if player.Resources.Workers != initialWorkers+2 {
		t.Errorf("expected %d workers, got %d", initialWorkers+2, player.Resources.Workers)
	}
}

func TestPowerAction_Coins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	initialCoins := player.Resources.Coins
	
	action := NewPowerAction("player1", PowerActionCoins)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected coins action to succeed, got error: %v", err)
	}
	
	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}
	
	// Verify 7 coins were gained
	if player.Resources.Coins != initialCoins+7 {
		t.Errorf("expected %d coins, got %d", initialCoins+7, player.Resources.Coins)
	}
}

func TestPowerAction_Spade1WithTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	
	// Place player1's initial dwelling at (0, 1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Forest (1 spade away from Plains)
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	// Use 1 spade power action to transform and build
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade1 action to succeed, got error: %v", err)
	}
	
	// Verify power was spent (4 power for 1 spade action)
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}
	
	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionSpade1) {
		t.Error("expected spade1 action to be marked as used")
	}
}

func TestPowerAction_Spade2WithTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	
	// Place player1's initial dwelling at (0, 1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Lake (2 spades away from Plains: Plains -> Swamp -> Lake)
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)
	
	// Use 2 spade power action to transform and build
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action to succeed, got error: %v", err)
	}
	
	// Verify power was spent (6 power for 2 spade action)
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (8-6), got %d", player.Resources.Power.Bowl3)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}
	
	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestPowerAction_Spade1WithAdditionalWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	
	// Place player1's initial dwelling at (0, 1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Lake (2 spades away from Plains: Plains -> Swamp -> Lake)
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)
	
	initialWorkers := player.Resources.Workers
	
	// Use 1 spade power action - need 1 more spade from workers
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade1 action with workers to succeed, got error: %v", err)
	}
	
	// Verify 1 spade was paid with workers (for the 2nd spade needed)
	// Halflings have 3 workers per spade at base level
	// Also, building the dwelling costs 1 worker
	dwellingCost := faction.GetDwellingCost()
	workersPerSpade := faction.GetTerraformCost(1)
	expectedWorkers := initialWorkers - workersPerSpade - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers remaining, got %d", expectedWorkers, player.Resources.Workers)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}
}

func TestPowerAction_Spade2WithAdditionalWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	
	// Place player1's initial dwelling at (0, 1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Forest (3 spades away from Plains: Plains -> Swamp -> Lake -> Forest)
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	initialWorkers := player.Resources.Workers
	
	// Use 2 spade power action - need 1 more spade from workers (3 total needed, 2 free)
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action with workers to succeed, got error: %v", err)
	}
	
	// Verify 1 spade was paid with workers (for the 3rd spade needed)
	// Halflings have 3 workers per spade at base level
	// Also, building the dwelling costs 1 worker
	dwellingCost := faction.GetDwellingCost()
	workersPerSpade := faction.GetTerraformCost(1)
	expectedWorkers := initialWorkers - workersPerSpade - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers remaining, got %d", expectedWorkers, player.Resources.Workers)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}
	
	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestPowerAction_Spade2TwoHexes(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	
	// Place player1's initial dwelling at (1, 1)
	initialHex := NewHex(1, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// First target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Swamp (1 spade away from Plains)
	targetHex1 := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex1, models.TerrainSwamp)
	
	// Second target hex at (2, 1) - adjacent to initial dwelling
	// Set it to Swamp (1 spade away from Plains)
	targetHex2 := NewHex(2, 1)
	gs.Map.TransformTerrain(targetHex2, models.TerrainSwamp)
	
	// Use 2 spade power action - transform first hex and build dwelling
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex1, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action to succeed, got error: %v", err)
	}
	
	// Verify first hex was transformed and has dwelling
	mapHex1 := gs.Map.GetHex(targetHex1)
	if mapHex1.Terrain != models.TerrainPlains {
		t.Errorf("expected first hex terrain to be Plains, got %v", mapHex1.Terrain)
	}
	if mapHex1.Building == nil {
		t.Fatal("expected dwelling to be built on first hex")
	}
	if mapHex1.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling on first hex, got %v", mapHex1.Building.Type)
	}
	
	// Second hex should still be Swamp (not transformed)
	mapHex2 := gs.Map.GetHex(targetHex2)
	if mapHex2.Terrain != models.TerrainSwamp {
		t.Errorf("expected second hex to remain Swamp, got %v", mapHex2.Terrain)
	}
	if mapHex2.Building != nil {
		t.Error("expected no building on second hex")
	}
	
	// Note: According to the rulebook, if you have 2 free spades and only need 1 for the first hex,
	// you MAY spend the second spade on another hex, but you may NOT place a dwelling on that other space.
	// This test verifies that the current implementation only transforms and builds on ONE hex.
	// A future enhancement could allow using the extra spade on a second hex for transform only.
}

func TestPowerAction_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings() // Plains
	faction2 := factions.NewSwarmlings() // Lake - different from Halflings
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Power.Bowl3 = 10
	player2.Resources.Power.Bowl3 = 10
	
	// Player1 takes bridge action
	action1 := NewPowerAction("player1", PowerActionBridge)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected player1 bridge action to succeed, got error: %v", err)
	}
	
	// Player2 tries to take same action - should fail
	action2 := NewPowerAction("player2", PowerActionBridge)
	err = action2.Execute(gs)
	if err == nil {
		t.Fatal("expected error when player2 tries to take already-used bridge action")
	}
	
	// Player2 can take a different action
	action3 := NewPowerAction("player2", PowerActionPriest)
	err = action3.Execute(gs)
	if err != nil {
		t.Fatalf("expected player2 priest action to succeed, got error: %v", err)
	}
}

func TestPowerAction_ResetBetweenRounds(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 20
	
	// Take bridge action in round 1
	action1 := NewPowerAction("player1", PowerActionBridge)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed, got error: %v", err)
	}
	
	// Verify action is used
	if gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be marked as used")
	}
	
	// Start new round
	gs.StartNewRound()
	
	// Verify action is available again
	if !gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be available after new round")
	}
	
	// Can take the action again
	action2 := NewPowerAction("player1", PowerActionBridge)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed in round 2, got error: %v", err)
	}
	
	// Should have 2 bridges now
	if player.BridgesBuilt != 2 {
		t.Errorf("expected 2 bridges built, got %d", player.BridgesBuilt)
	}
}

func TestPowerAction_InsufficientPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 2 // Not enough for any action
	
	action := NewPowerAction("player1", PowerActionBridge)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when player has insufficient power")
	}
}

// ============================================================================
// BRIDGE GEOMETRY TESTS
// ============================================================================

func TestBridge_ValidGeometry_BaseOrientation(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Test base orientation: delta (1,-2) with midpoints (0,-1) and (1,-1)
	// This is the canonical valid bridge pattern
	hex1 := NewHex(0, 0)
	river1 := NewHex(0, -1)
	river2 := NewHex(1, -1)
	hex2 := NewHex(1, -2)
	
	// Set up map
	gs.Map.Hexes[hex1] = &MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	
	// Build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("valid bridge should succeed: %v", err)
	}
	
	// Verify bridge exists
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Error("bridge should exist on map")
	}
	
	// Verify hexes are now considered adjacent
	if !gs.Map.IsDirectlyAdjacent(hex1, hex2) {
		t.Error("hexes should be adjacent via bridge")
	}
}

func TestBridge_ValidGeometry_BidirectionalBridge(t *testing.T) {
	// Bridges are bidirectional - can be built in either direction
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hex1 := NewHex(0, 0)
	river1 := NewHex(0, -1)
	river2 := NewHex(1, -1)
	hex2 := NewHex(1, -2)
	
	// Set up map
	gs.Map.Hexes[hex1] = &MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	
	// Build bridge in reverse direction (hex2 to hex1)
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex2, hex1)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("bridge should work in both directions: %v", err)
	}
	
	// Verify bridge exists (should work both ways)
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Error("bridge should exist on map")
	}
	if !gs.Map.HasBridge(hex2, hex1) {
		t.Error("bridge should work in reverse direction too")
	}
	
	// Verify hexes are adjacent
	if !gs.Map.IsDirectlyAdjacent(hex1, hex2) {
		t.Error("hexes should be adjacent via bridge")
	}
	if !gs.Map.IsDirectlyAdjacent(hex2, hex1) {
		t.Error("adjacency should be bidirectional")
	}
}

func TestBridge_InvalidGeometry_NonRiverMidpoint(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Try to build bridge where one midpoint is NOT a river
	hex1 := NewHex(0, 0)
	river1 := NewHex(0, -1)
	notRiver := NewHex(1, -1) // This should be river but isn't
	hex2 := NewHex(1, -2)
	
	// Set up map
	gs.Map.Hexes[hex1] = &MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[notRiver] = &MapHex{Coord: notRiver, Terrain: models.TerrainPlains} // NOT river!
	gs.Map.Hexes[hex2] = &MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}
	gs.Map.RiverHexes[river1] = true
	// notRiver is NOT marked as river
	
	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge with non-river midpoint should fail")
	}
	
	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, hex2) {
		t.Error("invalid bridge should not exist on map")
	}
}

func TestBridge_InvalidGeometry_WrongDistance(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Try to build bridge with wrong distance (adjacent hexes = distance 1, not distance 2)
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0) // Adjacent, but bridges must span distance 2
	
	// Set up map
	gs.Map.Hexes[hex1] = &MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[hex2] = &MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}
	
	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge between adjacent hexes should fail")
	}
	
	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, hex2) {
		t.Error("invalid bridge should not exist on map")
	}
}

func TestBridge_InvalidGeometry_RiverEndpoint(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Try to build bridge where one endpoint is a river (not allowed)
	hex1 := NewHex(0, 0)
	riverEndpoint := NewHex(1, -2)
	river1 := NewHex(0, -1)
	river2 := NewHex(1, -1)
	
	// Set up map
	gs.Map.Hexes[hex1] = &MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[riverEndpoint] = &MapHex{Coord: riverEndpoint, Terrain: models.TerrainRiver} // Endpoint is river!
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	gs.Map.RiverHexes[riverEndpoint] = true
	
	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, riverEndpoint)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge with river endpoint should fail")
	}
	
	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, riverEndpoint) {
		t.Error("invalid bridge should not exist on map")
	}
}
