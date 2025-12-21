package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestMermaids_BasicProperties(t *testing.T) {
	mermaids := NewMermaids()

	if mermaids.GetType() != models.FactionMermaids {
		t.Errorf("expected faction type Mermaids, got %v", mermaids.GetType())
	}

	if mermaids.GetHomeTerrain() != models.TerrainLake {
		t.Errorf("expected home terrain Lake, got %v", mermaids.GetHomeTerrain())
	}
}

func TestMermaids_StartingResources(t *testing.T) {
	mermaids := NewMermaids()
	resources := mermaids.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
	if resources.Power1 != 3 {
		t.Errorf("expected 3 power in bowl 1 (not standard 5), got %d", resources.Power1)
	}
	if resources.Power2 != 9 {
		t.Errorf("expected 9 power in bowl 2 (not standard 7), got %d", resources.Power2)
	}
	if resources.Power3 != 0 {
		t.Errorf("expected 0 power in bowl 3, got %d", resources.Power3)
	}
}

func TestMermaids_StartWithShippingLevel1(t *testing.T) {
	mermaids := NewMermaids()

	// Mermaids start with Shipping level 1 (not 0)
	shippingLevel := mermaids.GetShippingLevel()
	if shippingLevel != 1 {
		t.Errorf("expected starting shipping level 1, got %d", shippingLevel)
	}
}

func TestMermaids_MaxShippingLevel(t *testing.T) {
	mermaids := NewMermaids()

	// Mermaids can reach Shipping level 5 (not standard max of 3)
	maxShipping := mermaids.GetMaxShippingLevel()
	if maxShipping != 5 {
		t.Errorf("expected max shipping level 5, got %d", maxShipping)
	}
}

func TestMermaids_ExpensiveSanctuary(t *testing.T) {
	mermaids := NewMermaids()

	// Mermaids sanctuary costs 8 coins (more expensive than standard 6)
	sanctuaryCost := mermaids.GetSanctuaryCost()
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected sanctuary to cost 8 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestMermaids_BuildStrongholdGrantsFreeShipping(t *testing.T) {
	mermaids := NewMermaids()

	// Building stronghold should grant free shipping upgrade
	shouldGrantShipping := mermaids.BuildStronghold()
	if !shouldGrantShipping {
		t.Errorf("building stronghold should grant free shipping upgrade")
	}
}

func TestMermaids_CanSkipRiverForTown(t *testing.T) {
	mermaids := NewMermaids()

	// Mermaids can always skip river when founding town
	if !mermaids.CanSkipRiverForTown() {
		t.Errorf("Mermaids should be able to skip river for town")
	}
}

func TestMermaids_ShippingLevelManagement(t *testing.T) {
	mermaids := NewMermaids()

	// Start at level 1
	if mermaids.GetShippingLevel() != 1 {
		t.Errorf("expected starting level 1, got %d", mermaids.GetShippingLevel())
	}

	// Set to level 3
	mermaids.SetShippingLevel(3)
	if mermaids.GetShippingLevel() != 3 {
		t.Errorf("expected level 3 after setting, got %d", mermaids.GetShippingLevel())
	}

	// Set to level 5 (max for Mermaids)
	mermaids.SetShippingLevel(5)
	if mermaids.GetShippingLevel() != 5 {
		t.Errorf("expected level 5 after setting, got %d", mermaids.GetShippingLevel())
	}
}

func TestMermaids_StandardCosts(t *testing.T) {
	mermaids := NewMermaids()

	// Mermaids use standard costs for most buildings
	dwellingCost := mermaids.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := mermaids.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
