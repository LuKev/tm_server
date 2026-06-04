package selfplay

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

type Config struct {
	Episodes       int
	MaxPlies       int
	Scenario       string
	Workers        int
	ProgressWriter io.Writer
	Search         mcts.Config
	RandomSeed     int64
}

type Metrics struct {
	Episodes               int            `json:"episodes"`
	Records                int            `json:"records"`
	CompletedGames         int            `json:"completedGames"`
	TruncatedGames         int            `json:"truncatedGames"`
	TerminalGames          int            `json:"terminalGames"`
	ScenarioCounts         map[string]int `json:"scenarioCounts"`
	FinalRoundCounts       map[string]int `json:"finalRoundCounts,omitempty"`
	FinalPhaseCounts       map[string]int `json:"finalPhaseCounts,omitempty"`
	TerminalPhaseCounts    map[string]int `json:"terminalPhaseCounts,omitempty"`
	TruncatedPhaseCounts   map[string]int `json:"truncatedPhaseCounts,omitempty"`
	ActionTypeCounts       map[string]int `json:"actionTypeCounts,omitempty"`
	LastActionTypeCounts   map[string]int `json:"lastActionTypeCounts,omitempty"`
	ElapsedMillis          int64          `json:"elapsedMillis"`
	LegalMillis            int64          `json:"legalMillis"`
	SearchMillis           int64          `json:"searchMillis"`
	ApplyMillis            int64          `json:"applyMillis"`
	LegalNanos             int64          `json:"legalNanos,omitempty"`
	SearchNanos            int64          `json:"searchNanos,omitempty"`
	ApplyNanos             int64          `json:"applyNanos,omitempty"`
	AveragePliesPerEpisode float64        `json:"averagePliesPerEpisode"`
	AverageBranchingFactor float64        `json:"averageBranchingFactor"`
	RecordsPerSecond       float64        `json:"recordsPerSecond"`
	MaxFinalRound          int            `json:"maxFinalRound,omitempty"`
	Workers                int            `json:"workers,omitempty"`
}

type Record struct {
	Episode           int                `json:"episode"`
	Ply               int                `json:"ply"`
	Scenario          string             `json:"scenario"`
	PlayerID          string             `json:"playerId"`
	RootPlayerID      string             `json:"rootPlayerId"`
	Round             int                `json:"round,omitempty"`
	Phase             string             `json:"phase,omitempty"`
	State             string             `json:"state"`
	Encoding          []float64          `json:"encoding"`
	ObservationSchema string             `json:"observationSchema,omitempty"`
	ObservationShape  []int              `json:"observationShape,omitempty"`
	FeatureNames      []string           `json:"featureNames,omitempty"`
	LegalActions      []string           `json:"legalActions"`
	Policy            map[string]float64 `json:"policy"`
	ActionID          string             `json:"actionId"`
	Outcome           float64            `json:"outcome"`
	Terminal          bool               `json:"terminal"`
	Truncated         bool               `json:"truncated"`
}

func Run(writer io.Writer, evaluator model.Evaluator, config Config) error {
	_, err := RunWithMetrics(writer, evaluator, config)
	return err
}

func RunWithMetrics(writer io.Writer, evaluator model.Evaluator, config Config) (Metrics, error) {
	started := time.Now()
	metrics := newMetrics()
	if writer == nil {
		return metrics, fmt.Errorf("nil writer")
	}
	if evaluator == nil {
		evaluator = model.NewHeuristicEvaluator()
	}
	if config.Episodes <= 0 {
		config.Episodes = 1
	}
	if config.MaxPlies <= 0 {
		config.MaxPlies = 200
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
	if config.Workers <= 0 {
		config.Workers = 1
	}
	if config.Workers > config.Episodes {
		config.Workers = config.Episodes
	}
	metrics.Workers = config.Workers
	encoder := json.NewEncoder(writer)

	jobs := make(chan int)
	results := make(chan episodeResult)
	var wg sync.WaitGroup
	for workerID := 0; workerID < config.Workers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for episode := range jobs {
				results <- playEpisode(episode, workerID, evaluator, config)
			}
		}(workerID)
	}
	go func() {
		for episode := 0; episode < config.Episodes; episode++ {
			jobs <- episode
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var firstErr error
	for result := range results {
		if result.err != nil {
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		mergeMetrics(&metrics, result.metrics)
		for i := range result.records {
			if err := encoder.Encode(result.records[i]); err != nil {
				return metrics, err
			}
		}
		if config.ProgressWriter != nil {
			writeProgress(config.ProgressWriter, result, metrics, time.Since(started), config.Episodes)
		}
	}
	metrics.ElapsedMillis = time.Since(started).Milliseconds()
	finalizeMetrics(&metrics)
	if firstErr != nil {
		return metrics, firstErr
	}
	return metrics, nil
}

type episodeResult struct {
	episode    int
	workerID   int
	records    []Record
	metrics    Metrics
	finalRound int
	finalPhase string
	terminal   bool
	truncated  bool
	elapsed    time.Duration
	err        error
}

func playEpisode(episode, workerID int, evaluator model.Evaluator, config Config) episodeResult {
	started := time.Now()
	metrics := newMetrics()
	rng := rand.New(rand.NewSource(episodeSeed(config.RandomSeed, episode)))
	position, scenarioName, err := env.SampleScenario(config.Scenario, rng)
	if err != nil {
		return episodeResult{episode: episode, workerID: workerID, metrics: metrics, err: err}
	}
	metrics.Episodes = 1
	metrics.ScenarioCounts[scenarioName]++
	var records []Record
	lastActionType := ""
	for ply := 0; ply < config.MaxPlies && !position.IsTerminal(); ply++ {
		legalStarted := time.Now()
		legal := position.LegalActions()
		metrics.LegalNanos += time.Since(legalStarted).Nanoseconds()
		if len(legal) == 0 {
			break
		}
		metrics.AverageBranchingFactor += float64(len(legal))
		searchConfig := config.Search
		if searchConfig.RandomSeed == 0 {
			searchConfig.RandomSeed = rng.Int63()
		}
		searchStarted := time.Now()
		result := mcts.Search(position, evaluator, searchConfig)
		metrics.SearchNanos += time.Since(searchStarted).Nanoseconds()
		selected := selectAction(result, rng)
		if selected.ID == "" {
			break
		}
		action, ok := actionByID(legal, selected.ID)
		if !ok {
			return episodeResult{episode: episode, workerID: workerID, metrics: metrics, err: fmt.Errorf("selected illegal action %s", selected.ID)}
		}
		observation := position.Observation()
		record := Record{
			Episode:           episode,
			Ply:               ply,
			Scenario:          scenarioName,
			PlayerID:          action.PlayerID,
			RootPlayerID:      position.RootPlayerID,
			Round:             position.State.Round,
			Phase:             phaseName(position.State.Phase),
			State:             position.SnapshotJSON(),
			Encoding:          observation.Features,
			ObservationSchema: observation.Schema,
			ObservationShape:  observation.Shape,
			LegalActions:      actionIDs(legal),
			Policy:            policyMap(result.Actions),
			ActionID:          action.ID,
		}
		metrics.ActionTypeCounts[action.Type]++
		lastActionType = action.Type
		if ply == 0 {
			record.FeatureNames = observation.FeatureNames
		}
		records = append(records, record)
		applyStarted := time.Now()
		position, err = position.Apply(action)
		metrics.ApplyNanos += time.Since(applyStarted).Nanoseconds()
		if err != nil {
			return episodeResult{episode: episode, workerID: workerID, metrics: metrics, err: err}
		}
	}
	truncated := !position.IsTerminal() && len(records) >= config.MaxPlies
	finalRound := 0
	finalPhase := "unknown"
	if position != nil && position.State != nil {
		finalRound = position.State.Round
		finalPhase = phaseName(position.State.Phase)
	}
	terminal := position.IsTerminal()
	metrics.FinalRoundCounts[fmt.Sprint(finalRound)]++
	metrics.FinalPhaseCounts[finalPhase]++
	metrics.MaxFinalRound = finalRound
	metrics.Records = len(records)
	metrics.CompletedGames = 1
	if truncated {
		metrics.TruncatedGames = 1
		metrics.TruncatedPhaseCounts[finalPhase]++
	}
	if terminal {
		metrics.TerminalGames = 1
		metrics.TerminalPhaseCounts[finalPhase]++
	}
	if lastActionType != "" {
		metrics.LastActionTypeCounts[lastActionType]++
	}
	for i := range records {
		records[i].Outcome = position.ValueFor(records[i].PlayerID)
		records[i].Terminal = terminal
		records[i].Truncated = truncated
	}
	return episodeResult{
		episode:    episode,
		workerID:   workerID,
		records:    records,
		metrics:    metrics,
		finalRound: finalRound,
		finalPhase: finalPhase,
		terminal:   terminal,
		truncated:  truncated,
		elapsed:    time.Since(started),
	}
}

func newMetrics() Metrics {
	return Metrics{
		ScenarioCounts:       make(map[string]int),
		FinalRoundCounts:     make(map[string]int),
		FinalPhaseCounts:     make(map[string]int),
		TerminalPhaseCounts:  make(map[string]int),
		TruncatedPhaseCounts: make(map[string]int),
		ActionTypeCounts:     make(map[string]int),
		LastActionTypeCounts: make(map[string]int),
	}
}

func mergeMetrics(total *Metrics, part Metrics) {
	total.Episodes += part.Episodes
	total.Records += part.Records
	total.CompletedGames += part.CompletedGames
	total.TruncatedGames += part.TruncatedGames
	total.TerminalGames += part.TerminalGames
	total.LegalNanos += part.LegalNanos
	total.SearchNanos += part.SearchNanos
	total.ApplyNanos += part.ApplyNanos
	total.AverageBranchingFactor += part.AverageBranchingFactor
	if part.MaxFinalRound > total.MaxFinalRound {
		total.MaxFinalRound = part.MaxFinalRound
	}
	mergeStringInt(total.ScenarioCounts, part.ScenarioCounts)
	mergeStringInt(total.FinalRoundCounts, part.FinalRoundCounts)
	mergeStringInt(total.FinalPhaseCounts, part.FinalPhaseCounts)
	mergeStringInt(total.TerminalPhaseCounts, part.TerminalPhaseCounts)
	mergeStringInt(total.TruncatedPhaseCounts, part.TruncatedPhaseCounts)
	mergeStringInt(total.ActionTypeCounts, part.ActionTypeCounts)
	mergeStringInt(total.LastActionTypeCounts, part.LastActionTypeCounts)
}

func mergeStringInt(dst, src map[string]int) {
	for key, value := range src {
		dst[key] += value
	}
}

func finalizeMetrics(metrics *Metrics) {
	metrics.LegalMillis = metrics.LegalNanos / int64(time.Millisecond)
	metrics.SearchMillis = metrics.SearchNanos / int64(time.Millisecond)
	metrics.ApplyMillis = metrics.ApplyNanos / int64(time.Millisecond)
	if metrics.Episodes > 0 {
		metrics.AveragePliesPerEpisode = float64(metrics.Records) / float64(metrics.Episodes)
	}
	if metrics.Records > 0 {
		metrics.AverageBranchingFactor /= float64(metrics.Records)
	}
	elapsedSeconds := float64(metrics.ElapsedMillis) / 1000.0
	if elapsedSeconds > 0 {
		metrics.RecordsPerSecond = float64(metrics.Records) / elapsedSeconds
	}
}

func writeProgress(writer io.Writer, result episodeResult, metrics Metrics, elapsed time.Duration, totalEpisodes int) {
	progress := map[string]interface{}{
		"event":             "selfplay_game",
		"episode":           result.episode,
		"worker":            result.workerID,
		"records":           len(result.records),
		"finalRound":        result.finalRound,
		"finalPhase":        result.finalPhase,
		"terminal":          result.terminal,
		"truncated":         result.truncated,
		"gameElapsedMillis": result.elapsed.Milliseconds(),
		"completedGames":    metrics.CompletedGames,
		"totalGames":        totalEpisodes,
		"totalRecords":      metrics.Records,
		"elapsedMillis":     elapsed.Milliseconds(),
	}
	if elapsed > 0 {
		progress["recordsPerSecond"] = float64(metrics.Records) / elapsed.Seconds()
	}
	raw, err := json.Marshal(progress)
	if err == nil {
		_, _ = fmt.Fprintln(writer, string(raw))
	}
}

func episodeSeed(base int64, episode int) int64 {
	return base + int64(episode+1)*1000003
}

func phaseName(phase interface{}) string {
	switch fmt.Sprint(phase) {
	case "0":
		return "setup"
	case "1":
		return "faction_selection"
	case "2":
		return "income"
	case "3":
		return "action"
	case "4":
		return "cleanup"
	case "5":
		return "end"
	default:
		return fmt.Sprint(phase)
	}
}

func selectAction(result mcts.Result, rng *rand.Rand) mcts.RankedAction {
	if len(result.Actions) == 0 {
		return mcts.RankedAction{}
	}
	total := 0.0
	for _, action := range result.Actions {
		total += action.Prob
	}
	if total <= 0 {
		return result.Actions[0]
	}
	target := rng.Float64() * total
	accum := 0.0
	for _, action := range result.Actions {
		accum += action.Prob
		if accum >= target {
			return action
		}
	}
	return result.Actions[0]
}

func actionByID(options []actions.Option, id string) (actions.Option, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return actions.Option{}, false
}

func actionIDs(options []actions.Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.ID)
	}
	return out
}

func policyMap(ranked []mcts.RankedAction) map[string]float64 {
	out := make(map[string]float64, len(ranked))
	for _, action := range ranked {
		out[action.ID] = action.Prob
	}
	return out
}
