package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/replay"
)

func main() {
	input := flag.String("input", "", "replay text input path")
	inputDir := flag.String("input_dir", "", "directory of replay text files to import")
	pattern := flag.String("pattern", "*.txt", "filename glob for -input_dir")
	output := flag.String("output", "", "snapshot seed JSONL output path; stdout when empty")
	summaryPath := flag.String("summary", "", "optional seed coverage summary JSON output path")
	format := flag.String("format", "auto", "replay format: auto, bga, snellman, concise")
	gameID := flag.String("game_id", "az_replay_seed", "temporary replay game ID")
	every := flag.Int("every", 20, "emit one seed every N replay actions")
	maxSeeds := flag.Int("max", 200, "maximum seeds to emit")
	maxPerReplay := flag.Int("max_per_replay", 0, "maximum seeds per replay file; 0 disables the per-replay cap")
	phaseFilter := flag.String("phase", "", "optional phase filter, e.g. Action, Income, Cleanup")
	rootFactionFilter := flag.String("root_faction", "", "optional root/current faction filter")
	playerCountFilter := flag.Int("player_count", 0, "optional exact player-count filter; 0 disables")
	minRound := flag.Int("min_round", 0, "optional minimum round filter; 0 disables")
	maxRound := flag.Int("max_round", 0, "optional maximum round filter; 0 disables")
	scriptDir := flag.String("script_dir", "", "script/import scratch directory; defaults to temp dir")
	flag.Parse()

	if (*input == "" && *inputDir == "") || (*input != "" && *inputDir != "") {
		exitf("provide exactly one of -input or -input_dir")
	}
	if *every <= 0 {
		exitf("-every must be positive")
	}
	if *maxSeeds <= 0 {
		exitf("-max must be positive")
	}
	if *maxPerReplay < 0 {
		exitf("-max_per_replay must be non-negative")
	}
	if *playerCountFilter < 0 {
		exitf("-player_count must be non-negative")
	}
	if *minRound < 0 || *maxRound < 0 {
		exitf("-min_round and -max_round must be non-negative")
	}
	if *minRound > 0 && *maxRound > 0 && *minRound > *maxRound {
		exitf("-min_round cannot exceed -max_round")
	}
	filter := seedFilter{
		Phase:       *phaseFilter,
		RootFaction: *rootFactionFilter,
		PlayerCount: *playerCountFilter,
		MinRound:    *minRound,
		MaxRound:    *maxRound,
	}
	inputs, err := replayInputs(*input, *inputDir, *pattern)
	if err != nil {
		exitf("resolve inputs: %v", err)
	}
	if len(inputs) == 0 {
		exitf("no replay inputs matched")
	}
	dir := *scriptDir
	if dir == "" {
		dir, err = os.MkdirTemp("", "tm_az_replay_seeds_*")
		if err != nil {
			exitf("create temp dir: %v", err)
		}
		defer os.RemoveAll(dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		exitf("create script dir: %v", err)
	}
	writer := os.Stdout
	if *output != "" {
		if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
			exitf("create output dir: %v", err)
		}
		file, err := os.Create(*output)
		if err != nil {
			exitf("create output: %v", err)
		}
		defer file.Close()
		writer = file
	}
	encoder := json.NewEncoder(writer)
	summary := newSeedSummary(inputs, *every, *maxSeeds, *maxPerReplay, filter)
	totalWritten := 0
	for index, path := range inputs {
		remaining := *maxSeeds - totalWritten
		if remaining <= 0 {
			break
		}
		perReplayLimit := remaining
		if *maxPerReplay > 0 && *maxPerReplay < perReplayLimit {
			perReplayLimit = *maxPerReplay
		}
		baseGameID := strings.TrimSpace(*gameID)
		if len(inputs) > 1 {
			baseGameID = fmt.Sprintf("%s_%04d_%s", baseGameID, index+1, sanitizeName(filepath.Base(path)))
		}
		written, err := emitReplaySeeds(encoder, summary, filter, dir, path, *format, baseGameID, *every, perReplayLimit)
		if err != nil {
			exitf("%s: %v", path, err)
		}
		totalWritten += written
	}
	if *summaryPath != "" {
		if err := writeJSON(*summaryPath, summary); err != nil {
			exitf("write summary: %v", err)
		}
	}
	_, _ = fmt.Fprintf(os.Stderr, "wrote %d snapshot seeds from %d replay inputs\n", totalWritten, len(inputs))
}

type seedSummary struct {
	Version          int            `json:"version"`
	InputCount       int            `json:"inputCount"`
	RequestedMax     int            `json:"requestedMax"`
	MaxPerReplay     int            `json:"maxPerReplay,omitempty"`
	Every            int            `json:"every"`
	Filter           seedFilter     `json:"filter,omitempty"`
	Seeds            int            `json:"seeds"`
	Skipped          int            `json:"skipped"`
	SkippedByReason  map[string]int `json:"skippedByReason,omitempty"`
	BySource         map[string]int `json:"bySource"`
	ByRound          map[string]int `json:"byRound"`
	ByPhase          map[string]int `json:"byPhase"`
	ByPlayerCount    map[string]int `json:"byPlayerCount"`
	ByRootFaction    map[string]int `json:"byRootFaction"`
	ByFactionPresent map[string]int `json:"byFactionPresent"`
}

type seedFilter struct {
	Phase       string `json:"phase,omitempty"`
	RootFaction string `json:"rootFaction,omitempty"`
	PlayerCount int    `json:"playerCount,omitempty"`
	MinRound    int    `json:"minRound,omitempty"`
	MaxRound    int    `json:"maxRound,omitempty"`
}

func newSeedSummary(inputs []string, every, maxSeeds, maxPerReplay int, filter seedFilter) *seedSummary {
	return &seedSummary{
		Version:          1,
		InputCount:       len(inputs),
		RequestedMax:     maxSeeds,
		MaxPerReplay:     maxPerReplay,
		Every:            every,
		Filter:           filter.normalized(),
		SkippedByReason:  make(map[string]int),
		BySource:         make(map[string]int),
		ByRound:          make(map[string]int),
		ByPhase:          make(map[string]int),
		ByPlayerCount:    make(map[string]int),
		ByRootFaction:    make(map[string]int),
		ByFactionPresent: make(map[string]int),
	}
}

func replayInputs(input, inputDir, pattern string) ([]string, error) {
	if input != "" {
		return []string{input}, nil
	}
	var inputs []string
	err := filepath.WalkDir(inputDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			inputs = append(inputs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(inputs)
	return inputs, nil
}

func emitReplaySeeds(encoder *json.Encoder, summary *seedSummary, filter seedFilter, scriptDir, inputPath, format, gameID string, every, maxSeeds int) (int, error) {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return 0, fmt.Errorf("read input: %w", err)
	}
	manager := replay.NewReplayManager(scriptDir)
	if err := manager.ImportText(gameID, string(raw), format); err != nil {
		return 0, fmt.Errorf("import replay: %w", err)
	}
	session := manager.GetSession(gameID)
	if session == nil || session.Simulator == nil {
		return 0, fmt.Errorf("replay session missing")
	}
	written := 0
	for session.Simulator.CurrentIndex < len(session.Simulator.Actions) && written < maxSeeds {
		if err := session.Simulator.StepForward(); err != nil {
			return written, fmt.Errorf("step replay at index %d: %w", session.Simulator.CurrentIndex, err)
		}
		if session.Simulator.CurrentIndex%every != 0 {
			continue
		}
		gs := session.Simulator.GetState()
		if gs == nil || gs.IsGameOver() {
			continue
		}
		root := currentOrFirstPlayerID(gs)
		if root == "" {
			continue
		}
		seed := env.SnapshotSeed{
			Name:         fmt.Sprintf("%s_%04d", strings.TrimSpace(gameID), session.Simulator.CurrentIndex),
			RootPlayerID: root,
			Snapshot:     replay.GenerateSnapshot(gs),
			Source:       inputPath,
			ActionIndex:  session.Simulator.CurrentIndex,
			Round:        gs.Round,
			Phase:        phaseName(gs.Phase),
			PlayerCount:  len(gs.Players),
			RootFaction:  factionName(gs, root),
			Factions:     factionNames(gs),
		}
		if ok, reason := filter.matches(seed); !ok {
			recordSkipped(summary, reason)
			continue
		}
		if err := encoder.Encode(seed); err != nil {
			return written, fmt.Errorf("write seed: %w", err)
		}
		recordSummary(summary, seed)
		written++
	}
	return written, nil
}

func (filter seedFilter) normalized() seedFilter {
	filter.Phase = strings.TrimSpace(filter.Phase)
	filter.RootFaction = strings.TrimSpace(filter.RootFaction)
	return filter
}

func (filter seedFilter) matches(seed env.SnapshotSeed) (bool, string) {
	filter = filter.normalized()
	if filter.Phase != "" && !strings.EqualFold(seed.Phase, filter.Phase) {
		return false, "phase"
	}
	if filter.RootFaction != "" && !strings.EqualFold(seed.RootFaction, filter.RootFaction) {
		return false, "rootFaction"
	}
	if filter.PlayerCount > 0 && seed.PlayerCount != filter.PlayerCount {
		return false, "playerCount"
	}
	if filter.MinRound > 0 && seed.Round < filter.MinRound {
		return false, "minRound"
	}
	if filter.MaxRound > 0 && seed.Round > filter.MaxRound {
		return false, "maxRound"
	}
	return true, ""
}

func recordSummary(summary *seedSummary, seed env.SnapshotSeed) {
	if summary == nil {
		return
	}
	summary.Seeds++
	increment(summary.BySource, seed.Source)
	increment(summary.ByRound, fmt.Sprintf("%d", seed.Round))
	increment(summary.ByPhase, seed.Phase)
	increment(summary.ByPlayerCount, fmt.Sprintf("%d", seed.PlayerCount))
	increment(summary.ByRootFaction, seed.RootFaction)
	for _, faction := range seed.Factions {
		increment(summary.ByFactionPresent, faction)
	}
}

func recordSkipped(summary *seedSummary, reason string) {
	if summary == nil {
		return
	}
	summary.Skipped++
	increment(summary.SkippedByReason, reason)
}

func increment(counts map[string]int, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	counts[key]++
}

func factionName(gs *game.GameState, playerID string) string {
	if gs == nil {
		return ""
	}
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		return ""
	}
	return player.Faction.GetType().String()
}

func factionNames(gs *game.GameState) []string {
	if gs == nil {
		return nil
	}
	ids := make([]string, 0, len(gs.Players))
	for id := range gs.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name := factionName(gs, id); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func phaseName(phase game.GamePhase) string {
	switch phase {
	case game.PhaseSetup:
		return "Setup"
	case game.PhaseFactionSelection:
		return "FactionSelection"
	case game.PhaseIncome:
		return "Income"
	case game.PhaseAction:
		return "Action"
	case game.PhaseCleanup:
		return "Cleanup"
	case game.PhaseEnd:
		return "End"
	default:
		return "Unknown"
	}
}

func sanitizeName(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var builder strings.Builder
	for _, ch := range name {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
		} else {
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func currentOrFirstPlayerID(gs *game.GameState) string {
	if gs == nil {
		return ""
	}
	if current := gs.GetCurrentPlayer(); current != nil {
		return current.ID
	}
	for id := range gs.Players {
		return id
	}
	return ""
}

func writeJSON(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
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
