package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestEngineers_BasicProperties(t *testing.T) {
	engineers := NewEngineers()

	if engineers.GetType() != models.FactionEngineers {
		t.Errorf("expected faction type Engineers, got %v", engineers.GetType())
	}

	if engineers.GetHomeTerrain() != models.TerrainMountain {
		t.Errorf("expected home terrain Mountain, got %v", engineers.GetHomeTerrain())
	}
}

func TestEngineers_StartingResources(t *testing.T) {
	engineers := NewEngineers()
	resources := engineers.GetStartingResources()

	if resources.Coins != 10 {
		t.Errorf("expected 10 coins (not standard 15), got %d", resources.Coins)
	}
	if resources.Workers != 2 {
		t.Errorf("expected 2 workers (not standard 3), got %d", resources.Workers)
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

func TestEngineers_HasBridgeBuildingAbility(t *testing.T) {
	engineers := NewEngineers()

	if !engineers.HasSpecialAbility(AbilityBridgeBuilding) {
		t.Errorf("Engineers should have bridge building ability")
	}
}

func TestEngineers_CheaperDwellingCost(t *testing.T) {
	engineers := NewEngineers()

	// Engineers dwelling costs 1 worker, 1 coin
	dwellingCost := engineers.GetDwellingCost()
	if dwellingCost.Workers != 1 {
		t.Errorf("expected 1 worker for dwelling, got %d", dwellingCost.Workers)
	}
	if dwellingCost.Coins != 1 {
		t.Errorf("expected 1 coin for dwelling, got %d", dwellingCost.Coins)
	}
}

func TestEngineers_CheaperTradingHouseCost(t *testing.T) {
	engineers := NewEngineers()

	// Engineers trading house costs 1 worker, 4 coins (cheaper than standard 2 workers, 6 coins)
	tpCost := engineers.GetTradingHouseCost()
	if tpCost.Workers != 1 {
		t.Errorf("expected 1 worker for trading house, got %d", tpCost.Workers)
	}
	if tpCost.Coins != 4 {
		t.Errorf("expected 4 coins for trading house, got %d", tpCost.Coins)
	}
}

func TestEngineers_CheaperTempleCost(t *testing.T) {
	engineers := NewEngineers()

	// Engineers temple costs 1 worker, 4 coins (cheaper than standard 2 workers, 5 coins)
	templeCost := engineers.GetTempleCost()
	if templeCost.Workers != 1 {
		t.Errorf("expected 1 worker for temple, got %d", templeCost.Workers)
	}
	if templeCost.Coins != 4 {
		t.Errorf("expected 4 coins for temple, got %d", templeCost.Coins)
	}
}

func TestEngineers_CheaperSanctuaryCost(t *testing.T) {
	engineers := NewEngineers()

	// Engineers sanctuary costs 3 workers, 6 coins (cheaper than standard 4 workers, 6 coins)
	sanctuaryCost := engineers.GetSanctuaryCost()
	if sanctuaryCost.Workers != 3 {
		t.Errorf("expected 3 workers for sanctuary, got %d", sanctuaryCost.Workers)
	}
	if sanctuaryCost.Coins != 6 {
		t.Errorf("expected 6 coins for sanctuary, got %d", sanctuaryCost.Coins)
	}
}

func TestEngineers_CheaperStrongholdCost(t *testing.T) {
	engineers := NewEngineers()

	// Engineers stronghold costs 3 workers, 6 coins (cheaper than standard 4 workers, 6 coins)
	strongholdCost := engineers.GetStrongholdCost()
	if strongholdCost.Workers != 3 {
		t.Errorf("expected 3 workers for stronghold, got %d", strongholdCost.Workers)
	}
	if strongholdCost.Coins != 6 {
		t.Errorf("expected 6 coins for stronghold, got %d", strongholdCost.Coins)
	}
}

func TestEngineers_VPPerBridgeBeforeStronghold(t *testing.T) {
	engineers := NewEngineers()

	// Before stronghold, no VP bonus
	vpPerBridge := engineers.GetVPPerBridgeOnPass()
	if vpPerBridge != 0 {
		t.Errorf("expected 0 VP per bridge before stronghold, got %d", vpPerBridge)
	}
}

func TestEngineers_VPPerBridgeAfterStronghold(t *testing.T) {
	engineers := NewEngineers()

	// Build stronghold
	engineers.BuildStronghold()

	// After stronghold, 3 VP per bridge
	vpPerBridge := engineers.GetVPPerBridgeOnPass()
	if vpPerBridge != 3 {
		t.Errorf("expected 3 VP per bridge after stronghold, got %d", vpPerBridge)
	}
}

func TestEngineers_HasStronghold(t *testing.T) {
	engineers := NewEngineers()

	// Before building
	if engineers.HasStronghold() {
		t.Errorf("should not have stronghold before building")
	}

	// After building
	engineers.BuildStronghold()
	if !engineers.HasStronghold() {
		t.Errorf("should have stronghold after building")
	}
}
