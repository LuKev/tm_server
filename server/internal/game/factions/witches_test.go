package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestWitches_BasicProperties(t *testing.T) {
	witches := NewWitches()

	if witches.GetType() != models.FactionWitches {
		t.Errorf("expected faction type Witches, got %v", witches.GetType())
	}

	if witches.GetHomeTerrain() != models.TerrainForest {
		t.Errorf("expected home terrain Forest, got %v", witches.GetHomeTerrain())
	}
}

func TestWitches_StartingResources(t *testing.T) {
	witches := NewWitches()
	resources := witches.GetStartingResources()

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

func TestWitches_TownFoundingBonus(t *testing.T) {
	witches := NewWitches()

	bonus := witches.GetTownFoundingBonus()
	if bonus != 5 {
		t.Errorf("expected 5 VP bonus for founding town, got %d", bonus)
	}
}

func TestWitches_StandardCosts(t *testing.T) {
	witches := NewWitches()

	// Witches use standard building costs for normal builds
	dwellingCost := witches.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := witches.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
