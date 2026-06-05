package actions_test

import (
	"testing"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestLegalActionsAreExecutableOnScenario(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	if len(legal) == 0 {
		t.Fatal("expected legal actions")
	}
	for _, option := range legal {
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("legal action %s did not apply: %v", option.ID, err)
		}
	}
}

func TestLegalActionsExcludeMainTurnTransformOnly(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	hasTransformBuild := false
	for _, option := range legal {
		if option.Type == "transform" {
			t.Fatalf("main-turn transform-only action should be pruned from AZ surface: %s", option.ID)
		}
		if option.Type == "transform_build" {
			hasTransformBuild = true
		}
	}
	if !hasTransformBuild {
		t.Fatal("expected transform/build actions to remain legal")
	}
}

func TestLegalActionsIncludeExecutablePass(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.Type != "pass" {
			continue
		}
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("pass action %s did not apply: %v", option.ID, err)
		}
		return
	}
	t.Fatal("expected at least one legal pass action")
}

func TestLegalActionsExcludeFreeConversions(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.Type == "conversion" || option.Type == "burn" {
			t.Fatalf("free conversion action should be pruned from AZ surface: %s", option.ID)
		}
	}
}

func TestLegalActionsExcludeMainTurnActionsForPassedPlayer(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	current := position.State.GetCurrentPlayer()
	if current == nil {
		t.Fatal("expected current player")
	}
	current.HasPassed = true
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.PlayerID == current.ID {
			t.Fatalf("passed current player should not receive main-turn action: %s", option.ID)
		}
	}
}

func TestLegalActionsExcludeRepeatedMermaidsRiverTownConnect(t *testing.T) {
	gs := game.NewGameState()
	faction := factions.NewMermaids()
	if err := gs.AddPlayer("p1", faction); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0

	river := board.NewHex(0, 0)
	hexes := []board.Hex{
		board.NewHex(1, 0),
		board.NewHex(2, 0),
		board.NewHex(-1, 0),
		board.NewHex(-2, 0),
	}
	for _, hex := range append([]board.Hex{river}, hexes...) {
		if gs.Map.GetHex(hex) == nil {
			gs.Map.Hexes[hex] = &board.MapHex{Coord: hex}
		}
	}
	gs.Map.GetHex(river).Terrain = models.TerrainRiver
	gs.Map.RiverHexes[river] = true
	for _, hex := range hexes {
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "p1",
			PowerValue: 2,
		})
	}

	if err := game.NewMermaidsRiverTownAction("p1", river).Execute(gs); err != nil {
		t.Fatalf("first river town connect failed: %v", err)
	}
	for _, option := range actions.LegalActions(gs) {
		if option.Type == "special_mermaids_town" {
			t.Fatalf("repeated Mermaids river town connect should not be legal: %s", option.ID)
		}
	}
}
