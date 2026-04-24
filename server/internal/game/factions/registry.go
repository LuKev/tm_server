package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Registry holds all available factions
type Registry struct {
	factions map[models.FactionType]Faction
}

// NewRegistry creates a new faction registry with all factions
func NewRegistry() *Registry {
	r := &Registry{
		factions: make(map[models.FactionType]Faction),
	}

	// Register base factions plus supported fan factions.
	r.Register(NewNomads())
	r.Register(NewFakirs())
	r.Register(NewChaosMagicians())
	r.Register(NewGiants())
	r.Register(NewSwarmlings())
	r.Register(NewMermaids())
	r.Register(NewWitches())
	r.Register(NewAuren())
	r.Register(NewHalflings())
	r.Register(NewCultists())
	r.Register(NewAlchemists())
	r.Register(NewDarklings())
	r.Register(NewEngineers())
	r.Register(NewDwarves())
	r.Register(NewArchitects())
	r.Register(NewArchivists())
	r.Register(NewAtlanteans())
	r.Register(NewChashDallah())
	r.Register(NewChildrenOfTheWyrm())
	r.Register(NewConspirators())
	r.Register(NewDjinni())
	r.Register(NewDynionGeifr())
	r.Register(NewGoblins())
	r.Register(NewProspectors())
	r.Register(NewTheEnlightened())
	r.Register(NewTimeTravelers())
	r.Register(NewTreasurers())
	r.Register(NewWisps())
	r.Register(NewIceMaidens())
	r.Register(NewYetis())
	r.Register(NewDragonlords())
	r.Register(NewAcolytes())
	r.Register(NewShapeshifters())
	r.Register(NewRiverwalkers())
	r.Register(NewFirewalkers())
	r.Register(NewSelkies())
	r.Register(NewSnowShamans())

	return r
}

// Register adds a faction to the registry
func (r *Registry) Register(faction Faction) {
	r.factions[faction.GetType()] = faction
}

// Get retrieves a faction by type
func (r *Registry) Get(factionType models.FactionType) (Faction, error) {
	faction, ok := r.factions[factionType]
	if !ok {
		return nil, fmt.Errorf("faction %s not found", factionType)
	}
	return faction, nil
}

// GetAll returns all registered factions
func (r *Registry) GetAll() []Faction {
	factions := make([]Faction, 0, len(r.factions))
	for _, faction := range r.factions {
		factions = append(factions, faction)
	}
	return factions
}

// GetByTerrain returns all factions that have the given home terrain
func (r *Registry) GetByTerrain(terrain models.TerrainType) []Faction {
	factions := make([]Faction, 0)
	for _, faction := range r.factions {
		if faction.GetHomeTerrain() == terrain {
			factions = append(factions, faction)
		}
	}
	return factions
}

// Standard starting resources for most factions
func StandardStartingResources() Resources {
	return Resources{
		Coins:   15,
		Workers: 3,
		Priests: 0,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	}
}

// NewFaction creates a new instance of a faction by type
func NewFaction(t models.FactionType) Faction {
	// This is a bit inefficient but simple for now
	r := NewRegistry()
	f, _ := r.Get(t)
	return f
}
