package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Auren faction - Green/Forest
// Ability: None (no passive ability)
// Stronghold: After building, immediately get 1 Favor tile (once)
//             Special action (once per Action phase): Advance 2 spaces on a Cult track
//             (only advancing to space 10 if you have a key)
// Special: Sanctuary costs 4 workers/8 coins (more expensive than standard)
type Auren struct {
	BaseFaction
	hasStronghold              bool
	hasReceivedFavorTile       bool // One-time bonus when stronghold is built
	cultAdvanceUsedThisRound   bool // Special action usage tracking
}

func NewAuren() *Auren {
	return &Auren{
		BaseFaction: BaseFaction{
			Type:        models.FactionAuren,
			HomeTerrain: models.TerrainForest,
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
		hasStronghold:            false,
		hasReceivedFavorTile:     false,
		cultAdvanceUsedThisRound: false,
	}
}

// GetSanctuaryCost returns the Auren's expensive sanctuary cost
func (f *Auren) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility - Auren has no passive ability
func (f *Auren) HasSpecialAbility(ability SpecialAbility) bool {
	return false
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Auren) GetStrongholdAbility() string {
	return "On building: Get 1 Favor tile (once). Special action (once per Action phase): Advance 2 spaces on a Cult track (only to space 10 if you have a key)"
}

// BuildStronghold marks that the stronghold has been built
// Returns true if the player should receive a favor tile
// NOTE: Phase 7.2 (Favor Tiles) handles favor tile selection
func (f *Auren) BuildStronghold() bool {
	f.hasStronghold = true
	
	// Return whether favor tile should be granted
	// (only once, when stronghold is first built)
	if !f.hasReceivedFavorTile {
		f.hasReceivedFavorTile = true
		return true
	}
	return false
}

// CanUseCultAdvance checks if the cult advance special action can be used
func (f *Auren) CanUseCultAdvance() bool {
	return f.hasStronghold && !f.cultAdvanceUsedThisRound
}

// UseCultAdvance marks the cult advance special action as used
// NOTE: Full validation (cult track selection, key requirement for space 10, etc.) will be
// implemented in Phase 6.2 (Action System) as part of AurenCultAdvanceAction
// NOTE: Phase 7.1 (Cult Track System) handles cult track advancement
func (f *Auren) UseCultAdvance() error {
	if !f.hasStronghold {
		return fmt.Errorf("must build stronghold before using cult advance")
	}
	
	if f.cultAdvanceUsedThisRound {
		return fmt.Errorf("cult advance already used this Action phase")
	}
	
	f.cultAdvanceUsedThisRound = true
	return nil
}

// ResetCultAdvance resets the cult advance for a new Action phase
func (f *Auren) ResetCultAdvance() {
	f.cultAdvanceUsedThisRound = false
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Auren) ExecuteStrongholdAbility(gameState interface{}) error {
	return f.UseCultAdvance()
}

// GetCultAdvanceAmount returns how many spaces to advance on cult track
// NOTE: Phase 7.1 (Cult Track System) uses this value
func (f *Auren) GetCultAdvanceAmount() int {
	return 2
}
