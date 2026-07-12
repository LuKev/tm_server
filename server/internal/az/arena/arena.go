package arena

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/az/stats"
)

type Config struct {
	Games          int
	MaxPlies       int
	Scenario       string
	Workers        int
	ProgressWriter io.Writer
	Search         mcts.Config
	RandomSeed     int64
}

type Result struct {
	Games                          int                      `json:"games"`
	CandidateWins                  int                      `json:"candidateWins"`
	BaselineWins                   int                      `json:"baselineWins"`
	Draws                          int                      `json:"draws"`
	WinRate                        float64                  `json:"winRate"`
	WinRateStdErr                  float64                  `json:"winRateStdErr"`
	WinRateCI95                    [2]float64               `json:"winRateCi95"`
	AverageMargin                  float64                  `json:"averageMargin"`
	CandidateAverageScore          float64                  `json:"candidateAverageScore"`
	BaselineAverageScore           float64                  `json:"baselineAverageScore"`
	AverageScoreDifference         float64                  `json:"averageScoreDifference"`
	CandidateScoreTotal            int                      `json:"candidateScoreTotal"`
	BaselineScoreTotal             int                      `json:"baselineScoreTotal"`
	CandidateFinalScoresByFaction  stats.FinalScoreRates    `json:"candidateFinalScoresByFaction,omitempty"`
	BaselineFinalScoresByFaction   stats.FinalScoreRates    `json:"baselineFinalScoresByFaction,omitempty"`
	AveragePlies                   float64                  `json:"averagePlies"`
	TerminalGames                  int                      `json:"terminalGames"`
	TruncatedGames                 int                      `json:"truncatedGames"`
	ScenarioCounts                 map[string]int           `json:"scenarioCounts,omitempty"`
	OrderedMatchupCounts           map[string]int           `json:"orderedMatchupCounts,omitempty"`
	UnorderedMatchupStats          map[string]MatchupResult `json:"unorderedMatchupStats,omitempty"`
	CandidateR1BuildRatesByFaction stats.R1BuildRates       `json:"candidateR1BuildRatesByFaction,omitempty"`
	BaselineR1BuildRatesByFaction  stats.R1BuildRates       `json:"baselineR1BuildRatesByFaction,omitempty"`
	FinalRoundCounts               map[string]int           `json:"finalRoundCounts,omitempty"`
	FinalPhaseCounts               map[string]int           `json:"finalPhaseCounts,omitempty"`
	TerminalPhaseCounts            map[string]int           `json:"terminalPhaseCounts,omitempty"`
	TruncatedPhaseCounts           map[string]int           `json:"truncatedPhaseCounts,omitempty"`
	SearchSimulations              int                      `json:"searchSimulations"`
	ElapsedMillis                  int64                    `json:"elapsedMillis,omitempty"`
	SearchMillis                   int64                    `json:"searchMillis,omitempty"`
	SearchNanos                    int64                    `json:"searchNanos,omitempty"`
	Workers                        int                      `json:"workers,omitempty"`
}

type MatchupResult struct {
	Games          int     `json:"games"`
	CandidateWins  int     `json:"candidateWins"`
	BaselineWins   int     `json:"baselineWins"`
	Draws          int     `json:"draws"`
	WinRate        float64 `json:"winRate"`
	AverageMargin  float64 `json:"averageMargin"`
	AveragePlies   float64 `json:"averagePlies"`
	TerminalGames  int     `json:"terminalGames"`
	TruncatedGames int     `json:"truncatedGames"`
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
		config.MaxPlies = 500
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
	if config.Workers <= 0 {
		config.Workers = 1
	}
	if config.Workers > config.Games {
		config.Workers = config.Games
	}
	started := time.Now()
	result := Result{
		ScenarioCounts:                 make(map[string]int),
		OrderedMatchupCounts:           make(map[string]int),
		UnorderedMatchupStats:          make(map[string]MatchupResult),
		CandidateR1BuildRatesByFaction: make(stats.R1BuildRates),
		BaselineR1BuildRatesByFaction:  make(stats.R1BuildRates),
		CandidateFinalScoresByFaction:  make(stats.FinalScoreRates),
		BaselineFinalScoresByFaction:   make(stats.FinalScoreRates),
		FinalRoundCounts:               make(map[string]int),
		FinalPhaseCounts:               make(map[string]int),
		TerminalPhaseCounts:            make(map[string]int),
		TruncatedPhaseCounts:           make(map[string]int),
		SearchSimulations:              config.Search.Simulations,
		Workers:                        config.Workers,
	}
	jobs := make(chan int)
	results := make(chan gameResult)
	var wg sync.WaitGroup
	for workerID := 0; workerID < config.Workers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for gameIndex := range jobs {
				results <- evaluateGame(gameIndex, workerID, candidate, baseline, config)
			}
		}(workerID)
	}
	go func() {
		for gameIndex := 0; gameIndex < config.Games; gameIndex++ {
			jobs <- gameIndex
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	var firstErr error
	for game := range results {
		if game.err != nil {
			if firstErr == nil {
				firstErr = game.err
			}
			continue
		}
		mergeGameResult(&result, game)
		if config.ProgressWriter != nil {
			writeProgress(config.ProgressWriter, game, result, time.Since(started), config.Games)
		}
	}
	result.ElapsedMillis = time.Since(started).Milliseconds()
	if firstErr != nil {
		return result, firstErr
	}
	if result.Games > 0 {
		result.AverageMargin /= float64(result.Games)
		result.AveragePlies /= float64(result.Games)
		result.CandidateAverageScore = float64(result.CandidateScoreTotal) / float64(result.Games)
		result.BaselineAverageScore = float64(result.BaselineScoreTotal) / float64(result.Games)
		result.AverageScoreDifference = result.CandidateAverageScore - result.BaselineAverageScore
		result.WinRate = (float64(result.CandidateWins) + 0.5*float64(result.Draws)) / float64(result.Games)
		result.WinRateStdErr = math.Sqrt(result.WinRate * (1 - result.WinRate) / float64(result.Games))
		margin := 1.96 * result.WinRateStdErr
		result.WinRateCI95 = [2]float64{math.Max(0, result.WinRate-margin), math.Min(1, result.WinRate+margin)}
	}
	return result, nil
}

type gameResult struct {
	gameIndex          int
	workerID           int
	scenario           string
	metadata           env.ScenarioMetadata
	candidatePlayer    string
	plies              int
	margin             float64
	candidateScore     int
	baselineScore      int
	candidateFaction   string
	baselineFaction    string
	finalRound         int
	finalPhase         string
	terminal           bool
	truncated          bool
	searchNanos        int64
	candidateR1Samples []stats.R1BuildSample
	baselineR1Samples  []stats.R1BuildSample
	elapsed            time.Duration
	err                error
}

func evaluateGame(gameIndex, workerID int, candidate, baseline model.Evaluator, config Config) gameResult {
	started := time.Now()
	rng := rand.New(rand.NewSource(gameSeed(config.RandomSeed, gameIndex)))
	scenarioRequest := env.ScheduledScenario(config.Scenario, gameIndex)
	position, scenarioName, err := env.SampleScenario(scenarioRequest, rng)
	if err != nil {
		return gameResult{gameIndex: gameIndex, workerID: workerID, err: err}
	}
	candidatePlayer := "p1"
	if gameIndex%2 == 1 {
		candidatePlayer = "p2"
	}
	out := gameResult{
		gameIndex:       gameIndex,
		workerID:        workerID,
		scenario:        scenarioName,
		metadata:        position.Metadata,
		candidatePlayer: candidatePlayer,
	}
	if out.metadata.Scenario == "" {
		out.metadata.Scenario = scenarioName
	}
	r1Tracker := stats.NewR1BuildTracker(position.State, playerIDs(position))
	for ply := 0; ply < config.MaxPlies && !position.IsTerminal(); ply++ {
		out.plies++
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
		searchStarted := time.Now()
		search := mcts.Search(position, evaluator, searchConfig)
		out.searchNanos += time.Since(searchStarted).Nanoseconds()
		selected := selectAction(search.Actions)
		if selected.ID == "" {
			break
		}
		option, ok := actionByID(legal, selected.ID)
		if !ok {
			out.err = fmt.Errorf("selected illegal action %s", selected.ID)
			return out
		}
		r1Tracker.ObserveAction(position.State.Round, option.PlayerID, option.Type)
		position, err = position.Apply(option)
		if err != nil {
			out.err = err
			return out
		}
		r1Tracker.Observe(position.State)
	}
	r1Tracker.Finalize(position.State)
	for _, sample := range r1Tracker.Samples() {
		if sample.PlayerID == candidatePlayer {
			out.candidateR1Samples = append(out.candidateR1Samples, sample)
		} else {
			out.baselineR1Samples = append(out.baselineR1Samples, sample)
		}
	}
	out.margin = position.ValueFor(candidatePlayer)
	out.candidateScore = env.FinalScoreFor(position.State, candidatePlayer)
	baselinePlayer := "p1"
	if candidatePlayer == "p1" {
		baselinePlayer = "p2"
	}
	out.baselineScore = env.FinalScoreFor(position.State, baselinePlayer)
	if player := position.State.GetPlayer(candidatePlayer); player != nil && player.Faction != nil {
		out.candidateFaction = player.Faction.GetType().String()
	}
	if player := position.State.GetPlayer(baselinePlayer); player != nil && player.Faction != nil {
		out.baselineFaction = player.Faction.GetType().String()
	}
	if position != nil && position.State != nil {
		out.finalRound = position.State.Round
		out.finalPhase = phaseName(position.State.Phase)
	}
	out.terminal = position.IsTerminal()
	out.truncated = !out.terminal && out.plies >= config.MaxPlies
	out.elapsed = time.Since(started)
	return out
}

func mergeGameResult(result *Result, game gameResult) {
	result.Games++
	result.ScenarioCounts[game.scenario]++
	if game.metadata.OrderedMatchup != "" {
		result.OrderedMatchupCounts[game.metadata.OrderedMatchup]++
	}
	result.AveragePlies += float64(game.plies)
	result.AverageMargin += game.margin
	result.CandidateScoreTotal += game.candidateScore
	result.BaselineScoreTotal += game.baselineScore
	stats.AddFinalScore(result.CandidateFinalScoresByFaction, game.candidateFaction, game.candidateScore)
	stats.AddFinalScore(result.BaselineFinalScoresByFaction, game.baselineFaction, game.baselineScore)
	result.SearchNanos += game.searchNanos
	result.SearchMillis = result.SearchNanos / int64(time.Millisecond)
	result.FinalRoundCounts[fmt.Sprint(game.finalRound)]++
	result.FinalPhaseCounts[game.finalPhase]++
	if game.terminal {
		result.TerminalGames++
		result.TerminalPhaseCounts[game.finalPhase]++
	}
	if game.truncated {
		result.TruncatedGames++
		result.TruncatedPhaseCounts[game.finalPhase]++
	}
	stats.AddR1BuildSamples(result.CandidateR1BuildRatesByFaction, game.candidateR1Samples)
	stats.AddR1BuildSamples(result.BaselineR1BuildRatesByFaction, game.baselineR1Samples)
	matchupKey := game.metadata.UnorderedMatchup
	if matchupKey != "" {
		stats := result.UnorderedMatchupStats[matchupKey]
		previousGames := stats.Games
		stats.Games++
		stats.AveragePlies = runningAverage(stats.AveragePlies, float64(game.plies), previousGames)
		stats.AverageMargin = runningAverage(stats.AverageMargin, game.margin, previousGames)
		if game.terminal {
			stats.TerminalGames++
		}
		if game.truncated {
			stats.TruncatedGames++
		}
		switch {
		case game.margin > 0.01:
			stats.CandidateWins++
		case game.margin < -0.01:
			stats.BaselineWins++
		default:
			stats.Draws++
		}
		stats.WinRate = (float64(stats.CandidateWins) + 0.5*float64(stats.Draws)) / float64(stats.Games)
		result.UnorderedMatchupStats[matchupKey] = stats
	}
	switch {
	case game.margin > 0.01:
		result.CandidateWins++
	case game.margin < -0.01:
		result.BaselineWins++
	default:
		result.Draws++
	}
}

func runningAverage(current, next float64, previousCount int) float64 {
	if previousCount <= 0 {
		return next
	}
	return (current*float64(previousCount) + next) / float64(previousCount+1)
}

func writeProgress(writer io.Writer, game gameResult, result Result, elapsed time.Duration, totalGames int) {
	progress := map[string]interface{}{
		"event":             "arena_game",
		"game":              game.gameIndex,
		"worker":            game.workerID,
		"scenario":          game.scenario,
		"orderedMatchup":    game.metadata.OrderedMatchup,
		"unorderedMatchup":  game.metadata.UnorderedMatchup,
		"rootFaction":       game.metadata.RootFaction,
		"candidatePlayer":   game.candidatePlayer,
		"plies":             game.plies,
		"finalRound":        game.finalRound,
		"finalPhase":        game.finalPhase,
		"terminal":          game.terminal,
		"truncated":         game.truncated,
		"margin":            game.margin,
		"candidateScore":    game.candidateScore,
		"baselineScore":     game.baselineScore,
		"gameElapsedMillis": game.elapsed.Milliseconds(),
		"completedGames":    result.Games,
		"totalGames":        totalGames,
		"elapsedMillis":     elapsed.Milliseconds(),
		"candidateWins":     result.CandidateWins,
		"baselineWins":      result.BaselineWins,
		"draws":             result.Draws,
	}
	raw, err := json.Marshal(progress)
	if err == nil {
		_, _ = fmt.Fprintln(writer, string(raw))
	}
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

func gameSeed(base int64, gameIndex int) int64 {
	return base + int64(gameIndex+1)*1000003
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

func playerIDs(position *env.Position) []string {
	if position == nil || position.State == nil {
		return nil
	}
	out := make([]string, 0, len(position.State.Players))
	for _, player := range position.State.Players {
		if player != nil && player.ID != "" {
			out = append(out, player.ID)
		}
	}
	return out
}
