package game

import (
	"testing"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Helper to create a building for tests
func testBuilding(playerID string, faction models.FactionType, buildingType models.BuildingType) *models.Building {
	powerValue := 1
	if buildingType == models.BuildingTradingHouse || buildingType == models.BuildingTemple {
		powerValue = 2
	} else if buildingType == models.BuildingSanctuary || buildingType == models.BuildingStronghold {
		powerValue = 3
	}
	return &models.Building{
		Type:       buildingType,
		Faction:    faction,
		PlayerID:   playerID,
		PowerValue: powerValue,
	}
}

func TestTransformAndBuild_ValidAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Give player enough resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 1) // Desert terrain
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Transform and build adjacent
	// Neighbors of (0,1) include (1,1), (0,2), (-1,2), (-1,1), (0,0), (1,0)
	// (1,1) is Plains (already home terrain), so use (0,0) which is Forest
	targetHex := NewHex(0, 0) // Forest terrain - directly adjacent to (0,1)
	initialTerrain := gs.Map.GetHex(targetHex).Terrain
	action := NewTransformAndBuildAction("player1", targetHex, true) // true = build dwelling
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Verify terrain was transformed to Plains (Halflings home terrain)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be transformed from %v to Plains, got %v", initialTerrain, mapHex.Terrain)
	}
}

func TestTransformAndBuild_NotAdjacent(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling at (0,1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to build at a non-adjacent location (5,5)
	// (5,5) is far from (0,1) and not adjacent
	targetHex := NewHex(5, 5)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for non-adjacent building")
	}
}

func TestTransformAndBuild_InsufficientResources(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	// Don't give enough resources
	player.Resources.Coins = 0
	player.Resources.Workers = 0
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Use adjacent hex (0,0) which is Forest
	targetHex := NewHex(0, 0)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for insufficient resources")
	}
}

func TestTransformAndBuild_SkipTerraform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling on Plains
	initialHex := NewHex(3, 1) // Plains terrain
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Build on adjacent home terrain (Plains for Halflings) - no transform needed
	// Use adjacent hex (4,1) which should also be Plains or transform it to Plains first
	targetHex := NewHex(4, 1)
	// Ensure target is Plains (home terrain)
	gs.Map.TransformTerrain(targetHex, models.TerrainPlains)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	initialWorkers := player.Resources.Workers
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify no workers were spent on terraform (only dwelling cost)
	dwellingCost := faction.GetDwellingCost()
	expectedWorkers := initialWorkers - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers (started with %d, dwelling cost %d), got %d", 
			expectedWorkers, initialWorkers, dwellingCost.Workers, player.Resources.Workers)
	}
}

func TestTransformAndBuild_PowerLeech(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 10
	player1.Resources.Workers = 10
	
	// Place player1's initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction1.GetType(), models.BuildingDwelling)
	
	// Place player2's dwelling adjacent to where player1 will build
	player2Hex := NewHex(2, 2)
	gs.Map.GetHex(player2Hex).Building = testBuilding("player2", faction2.GetType(), models.BuildingDwelling)
	
	// Player1 builds adjacent to both buildings
	// Use (1,1) which is adjacent to (0,1) and close to (2,2)
	targetHex := NewHex(1, 1)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	// Record player2's initial power
	initialPower2Bowl1 := player2.Resources.Power.Bowl1
	initialPower2Bowl2 := player2.Resources.Power.Bowl2
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Power leech is triggered but not automatically accepted
	// The offer is created in TriggerPowerLeech
	// For now, we just verify the action executed successfully
	// TODO: When Phase 6.1 is complete, verify power leech offers are stored
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	}
	
	// Power should not have changed yet (player2 hasn't accepted/declined)
	if player2.Resources.Power.Bowl1 != initialPower2Bowl1 {
		t.Errorf("power should not change until leech is accepted")
	}
	if player2.Resources.Power.Bowl2 != initialPower2Bowl2 {
		t.Errorf("power should not change until leech is accepted")
	}
}

func TestTransformAndBuild_HexAlreadyOccupied(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to build on occupied hex
	action := NewTransformAndBuildAction("player1", initialHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for building on occupied hex")
	}
}

func TestTransformAndBuild_TransformOnly(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Transform adjacent hex WITHOUT building (buildDwelling = false)
	// Use (0,0) which is Forest
	targetHex := NewHex(0, 0)
	action := NewTransformAndBuildAction("player1", targetHex, false)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected transform-only to succeed, got error: %v", err)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be transformed to Plains, got %v", mapHex.Terrain)
	}
	
	// Verify NO building was placed
	if mapHex.Building != nil {
		t.Errorf("expected no building, but found %v", mapHex.Building.Type)
	}
}

func TestTransformAndBuild_InsufficientWorkersForTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 2 // Not enough for transform + dwelling
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to transform and build on non-Plains terrain
	// Use (1,0) which should be adjacent to (0,1)
	// We need to ensure it's NOT Plains
	targetHex := NewHex(1, 0)
	// Force it to be Forest to ensure transform is needed
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	// Forest -> Plains: distance 3 on the wheel
	// Halflings at digging level 0: 3 workers per spade
	// Total terraform cost: 3 * 3 = 9 workers
	// Plus 1 worker for dwelling = 10 workers total needed
	// But player only has 2 workers
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for insufficient workers to transform")
	}
}

func TestTransformAndBuild_IndirectAdjacency(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.ShippingLevel = 1 // Shipping level 1 allows indirect adjacency via river
	
	// Base map layout (from map_indirect_base_test.go):
	// Row 1: Desert(0,1), River(1,1), River(2,1), Plains(3,1), Swamp(4,1), ...
	// Row 2: River(0,2), River(1,2), Swamp(1,2), River(3,2), Mountain(4,2), ...
	//
	// (0,1) Desert and (1,2) Swamp are indirectly adjacent with shipping=1
	// via river path: (0,1) -> river neighbor -> (1,2)
	
	// Place initial dwelling at (0, 1) - Desert
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to build at (1, 2) - Swamp
	// This is indirectly adjacent to (0,1) with shipping level 1
	targetHex := NewHex(1, 2)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed with shipping level 1 (indirect adjacency), got error: %v", err)
	}
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestTransformAndBuild_AdvancedDiggingLevel(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Advance digging level to 2
	// Base cost: 3 workers per spade
	// With digging level 2: 3 - 2 = 1 worker per spade
	player.Faction.(*factions.Halflings).DiggingLevel = 2
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Transform and build on Forest terrain
	// Forest -> Plains: distance 3
	// With digging level 2: 1 worker per spade * 3 = 3 workers for terraform
	// Plus 1 worker for dwelling = 4 workers total
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	initialWorkers := player.Resources.Workers
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed with advanced digging, got error: %v", err)
	}
	
	// Verify correct number of workers were spent
	// Should be 3 for terraform + 1 for dwelling = 4 total
	expectedWorkers := initialWorkers - 4
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers (started with %d, should spend 4), got %d", 
			expectedWorkers, initialWorkers, player.Resources.Workers)
	}
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	} // Wasteland (row 3)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be transformed to Plains, got %v", mapHex.Terrain)
	}
}

func TestTransformAndBuild_PowerLeechOffers(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 20
	player1.Resources.Workers = 20
	
	// Place player1's initial dwelling at (0, 1)
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction1.GetType(), models.BuildingDwelling)
	
	// Place player2's dwelling adjacent to where player1 will build
	// Neighbors of (0,0) are: (1,0), (0,1), (-1,1), (-1,0), (0,-1), (1,-1)
	// We'll place player2 at (1,0) which is adjacent to (0,0)
	player2Hex := NewHex(1, 0)
	gs.Map.GetHex(player2Hex).Building = testBuilding("player2", faction2.GetType(), models.BuildingDwelling)
	
	// Player1 builds at (0,0) which is adjacent to both (0,1) and (1,0)
	targetHex := NewHex(0, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	// Record player2's initial state
	initialVP := player2.VictoryPoints
	initialBowl1 := player2.Resources.Power.Bowl1
	initialBowl2 := player2.Resources.Power.Bowl2
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify player2 has a pending leech offer
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) == 0 {
		t.Fatalf("expected player2 to have a pending leech offer")
	}
	
	offer := offers[0]
	if offer.Amount != 1 {
		t.Errorf("expected offer amount of 1 (dwelling power value), got %d", offer.Amount)
	}
	if offer.VPCost != 0 {
		t.Errorf("expected VP cost of 0 (amount - 1), got %d", offer.VPCost)
	}
	if offer.FromPlayerID != "player1" {
		t.Errorf("expected offer from player1, got %s", offer.FromPlayerID)
	}
	
	// Test accepting the offer
	err = gs.AcceptLeechOffer("player2", 0)
	if err != nil {
		t.Fatalf("expected to accept leech offer, got error: %v", err)
	}
	
	// Verify player2 gained power (power moves from Bowl1 to Bowl2)
	// GainPower(1) should move 1 power from Bowl1 to Bowl2
	if player2.Resources.Power.Bowl1 != initialBowl1 - 1 {
		t.Errorf("expected Bowl1 to decrease by 1, initial: %d, new: %d", initialBowl1, player2.Resources.Power.Bowl1)
	}
	if player2.Resources.Power.Bowl2 != initialBowl2 + 1 {
		t.Errorf("expected Bowl2 to increase by 1, initial: %d, new: %d", initialBowl2, player2.Resources.Power.Bowl2)
	}
	
	// Verify player2 lost VP
	if player2.VictoryPoints != initialVP - offer.VPCost {
		t.Errorf("expected player2 to lose %d VP, initial: %d, new: %d", offer.VPCost, initialVP, player2.VictoryPoints)
	}
	
	// Verify offer was removed
	offers = gs.GetPendingLeechOffers("player2")
	if len(offers) != 0 {
		t.Errorf("expected no pending offers after accepting, got %d", len(offers))
	}
}

func TestTransformAndBuild_DeclineLeechOffer(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 20
	player1.Resources.Workers = 20
	
	// Place player1's initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction1.GetType(), models.BuildingDwelling)
	
	// Place player2's dwelling adjacent to where player1 will build
	// We'll place player2 at (1,0) which is adjacent to (0,0)
	player2Hex := NewHex(1, 0)
	gs.Map.GetHex(player2Hex).Building = testBuilding("player2", faction2.GetType(), models.BuildingDwelling)
	
	// Player1 builds at (0,0) which is adjacent to both (0,1) and (1,0)
	targetHex := NewHex(0, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	// Record player2's initial state
	initialVP := player2.VictoryPoints
	initialBowl1 := player2.Resources.Power.Bowl1
	initialBowl2 := player2.Resources.Power.Bowl2
	initialBowl3 := player2.Resources.Power.Bowl3
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify player2 has a pending leech offer
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) == 0 {
		t.Fatalf("expected player2 to have a pending leech offer")
	}
	
	// Test declining the offer
	err = gs.DeclineLeechOffer("player2", 0)
	if err != nil {
		t.Fatalf("expected to decline leech offer, got error: %v", err)
	}
	
	// Verify player2 did NOT gain power (all bowls unchanged)
	if player2.Resources.Power.Bowl1 != initialBowl1 {
		t.Errorf("expected Bowl1 to remain unchanged, initial: %d, new: %d", initialBowl1, player2.Resources.Power.Bowl1)
	}
	if player2.Resources.Power.Bowl2 != initialBowl2 {
		t.Errorf("expected Bowl2 to remain unchanged, initial: %d, new: %d", initialBowl2, player2.Resources.Power.Bowl2)
	}
	if player2.Resources.Power.Bowl3 != initialBowl3 {
		t.Errorf("expected Bowl3 to remain unchanged, initial: %d, new: %d", initialBowl3, player2.Resources.Power.Bowl3)
	}
	
	// Verify player2 did NOT lose VP
	if player2.VictoryPoints != initialVP {
		t.Errorf("expected player2 VP to remain unchanged, initial: %d, new: %d", initialVP, player2.VictoryPoints)
	}
	
	// Verify offer was removed
	offers = gs.GetPendingLeechOffers("player2")
	if len(offers) != 0 {
		t.Errorf("expected no pending offers after declining, got %d", len(offers))
	}
}

func TestTransformAndBuild_MultipleAdjacentBuildings(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 20
	player1.Resources.Workers = 20
	
	// Place player1's initial dwelling at (2, 1) - River
	initialHex := NewHex(2, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction1.GetType(), models.BuildingDwelling)
	
	// Place player2's Temple (power value 2) at (1, 1) - adjacent to (1,2)
	player2Temple := NewHex(1, 1)
	gs.Map.GetHex(player2Temple).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 2,
	}
	
	// Place player2's Stronghold (power value 3) at (2, 2) - also adjacent to (1,2)
	player2Stronghold := NewHex(2, 2)
	gs.Map.GetHex(player2Stronghold).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 3,
	}
	
	// Player1 builds at (1, 2) which is adjacent to:
	// - player1's dwelling at (2,1) ✓
	// - player2's Temple at (1,1) ✓
	// - player2's Stronghold at (2,2) ✓
	// Neighbors of (1,2): (2,2), (1,3), (0,3), (0,2), (1,1), (2,1)
	targetHex := NewHex(1, 2)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify player2 has ONE leech offer with TOTAL power from both buildings
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) != 1 {
		t.Fatalf("expected player2 to have exactly 1 leech offer, got %d", len(offers))
	}
	
	offer := offers[0]
	// Total power should be Temple (2) + Stronghold (3) = 5
	if offer.Amount != 5 {
		t.Errorf("expected offer amount of 5 (2 from temple + 3 from stronghold), got %d", offer.Amount)
	}
	// VP cost should be 5 - 1 = 4
	if offer.VPCost != 4 {
		t.Errorf("expected VP cost of 4 (amount - 1), got %d", offer.VPCost)
	}
	if offer.FromPlayerID != "player1" {
		t.Errorf("expected offer from player1, got %s", offer.FromPlayerID)
	}
	
	// Test accepting the offer
	initialBowl1 := player2.Resources.Power.Bowl1
	initialBowl2 := player2.Resources.Power.Bowl2
	initialVP := player2.VictoryPoints
	
	err = gs.AcceptLeechOffer("player2", 0)
	if err != nil {
		t.Fatalf("expected to accept leech offer, got error: %v", err)
	}
	
	// Verify player2 gained 5 power
	// With 5 power to gain and starting with Bowl1=5, Bowl2=7:
	// All 5 from Bowl1 moves to Bowl2
	if player2.Resources.Power.Bowl1 != initialBowl1 - 5 {
		t.Errorf("expected Bowl1 to decrease by 5, initial: %d, new: %d", initialBowl1, player2.Resources.Power.Bowl1)
	}
	if player2.Resources.Power.Bowl2 != initialBowl2 + 5 {
		t.Errorf("expected Bowl2 to increase by 5, initial: %d, new: %d", initialBowl2, player2.Resources.Power.Bowl2)
	}
	
	// Verify player2 lost 4 VP
	if player2.VictoryPoints != initialVP - 4 {
		t.Errorf("expected player2 to lose 4 VP, initial: %d, new: %d", initialVP, player2.VictoryPoints)
	}
}
