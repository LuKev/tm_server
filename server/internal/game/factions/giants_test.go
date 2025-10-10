package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestGiants_BasicProperties(t *testing.T) {
	giants := NewGiants()

	if giants.GetType() != models.FactionGiants {
		t.Errorf("expected faction type Giants, got %v", giants.GetType())
	}

	if giants.GetHomeTerrain() != models.TerrainWasteland {
		t.Errorf("expected home terrain Wasteland, got %v", giants.GetHomeTerrain())
	}
}

func TestGiants_StartingResources(t *testing.T) {
	giants := NewGiants()
	resources := giants.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
}

func TestGiants_HasSpadeEfficiencyAbility(t *testing.T) {
	giants := NewGiants()

	if !giants.HasSpecialAbility(AbilitySpadeEfficiency) {
		t.Errorf("Giants should have spade efficiency ability")
	}
}

func TestGiants_AlwaysTwoSpades(t *testing.T) {
	giants := NewGiants()

	// Giants always need exactly 2 spades
	spades := giants.GetTerraformSpades()
	if spades != 2 {
		t.Errorf("expected 2 spades, got %d", spades)
	}
}

func TestGiants_TerraformCostWithDigging0(t *testing.T) {
	giants := NewGiants()

	// With digging level 0: 3 workers per spade
	// Giants always need 2 spades = 6 workers
	cost := giants.GetTerraformCost(1) // Distance doesn't matter for Giants
	if cost != 6 {
		t.Errorf("expected 6 workers (2 spades * 3 workers/spade), got %d", cost)
	}

	// Same cost regardless of distance
	cost = giants.GetTerraformCost(3)
	if cost != 6 {
		t.Errorf("expected 6 workers regardless of distance, got %d", cost)
	}
}

func TestGiants_StrongholdAbility(t *testing.T) {
	giants := NewGiants()

	ability := giants.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Giants should have a stronghold ability")
	}
}

func TestGiants_FreeSpadesBeforeStronghold(t *testing.T) {
	giants := NewGiants()

	// Should not be able to use free spades before building stronghold
	if giants.CanUseFreeSpades() {
		t.Errorf("should not be able to use free spades before building stronghold")
	}

	_, err := giants.UseFreeSpades()
	if err == nil {
		t.Errorf("expected error when using free spades without stronghold")
	}
}

func TestGiants_FreeSpadesAfterStronghold(t *testing.T) {
	giants := NewGiants()

	// Build stronghold
	giants.BuildStronghold()

	// Should be able to use free spades
	if !giants.CanUseFreeSpades() {
		t.Errorf("should be able to use free spades after building stronghold")
	}

	// Use free spades
	spades, err := giants.UseFreeSpades()
	if err != nil {
		t.Fatalf("failed to use free spades: %v", err)
	}
	if spades != 2 {
		t.Errorf("expected 2 free spades, got %d", spades)
	}

	// Should not be able to use again this Action phase
	if giants.CanUseFreeSpades() {
		t.Errorf("should not be able to use free spades twice in one Action phase")
	}

	// Try to use again (should fail)
	_, err = giants.UseFreeSpades()
	if err == nil {
		t.Errorf("expected error when using free spades twice")
	}
}

func TestGiants_FreeSpadesReset(t *testing.T) {
	giants := NewGiants()
	giants.BuildStronghold()

	// Use free spades
	_, err := giants.UseFreeSpades()
	if err != nil {
		t.Fatalf("failed to use free spades: %v", err)
	}

	// Should not be able to use again
	if giants.CanUseFreeSpades() {
		t.Errorf("should not be able to use free spades before reset")
	}

	// Reset for new Action phase
	giants.ResetFreeSpades()

	// Should be able to use again
	if !giants.CanUseFreeSpades() {
		t.Errorf("should be able to use free spades after reset")
	}
}

func TestGiants_StandardCosts(t *testing.T) {
	giants := NewGiants()

	// Giants use standard building costs
	dwellingCost := giants.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 0 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := giants.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	sanctuaryCost := giants.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}

	strongholdCost := giants.GetStrongholdCost()
	if strongholdCost.Workers != 4 || strongholdCost.Coins != 6 {
		t.Errorf("unexpected stronghold cost: %+v", strongholdCost)
	}
}
