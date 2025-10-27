package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Cultists faction - Brown/Plains
// Ability: When at least one opponent takes Power from your building, advance 1 space on a Cult track
//          (Only 1 space total regardless of number of opponents)
//          If all opponents refuse Power, gain 1 Power instead
//          If no opponents can take Power, gain nothing
// Stronghold: After building, immediately and only once get 7 VP
// Special: Sanctuary costs 4 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
//          Stronghold costs 4 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
type Cultists struct {
	BaseFaction
	hasStronghold              bool
	hasReceivedStrongholdVP    bool // One-time 7 VP bonus
}

func NewCultists() *Cultists {
	return &Cultists{
		BaseFaction: BaseFaction{
			Type:        models.FactionCultists,
			HomeTerrain: models.TerrainPlains,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  5,
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:           false,
		hasReceivedStrongholdVP: false,
	}
}

// GetStartingCultPositions returns Cultists starting cult track positions
func (f *Cultists) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 0, Earth: 1, Air: 0}
}

// GetSanctuaryCost returns the expensive sanctuary cost for Cultists
func (f *Cultists) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// GetStrongholdCost returns the expensive stronghold cost for Cultists
func (f *Cultists) GetStrongholdCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for cult track bonuses
func (f *Cultists) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityCultTrackBonus
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Cultists) GetStrongholdAbility() string {
	return "Immediately and only once: Get 7 Victory points"
}

// BuildStronghold marks that the stronghold has been built
// Returns the one-time VP bonus (7 VP)
// NOTE: Phase 8 (Scoring System) tracks VP
func (f *Cultists) BuildStronghold() int {
	f.hasStronghold = true
	
	// Return one-time VP bonus
	if !f.hasReceivedStrongholdVP {
		f.hasReceivedStrongholdVP = true
		return 7 // Grant 7 VP
	}
	return 0
}

// GetCultAdvanceFromPowerLeech returns how many cult spaces to advance
// when opponents take power from Cultists' building
// NOTE: Phase 5.1 (Power System) implements power leech
// NOTE: Phase 7.1 (Cult Track System) handles cult advancement
// NOTE: Phase 6.2 (Action System) must trigger this when building
func (f *Cultists) GetCultAdvanceFromPowerLeech() int {
	return 1 // Advance 1 space on cult track (if at least one opponent takes power)
}

// GetPowerIfAllRefuse returns how much power to gain if all opponents refuse power
// NOTE: Phase 5.1 (Power System) implements this
func (f *Cultists) GetPowerIfAllRefuse() int {
	return 1 // Gain 1 power if all opponents refuse
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Cultists) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is passive (one-time VP bonus)
	// The VP bonus is handled in BuildStronghold()
	return nil
}
