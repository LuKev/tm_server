package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/az/dataset"
)

func main() {
	input := flag.String("input", "", "self-play JSONL input")
	samples := flag.String("samples", "az_samples.jsonl", "output neural-ready samples JSONL")
	vocab := flag.String("vocab", "az_action_vocab.json", "output action vocabulary JSON")
	manifest := flag.String("manifest", "az_dataset_manifest.json", "output dataset manifest JSON")
	seedVocab := flag.String("seed_vocab", "", "optional existing action vocabulary to preserve and extend")
	flag.Parse()
	result, err := dataset.Export(dataset.ExportConfig{
		Input:         *input,
		SamplesPath:   *samples,
		VocabPath:     *vocab,
		ManifestPath:  *manifest,
		SeedVocabPath: *seedVocab,
	})
	if err != nil {
		exitf("export: %v", err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "exported %d samples, %d actions\n", result.SampleCount, result.ActionCount)
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
