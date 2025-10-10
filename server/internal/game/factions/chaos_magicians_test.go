package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestChaosMagicians_BasicProperties(t *testing.T) {
	cm := NewChaosMagicians()

	if cm.GetType() != models.FactionChaosMagicians {
		t.Errorf("expected faction type ChaosMagicians, got %v", cm.GetType())
	}

	if cm.GetHomeTerrain() != models.TerrainWasteland {
		t.Errorf("expected home terrain Wasteland, got %v", cm.GetHomeTerrain())
	}
}

func TestChaosMagicians_StartingResources(t *testing.T) {
	cm := NewChaosMagicians()
	resources := cm.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 4 {
		t.Errorf("expected 4 workers (not standard 3), got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
}

func TestChaosMagicians_HasFavorTransformAbility(t *testing.T) {
	cm := NewChaosMagicians()

	if !cm.HasSpecialAbility(AbilityFavorTransform) {
		t.Errorf("Chaos Magicians should have favor transform ability")
	}
}

func TestChaosMagicians_ExpensiveSanctuary(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians sanctuary costs 8 coins (more expensive than standard 6)
	sanctuaryCost := cm.GetSanctuaryCost()
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected sanctuary to cost 8 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestChaosMagicians_CheapStronghold(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians stronghold costs 4 coins (cheaper than standard 6)
	strongholdCost := cm.GetStrongholdCost()
	if strongholdCost.Coins != 4 {
		t.Errorf("expected stronghold to cost 4 coins (cheaper than standard 6), got %d", strongholdCost.Coins)
	}
	if strongholdCost.Workers != 4 {
		t.Errorf("expected stronghold to cost 4 workers, got %d", strongholdCost.Workers)
	}
}

func TestChaosMagicians_StrongholdAbility(t *testing.T) {
	cm := NewChaosMagicians()

	ability := cm.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Chaos Magicians should have a stronghold ability")
	}
}

func TestChaosMagicians_DoubleTurnBeforeStronghold(t *testing.T) {
	cm := NewChaosMagicians()

	// Should not be able to use double-turn before building stronghold
	if cm.CanUseDoubleTurn() {
		t.Errorf("should not be able to use double-turn before building stronghold")
	}

	err := cm.UseDoubleTurn()
	if err == nil {
		t.Errorf("expected error when using double-turn without stronghold")
	}
}

func TestChaosMagicians_DoubleTurnAfterStronghold(t *testing.T) {
	cm := NewChaosMagicians()

	// Build stronghold
	cm.BuildStronghold()

	// Should be able to use double-turn
	if !cm.CanUseDoubleTurn() {
		t.Errorf("should be able to use double-turn after building stronghold")
	}

	// Use double-turn
	err := cm.UseDoubleTurn()
	if err != nil {
		t.Fatalf("failed to use double-turn: %v", err)
	}

	// Should not be able to use again this Action phase
	if cm.CanUseDoubleTurn() {
		t.Errorf("should not be able to use double-turn twice in one Action phase")
	}

	// Try to use again (should fail)
	err = cm.UseDoubleTurn()
	if err == nil {
		t.Errorf("expected error when using double-turn twice")
	}
}

func TestChaosMagicians_DoubleTurnReset(t *testing.T) {
	cm := NewChaosMagicians()
	cm.BuildStronghold()

	// Use double-turn
	err := cm.UseDoubleTurn()
	if err != nil {
		t.Fatalf("failed to use double-turn: %v", err)
	}

	// Should not be able to use again
	if cm.CanUseDoubleTurn() {
		t.Errorf("should not be able to use double-turn before reset")
	}

	// Reset for new Action phase
	cm.ResetDoubleTurn()

	// Should be able to use again
	if !cm.CanUseDoubleTurn() {
		t.Errorf("should be able to use double-turn after reset")
	}
}

func TestChaosMagicians_FavorTilesForTemple(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians get 2 favor tiles for Temple (not standard 1)
	favorTiles := cm.GetFavorTilesForTemple()
	if favorTiles != 2 {
		t.Errorf("expected 2 favor tiles for Temple, got %d", favorTiles)
	}
}

func TestChaosMagicians_FavorTilesForSanctuary(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians get 2 favor tiles for Sanctuary (not standard 1)
	favorTiles := cm.GetFavorTilesForSanctuary()
	if favorTiles != 2 {
		t.Errorf("expected 2 favor tiles for Sanctuary, got %d", favorTiles)
	}
}

func TestChaosMagicians_StartsWithOneDwelling(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians start with only 1 dwelling
	if !cm.StartsWithOneDwelling() {
		t.Errorf("Chaos Magicians should start with only 1 dwelling")
	}
}

func TestChaosMagicians_PlacesDwellingLast(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians place dwelling after all other players
	if !cm.PlacesDwellingLast() {
		t.Errorf("Chaos Magicians should place dwelling last")
	}
}

func TestChaosMagicians_StandardCosts(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians use standard costs for most buildings
	dwellingCost := cm.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 0 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := cm.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
