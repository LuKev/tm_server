package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestBaseFaction_GetType(t *testing.T) {
	f := &BaseFaction{
		Type:        models.FactionNomads,
		HomeTerrain: models.TerrainDesert,
	}

	if f.GetType() != models.FactionNomads {
		t.Errorf("expected faction type Nomads, got %v", f.GetType())
	}
}

func TestBaseFaction_GetHomeTerrain(t *testing.T) {
	f := &BaseFaction{
		Type:        models.FactionNomads,
		HomeTerrain: models.TerrainDesert,
	}

	if f.GetHomeTerrain() != models.TerrainDesert {
		t.Errorf("expected home terrain Desert, got %v", f.GetHomeTerrain())
	}
}

func TestBaseFaction_GetTerraformCost(t *testing.T) {
	tests := []struct {
		name         string
		diggingLevel int
		distance     int
		expected     int
	}{
		{"No digging, 1 spade", 0, 1, 3},
		{"No digging, 2 spades", 0, 2, 6},
		{"Level 1 digging, 1 spade", 1, 1, 2},
		{"Level 2 digging, 1 spade", 2, 1, 1},
		{"Level 2 digging, 3 spades", 2, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &BaseFaction{
				DiggingLevel: tt.diggingLevel,
			}
			result := f.GetTerraformCost(tt.distance)
			if result != tt.expected {
				t.Errorf("GetTerraformCost(%d) with digging %d = %d, want %d",
					tt.distance, tt.diggingLevel, result, tt.expected)
			}
		})
	}
}

func TestStandardBuildingCosts(t *testing.T) {
	f := &BaseFaction{}

	// Test dwelling cost
	dwellingCost := f.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	// Test trading house cost
	tpCost := f.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}

	// Test temple cost
	templeCost := f.GetTempleCost()
	if templeCost.Workers != 2 || templeCost.Coins != 5 {
		t.Errorf("unexpected temple cost: %+v", templeCost)
	}

	// Test sanctuary cost
	sanctuaryCost := f.GetSanctuaryCost()
	if sanctuaryCost.Workers != 4 || sanctuaryCost.Coins != 6 {
		t.Errorf("unexpected sanctuary cost: %+v", sanctuaryCost)
	}

	// Test stronghold cost
	strongholdCost := f.GetStrongholdCost()
	if strongholdCost.Workers != 4 || strongholdCost.Coins != 6 {
		t.Errorf("unexpected stronghold cost: %+v", strongholdCost)
	}
}

func TestCanAfford(t *testing.T) {
	resources := Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power3:  3,
	}

	tests := []struct {
		name     string
		cost     Cost
		expected bool
	}{
		{"Can afford", Cost{Coins: 5, Workers: 3, Priests: 1, Power: 2}, true},
		{"Exact match", Cost{Coins: 10, Workers: 5, Priests: 2, Power: 3}, true},
		{"Not enough coins", Cost{Coins: 11, Workers: 5, Priests: 2, Power: 3}, false},
		{"Not enough workers", Cost{Coins: 10, Workers: 6, Priests: 2, Power: 3}, false},
		{"Not enough priests", Cost{Coins: 10, Workers: 5, Priests: 3, Power: 3}, false},
		{"Not enough power", Cost{Coins: 10, Workers: 5, Priests: 2, Power: 4}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanAfford(resources, tt.cost)
			if result != tt.expected {
				t.Errorf("CanAfford(%+v, %+v) = %v, want %v",
					resources, tt.cost, result, tt.expected)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	resources := Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power3:  3,
	}

	cost := Cost{
		Coins:   3,
		Workers: 2,
		Priests: 1,
		Power:   1,
	}

	result := Subtract(resources, cost)

	if result.Coins != 7 {
		t.Errorf("expected 7 coins, got %d", result.Coins)
	}
	if result.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", result.Workers)
	}
	if result.Priests != 1 {
		t.Errorf("expected 1 priest, got %d", result.Priests)
	}
	if result.Power3 != 2 {
		t.Errorf("expected 2 power, got %d", result.Power3)
	}
}

func TestAdd(t *testing.T) {
	a := Resources{
		Coins:   5,
		Workers: 3,
		Priests: 1,
		Power1:  2,
		Power2:  3,
		Power3:  1,
	}

	b := Resources{
		Coins:   3,
		Workers: 2,
		Priests: 1,
		Power1:  1,
		Power2:  2,
		Power3:  2,
	}

	result := Add(a, b)

	if result.Coins != 8 {
		t.Errorf("expected 8 coins, got %d", result.Coins)
	}
	if result.Workers != 5 {
		t.Errorf("expected 5 workers, got %d", result.Workers)
	}
	if result.Priests != 2 {
		t.Errorf("expected 2 priests, got %d", result.Priests)
	}
	if result.Power1 != 3 {
		t.Errorf("expected 3 power1, got %d", result.Power1)
	}
	if result.Power2 != 5 {
		t.Errorf("expected 5 power2, got %d", result.Power2)
	}
	if result.Power3 != 3 {
		t.Errorf("expected 3 power3, got %d", result.Power3)
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Create a test faction
	testFaction := &BaseFaction{
		Type:        models.FactionNomads,
		HomeTerrain: models.TerrainDesert,
	}

	// Register it
	registry.Register(testFaction)

	// Retrieve it
	faction, err := registry.Get(models.FactionNomads)
	if err != nil {
		t.Fatalf("failed to get faction: %v", err)
	}

	if faction.GetType() != models.FactionNomads {
		t.Errorf("expected Nomads, got %v", faction.GetType())
	}

	// Try to get another faction (Fakirs now exists)
	faction2, err := registry.Get(models.FactionFakirs)
	if err != nil {
		t.Fatalf("failed to get Fakirs faction: %v", err)
	}
	if faction2.GetType() != models.FactionFakirs {
		t.Errorf("expected Fakirs, got %v", faction2.GetType())
	}
}

func TestGetByTerrain(t *testing.T) {
	registry := NewRegistry()

	// Register two factions with same terrain
	registry.Register(&BaseFaction{
		Type:        models.FactionNomads,
		HomeTerrain: models.TerrainDesert,
	})
	registry.Register(&BaseFaction{
		Type:        models.FactionFakirs,
		HomeTerrain: models.TerrainDesert,
	})
	registry.Register(&BaseFaction{
		Type:        models.FactionWitches,
		HomeTerrain: models.TerrainForest,
	})

	// Get desert factions
	desertFactions := registry.GetByTerrain(models.TerrainDesert)
	if len(desertFactions) != 2 {
		t.Errorf("expected 2 desert factions, got %d", len(desertFactions))
	}

	// Get forest factions
	forestFactions := registry.GetByTerrain(models.TerrainForest)
	if len(forestFactions) != 2 {
		t.Errorf("expected 2 forest factions (Witches + Auren), got %d", len(forestFactions))
	}
}
