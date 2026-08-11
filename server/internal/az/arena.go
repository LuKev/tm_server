package az

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

const (
	EvaluationFormatVersion = 4
	HoldoutSuiteID          = "tm-v0-paired-v2"
	arenaCandidateAgent     = "candidate"
	arenaBaselineAgent      = "baseline"
)

type ArenaCase struct {
	Seed   int64              `json:"seed"`
	First  models.FactionType `json:"first_faction"`
	Second models.FactionType `json:"second_faction"`
}

var holdoutCases = []ArenaCase{
	{Seed: 910000001, First: models.FactionNomads, Second: models.FactionGiants},
	{Seed: 910000002, First: models.FactionFakirs, Second: models.FactionSwarmlings},
	{Seed: 910000003, First: models.FactionChaosMagicians, Second: models.FactionWitches},
	{Seed: 910000004, First: models.FactionMermaids, Second: models.FactionHalflings},
	{Seed: 910000005, First: models.FactionAuren, Second: models.FactionCultists},
	{Seed: 910000006, First: models.FactionAlchemists, Second: models.FactionEngineers},
	{Seed: 910000007, First: models.FactionDarklings, Second: models.FactionDwarves},
	{Seed: 910000008, First: models.FactionEngineers, Second: models.FactionNomads},
	{Seed: 910000009, First: models.FactionGiants, Second: models.FactionNomads},
	{Seed: 910000010, First: models.FactionSwarmlings, Second: models.FactionFakirs},
	{Seed: 910000011, First: models.FactionWitches, Second: models.FactionChaosMagicians},
	{Seed: 910000012, First: models.FactionHalflings, Second: models.FactionMermaids},
	{Seed: 910000013, First: models.FactionCultists, Second: models.FactionAuren},
	{Seed: 910000014, First: models.FactionEngineers, Second: models.FactionAlchemists},
	{Seed: 910000015, First: models.FactionDwarves, Second: models.FactionDarklings},
	{Seed: 910000016, First: models.FactionNomads, Second: models.FactionEngineers},
	{Seed: 910000017, First: models.FactionFakirs, Second: models.FactionMermaids},
	{Seed: 910000018, First: models.FactionChaosMagicians, Second: models.FactionAlchemists},
	{Seed: 910000019, First: models.FactionGiants, Second: models.FactionDarklings},
	{Seed: 910000020, First: models.FactionSwarmlings, Second: models.FactionAuren},
	{Seed: 910000021, First: models.FactionWitches, Second: models.FactionDwarves},
	{Seed: 910000022, First: models.FactionHalflings, Second: models.FactionEngineers},
	{Seed: 910000023, First: models.FactionMermaids, Second: models.FactionFakirs},
	{Seed: 910000024, First: models.FactionAlchemists, Second: models.FactionChaosMagicians},
	{Seed: 910000025, First: models.FactionDarklings, Second: models.FactionGiants},
	{Seed: 910000026, First: models.FactionAuren, Second: models.FactionSwarmlings},
	{Seed: 910000027, First: models.FactionDwarves, Second: models.FactionWitches},
	{Seed: 910000028, First: models.FactionEngineers, Second: models.FactionHalflings},
}

func HoldoutCases() []ArenaCase { return append([]ArenaCase(nil), holdoutCases...) }

func HoldoutSuiteIdentity(cases []ArenaCase) (string, error) {
	if len(cases) == 0 || len(cases) > len(holdoutCases) || !reflect.DeepEqual(cases, holdoutCases[:len(cases)]) {
		return "", fmt.Errorf("cases are not an ordered prefix of %s", HoldoutSuiteID)
	}
	if len(cases) == len(holdoutCases) {
		return HoldoutSuiteID, nil
	}
	return fmt.Sprintf("%s-prefix-%d", HoldoutSuiteID, len(cases)), nil
}

func IsHoldoutSeed(seed int64) bool {
	for _, item := range holdoutCases {
		if item.Seed == seed {
			return true
		}
	}
	return false
}

type ArenaConfig struct {
	HoldoutSuiteID       string  `json:"holdout_suite_id"`
	CandidateSimulations int     `json:"candidate_simulations"`
	BaselineSimulations  int     `json:"baseline_simulations"`
	CPUCT                float64 `json:"c_puct"`
	MaxPlies             int     `json:"max_plies"`
}

type AgentPrediction struct {
	AgentID       string  `json:"agent_id"`
	Value         float32 `json:"value"`
	PolicyEntropy float64 `json:"policy_entropy"`
}

type ArenaDecision struct {
	Seat        int               `json:"seat"`
	Phase       game.GamePhase    `json:"phase"`
	Round       int               `json:"round"`
	StateHash   string            `json:"state_hash"`
	Action      SearchAction      `json:"action"`
	Predictions []AgentPrediction `json:"predictions"`
}

type ArenaGame struct {
	Seed            int64                 `json:"seed"`
	Factions        [2]models.FactionType `json:"factions"`
	AgentBySeat     [2]string             `json:"agent_by_seat"`
	CandidateSeat   int                   `json:"candidate_seat"`
	FinalVP         [2]int                `json:"final_vp"`
	PlyCount        int                   `json:"ply_count"`
	Completed       bool                  `json:"completed"`
	Failure         string                `json:"failure,omitempty"`
	InvalidActions  int                   `json:"invalid_actions"`
	TripwireHits    int                   `json:"tripwire_hits"`
	DuplicateStates int                   `json:"duplicate_states"`
	Decisions       []ArenaDecision       `json:"decisions"`
}

type Proportion struct {
	Count  int     `json:"count"`
	Rate   float64 `json:"rate"`
	Low95  float64 `json:"low_95"`
	High95 float64 `json:"high_95"`
}

type SplitSummary struct {
	Games     int     `json:"games"`
	Wins      int     `json:"wins"`
	Draws     int     `json:"draws"`
	Losses    int     `json:"losses"`
	AvgMargin float64 `json:"average_vp_margin"`
}

type PhaseDiagnostic struct {
	Positions     int     `json:"positions"`
	Brier         float64 `json:"brier"`
	PolicyEntropy float64 `json:"policy_entropy"`
}

type ArenaSummary struct {
	RatingMethod       string                                `json:"rating_method"`
	ConfidenceMethod   string                                `json:"confidence_interval_method"`
	CalibrationMetric  string                                `json:"calibration_metric"`
	EntropyUnits       string                                `json:"policy_entropy_units"`
	Games              int                                   `json:"games"`
	CompletePairs      int                                   `json:"complete_pairs"`
	StrengthValid      bool                                  `json:"strength_valid"`
	Wins               Proportion                            `json:"wins"`
	Draws              Proportion                            `json:"draws"`
	Losses             Proportion                            `json:"losses"`
	ScoreRate          float64                               `json:"score_rate"`
	ScoreLow95         float64                               `json:"score_low_95"`
	ScoreHigh95        float64                               `json:"score_high_95"`
	PairedUncertainty  string                                `json:"paired_uncertainty_method"`
	Elo                float64                               `json:"elo_vs_baseline"`
	AverageCandidateVP float64                               `json:"average_candidate_vp"`
	AverageBaselineVP  float64                               `json:"average_baseline_vp"`
	AverageWinningVP   float64                               `json:"average_winning_vp"`
	AverageLosingVP    float64                               `json:"average_losing_vp"`
	AverageVPMargin    float64                               `json:"average_vp_margin"`
	ByMatchup          map[string]SplitSummary               `json:"by_ordered_matchup"`
	BySeat             map[string]SplitSummary               `json:"by_candidate_seat"`
	SetupActions       map[string]map[string]int             `json:"setup_actions_by_agent"`
	Round1Actions      map[string]map[string]int             `json:"round_1_actions_by_agent"`
	Round1Buildings    map[string]map[string]int             `json:"round_1_buildings_by_agent"`
	DiagnosticsByAgent map[string]map[string]PhaseDiagnostic `json:"diagnostics_by_agent_and_phase"`
	NaturalCompletions int                                   `json:"natural_completions"`
	InvalidActions     int                                   `json:"invalid_actions"`
	DuplicateStates    int                                   `json:"duplicate_states"`
	TripwireHits       int                                   `json:"tripwire_hits"`
}

type ArenaReport struct {
	FormatVersion             int          `json:"format_version"`
	EngineCommit              string       `json:"engine_commit"`
	CandidateID               string       `json:"candidate_id"`
	CandidateCheckpointSHA256 string       `json:"candidate_checkpoint_sha256,omitempty"`
	BaselineID                string       `json:"baseline_id"`
	BaselineCheckpointSHA256  string       `json:"baseline_checkpoint_sha256,omitempty"`
	Config                    ArenaConfig  `json:"config"`
	Cases                     []ArenaCase  `json:"cases"`
	Games                     []ArenaGame  `json:"games"`
	Summary                   ArenaSummary `json:"summary"`
}

func RunPairedArena(ctx context.Context, candidate, baseline Evaluator, cases []ArenaCase, config ArenaConfig, engineCommit string) (ArenaReport, error) {
	candidateID, err := evaluatorModelID(candidate)
	if err != nil {
		return ArenaReport{}, fmt.Errorf("candidate: %w", err)
	}
	baselineID, err := evaluatorModelID(baseline)
	if err != nil {
		return ArenaReport{}, fmt.Errorf("baseline: %w", err)
	}
	if len(cases) == 0 {
		return ArenaReport{}, fmt.Errorf("arena requires paired setup cases")
	}
	if config.HoldoutSuiteID == "" {
		return ArenaReport{}, fmt.Errorf("arena holdout suite identity is required")
	}
	if err := validateHoldoutSuiteIdentity(config.HoldoutSuiteID, cases); err != nil {
		return ArenaReport{}, err
	}
	if config.CandidateSimulations <= 0 || config.BaselineSimulations <= 0 {
		return ArenaReport{}, fmt.Errorf("candidate and baseline arena simulations must be positive")
	}
	if engineCommit == "" {
		return ArenaReport{}, fmt.Errorf("arena engine commit is required")
	}
	if config.CPUCT == 0 {
		config.CPUCT = 1.5
	}
	if config.MaxPlies <= 0 {
		config.MaxPlies = 2000
	}
	agents := map[string]Evaluator{arenaCandidateAgent: candidate, arenaBaselineAgent: baseline}
	games := make([]ArenaGame, 0, len(cases)*2)
	for _, item := range cases {
		for candidateSeat := 0; candidateSeat < 2; candidateSeat++ {
			position, err := NewBaseGame(item.Seed, item.First, item.Second)
			if err != nil {
				return ArenaReport{}, err
			}
			agentBySeat := [2]string{arenaBaselineAgent, arenaBaselineAgent}
			agentBySeat[candidateSeat] = arenaCandidateAgent
			gameResult, err := playArenaGame(ctx, position, item, agentBySeat, candidateSeat, agents, config)
			if err != nil {
				return ArenaReport{}, fmt.Errorf("seed %d candidate seat %d: %w", item.Seed, candidateSeat, err)
			}
			games = append(games, gameResult)
		}
	}
	report := ArenaReport{
		FormatVersion: EvaluationFormatVersion, EngineCommit: engineCommit,
		CandidateID: candidateID, CandidateCheckpointSHA256: evaluatorCheckpointHash(candidate),
		BaselineID: baselineID, BaselineCheckpointSHA256: evaluatorCheckpointHash(baseline),
		Config: config, Cases: append([]ArenaCase(nil), cases...), Games: games,
	}
	report.Summary = summarizeArena(report)
	return report, nil
}

func evaluatorCheckpointHash(evaluator Evaluator) string {
	if checkpoint, ok := evaluator.(checkpointEvaluator); ok {
		return checkpoint.CheckpointSHA256()
	}
	return ""
}

func evaluatorModelID(evaluator Evaluator) (string, error) {
	identified, ok := evaluator.(identifiedEvaluator)
	if !ok || identified.ModelID() == "" {
		return "", fmt.Errorf("evaluator has no authoritative model identity")
	}
	return identified.ModelID(), nil
}

func playArenaGame(ctx context.Context, position *GamePosition, arenaCase ArenaCase, agentBySeat [2]string, candidateSeat int, agents map[string]Evaluator, config ArenaConfig) (ArenaGame, error) {
	state := position.StateClone()
	playerIDs := [2]string{state.TurnOrder[0], state.TurnOrder[1]}
	seatByID := map[string]int{playerIDs[0]: 0, playerIDs[1]: 1}
	result := ArenaGame{
		Seed: arenaCase.Seed, Factions: [2]models.FactionType{arenaCase.First, arenaCase.Second},
		AgentBySeat: agentBySeat, CandidateSeat: candidateSeat,
	}
	seen := make(map[Hash128]bool)
	for !position.IsTerminal() {
		if len(result.Decisions) >= config.MaxPlies {
			result.PlyCount = len(result.Decisions)
			result.TripwireHits = 1
			result.Failure = fmt.Sprintf("natural-completion tripwire at %d plies", config.MaxPlies)
			return result, nil
		}
		owner := string(position.DecisionPlayer())
		seat, ok := seatByID[owner]
		if !ok {
			return ArenaGame{}, fmt.Errorf("unknown decision owner %q", owner)
		}
		hash := position.CanonicalHash()
		if seen[hash] {
			result.DuplicateStates++
		}
		seen[hash] = true
		actions := position.LegalActions()
		if len(actions) == 0 {
			result.PlyCount = len(result.Decisions)
			result.InvalidActions = 1
			result.Failure = "non-terminal position has no legal actions"
			return result, nil
		}
		predictions := make([]AgentPrediction, 0, len(agents))
		agentIDs := make([]string, 0, len(agents))
		for agentID := range agents {
			agentIDs = append(agentIDs, agentID)
		}
		sort.Strings(agentIDs)
		for _, agentID := range agentIDs {
			evaluations, err := agents[agentID].Evaluate(ctx, []EvaluationRequest{{Position: position.Clone(), Actions: actions}})
			if err != nil || len(evaluations) != 1 {
				return ArenaGame{}, fmt.Errorf("evaluate %s: results=%d error=%v", agentID, len(evaluations), err)
			}
			if err := validateEvaluation(evaluations[0], len(actions)); err != nil {
				return ArenaGame{}, fmt.Errorf("evaluate %s: %w", agentID, err)
			}
			predictions = append(predictions, AgentPrediction{
				AgentID: agentID, Value: evaluations[0].Value,
				PolicyEntropy: probabilityEntropy(softmax(evaluations[0].PolicyLogits)),
			})
		}
		search, err := NewMCTS(agents[agentBySeat[seat]], SearchConfig{
			Simulations: simulationsForSeat(config, seat, candidateSeat), CPUCT: config.CPUCT,
			Seed: arenaCase.Seed + int64(len(result.Decisions)),
		})
		if err != nil {
			return ArenaGame{}, err
		}
		searchResult, err := search.Search(ctx, position)
		if err != nil {
			return ArenaGame{}, err
		}
		action, err := searchResult.SelectAction(0, nil)
		if err != nil {
			return ArenaGame{}, err
		}
		before := position.StateClone()
		result.Decisions = append(result.Decisions, ArenaDecision{
			Seat: seat, Phase: before.Phase, Round: before.Round, StateHash: hash.String(),
			Action: action, Predictions: predictions,
		})
		if err := position.Apply(action); err != nil {
			result.PlyCount = len(result.Decisions)
			result.InvalidActions = 1
			result.Failure = fmt.Sprintf("apply emitted action: %v", err)
			return result, nil
		}
	}
	for seat, playerID := range playerIDs {
		vp, ok := position.FinalVP(PlayerID(playerID))
		if !ok {
			return ArenaGame{}, fmt.Errorf("missing final VP for seat %d", seat)
		}
		result.FinalVP[seat] = vp
	}
	result.PlyCount = len(result.Decisions)
	result.Completed = true
	return result, nil
}

func simulationsForSeat(config ArenaConfig, seat, candidateSeat int) int {
	if seat == candidateSeat {
		return config.CandidateSimulations
	}
	return config.BaselineSimulations
}

func probabilityEntropy(probabilities []float64) float64 {
	entropy := 0.0
	for _, probability := range probabilities {
		if probability > 0 {
			entropy -= probability * math.Log(probability)
		}
	}
	return entropy
}

type splitAccumulator struct {
	games, wins, draws, losses int
	margin                     float64
}

type diagnosticAccumulator struct {
	count   int
	brier   float64
	entropy float64
}

func summarizeArena(report ArenaReport) ArenaSummary {
	summary := ArenaSummary{
		RatingMethod:      "400*log10(smoothed_score/(1-smoothed_score)); Jeffreys 0.5 pseudo-game",
		ConfidenceMethod:  "paired-case mean with a finite-sample Hoeffding bound, 95%",
		CalibrationMetric: "Brier score on p=(value+1)/2 against y=(outcome+1)/2",
		EntropyUnits:      "natural-log nats over network legal-action priors",
		Games:             len(report.Games), ByMatchup: make(map[string]SplitSummary), BySeat: make(map[string]SplitSummary),
		SetupActions: make(map[string]map[string]int), Round1Actions: make(map[string]map[string]int), Round1Buildings: make(map[string]map[string]int),
		DiagnosticsByAgent: make(map[string]map[string]PhaseDiagnostic),
	}
	matchup := make(map[string]*splitAccumulator)
	seatSplits := make(map[string]*splitAccumulator)
	diagnostics := make(map[string]map[string]*diagnosticAccumulator)
	wins, draws, losses := 0, 0, 0
	winningVP, losingVP, winningCount, losingCount := 0.0, 0.0, 0, 0
	for _, gameResult := range report.Games {
		summary.InvalidActions += gameResult.InvalidActions
		summary.TripwireHits += gameResult.TripwireHits
		summary.DuplicateStates += gameResult.DuplicateStates
		if !gameResult.Completed {
			continue
		}
		summary.NaturalCompletions++
		candidateSeat := gameResult.CandidateSeat
		baselineSeat := 1 - candidateSeat
		candidateVP := gameResult.FinalVP[candidateSeat]
		baselineVP := gameResult.FinalVP[baselineSeat]
		margin := candidateVP - baselineVP
		summary.AverageCandidateVP += float64(candidateVP)
		summary.AverageBaselineVP += float64(baselineVP)
		summary.AverageVPMargin += float64(margin)
		outcome := 0
		if margin > 0 {
			wins++
			outcome = 1
		} else if margin < 0 {
			losses++
			outcome = -1
		}
		if margin != 0 {
			winningVP += float64(max(candidateVP, baselineVP))
			losingVP += float64(min(candidateVP, baselineVP))
			winningCount++
			losingCount++
		} else {
			draws++
		}
		matchupKey := fmt.Sprintf("%s_vs_%s", gameResult.Factions[candidateSeat], gameResult.Factions[baselineSeat])
		seatKey := fmt.Sprintf("seat_%d", candidateSeat)
		if matchup[matchupKey] == nil {
			matchup[matchupKey] = &splitAccumulator{}
		}
		accumulateSplit(matchup[matchupKey], outcome, margin)
		if seatSplits[seatKey] == nil {
			seatSplits[seatKey] = &splitAccumulator{}
		}
		accumulateSplit(seatSplits[seatKey], outcome, margin)
		for _, decision := range gameResult.Decisions {
			agentID := gameResult.AgentBySeat[decision.Seat]
			actionKey := decision.Action.Key()
			if decision.Phase == game.PhaseSetup {
				incrementAgentMetric(summary.SetupActions, agentID, actionKey)
			}
			if decision.Phase == game.PhaseAction && decision.Round == 1 {
				incrementAgentMetric(summary.Round1Actions, agentID, actionKey)
				if building, ok := actionBuilding(decision.Action); ok {
					incrementAgentMetric(summary.Round1Buildings, agentID, fmt.Sprint(building))
				}
			}
			seatMargin := gameResult.FinalVP[decision.Seat] - gameResult.FinalVP[1-decision.Seat]
			targetValue := float64(outcomeFromMargin(seatMargin))
			phase := phaseBucket(decision.Phase, decision.Round)
			for _, prediction := range decision.Predictions {
				if diagnostics[prediction.AgentID] == nil {
					diagnostics[prediction.AgentID] = make(map[string]*diagnosticAccumulator)
				}
				if diagnostics[prediction.AgentID][phase] == nil {
					diagnostics[prediction.AgentID][phase] = &diagnosticAccumulator{}
				}
				item := diagnostics[prediction.AgentID][phase]
				item.count++
				error := (float64(prediction.Value) - targetValue) / 2
				item.brier += error * error
				item.entropy += prediction.PolicyEntropy
			}
		}
	}
	n := float64(summary.NaturalCompletions)
	intervals := pairedIntervals95(report.Games)
	summary.CompletePairs = intervals.pairs
	summary.StrengthValid = summary.NaturalCompletions == len(report.Games) && intervals.pairs*2 == len(report.Games) && summary.InvalidActions == 0 && summary.TripwireHits == 0 && summary.DuplicateStates == 0
	if n == 0 {
		summary.Wins = Proportion{Low95: intervals.win[0], High95: intervals.win[1]}
		summary.Draws = Proportion{Low95: intervals.draw[0], High95: intervals.draw[1]}
		summary.Losses = Proportion{Low95: intervals.loss[0], High95: intervals.loss[1]}
		summary.ScoreLow95, summary.ScoreHigh95 = intervals.score[0], intervals.score[1]
		summary.PairedUncertainty = summary.ConfidenceMethod
		return summary
	}
	summary.Wins = Proportion{Count: wins, Rate: float64(wins) / n, Low95: intervals.win[0], High95: intervals.win[1]}
	summary.Draws = Proportion{Count: draws, Rate: float64(draws) / n, Low95: intervals.draw[0], High95: intervals.draw[1]}
	summary.Losses = Proportion{Count: losses, Rate: float64(losses) / n, Low95: intervals.loss[0], High95: intervals.loss[1]}
	summary.ScoreRate = (float64(wins) + 0.5*float64(draws)) / n
	summary.ScoreLow95, summary.ScoreHigh95 = intervals.score[0], intervals.score[1]
	summary.PairedUncertainty = summary.ConfidenceMethod
	if summary.StrengthValid {
		smoothedScore := (float64(wins) + 0.5*float64(draws) + 0.5) / (n + 1)
		summary.Elo = 400 * math.Log10(smoothedScore/(1-smoothedScore))
	}
	summary.AverageCandidateVP /= n
	summary.AverageBaselineVP /= n
	summary.AverageVPMargin /= n
	if winningCount > 0 {
		summary.AverageWinningVP = winningVP / float64(winningCount)
	}
	if losingCount > 0 {
		summary.AverageLosingVP = losingVP / float64(losingCount)
	}
	for key, value := range matchup {
		summary.ByMatchup[key] = finishSplit(value)
	}
	for key, value := range seatSplits {
		summary.BySeat[key] = finishSplit(value)
	}
	for agentID, phases := range diagnostics {
		summary.DiagnosticsByAgent[agentID] = make(map[string]PhaseDiagnostic)
		for phase, value := range phases {
			summary.DiagnosticsByAgent[agentID][phase] = PhaseDiagnostic{
				Positions: value.count, Brier: value.brier / float64(value.count), PolicyEntropy: value.entropy / float64(value.count),
			}
		}
	}
	return summary
}

type pairedIntervals struct {
	win, draw, loss, score [2]float64
	pairs                  int
}

func pairedIntervals95(games []ArenaGame) pairedIntervals {
	bySeed := make(map[int64][]int)
	for _, gameResult := range games {
		if !gameResult.Completed {
			continue
		}
		seat := gameResult.CandidateSeat
		margin := gameResult.FinalVP[seat] - gameResult.FinalVP[1-seat]
		outcome := 1
		if margin > 0 {
			outcome = 0
		} else if margin < 0 {
			outcome = 2
		}
		bySeed[gameResult.Seed] = append(bySeed[gameResult.Seed], outcome)
	}
	seeds := make([]int64, 0, len(bySeed))
	for seed := range bySeed {
		seeds = append(seeds, seed)
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	pairs := make([][3]int, 0, len(seeds))
	for _, seed := range seeds {
		outcomes := bySeed[seed]
		if len(outcomes) != 2 {
			continue
		}
		counts := [3]int{}
		for _, outcome := range outcomes {
			counts[outcome]++
		}
		pairs = append(pairs, counts)
	}
	if len(pairs) == 0 {
		return pairedIntervals{win: [2]float64{0, 1}, draw: [2]float64{0, 1}, loss: [2]float64{0, 1}, score: [2]float64{0, 1}}
	}
	totalCounts := [3]int{}
	for _, pair := range pairs {
		for outcome := range totalCounts {
			totalCounts[outcome] += pair[outcome]
		}
	}
	totalGames := float64(2 * len(pairs))
	interval := func(mean float64) [2]float64 {
		epsilon := math.Sqrt(math.Log(2/0.05) / (2 * float64(len(pairs))))
		return [2]float64{math.Max(0, mean-epsilon), math.Min(1, mean+epsilon)}
	}
	winMean := float64(totalCounts[0]) / totalGames
	drawMean := float64(totalCounts[1]) / totalGames
	lossMean := float64(totalCounts[2]) / totalGames
	scoreMean := (float64(totalCounts[0]) + 0.5*float64(totalCounts[1])) / totalGames
	return pairedIntervals{
		win: interval(winMean), draw: interval(drawMean), loss: interval(lossMean), score: interval(scoreMean), pairs: len(pairs),
	}
}

func incrementAgentMetric(metric map[string]map[string]int, agentID, key string) {
	if metric[agentID] == nil {
		metric[agentID] = make(map[string]int)
	}
	metric[agentID][key]++
}

func accumulateSplit(value *splitAccumulator, outcome, margin int) {
	value.games++
	value.margin += float64(margin)
	if outcome > 0 {
		value.wins++
	} else if outcome < 0 {
		value.losses++
	} else {
		value.draws++
	}
}

func finishSplit(value *splitAccumulator) SplitSummary {
	return SplitSummary{Games: value.games, Wins: value.wins, Draws: value.draws, Losses: value.losses, AvgMargin: value.margin / float64(value.games)}
}

func phaseBucket(phase game.GamePhase, round int) string {
	if phase == game.PhaseSetup {
		return "setup"
	}
	return fmt.Sprintf("round_%d", round)
}

func actionBuilding(action SearchAction) (models.BuildingType, bool) {
	switch action.Kind {
	case game.ActionUpgradeBuilding:
		return action.Building, true
	case game.ActionTransformAndBuild, game.ActionBuildHalflingsDwelling:
		return models.BuildingDwelling, action.Kind == game.ActionBuildHalflingsDwelling || action.Build
	case game.ActionPowerAction:
		return models.BuildingDwelling, action.Build && (action.Power == game.PowerActionSpade1 || action.Power == game.PowerActionSpade2)
	case game.ActionSpecialAction:
		switch action.Special {
		case game.SpecialActionWitchesRide:
			return models.BuildingDwelling, true
		case game.SpecialActionSwarmlingsUpgrade:
			return models.BuildingTradingHouse, true
		case game.SpecialActionGiantsTransform, game.SpecialActionNomadsSandstorm, game.SpecialActionBonusCardSpade, game.SpecialActionSelkiesStronghold:
			return models.BuildingDwelling, action.Build
		}
	}
	return models.BuildingDwelling, false
}

type ReportRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

func WriteArenaReport(directory string, report ArenaReport) (ReportRef, error) {
	if err := validateArenaReport(report); err != nil {
		return ReportRef{}, err
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return ReportRef{}, err
	}
	digest := sha256.Sum256(raw)
	hash := hex.EncodeToString(digest[:])
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return ReportRef{}, err
	}
	path := filepath.Join(directory, "evaluation-"+hash+".json")
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(raw) {
			return ReportRef{}, fmt.Errorf("evaluation hash collision at %s", path)
		}
		return ReportRef{Path: path, SHA256: hash, Bytes: len(raw)}, nil
	} else if !os.IsNotExist(err) {
		return ReportRef{}, err
	}
	temporary, err := os.CreateTemp(directory, ".evaluation-*.tmp")
	if err != nil {
		return ReportRef{}, err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if _, err := temporary.Write(raw); err != nil {
		return ReportRef{}, err
	}
	if err := temporary.Sync(); err != nil {
		return ReportRef{}, err
	}
	if err := temporary.Close(); err != nil {
		return ReportRef{}, err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return ReportRef{}, err
	}
	if err := os.Chmod(path, 0o444); err != nil {
		return ReportRef{}, err
	}
	return ReportRef{Path: path, SHA256: hash, Bytes: len(raw)}, nil
}

func validateArenaReport(report ArenaReport) error {
	if report.FormatVersion != EvaluationFormatVersion || report.EngineCommit == "" || report.CandidateID == "" || report.BaselineID == "" {
		return fmt.Errorf("refusing to persist incomplete arena report")
	}
	if report.Config.HoldoutSuiteID == "" || report.Config.CandidateSimulations <= 0 || report.Config.BaselineSimulations <= 0 {
		return fmt.Errorf("refusing to persist arena report without versioned search configuration")
	}
	if err := validateHoldoutSuiteIdentity(report.Config.HoldoutSuiteID, report.Cases); err != nil {
		return err
	}
	if len(report.Cases) == 0 || len(report.Games) != 2*len(report.Cases) {
		return fmt.Errorf("refusing to persist incomplete arena report")
	}
	if expected := summarizeArena(report); !reflect.DeepEqual(report.Summary, expected) {
		return fmt.Errorf("arena report summary does not match its games")
	}
	caseBySeed := make(map[int64]ArenaCase, len(report.Cases))
	for _, item := range report.Cases {
		if _, exists := caseBySeed[item.Seed]; exists {
			return fmt.Errorf("arena report contains duplicate case seed %d", item.Seed)
		}
		caseBySeed[item.Seed] = item
	}
	paired := make(map[int64][2]bool)
	for _, gameResult := range report.Games {
		if gameResult.CandidateSeat < 0 || gameResult.CandidateSeat > 1 || gameResult.AgentBySeat[gameResult.CandidateSeat] != arenaCandidateAgent || gameResult.AgentBySeat[1-gameResult.CandidateSeat] != arenaBaselineAgent {
			return fmt.Errorf("arena report contains a mislabeled game")
		}
		if gameResult.Completed && (gameResult.Failure != "" || gameResult.InvalidActions != 0 || gameResult.TripwireHits != 0) {
			return fmt.Errorf("completed arena game contains failure diagnostics")
		}
		if !gameResult.Completed && (gameResult.Failure == "" || gameResult.InvalidActions+gameResult.TripwireHits == 0) {
			return fmt.Errorf("incomplete arena game is missing durable failure diagnostics")
		}
		item, exists := caseBySeed[gameResult.Seed]
		if !exists || gameResult.Factions != [2]models.FactionType{item.First, item.Second} {
			return fmt.Errorf("arena report game does not match case seed %d", gameResult.Seed)
		}
		seats := paired[gameResult.Seed]
		if seats[gameResult.CandidateSeat] {
			return fmt.Errorf("arena report contains duplicate candidate seat for seed %d", gameResult.Seed)
		}
		seats[gameResult.CandidateSeat] = true
		paired[gameResult.Seed] = seats
	}
	for _, item := range report.Cases {
		if seats := paired[item.Seed]; !seats[0] || !seats[1] {
			return fmt.Errorf("arena report is missing paired seats for seed %d", item.Seed)
		}
	}
	return nil
}

func validateHoldoutSuiteIdentity(identity string, cases []ArenaCase) error {
	if !strings.HasPrefix(identity, HoldoutSuiteID) {
		return nil
	}
	expected, err := HoldoutSuiteIdentity(cases)
	if err != nil || identity != expected {
		return fmt.Errorf("holdout suite identity %q does not match its cases", identity)
	}
	return nil
}
