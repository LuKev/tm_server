package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestNomads_BasicProperties(t *testing.T) {
	nomads := NewNomads()

	if nomads.GetType() != models.FactionNomads {
		t.Errorf("expected faction type Nomads, got %v", nomads.GetType())
	}

	if nomads.GetHomeTerrain() != models.TerrainDesert {
		t.Errorf("expected home terrain Desert, got %v", nomads.GetHomeTerrain())
	}
}

func TestNomads_StartingResources(t *testing.T) {
	nomads := NewNomads()
	resources := nomads.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 2 {
		t.Errorf("expected 2 workers (not standard 3), got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
}

func TestNomads_HasSandstormAbility(t *testing.T) {
	nomads := NewNomads()

	if !nomads.HasSpecialAbility(AbilitySandstorm) {
		t.Errorf("Nomads should have sandstorm ability")
	}
}

func TestNomads_StandardCosts(t *testing.T) {
	nomads := NewNomads()

	// Nomads use standard building costs
	dwellingCost := nomads.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := nomads.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	sanctuaryCost := nomads.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}

	strongholdCost := nomads.GetStrongholdCost()
	if strongholdCost.Workers != 4 || strongholdCost.Coins != 8 {
		t.Errorf("unexpected stronghold cost: %+v (expected 4 workers, 8 coins)", strongholdCost)
	}
}
