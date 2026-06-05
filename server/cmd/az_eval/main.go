package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/lukev/tm_server/internal/az/arena"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

type report struct {
	StartedAt  string                  `json:"startedAt"`
	FinishedAt string                  `json:"finishedAt"`
	Candidate  evaluatorRef            `json:"candidate"`
	Baseline   evaluatorRef            `json:"baseline"`
	Scenario   string                  `json:"scenario"`
	MaxPlies   int                     `json:"maxPlies"`
	Search     mcts.Config             `json:"search"`
	RandomSeed int64                   `json:"randomSeed"`
	Result     arena.Result            `json:"result"`
	Promotion  arena.PromotionDecision `json:"promotion"`
	Runtime    runtimeInfo             `json:"runtime"`
}

type evaluatorRef struct {
	ModelPath string `json:"modelPath,omitempty"`
	ModelURL  string `json:"modelUrl,omitempty"`
	Kind      string `json:"kind"`
}

type runtimeInfo struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	NumCPU    int    `json:"numCpu"`
	GoVersion string `json:"goVersion"`
}

func main() {
	candidateModel := flag.String("candidate_model", "", "candidate table model JSON path")
	candidateURL := flag.String("candidate_url", "", "candidate HTTP evaluator URL")
	baselineModel := flag.String("baseline_model", "", "baseline table model JSON path")
	baselineURL := flag.String("baseline_url", "", "baseline HTTP evaluator URL")
	output := flag.String("output", "", "optional JSON report output path; stdout when empty")
	scenario := flag.String("scenario", "training_mix", "scenario name, snapshot source, or comma-separated scenario set")
	games := flag.Int("games", 20, "arena games")
	maxPlies := flag.Int("max_plies", 500, "maximum plies per game")
	workers := flag.Int("workers", 1, "parallel arena game workers")
	sims := flag.Int("sims", 32, "MCTS simulations per move")
	batchSize := flag.Int("batch_size", 1, "MCTS neural evaluation batch size when evaluator supports it")
	globalBatchSize := flag.Int("global_batch_size", 0, "merge concurrent evaluator batches up to this size; 0 disables")
	globalBatchDelay := flag.Int("global_batch_delay_ms", 1, "maximum delay before flushing a partial global evaluator batch")
	maxDepth := flag.Int("max_depth", 120, "MCTS simulation max depth")
	promoteWinRate := flag.Float64("promote_win_rate", 0.55, "minimum arena score for the promotion decision report")
	promoteMinGames := flag.Int("promote_min_games", 0, "minimum arena games for the promotion decision report; 0 disables")
	promoteCI95LowerBound := flag.Float64("promote_ci95_lower_bound", 0, "minimum 95% confidence interval lower bound for the promotion decision report; 0 disables")
	seed := flag.Int64("seed", 1, "random seed")
	progress := flag.Bool("progress", false, "write per-game progress JSON lines to stderr")
	flag.Parse()

	if (*candidateModel == "" && *candidateURL == "") || (*candidateModel != "" && *candidateURL != "") {
		exitf("provide exactly one of -candidate_model or -candidate_url")
	}
	if *games <= 0 {
		exitf("-games must be positive")
	}
	startedAt := time.Now()
	candidate := loadEvaluator(*candidateModel, *candidateURL)
	baseline := loadEvaluator(*baselineModel, *baselineURL)
	candidate = model.NewAsyncBatchEvaluator(candidate, *globalBatchSize, time.Duration(*globalBatchDelay)*time.Millisecond)
	baseline = model.NewAsyncBatchEvaluator(baseline, *globalBatchSize, time.Duration(*globalBatchDelay)*time.Millisecond)
	search := mcts.Config{
		Simulations: *sims,
		BatchSize:   *batchSize,
		CPUCT:       1.5,
		Temperature: 0,
		MaxDepth:    *maxDepth,
	}
	result, err := arena.Evaluate(candidate, baseline, arena.Config{
		Games:          *games,
		MaxPlies:       *maxPlies,
		Scenario:       *scenario,
		Workers:        *workers,
		ProgressWriter: progressWriter(*progress),
		RandomSeed:     *seed,
		Search:         search,
	})
	if err != nil {
		exitf("arena: %v", err)
	}
	out := report{
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
		Candidate:  evaluatorRef{ModelPath: *candidateModel, ModelURL: *candidateURL, Kind: evaluatorKind(*candidateModel, *candidateURL, false)},
		Baseline:   evaluatorRef{ModelPath: *baselineModel, ModelURL: *baselineURL, Kind: evaluatorKind(*baselineModel, *baselineURL, true)},
		Scenario:   *scenario,
		MaxPlies:   *maxPlies,
		Search:     search,
		RandomSeed: *seed,
		Result:     result,
		Promotion: arena.DecidePromotion(result, arena.PromotionPolicy{
			MinWinRate:        *promoteWinRate,
			MinGames:          *promoteMinGames,
			MinCI95LowerBound: *promoteCI95LowerBound,
		}),
		Runtime: runtimeInfo{
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
			NumCPU:    runtime.NumCPU(),
			GoVersion: runtime.Version(),
		},
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		exitf("encode report: %v", err)
	}
	if *output != "" {
		if err := os.WriteFile(*output, raw, 0644); err != nil {
			exitf("write output: %v", err)
		}
		return
	}
	_, _ = os.Stdout.Write(raw)
	_, _ = fmt.Fprintln(os.Stdout)
}

func progressWriter(enabled bool) *os.File {
	if !enabled {
		return nil
	}
	return os.Stderr
}

func loadEvaluator(path, url string) model.Evaluator {
	evaluator, err := model.LoadEvaluator(model.EvaluatorConfig{
		TableModelPath: path,
		HTTPURL:        url,
	})
	if err == nil {
		return evaluator
	}
	return model.NewHeuristicEvaluator()
}

func evaluatorKind(path, url string, fallback bool) string {
	switch {
	case url != "":
		return "http"
	case path != "":
		return "table"
	case fallback:
		return "heuristic"
	default:
		return "unknown"
	}
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
