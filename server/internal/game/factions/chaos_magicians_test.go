package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestChaosMagicians_BasicProperties(t *testing.T) {
	cm := NewChaosMagicians()

	if cm.GetType() != models.FactionChaosMagicians {
		t.Errorf("expected faction type ChaosMagicians, got %v", cm.GetType())
	}

	if cm.GetHomeTerrain() != models.TerrainWasteland {
		t.Errorf("expected home terrain Wasteland, got %v", cm.GetHomeTerrain())
	}
}

func TestChaosMagicians_StartingResources(t *testing.T) {
	cm := NewChaosMagicians()
	resources := cm.GetStartingResources()

	if resources.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", resources.Coins)
	}
	if resources.Workers != 4 {
		t.Errorf("expected 4 workers (not standard 3), got %d", resources.Workers)
	}
	if resources.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", resources.Priests)
	}
}

func TestChaosMagicians_ExpensiveSanctuary(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians sanctuary costs 8 coins (more expensive than standard 6)
	sanctuaryCost := cm.GetSanctuaryCost()
	if sanctuaryCost.Coins != 8 {
		t.Errorf("expected sanctuary to cost 8 coins, got %d", sanctuaryCost.Coins)
	}
	if sanctuaryCost.Workers != 4 {
		t.Errorf("expected sanctuary to cost 4 workers, got %d", sanctuaryCost.Workers)
	}
}

func TestChaosMagicians_CheapStronghold(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians stronghold costs 4 coins (cheaper than standard 6)
	strongholdCost := cm.GetStrongholdCost()
	if strongholdCost.Coins != 4 {
		t.Errorf("expected stronghold to cost 4 coins (cheaper than standard 6), got %d", strongholdCost.Coins)
	}
	if strongholdCost.Workers != 4 {
		t.Errorf("expected stronghold to cost 4 workers, got %d", strongholdCost.Workers)
	}
}

func TestChaosMagicians_FavorTilesForTemple(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians get 2 favor tiles for Temple (not standard 1)
	favorTiles := cm.GetFavorTilesForTemple()
	if favorTiles != 2 {
		t.Errorf("expected 2 favor tiles for Temple, got %d", favorTiles)
	}
}

func TestChaosMagicians_FavorTilesForSanctuary(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians get 2 favor tiles for Sanctuary (not standard 1)
	favorTiles := cm.GetFavorTilesForSanctuary()
	if favorTiles != 2 {
		t.Errorf("expected 2 favor tiles for Sanctuary, got %d", favorTiles)
	}
}

func TestChaosMagicians_StartsWithOneDwelling(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians start with only 1 dwelling
	if !cm.StartsWithOneDwelling() {
		t.Errorf("Chaos Magicians should start with only 1 dwelling")
	}
}

func TestChaosMagicians_PlacesDwellingLast(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians place dwelling after all other players
	if !cm.PlacesDwellingLast() {
		t.Errorf("Chaos Magicians should place dwelling last")
	}
}

func TestChaosMagicians_StandardCosts(t *testing.T) {
	cm := NewChaosMagicians()

	// Chaos Magicians use standard costs for most buildings
	dwellingCost := cm.GetDwellingCost()
	if dwellingCost.Workers != 1 || dwellingCost.Coins != 2 {
		t.Errorf("unexpected dwelling cost: %+v", dwellingCost)
	}

	tpCost := cm.GetTradingHouseCost()
	if tpCost.Workers != 2 || tpCost.Coins != 6 {
		t.Errorf("unexpected trading house cost: %+v", tpCost)
	}
}
