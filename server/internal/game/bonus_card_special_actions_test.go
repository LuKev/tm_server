package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestBonusCardSpade_BasicTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Give player resources (need fewer workers due to free spade)
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	
	// Hex (0,0) is Plains, Auren home is Forest, distance is 3
	// Normal cost: 3 spades * 3 workers = 9 workers
	// With 1 free spade: 2 spades * 3 workers = 6 workers
	hex := NewHex(0, 0)
	
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  false,
	}
	
	initialWorkers := player.Resources.Workers
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action: %v", err)
	}
	
	// Should have used 6 workers (9 - 3 for free spade)
	workersUsed := initialWorkers - player.Resources.Workers
	if workersUsed != 6 {
		t.Errorf("expected 6 workers used, got %d", workersUsed)
	}
	
	// Terrain should be transformed
	mapHex := gs.Map.GetHex(hex)
	if mapHex.Terrain != faction.GetHomeTerrain() {
		t.Errorf("terrain not transformed: got %v, want %v", mapHex.Terrain, faction.GetHomeTerrain())
	}
}

func TestBonusCardSpade_WithDwelling(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	
	hex := NewHex(0, 0)
	
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  true,
	}
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action with dwelling: %v", err)
	}
	
	// Should have a dwelling built
	mapHex := gs.Map.GetHex(hex)
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestBonusCardSpade_ScoringTileVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants() // Giants always use 2 spades
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up scoring tile: Spades + Earth (2 VP per spade)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringSpades,
			ActionType: ScoringActionSpades,
			ActionVP:   2,
		},
	}
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	
	hex := NewHex(0, 0)
	initialVP := player.VictoryPoints
	
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  false,
	}
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action: %v", err)
	}
	
	// Giants use 2 spades, should get 2 VP per spade = 4 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 2 * 2 // 2 spades * 2 VP
	if vpGained != expectedVP {
		t.Errorf("expected %d VP from spades scoring tile, got %d", expectedVP, vpGained)
	}
}

func TestBonusCardSpade_WithDwellingScoringTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up scoring tiles: Spades (2 VP) AND Dwelling (2 VP)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringSpades,
			ActionType: ScoringActionSpades,
			ActionVP:   2,
		},
	}
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	
	hex := NewHex(0, 0)
	initialVP := player.VictoryPoints
	
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  true,
	}
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action with dwelling: %v", err)
	}
	
	// Should get VP from spades (3 spades * 2 VP = 6 VP)
	// Dwelling scoring is not active in this test
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3 * 2 // 3 spades * 2 VP
	if vpGained != expectedVP {
		t.Errorf("expected %d VP from spades, got %d", expectedVP, vpGained)
	}
}

func TestBonusCardSpade_WithDwellingScoringTileBoth(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up scoring tile: Dwelling (2 VP per dwelling)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringDwellingWater,
			ActionType: ScoringActionDwelling,
			ActionVP:   2,
		},
	}
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	
	hex := NewHex(0, 0)
	initialVP := player.VictoryPoints
	
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  true,
	}
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card spade action with dwelling: %v", err)
	}
	
	// Should get 2 VP from dwelling scoring tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected 2 VP from dwelling scoring tile, got %d", vpGained)
	}
}

func TestBonusCardCultAdvance_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Give player the cult advance bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardCultAdvance
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	cultTrack := CultFire
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType: SpecialActionBonusCardCultAdvance,
		CultTrack:  &cultTrack,
	}
	
	initialPosition := gs.CultTracks.GetPosition("player1", CultFire)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card cult advance action: %v", err)
	}
	
	// Should have advanced 1 space on Fire cult track
	newPosition := gs.CultTracks.GetPosition("player1", CultFire)
	if newPosition != initialPosition+1 {
		t.Errorf("expected position %d, got %d", initialPosition+1, newPosition)
	}
}

func TestBonusCardCultAdvance_WithPowerBonus(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	_ = player // Used for power setup
	
	// Give player the cult advance bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardCultAdvance
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up power for gaining
	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0
	
	// Advance to position 2 (so next advance to 3 gives power bonus)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 2, player, gs)
	
	initialBowl2 := player.Resources.Power.Bowl2
	
	cultTrack := CultWater
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType: SpecialActionBonusCardCultAdvance,
		CultTrack:  &cultTrack,
	}
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute bonus card cult advance action: %v", err)
	}
	
	// Should be at position 3 now
	position := gs.CultTracks.GetPosition("player1", CultWater)
	if position != 3 {
		t.Errorf("expected position 3, got %d", position)
	}
	
	// Should have received 1 power (milestone at position 3)
	powerGained := player.Resources.Power.Bowl2 - initialBowl2
	if powerGained != 1 {
		t.Errorf("expected 1 power from milestone, got %d", powerGained)
	}
}

func TestBonusCardSpade_WithoutCard(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Don't give player the spade bonus card
	
	hex := NewHex(0, 0)
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:     SpecialActionBonusCardSpade,
		TargetHex:      &hex,
		BuildDwelling:  false,
	}
	
	err := action.Execute(gs)
	if err == nil {
		t.Error("expected error when player doesn't have spade bonus card")
	}
}

func TestBonusCardCultAdvance_WithoutCard(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Don't give player the cult advance bonus card
	
	cultTrack := CultFire
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType: SpecialActionBonusCardCultAdvance,
		CultTrack:  &cultTrack,
	}
	
	err := action.Execute(gs)
	if err == nil {
		t.Error("expected error when player doesn't have cult advance bonus card")
	}
}
