package arena

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

type Config struct {
	Games        int
	MaxPlies     int
	Scenario     string
	MinPassRound int
	Search       mcts.Config
	RandomSeed   int64
}

type Result struct {
	Games             int            `json:"games"`
	CandidateWins     int            `json:"candidateWins"`
	BaselineWins      int            `json:"baselineWins"`
	Draws             int            `json:"draws"`
	WinRate           float64        `json:"winRate"`
	WinRateStdErr     float64        `json:"winRateStdErr"`
	WinRateCI95       [2]float64     `json:"winRateCi95"`
	AverageMargin     float64        `json:"averageMargin"`
	AveragePlies      float64        `json:"averagePlies"`
	ScenarioCounts    map[string]int `json:"scenarioCounts,omitempty"`
	SearchSimulations int            `json:"searchSimulations"`
}

type PromotionPolicy struct {
	MinWinRate        float64 `json:"minWinRate"`
	MinGames          int     `json:"minGames,omitempty"`
	MinCI95LowerBound float64 `json:"minCi95LowerBound,omitempty"`
	AutoPromote       bool    `json:"autoPromote,omitempty"`
}

type PromotionDecision struct {
	Promoted        bool            `json:"promoted"`
	Games           int             `json:"games"`
	WinRate         float64         `json:"winRate"`
	WinRateCI95     [2]float64      `json:"winRateCi95"`
	Policy          PromotionPolicy `json:"policy"`
	BlockingReasons []string        `json:"blockingReasons,omitempty"`
	Notes           []string        `json:"notes,omitempty"`
}

// Evaluate plays candidate and baseline through MCTS and returns candidate
// promotion metrics. Candidate alternates between p1 and p2 across games.
func Evaluate(candidate, baseline model.Evaluator, config Config) (Result, error) {
	if candidate == nil {
		return Result{}, fmt.Errorf("nil candidate evaluator")
	}
	if baseline == nil {
		baseline = model.NewHeuristicEvaluator()
	}
	if config.Games <= 0 {
		config.Games = 2
	}
	if config.MaxPlies <= 0 {
		config.MaxPlies = 200
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(config.RandomSeed))
	result := Result{
		ScenarioCounts:    make(map[string]int),
		SearchSimulations: config.Search.Simulations,
	}
	for gameIndex := 0; gameIndex < config.Games; gameIndex++ {
		position, scenarioName, err := env.SampleScenario(config.Scenario, rng)
		if err != nil {
			return result, err
		}
		position.MinPassRound = config.MinPassRound
		result.ScenarioCounts[scenarioName]++
		candidatePlayer := "p1"
		if gameIndex%2 == 1 {
			candidatePlayer = "p2"
		}
		plies := 0
		for ply := 0; ply < config.MaxPlies && !position.IsTerminal(); ply++ {
			plies++
			legal := position.LegalActions()
			if len(legal) == 0 {
				break
			}
			currentPlayer := legal[0].PlayerID
			evaluator := baseline
			if currentPlayer == candidatePlayer {
				evaluator = candidate
			}
			searchConfig := config.Search
			if searchConfig.RandomSeed == 0 {
				searchConfig.RandomSeed = rng.Int63()
			}
			searchConfig.Temperature = 0
			search := mcts.Search(position, evaluator, searchConfig)
			selected := selectAction(search.Actions)
			if selected.ID == "" {
				break
			}
			option, ok := actionByID(legal, selected.ID)
			if !ok {
				return result, fmt.Errorf("selected illegal action %s", selected.ID)
			}
			position, err = position.Apply(option)
			if err != nil {
				return result, err
			}
		}
		margin := position.ValueFor(candidatePlayer)
		result.Games++
		result.AveragePlies += float64(plies)
		result.AverageMargin += margin
		switch {
		case margin > 0.01:
			result.CandidateWins++
		case margin < -0.01:
			result.BaselineWins++
		default:
			result.Draws++
		}
	}
	if result.Games > 0 {
		result.AverageMargin /= float64(result.Games)
		result.AveragePlies /= float64(result.Games)
		result.WinRate = (float64(result.CandidateWins) + 0.5*float64(result.Draws)) / float64(result.Games)
		result.WinRateStdErr = math.Sqrt(result.WinRate * (1 - result.WinRate) / float64(result.Games))
		margin := 1.96 * result.WinRateStdErr
		result.WinRateCI95 = [2]float64{math.Max(0, result.WinRate-margin), math.Min(1, result.WinRate+margin)}
	}
	return result, nil
}

func DecidePromotion(result Result, policy PromotionPolicy) PromotionDecision {
	decision := PromotionDecision{
		Games:       result.Games,
		WinRate:     result.WinRate,
		WinRateCI95: result.WinRateCI95,
		Policy:      policy,
	}
	if policy.AutoPromote {
		decision.Promoted = true
		decision.Notes = append(decision.Notes, "auto-promoted because no retained incumbent exists")
		return decision
	}
	if policy.MinGames > 0 && result.Games < policy.MinGames {
		decision.BlockingReasons = append(decision.BlockingReasons, fmt.Sprintf("games %d below minimum %d", result.Games, policy.MinGames))
	}
	if result.WinRate < policy.MinWinRate {
		decision.BlockingReasons = append(decision.BlockingReasons, fmt.Sprintf("win rate %.4f below minimum %.4f", result.WinRate, policy.MinWinRate))
	}
	if policy.MinCI95LowerBound > 0 && result.WinRateCI95[0] < policy.MinCI95LowerBound {
		decision.BlockingReasons = append(decision.BlockingReasons, fmt.Sprintf("95%% CI lower bound %.4f below minimum %.4f", result.WinRateCI95[0], policy.MinCI95LowerBound))
	}
	decision.Promoted = len(decision.BlockingReasons) == 0
	return decision
}

func selectAction(actions []mcts.RankedAction) mcts.RankedAction {
	if len(actions) == 0 {
		return mcts.RankedAction{}
	}
	return actions[0]
}

func actionByID(options []actions.Option, id string) (actions.Option, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return actions.Option{}, false
}
