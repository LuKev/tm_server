package dataset

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/lukev/tm_server/internal/az/selfplay"
)

type ExportConfig struct {
	Input        string
	SamplesPath  string
	VocabPath    string
	ManifestPath string
}

type PolicyTarget struct {
	ActionIndex int     `json:"actionIndex"`
	Probability float64 `json:"probability"`
}

type Sample struct {
	Scenario           string         `json:"scenario"`
	Episode            int            `json:"episode"`
	Ply                int            `json:"ply"`
	PlayerID           string         `json:"playerId"`
	Encoding           []float64      `json:"encoding"`
	ObservationSchema  string         `json:"observationSchema,omitempty"`
	ObservationShape   []int          `json:"observationShape,omitempty"`
	LegalActionIndices []int          `json:"legalActionIndices"`
	PolicyTargets      []PolicyTarget `json:"policyTargets"`
	Value              float64        `json:"value"`
	Terminal           bool           `json:"terminal"`
	Truncated          bool           `json:"truncated"`
}

type Manifest struct {
	Version           int      `json:"version"`
	Input             string   `json:"input"`
	SamplesPath       string   `json:"samplesPath"`
	VocabPath         string   `json:"vocabPath"`
	SampleCount       int      `json:"sampleCount"`
	ActionCount       int      `json:"actionCount"`
	EncodingSize      int      `json:"encodingSize"`
	ObservationSchema string   `json:"observationSchema,omitempty"`
	ObservationShape  []int    `json:"observationShape,omitempty"`
	FeatureNames      []string `json:"featureNames,omitempty"`
	Scenarios         []string `json:"scenarios"`
}

func Export(config ExportConfig) (Manifest, error) {
	if config.Input == "" {
		return Manifest{}, fmt.Errorf("input is required")
	}
	if config.SamplesPath == "" {
		return Manifest{}, fmt.Errorf("samples path is required")
	}
	if config.VocabPath == "" {
		return Manifest{}, fmt.Errorf("vocab path is required")
	}
	if config.ManifestPath == "" {
		return Manifest{}, fmt.Errorf("manifest path is required")
	}
	records, err := readRecords(config.Input)
	if err != nil {
		return Manifest{}, err
	}
	vocab := buildVocab(records)
	indexByAction := make(map[string]int, len(vocab))
	for i, actionID := range vocab {
		indexByAction[actionID] = i
	}
	sampleFile, err := os.Create(config.SamplesPath)
	if err != nil {
		return Manifest{}, err
	}
	defer sampleFile.Close()
	encoder := json.NewEncoder(sampleFile)
	manifest := Manifest{
		Version:     1,
		Input:       config.Input,
		SamplesPath: config.SamplesPath,
		VocabPath:   config.VocabPath,
		SampleCount: len(records),
		ActionCount: len(vocab),
		Scenarios:   scenarios(records),
	}
	for _, record := range records {
		if len(record.Encoding) > manifest.EncodingSize {
			manifest.EncodingSize = len(record.Encoding)
		}
		if manifest.ObservationSchema == "" && record.ObservationSchema != "" {
			manifest.ObservationSchema = record.ObservationSchema
		}
		if len(manifest.ObservationShape) == 0 && len(record.ObservationShape) > 0 {
			manifest.ObservationShape = append([]int(nil), record.ObservationShape...)
		}
		if len(manifest.FeatureNames) == 0 && len(record.FeatureNames) > 0 {
			manifest.FeatureNames = append([]string(nil), record.FeatureNames...)
		}
		sample := Sample{
			Scenario:           record.Scenario,
			Episode:            record.Episode,
			Ply:                record.Ply,
			PlayerID:           record.PlayerID,
			Encoding:           record.Encoding,
			ObservationSchema:  record.ObservationSchema,
			ObservationShape:   record.ObservationShape,
			LegalActionIndices: actionIndices(record.LegalActions, indexByAction),
			PolicyTargets:      policyTargets(record.Policy, indexByAction),
			Value:              record.Outcome,
			Terminal:           record.Terminal,
			Truncated:          record.Truncated,
		}
		if err := encoder.Encode(sample); err != nil {
			return Manifest{}, err
		}
	}
	if err := writeJSON(config.VocabPath, vocab); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(config.ManifestPath, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func readRecords(path string) ([]selfplay.Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []selfplay.Record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 32*1024*1024)
	for scanner.Scan() {
		var record selfplay.Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, scanner.Err()
}

func buildVocab(records []selfplay.Record) []string {
	seen := make(map[string]bool)
	for _, record := range records {
		for _, actionID := range record.LegalActions {
			seen[actionID] = true
		}
		for actionID := range record.Policy {
			seen[actionID] = true
		}
		if record.ActionID != "" {
			seen[record.ActionID] = true
		}
	}
	out := make([]string, 0, len(seen))
	for actionID := range seen {
		out = append(out, actionID)
	}
	sort.Strings(out)
	return out
}

func actionIndices(actionIDs []string, indexByAction map[string]int) []int {
	out := make([]int, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		if index, ok := indexByAction[actionID]; ok {
			out = append(out, index)
		}
	}
	sort.Ints(out)
	return out
}

func policyTargets(policy map[string]float64, indexByAction map[string]int) []PolicyTarget {
	out := make([]PolicyTarget, 0, len(policy))
	for actionID, prob := range policy {
		if index, ok := indexByAction[actionID]; ok {
			out = append(out, PolicyTarget{ActionIndex: index, Probability: prob})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ActionIndex < out[j].ActionIndex
	})
	return out
}

func scenarios(records []selfplay.Record) []string {
	seen := make(map[string]bool)
	for _, record := range records {
		if record.Scenario != "" {
			seen[record.Scenario] = true
		}
	}
	out := make([]string, 0, len(seen))
	for scenario := range seen {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out
}

func writeJSON(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
