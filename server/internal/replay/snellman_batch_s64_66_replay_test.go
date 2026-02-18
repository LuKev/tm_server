package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnellmanBatchReplayS64to66_FinalScoresMatch(t *testing.T) {
	manifestPath := filepath.Join("testdata", "snellman_batch_s64_66", "manifest.json")
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
			logPath := filepath.Join("testdata", "snellman_batch_s64_66", tc.LogFile)
			logBytes, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("read log fixture %s: %v", tc.LogFile, err)
			}
			if strings.Contains(strings.ToLower(string(logBytes)), "dropped from the game") {
				t.Skipf("skipping dropout game fixture %s", tc.LogFile)
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
				token := describeFailingToken(session, failingIndex)
				item := describeFailingItem(session, failingIndex)
				context := describeFailingContext(session, failingIndex, 20)
				t.Fatalf("JumpTo(%d) failed at index %d token %s item %s: %v\ncontext:\n%s", totalActions, failingIndex, token, item, err, context)
			}

			state := session.Simulator.GetState()
			if state == nil {
				t.Fatalf("state is nil after replay")
			}
			if state.FinalScoring == nil {
				t.Fatalf("final scoring is nil")
			}

			for expectedPlayer, expectedTotal := range tc.ExpectedTotalVP {
				var matchedTotal *int
				for actualPlayer, score := range state.FinalScoring {
					if normalizePlayerKey(actualPlayer) == normalizePlayerKey(expectedPlayer) {
						if score != nil {
							v := score.TotalVP
							matchedTotal = &v
						}
						break
					}
				}
				if matchedTotal == nil {
					t.Fatalf("missing final scoring entry for %q", expectedPlayer)
				}
				if *matchedTotal != expectedTotal {
					t.Fatalf("%s final total VP mismatch: got %d, want %d", expectedPlayer, *matchedTotal, expectedTotal)
				}
			}
		})
	}
}
