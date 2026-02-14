package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/notation"
)

type snellmanBatchManifest struct {
	Games []snellmanBatchGame `json:"games"`
}

type snellmanBatchGame struct {
	GameID          string         `json:"game_id"`
	LogFile         string         `json:"log_file"`
	ExpectedTotalVP map[string]int `json:"expected_total_vp"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func normalizePlayerKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return nonAlnum.ReplaceAllString(s, "")
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
				token := describeFailingToken(session, failingIndex)
				item := describeFailingItem(session, failingIndex)
				t.Fatalf("JumpTo(%d) failed at index %d token %s item %s: %v", totalActions, failingIndex, token, item, err)
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
					if normalizePlayerKey(actualPlayer) == normalizePlayerKey(expectedPlayer) {
						matched = score
						break
					}
				}
				if matched == nil {
					t.Fatalf("missing final scoring entry for %q", expectedPlayer)
				}
				if matched.TotalVP != expectedTotal {
					t.Fatalf("%s final total VP mismatch: got %d, want %d (score=%+v)", expectedPlayer, matched.TotalVP, expectedTotal, *matched)
				}
			}
		})
	}
}

func describeFailingToken(session *ReplaySession, idx int) string {
	if session == nil || idx < 0 || idx >= len(session.LogLocations) {
		return "<unknown>"
	}
	loc := session.LogLocations[idx]
	if loc.LineIndex < 0 || loc.LineIndex >= len(session.LogStrings) || loc.ColumnIndex < 0 {
		return "<unknown>"
	}
	line := session.LogStrings[loc.LineIndex]
	cols := strings.Split(line, "|")
	if loc.ColumnIndex >= len(cols) {
		return fmt.Sprintf("<line %d col %d out of range>", loc.LineIndex, loc.ColumnIndex)
	}
	cell := strings.TrimSpace(cols[loc.ColumnIndex])
	return fmt.Sprintf("%q @line=%d col=%d", cell, loc.LineIndex+1, loc.ColumnIndex+1)
}

func describeFailingItem(session *ReplaySession, idx int) string {
	if session == nil || session.Simulator == nil || idx < 0 || idx >= len(session.Simulator.Actions) {
		return "<unknown>"
	}
	item := session.Simulator.Actions[idx]
	actionItem, ok := item.(notation.ActionItem)
	if !ok || actionItem.Action == nil {
		return fmt.Sprintf("%T", item)
	}
	return describeAction(actionItem.Action)
}

func describeAction(a game.Action) string {
	switch v := a.(type) {
	case *notation.LogCompoundAction:
		var parts []string
		for _, sub := range v.Actions {
			parts = append(parts, describeAction(sub))
		}
		return fmt.Sprintf("compound[%s]", strings.Join(parts, ", "))
	case *notation.LogConversionAction:
		return fmt.Sprintf("convert(cost=%v reward=%v)", v.Cost, v.Reward)
	case *notation.LogBurnAction:
		return fmt.Sprintf("burn(%d)", v.Amount)
	case *notation.LogFavorTileAction:
		return fmt.Sprintf("fav(%s)", v.Tile)
	case *notation.LogPowerAction:
		return fmt.Sprintf("power(%s)", v.ActionCode)
	default:
		return fmt.Sprintf("%T", a)
	}
}
