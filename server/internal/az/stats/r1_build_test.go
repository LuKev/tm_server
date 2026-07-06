package stats

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestR1BuildTrackerCapturesFirstPostRoundOneState(t *testing.T) {
	gs := &game.GameState{
		Round: 1,
		Map:   board.NewTerraMysticaMap(),
		Players: map[string]*game.Player{
			"p1": {ID: "p1", Faction: factions.NewGiants()},
			"p2": {ID: "p2", Faction: factions.NewSwarmlings()},
		},
	}
	gs.Map.GetHex(board.NewHex(0, 0)).Building = &models.Building{Type: models.BuildingTemple, PlayerID: "p1"}
	gs.Map.GetHex(board.NewHex(1, 0)).Building = &models.Building{Type: models.BuildingStronghold, PlayerID: "p2"}

	tracker := NewR1BuildTracker([]string{"p1", "p2"})
	tracker.Observe(gs)
	if len(tracker.Samples()) != 0 {
		t.Fatalf("captured during round 1: %#v", tracker.Samples())
	}

	gs.Round = 2
	tracker.Observe(gs)
	rates := R1BuildRates{}
	AddR1BuildSamples(rates, tracker.Samples())

	if got := rates["Giants"].TempleOrSanctuaryRate; got != 1 {
		t.Fatalf("Giants temple rate = %v, want 1", got)
	}
	if got := rates["Swarmlings"].StrongholdRate; got != 1 {
		t.Fatalf("Swarmlings stronghold rate = %v, want 1", got)
	}
	if got := rates["Swarmlings"].TempleOrStrongholdRate; got != 1 {
		t.Fatalf("Swarmlings temple/SH rate = %v, want 1", got)
	}
}

func TestR1BuildTrackerFinalizeSkipsGamesThatRemainInRoundOne(t *testing.T) {
	gs := &game.GameState{
		Round: 1,
		Map:   board.NewTerraMysticaMap(),
		Players: map[string]*game.Player{
			"p1": {ID: "p1", Faction: factions.NewGiants()},
		},
	}
	gs.Map.GetHex(board.NewHex(0, 0)).Building = &models.Building{Type: models.BuildingTemple, PlayerID: "p1"}

	tracker := NewR1BuildTracker([]string{"p1"})
	tracker.Finalize(gs)
	if got := len(tracker.Samples()); got != 0 {
		t.Fatalf("Finalize captured %d samples before round 1 ended", got)
	}
}
