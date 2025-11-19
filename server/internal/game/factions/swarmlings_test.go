package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestSwarmlings_BasicProperties(t *testing.T) {
	swarmlings := NewSwarmlings()

	if swarmlings.GetType() != models.FactionSwarmlings {
		t.Errorf("expected faction type Swarmlings, got %v", swarmlings.GetType())
	}

	if swarmlings.GetHomeTerrain() != models.TerrainLake {
		t.Errorf("expected home terrain Lake, got %v", swarmlings.GetHomeTerrain())
	}
}

func TestSwarmlings_StartingResources(t *testing.T) {
	swarmlings := NewSwarmlings()
	resources := swarmlings.GetStartingResources()

	if resources.Coins != 20 {
		t.Errorf("expected 20 coins (not standard 15), got %d", resources.Coins)
	}
	if resources.Workers != 8 {
		t.Errorf("expected 8 workers (not standard 3), got %d", resources.Workers)
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

func TestSwarmlings_HasCheapDwellingsAbility(t *testing.T) {
	swarmlings := NewSwarmlings()

	if !swarmlings.HasSpecialAbility(AbilityCheapDwellings) {
		t.Errorf("Swarmlings should have cheap dwellings ability")
	}
}

func TestSwarmlings_ExpensiveDwellingCost(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings dwelling costs 2 workers, 3 coins (more expensive than standard 1 worker, 0 coins)
	dwellingCost := swarmlings.GetDwellingCost()
	if dwellingCost.Workers != 2 {
		t.Errorf("expected 2 workers for dwelling, got %d", dwellingCost.Workers)
	}
	if dwellingCost.Coins != 3 {
		t.Errorf("expected 3 coins for dwelling, got %d", dwellingCost.Coins)
	}
}

func TestSwarmlings_ExpensiveTradingHouseCost(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings trading house costs 3 workers, 8 coins (more expensive than standard 2 workers, 6 coins)
	tpCost := swarmlings.GetTradingHouseCost()
	if tpCost.Workers != 3 {
		t.Errorf("expected 3 workers for trading house, got %d", tpCost.Workers)
	}
	if tpCost.Coins != 8 {
		t.Errorf("expected 8 coins for trading house, got %d", tpCost.Coins)
	}
}

func TestSwarmlings_ExpensiveTempleCost(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings temple costs 3 workers, 6 coins (more expensive than standard 2 workers, 5 coins)
	templeCost := swarmlings.GetTempleCost()
	if templeCost.Workers != 3 {
		t.Errorf("expected 3 workers for temple, got %d", templeCost.Workers)
	}
	if templeCost.Coins != 6 {
		t.Errorf("expected 6 coins for temple, got %d", templeCost.Coins)
	}
}

func TestSwarmlings_ExpensiveSanctuaryCost(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings sanctuary costs 5 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
	sanctuaryCost := swarmlings.GetSanctuaryCost()
	if sanctuaryCost.Workers != 5 {
		t.Errorf("expected 5 workers for sanctuary, got %d", sanctuaryCost.Workers)
	}
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected 8 coins for sanctuary, got %d", sanctuaryCost.Coins)
	}
}

func TestSwarmlings_ExpensiveStrongholdCost(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings stronghold costs 5 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
	strongholdCost := swarmlings.GetStrongholdCost()
	if strongholdCost.Workers != 5 {
		t.Errorf("expected 5 workers for stronghold, got %d", strongholdCost.Workers)
	}
	if strongholdCost.Coins != 8 {
		t.Errorf("expected 8 coins for stronghold, got %d", strongholdCost.Coins)
	}
}

func TestSwarmlings_TownFoundingWorkerBonus(t *testing.T) {
	swarmlings := NewSwarmlings()

	// Swarmlings get +3 workers when founding a town
	workerBonus := swarmlings.GetTownFoundingWorkerBonus()
	if workerBonus != 3 {
		t.Errorf("expected 3 workers bonus for founding town, got %d", workerBonus)
	}
}
