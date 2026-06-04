package buffer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCopiesAndSamplesSources(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")
	output := filepath.Join(dir, "out", "buffer.jsonl")
	summaryPath := filepath.Join(dir, "out", "summary.json")

	if err := os.WriteFile(first, []byte("a\nb\n\nc\n"), 0644); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := os.WriteFile(second, []byte("d\ne\nf\ng\n"), 0644); err != nil {
		t.Fatalf("write second: %v", err)
	}

	summary, err := Build(Config{
		Sources: []Source{
			{Path: first},
			{Path: second, Limit: 2},
		},
		OutputPath:  output,
		SummaryPath: summaryPath,
		Seed:        7,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 output rows, got %d: %q", len(lines), string(raw))
	}
	if strings.Join(lines[:3], ",") != "a,b,c" {
		t.Fatalf("expected uncapped source to preserve all rows, got %v", lines[:3])
	}
	if summary.TotalInputRecords != 7 || summary.TotalOutputRecords != 5 {
		t.Fatalf("unexpected summary totals: %+v", summary)
	}
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("summary was not written: %v", err)
	}
}
