package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestHalflingsStronghold_Creates3PendingSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainPlains)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify pending spades was created
	if gs.PendingHalflingsSpades == nil {
		t.Fatal("expected pending Halflings spades after building stronghold")
	}

	if gs.PendingHalflingsSpades.PlayerID != "player1" {
		t.Errorf("expected player1, got %s", gs.PendingHalflingsSpades.PlayerID)
	}

	if gs.PendingHalflingsSpades.SpadesRemaining != 3 {
		t.Errorf("expected 3 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}
}

func TestApplyHalflingsSpade_TransformsHexAndAwardsVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Create pending spades (simulate stronghold build)
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	// Target hex that needs transformation
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Not home terrain

	initialVP := player.VictoryPoints

	// Apply one spade
	action := &ApplyHalflingsSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionApplyHalflingsSpade,
			PlayerID: "player1",
		},
		TargetHex: targetHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to apply spade: %v", err)
	}

	// Verify terrain was transformed
	if gs.Map.GetHex(targetHex).Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", gs.Map.GetHex(targetHex).Terrain)
	}

	// Verify VP was awarded (Halflings get +1 VP per spade)
	if player.VictoryPoints != initialVP+1 {
		t.Errorf("expected %d VP, got %d", initialVP+1, player.VictoryPoints)
	}

	// Verify spades remaining decreased
	if gs.PendingHalflingsSpades.SpadesRemaining != 2 {
		t.Errorf("expected 2 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}

	// Verify hex was tracked
	if len(gs.PendingHalflingsSpades.TransformedHexes) != 1 {
		t.Errorf("expected 1 transformed hex, got %d", len(gs.PendingHalflingsSpades.TransformedHexes))
	}
}

func TestApplyHalflingsSpade_AllThreeSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Create pending spades
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	// Apply 3 spades
	hexes := []Hex{NewHex(0, 0), NewHex(1, 0), NewHex(2, 0)}
	for i, hex := range hexes {
		gs.Map.GetHex(hex).Terrain = models.TerrainForest

		action := &ApplyHalflingsSpadeAction{
			BaseAction: BaseAction{
				Type:     ActionApplyHalflingsSpade,
				PlayerID: "player1",
			},
			TargetHex: hex,
		}

		err := action.Execute(gs)
		if err != nil {
			t.Fatalf("failed to apply spade %d: %v", i+1, err)
		}
	}

	// Verify all spades applied
	if gs.PendingHalflingsSpades.SpadesRemaining != 0 {
		t.Errorf("expected 0 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}

	// Verify all hexes transformed
	if len(gs.PendingHalflingsSpades.TransformedHexes) != 3 {
		t.Errorf("expected 3 transformed hexes, got %d", len(gs.PendingHalflingsSpades.TransformedHexes))
	}

	// Verify VP was awarded (3 spades × 1 VP = 3 VP)
	// Note: This is just the Halflings bonus, not counting scoring tiles
	if player.VictoryPoints < 3 {
		t.Errorf("expected at least 3 VP from spades, got %d", player.VictoryPoints)
	}

	// Verify faction method was called
	spades := faction.UseStrongholdSpades()
	if spades != 0 {
		t.Error("stronghold spades should already be used")
	}
}

func TestBuildHalflingsDwelling_OnTransformedHex(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with all spades applied
	transformedHex := NewHex(0, 0)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{transformedHex, NewHex(1, 0), NewHex(2, 0)},
	}

	// Transform the hex
	gs.Map.GetHex(transformedHex).Terrain = models.TerrainPlains

	// Build dwelling on one of the transformed hexes
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: transformedHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to build dwelling: %v", err)
	}

	// Verify dwelling was placed
	building := gs.Map.GetHex(transformedHex).Building
	if building == nil {
		t.Fatal("expected building on transformed hex")
	}
	if building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", building.Type)
	}

	// Verify pending spades was cleared
	if gs.PendingHalflingsSpades != nil {
		t.Error("expected pending spades to be cleared after building dwelling")
	}
}

func TestBuildHalflingsDwelling_CannotBuildOnUntransformedHex(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with all spades applied
	transformedHex := NewHex(0, 0)
	untransformedHex := NewHex(5, 5)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{transformedHex, NewHex(1, 0), NewHex(2, 0)},
	}

	// Try to build dwelling on a non-transformed hex
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: untransformedHex,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when building on non-transformed hex")
	}
}

func TestBuildHalflingsDwelling_MustApplyAllSpadesFirst(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with spades still remaining
	transformedHex := NewHex(0, 0)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 2, // Still have 2 spades left
		TransformedHexes: []Hex{transformedHex},
	}

	// Try to build dwelling before applying all spades
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: transformedHex,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when trying to build dwelling before applying all spades")
	}
}

func TestSkipHalflingsDwelling_ClearsPendingSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	// Create pending spades with all spades applied
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{NewHex(0, 0), NewHex(1, 0), NewHex(2, 0)},
	}

	// Skip the optional dwelling
	action := &SkipHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionSkipHalflingsDwelling,
			PlayerID: "player1",
		},
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to skip dwelling: %v", err)
	}

	// Verify pending spades was cleared
	if gs.PendingHalflingsSpades != nil {
		t.Error("expected pending spades to be cleared after skipping dwelling")
	}
}

func TestHalflingsSpades_WithScoringTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

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

	// Create pending spades
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints

	// Apply one spade
	action := &ApplyHalflingsSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionApplyHalflingsSpade,
			PlayerID: "player1",
		},
		TargetHex: targetHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to apply spade: %v", err)
	}

	// Verify VP: +1 (Halflings bonus) + 2 (scoring tile) = 3 VP
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, player.VictoryPoints)
	}
}
