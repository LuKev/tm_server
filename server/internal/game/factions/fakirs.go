package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Fakirs faction - Yellow/Desert
// Ability: When taking "Transform and Build" action, may skip one Terrain or River space by paying 1 more Priest (Carpet Flight)
//          Get 4 VP each time doing Carpet Flight
//          In final Area scoring, Structures reachable via Carpet Flight are considered connected
// Stronghold: After building, may skip 2 Terrain, or 2 River spaces, or one each when doing Carpet Flight
// Special: Start with 7/5 power (not standard 5/7)
//          Cannot upgrade Shipping
//          Can only upgrade Digging by 1 level (max level 1, not 2)
//          Expensive Stronghold (4 workers, 10 coins vs standard 4 workers, 6 coins)
//          Shipping town tile increases Carpet Flight range by 1
type Fakirs struct {
	BaseFaction
	hasStronghold      bool
	hasShippingTownTile bool // Shipping town tile bonus
}

func NewFakirs() *Fakirs {
	return &Fakirs{
		BaseFaction: BaseFaction{
			Type:        models.FactionFakirs,
			HomeTerrain: models.TerrainDesert,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  7, // Fakirs start with 7/5 power (not standard 5/7)
				Power2:  5,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:      false,
		hasShippingTownTile: false,
	}
}

// GetStrongholdCost returns the expensive stronghold cost for Fakirs
func (f *Fakirs) GetStrongholdCost() Cost {
	return Cost{
		Coins:   10, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// GetShippingCost overrides the base method
// Fakirs cannot increase shipping
// NOTE: Phase 6.2 (Action System) must prevent Fakirs from taking shipping actions
func (f *Fakirs) GetShippingCost(currentLevel int) Cost {
	// Fakirs cannot upgrade shipping
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// GetDiggingCost overrides the base method
// Fakirs can only upgrade digging once (to level 1)
// NOTE: Phase 6.2 (Action System) must prevent Fakirs from upgrading past level 1
func (f *Fakirs) GetDiggingCost(currentLevel int) Cost {
	if currentLevel >= 1 {
		// Cannot upgrade past level 1
		return Cost{
			Coins:   0,
			Workers: 0,
			Priests: 0,
			Power:   0,
		}
	}
	// Standard cost for first upgrade (0 -> 1)
	return StandardDiggingCost(currentLevel)
}

// GetMaxDiggingLevel returns the maximum digging level for Fakirs
func (f *Fakirs) GetMaxDiggingLevel() int {
	return 1 // Fakirs can only reach digging level 1 (not 2)
}

// HasSpecialAbility returns true for carpet flying
func (f *Fakirs) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityCarpetFlying
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Fakirs) GetStrongholdAbility() string {
	return "After building: May skip 2 Terrain, or 2 River spaces, or one each when doing Carpet Flight"
}

// BuildStronghold marks that the stronghold has been built
func (f *Fakirs) BuildStronghold() {
	f.hasStronghold = true
}

// HasStronghold returns whether the stronghold has been built
func (f *Fakirs) HasStronghold() bool {
	return f.hasStronghold
}

// GetCarpetFlightCost returns the priest cost for carpet flight
// NOTE: Phase 6.2 (Action System) implements carpet flight mechanic
func (f *Fakirs) GetCarpetFlightCost() int {
	return 1 // Pay 1 priest to skip one space
}

// GetCarpetFlightRange returns how many spaces can be skipped
// Base: 1 space
// +1 if Stronghold built
// +1 if Shipping town tile acquired
// NOTE: Phase 7.3 (Town Tiles) handles shipping town tile acquisition
func (f *Fakirs) GetCarpetFlightRange() int {
	range_ := 1 // Base range
	
	if f.hasStronghold {
		range_++ // Stronghold adds +1
	}
	
	if f.hasShippingTownTile {
		range_++ // Shipping town tile adds +1
	}
	
	return range_
}

// SetShippingTownTile marks that the Fakirs have acquired the Shipping town tile
// NOTE: Phase 7.3 (Town Tiles) calls this when Fakirs select the Shipping town tile
func (f *Fakirs) SetShippingTownTile(has bool) {
	f.hasShippingTownTile = has
}

// GetCarpetFlightVPBonus returns the VP bonus for carpet flight
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 6.2 (Action System) must apply this bonus when Fakirs use carpet flight
func (f *Fakirs) GetCarpetFlightVPBonus() int {
	return 4 // 4 VP each time doing carpet flight
}

// CanCarpetFlight returns whether Fakirs can use carpet flight
// Carpet flight is always available for Fakirs (it's their core ability)
func (f *Fakirs) CanCarpetFlight() bool {
	return true
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Fakirs) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is passive (increased carpet flight range)
	return nil
}
