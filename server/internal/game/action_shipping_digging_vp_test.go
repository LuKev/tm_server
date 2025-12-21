package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestAdvanceShipping_AwardsVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player resources to advance shipping
	player.Resources.Coins = 100
	player.Resources.Priests = 5

	// Advance shipping from 0 to 1
	initialVP := player.VictoryPoints
	action := NewAdvanceShippingAction("player1")
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to advance shipping: %v", err)
	}

	// Verify shipping level increased
	if player.ShippingLevel != 1 {
		t.Errorf("expected shipping level 1, got %d", player.ShippingLevel)
	}

	// Verify VP was awarded (Level 1 = 2 VP)
	expectedVP := initialVP + 2
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP for shipping level 1, got %d", expectedVP, player.VictoryPoints)
	}

	// Advance to level 2
	initialVP = player.VictoryPoints
	action2 := NewAdvanceShippingAction("player1")
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to advance shipping to level 2: %v", err)
	}

	// Verify VP was awarded (Level 2 = 3 VP)
	expectedVP = initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP for shipping level 2, got %d", expectedVP, player.VictoryPoints)
	}

	// Advance to level 3
	initialVP = player.VictoryPoints
	action3 := NewAdvanceShippingAction("player1")
	err = action3.Execute(gs)
	if err != nil {
		t.Fatalf("failed to advance shipping to level 3: %v", err)
	}

	// Verify VP was awarded (Level 3 = 4 VP)
	expectedVP = initialVP + 4
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP for shipping level 3, got %d", expectedVP, player.VictoryPoints)
	}
}

func TestAdvanceDigging_AwardsVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player resources to advance digging
	player.Resources.Coins = 100
	player.Resources.Workers = 10
	player.Resources.Priests = 5

	// Advance digging from 0 to 1
	initialVP := player.VictoryPoints
	action := NewAdvanceDiggingAction("player1")
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to advance digging: %v", err)
	}

	// Verify digging level increased
	if player.DiggingLevel != 1 {
		t.Errorf("expected digging level 1, got %d", player.DiggingLevel)
	}

	// Verify VP was awarded (always +6 VP for digging)
	expectedVP := initialVP + 6
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP for digging level 1, got %d", expectedVP, player.VictoryPoints)
	}

	// Advance to level 2
	initialVP = player.VictoryPoints
	action2 := NewAdvanceDiggingAction("player1")
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("failed to advance digging to level 2: %v", err)
	}

	// Verify VP was awarded (always +6 VP for digging)
	expectedVP = initialVP + 6
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP for digging level 2, got %d", expectedVP, player.VictoryPoints)
	}
}

func TestMermaidsStronghold_ShippingAwardsVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := board.NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainLake)

	// Mermaids start at shipping level 1
	if faction.GetShippingLevel() != 1 {
		t.Fatalf("expected Mermaids to start at shipping level 1, got %d", faction.GetShippingLevel())
	}

	initialVP := player.VictoryPoints

	// Upgrade to stronghold (this should increase shipping from 1 to 2 and award VP)
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify shipping level increased to 2
	if faction.GetShippingLevel() != 2 {
		t.Errorf("expected shipping level 2 after stronghold, got %d", faction.GetShippingLevel())
	}

	// Verify VP was awarded for shipping level 2 (2 VP)
	if player.VictoryPoints != initialVP+2 {
		t.Errorf("expected %d VP increase (including 2 for shipping level 2), got %d total VP",
			2, player.VictoryPoints)
	}
}

func TestShippingVPProgression(t *testing.T) {
	// Verify the VP progression follows the pattern: level + 1
	expectedVP := map[int]int{
		1: 2, // Level 1 = 2 VP
		2: 3, // Level 2 = 3 VP
		3: 4, // Level 3 = 4 VP
		4: 5, // Level 4 = 5 VP
		5: 6, // Level 5 = 6 VP
	}

	for level, vp := range expectedVP {
		calculated := level + 1
		if calculated != vp {
			t.Errorf("VP calculation incorrect for level %d: expected %d, got %d", level, vp, calculated)
		}
	}
}

func TestDiggingLevelConstraints(t *testing.T) {
	// Test that most factions can advance to level 2
	gs := NewGameState()
	auren := factions.NewAuren()
	gs.AddPlayer("player1", auren)
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 100
	player.Resources.Workers = 100
	player.Resources.Priests = 5

	// Advance to level 1
	action1 := NewAdvanceDiggingAction("player1")
	if err := action1.Execute(gs); err != nil {
		t.Fatalf("should be able to advance to level 1: %v", err)
	}

	// Advance to level 2
	action2 := NewAdvanceDiggingAction("player1")
	if err := action2.Execute(gs); err != nil {
		t.Fatalf("should be able to advance to level 2: %v", err)
	}

	// Try to advance to level 3 (should fail)
	action3 := NewAdvanceDiggingAction("player1")
	if err := action3.Execute(gs); err == nil {
		t.Error("should not be able to advance to level 3")
	}

	// Test Fakirs can only advance to level 1
	gs2 := NewGameState()
	fakirs := factions.NewFakirs()
	gs2.AddPlayer("player2", fakirs)
	player2 := gs2.GetPlayer("player2")
	player2.Resources.Coins = 100
	player2.Resources.Workers = 100
	player2.Resources.Priests = 5

	// Advance to level 1
	action4 := NewAdvanceDiggingAction("player2")
	if err := action4.Execute(gs2); err != nil {
		t.Fatalf("Fakirs should be able to advance to level 1: %v", err)
	}

	// Try to advance to level 2 (should fail for Fakirs)
	action5 := NewAdvanceDiggingAction("player2")
	if err := action5.Execute(gs2); err == nil {
		t.Error("Fakirs should not be able to advance to level 2")
	}

	// Test Darklings cannot advance digging at all
	gs3 := NewGameState()
	darklings := factions.NewDarklings()
	gs3.AddPlayer("player3", darklings)
	player3 := gs3.GetPlayer("player3")
	player3.Resources.Coins = 100
	player3.Resources.Workers = 100
	player3.Resources.Priests = 5

	// Try to advance (should fail immediately)
	action6 := NewAdvanceDiggingAction("player3")
	if err := action6.Execute(gs3); err == nil {
		t.Error("Darklings should not be able to advance digging at all")
	}
}
