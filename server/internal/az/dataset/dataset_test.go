package dataset

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/lukev/tm_server/internal/az/selfplay"
)

func TestExportWritesManifestAndVocab(t *testing.T) {
	dir := t.TempDir()
	input := writeRecords(t, dir, []selfplay.Record{
		{
			Scenario:          "s1",
			Episode:           1,
			Ply:               2,
			PlayerID:          "p1",
			Encoding:          []float64{0.1, 0.2},
			ObservationSchema: "test_schema",
			ObservationShape:  []int{2, 0, 0},
			FeatureNames:      []string{"a", "b"},
			LegalActions:      []string{"a", "b"},
			Policy:            map[string]float64{"a": 1},
			ActionID:          "a",
			Outcome:           0.5,
		},
	})
	manifest, err := Export(ExportConfig{
		Input:        input,
		SamplesPath:  dir + "/samples.jsonl",
		VocabPath:    dir + "/vocab.json",
		ManifestPath: dir + "/manifest.json",
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if manifest.SampleCount != 1 || manifest.ActionCount != 2 || manifest.EncodingSize != 2 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if manifest.ObservationSchema != "test_schema" || len(manifest.ObservationShape) != 3 || len(manifest.FeatureNames) != 2 {
		t.Fatalf("missing observation metadata: %#v", manifest)
	}
	if _, err := os.Stat(dir + "/samples.jsonl"); err != nil {
		t.Fatal(err)
	}
}

func writeRecords(t *testing.T, dir string, records []selfplay.Record) string {
	t.Helper()
	path := dir + "/records.jsonl"
	file, err := os.Create(path)
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
	return path
}
