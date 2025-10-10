package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Dwarves faction - Gray/Mountain
// Ability: When taking "Transform and Build" action, may skip one Terrain or River space by paying 2 more Workers (Tunneling)
//          Get 4 VP each time when Tunneling
//          Have no Shipping (cannot increase shipping level)
//          In final Area scoring, Structures reachable via Tunneling are considered connected
// Stronghold: After building, only pay 1 more Worker instead of 2 when Tunneling
// Special: Cannot increase shipping level
type Dwarves struct {
	BaseFaction
	hasStronghold bool
}

func NewDwarves() *Dwarves {
	return &Dwarves{
		BaseFaction: BaseFaction{
			Type:        models.FactionDwarves,
			HomeTerrain: models.TerrainMountain,
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
		hasStronghold: false,
	}
}

// GetShippingCost overrides the base method
// Dwarves cannot increase shipping (they have no shipping)
// NOTE: Phase 6.2 (Action System) must prevent Dwarves from taking shipping actions
func (f *Dwarves) GetShippingCost(currentLevel int) Cost {
	// Dwarves cannot upgrade shipping
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for tunnel digging
func (f *Dwarves) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityTunnelDigging
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Dwarves) GetStrongholdAbility() string {
	return "After building: Only pay 1 more Worker instead of 2 when Tunneling"
}

// BuildStronghold marks that the stronghold has been built
func (f *Dwarves) BuildStronghold() {
	f.hasStronghold = true
}

// HasStronghold returns whether the stronghold has been built
func (f *Dwarves) HasStronghold() bool {
	return f.hasStronghold
}

// GetTunnelingCost returns the worker cost for tunneling
// NOTE: Phase 6.2 (Action System) implements tunneling mechanic
func (f *Dwarves) GetTunnelingCost() int {
	if f.hasStronghold {
		return 1 // After stronghold, only 1 extra worker
	}
	return 2 // Before stronghold, 2 extra workers
}

// GetTunnelingVPBonus returns the VP bonus for tunneling
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 6.2 (Action System) must apply this bonus when Dwarves tunnel
func (f *Dwarves) GetTunnelingVPBonus() int {
	return 4 // 4 VP each time tunneling
}

// CanTunnel returns whether Dwarves can use tunneling
// Tunneling is always available for Dwarves (it's their core ability)
func (f *Dwarves) CanTunnel() bool {
	return true
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Dwarves) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is passive (reduced tunneling cost)
	return nil
}
