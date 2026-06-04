package selfplay

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

type Config struct {
	Episodes     int
	MaxPlies     int
	Scenario     string
	MinPassRound int
	Search       mcts.Config
	RandomSeed   int64
}

type Metrics struct {
	Episodes               int            `json:"episodes"`
	Records                int            `json:"records"`
	CompletedGames         int            `json:"completedGames"`
	TruncatedGames         int            `json:"truncatedGames"`
	TerminalGames          int            `json:"terminalGames"`
	ScenarioCounts         map[string]int `json:"scenarioCounts"`
	FinalRoundCounts       map[string]int `json:"finalRoundCounts,omitempty"`
	FinalPhaseCounts       map[string]int `json:"finalPhaseCounts,omitempty"`
	TerminalPhaseCounts    map[string]int `json:"terminalPhaseCounts,omitempty"`
	TruncatedPhaseCounts   map[string]int `json:"truncatedPhaseCounts,omitempty"`
	ActionTypeCounts       map[string]int `json:"actionTypeCounts,omitempty"`
	LastActionTypeCounts   map[string]int `json:"lastActionTypeCounts,omitempty"`
	ElapsedMillis          int64          `json:"elapsedMillis"`
	LegalMillis            int64          `json:"legalMillis"`
	SearchMillis           int64          `json:"searchMillis"`
	ApplyMillis            int64          `json:"applyMillis"`
	AveragePliesPerEpisode float64        `json:"averagePliesPerEpisode"`
	AverageBranchingFactor float64        `json:"averageBranchingFactor"`
	RecordsPerSecond       float64        `json:"recordsPerSecond"`
	MaxFinalRound          int            `json:"maxFinalRound,omitempty"`
}

type Record struct {
	Episode           int                `json:"episode"`
	Ply               int                `json:"ply"`
	Scenario          string             `json:"scenario"`
	PlayerID          string             `json:"playerId"`
	RootPlayerID      string             `json:"rootPlayerId"`
	Round             int                `json:"round,omitempty"`
	Phase             string             `json:"phase,omitempty"`
	State             string             `json:"state"`
	Encoding          []float64          `json:"encoding"`
	ObservationSchema string             `json:"observationSchema,omitempty"`
	ObservationShape  []int              `json:"observationShape,omitempty"`
	FeatureNames      []string           `json:"featureNames,omitempty"`
	LegalActions      []string           `json:"legalActions"`
	Policy            map[string]float64 `json:"policy"`
	ActionID          string             `json:"actionId"`
	Outcome           float64            `json:"outcome"`
	Terminal          bool               `json:"terminal"`
	Truncated         bool               `json:"truncated"`
}

func Run(writer io.Writer, evaluator model.Evaluator, config Config) error {
	_, err := RunWithMetrics(writer, evaluator, config)
	return err
}

func RunWithMetrics(writer io.Writer, evaluator model.Evaluator, config Config) (Metrics, error) {
	started := time.Now()
	metrics := Metrics{
		ScenarioCounts:       make(map[string]int),
		FinalRoundCounts:     make(map[string]int),
		FinalPhaseCounts:     make(map[string]int),
		TerminalPhaseCounts:  make(map[string]int),
		TruncatedPhaseCounts: make(map[string]int),
		ActionTypeCounts:     make(map[string]int),
		LastActionTypeCounts: make(map[string]int),
	}
	if writer == nil {
		return metrics, fmt.Errorf("nil writer")
	}
	if evaluator == nil {
		evaluator = model.NewHeuristicEvaluator()
	}
	if config.Episodes <= 0 {
		config.Episodes = 1
	}
	if config.MaxPlies <= 0 {
		config.MaxPlies = 200
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(config.RandomSeed))
	encoder := json.NewEncoder(writer)
	for episode := 0; episode < config.Episodes; episode++ {
		position, scenarioName, err := env.SampleScenario(config.Scenario, rng)
		if err != nil {
			return metrics, err
		}
		position.MinPassRound = config.MinPassRound
		metrics.Episodes++
		metrics.ScenarioCounts[scenarioName]++
		var records []Record
		truncated := false
		lastActionType := ""
		for ply := 0; ply < config.MaxPlies && !position.IsTerminal(); ply++ {
			legalStarted := time.Now()
			legal := position.LegalActions()
			metrics.LegalMillis += time.Since(legalStarted).Milliseconds()
			if len(legal) == 0 {
				break
			}
			metrics.AverageBranchingFactor += float64(len(legal))
			searchConfig := config.Search
			if searchConfig.RandomSeed == 0 {
				searchConfig.RandomSeed = rng.Int63()
			}
			searchStarted := time.Now()
			result := mcts.Search(position, evaluator, searchConfig)
			metrics.SearchMillis += time.Since(searchStarted).Milliseconds()
			selected := selectAction(result, rng)
			if selected.ID == "" {
				break
			}
			action, ok := actionByID(legal, selected.ID)
			if !ok {
				return metrics, fmt.Errorf("selected illegal action %s", selected.ID)
			}
			observation := position.Observation()
			record := Record{
				Episode:           episode,
				Ply:               ply,
				Scenario:          scenarioName,
				PlayerID:          action.PlayerID,
				RootPlayerID:      position.RootPlayerID,
				Round:             position.State.Round,
				Phase:             phaseName(position.State.Phase),
				State:             position.SnapshotJSON(),
				Encoding:          observation.Features,
				ObservationSchema: observation.Schema,
				ObservationShape:  observation.Shape,
				LegalActions:      actionIDs(legal),
				Policy:            policyMap(result.Actions),
				ActionID:          action.ID,
			}
			metrics.ActionTypeCounts[action.Type]++
			lastActionType = action.Type
			if ply == 0 {
				record.FeatureNames = observation.FeatureNames
			}
			records = append(records, record)
			applyStarted := time.Now()
			position, err = position.Apply(action)
			metrics.ApplyMillis += time.Since(applyStarted).Milliseconds()
			if err != nil {
				return metrics, err
			}
		}
		if !position.IsTerminal() && len(records) >= config.MaxPlies {
			truncated = true
		}
		finalRound := 0
		finalPhase := "unknown"
		if position != nil && position.State != nil {
			finalRound = position.State.Round
			finalPhase = phaseName(position.State.Phase)
		}
		metrics.FinalRoundCounts[fmt.Sprint(finalRound)]++
		metrics.FinalPhaseCounts[finalPhase]++
		if finalRound > metrics.MaxFinalRound {
			metrics.MaxFinalRound = finalRound
		}
		metrics.Records += len(records)
		metrics.CompletedGames++
		if truncated {
			metrics.TruncatedGames++
			metrics.TruncatedPhaseCounts[finalPhase]++
		}
		if position.IsTerminal() {
			metrics.TerminalGames++
			metrics.TerminalPhaseCounts[finalPhase]++
		}
		if lastActionType != "" {
			metrics.LastActionTypeCounts[lastActionType]++
		}
		for i := range records {
			records[i].Outcome = position.ValueFor(records[i].PlayerID)
			records[i].Terminal = position.IsTerminal()
			records[i].Truncated = truncated
			if err := encoder.Encode(records[i]); err != nil {
				return metrics, err
			}
		}
	}
	metrics.ElapsedMillis = time.Since(started).Milliseconds()
	if metrics.Episodes > 0 {
		metrics.AveragePliesPerEpisode = float64(metrics.Records) / float64(metrics.Episodes)
	}
	if metrics.Records > 0 {
		metrics.AverageBranchingFactor /= float64(metrics.Records)
	}
	elapsedSeconds := float64(metrics.ElapsedMillis) / 1000.0
	if elapsedSeconds > 0 {
		metrics.RecordsPerSecond = float64(metrics.Records) / elapsedSeconds
	}
	return metrics, nil
}

func phaseName(phase interface{}) string {
	switch fmt.Sprint(phase) {
	case "0":
		return "setup"
	case "1":
		return "faction_selection"
	case "2":
		return "income"
	case "3":
		return "action"
	case "4":
		return "cleanup"
	case "5":
		return "end"
	default:
		return fmt.Sprint(phase)
	}
}

func selectAction(result mcts.Result, rng *rand.Rand) mcts.RankedAction {
	if len(result.Actions) == 0 {
		return mcts.RankedAction{}
	}
	total := 0.0
	for _, action := range result.Actions {
		total += action.Prob
	}
	if total <= 0 {
		return result.Actions[0]
	}
	target := rng.Float64() * total
	accum := 0.0
	for _, action := range result.Actions {
		accum += action.Prob
		if accum >= target {
			return action
		}
	}
	return result.Actions[0]
}

func actionByID(options []actions.Option, id string) (actions.Option, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return actions.Option{}, false
}

func actionIDs(options []actions.Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.ID)
	}
	return out
}

func policyMap(ranked []mcts.RankedAction) map[string]float64 {
	out := make(map[string]float64, len(ranked))
	for _, action := range ranked {
		out[action.ID] = action.Prob
	}
	return out
}
