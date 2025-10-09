package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Witches faction - Green/Forest
// Ability: Get 5 additional Victory points when founding a Town
// Stronghold: Witches' Ride - Once per Action phase, build 1 Dwelling on any free Forest space
//             (that was Forest at start of Action phase) without paying 1 Worker or 2 Coins,
//             and ignoring adjacency rule
type Witches struct {
	BaseFaction
	hasStronghold           bool
	witchesRideUsedThisRound bool
}

func NewWitches() *Witches {
	return &Witches{
		BaseFaction: BaseFaction{
			Type:        models.FactionWitches,
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
		hasStronghold:           false,
		witchesRideUsedThisRound: false,
	}
}

// HasSpecialAbility returns true for town bonus
func (f *Witches) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityTownBonus
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Witches) GetStrongholdAbility() string {
	return "Witches' Ride: Once per Action phase, build 1 Dwelling on any free Forest space (that was Forest at start of Action) without paying 1 Worker or 2 Coins, ignoring adjacency"
}

// BuildStronghold marks that the stronghold has been built
func (f *Witches) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseWitchesRide checks if the Witches' Ride special action can be used
func (f *Witches) CanUseWitchesRide() bool {
	return f.hasStronghold && !f.witchesRideUsedThisRound
}

// GetWitchesRideCost returns the cost for Witches' Ride dwelling
// Cost is 0 workers and 0 coins (free dwelling, but still costs 2 coins normally)
// The special action waives the 1 worker and 2 coins
func (f *Witches) GetWitchesRideCost() Cost {
	return Cost{
		Coins:   0, // Normally 2 coins, but waived
		Workers: 0, // Normally 1 worker, but waived
		Priests: 0,
		Power:   0,
	}
}

// UseWitchesRide marks the Witches' Ride special action as used
// Full validation (forest tile availability, dwelling supply, etc.) will be
// implemented in Phase 6.2 (Action System) as part of WitchesRideAction
func (f *Witches) UseWitchesRide() error {
	if !f.hasStronghold {
		return fmt.Errorf("must build stronghold before using Witches' Ride")
	}
	
	if f.witchesRideUsedThisRound {
		return fmt.Errorf("Witches' Ride already used this Action phase")
	}
	
	f.witchesRideUsedThisRound = true
	return nil
}

// ResetWitchesRide resets the Witches' Ride for a new Action phase
func (f *Witches) ResetWitchesRide() {
	f.witchesRideUsedThisRound = false
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Witches) ExecuteStrongholdAbility(gameState interface{}) error {
	return f.UseWitchesRide()
}

// GetTownFoundingBonus returns the bonus VP for founding a town
func (f *Witches) GetTownFoundingBonus() int {
	return 5 // Witches get +5 VP when founding a town
}

// ModifyIncome adds the town founding bonus if applicable
func (f *Witches) ModifyIncome(baseIncome Resources) Resources {
	// Town founding bonus is applied separately when a town is founded
	// This method is for ongoing income modifications
	return baseIncome
}
