package az

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/lukev/tm_server/internal/models"
)

const TrajectoryFormatVersion = 2

type SelfPlayConfig struct {
	Search           SearchConfig
	SearchSeeds      []int64
	Temperature      float64
	MaxPlies         int
	EngineCommit     string
	ModelID          string
	CheckpointSHA256 string
}

type selfPlayGame struct {
	position   *GamePosition
	seed       int64
	searchSeed int64
	playerIDs  [2]string
	factions   [2]models.FactionType
	seatByID   map[string]int
	rng        *rand.Rand
	steps      []TrajectoryStep
	state      json.RawMessage
	stateHash  string
	completed  bool
}

type identifiedEvaluator interface {
	ModelID() string
}

type checkpointEvaluator interface {
	CheckpointSHA256() string
}

// GenerateSelfPlay runs games in lockstep so each search call batches one root
// from every unfinished game through the evaluator.
func GenerateSelfPlay(ctx context.Context, positions []*GamePosition, seeds []int64, evaluator Evaluator, config SelfPlayConfig) ([]Trajectory, error) {
	if len(positions) == 0 || len(positions) != len(seeds) {
		return nil, fmt.Errorf("positions and seeds must have equal nonzero length")
	}
	if len(config.SearchSeeds) != len(positions) {
		return nil, fmt.Errorf("search seeds and positions must have equal nonzero length")
	}
	for index, searchSeed := range config.SearchSeeds {
		if searchSeed == 0 {
			return nil, fmt.Errorf("game %d search seed must be nonzero", index)
		}
	}
	if config.Search.Simulations <= 0 {
		return nil, fmt.Errorf("self-play requires at least one search simulation")
	}
	if config.EngineCommit == "" {
		return nil, fmt.Errorf("self-play engine commit is required")
	}
	if identified, ok := evaluator.(identifiedEvaluator); ok {
		if config.ModelID != "" && config.ModelID != identified.ModelID() {
			return nil, fmt.Errorf("configured model ID %q does not match evaluator %q", config.ModelID, identified.ModelID())
		}
		config.ModelID = identified.ModelID()
	}
	if config.ModelID == "" {
		return nil, fmt.Errorf("self-play evaluator model ID is required")
	}
	if identified, ok := evaluator.(checkpointEvaluator); ok {
		if config.CheckpointSHA256 != "" && config.CheckpointSHA256 != identified.CheckpointSHA256() {
			return nil, fmt.Errorf("configured checkpoint hash does not match evaluator")
		}
		config.CheckpointSHA256 = identified.CheckpointSHA256()
	}
	if config.MaxPlies <= 0 {
		config.MaxPlies = 2000
	}
	games := make([]selfPlayGame, len(positions))
	for i, position := range positions {
		if IsHoldoutSeed(seeds[i]) {
			return nil, fmt.Errorf("seed %d is reserved for evaluation holdout", seeds[i])
		}
		state := position.StateClone()
		if state == nil || len(state.TurnOrder) != 2 {
			return nil, fmt.Errorf("game %d is not a two-player position", i)
		}
		seatByID := map[string]int{state.TurnOrder[0]: 0, state.TurnOrder[1]: 1}
		playerIDs := [2]string{state.TurnOrder[0], state.TurnOrder[1]}
		canonicalState, err := position.CanonicalJSON()
		if err != nil {
			return nil, err
		}
		games[i] = selfPlayGame{
			position: position, seed: seeds[i], searchSeed: config.SearchSeeds[i], playerIDs: playerIDs, seatByID: seatByID,
			rng: rand.New(rand.NewSource(config.SearchSeeds[i])), state: canonicalState, stateHash: canonicalStateHash(canonicalState),
			factions: [2]models.FactionType{
				state.GetPlayer(playerIDs[0]).Faction.GetType(),
				state.GetPlayer(playerIDs[1]).Faction.GetType(),
			},
		}
	}

	for ply := 0; ; ply++ {
		activePositions := make([]Position, 0, len(games))
		activeGames := make([]int, 0, len(games))
		for i := range games {
			if games[i].position.IsTerminal() {
				games[i].completed = true
				continue
			}
			if len(games[i].steps) >= config.MaxPlies {
				return nil, fmt.Errorf("game %d exceeded natural-completion tripwire at %d plies", i, config.MaxPlies)
			}
			activePositions = append(activePositions, games[i].position)
			activeGames = append(activeGames, i)
		}
		if len(activePositions) == 0 {
			break
		}
		searchConfig := config.Search
		searchConfig.RootSeeds = make([]int64, len(activeGames))
		for rootIndex, gameIndex := range activeGames {
			searchConfig.RootSeeds[rootIndex] = selfPlayRootSeed(games[gameIndex].searchSeed, ply)
		}
		search, err := NewMCTS(evaluator, searchConfig)
		if err != nil {
			return nil, err
		}
		results, err := search.SearchBatch(ctx, activePositions)
		if err != nil {
			return nil, err
		}
		for resultIndex, result := range results {
			gameIndex := activeGames[resultIndex]
			game := &games[gameIndex]
			policy, err := result.VisitPolicy(config.Temperature)
			if err != nil {
				return nil, err
			}
			action, err := result.SelectAction(config.Temperature, game.rng)
			if err != nil {
				return nil, err
			}
			chosenIndex := -1
			for i := range result.Actions {
				if result.Actions[i].Key() == action.Key() {
					chosenIndex = i
					break
				}
			}
			if chosenIndex < 0 || policy[chosenIndex] <= 0 {
				return nil, fmt.Errorf("selected action missing from root policy")
			}
			owner := string(game.position.DecisionPlayer())
			seat, ok := game.seatByID[owner]
			if !ok {
				return nil, fmt.Errorf("unknown decision player %q", owner)
			}
			game.steps = append(game.steps, TrajectoryStep{
				State: append(json.RawMessage(nil), game.state...), StateHash: game.stateHash,
				Actions: append([]SearchAction(nil), result.Actions...),
				Visits:  append([]int(nil), result.Visits...), ChosenIndex: chosenIndex,
				DecisionSeat: seat,
			})
			if err := game.position.Apply(action); err != nil {
				return nil, fmt.Errorf("game %d apply selected action: %w", gameIndex, err)
			}
			game.state, err = game.position.CanonicalJSON()
			if err != nil {
				return nil, err
			}
			game.stateHash = canonicalStateHash(game.state)
			game.steps[len(game.steps)-1].NextStateHash = game.stateHash
		}
	}

	trajectories := make([]Trajectory, len(games))
	for i := range games {
		game := &games[i]
		state := game.position.StateClone()
		if !game.completed || state == nil {
			return nil, fmt.Errorf("game %d did not complete naturally", i)
		}
		finalVP := [2]int{}
		for seat, id := range game.playerIDs {
			value, ok := game.position.FinalVP(PlayerID(id))
			if !ok {
				return nil, fmt.Errorf("missing final VP for game %d seat %d", i, seat)
			}
			finalVP[seat] = value
		}
		for step := range game.steps {
			seat := game.steps[step].DecisionSeat
			game.steps[step].FinalVP = finalVP[seat]
			game.steps[step].VPMargin = finalVP[seat] - finalVP[1-seat]
			game.steps[step].Outcome = outcomeFromMargin(game.steps[step].VPMargin)
		}
		trajectories[i] = Trajectory{
			Manifest: TrajectoryManifest{
				FormatVersion: TrajectoryFormatVersion, RulesVersion: 1,
				StateVersion: StateSchemaVersion, ActionVersion: ActionSchemaVersion,
				EngineCommit: config.EngineCommit, ModelID: config.ModelID, Seed: game.seed,
				SearchSeed:       game.searchSeed,
				CheckpointSHA256: config.CheckpointSHA256,
				Factions:         game.factions,
			},
			Steps: game.steps, FinalVP: finalVP, PlyCount: len(game.steps), Completed: true,
			TerminalState: append(json.RawMessage(nil), game.state...), TerminalStateHash: game.stateHash,
		}
	}
	return trajectories, nil
}

func selfPlayRootSeed(searchSeed int64, ply int) int64 {
	value := splitMix64(uint64(searchSeed) + 0x243f6a8885a308d3)
	value ^= rotateLeft64(splitMix64(uint64(ply)+0xa4093822299f31d0), 43)
	return int64(splitMix64(value))
}

func splitMix64(value uint64) uint64 {
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	value *= 0x94d049bb133111eb
	value ^= value >> 31
	return value
}

func rotateLeft64(value uint64, count uint) uint64 {
	return value<<count | value>>(64-count)
}

func canonicalStateHash(state []byte) string {
	digest := sha256.Sum256(state)
	return hex.EncodeToString(digest[:16])
}

func outcomeFromMargin(margin int) float32 {
	if margin > 0 {
		return 1
	}
	if margin < 0 {
		return -1
	}
	return 0
}
