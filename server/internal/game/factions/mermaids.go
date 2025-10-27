package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Mermaids faction - Blue/Lake
// Ability: May skip one River space when founding a Town (put Town tile on skipped River space)
// Stronghold: After building, immediately and only once move forward 1 space on Shipping track
//             (no cost: neither 1 Priest nor 4 Coins)
// Special: Expensive Sanctuary (4 workers, 8 coins vs standard 4 workers, 6 coins)
//          Start with Shipping level 1 (not 0)
//          Can advance Shipping to level 5 (not standard max of 3)
type Mermaids struct {
	BaseFaction
	hasStronghold              bool
	hasUsedStrongholdShipping  bool // One-time free shipping upgrade
	shippingLevel              int  // Mermaids start at level 1
}

func NewMermaids() *Mermaids {
	return &Mermaids{
		BaseFaction: BaseFaction{
			Type:        models.FactionMermaids,
			HomeTerrain: models.TerrainLake,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  3, // Mermaids start with 3/9 power (not standard 5/7)
				Power2:  9,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:             false,
		hasUsedStrongholdShipping: false,
		shippingLevel:             1, // Mermaids start with Shipping level 1
	}
}

// GetStartingCultPositions returns Mermaids starting cult track positions
func (f *Mermaids) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 2, Earth: 0, Air: 0}
}

// GetSanctuaryCost returns the expensive sanctuary cost for Mermaids
func (f *Mermaids) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for Mermaids' river-skipping ability
// Note: This is NOT a town founding bonus like Witches/Swarmlings
// It allows river hexes to count as connections when forming towns
func (f *Mermaids) HasSpecialAbility(ability SpecialAbility) bool {
	// Mermaids don't have a predefined special ability constant
	// Their ability (skip river for town connectivity) is checked via CanSkipRiverForTown()
	return false
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Mermaids) GetStrongholdAbility() string {
	return "Immediately and only once: Move forward 1 space on Shipping track (no cost)"
}

// BuildStronghold marks that the stronghold has been built
// Returns true if free shipping upgrade should be granted
func (f *Mermaids) BuildStronghold() bool {
	f.hasStronghold = true
	
	// Return whether free shipping upgrade should be granted
	if !f.hasUsedStrongholdShipping {
		f.hasUsedStrongholdShipping = true
		return true
	}
	return false
}

// GetShippingLevel returns the current shipping level
func (f *Mermaids) GetShippingLevel() int {
	return f.shippingLevel
}

// SetShippingLevel sets the shipping level
// NOTE: Phase 6.2 (Action System) uses this when upgrading shipping
func (f *Mermaids) SetShippingLevel(level int) {
	f.shippingLevel = level
}

// GetMaxShippingLevel returns the maximum shipping level for Mermaids
// Mermaids can reach level 5 (not standard max of 3)
func (f *Mermaids) GetMaxShippingLevel() int {
	return 5
}

// CanSkipRiverForTown returns whether Mermaids can use their town ability
// NOTE: Phase 6.2 (Action System) and Phase 3.2 (Town Formation) implement this
func (f *Mermaids) CanSkipRiverForTown() bool {
	return true // Mermaids can always use this ability when founding a town
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Mermaids) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is the one-time free shipping upgrade
	// This is handled by BuildStronghold()
	return nil
}
