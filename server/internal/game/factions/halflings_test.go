package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestHalflings_BasicProperties(t *testing.T) {
	halflings := NewHalflings()

	if halflings.GetType() != models.FactionHalflings {
		t.Errorf("expected faction type Halflings, got %v", halflings.GetType())
	}

	if halflings.GetHomeTerrain() != models.TerrainPlains {
		t.Errorf("expected home terrain Plains, got %v", halflings.GetHomeTerrain())
	}
}

func TestHalflings_StartingResources(t *testing.T) {
	halflings := NewHalflings()
	resources := halflings.GetStartingResources()

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

func TestHalflings_HasSpadeEfficiencyAbility(t *testing.T) {
	halflings := NewHalflings()

	if !halflings.HasSpecialAbility(AbilitySpadeEfficiency) {
		t.Errorf("Halflings should have spade efficiency ability")
	}
}

func TestHalflings_CheaperDiggingCost(t *testing.T) {
	halflings := NewHalflings()

	// Halflings digging costs 2 workers, 1 coin, 1 priest (cheaper than standard)
	diggingCost := halflings.GetDiggingCost(0)
	if diggingCost.Workers != 2 {
		t.Errorf("expected 2 workers for digging, got %d", diggingCost.Workers)
	}
	if diggingCost.Coins != 1 {
		t.Errorf("expected 1 coin for digging (cheaper than standard 5), got %d", diggingCost.Coins)
	}
	if diggingCost.Priests != 1 {
		t.Errorf("expected 1 priest for digging, got %d", diggingCost.Priests)
	}
}

func TestHalflings_ExpensiveStronghold(t *testing.T) {
	halflings := NewHalflings()

	// Halflings stronghold costs 8 coins (more expensive than standard 6)
	strongholdCost := halflings.GetStrongholdCost()
	if strongholdCost.Coins != 8 {
		t.Errorf("expected stronghold to cost 8 coins, got %d", strongholdCost.Coins)
	}
	if strongholdCost.Workers != 4 {
		t.Errorf("expected stronghold to cost 4 workers, got %d", strongholdCost.Workers)
	}
}

func TestHalflings_StrongholdAbility(t *testing.T) {
	halflings := NewHalflings()

	ability := halflings.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Halflings should have a stronghold ability")
	}
}

func TestHalflings_StrongholdSpadesBeforeBuilding(t *testing.T) {
	halflings := NewHalflings()

	// Should not be able to use stronghold spades before building stronghold
	if halflings.CanUseStrongholdSpades() {
		t.Errorf("should not be able to use stronghold spades before building stronghold")
	}

	spades := halflings.UseStrongholdSpades()
	if spades != 0 {
		t.Errorf("expected 0 spades before stronghold, got %d", spades)
	}
}

func TestHalflings_StrongholdSpadesAfterBuilding(t *testing.T) {
	halflings := NewHalflings()

	// Build stronghold
	halflings.BuildStronghold()

	// Should be able to use stronghold spades
	if !halflings.CanUseStrongholdSpades() {
		t.Errorf("should be able to use stronghold spades after building stronghold")
	}

	// Use stronghold spades
	spades := halflings.UseStrongholdSpades()
	if spades != 3 {
		t.Errorf("expected 3 spades from stronghold, got %d", spades)
	}

	// Should not be able to use again
	if halflings.CanUseStrongholdSpades() {
		t.Errorf("should not be able to use stronghold spades twice")
	}

	// Try to use again (should return 0)
	spades = halflings.UseStrongholdSpades()
	if spades != 0 {
		t.Errorf("expected 0 spades on second use, got %d", spades)
	}
}

func TestHalflings_VPPerSpade(t *testing.T) {
	halflings := NewHalflings()

	// Halflings get +1 VP per spade
	vpPerSpade := halflings.GetVPPerSpade()
	if vpPerSpade != 1 {
		t.Errorf("expected 1 VP per spade, got %d", vpPerSpade)
	}
}

func TestHalflings_StandardCosts(t *testing.T) {
	halflings := NewHalflings()

	// Halflings use standard costs for most buildings
	dwellingCost := halflings.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 0 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	sanctuaryCost := halflings.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}
}
