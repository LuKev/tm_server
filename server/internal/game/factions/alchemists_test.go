package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAlchemists_BasicProperties(t *testing.T) {
	alchemists := NewAlchemists()

	if alchemists.GetType() != models.FactionAlchemists {
		t.Errorf("expected faction type Alchemists, got %v", alchemists.GetType())
	}

	if alchemists.GetHomeTerrain() != models.TerrainSwamp {
		t.Errorf("expected home terrain Swamp, got %v", alchemists.GetHomeTerrain())
	}
}

func TestAlchemists_StartingResources(t *testing.T) {
	alchemists := NewAlchemists()
	resources := alchemists.GetStartingResources()

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

func TestAlchemists_HasConversionAbility(t *testing.T) {
	alchemists := NewAlchemists()

	if !alchemists.HasSpecialAbility(AbilityConversionEfficiency) {
		t.Errorf("Alchemists should have conversion efficiency ability")
	}
}

func TestAlchemists_StrongholdAbility(t *testing.T) {
	alchemists := NewAlchemists()

	ability := alchemists.GetStrongholdAbility()
	if ability == "" {
		t.Errorf("Alchemists should have a stronghold ability")
	}
}

func TestAlchemists_BuildStrongholdGrantsPower(t *testing.T) {
	alchemists := NewAlchemists()

	// First time building stronghold should grant 12 power
	powerBonus := alchemists.BuildStronghold()
	if powerBonus != 12 {
		t.Errorf("expected 12 power bonus, got %d", powerBonus)
	}

	// Building again (hypothetically) should not grant more power
	powerBonus = alchemists.BuildStronghold()
	if powerBonus != 0 {
		t.Errorf("should only grant power bonus once, got %d", powerBonus)
	}
}

func TestAlchemists_PowerPerSpadeBeforeStronghold(t *testing.T) {
	alchemists := NewAlchemists()

	// Before stronghold, no bonus power per spade
	powerPerSpade := alchemists.GetPowerPerSpade()
	if powerPerSpade != 0 {
		t.Errorf("expected 0 power per spade before stronghold, got %d", powerPerSpade)
	}
}

func TestAlchemists_PowerPerSpadeAfterStronghold(t *testing.T) {
	alchemists := NewAlchemists()

	// Build stronghold
	alchemists.BuildStronghold()

	// After stronghold, gain 2 power per spade
	powerPerSpade := alchemists.GetPowerPerSpade()
	if powerPerSpade != 2 {
		t.Errorf("expected 2 power per spade after stronghold, got %d", powerPerSpade)
	}
}

func TestAlchemists_ConvertVPToCoins(t *testing.T) {
	alchemists := NewAlchemists()

	tests := []struct {
		name     string
		vp       int
		expected int
		wantErr  bool
	}{
		{"1 VP to 1 Coin", 1, 1, false},
		{"5 VP to 5 Coins", 5, 5, false},
		{"10 VP to 10 Coins", 10, 10, false},
		{"0 VP invalid", 0, 0, true},
		{"Negative VP invalid", -1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coins, err := alchemists.ConvertVPToCoins(tt.vp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertVPToCoins(%d) error = %v, wantErr %v", tt.vp, err, tt.wantErr)
				return
			}
			if coins != tt.expected {
				t.Errorf("ConvertVPToCoins(%d) = %d, want %d", tt.vp, coins, tt.expected)
			}
		})
	}
}

func TestAlchemists_ConvertCoinsToVP(t *testing.T) {
	alchemists := NewAlchemists()

	tests := []struct {
		name     string
		coins    int
		expected int
		wantErr  bool
	}{
		{"2 Coins to 1 VP", 2, 1, false},
		{"4 Coins to 2 VP", 4, 2, false},
		{"10 Coins to 5 VP", 10, 5, false},
		{"1 Coin invalid (odd)", 1, 0, true},
		{"3 Coins invalid (odd)", 3, 0, true},
		{"0 Coins invalid", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vp, err := alchemists.ConvertCoinsToVP(tt.coins)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertCoinsToVP(%d) error = %v, wantErr %v", tt.coins, err, tt.wantErr)
				return
			}
			if vp != tt.expected {
				t.Errorf("ConvertCoinsToVP(%d) = %d, want %d", tt.coins, vp, tt.expected)
			}
		})
	}
}

func TestAlchemists_StandardCosts(t *testing.T) {
	alchemists := NewAlchemists()

	// Alchemists use standard building costs
	dwellingCost := alchemists.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 0 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	sanctuaryCost := alchemists.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}
}
