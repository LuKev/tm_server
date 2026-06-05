package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/az/arena"
	"github.com/lukev/tm_server/internal/az/dataset"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/az/selfplay"
	"github.com/lukev/tm_server/internal/az/train"
)

type iterationReport struct {
	Iteration       int                     `json:"iteration"`
	StartedAt       string                  `json:"startedAt"`
	FinishedAt      string                  `json:"finishedAt"`
	SelfPlayPath    string                  `json:"selfPlayPath"`
	SelfPlayMetrics selfplay.Metrics        `json:"selfPlayMetrics"`
	SamplesPath     string                  `json:"samplesPath"`
	VocabPath       string                  `json:"vocabPath"`
	ManifestPath    string                  `json:"manifestPath"`
	Candidate       string                  `json:"candidate"`
	Incumbent       string                  `json:"incumbent,omitempty"`
	IncumbentURL    string                  `json:"incumbentUrl,omitempty"`
	Promoted        bool                    `json:"promoted"`
	Promotion       arena.PromotionDecision `json:"promotion"`
	Arena           arena.Result            `json:"arena"`
	Ratings         ratingsFile             `json:"ratings"`
	Config          runConfig               `json:"config"`
	Runtime         runtimeInfo             `json:"runtime"`
}

type runConfig struct {
	Iterations            int         `json:"iterations"`
	Episodes              int         `json:"episodes"`
	Shards                int         `json:"shards"`
	SelfPlayWorkers       int         `json:"selfPlayWorkers,omitempty"`
	ArenaWorkers          int         `json:"arenaWorkers,omitempty"`
	Progress              bool        `json:"progress,omitempty"`
	CompactRecords        bool        `json:"compactRecords,omitempty"`
	ReuseTree             bool        `json:"reuseTree,omitempty"`
	GlobalBatchSize       int         `json:"globalBatchSize,omitempty"`
	GlobalBatchDelayMS    int         `json:"globalBatchDelayMs,omitempty"`
	Scenario              string      `json:"scenario"`
	MaxPlies              int         `json:"maxPlies"`
	Search                mcts.Config `json:"search"`
	ArenaGames            int         `json:"arenaGames"`
	PromoteWinRate        float64     `json:"promoteWinRate"`
	PromoteMinGames       int         `json:"promoteMinGames,omitempty"`
	PromoteCI95LowerBound float64     `json:"promoteCi95LowerBound,omitempty"`
	PromoteFirstCandidate bool        `json:"promoteFirstCandidate"`
	Seed                  int64       `json:"seed"`
}

type runtimeInfo struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	NumCPU    int    `json:"numCpu"`
	GoVersion string `json:"goVersion"`
}

type ratingsFile struct {
	Version int                `json:"version"`
	Models  map[string]float64 `json:"models"`
	History []ratingEvent      `json:"history"`
}

type ratingEvent struct {
	Iteration    int     `json:"iteration"`
	Candidate    string  `json:"candidate"`
	Incumbent    string  `json:"incumbent"`
	Games        int     `json:"games"`
	Score        float64 `json:"score"`
	OldCandidate float64 `json:"oldCandidate"`
	NewCandidate float64 `json:"newCandidate"`
	OldIncumbent float64 `json:"oldIncumbent"`
	NewIncumbent float64 `json:"newIncumbent"`
}

func main() {
	workDir := flag.String("work_dir", "az_runs", "training loop output directory")
	iterations := flag.Int("iterations", 1, "training iterations")
	episodes := flag.Int("episodes", 8, "self-play episodes per iteration")
	shards := flag.Int("shards", 1, "parallel self-play shards per iteration")
	selfPlayWorkers := flag.Int("selfplay_workers", 1, "parallel episode workers inside each self-play shard")
	arenaWorkers := flag.Int("arena_workers", 1, "parallel arena game workers")
	progress := flag.Bool("progress", false, "write per-game self-play and arena progress JSON to stderr")
	compactRecords := flag.Bool("compact_records", false, "omit debug state snapshots from self-play JSONL records")
	reuseTree := flag.Bool("reuse_tree", false, "reuse the selected MCTS subtree between moves in each self-play game")
	globalBatchSize := flag.Int("global_batch_size", 0, "merge concurrent evaluator batches up to this size; 0 disables")
	globalBatchDelay := flag.Int("global_batch_delay_ms", 1, "maximum delay before flushing a partial global evaluator batch")
	scenario := flag.String("scenario", "random_base", "scenario name, random_base, or comma-separated scenario set")
	maxPlies := flag.Int("max_plies", 500, "maximum plies per self-play game")
	sims := flag.Int("sims", 8, "MCTS simulations per move")
	batchSize := flag.Int("batch_size", 1, "MCTS neural evaluation batch size when evaluator supports it")
	maxDepth := flag.Int("max_depth", 80, "MCTS simulation max depth")
	arenaGames := flag.Int("arena_games", 2, "candidate-vs-incumbent arena games")
	promoteWinRate := flag.Float64("promote_win_rate", 0.55, "minimum arena score to promote candidate")
	promoteMinGames := flag.Int("promote_min_games", 0, "minimum arena games required before promotion; 0 disables")
	promoteCI95LowerBound := flag.Float64("promote_ci95_lower_bound", 0, "minimum 95% confidence interval lower bound required before promotion; 0 disables")
	promoteFirstCandidate := flag.Bool("promote_first_candidate", true, "promote the first candidate when no retained best_model.json exists")
	baseModel := flag.String("base_model", "", "optional table model JSON used before best_model.json exists")
	baseModelURL := flag.String("base_model_url", "", "optional HTTP evaluator URL used before best_model.json exists")
	seed := flag.Int64("seed", 1, "random seed")
	flag.Parse()

	if *iterations <= 0 {
		exitf("-iterations must be positive")
	}
	if *episodes <= 0 {
		exitf("-episodes must be positive")
	}
	if *shards <= 0 {
		exitf("-shards must be positive")
	}
	if *selfPlayWorkers <= 0 {
		exitf("-selfplay_workers must be positive")
	}
	if *arenaWorkers <= 0 {
		exitf("-arena_workers must be positive")
	}
	if err := os.MkdirAll(*workDir, 0755); err != nil {
		exitf("create work dir: %v", err)
	}

	bestPath := filepath.Join(*workDir, "best_model.json")
	for iteration := 1; iteration <= *iterations; iteration++ {
		iterDir := filepath.Join(*workDir, fmt.Sprintf("iter_%04d", iteration))
		if err := os.MkdirAll(iterDir, 0755); err != nil {
			exitf("create iteration dir: %v", err)
		}
		startedAt := time.Now()
		incumbentPath := bestPath
		incumbentURL := ""
		if !fileExists(bestPath) {
			incumbentPath = *baseModel
			incumbentURL = *baseModelURL
		}
		incumbent := loadEvaluator(incumbentPath, incumbentURL)
		incumbent = model.NewAsyncBatchEvaluator(incumbent, *globalBatchSize, time.Duration(*globalBatchDelay)*time.Millisecond)
		selfPlayPath := filepath.Join(iterDir, "selfplay.jsonl")
		searchConfig := mcts.Config{
			Simulations: *sims,
			BatchSize:   *batchSize,
			CPUCT:       1.5,
			Temperature: 1,
			MaxDepth:    *maxDepth,
		}
		progressOut := progressWriter(*progress)
		selfPlayMetrics, err := generateSelfPlay(selfPlayPath, incumbent, selfplay.Config{
			Episodes:       *episodes,
			MaxPlies:       *maxPlies,
			Scenario:       *scenario,
			Workers:        *selfPlayWorkers,
			ProgressWriter: progressOut,
			CompactRecords: *compactRecords,
			ReuseTree:      *reuseTree,
			RandomSeed:     *seed + int64(iteration*1000),
			Search:         searchConfig,
		}, *shards)
		if err != nil {
			exitf("self-play iteration %d: %v", iteration, err)
		}
		candidate, err := train.TrainFile(selfPlayPath)
		if err != nil {
			exitf("train iteration %d: %v", iteration, err)
		}
		candidatePath := filepath.Join(iterDir, "candidate_model.json")
		if err := model.SaveTableModel(candidatePath, candidate); err != nil {
			exitf("save candidate: %v", err)
		}
		samplesPath := filepath.Join(iterDir, "samples.jsonl")
		vocabPath := filepath.Join(iterDir, "action_vocab.json")
		manifestPath := filepath.Join(iterDir, "dataset_manifest.json")
		if _, err := dataset.Export(dataset.ExportConfig{
			Input:        selfPlayPath,
			SamplesPath:  samplesPath,
			VocabPath:    vocabPath,
			ManifestPath: manifestPath,
		}); err != nil {
			exitf("export dataset: %v", err)
		}
		arenaResult, err := arena.Evaluate(candidate, incumbent, arena.Config{
			Games:          *arenaGames,
			MaxPlies:       *maxPlies,
			Scenario:       *scenario,
			Workers:        *arenaWorkers,
			ProgressWriter: progressOut,
			RandomSeed:     *seed + int64(iteration*1000) + 77,
			Search: mcts.Config{
				Simulations: *sims,
				BatchSize:   *batchSize,
				CPUCT:       1.5,
				Temperature: 0,
				MaxDepth:    *maxDepth,
			},
		})
		if err != nil {
			exitf("arena iteration %d: %v", iteration, err)
		}
		ratingsPath := filepath.Join(*workDir, "ratings.json")
		ratings := loadRatings(ratingsPath)
		incumbentID := incumbentPath
		if incumbentID == "" {
			incumbentID = incumbentURL
		}
		if incumbentID == "" {
			incumbentID = "heuristic"
		}
		ratings = updateRatings(ratings, iteration, candidatePath, incumbentID, arenaResult)
		if err := writeJSON(ratingsPath, ratings); err != nil {
			exitf("write ratings: %v", err)
		}
		promotion := arena.DecidePromotion(arenaResult, arena.PromotionPolicy{
			MinWinRate:        *promoteWinRate,
			MinGames:          *promoteMinGames,
			MinCI95LowerBound: *promoteCI95LowerBound,
			AutoPromote:       !fileExists(bestPath) && *promoteFirstCandidate,
		})
		promoted := promotion.Promoted
		if promoted {
			if err := model.SaveTableModel(bestPath, candidate); err != nil {
				exitf("promote candidate: %v", err)
			}
		}
		report := iterationReport{
			Iteration:       iteration,
			StartedAt:       startedAt.UTC().Format(time.RFC3339),
			FinishedAt:      time.Now().UTC().Format(time.RFC3339),
			SelfPlayPath:    selfPlayPath,
			SelfPlayMetrics: selfPlayMetrics,
			SamplesPath:     samplesPath,
			VocabPath:       vocabPath,
			ManifestPath:    manifestPath,
			Candidate:       candidatePath,
			Incumbent:       incumbentPath,
			IncumbentURL:    incumbentURL,
			Promoted:        promoted,
			Promotion:       promotion,
			Arena:           arenaResult,
			Ratings:         ratings,
			Config: runConfig{
				Iterations:            *iterations,
				Episodes:              *episodes,
				Shards:                *shards,
				SelfPlayWorkers:       *selfPlayWorkers,
				ArenaWorkers:          *arenaWorkers,
				Progress:              *progress,
				CompactRecords:        *compactRecords,
				ReuseTree:             *reuseTree,
				GlobalBatchSize:       *globalBatchSize,
				GlobalBatchDelayMS:    *globalBatchDelay,
				Scenario:              *scenario,
				MaxPlies:              *maxPlies,
				Search:                searchConfig,
				ArenaGames:            *arenaGames,
				PromoteWinRate:        *promoteWinRate,
				PromoteMinGames:       *promoteMinGames,
				PromoteCI95LowerBound: *promoteCI95LowerBound,
				PromoteFirstCandidate: *promoteFirstCandidate,
				Seed:                  *seed,
			},
			Runtime: runtimeInfo{
				GOOS:      runtime.GOOS,
				GOARCH:    runtime.GOARCH,
				NumCPU:    runtime.NumCPU(),
				GoVersion: runtime.Version(),
			},
		}
		if err := writeJSON(filepath.Join(iterDir, "report.json"), report); err != nil {
			exitf("write report: %v", err)
		}
		raw, _ := json.Marshal(report)
		_, _ = fmt.Fprintln(os.Stderr, string(raw))
	}
}

func generateSelfPlay(path string, evaluator model.Evaluator, config selfplay.Config, shards int) (selfplay.Metrics, error) {
	started := time.Now()
	remaining := config.Episodes
	shardConfigs := make([]selfplay.Config, 0, shards)
	shardPaths := make([]string, 0, shards)
	for shard := 0; shard < shards; shard++ {
		shardEpisodes := remaining / (shards - shard)
		remaining -= shardEpisodes
		if shardEpisodes <= 0 {
			continue
		}
		shardConfig := config
		shardConfig.Episodes = shardEpisodes
		shardConfig.RandomSeed += int64(shard)
		shardConfigs = append(shardConfigs, shardConfig)
		shardPaths = append(shardPaths, fmt.Sprintf("%s.shard_%04d", path, shard))
	}
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		total  selfplay.Metrics
		runErr error
	)
	for i, shardConfig := range shardConfigs {
		wg.Add(1)
		go func(shardPath string, cfg selfplay.Config) {
			defer wg.Done()
			file, err := os.Create(shardPath)
			if err != nil {
				mu.Lock()
				if runErr == nil {
					runErr = err
				}
				mu.Unlock()
				return
			}
			metrics, err := selfplay.RunWithMetrics(file, evaluator, cfg)
			closeErr := file.Close()
			mu.Lock()
			defer mu.Unlock()
			if err != nil && runErr == nil {
				runErr = err
				return
			}
			if closeErr != nil && runErr == nil {
				runErr = closeErr
				return
			}
			total = mergeMetrics(total, metrics)
		}(shardPaths[i], shardConfig)
	}
	wg.Wait()
	if runErr != nil {
		return total, runErr
	}
	if err := concatenateFiles(path, shardPaths); err != nil {
		return total, err
	}
	for _, shardPath := range shardPaths {
		_ = os.Remove(shardPath)
	}
	total.ElapsedMillis = time.Since(started).Milliseconds()
	return finalizeMergedMetrics(total), nil
}

func concatenateFiles(path string, inputs []string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	for _, input := range inputs {
		in, err := os.Open(input)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}

func mergeMetrics(total, shard selfplay.Metrics) selfplay.Metrics {
	if total.ScenarioCounts == nil {
		total.ScenarioCounts = make(map[string]int)
	}
	ensureSelfPlayMetricMaps(&total)
	weightedBranching := total.AverageBranchingFactor * float64(total.Records)
	weightedPlies := total.AveragePliesPerEpisode * float64(total.Episodes)
	total.Episodes += shard.Episodes
	total.Records += shard.Records
	total.CompletedGames += shard.CompletedGames
	total.TruncatedGames += shard.TruncatedGames
	total.TerminalGames += shard.TerminalGames
	total.LegalMillis += shard.LegalMillis
	total.SearchMillis += shard.SearchMillis
	total.ApplyMillis += shard.ApplyMillis
	total.LegalNanos += shard.LegalNanos
	total.SearchNanos += shard.SearchNanos
	total.ApplyNanos += shard.ApplyNanos
	total.Workers += shard.Workers
	if shard.MaxFinalRound > total.MaxFinalRound {
		total.MaxFinalRound = shard.MaxFinalRound
	}
	weightedBranching += shard.AverageBranchingFactor * float64(shard.Records)
	weightedPlies += shard.AveragePliesPerEpisode * float64(shard.Episodes)
	for scenario, count := range shard.ScenarioCounts {
		total.ScenarioCounts[scenario] += count
	}
	mergeIntMap(total.OrderedMatchupCounts, shard.OrderedMatchupCounts)
	mergeIntMap(total.UnorderedMatchupCounts, shard.UnorderedMatchupCounts)
	mergeIntMap(total.RootFactionCounts, shard.RootFactionCounts)
	mergeIntMap(total.FinalRoundCounts, shard.FinalRoundCounts)
	mergeIntMap(total.FinalPhaseCounts, shard.FinalPhaseCounts)
	mergeIntMap(total.TerminalPhaseCounts, shard.TerminalPhaseCounts)
	mergeIntMap(total.TruncatedPhaseCounts, shard.TruncatedPhaseCounts)
	mergeIntMap(total.ActionTypeCounts, shard.ActionTypeCounts)
	mergeIntMap(total.LastActionTypeCounts, shard.LastActionTypeCounts)
	if total.Records > 0 {
		total.AverageBranchingFactor = weightedBranching / float64(total.Records)
	}
	if total.Episodes > 0 {
		total.AveragePliesPerEpisode = weightedPlies / float64(total.Episodes)
	}
	return total
}

func ensureSelfPlayMetricMaps(metrics *selfplay.Metrics) {
	if metrics.FinalRoundCounts == nil {
		metrics.FinalRoundCounts = make(map[string]int)
	}
	if metrics.OrderedMatchupCounts == nil {
		metrics.OrderedMatchupCounts = make(map[string]int)
	}
	if metrics.UnorderedMatchupCounts == nil {
		metrics.UnorderedMatchupCounts = make(map[string]int)
	}
	if metrics.RootFactionCounts == nil {
		metrics.RootFactionCounts = make(map[string]int)
	}
	if metrics.FinalPhaseCounts == nil {
		metrics.FinalPhaseCounts = make(map[string]int)
	}
	if metrics.TerminalPhaseCounts == nil {
		metrics.TerminalPhaseCounts = make(map[string]int)
	}
	if metrics.TruncatedPhaseCounts == nil {
		metrics.TruncatedPhaseCounts = make(map[string]int)
	}
	if metrics.ActionTypeCounts == nil {
		metrics.ActionTypeCounts = make(map[string]int)
	}
	if metrics.LastActionTypeCounts == nil {
		metrics.LastActionTypeCounts = make(map[string]int)
	}
}

func mergeIntMap(total, shard map[string]int) {
	for key, count := range shard {
		total[key] += count
	}
}

func finalizeMergedMetrics(metrics selfplay.Metrics) selfplay.Metrics {
	metrics.LegalMillis = metrics.LegalNanos / int64(time.Millisecond)
	metrics.SearchMillis = metrics.SearchNanos / int64(time.Millisecond)
	metrics.ApplyMillis = metrics.ApplyNanos / int64(time.Millisecond)
	elapsedSeconds := float64(metrics.ElapsedMillis) / 1000.0
	if elapsedSeconds > 0 {
		metrics.RecordsPerSecond = float64(metrics.Records) / elapsedSeconds
	}
	return metrics
}

type lockedWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

func progressWriter(enabled bool) io.Writer {
	if !enabled {
		return nil
	}
	return &lockedWriter{writer: os.Stderr}
}

func loadRatings(path string) ratingsFile {
	ratings := ratingsFile{Version: 1, Models: make(map[string]float64)}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ratings
	}
	if err := json.Unmarshal(raw, &ratings); err != nil {
		return ratingsFile{Version: 1, Models: make(map[string]float64)}
	}
	if ratings.Version == 0 {
		ratings.Version = 1
	}
	if ratings.Models == nil {
		ratings.Models = make(map[string]float64)
	}
	return ratings
}

func updateRatings(ratings ratingsFile, iteration int, candidateID, incumbentID string, result arena.Result) ratingsFile {
	if ratings.Version == 0 {
		ratings.Version = 1
	}
	if ratings.Models == nil {
		ratings.Models = make(map[string]float64)
	}
	oldCandidate := ratingFor(ratings, candidateID)
	oldIncumbent := ratingFor(ratings, incumbentID)
	expectedCandidate := 1.0 / (1.0 + pow10((oldIncumbent-oldCandidate)/400.0))
	k := 32.0
	if result.Games >= 50 {
		k = 24.0
	}
	if result.Games >= 200 {
		k = 16.0
	}
	newCandidate := oldCandidate + k*(result.WinRate-expectedCandidate)
	newIncumbent := oldIncumbent + k*((1-result.WinRate)-(1-expectedCandidate))
	ratings.Models[candidateID] = newCandidate
	ratings.Models[incumbentID] = newIncumbent
	ratings.History = append(ratings.History, ratingEvent{
		Iteration:    iteration,
		Candidate:    candidateID,
		Incumbent:    incumbentID,
		Games:        result.Games,
		Score:        result.WinRate,
		OldCandidate: oldCandidate,
		NewCandidate: newCandidate,
		OldIncumbent: oldIncumbent,
		NewIncumbent: newIncumbent,
	})
	return ratings
}

func ratingFor(ratings ratingsFile, id string) float64 {
	if id == "" {
		id = "unknown"
	}
	if rating, ok := ratings.Models[id]; ok {
		return rating
	}
	return 1500
}

func pow10(value float64) float64 {
	return math.Pow(10, value)
}

func loadEvaluator(path, url string) model.Evaluator {
	if path != "" && !fileExists(path) {
		path = ""
	}
	evaluator, err := model.LoadEvaluator(model.EvaluatorConfig{
		TableModelPath: path,
		HTTPURL:        url,
	})
	if err == nil {
		return evaluator
	}
	return model.NewHeuristicEvaluator()
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func writeJSON(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
