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

func TestWitches_HasTownBonusAbility(t *testing.T) {
	witches := NewWitches()

	if !witches.HasSpecialAbility(AbilityTownBonus) {
		t.Errorf("Witches should have town bonus ability")
	}

	if witches.HasSpecialAbility(AbilityFlying) {
		t.Errorf("Witches should not have flying ability (that's a different mechanic)")
	}
}

func TestWitches_TownFoundingBonus(t *testing.T) {
	witches := NewWitches()

	bonus := witches.GetTownFoundingBonus()
	if bonus != 5 {
		t.Errorf("expected 5 VP bonus for founding town, got %d", bonus)
	}
}

func TestWitches_StrongholdAbility(t *testing.T) {
	witches := NewWitches()

	ability := witches.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Witches should have a stronghold ability")
	}
}

func TestWitches_WitchesRideBeforeStronghold(t *testing.T) {
	witches := NewWitches()

	// Should not be able to use Witches' Ride before building stronghold
	if witches.CanUseWitchesRide() {
		t.Errorf("should not be able to use Witches' Ride before building stronghold")
	}

	err := witches.UseWitchesRide()
	if err == nil {
		t.Errorf("expected error when using Witches' Ride without stronghold")
	}
}

func TestWitches_WitchesRideAfterStronghold(t *testing.T) {
	witches := NewWitches()

	// Build stronghold
	witches.BuildStronghold()

	// Should be able to use Witches' Ride
	if !witches.CanUseWitchesRide() {
		t.Errorf("should be able to use Witches' Ride after building stronghold")
	}

	// Use Witches' Ride
	err := witches.UseWitchesRide()
	if err != nil {
		t.Fatalf("failed to use Witches' Ride: %v", err)
	}

	// Should not be able to use again this Action phase
	if witches.CanUseWitchesRide() {
		t.Errorf("should not be able to use Witches' Ride twice in one Action phase")
	}

	// Try to use again (should fail)
	err = witches.UseWitchesRide()
	if err == nil {
		t.Errorf("expected error when using Witches' Ride twice")
	}
}

func TestWitches_WitchesRideReset(t *testing.T) {
	witches := NewWitches()
	witches.BuildStronghold()

	// Use Witches' Ride
	err := witches.UseWitchesRide()
	if err != nil {
		t.Fatalf("failed to use Witches' Ride: %v", err)
	}

	// Should not be able to use again
	if witches.CanUseWitchesRide() {
		t.Errorf("should not be able to use Witches' Ride before reset")
	}

	// Reset for new Action phase
	witches.ResetWitchesRide()

	// Should be able to use again
	if !witches.CanUseWitchesRide() {
		t.Errorf("should be able to use Witches' Ride after reset")
	}
}

func TestWitches_WitchesRideCost(t *testing.T) {
	witches := NewWitches()

	// Witches' Ride dwelling is free (no worker, no coins)
	cost := witches.GetWitchesRideCost()
	if cost.Workers != 0 {
		t.Errorf("expected 0 workers for Witches' Ride, got %d", cost.Workers)
	}
	if cost.Coins != 0 {
		t.Errorf("expected 0 coins for Witches' Ride, got %d", cost.Coins)
	}
}

func TestWitches_StandardCosts(t *testing.T) {
	witches := NewWitches()

	// Witches use standard building costs for normal builds
	dwellingCost := witches.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 0 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := witches.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
