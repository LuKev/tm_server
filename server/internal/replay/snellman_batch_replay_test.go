package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

type snellmanBatchManifest struct {
	Games []snellmanBatchGame `json:"games"`
}

type snellmanBatchGame struct {
	GameID          string         `json:"game_id"`
	LogFile         string         `json:"log_file"`
	ExpectedTotalVP map[string]int `json:"expected_total_vp"`
}

func TestSnellmanBatchReplay_FinalScoresMatch(t *testing.T) {
	manifestPath := filepath.Join("testdata", "snellman_batch", "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest snellmanBatchManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifest json: %v", err)
	}
	if len(manifest.Games) != 21 {
		t.Fatalf("unexpected game count in manifest: got %d, want 21", len(manifest.Games))
	}

	manager := NewReplayManager(t.TempDir())

	for _, tc := range manifest.Games {
		tc := tc
		t.Run(tc.GameID, func(t *testing.T) {
			logPath := filepath.Join("testdata", "snellman_batch", tc.LogFile)
			logBytes, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("read log fixture %s: %v", tc.LogFile, err)
			}

			if err := manager.ImportText(tc.GameID, string(logBytes), "snellman"); err != nil {
				t.Fatalf("ImportText failed: %v", err)
			}

			session, err := manager.StartReplay(tc.GameID, true)
			if err != nil {
				t.Fatalf("StartReplay failed: %v", err)
			}

			totalActions := len(session.Simulator.Actions)
			if err := manager.JumpTo(tc.GameID, totalActions); err != nil {
				failingIndex := session.Simulator.CurrentIndex
				var logToken string
				if failingIndex >= 0 && failingIndex < len(session.LogStrings) {
					logToken = session.LogStrings[failingIndex]
				}
				t.Fatalf("JumpTo(%d) failed at index %d token %q: %v", totalActions, failingIndex, logToken, err)
			}

			state := session.Simulator.GetState()
			if state == nil {
				t.Fatalf("state is nil after replay")
			}
			if state.Phase != game.PhaseEnd {
				t.Fatalf("expected phase end (%v), got %v", game.PhaseEnd, state.Phase)
			}
			if state.FinalScoring == nil {
				t.Fatalf("final scoring is nil")
			}

			for expectedPlayer, expectedTotal := range tc.ExpectedTotalVP {
				var matched *game.PlayerFinalScore
				for actualPlayer, score := range state.FinalScoring {
					if strings.EqualFold(actualPlayer, expectedPlayer) {
						matched = score
						break
					}
				}
				if matched == nil {
					t.Fatalf("missing final scoring entry for %q", expectedPlayer)
				}
				if matched.TotalVP != expectedTotal {
					t.Fatalf("%s final total VP mismatch: got %d, want %d", expectedPlayer, matched.TotalVP, expectedTotal)
				}
			}
		})
	}
}
