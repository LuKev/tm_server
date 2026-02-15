package replay

import (
	"os"
	"path/filepath"
	"testing"
	"strings"

	"github.com/lukev/tm_server/internal/notation"
)

func loadSnellmanBatchFixture(t *testing.T, filename string) []notation.LogItem {
	t.Helper()

	fixturePath := filepath.Join("testdata", "snellman_batch", filename)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	concise, err := notation.ConvertSnellmanToConciseForReplay(string(content))
	if err != nil {
		t.Fatalf("ConvertSnellmanToConciseForReplay: %v", err)
	}

	items, err := notation.ParseConciseLogStrict(concise)
	if err != nil {
		t.Fatalf("parse concise: %v", err)
	}
	return items
}

func TestFullSnellmanS69Replay_Completes(t *testing.T) {
	items := loadSnellmanBatchFixture(t, "4pLeague_S69_D1L1_G3.txt")
	logStrings, logLocations := notation.GenerateConciseLog(items)

	initialState := createInitialState(items)
	sim := NewGameSimulator(initialState, items)

	for sim.CurrentIndex < len(items) {
		if err := sim.StepForward(); err != nil {
			token := "<unknown>"
			if sim.CurrentIndex >= 0 && sim.CurrentIndex < len(logLocations) {
				loc := logLocations[sim.CurrentIndex]
				if loc.LineIndex >= 0 && loc.LineIndex < len(logStrings) && loc.ColumnIndex >= 0 {
					cols := strings.Split(logStrings[loc.LineIndex], "|")
					if loc.ColumnIndex < len(cols) {
						token = strings.TrimSpace(cols[loc.ColumnIndex])
					}
				}
			}
			t.Fatalf("step %d/%d failed at token %q: %v", sim.CurrentIndex, len(items), token, err)
		}
	}
}

func TestFullSnellmanS69Replay_FinalScoresMatchSnellman(t *testing.T) {
	items := loadSnellmanBatchFixture(t, "4pLeague_S69_D1L1_G3.txt")
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
