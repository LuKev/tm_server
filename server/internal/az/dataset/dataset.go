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
	Input         string
	SamplesPath   string
	VocabPath     string
	ManifestPath  string
	SeedVocabPath string
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
	vocab, manifest, err := scanManifest(config.Input)
	if err != nil {
		return Manifest{}, err
	}
	if config.SeedVocabPath != "" {
		vocab, err = mergeSeedVocab(config.SeedVocabPath, vocab)
		if err != nil {
			return Manifest{}, err
		}
		manifest.ActionCount = len(vocab)
	}
	manifest.Input = config.Input
	manifest.SamplesPath = config.SamplesPath
	manifest.VocabPath = config.VocabPath
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
	if err := scanRecords(config.Input, func(record selfplay.Record) error {
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
			return err
		}
		return nil
	}); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(config.VocabPath, vocab); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(config.ManifestPath, manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func mergeSeedVocab(path string, observed []string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seed vocab: %w", err)
	}
	var seed []string
	if err := json.Unmarshal(raw, &seed); err != nil {
		return nil, fmt.Errorf("parse seed vocab: %w", err)
	}
	seen := make(map[string]bool, len(seed)+len(observed))
	for _, actionID := range seed {
		if actionID == "" || seen[actionID] {
			return nil, fmt.Errorf("seed vocab contains empty or duplicate action ID %q", actionID)
		}
		seen[actionID] = true
	}
	for _, actionID := range observed {
		seen[actionID] = true
	}
	return sortedKeys(seen), nil
}

func scanManifest(path string) ([]string, Manifest, error) {
	seenActions := make(map[string]bool)
	seenScenarios := make(map[string]bool)
	manifest := Manifest{Version: 1}
	if err := scanManifestRecords(path, func(record manifestRecord) error {
		manifest.SampleCount++
		if manifest.ObservationSchema == "" && record.ObservationSchema != "" {
			manifest.ObservationSchema = record.ObservationSchema
		}
		if len(manifest.ObservationShape) == 0 && len(record.ObservationShape) > 0 {
			manifest.ObservationShape = append([]int(nil), record.ObservationShape...)
		}
		if manifest.EncodingSize == 0 && len(record.ObservationShape) == 3 {
			manifest.EncodingSize = record.ObservationShape[0] + record.ObservationShape[1]*record.ObservationShape[2]
		}
		if len(manifest.FeatureNames) == 0 && len(record.FeatureNames) > 0 {
			manifest.FeatureNames = append([]string(nil), record.FeatureNames...)
		}
		if record.Scenario != "" {
			seenScenarios[record.Scenario] = true
		}
		for _, actionID := range record.LegalActions {
			seenActions[actionID] = true
		}
		for actionID := range record.Policy {
			seenActions[actionID] = true
		}
		if record.ActionID != "" {
			seenActions[record.ActionID] = true
		}
		return nil
	}); err != nil {
		return nil, Manifest{}, err
	}
	if manifest.SampleCount == 0 {
		return nil, Manifest{}, fmt.Errorf("no records in %s", path)
	}
	vocab := sortedKeys(seenActions)
	manifest.ActionCount = len(vocab)
	manifest.Scenarios = sortedKeys(seenScenarios)
	return vocab, manifest, nil
}

type manifestRecord struct {
	Scenario          string             `json:"scenario"`
	ObservationSchema string             `json:"observationSchema,omitempty"`
	ObservationShape  []int              `json:"observationShape,omitempty"`
	FeatureNames      []string           `json:"featureNames,omitempty"`
	LegalActions      []string           `json:"legalActions"`
	Policy            map[string]float64 `json:"policy"`
	ActionID          string             `json:"actionId"`
}

func scanManifestRecords(path string, visit func(manifestRecord) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 32*1024*1024)
	for scanner.Scan() {
		var record manifestRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func scanRecords(path string, visit func(selfplay.Record) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 32*1024*1024)
	for scanner.Scan() {
		var record selfplay.Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func sortedKeys(seen map[string]bool) []string {
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
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

func writeJSON(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
