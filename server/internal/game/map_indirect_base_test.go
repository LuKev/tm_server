package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// These tests use the BaseGameTerrainLayout via board.NewTerraMysticaMap()
// and assert shipping (indirect adjacency) over rivers only.
//
// Base map layout reference (rows 0-2):
// Row 0: Plains, Mountain, Forest, Lake, Desert, Wasteland, Plains, Swamp, Wasteland, Forest, Lake, Wasteland, Swamp
// Row 1: Desert, River, River, Plains, Swamp, River, River, Desert, Swamp, River, River, Desert
// Row 2: River, River, Swamp, River, Mountain, River, Forest, River, Forest, River, Mountain, River, River

// Tests moved to board/map_indirect_test.go

func TestBonusCard_ShippingBonus_IndirectAdjacency(t *testing.T) {
	gs := NewGameState()

	// Add player with shipping level 1
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	player.ShippingLevel = 1

	// Set up bonus cards and give player the Shipping bonus card (+1 shipping)
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShipping})
	bonusCard := BonusCardShipping
	gs.BonusCards.TakeBonusCard("player1", bonusCard)

	// Place a building at b1
	b1 := board.NewHex(0, 1)
	gs.Map.GetHex(b1).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(b1).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// b2 requires shipping=2 to reach from b1
	b2 := board.NewHex(3, 1)

	// c2 requires shipping=3 to reach from b1
	c2 := board.NewHex(5, 1)

	// Verify effective shipping level with bonus card
	if bonusCardType, ok := gs.BonusCards.GetPlayerCard("player1"); ok {
		shippingBonus := GetBonusCardShippingBonus(bonusCardType, player.Faction.GetType())
		effectiveShipping := player.ShippingLevel + shippingBonus

		// With bonus card, effective shipping should be 2 (1 base + 1 bonus)
		if effectiveShipping != 2 {
			t.Errorf("expected effective shipping=2 with bonus card, got %d", effectiveShipping)
		}
	} else {
		t.Fatal("player should have bonus card")
	}

	// With bonus card (effective shipping=2), b2 SHOULD be reachable
	if !gs.IsAdjacentToPlayerBuilding(b2, "player1") {
		t.Fatal("expected b2 to be adjacent with effective shipping=2 (base 1 + bonus 1)")
	}

	// With bonus card (effective shipping=2), c2 should NOT be reachable (needs shipping=3)
	if gs.IsAdjacentToPlayerBuilding(c2, "player1") {
		t.Fatal("expected c2 to NOT be adjacent with effective shipping=2 (needs shipping=3)")
	}
}

func TestBonusCard_ShippingBonus_DwarvesNoEffect(t *testing.T) {
	gs := NewGameState()
	
	// Add Dwarves player with shipping level 1
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	player.ShippingLevel = 1
	
	// Set up bonus cards and give player the Shipping bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShipping})
	bonusCard := BonusCardShipping
	gs.BonusCards.TakeBonusCard("player1", bonusCard)
	
	// Get effective shipping level with bonus card
	if bonusCardType, ok := gs.BonusCards.GetPlayerCard("player1"); ok {
		shippingBonus := GetBonusCardShippingBonus(bonusCardType, player.Faction.GetType())
		effectiveShipping := player.ShippingLevel + shippingBonus
		
		// Dwarves should NOT benefit from shipping bonus
		if effectiveShipping != 1 {
			t.Errorf("expected effective shipping=1 for Dwarves (no bonus), got %d", effectiveShipping)
		}
		
		if shippingBonus != 0 {
			t.Errorf("expected shipping bonus=0 for Dwarves, got %d", shippingBonus)
		}
	} else {
		t.Fatal("player should have bonus card")
	}
}

func TestBonusCard_ShippingBonus_FakirsNoEffect(t *testing.T) {
	gs := NewGameState()
	
	// Add Fakirs player with shipping level 1
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	player.ShippingLevel = 1
	
	// Set up bonus cards and give player the Shipping bonus card
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardShipping})
	bonusCard := BonusCardShipping
	gs.BonusCards.TakeBonusCard("player1", bonusCard)
	
	// Get effective shipping level with bonus card
	if bonusCardType, ok := gs.BonusCards.GetPlayerCard("player1"); ok {
		shippingBonus := GetBonusCardShippingBonus(bonusCardType, player.Faction.GetType())
		effectiveShipping := player.ShippingLevel + shippingBonus
		
		// Fakirs should NOT benefit from shipping bonus
		if effectiveShipping != 1 {
			t.Errorf("expected effective shipping=1 for Fakirs (no bonus), got %d", effectiveShipping)
		}
		
		if shippingBonus != 0 {
			t.Errorf("expected shipping bonus=0 for Fakirs, got %d", shippingBonus)
		}
	} else {
		t.Fatal("player should have bonus card")
	}
}
