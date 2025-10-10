package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Giants faction - Red/Wasteland
// Ability: Always pay exactly 2 Spades to transform terrain (even for Mountains and Desert)
//          (A single Spade will be forfeit when gained in Phase III as a Cult bonus)
// Stronghold: After building, take an Action token
//             Special action (once per Action phase): Get 2 free Spades to transform a reachable space
//             May immediately build a Dwelling on that space by paying its costs
// Special: All standard building costs
type Giants struct {
	BaseFaction
	hasStronghold              bool
	freeSpadesUsedThisRound    bool // Special action usage tracking
}

func NewGiants() *Giants {
	return &Giants{
		BaseFaction: BaseFaction{
			Type:        models.FactionGiants,
			HomeTerrain: models.TerrainWasteland,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  5, // Standard 5/7 power
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:           false,
		freeSpadesUsedThisRound: false,
	}
}

// GetTerraformCost overrides the base method
// Giants always pay exactly 2 spades (regardless of terrain distance)
// NOTE: Phase 6.2 (Action System) must use this for Giants
func (f *Giants) GetTerraformCost(distance int) int {
	// Giants always pay 2 spades, regardless of distance
	// This is in terms of workers needed
	// Base terraform cost is 3 workers per spade, reduced by digging level
	workersPerSpade := 3 - f.DiggingLevel
	if workersPerSpade < 1 {
		workersPerSpade = 1
	}
	return 2 * workersPerSpade // Always 2 spades worth of workers
}

// GetTerraformSpades returns the number of spades needed for Giants
// Giants always need exactly 2 spades
func (f *Giants) GetTerraformSpades() int {
	return 2 // Always 2 spades, regardless of terrain distance
}

// HasSpecialAbility returns true for spade efficiency (fixed 2 spades)
func (f *Giants) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilitySpadeEfficiency
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Giants) GetStrongholdAbility() string {
	return "Special action (once per Action phase): Get 2 free Spades to transform a reachable space. May immediately build a Dwelling on that space by paying its costs"
}

// BuildStronghold marks that the stronghold has been built
func (f *Giants) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseFreeSpades checks if the free spades special action can be used
func (f *Giants) CanUseFreeSpades() bool {
	return f.hasStronghold && !f.freeSpadesUsedThisRound
}

// UseFreeSpades marks the free spades special action as used
// Returns the number of free spades granted (2)
// NOTE: Phase 6.2 (Action System) implements the actual terraform and optional dwelling placement
func (f *Giants) UseFreeSpades() (int, error) {
	if !f.hasStronghold {
		return 0, fmt.Errorf("must build stronghold before using free spades")
	}
	
	if f.freeSpadesUsedThisRound {
		return 0, fmt.Errorf("free spades already used this Action phase")
	}
	
	f.freeSpadesUsedThisRound = true
	return 2, nil // Grant 2 free spades
}

// ResetFreeSpades resets the free spades for a new Action phase
func (f *Giants) ResetFreeSpades() {
	f.freeSpadesUsedThisRound = false
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Giants) ExecuteStrongholdAbility(gameState interface{}) error {
	_, err := f.UseFreeSpades()
	return err
}
