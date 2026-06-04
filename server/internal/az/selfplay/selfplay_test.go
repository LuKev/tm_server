package selfplay

import (
	"bytes"
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
