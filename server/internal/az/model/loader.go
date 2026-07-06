package model

type EvaluatorConfig struct {
	HTTPURL string
}

func LoadEvaluator(config EvaluatorConfig) Evaluator {
	evaluator := Evaluator(NewHeuristicEvaluator())
	if config.HTTPURL != "" {
		evaluator = NewHTTPEvaluator(config.HTTPURL, evaluator)
	}
	return evaluator
}
