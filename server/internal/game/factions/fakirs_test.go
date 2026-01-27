package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestFakirs_BasicProperties(t *testing.T) {
	fakirs := NewFakirs()

	if fakirs.GetType() != models.FactionFakirs {
		t.Errorf("expected faction type Fakirs, got %v", fakirs.GetType())
	}

	if fakirs.GetHomeTerrain() != models.TerrainDesert {
		t.Errorf("expected home terrain Desert, got %v", fakirs.GetHomeTerrain())
	}
}

func TestFakirs_StartingResources(t *testing.T) {
	fakirs := NewFakirs()
	resources := fakirs.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
	if resources.Power1 != 7 {
		t.Errorf("expected 7 power in bowl 1 (not standard 5), got %d", resources.Power1)
	}
	if resources.Power2 != 5 {
		t.Errorf("expected 5 power in bowl 2 (not standard 7), got %d", resources.Power2)
	}
	if resources.Power3 != 0 {
		t.Errorf("expected 0 power in bowl 3, got %d", resources.Power3)
	}
}

func TestFakirs_ExpensiveStronghold(t *testing.T) {
	fakirs := NewFakirs()

	// Fakirs stronghold costs 10 coins (more expensive than standard 6)
	strongholdCost := fakirs.GetStrongholdCost()
	if strongholdCost.Coins != 10 {
		t.Errorf("expected stronghold to cost 10 coins, got %d", strongholdCost.Coins)
	}
	if strongholdCost.Workers != 4 {
		t.Errorf("expected stronghold to cost 4 workers, got %d", strongholdCost.Workers)
	}
}

func TestFakirs_CannotUpgradeShipping(t *testing.T) {
	fakirs := NewFakirs()

	// Fakirs can never upgrade shipping
	shippingCost := fakirs.GetShippingCost(0)
	if shippingCost.Workers != 0 || shippingCost.Coins != 0 {
		t.Errorf("Fakirs should not be able to upgrade shipping, got cost: %+v", shippingCost)
	}

	// Try level 1 as well
	shippingCost = fakirs.GetShippingCost(1)
	if shippingCost.Workers != 0 || shippingCost.Coins != 0 {
		t.Errorf("Fakirs should not be able to upgrade shipping at any level, got cost: %+v", shippingCost)
	}
}

func TestFakirs_CanOnlyUpgradeDiggingOnce(t *testing.T) {
	fakirs := NewFakirs()

	// Can upgrade from 0 to 1
	diggingCost := fakirs.GetDiggingCost(0)
	if diggingCost.Workers == 0 && diggingCost.Coins == 0 {
		t.Errorf("Fakirs should be able to upgrade digging from 0 to 1")
	}

	// Cannot upgrade from 1 to 2
	diggingCost = fakirs.GetDiggingCost(1)
	if diggingCost.Workers != 0 || diggingCost.Coins != 0 {
		t.Errorf("Fakirs should not be able to upgrade digging past level 1, got cost: %+v", diggingCost)
	}
}

func TestFakirs_MaxDiggingLevel(t *testing.T) {
	fakirs := NewFakirs()

	// Fakirs can only reach digging level 1
	maxDigging := fakirs.GetMaxDiggingLevel()
	if maxDigging != 1 {
		t.Errorf("expected max digging level 1, got %d", maxDigging)
	}
}

func TestFakirs_HasStronghold(t *testing.T) {
	fakirs := NewFakirs()

	// Before building
	if fakirs.HasStronghold() {
		t.Errorf("should not have stronghold before building")
	}

	// After building
	fakirs.BuildStronghold()
	if !fakirs.HasStronghold() {
		t.Errorf("should have stronghold after building")
	}
}

func TestFakirs_StandardCosts(t *testing.T) {
	fakirs := NewFakirs()

	// Fakirs use standard costs for most buildings
	dwellingCost := fakirs.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := fakirs.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	sanctuaryCost := fakirs.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}
}
func TestFakirs_FlightRange(t *testing.T) {
	fakirs := NewFakirs()

	// Base flight range is 1
	if fakirs.GetFlightRange() != 1 {
		t.Errorf("expected base flight range 1, got %d", fakirs.GetFlightRange())
	}

	// Stronghold increases range by 1
	fakirs.BuildStronghold()
	if fakirs.GetFlightRange() != 2 {
		t.Errorf("expected flight range 2 after stronghold, got %d", fakirs.GetFlightRange())
	}

	// Shipping town tile increases range by 1
	fakirs.IncrementFlightRange()
	if fakirs.GetFlightRange() != 3 {
		t.Errorf("expected flight range 3 after shipping town tile, got %d", fakirs.GetFlightRange())
	}

	// Multiple shipping town tiles increase range further
	fakirs.IncrementFlightRange()
	if fakirs.GetFlightRange() != 4 {
		t.Errorf("expected flight range 4 after second shipping town tile, got %d", fakirs.GetFlightRange())
	}
}
