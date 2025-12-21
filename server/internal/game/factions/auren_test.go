package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAuren_BasicProperties(t *testing.T) {
	auren := NewAuren()

	if auren.GetType() != models.FactionAuren {
		t.Errorf("expected faction type Auren, got %v", auren.GetType())
	}

	if auren.GetHomeTerrain() != models.TerrainForest {
		t.Errorf("expected home terrain Forest, got %v", auren.GetHomeTerrain())
	}
}

func TestAuren_StartingResources(t *testing.T) {
	auren := NewAuren()
	resources := auren.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
	if resources.Power1 != 5 {
		t.Errorf("expected 5 power in bowl 1, got %d", resources.Power1)
	}
	if resources.Power2 != 7 {
		t.Errorf("expected 7 power in bowl 2, got %d", resources.Power2)
	}
	if resources.Power3 != 0 {
		t.Errorf("expected 0 power in bowl 3, got %d", resources.Power3)
	}
}

func TestAuren_ExpensiveSanctuary(t *testing.T) {
	auren := NewAuren()

	// Auren sanctuary costs 8 coins (more expensive than standard 6)
	sanctuaryCost := auren.GetSanctuaryCost()
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected sanctuary to cost 8 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestAuren_BuildStrongholdGrantsFavorTile(t *testing.T) {
	auren := NewAuren()

	// Building stronghold should grant favor tile
	shouldGrantFavor := auren.BuildStronghold()
	if !shouldGrantFavor {
		t.Errorf("building stronghold should grant favor tile")
	}
}

func TestAuren_StandardCosts(t *testing.T) {
	auren := NewAuren()

	// Auren uses standard costs for most buildings
	dwellingCost := auren.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := auren.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	templeCost := auren.GetTempleCost()
	if templeCost.Workers != 2 || templeCost.Coins != 5 {
		t.Errorf("unexpected temple cost: %+v", templeCost)
	}
}
