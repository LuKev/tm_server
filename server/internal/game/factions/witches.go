package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Witches faction - Green/Forest
// Ability: Get 5 additional Victory points when founding a Town
// Stronghold: Witches' Ride - Once per Action phase, build 1 Dwelling on any free Forest space
//             (that was Forest at start of Action phase) without paying 1 Worker or 2 Coins,
//             and ignoring adjacency rule
type Witches struct {
	BaseFaction
	hasStronghold            bool
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
		hasStronghold:            false,
		witchesRideUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Witches starting cult track positions
func (f *Witches) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 0, Earth: 0, Air: 2}
}

// HasSpecialAbility returns true for town bonus
func (f *Witches) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityTownBonus
}

// BuildStronghold marks that the stronghold has been built
func (f *Witches) BuildStronghold() {
	f.hasStronghold = true
}

// GetTownFoundingBonus returns the bonus VP for founding a town
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 3.2 (Town Formation) detects towns
func (f *Witches) GetTownFoundingBonus() int {
	return 5 // Witches get +5 VP when founding a town
}
