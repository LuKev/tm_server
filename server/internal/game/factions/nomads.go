package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Nomads faction - Yellow/Desert
// Ability: Start with 3 Dwellings instead of 2
//          Place third Dwelling after all players have placed their second ones (but before Chaos Magicians)
// Stronghold: After building, take an Action token
//             Special action (once per Action phase): Transform a Terrain space directly adjacent to one of your Structures
//             into your Home terrain (Sandstorm). May immediately build a Dwelling on that space by paying its costs.
//             (Not applicable past a River space or Bridge. Sandstorm is not considered a Spade.)
// Special: Start with 2 workers, 15 coins (not standard 3 workers, 15 coins)
type Nomads struct {
	BaseFaction
	hasStronghold           bool
	sandstormUsedThisRound  bool // Special action usage tracking
}

func NewNomads() *Nomads {
	return &Nomads{
		BaseFaction: BaseFaction{
			Type:        models.FactionNomads,
			HomeTerrain: models.TerrainDesert,
			StartingRes: Resources{
				Coins:   15,
				Workers: 2, // Nomads start with 2 workers (not standard 3)
				Priests: 0,
				Power1:  5, // Standard 5/7 power
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:          false,
		sandstormUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Nomads starting cult track positions
func (f *Nomads) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 0, Earth: 1, Air: 0}
}

// HasSpecialAbility returns true for sandstorm
func (f *Nomads) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilitySandstorm
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Nomads) GetStrongholdAbility() string {
	return "Special action (once per Action phase): Transform a Terrain space directly adjacent to one of your Structures into your Home terrain (Sandstorm). May immediately build a Dwelling on that space by paying its costs. (Not applicable past River/Bridge. Sandstorm is not a Spade.)"
}

// BuildStronghold marks that the stronghold has been built
func (f *Nomads) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseSandstorm checks if the sandstorm special action can be used
func (f *Nomads) CanUseSandstorm() bool {
	return f.hasStronghold && !f.sandstormUsedThisRound
}

// UseSandstorm marks the sandstorm special action as used
// NOTE: Phase 6.2 (Action System) implements the actual sandstorm logic
// NOTE: Sandstorm is NOT considered a Spade (no VP bonuses for spade-related abilities)
func (f *Nomads) UseSandstorm() error {
	if !f.hasStronghold {
		return fmt.Errorf("must build stronghold before using sandstorm")
	}
	
	if f.sandstormUsedThisRound {
		return fmt.Errorf("sandstorm already used this Action phase")
	}
	
	f.sandstormUsedThisRound = true
	return nil
}

// ResetSandstorm resets the sandstorm for a new Action phase
func (f *Nomads) ResetSandstorm() {
	f.sandstormUsedThisRound = false
}

// StartsWithThreeDwellings returns true - Nomads start with 3 dwellings
// NOTE: Game setup (Phase 1) uses this
func (f *Nomads) StartsWithThreeDwellings() bool {
	return true
}

// PlacesThirdDwellingAfterSecondRound returns true - Nomads place third dwelling after all players place their second
// NOTE: Game setup (Phase 1) uses this
func (f *Nomads) PlacesThirdDwellingAfterSecondRound() bool {
	return true
}

// IsSandstormASpade returns false - Sandstorm is NOT considered a Spade
// This means no VP bonuses from spade-related abilities (e.g., Halflings, Alchemists)
func (f *Nomads) IsSandstormASpade() bool {
	return false
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Nomads) ExecuteStrongholdAbility(gameState interface{}) error {
	return f.UseSandstorm()
}
