package az

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type batchingArenaEvaluator struct {
	id       string
	maxBatch int
	err      error
}

func (e *batchingArenaEvaluator) ModelID() string { return e.id }

func (e *batchingArenaEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	if e.err != nil {
		return nil, e.err
	}
	if len(requests) > e.maxBatch {
		e.maxBatch = len(requests)
	}
	results := make([]EvaluationResult, len(requests))
	for index, request := range requests {
		results[index].PolicyLogits = make([]float32, len(request.Actions))
	}
	return results, nil
}

func TestArenaSummaryUsesPairedSeatsAndComparableDiagnostics(t *testing.T) {
	report := ArenaReport{
		FormatVersion: EvaluationFormatVersion,
		EngineCommit:  "test-commit",
		CandidateID:   "candidate",
		BaselineID:    "baseline",
		Config:        ArenaConfig{HoldoutSuiteID: "test-suite", CandidateSimulations: 8, BaselineSimulations: 4, GamesPerBatch: 2},
		Cases: []ArenaCase{
			{Seed: 1, First: models.FactionNomads, Second: models.FactionGiants},
			{Seed: 2, First: models.FactionNomads, Second: models.FactionGiants},
		},
		Games: []ArenaGame{
			testArenaGame(1, 0, [2]int{100, 90}),
			testArenaGame(1, 1, [2]int{80, 95}),
			testArenaGame(2, 0, [2]int{85, 90}),
			testArenaGame(2, 1, [2]int{92, 92}),
		},
	}
	report.Summary = summarizeArena(report)
	summary := report.Summary
	if summary.Games != 4 || summary.Wins.Count != 2 || summary.Draws.Count != 1 || summary.Losses.Count != 1 {
		t.Fatalf("unexpected WDL summary: %+v", summary)
	}
	if !summary.StrengthValid || summary.CompletePairs != 2 {
		t.Fatalf("complete paired report was not strength-valid: %+v", summary)
	}
	if summary.BySeat["seat_0"].Games != 2 || summary.BySeat["seat_1"].Games != 2 {
		t.Fatalf("seat swaps missing: %+v", summary.BySeat)
	}
	if len(summary.ByMatchup) != 2 {
		t.Fatalf("ordered matchup splits = %v", summary.ByMatchup)
	}
	if summary.Wins.Low95 < 0 || summary.Wins.High95 > 1 || summary.Wins.Low95 > summary.Wins.High95 {
		t.Fatalf("invalid paired interval: %+v", summary.Wins)
	}
	if summary.AverageWinningVP != 95 || summary.AverageLosingVP != 85 {
		t.Fatalf("winning/losing VP must include either agent: %+v", summary)
	}
	candidate := summary.DiagnosticsByAgent["candidate"]["round_1"]
	baseline := summary.DiagnosticsByAgent["baseline"]["round_1"]
	if candidate.Positions != baseline.Positions || candidate.Brier >= baseline.Brier {
		t.Fatalf("diagnostics are not comparable: candidate=%+v baseline=%+v", candidate, baseline)
	}
	actionKey := (SearchAction{Kind: game.ActionTransformAndBuild, Build: true}).Key()
	if summary.Round1Actions["candidate"][actionKey] != 2 || summary.Round1Actions["baseline"][actionKey] != 2 ||
		summary.Round1Buildings["candidate"][fmt.Sprint(models.BuildingDwelling)] != 2 || summary.Round1Buildings["baseline"][fmt.Sprint(models.BuildingDwelling)] != 2 {
		t.Fatalf("opening distributions are not split by agent: actions=%v buildings=%v", summary.Round1Actions, summary.Round1Buildings)
	}
	first, err := WriteArenaReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := WriteArenaReport(filepath.Dir(first.Path), report)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("content-addressed report changed: %+v %+v", first, second)
	}
	info, err := os.Stat(first.Path)
	if err != nil || info.Mode().Perm()&0o222 != 0 {
		t.Fatalf("report is not immutable: info=%v error=%v", info, err)
	}
	tampered := report
	tampered.Summary.Wins.Count++
	if _, err := WriteArenaReport(t.TempDir(), tampered); err == nil || !strings.Contains(err.Error(), "summary") {
		t.Fatalf("tampered summary was accepted: %v", err)
	}
}

func TestArenaFailureReportPersistsWithoutClaimingStrength(t *testing.T) {
	complete := testArenaGame(1, 0, [2]int{100, 90})
	failed := testArenaGame(1, 1, [2]int{})
	failed.Completed = false
	failed.Failure = "natural-completion tripwire at 1 plies"
	failed.TripwireHits = 1
	report := ArenaReport{
		FormatVersion: EvaluationFormatVersion, EngineCommit: "test", CandidateID: "candidate", BaselineID: "baseline",
		Config: ArenaConfig{HoldoutSuiteID: "test-suite", CandidateSimulations: 8, BaselineSimulations: 4, GamesPerBatch: 2},
		Cases:  []ArenaCase{{Seed: 1, First: models.FactionNomads, Second: models.FactionGiants}},
		Games:  []ArenaGame{complete, failed},
	}
	report.Summary = summarizeArena(report)
	if report.Summary.StrengthValid || report.Summary.NaturalCompletions != 1 || report.Summary.TripwireHits != 1 || report.Summary.CompletePairs != 0 {
		t.Fatalf("failure metrics were not preserved: %+v", report.Summary)
	}
	if _, err := WriteArenaReport(t.TempDir(), report); err != nil {
		t.Fatalf("durable failure report was rejected: %v", err)
	}
}

func TestArenaDuplicateStateInvalidatesStrength(t *testing.T) {
	first := testArenaGame(1, 0, [2]int{100, 90})
	first.DuplicateStates = 1
	report := ArenaReport{
		FormatVersion: EvaluationFormatVersion, EngineCommit: "test", CandidateID: "candidate", BaselineID: "baseline",
		Config: ArenaConfig{HoldoutSuiteID: "test-suite", CandidateSimulations: 8, BaselineSimulations: 4, GamesPerBatch: 2},
		Cases:  []ArenaCase{{Seed: 1, First: models.FactionNomads, Second: models.FactionGiants}},
		Games:  []ArenaGame{first, testArenaGame(1, 1, [2]int{90, 100})},
	}
	report.Summary = summarizeArena(report)
	if report.Summary.StrengthValid || report.Summary.DuplicateStates != 1 || report.Summary.Elo != 0 {
		t.Fatalf("duplicate state did not invalidate strength: %+v", report.Summary)
	}
	if _, err := WriteArenaReport(t.TempDir(), report); err != nil {
		t.Fatalf("duplicate diagnostic report was rejected: %v", err)
	}
}

func TestArenaReportAllowsSameNetworkAtDifferentSearchBudgets(t *testing.T) {
	report := ArenaReport{
		FormatVersion: EvaluationFormatVersion, EngineCommit: "test", CandidateID: "same-model", BaselineID: "same-model",
		Config: ArenaConfig{HoldoutSuiteID: "test-suite", CandidateSimulations: 32, BaselineSimulations: 1, GamesPerBatch: 2},
		Cases:  []ArenaCase{{Seed: 1, First: models.FactionNomads, Second: models.FactionGiants}},
		Games: []ArenaGame{
			testArenaGame(1, 0, [2]int{100, 90}),
			testArenaGame(1, 1, [2]int{90, 100}),
		},
	}
	report.Summary = summarizeArena(report)
	if _, err := WriteArenaReport(t.TempDir(), report); err != nil {
		t.Fatalf("same-network search-budget comparison was rejected: %v", err)
	}
}

func TestActionBuildingCoversEveryBaseRoundOneBuildPath(t *testing.T) {
	tests := []struct {
		action SearchAction
		want   models.BuildingType
		ok     bool
	}{
		{SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionSpade1, Build: true}, models.BuildingDwelling, true},
		{SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionWitchesRide}, models.BuildingDwelling, true},
		{SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionSwarmlingsUpgrade}, models.BuildingTradingHouse, true},
		{SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionNomadsSandstorm, Build: true}, models.BuildingDwelling, true},
		{SearchAction{Kind: game.ActionBuildHalflingsDwelling}, models.BuildingDwelling, true},
		{SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionNomadsSandstorm}, models.BuildingDwelling, false},
	}
	for _, test := range tests {
		got, ok := actionBuilding(test.action)
		if got != test.want || ok != test.ok {
			t.Fatalf("action %s building=(%v,%v), want (%v,%v)", test.action.Key(), got, ok, test.want, test.ok)
		}
	}
}

func TestPairedIntervalsRemainUncertainAtExtremeSmallSamples(t *testing.T) {
	allWins := []ArenaGame{
		testArenaGame(1, 0, [2]int{100, 90}),
		testArenaGame(1, 1, [2]int{90, 100}),
	}
	wins := pairedIntervals95(allWins)
	if wins.win[0] >= 1 || wins.score[0] >= 1 {
		t.Fatalf("one all-win pair asserted certainty: %+v", wins)
	}
	allLosses := []ArenaGame{
		testArenaGame(1, 0, [2]int{90, 100}),
		testArenaGame(1, 1, [2]int{100, 90}),
	}
	losses := pairedIntervals95(allLosses)
	if losses.win[1] <= 0 || losses.score[1] <= 0 {
		t.Fatalf("one all-loss pair asserted certainty: %+v", losses)
	}
}

func TestHoldoutSeedsAreUnique(t *testing.T) {
	fullSuite, err := HoldoutSuiteIdentity(HoldoutCases())
	if err != nil || fullSuite != HoldoutSuiteID {
		t.Fatalf("full holdout identity=(%q,%v)", fullSuite, err)
	}
	prefixSuite, err := HoldoutSuiteIdentity(HoldoutCases()[:1])
	if err != nil || prefixSuite != HoldoutSuiteID+"-prefix-1" {
		t.Fatalf("prefix holdout identity=(%q,%v)", prefixSuite, err)
	}
	seen := make(map[int64]bool)
	factions := make(map[models.FactionType]bool)
	pairs := make(map[[2]models.FactionType]bool)
	for _, item := range HoldoutCases() {
		if seen[item.Seed] || !IsHoldoutSeed(item.Seed) {
			t.Fatalf("invalid holdout seed %d", item.Seed)
		}
		if (int(item.First)-1)/2 == (int(item.Second)-1)/2 {
			t.Fatalf("holdout contains same-home factions: %+v", item)
		}
		seen[item.Seed] = true
		factions[item.First], factions[item.Second] = true, true
		pairs[[2]models.FactionType{item.First, item.Second}] = true
	}
	if len(seen) != 28 || len(factions) != 14 {
		t.Fatalf("holdout coverage: cases=%d factions=%d", len(seen), len(factions))
	}
	for pair := range pairs {
		if !pairs[[2]models.FactionType{pair[1], pair[0]}] {
			t.Fatalf("holdout is missing reverse faction order for %v", pair)
		}
	}
	if IsHoldoutSeed(7301) {
		t.Fatal("ordinary self-play seed was reserved")
	}
	altered := HoldoutCases()[:1]
	altered[0].Seed++
	if _, err := HoldoutSuiteIdentity(altered); err == nil {
		t.Fatal("altered cases retained an official holdout identity")
	}
	item := HoldoutCases()[0]
	position, err := NewBaseGame(item.Seed, item.First, item.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateSelfPlay(context.Background(), []*GamePosition{position}, []int64{item.Seed}, UniformEvaluator{}, SelfPlayConfig{
		Search: SearchConfig{Simulations: 1}, EngineCommit: "test",
	}); err == nil || !strings.Contains(err.Error(), "reserved for evaluation holdout") {
		t.Fatalf("self-play accepted holdout seed: %v", err)
	}
}

func TestArenaUsesPerAgentSearchBudgets(t *testing.T) {
	config := ArenaConfig{CandidateSimulations: 64, BaselineSimulations: 8}
	if got := searchBudgetForAgent(config, arenaCandidateAgent); got != 64 {
		t.Fatalf("candidate budget=%d", got)
	}
	if got := searchBudgetForAgent(config, arenaBaselineAgent); got != 8 {
		t.Fatalf("baseline budget=%d", got)
	}
}

func TestArenaRejectsNegativeBatchBound(t *testing.T) {
	evaluator := &batchingArenaEvaluator{id: "test"}
	_, err := RunPairedArena(context.Background(), evaluator, evaluator, HoldoutCases()[:1], ArenaConfig{
		HoldoutSuiteID: "test-suite", CandidateSimulations: 1, BaselineSimulations: 1,
		GamesPerBatch: -2,
	}, "test")
	if err == nil || !strings.Contains(err.Error(), "cannot be negative") {
		t.Fatalf("negative arena batch bound was accepted: %v", err)
	}
}

func TestBatchedArenaMatchesIndependentGames(t *testing.T) {
	config := ArenaConfig{
		HoldoutSuiteID: "test-suite", CandidateSimulations: 1, BaselineSimulations: 1,
		GamesPerBatch: 2, CPUCT: 1.5, MaxPlies: 2000,
	}
	makeRuntimes := func() []arenaRuntime {
		runtimes := make([]arenaRuntime, 0, 4)
		for _, arenaCase := range HoldoutCases()[:2] {
			for candidateSeat := 0; candidateSeat < 2; candidateSeat++ {
				position, err := NewBaseGame(arenaCase.Seed, arenaCase.First, arenaCase.Second)
				if err != nil {
					t.Fatal(err)
				}
				agents := [2]string{arenaBaselineAgent, arenaBaselineAgent}
				agents[candidateSeat] = arenaCandidateAgent
				runtime, err := newArenaRuntime(position, arenaCase, agents, candidateSeat)
				if err != nil {
					t.Fatal(err)
				}
				runtimes = append(runtimes, runtime)
			}
		}
		return runtimes
	}
	batchedEvaluator := &batchingArenaEvaluator{id: "batched"}
	report, err := RunPairedArena(
		context.Background(), batchedEvaluator, batchedEvaluator, HoldoutCases()[:2], config, "test",
	)
	if err != nil {
		t.Fatal(err)
	}
	batched := report.Games
	serialEvaluator := &batchingArenaEvaluator{id: "serial"}
	serial := make([]ArenaGame, 0, len(batched))
	for _, runtime := range makeRuntimes() {
		results, err := playArenaBatch(context.Background(), []arenaRuntime{runtime}, map[string]Evaluator{
			arenaCandidateAgent: serialEvaluator, arenaBaselineAgent: serialEvaluator,
		}, config)
		if err != nil {
			t.Fatal(err)
		}
		serial = append(serial, results[0])
	}
	if !reflect.DeepEqual(batched, serial) {
		t.Fatal("batched arena changed deterministic game results")
	}
	if batchedEvaluator.maxBatch != config.GamesPerBatch || serialEvaluator.maxBatch != 1 {
		t.Fatalf("inference batches: batched=%d serial=%d", batchedEvaluator.maxBatch, serialEvaluator.maxBatch)
	}
}

func TestArenaInfrastructureFailureIsPersistable(t *testing.T) {
	evaluator := &batchingArenaEvaluator{id: "broken", err: fmt.Errorf("inference unavailable")}
	report, err := RunPairedArena(context.Background(), evaluator, evaluator, HoldoutCases()[:2], ArenaConfig{
		HoldoutSuiteID: "test-suite", CandidateSimulations: 1, BaselineSimulations: 1,
		GamesPerBatch: 2,
	}, "test")
	if err == nil || !strings.Contains(err.Error(), "inference unavailable") {
		t.Fatalf("missing infrastructure error: %v", err)
	}
	if len(report.Games) != 4 || report.Summary.InfrastructureFailures != 4 || report.Summary.StrengthValid {
		t.Fatalf("infrastructure failure was not preserved: %+v", report.Summary)
	}
	if _, err := WriteArenaReport(t.TempDir(), report); err != nil {
		t.Fatalf("infrastructure failure report was rejected: %v", err)
	}
}

func testArenaGame(seed int64, candidateSeat int, finalVP [2]int) ArenaGame {
	factions := [2]models.FactionType{models.FactionNomads, models.FactionGiants}
	agents := [2]string{"baseline", "baseline"}
	agents[candidateSeat] = "candidate"
	seatMargin := finalVP[0] - finalVP[1]
	target := outcomeFromMargin(seatMargin)
	return ArenaGame{
		Seed: seed, Factions: factions, AgentBySeat: agents, CandidateSeat: candidateSeat,
		FinalVP: finalVP, PlyCount: 1, Completed: true,
		Decisions: []ArenaDecision{{
			Seat: 0, Phase: game.PhaseAction, Round: 1,
			Action: SearchAction{Kind: game.ActionTransformAndBuild, Build: true},
			Predictions: []AgentPrediction{
				{AgentID: "candidate", Value: target, PolicyEntropy: 0.5},
				{AgentID: "baseline", Value: -target, PolicyEntropy: 1.0},
			},
		}},
	}
}
