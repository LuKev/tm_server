package az

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

func benchmarkActionPosition(b *testing.B) *GamePosition {
	return benchmarkActionPositionForSeed(b, 73001)
}

func benchmarkActionPositionForSeed(b *testing.B, seed int64) *GamePosition {
	return benchmarkActionPositionForFactions(b, seed, models.FactionWitches, models.FactionEngineers)
}

func benchmarkActionPositionForFactions(b *testing.B, seed int64, first, second models.FactionType) *GamePosition {
	b.Helper()
	position, err := NewBaseGame(seed, first, second)
	if err != nil {
		b.Fatal(err)
	}
	for ply := 0; ply < 32; ply++ {
		if position.state.Phase == game.PhaseAction {
			return position
		}
		actions := position.LegalActions()
		if len(actions) == 0 {
			b.Fatalf("setup position has no actions at ply %d", ply)
		}
		if err := position.Apply(actions[(ply*7)%len(actions)]); err != nil {
			b.Fatalf("advance setup at ply %d: %v", ply, err)
		}
	}
	b.Fatal("setup did not reach the action phase")
	return nil
}

func BenchmarkPositionOperations(b *testing.B) {
	setup, err := NewBaseGame(73000, models.FactionNomads, models.FactionAuren)
	if err != nil {
		b.Fatal(err)
	}
	actionR1 := benchmarkActionPosition(b)
	actionR3 := benchmarkReachedActionPosition(b, 73002, models.FactionEngineers, models.FactionNomads, 3)
	actionR6 := benchmarkReachedActionPosition(b, 73003, models.FactionFakirs, models.FactionAuren, 6)
	pendingState := actionR3.StateClone()
	pendingState.GetPlayer("p1").Resources.Power.Bowl1 = 2
	pendingState.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{Amount: 2, VPCost: 1, FromPlayerID: "p0", EventID: 1}}
	pending, err := NewPosition(pendingState)
	if err != nil {
		b.Fatal(err)
	}

	cases := []struct {
		name     string
		position *GamePosition
	}{
		{"setup_nomads_auren", setup},
		{"round1_witches_engineers", actionR1},
		{"round3_engineers_nomads", actionR3},
		{"round6_fakirs_auren", actionR6},
		{"pending_leech_engineers_nomads", pending},
	}
	for _, benchmarkCase := range cases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			benchmarkPositionOperations(b, benchmarkCase.position)
		})
	}

	b.Run("natural_terminal_scoring", func(b *testing.B) {
		terminal := benchmarkNaturalTerminalPosition(b)
		player := PlayerID(terminal.state.TurnOrder[0])
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = terminal.Outcome(player)
		}
	})
}

func benchmarkReachedActionPosition(b *testing.B, seed int64, first, second models.FactionType, targetRound int) *GamePosition {
	b.Helper()
	position, err := NewBaseGame(seed, first, second)
	if err != nil {
		b.Fatal(err)
	}
	rng := rand.New(rand.NewSource(seed))
	for ply := 0; ply < 700 && !position.IsTerminal(); ply++ {
		if position.state.Phase == game.PhaseAction && position.state.Round == targetRound {
			return position
		}
		actions := position.LegalActions()
		if len(actions) == 0 {
			b.Fatalf("round %d fixture has no actions at ply %d", targetRound, ply)
		}
		if err := position.Apply(actions[rng.Intn(len(actions))]); err != nil {
			b.Fatal(err)
		}
	}
	b.Fatalf("did not reach round %d action phase naturally", targetRound)
	return nil
}

func benchmarkPositionOperations(b *testing.B, position *GamePosition) {
	b.Helper()
	actions := position.LegalActions()
	if len(actions) == 0 {
		b.Fatal("position has no legal actions")
	}
	action := representativeBenchmarkAction(actions)
	b.Run("Clone", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if position.Clone() == nil {
				b.Fatal("nil clone")
			}
		}
	})
	b.Run("LegalActions", func(b *testing.B) {
		b.ReportAllocs()
		candidates := productionCandidates(position.state, string(position.DecisionPlayer()))
		b.ReportMetric(float64(len(candidates)), "candidates")
		b.ReportMetric(float64(len(actions)), "legal-actions")
		byKind := make(map[game.ActionType]int)
		for _, candidate := range candidates {
			byKind[candidate.Kind]++
		}
		for kind, count := range byKind {
			b.ReportMetric(float64(count), fmt.Sprintf("kind-%d-candidates", kind))
		}
		for i := 0; i < b.N; i++ {
			if len(position.LegalActions()) == 0 {
				b.Fatal("no legal actions")
			}
		}
	})
	b.Run("CanonicalHashCold", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			uncached := &GamePosition{state: position.state}
			if uncached.CanonicalHash() == (Hash128{}) {
				b.Fatal("zero hash")
			}
		}
	})
	b.Run("CanonicalHashCached", func(b *testing.B) {
		if position.CanonicalHash() == (Hash128{}) {
			b.Fatal("zero hash")
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if position.CanonicalHash() == (Hash128{}) {
				b.Fatal("zero hash")
			}
		}
	})
	b.Run("Apply", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			clone := position.Clone()
			b.StartTimer()
			if err := clone.Apply(action); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func representativeBenchmarkAction(actions []SearchAction) SearchAction {
	for _, action := range actions {
		if action.Kind == game.ActionTransformAndBuild && action.Build {
			return action
		}
	}
	for _, action := range actions {
		if action.Kind == game.ActionUpgradeBuilding {
			return action
		}
	}
	return actions[0]
}

func benchmarkNaturalTerminalPosition(b *testing.B) *GamePosition {
	b.Helper()
	position, err := NewBaseGame(42, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		b.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	for ply := 0; ply < 700 && !position.IsTerminal(); ply++ {
		actions := position.LegalActions()
		if len(actions) == 0 {
			b.Fatalf("natural terminal fixture stuck at ply %d", ply)
		}
		if err := position.Apply(actions[rng.Intn(len(actions))]); err != nil {
			b.Fatal(err)
		}
	}
	if !position.IsTerminal() {
		b.Fatal("natural terminal fixture hit the 700-ply tripwire")
	}
	return position
}

func BenchmarkSearchBatch(b *testing.B) {
	for _, batchSize := range []int{1, 8, 32} {
		for _, simulations := range []int{1, 8, 32} {
			b.Run(fmt.Sprintf("batch=%d/simulations=%d", batchSize, simulations), func(b *testing.B) {
				positions := make([]Position, batchSize)
				for index := range positions {
					positions[index] = benchmarkActionPositionForSeed(b, 73001+int64(index))
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					search, err := NewMCTS(UniformEvaluator{}, SearchConfig{Simulations: simulations, Seed: int64(i)})
					if err != nil {
						b.Fatal(err)
					}
					if _, err := search.SearchBatch(context.Background(), positions); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
