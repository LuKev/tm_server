package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestDarklings_BasicProperties(t *testing.T) {
	darklings := NewDarklings()

	if darklings.GetType() != models.FactionDarklings {
		t.Errorf("expected faction type Darklings, got %v", darklings.GetType())
	}

	if darklings.GetHomeTerrain() != models.TerrainSwamp {
		t.Errorf("expected home terrain Swamp, got %v", darklings.GetHomeTerrain())
	}
}

func TestDarklings_StartingResources(t *testing.T) {
	darklings := NewDarklings()
	resources := darklings.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 1 {
		t.Errorf("expected 1 worker (not 3), got %d", resources.Workers)
	}
	if resources.Priests != 1 {
		t.Errorf("expected 1 priest (not 0), got %d", resources.Priests)
	}
}

func TestDarklings_ExpensiveSanctuary(t *testing.T) {
	darklings := NewDarklings()

	// Darklings sanctuary costs 10 coins (more expensive than standard 6)
	sanctuaryCost := darklings.GetSanctuaryCost()
	if sanctuaryCost.Coins != 10 {
		t.Errorf("expected sanctuary to cost 10 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestDarklings_PriestOrdinationBeforeStronghold(t *testing.T) {
	darklings := NewDarklings()

	// Should not be able to use priest ordination before building stronghold
	if darklings.CanUsePriestOrdination() {
		t.Errorf("should not be able to use priest ordination before building stronghold")
	}
}

func TestDarklings_PriestOrdinationAfterStronghold(t *testing.T) {
	darklings := NewDarklings()

	// Build stronghold
	darklings.BuildStronghold()

	// Should be able to use priest ordination
	if !darklings.CanUsePriestOrdination() {
		t.Errorf("should be able to use priest ordination after building stronghold")
	}

	// Trade 2 workers for 2 priests
	priests, err := darklings.UsePriestOrdination(2)
	if err != nil {
		t.Fatalf("failed to use priest ordination: %v", err)
	}
	if priests != 2 {
		t.Errorf("expected 2 priests, got %d", priests)
	}
}

func TestDarklings_PriestOrdinationLimits(t *testing.T) {
	darklings := NewDarklings()
	darklings.BuildStronghold()

	tests := []struct {
		name    string
		workers int
		wantErr bool
	}{
		{"1 worker valid", 1, false},
		{"2 workers valid", 2, false},
		{"3 workers valid", 3, false},
		{"0 workers valid (decline conversion)", 0, false},
		{"4 workers invalid", 4, true},
		{"5 workers invalid", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh instance for each test
			d := NewDarklings()
			d.BuildStronghold()

			_, err := d.UsePriestOrdination(tt.workers)
			if (err != nil) != tt.wantErr {
				t.Errorf("UsePriestOrdination(%d) error = %v, wantErr %v", tt.workers, err, tt.wantErr)
			}
		})
	}
}

func TestDarklings_TerraformCostReturnsZeroWorkers(t *testing.T) {
	darklings := NewDarklings()

	// Darklings don't use workers for terraform
	workerCost := darklings.GetTerraformCost(3)
	if workerCost != 0 {
		t.Errorf("expected 0 workers for terraform, got %d", workerCost)
	}
}

func TestDarklings_CannotUpgradeDigging(t *testing.T) {
	darklings := NewDarklings()

	// Darklings can never upgrade digging
	diggingCost := darklings.GetDiggingCost(0)
	if diggingCost.Workers != 0 || diggingCost.Coins != 0 {
		t.Errorf("Darklings should not be able to upgrade digging, got cost: %+v", diggingCost)
	}

	// Try level 1 and 2 as well
	diggingCost = darklings.GetDiggingCost(1)
	if diggingCost.Workers != 0 || diggingCost.Coins != 0 {
		t.Errorf("Darklings should not be able to upgrade digging at any level, got cost: %+v", diggingCost)
	}
}

func TestDarklings_StandardCosts(t *testing.T) {
	darklings := NewDarklings()

	// Darklings use standard costs for most buildings
	dwellingCost := darklings.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := darklings.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
