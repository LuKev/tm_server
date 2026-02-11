package replay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukev/tm_server/internal/notation"
)

func TestFullSnellmanS69Replay_Completes(t *testing.T) {
	fixturePath := filepath.Join("testdata", "s69_full_game_concise.txt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	items, err := notation.ParseConciseLogStrict(string(content))
	if err != nil {
		t.Fatalf("parse concise fixture: %v", err)
	}

	initialState := createInitialState(items)
	sim := NewGameSimulator(initialState, items)

	for sim.CurrentIndex < len(items) {
		if err := sim.StepForward(); err != nil {
			t.Fatalf("step %d/%d failed: %v", sim.CurrentIndex, len(items), err)
		}
	}
}

func TestFullSnellmanS69Replay_FinalScoresMatchSnellman(t *testing.T) {
	fixturePath := filepath.Join("testdata", "s69_full_game_concise.txt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	items, err := notation.ParseConciseLogStrict(string(content))
	if err != nil {
		t.Fatalf("parse concise fixture: %v", err)
	}

	initialState := createInitialState(items)
	sim := NewGameSimulator(initialState, items)
	for sim.CurrentIndex < len(items) {
		if err := sim.StepForward(); err != nil {
			t.Fatalf("step %d/%d failed: %v", sim.CurrentIndex, len(items), err)
		}
	}

	want := map[string]int{
		"Cultists":  148,
		"Engineers": 147,
		"Darklings": 143,
		"Witches":   119,
	}
	scores := sim.GetState().CalculateFinalScoring()
	for faction := range want {
		if score, ok := scores[faction]; ok && score != nil {
			t.Logf("final total vp %s=%d (base=%d area=%d cult=%d resource=%d)", faction, score.TotalVP, score.BaseVP, score.AreaVP, score.CultVP, score.ResourceVP)
		}
	}
	for faction, expectedVP := range want {
		score := scores[faction]
		if score == nil {
			t.Fatalf("missing final score for player: %s", faction)
		}
		if score.TotalVP != expectedVP {
			t.Fatalf("%s final VP mismatch: got %d, want %d", faction, score.TotalVP, expectedVP)
		}
	}
}
