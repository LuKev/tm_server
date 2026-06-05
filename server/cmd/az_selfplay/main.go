package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/az/selfplay"
)

func main() {
	var config selfplay.Config
	flag.IntVar(&config.Episodes, "episodes", 1, "number of self-play episodes")
	flag.IntVar(&config.MaxPlies, "max_plies", 500, "maximum plies per episode")
	flag.StringVar(&config.Scenario, "scenario", "base_nomads_witches", "built-in scenario name")
	flag.IntVar(&config.Workers, "workers", 1, "parallel self-play game workers")
	flag.BoolVar(&config.CompactRecords, "compact_records", false, "omit debug state snapshots from self-play JSONL records")
	flag.BoolVar(&config.ReuseTree, "reuse_tree", false, "reuse the selected MCTS subtree between moves in each self-play game")
	flag.IntVar(&config.Search.Simulations, "sims", 64, "MCTS simulations per move")
	flag.IntVar(&config.Search.BatchSize, "batch_size", 1, "MCTS neural evaluation batch size when evaluator supports it")
	flag.Float64Var(&config.Search.CPUCT, "cpuct", 1.5, "PUCT exploration constant")
	flag.Float64Var(&config.Search.Temperature, "temperature", 1.0, "root visit-count temperature")
	flag.Float64Var(&config.Search.DirichletAlpha, "dirichlet_alpha", 0.3, "root Dirichlet noise alpha")
	flag.Float64Var(&config.Search.DirichletWeight, "dirichlet_weight", 0.25, "root Dirichlet noise weight")
	flag.IntVar(&config.Search.MaxDepth, "max_depth", 200, "MCTS simulation max depth")
	flag.Int64Var(&config.RandomSeed, "seed", 0, "random seed")
	output := flag.String("output", "", "output JSONL path; stdout when empty")
	metricsPath := flag.String("metrics", "", "optional metrics JSON output path")
	modelPath := flag.String("model", "", "optional table model JSON used as evaluator")
	modelURL := flag.String("model_url", "", "optional HTTP policy/value evaluator URL")
	globalBatchSize := flag.Int("global_batch_size", 0, "merge concurrent evaluator batches up to this size; 0 disables")
	globalBatchDelay := flag.Int("global_batch_delay_ms", 1, "maximum delay before flushing a partial global evaluator batch")
	progress := flag.Bool("progress", false, "write per-game progress JSON lines to stderr")
	listScenarios := flag.Bool("list_scenarios", false, "print available built-in scenario names")
	flag.Parse()
	if *listScenarios {
		_, _ = fmt.Fprintln(os.Stdout, strings.Join(env.ScenarioNames(), "\n"))
		return
	}

	var writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			exitf("create output: %v", err)
		}
		defer file.Close()
		writer = file
	}
	evaluator, err := model.LoadEvaluator(model.EvaluatorConfig{
		TableModelPath: *modelPath,
		HTTPURL:        *modelURL,
	})
	if err != nil {
		exitf("load evaluator: %v", err)
	}
	evaluator = model.NewAsyncBatchEvaluator(evaluator, *globalBatchSize, time.Duration(*globalBatchDelay)*time.Millisecond)
	if *progress {
		config.ProgressWriter = os.Stderr
	}
	metrics, err := selfplay.RunWithMetrics(writer, evaluator, config)
	if err != nil {
		exitf("self-play failed: %v", err)
	}
	if *metricsPath != "" {
		raw, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			exitf("encode metrics: %v", err)
		}
		if err := os.WriteFile(*metricsPath, raw, 0644); err != nil {
			exitf("write metrics: %v", err)
		}
	}
	raw, _ := json.Marshal(metrics)
	_, _ = fmt.Fprintln(os.Stderr, string(raw))
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

var _ = mcts.Config{}
