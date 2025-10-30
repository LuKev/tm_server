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

func TestNomads_StrongholdAbility(t *testing.T) {
	nomads := NewNomads()

	ability := nomads.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Nomads should have a stronghold ability")
	}
}

func TestNomads_SandstormBeforeStronghold(t *testing.T) {
	nomads := NewNomads()

	// Should not be able to use sandstorm before building stronghold
	if nomads.CanUseSandstorm() {
		t.Errorf("should not be able to use sandstorm before building stronghold")
	}

	err := nomads.UseSandstorm()
	if err == nil {
		t.Errorf("expected error when using sandstorm without stronghold")
	}
}

func TestNomads_SandstormAfterStronghold(t *testing.T) {
	nomads := NewNomads()

	// Build stronghold
	nomads.BuildStronghold()

	// Should be able to use sandstorm
	if !nomads.CanUseSandstorm() {
		t.Errorf("should be able to use sandstorm after building stronghold")
	}

	// Use sandstorm
	err := nomads.UseSandstorm()
	if err != nil {
		t.Fatalf("failed to use sandstorm: %v", err)
	}

	// Should not be able to use again this Action phase
	if nomads.CanUseSandstorm() {
		t.Errorf("should not be able to use sandstorm twice in one Action phase")
	}

	// Try to use again (should fail)
	err = nomads.UseSandstorm()
	if err == nil {
		t.Errorf("expected error when using sandstorm twice")
	}
}

func TestNomads_SandstormReset(t *testing.T) {
	nomads := NewNomads()
	nomads.BuildStronghold()

	// Use sandstorm
	err := nomads.UseSandstorm()
	if err != nil {
		t.Fatalf("failed to use sandstorm: %v", err)
	}

	// Should not be able to use again
	if nomads.CanUseSandstorm() {
		t.Errorf("should not be able to use sandstorm before reset")
	}

	// Reset for new Action phase
	nomads.ResetSandstorm()

	// Should be able to use again
	if !nomads.CanUseSandstorm() {
		t.Errorf("should be able to use sandstorm after reset")
	}
}

func TestNomads_StartsWithThreeDwellings(t *testing.T) {
	nomads := NewNomads()

	// Nomads start with 3 dwellings (not 2)
	if !nomads.StartsWithThreeDwellings() {
		t.Errorf("Nomads should start with 3 dwellings")
	}
}

func TestNomads_PlacesThirdDwellingAfterSecondRound(t *testing.T) {
	nomads := NewNomads()

	// Nomads place third dwelling after all players place their second
	if !nomads.PlacesThirdDwellingAfterSecondRound() {
		t.Errorf("Nomads should place third dwelling after second round")
	}
}

func TestNomads_SandstormIsNotASpade(t *testing.T) {
	nomads := NewNomads()

	// Sandstorm is NOT considered a Spade
	if nomads.IsSandstormASpade() {
		t.Errorf("Sandstorm should not be considered a Spade")
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
	if strongholdCost.Workers != 4 || strongholdCost.Coins != 6 {
		t.Errorf("unexpected stronghold cost: %+v", strongholdCost)
	}
}
