package az

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

func TestTrajectoryShardIsContentAddressedAndRoundTrips(t *testing.T) {
	directory := t.TempDir()
	trajectory := testTrajectory(t)
	first, err := WriteTrajectoryShard(directory, trajectory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := WriteTrajectoryShard(directory, trajectory)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || filepath.Base(first.Path) != "trajectory-"+first.SHA256+".json.gz" {
		t.Fatalf("content-addressed write was not stable: first=%+v second=%+v", first, second)
	}
	info, err := os.Stat(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o222 != 0 {
		t.Fatalf("shard is writable: mode=%o", info.Mode().Perm())
	}
	loaded, ref, err := ReadTrajectoryShard(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	if ref != first || loaded.PlyCount != 1 || loaded.Steps[0].Actions[0].Kind != trajectory.Steps[0].Actions[0].Kind {
		t.Fatalf("unexpected trajectory round trip: ref=%+v trajectory=%+v", ref, loaded)
	}
}

func TestTrajectoryShardRejectsInvalidAndMislabeledData(t *testing.T) {
	invalid := testTrajectory(t)
	invalid.Steps[0].Visits[0] = 0
	if _, err := WriteTrajectoryShard(t.TempDir(), invalid); err == nil || !strings.Contains(err.Error(), "no searched chosen action") {
		t.Fatalf("expected zero-visit rejection, got %v", err)
	}

	directory := t.TempDir()
	ref, err := WriteTrajectoryShard(directory, testTrajectory(t))
	if err != nil {
		t.Fatal(err)
	}
	mislabeled := filepath.Join(directory, "trajectory-"+strings.Repeat("0", 64)+".json.gz")
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mislabeled, data, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadTrajectoryShard(mislabeled); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("expected filename hash rejection, got %v", err)
	}
}

func TestTrajectoryValidationRejectsSemanticCorruption(t *testing.T) {
	tests := map[string]struct {
		mutate func(*Trajectory)
		want   string
	}{
		"state hash":          {func(value *Trajectory) { value.Steps[0].StateHash = strings.Repeat("0", 32) }, "state hash mismatch"},
		"hash chain":          {func(value *Trajectory) { value.Steps[0].NextStateHash = strings.Repeat("0", 32) }, "hash chain"},
		"seat faction":        {func(value *Trajectory) { value.Steps[0].DecisionSeat = 1 }, "decision seat"},
		"result":              {func(value *Trajectory) { value.Steps[0].FinalVP++ }, "inconsistent with final VP"},
		"negative visits":     {func(value *Trajectory) { value.Steps[0].Visits[1] = -1 }, "negative visits"},
		"engine identity":     {func(value *Trajectory) { value.Manifest.EngineCommit = "" }, "engine commit"},
		"model identity":      {func(value *Trajectory) { value.Manifest.ModelID = "" }, "model ID"},
		"search identity":     {func(value *Trajectory) { value.Manifest.SearchSeed = 0 }, "search seed"},
		"checkpoint identity": {func(value *Trajectory) { value.Manifest.ModelID = "model-" + strings.Repeat("a", 64) }, "checkpoint hash"},
		"duplicate action":    {func(value *Trajectory) { value.Steps[0].Actions[1] = value.Steps[0].Actions[0] }, "duplicate actions"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			trajectory := testTrajectory(t)
			test.mutate(&trajectory)
			if err := validateTrajectory(trajectory); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want error containing %q", err, test.want)
			}
		})
	}
}

func testTrajectory(t *testing.T) Trajectory {
	t.Helper()
	position, err := NewBaseGame(7, models.FactionNomads, models.FactionWitches)
	if err != nil {
		t.Fatal(err)
	}
	state, err := position.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(state)
	actions := position.LegalActions()
	if len(actions) == 0 {
		t.Fatal("test position has no legal actions")
	}
	terminalGame := position.StateClone()
	terminalGame.Phase = game.PhaseEnd
	terminal, err := NewPosition(terminalGame)
	if err != nil {
		t.Fatal(err)
	}
	terminalState, err := terminal.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	finalVP := [2]int{}
	for seat, id := range []string{"p0", "p1"} {
		finalVP[seat], _ = terminal.FinalVP(PlayerID(id))
	}
	margin := finalVP[0] - finalVP[1]
	outcome := float32(0)
	if margin > 0 {
		outcome = 1
	} else if margin < 0 {
		outcome = -1
	}
	return Trajectory{
		Manifest: TrajectoryManifest{
			FormatVersion: TrajectoryFormatVersion,
			RulesVersion:  1,
			StateVersion:  StateSchemaVersion,
			ActionVersion: ActionSchemaVersion,
			EngineCommit:  "test",
			ModelID:       "random",
			Seed:          7,
			SearchSeed:    17,
			Factions:      [2]models.FactionType{models.FactionNomads, models.FactionWitches},
		},
		Steps: []TrajectoryStep{{
			State:         state,
			StateHash:     hex.EncodeToString(digest[:16]),
			Actions:       actions,
			Visits:        append([]int{1}, make([]int, len(actions)-1)...),
			ChosenIndex:   0,
			DecisionSeat:  0,
			NextStateHash: terminal.CanonicalHash().String(),
			Outcome:       outcome,
			FinalVP:       finalVP[0],
			VPMargin:      margin,
		}},
		FinalVP:           finalVP,
		PlyCount:          1,
		Completed:         true,
		TerminalState:     terminalState,
		TerminalStateHash: terminal.CanonicalHash().String(),
	}
}
