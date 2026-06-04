package model

import "fmt"

type EvaluatorConfig struct {
	TableModelPath string
	HTTPURL        string
}

func LoadEvaluator(config EvaluatorConfig) (Evaluator, error) {
	evaluator := Evaluator(NewHeuristicEvaluator())
	if config.TableModelPath != "" {
		table, err := LoadTableModel(config.TableModelPath)
		if err != nil {
			return nil, fmt.Errorf("load table model: %w", err)
		}
		evaluator = table
	}
	if config.HTTPURL != "" {
		evaluator = NewHTTPEvaluator(config.HTTPURL, evaluator)
	}
	return evaluator, nil
}
