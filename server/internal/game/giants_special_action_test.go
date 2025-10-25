package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestGiantsTransform_AwardsScoringTileVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold to enable special action
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Find a hex to transform
	targetHex := NewHex(0, 0)
	
	initialVP := player.VictoryPoints
	
	// Execute Giants Transform (2 free spades)
	action := NewGiantsTransformAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Giants transform: %v", err)
	}
	
	// Should get 2 VP per spade * 2 spades = 4 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 2 * 2 // 2 spades * 2 VP per spade
	if vpGained != expectedVP {
		t.Errorf("expected %d VP from Giants transform, got %d", expectedVP, vpGained)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != faction.GetHomeTerrain() {
		t.Errorf("terrain not transformed to home terrain")
	}
}

func TestGiantsTransform_WithDwelling(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Give player resources for dwelling
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	
	targetHex := NewHex(0, 0)
	
	// Execute Giants Transform with dwelling
	action := NewGiantsTransformAction("player1", targetHex, true)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Giants transform with dwelling: %v", err)
	}
	
	// Verify dwelling was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Error("dwelling not placed")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Verify resources were spent
	dwellingCost := faction.GetDwellingCost()
	if player.Resources.Workers != 5-dwellingCost.Workers {
		t.Errorf("workers not spent correctly")
	}
}

func TestGiantsTransform_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.HasStrongholdAbility = true
	
	targetHex1 := NewHex(0, 0)
	targetHex2 := NewHex(1, 0)
	
	// First use should succeed
	action1 := NewGiantsTransformAction("player1", targetHex1, false)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("first Giants transform should succeed: %v", err)
	}
	
	// Second use in same round should fail
	action2 := NewGiantsTransformAction("player1", targetHex2, false)
	err = action2.Execute(gs)
	if err == nil {
		t.Error("second Giants transform in same round should fail")
	}
}
