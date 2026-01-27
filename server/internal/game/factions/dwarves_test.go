package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestDwarves_BasicProperties(t *testing.T) {
	dwarves := NewDwarves()

	if dwarves.GetType() != models.FactionDwarves {
		t.Errorf("expected faction type Dwarves, got %v", dwarves.GetType())
	}

	if dwarves.GetHomeTerrain() != models.TerrainMountain {
		t.Errorf("expected home terrain Mountain, got %v", dwarves.GetHomeTerrain())
	}
}

func TestDwarves_StartingResources(t *testing.T) {
	dwarves := NewDwarves()
	resources := dwarves.GetStartingResources()

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

func TestDwarves_CannotUpgradeShipping(t *testing.T) {
	dwarves := NewDwarves()

	// Dwarves can never upgrade shipping
	shippingCost := dwarves.GetShippingCost(0)
	if shippingCost.Workers != 0 || shippingCost.Coins != 0 {
		t.Errorf("Dwarves should not be able to upgrade shipping, got cost: %+v", shippingCost)
	}

	// Try level 1 as well
	shippingCost = dwarves.GetShippingCost(1)
	if shippingCost.Workers != 0 || shippingCost.Coins != 0 {
		t.Errorf("Dwarves should not be able to upgrade shipping at any level, got cost: %+v", shippingCost)
	}
}

func TestDwarves_HasStronghold(t *testing.T) {
	dwarves := NewDwarves()

	// Before building
	if dwarves.HasStronghold() {
		t.Errorf("should not have stronghold before building")
	}

	// After building
	dwarves.BuildStronghold()
	if !dwarves.HasStronghold() {
		t.Errorf("should have stronghold after building")
	}
}

func TestDwarves_StandardCosts(t *testing.T) {
	dwarves := NewDwarves()

	// Dwarves use standard building costs
	dwellingCost := dwarves.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := dwarves.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	sanctuaryCost := dwarves.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}

	strongholdCost := dwarves.GetStrongholdCost()
	if strongholdCost.Workers != 4 || strongholdCost.Coins != 6 {
		t.Errorf("unexpected stronghold cost: %+v", strongholdCost)
	}
}
