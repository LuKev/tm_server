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
	format := flag.String("format", "auto", "replay format: auto, bga, snellman, concise")
	gameID := flag.String("game_id", "az_replay_seed", "temporary replay game ID")
	every := flag.Int("every", 20, "emit one seed every N replay actions")
	maxSeeds := flag.Int("max", 200, "maximum seeds to emit")
	maxPerReplay := flag.Int("max_per_replay", 0, "maximum seeds per replay file; 0 disables the per-replay cap")
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
		written, err := emitReplaySeeds(encoder, dir, path, *format, baseGameID, *every, perReplayLimit)
		if err != nil {
			exitf("%s: %v", path, err)
		}
		totalWritten += written
	}
	_, _ = fmt.Fprintf(os.Stderr, "wrote %d snapshot seeds from %d replay inputs\n", totalWritten, len(inputs))
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

func emitReplaySeeds(encoder *json.Encoder, scriptDir, inputPath, format, gameID string, every, maxSeeds int) (int, error) {
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
		}
		if err := encoder.Encode(seed); err != nil {
			return written, fmt.Errorf("write seed: %w", err)
		}
		written++
	}
	return written, nil
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

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
