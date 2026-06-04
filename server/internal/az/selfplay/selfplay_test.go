package selfplay

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

func TestRunWritesJSONLRecords(t *testing.T) {
	var buf bytes.Buffer
	metrics, err := RunWithMetrics(&buf, model.NewHeuristicEvaluator(), Config{
		Episodes:   1,
		MaxPlies:   1,
		Scenario:   "base_nomads_witches",
		RandomSeed: 1,
		Search: mcts.Config{
			Simulations: 1,
			CPUCT:       1.5,
			Temperature: 1,
			RandomSeed:  2,
			MaxDepth:    1,
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got == "" {
		t.Fatal("expected JSONL output")
	}
	if metrics.Episodes != 1 || metrics.Records != 1 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	if metrics.AverageBranchingFactor <= 0 {
		t.Fatalf("expected branching metrics: %#v", metrics)
	}
}

func TestRunCompactRecordsOmitStateSnapshot(t *testing.T) {
	var buf bytes.Buffer
	_, err := RunWithMetrics(&buf, model.NewHeuristicEvaluator(), Config{
		Episodes:       1,
		MaxPlies:       1,
		Scenario:       "base_nomads_witches",
		CompactRecords: true,
		ReuseTree:      true,
		RandomSeed:     1,
		Search: mcts.Config{
			Simulations: 1,
			CPUCT:       1.5,
			Temperature: 1,
			RandomSeed:  2,
			MaxDepth:    1,
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &raw); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if _, ok := raw["state"]; ok {
		t.Fatal("compact record should omit state snapshot")
	}
	if _, ok := raw["encoding"]; !ok {
		t.Fatal("compact record should keep encoding")
	}
}

func TestRunWithWorkersWritesRecords(t *testing.T) {
	var buf bytes.Buffer
	metrics, err := RunWithMetrics(&buf, model.NewHeuristicEvaluator(), Config{
		Episodes:   4,
		MaxPlies:   2,
		Scenario:   "training_mix",
		Workers:    2,
		RandomSeed: 3,
		Search: mcts.Config{
			Simulations: 0,
			CPUCT:       1.5,
			Temperature: 1,
			MaxDepth:    1,
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if metrics.Episodes != 4 || metrics.CompletedGames != 4 {
		t.Fatalf("unexpected worker metrics: %#v", metrics)
	}
	if metrics.Workers != 2 {
		t.Fatalf("workers = %d, want 2", metrics.Workers)
	}
	if got := strings.TrimSpace(buf.String()); got == "" {
		t.Fatal("expected JSONL output")
	}
	if metrics.LegalNanos <= 0 || metrics.SearchNanos <= 0 || metrics.ApplyNanos <= 0 {
		t.Fatalf("expected nanosecond timing metrics: %#v", metrics)
	}
}
