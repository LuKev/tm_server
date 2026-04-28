package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

func TestGameSimulator_RunsCleanupCultRewardsBeforePostPassAction(t *testing.T) {
	gs := game.NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true}
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasPassed = true
	player.Resources.Workers = 0
	gs.PassOrder = []string{"p1"}
	gs.Round = 1
	gs.Phase = game.PhaseAction
	gs.ScoringTiles = game.NewScoringTileState()
	gs.ScoringTiles.Tiles = []game.ScoringTile{
		{
			Type:             game.ScoringTown,
			ActionType:       game.ScoringActionTown,
			ActionVP:         5,
			CultTrack:        game.CultEarth,
			CultThreshold:    4,
			CultRewardType:   game.CultRewardSpade,
			CultRewardAmount: 1,
		},
	}
	player.CultPositions[game.CultEarth] = 4
	gs.CultTracks.PlayerPositions["p1"][game.CultEarth] = 4

	sourceHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(sourceHex, models.TerrainForest)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)
	gs.Map.GetHex(sourceHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    player.Faction.GetType(),
		PlayerID:   "p1",
		PowerValue: 1,
	}

	actions := []notation.LogItem{
		notation.ActionItem{Action: &game.TransformAndBuildAction{
			BaseAction:    game.BaseAction{Type: game.ActionTransformAndBuild, PlayerID: "p1"},
			TargetHex:     targetHex,
			TargetTerrain: models.TerrainTypeUnknown,
		}},
		notation.RoundStartItem{Round: 2, TurnOrder: []string{"p1"}},
	}

	sim := NewGameSimulator(gs, actions)
	if err := sim.StepForward(); err != nil {
		t.Fatalf("first StepForward() error = %v", err)
	}

	if terrain := sim.CurrentState.Map.GetHex(targetHex).Terrain; terrain != models.TerrainForest {
		t.Fatalf("target terrain = %v, want %v", terrain, models.TerrainForest)
	}
	if got := sim.CurrentState.PendingCultRewardSpades["p1"]; got != 0 {
		t.Fatalf("pending cult reward spades after cleanup transform = %d, want 0", got)
	}

	if err := sim.StepForward(); err != nil {
		t.Fatalf("second StepForward() error = %v", err)
	}
	if sim.CurrentState.Round != 2 {
		t.Fatalf("round after RoundStartItem = %d, want 2", sim.CurrentState.Round)
	}
	if got := sim.CurrentState.PendingCultRewardSpades["p1"]; got != 0 {
		t.Fatalf("pending cult reward spades after RoundStartItem = %d, want 0", got)
	}
}

func TestGameSimulator_CleanupPhaseItemAwardsCultRewardsBeforeAction(t *testing.T) {
	gs := game.NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true}
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasPassed = true
	player.Resources.Workers = 0
	gs.PassOrder = []string{"p1"}
	gs.Round = 1
	gs.Phase = game.PhaseAction
	gs.ScoringTiles = game.NewScoringTileState()
	gs.ScoringTiles.Tiles = []game.ScoringTile{
		{
			Type:             game.ScoringTown,
			ActionType:       game.ScoringActionTown,
			ActionVP:         5,
			CultTrack:        game.CultEarth,
			CultThreshold:    4,
			CultRewardType:   game.CultRewardSpade,
			CultRewardAmount: 1,
		},
	}
	player.CultPositions[game.CultEarth] = 4
	gs.CultTracks.PlayerPositions["p1"][game.CultEarth] = 4

	sourceHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(sourceHex, models.TerrainForest)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)
	gs.Map.GetHex(sourceHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    player.Faction.GetType(),
		PlayerID:   "p1",
		PowerValue: 1,
	}

	sim := NewGameSimulator(gs, []notation.LogItem{
		notation.CleanupPhaseItem{},
		notation.ActionItem{Action: &game.TransformAndBuildAction{
			BaseAction:    game.BaseAction{Type: game.ActionTransformAndBuild, PlayerID: "p1"},
			TargetHex:     targetHex,
			TargetTerrain: models.TerrainTypeUnknown,
		}},
	})
	if err := sim.StepForward(); err != nil {
		t.Fatalf("cleanup StepForward() error = %v", err)
	}
	if got := sim.CurrentState.PendingCultRewardSpades["p1"]; got != 1 {
		t.Fatalf("pending cult reward spades after cleanup marker = %d, want 1", got)
	}
	if err := sim.StepForward(); err != nil {
		t.Fatalf("transform StepForward() error = %v", err)
	}
	if terrain := sim.CurrentState.Map.GetHex(targetHex).Terrain; terrain != models.TerrainForest {
		t.Fatalf("target terrain = %v, want %v", terrain, models.TerrainForest)
	}
}

func TestGameSimulator_AwardsTemplePriestCleanupCoinsWhenRoundStartTriggersCleanup(t *testing.T) {
	gs := game.NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true}
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasPassed = true
	gs.PassOrder = []string{"p1"}
	gs.Round = 1
	gs.Phase = game.PhaseAction
	gs.ScoringTiles = game.NewScoringTileState()
	gs.ScoringTiles.Tiles = []game.ScoringTile{
		{
			Type:             game.ScoringTemplePriest,
			ActionType:       game.ScoringActionTemple,
			ActionVP:         4,
			CultRewardType:   game.CultRewardCoin,
			CultRewardAmount: 2,
		},
	}
	gs.ScoringTiles.RecordPriestSent("p1")
	gs.ScoringTiles.RecordPriestSent("p1")
	gs.ScoringTiles.RecordPriestSent("p1")

	sim := NewGameSimulator(gs, []notation.LogItem{
		notation.RoundStartItem{Round: 2, TurnOrder: []string{"p1"}},
	})

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v", err)
	}
	if sim.CurrentState.Round != 2 {
		t.Fatalf("round after RoundStartItem = %d, want 2", sim.CurrentState.Round)
	}
	if got := sim.CurrentState.GetPlayer("p1").Resources.Coins; got != 21 {
		t.Fatalf("coins after temple-priest cleanup reward = %d, want 21", got)
	}
	if got := sim.CurrentState.ScoringTiles.GetPriestsSent("p1"); got != 0 {
		t.Fatalf("priests sent after cleanup reward = %d, want 0", got)
	}
}
