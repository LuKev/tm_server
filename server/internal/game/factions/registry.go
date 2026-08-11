package factions

import (
	"fmt"
	"reflect"

	"github.com/lukev/tm_server/internal/models"
)

// SearchState contains faction implementation state that affects rules but is
// intentionally private and therefore absent from ordinary GameState JSON.
type SearchState struct {
	Type             models.FactionType `json:"type"`
	TerraformCostOne int                `json:"terraform_cost_one"`
	HasStronghold    bool               `json:"has_stronghold,omitempty"`
	FlightRange      int                `json:"flight_range,omitempty"`
	ShippingLevel    int                `json:"shipping_level,omitempty"`
}

// StateForSearch returns the canonical rules-relevant faction fingerprint.
func StateForSearch(faction Faction) SearchState {
	if faction == nil {
		return SearchState{}
	}
	state := SearchState{Type: faction.GetType(), TerraformCostOne: faction.GetTerraformCost(1)}
	switch typed := faction.(type) {
	case *Fakirs:
		state.HasStronghold = typed.HasStronghold()
		state.FlightRange = typed.GetFlightRange()
	case *Dwarves:
		state.HasStronghold = typed.HasStronghold()
	case *Halflings:
		state.HasStronghold = typed.hasStronghold
	case *Darklings:
		state.HasStronghold = typed.hasStronghold
	case *Engineers:
		state.HasStronghold = typed.HasStronghold()
	case *Mermaids:
		state.ShippingLevel = typed.GetShippingLevel()
	}
	return state
}

// Clone returns an independent copy of a faction implementation. Faction
// structs contain mutable rule state (for example digging level), so sharing an
// interface pointer across GameState clones corrupts search parents and undo
// snapshots. Current faction implementations contain only value fields.
func Clone(src Faction) Faction {
	if src == nil {
		return nil
	}
	switch typed := src.(type) {
	case *configuredFaction:
		return cloneConfiguredFaction(typed)
	case *configuredDiggingFaction:
		clone := *typed
		clone.configuredFaction = cloneConfiguredFaction(typed.configuredFaction)
		return &clone
	case *configuredShippingFaction:
		clone := *typed
		clone.configuredFaction = cloneConfiguredFaction(typed.configuredFaction)
		return &clone
	}
	value := reflect.ValueOf(src)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		panic("factions.Clone requires a non-nil faction pointer")
	}
	clone := reflect.New(value.Elem().Type())
	clone.Elem().Set(value.Elem())
	return clone.Interface().(Faction)
}

func cloneConfiguredFaction(src *configuredFaction) *configuredFaction {
	if src == nil {
		return nil
	}
	clone := *src
	clone.dwellingIncomeSeq = append([]Income(nil), src.dwellingIncomeSeq...)
	clone.tradingIncomeSeq = append([]Income(nil), src.tradingIncomeSeq...)
	clone.templeIncomeSeq = append([]Income(nil), src.templeIncomeSeq...)
	clone.fixedTerraformCost = cloneInt(src.fixedTerraformCost)
	clone.diggingCost = cloneCost(src.diggingCost)
	clone.dwellingCost = cloneCost(src.dwellingCost)
	clone.tradingHouseCost = cloneCost(src.tradingHouseCost)
	clone.templeCost = cloneCost(src.templeCost)
	clone.sanctuaryCost = cloneCost(src.sanctuaryCost)
	clone.strongholdCost = cloneCost(src.strongholdCost)
	return &clone
}

func cloneInt(src *int) *int {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneCost(src *Cost) *Cost {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

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
