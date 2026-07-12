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
	gs.Map.GetHex(board.NewHex(0, 0)).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1"}
	gs.Map.GetHex(board.NewHex(1, 0)).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p2"}

	tracker := NewR1BuildTracker(gs, []string{"p1", "p2"})
	tracker.ObserveAction(1, "p1", "upgrade")
	tracker.ObserveAction(1, "p1", "pass")
	tracker.ObserveAction(1, "p2", "pass")
	gs.Map.GetHex(board.NewHex(0, 0)).Building = &models.Building{Type: models.BuildingTemple, PlayerID: "p1"}
	gs.Map.GetHex(board.NewHex(1, 0)).Building = &models.Building{Type: models.BuildingStronghold, PlayerID: "p2"}
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
	if got := rates["Giants"].AnyBuildRate; got != 1 {
		t.Fatalf("Giants any-build rate = %v, want 1", got)
	}
	if got := rates["Giants"].AverageActionsBeforePass; got != 1 {
		t.Fatalf("Giants actions before pass = %v, want 1", got)
	}
	if got := rates["Giants"].BuildingCounts[models.BuildingTemple.String()]; got != 1 {
		t.Fatalf("Giants Temple builds = %d, want 1", got)
	}
	if got := rates["Swarmlings"].PassedBeforeActionRate; got != 1 {
		t.Fatalf("Swarmlings immediate-pass rate = %v, want 1", got)
	}
	if got := rates["Swarmlings"].StrongholdRate; got != 1 {
		t.Fatalf("Swarmlings stronghold rate = %v, want 1", got)
	}
	if got := rates["Swarmlings"].TempleOrStrongholdRate; got != 1 {
		t.Fatalf("Swarmlings temple/SH rate = %v, want 1", got)
	}
}

func TestR1BuildTrackerCountsEachBuildingTransition(t *testing.T) {
	gs := &game.GameState{
		Round: 1,
		Map:   board.NewTerraMysticaMap(),
		Players: map[string]*game.Player{
			"p1": {ID: "p1", Faction: factions.NewGiants()},
		},
	}
	hex := gs.Map.GetHex(board.NewHex(0, 0))
	tracker := NewR1BuildTracker(gs, []string{"p1"})
	hex.Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1"}
	tracker.Observe(gs)
	hex.Building = &models.Building{Type: models.BuildingTradingHouse, PlayerID: "p1"}
	tracker.Observe(gs)
	hex.Building = &models.Building{Type: models.BuildingTemple, PlayerID: "p1"}
	tracker.Observe(gs)
	gs.Round = 2
	tracker.Observe(gs)

	rates := R1BuildRates{}
	AddR1BuildSamples(rates, tracker.Samples())
	entry := rates["Giants"]
	for _, building := range []models.BuildingType{models.BuildingDwelling, models.BuildingTradingHouse, models.BuildingTemple} {
		if got := entry.BuildingCounts[building.String()]; got != 1 {
			t.Fatalf("%s builds = %d, want 1", building, got)
		}
		if got := entry.AverageBuildings[building.String()]; got != 1 {
			t.Fatalf("average %s builds = %v, want 1", building, got)
		}
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

	tracker := NewR1BuildTracker(gs, []string{"p1"})
	tracker.Finalize(gs)
	if got := len(tracker.Samples()); got != 0 {
		t.Fatalf("Finalize captured %d samples before round 1 ended", got)
	}
}

func TestFinalScoreRatesMergeWeightedAverages(t *testing.T) {
	left := FinalScoreRates{}
	AddFinalScore(left, "Giants", 100)
	AddFinalScore(left, "Giants", 80)
	right := FinalScoreRates{}
	AddFinalScore(right, "Giants", 50)
	AddFinalScore(right, "Nomads", 70)

	MergeFinalScoreRates(left, right)
	if got := left["Giants"].AverageScore; got != 230.0/3.0 {
		t.Fatalf("Giants average score = %v, want %v", got, 230.0/3.0)
	}
	if got := left["Nomads"].AverageScore; got != 70 {
		t.Fatalf("Nomads average score = %v, want 70", got)
	}
}
