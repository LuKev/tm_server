package model

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

type TableModel struct {
	Buckets      map[string]TableBucket `json:"buckets"`
	GlobalPolicy map[string]float64     `json:"globalPolicy"`
}

type TableBucket struct {
	Policy map[string]float64 `json:"policy"`
	Value  float64            `json:"value"`
	Count  int                `json:"count"`
}

func LoadTableModel(path string) (*TableModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table TableModel
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, err
	}
	return &table, nil
}

func (m *TableModel) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	if m == nil {
		return NewHeuristicEvaluator().Evaluate(position, legal, perspectivePlayerID)
	}
	key := EncodingKey(position.Encode())
	bucket, ok := m.Buckets[key]
	priors := make(map[string]float64, len(legal))
	total := 0.0
	for _, option := range legal {
		weight := 0.001
		if ok {
			weight += bucket.Policy[option.ID]
		}
		weight += 0.01 * m.GlobalPolicy[option.ID]
		priors[option.ID] = weight
		total += weight
	}
	if total > 0 {
		for id, weight := range priors {
			priors[id] = weight / total
		}
	}
	value := 0.0
	if ok {
		value = bucket.Value
	} else if position != nil {
		value = position.ValueFor(perspectivePlayerID)
	}
	return Evaluation{Priors: priors, Value: math.Max(-1, math.Min(1, value))}
}

func EncodingKey(encoding []float64) string {
	parts := make([]string, 0, len(encoding))
	for _, value := range encoding {
		parts = append(parts, strconv.FormatFloat(math.Round(value*100)/100, 'f', 2, 64))
	}
	return strings.Join(parts, ",")
}

func SaveTableModel(path string, table *TableModel) error {
	if table == nil {
		return fmt.Errorf("nil table model")
	}
	raw, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
