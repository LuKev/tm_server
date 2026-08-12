package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukev/tm_server/internal/az"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

var baseFactions = []models.FactionType{
	models.FactionNomads,
	models.FactionFakirs,
	models.FactionChaosMagicians,
	models.FactionGiants,
	models.FactionSwarmlings,
	models.FactionMermaids,
	models.FactionWitches,
	models.FactionAuren,
	models.FactionHalflings,
	models.FactionCultists,
	models.FactionAlchemists,
	models.FactionDarklings,
	models.FactionEngineers,
	models.FactionDwarves,
}

type selfPlayMetrics struct {
	WallSeconds                       float64              `json:"wall_seconds"`
	GenerationSeconds                 float64              `json:"generation_seconds"`
	ShardWriteSeconds                 float64              `json:"shard_write_seconds"`
	GamesPerHour                      float64              `json:"games_per_hour"`
	Decisions                         int                  `json:"decisions"`
	DecisionsPerGame                  float64              `json:"decisions_per_game"`
	ConfiguredSimulations             int                  `json:"configured_simulations_per_decision"`
	SetupSeedStart                    int64                `json:"setup_seed_start"`
	SearchSeedStart                   int64                `json:"search_seed_start"`
	SimulationTraversalsPerSecond     float64              `json:"simulation_traversals_per_second"`
	EvaluationPositionsPerSecond      float64              `json:"evaluation_positions_per_second"`
	InferencePositionsPerSecond       float64              `json:"inference_positions_per_second"`
	MeanInferenceBatchSize            float64              `json:"mean_inference_batch_size"`
	SearchInclusiveWallSeconds        float64              `json:"search_inclusive_wall_seconds"`
	InferenceCumulativeLatencySeconds float64              `json:"inference_cumulative_latency_seconds"`
	InferenceServiceSeconds           float64              `json:"inference_service_seconds"`
	InferenceQueueWaitSeconds         float64              `json:"inference_queue_wait_seconds"`
	RecentInferenceLatencyP50Millis   float64              `json:"recent_inference_latency_p50_ms"`
	RecentInferenceLatencyP95Millis   float64              `json:"recent_inference_latency_p95_ms"`
	RecentInferenceQueueP95Millis     float64              `json:"recent_inference_queue_wait_p95_ms"`
	EvaluationReuseRate               float64              `json:"evaluation_reuse_rate"`
	TrajectoryBytes                   int                  `json:"trajectory_bytes"`
	BytesPerGame                      float64              `json:"bytes_per_game"`
	GameBatches                       int                  `json:"game_batches"`
	PeakConcurrentGames               int                  `json:"peak_concurrent_games"`
	Search                            az.SearchStats       `json:"search"`
	Inference                         az.InferenceStats    `json:"inference"`
	InferenceModel                    az.InferenceMetadata `json:"inference_model"`
}

func main() {
	var (
		inference        = flag.String("inference", "", "path to the az_inference executable")
		checkpoint       = flag.String("checkpoint", "", "optional model checkpoint")
		latest           = flag.String("checkpoint-latest", "", "optional atomic latest.json reference")
		modelConfig      = flag.String("model-config", "auto", "auto, debug, main, or large")
		modelSeed        = flag.Int64("model-seed", 0, "random initialization seed, independent of setup")
		inferenceDevice  = flag.String("inference-device", "cpu", "inference device: cpu, cuda, mps, or auto")
		inferenceThreads = flag.Int("inference-torch-threads", 1, "PyTorch CPU threads; zero keeps the default")
		output           = flag.String("output", "", "trajectory shard directory")
		engineCommit     = flag.String("engine-commit", "", "required engine source revision")
		games            = flag.Int("games", 1, "total number of games")
		gamesPerBatch    = flag.Int("games-per-batch", 8, "maximum lockstep games retained before shard publication")
		simulations      = flag.Int("simulations", 1, "MCTS simulations per decision")
		temperature      = flag.Float64("temperature", 1, "self-play visit temperature")
		seed             = flag.Int64("seed", 1, "first deterministic setup seed")
		searchSeed       = flag.Int64("search-seed", 2000000, "first deterministic search/action seed, independent of setup")
		maxPlies         = flag.Int("max-plies", 2000, "natural-completion tripwire")
		noiseEpsilon     = flag.Float64("dirichlet-epsilon", 0.25, "root exploration noise fraction")
		concentration    = flag.Float64("dirichlet-concentration", 10, "root noise total concentration")
	)
	flag.Parse()
	if *inference == "" || *output == "" || *engineCommit == "" || *games <= 0 || *gamesPerBatch <= 0 || *searchSeed <= 0 {
		fmt.Fprintln(os.Stderr, "--inference, --output, --engine-commit, positive --games, positive --games-per-batch, and positive --search-seed are required")
		os.Exit(2)
	}
	resolvedCheckpoint, err := az.ResolveCheckpointReference(*checkpoint, *latest)
	if err != nil {
		fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	evaluator, err := az.StartPythonEvaluatorWithOptions(ctx, *inference, az.PythonEvaluatorOptions{
		Checkpoint: resolvedCheckpoint, ModelConfig: *modelConfig, Device: *inferenceDevice,
		Seed: *modelSeed, TorchThreads: *inferenceThreads,
	})
	if err != nil {
		fatal(err)
	}
	defer func() {
		if err := evaluator.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close inference:", err)
		}
	}()

	matchups := orderedBaseMatchups()
	offset := int(*seed % int64(len(matchups)))
	if offset < 0 {
		offset += len(matchups)
	}
	telemetry := &az.SearchTelemetry{}
	started := time.Now()
	var generationDuration, writeDuration time.Duration
	shardsWritten := 0
	decisions := 0
	trajectoryBytes := 0
	peakConcurrentGames := 0
	batchCount := 0
	for batchStart := 0; batchStart < *games; batchStart += *gamesPerBatch {
		batchEnd := gameBatchEnd(batchStart, *games, *gamesPerBatch)
		batchSize := batchEnd - batchStart
		batchCount++
		if batchSize > peakConcurrentGames {
			peakConcurrentGames = batchSize
		}
		positions := make([]*az.GamePosition, batchSize)
		seeds := make([]int64, batchSize)
		searchSeeds := make([]int64, batchSize)
		for localIndex := range positions {
			globalIndex := batchStart + localIndex
			seeds[localIndex], searchSeeds[localIndex] = selfPlaySeeds(*seed, *searchSeed, globalIndex)
			matchup := matchups[(offset+globalIndex)%len(matchups)]
			positions[localIndex], err = az.NewBaseGame(seeds[localIndex], matchup[0], matchup[1])
			if err != nil {
				fatal(err)
			}
		}

		generationStarted := time.Now()
		trajectories, generateErr := az.GenerateSelfPlay(ctx, positions, seeds, evaluator, az.SelfPlayConfig{
			SearchSeeds: searchSeeds,
			Search: az.SearchConfig{
				Simulations:                 *simulations,
				CPUCT:                       1.5,
				DirichletTotalConcentration: *concentration,
				DirichletEpsilon:            *noiseEpsilon,
				Telemetry:                   telemetry,
			},
			Temperature:      *temperature,
			MaxPlies:         *maxPlies,
			EngineCommit:     *engineCommit,
			ModelID:          evaluator.ModelID(),
			CheckpointSHA256: evaluator.CheckpointSHA256(),
		})
		generationDuration += time.Since(generationStarted)
		if generateErr != nil {
			fatal(generateErr)
		}

		writeStarted := time.Now()
		for _, trajectory := range trajectories {
			ref, writeErr := az.WriteTrajectoryShard(*output, trajectory)
			if writeErr != nil {
				fatal(writeErr)
			}
			shardsWritten++
			decisions += trajectory.PlyCount
			trajectoryBytes += ref.Bytes
		}
		writeDuration += time.Since(writeStarted)
	}
	wallDuration := time.Since(started)
	searchStats := telemetry.Snapshot()
	inferenceStats := evaluator.Stats()
	reuseDenominator := searchStats.EvaluationCacheHits + searchStats.EvaluationCacheMisses + searchStats.EvaluationBatchDedupHits
	metrics := selfPlayMetrics{
		WallSeconds: wallDuration.Seconds(), GenerationSeconds: generationDuration.Seconds(),
		ShardWriteSeconds: writeDuration.Seconds(), GamesPerHour: perSecond(shardsWritten, wallDuration) * 3600,
		Decisions: decisions, DecisionsPerGame: ratio(decisions, shardsWritten),
		ConfiguredSimulations:             *simulations,
		SetupSeedStart:                    *seed,
		SearchSeedStart:                   *searchSeed,
		SimulationTraversalsPerSecond:     perSecond64(searchStats.SimulationTraversals, generationDuration),
		EvaluationPositionsPerSecond:      perSecond64(searchStats.EvaluationPositions, generationDuration),
		InferencePositionsPerSecond:       perSecond64(inferenceStats.Positions, inferenceStats.ServiceDuration),
		MeanInferenceBatchSize:            ratio64(inferenceStats.Positions, inferenceStats.Batches),
		SearchInclusiveWallSeconds:        searchStats.WallDuration.Seconds(),
		InferenceCumulativeLatencySeconds: inferenceStats.TotalDuration.Seconds(),
		InferenceServiceSeconds:           inferenceStats.ServiceDuration.Seconds(),
		InferenceQueueWaitSeconds:         inferenceStats.QueueWaitDuration.Seconds(),
		RecentInferenceLatencyP50Millis:   durationMillis(inferenceStats.LatencyP50),
		RecentInferenceLatencyP95Millis:   durationMillis(inferenceStats.LatencyP95),
		RecentInferenceQueueP95Millis:     durationMillis(inferenceStats.QueueWaitP95),
		EvaluationReuseRate:               ratio64(searchStats.EvaluationCacheHits+searchStats.EvaluationBatchDedupHits, reuseDenominator),
		TrajectoryBytes:                   trajectoryBytes, BytesPerGame: ratio(trajectoryBytes, shardsWritten),
		GameBatches: batchCount, PeakConcurrentGames: peakConcurrentGames,
		Search: searchStats, Inference: inferenceStats, InferenceModel: evaluator.Metadata(),
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Games         int             `json:"games"`
		ShardsWritten int             `json:"shards_written"`
		ModelID       string          `json:"model_id"`
		Metrics       selfPlayMetrics `json:"metrics"`
	}{Games: shardsWritten, ShardsWritten: shardsWritten, ModelID: evaluator.ModelID(), Metrics: metrics}); err != nil {
		fatal(err)
	}
}

func perSecond(count int, duration time.Duration) float64 {
	return perSecond64(int64(count), duration)
}

func durationMillis(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func perSecond64(count int64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	return float64(count) / duration.Seconds()
}

func ratio(numerator, denominator int) float64 {
	return ratio64(int64(numerator), int64(denominator))
}

func ratio64(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func gameBatchEnd(start, total, limit int) int {
	end := start + limit
	if end > total {
		return total
	}
	return end
}

func selfPlaySeeds(setupSeedStart, searchSeedStart int64, gameIndex int) (int64, int64) {
	return setupSeedStart + int64(gameIndex), searchSeedStart + int64(gameIndex)
}

func orderedBaseMatchups() [][2]models.FactionType {
	matchups := make([][2]models.FactionType, 0, len(baseFactions)*(len(baseFactions)-2))
	// Diagonal round-robin order keeps every small actor batch faction-diverse,
	// while the complete cycle still contains every legal ordered pairing once.
	for offset := 1; offset < len(baseFactions); offset++ {
		for firstIndex, first := range baseFactions {
			second := baseFactions[(firstIndex+offset)%len(baseFactions)]
			if first == second || factions.NewFaction(first).GetHomeTerrain() == factions.NewFaction(second).GetHomeTerrain() {
				continue
			}
			matchups = append(matchups, [2]models.FactionType{first, second})
		}
	}
	return matchups
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
