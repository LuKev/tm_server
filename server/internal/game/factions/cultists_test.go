package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestCultists_BasicProperties(t *testing.T) {
	cultists := NewCultists()

	if cultists.GetType() != models.FactionCultists {
		t.Errorf("expected faction type Cultists, got %v", cultists.GetType())
	}

	if cultists.GetHomeTerrain() != models.TerrainPlains {
		t.Errorf("expected home terrain Plains, got %v", cultists.GetHomeTerrain())
	}
}

func TestCultists_StartingResources(t *testing.T) {
	cultists := NewCultists()
	resources := cultists.GetStartingResources()

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

func TestCultists_HasCultTrackBonusAbility(t *testing.T) {
	cultists := NewCultists()

	if !cultists.HasSpecialAbility(AbilityCultTrackBonus) {
		t.Errorf("Cultists should have cult track bonus ability")
	}
}

func TestCultists_ExpensiveSanctuary(t *testing.T) {
	cultists := NewCultists()

	// Cultists sanctuary costs 8 coins (more expensive than standard 6)
	sanctuaryCost := cultists.GetSanctuaryCost()
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected sanctuary to cost 8 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestCultists_ExpensiveStronghold(t *testing.T) {
	cultists := NewCultists()

	// Cultists stronghold costs 8 coins (more expensive than standard 6)
	strongholdCost := cultists.GetStrongholdCost()
	if strongholdCost.Coins != 8 {
		t.Errorf("expected stronghold to cost 8 coins, got %d", strongholdCost.Coins)
	}
	if strongholdCost.Workers != 4 {
		t.Errorf("expected stronghold to cost 4 workers, got %d", strongholdCost.Workers)
	}
}


func TestCultists_BuildStrongholdGrantsVP(t *testing.T) {
	cultists := NewCultists()

	// Building stronghold should grant 7 VP
	vpBonus := cultists.BuildStronghold()
	if vpBonus != 7 {
		t.Errorf("expected 7 VP bonus, got %d", vpBonus)
	}
}

func TestCultists_CultAdvanceFromPowerLeech(t *testing.T) {
	cultists := NewCultists()

	// Cultists advance 1 space on cult track when opponents take power
	cultAdvance := cultists.GetCultAdvanceFromPowerLeech()
	if cultAdvance != 1 {
		t.Errorf("expected 1 cult space advance, got %d", cultAdvance)
	}
}

func TestCultists_PowerIfAllRefuse(t *testing.T) {
	cultists := NewCultists()

	// Cultists gain 1 power if all opponents refuse power
	power := cultists.GetPowerIfAllRefuse()
	if power != 1 {
		t.Errorf("expected 1 power if all refuse, got %d", power)
	}
}

func TestCultists_StandardCosts(t *testing.T) {
	cultists := NewCultists()

	// Cultists use standard costs for most buildings
	dwellingCost := cultists.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := cultists.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
