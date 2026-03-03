package replay

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/notation"
)

func TestDebugReplayActionStateWindow(t *testing.T) {
	fixture := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_FIXTURE"))
	if fixture == "" {
		t.Skip("set TM_DEBUG_REPLAY_FIXTURE to print replay pre-action state window")
	}

	fixturePath := fixture
	if !filepath.IsAbs(fixturePath) {
		fixturePath = filepath.Join("testdata", "snellman_batch", fixture)
	}
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}

	manager := NewReplayManager(t.TempDir())
	if raw := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_SOURCE_ORDERING")); raw != "" {
		manager.SetSourceAnchoredLeechOrdering(strings.EqualFold(raw, "1") || strings.EqualFold(raw, "true"))
	}
	if err := manager.ImportText("debug_replay_window", string(raw), "snellman"); err != nil {
		t.Fatalf("import fixture: %v", err)
	}
	session, err := manager.StartReplay("debug_replay_window", true)
	if err != nil {
		t.Fatalf("start replay: %v", err)
	}

	start := 0
	if rawStart := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_START")); rawStart != "" {
		if v, convErr := strconv.Atoi(rawStart); convErr == nil && v >= 0 {
			start = v
		}
	}
	end := len(session.Simulator.Actions)
	if rawEnd := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_END")); rawEnd != "" {
		if v, convErr := strconv.Atoi(rawEnd); convErr == nil && v >= 0 && v <= len(session.Simulator.Actions) {
			end = v
		}
	}
	if start > end {
		start, end = end, start
	}
	actionStart := -1
	if raw := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_ACTION_START")); raw != "" {
		if v, convErr := strconv.Atoi(raw); convErr == nil && v >= 0 {
			actionStart = v
		}
	}
	actionEnd := -1
	if raw := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_ACTION_END")); raw != "" {
		if v, convErr := strconv.Atoi(raw); convErr == nil && v >= 0 {
			actionEnd = v
		}
	}
	playerFilter := strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_PLAYER"))
	dumpAll := strings.EqualFold(strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_DUMP_ALL")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("TM_DEBUG_REPLAY_DUMP_ALL")), "true")
	actionOrdinal := 0

	sim := session.Simulator
	for sim.CurrentIndex < len(sim.Actions) {
		idx := sim.CurrentIndex
		item := sim.Actions[idx]
		actionItem, ok := item.(notation.ActionItem)
		if !ok || actionItem.Action == nil {
			if err := sim.StepForward(); err != nil {
				t.Fatalf("step forward idx=%d non-action item failed: %v", idx, err)
			}
			continue
		}

		playerID := strings.TrimSpace(actionItem.Action.GetPlayerID())
		include := idx >= start && idx < end
		if dumpAll {
			include = true
		}
		if actionStart >= 0 && actionOrdinal < actionStart {
			include = false
		}
		if actionEnd >= 0 && actionOrdinal >= actionEnd {
			include = false
		}
		if include {
			if playerFilter == "" || strings.EqualFold(playerFilter, playerID) {
				player := sim.CurrentState.GetPlayer(playerID)
				if player == nil {
					t.Logf("idx=%d actionIdx=%d player=%s action=%s pre=(missing player)", idx, actionOrdinal, playerID, describeAction(actionItem.Action))
				} else {
					t.Logf(
						"idx=%d actionIdx=%d player=%s action=%s pre=vp=%d c=%d w=%d p=%d pw=%d/%d/%d cult=%d/%d/%d/%d",
						idx,
						actionOrdinal,
						playerID,
						describeAction(actionItem.Action),
						player.VictoryPoints,
						player.Resources.Coins,
						player.Resources.Workers,
						player.Resources.Priests,
						player.Resources.Power.Bowl1,
						player.Resources.Power.Bowl2,
						player.Resources.Power.Bowl3,
						player.CultPositions[0],
						player.CultPositions[1],
						player.CultPositions[2],
						player.CultPositions[3],
					)
				}
			}
		}
		actionOrdinal++

		if err := sim.StepForward(); err != nil {
			t.Fatalf("step forward idx=%d action=%T: %v", idx, actionItem.Action, err)
		}
	}
}
