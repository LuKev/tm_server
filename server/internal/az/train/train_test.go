package train

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/lukev/tm_server/internal/az/selfplay"
)

func TestTrainFileRejectsEmptyInput(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "empty-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := TrainFile(file.Name()); err == nil {
		t.Fatal("expected empty input error")
	}
}

func writeRecords(t *testing.T, records []selfplay.Record) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "records-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

func TestTrainFileBuildsTableModel(t *testing.T) {
	path := writeRecords(t, []selfplay.Record{
		{
			Encoding: []float64{0.1, 0.2},
			Policy:   map[string]float64{"a": 1},
			Outcome:  0.5,
		},
		{
			Encoding: []float64{0.1, 0.2},
			Policy:   map[string]float64{"a": 0.25, "b": 0.75},
			Outcome:  -0.5,
		},
	})
	table, err := TrainFile(path)
	if err != nil {
		t.Fatalf("TrainFile failed: %v", err)
	}
	if len(table.Buckets) != 1 {
		t.Fatalf("bucket count = %d, want 1", len(table.Buckets))
	}
	if table.GlobalPolicy["a"] <= 0 || table.GlobalPolicy["b"] <= 0 {
		t.Fatalf("missing global policy weights: %#v", table.GlobalPolicy)
	}
}
