package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/notation"
)

type snellmanExpectedState struct {
	Line     int
	PlayerID string
	Action   string

	VP int
	C  int
	W  int
	P  int

	PW1 int
	PW2 int
	PW3 int

	Fire  int
	Water int
	Earth int
	Air   int
}

func snellmanFactionToPlayerID(lower string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(lower)) {
	case "nomads":
		return "Nomads", true
	case "fakirs":
		return "Fakirs", true
	case "chaosmagicians":
		return "Chaos Magicians", true
	case "giants":
		return "Giants", true
	case "swarmlings":
		return "Swarmlings", true
	case "mermaids":
		return "Mermaids", true
	case "witches":
		return "Witches", true
	case "auren":
		return "Auren", true
	case "halflings":
		return "Halflings", true
	case "cultists":
		return "Cultists", true
	case "alchemists":
		return "Alchemists", true
	case "darklings":
		return "Darklings", true
	case "engineers":
		return "Engineers", true
	case "dwarves":
		return "Dwarves", true
	default:
		return "", false
	}
}

func parseSnellmanExpectedStates(content string) ([]snellmanExpectedState, error) {
	lines := strings.Split(content, "\n")
	out := make([]snellmanExpectedState, 0, len(lines))

	lastNonEmpty := func(parts []string) string {
		for i := len(parts) - 1; i >= 0; i-- {
			s := strings.TrimSpace(parts[i])
			if s != "" {
				return s
			}
		}
		return ""
	}

	reCultPos := regexp.MustCompile(`^\d+/\d+/\d+/\d+$`)
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		playerID, ok := snellmanFactionToPlayerID(parts[0])
		if !ok {
			continue
		}
		action := lastNonEmpty(parts)
		actionLower := strings.ToLower(strings.TrimSpace(action))
		if actionLower == "" || actionLower == "setup" || actionLower == "wait" {
			continue
		}
		if strings.HasPrefix(actionLower, "[") {
			// Meta rows like "[opponent accepted power]" are not replayed as actions.
			continue
		}
		if actionLower == "cult_income_for_faction" || actionLower == "other_income_for_faction" {
			continue
		}
		// End-of-game scoring rows are not replayed as actions; the server computes final
		// scoring separately. Snellman logs them as faction rows like "+8vp for FIRE".
		if actionLower == "score_resources" {
			continue
		}
		if strings.HasPrefix(actionLower, "+") && strings.Contains(actionLower, "vp for ") {
			continue
		}

		st := snellmanExpectedState{
			Line:     idx + 1,
			PlayerID: playerID,
			Action:   action,
			VP:       -1,
		}

		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.HasSuffix(p, "VP") && strings.Contains(p, " ") {
				// "64 VP"
				if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(p, "VP"))); err == nil {
					st.VP = n
				}
				continue
			}
			if strings.HasSuffix(p, " C") {
				if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(p, "C"))); err == nil {
					st.C = n
				}
				continue
			}
			if strings.HasSuffix(p, " W") {
				if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(p, "W"))); err == nil {
					st.W = n
				}
				continue
			}
			if strings.HasSuffix(p, " P") {
				if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(p, "P"))); err == nil {
					st.P = n
				}
				continue
			}
			if strings.HasSuffix(p, " PW") {
				pw := strings.TrimSpace(strings.TrimSuffix(p, "PW"))
				seg := strings.Split(pw, "/")
				if len(seg) == 3 {
					if a, err := strconv.Atoi(strings.TrimSpace(seg[0])); err == nil {
						st.PW1 = a
					}
					if b, err := strconv.Atoi(strings.TrimSpace(seg[1])); err == nil {
						st.PW2 = b
					}
					if c, err := strconv.Atoi(strings.TrimSpace(seg[2])); err == nil {
						st.PW3 = c
					}
				}
				continue
			}
			if reCultPos.MatchString(p) {
				seg := strings.Split(p, "/")
				if len(seg) == 4 {
					st.Fire, _ = strconv.Atoi(seg[0])
					st.Water, _ = strconv.Atoi(seg[1])
					st.Earth, _ = strconv.Atoi(seg[2])
					st.Air, _ = strconv.Atoi(seg[3])
				}
				continue
			}
		}
		if st.VP < 0 {
			return nil, fmt.Errorf("missing VP total on snellman line %d: %q", st.Line, line)
		}
		out = append(out, st)
	}

	return out, nil
}

func TestSnellmanLedgerResourcesMatchReplay_S67_G5(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S67_D1L1_G5.txt")
}

func TestSnellmanLedgerResourcesMatchReplay_S68_G2(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S68_D1L1_G2.txt")
}

func TestSnellmanLedgerResourcesMatchReplay_S68_G4(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S68_D1L1_G4.txt")
}

func TestSnellmanLedgerResourcesMatchReplay_S68_G7(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S68_D1L1_G7.txt")
}

func TestSnellmanLedgerResourcesMatchReplay_S69_G2(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S69_D1L1_G2.txt")
}

func TestSnellmanLedgerResourcesMatchReplay_S69_G5(t *testing.T) {
	runSnellmanLedgerResourcesMatch(t, "4pLeague_S69_D1L1_G5.txt")
}

func runSnellmanLedgerResourcesMatch(t *testing.T, fixtureFile string) {
	t.Helper()

	fixture := filepath.Join("testdata", "snellman_batch", fixtureFile)
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	snellman := string(raw)

	expected, err := parseSnellmanExpectedStates(snellman)
	if err != nil {
		t.Fatalf("parse expected states: %v", err)
	}

	concise, err := notation.ConvertSnellmanToConciseForReplay(snellman)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConciseForReplay: %v", err)
	}
	items, err := notation.ParseConciseLogStrict(concise)
	if err != nil {
		t.Fatalf("ParseConciseLogStrict: %v", err)
	}

	sim := NewGameSimulator(createInitialState(items), items)

	expIdx := 0
	for sim.CurrentIndex < len(sim.Actions) {
		item := sim.Actions[sim.CurrentIndex]
		if err := sim.StepForward(); err != nil {
			if expIdx < len(expected) {
				e := expected[expIdx]
				t.Fatalf("replay failed at simIndex=%d nextExpected(line=%d player=%s action=%q): %v", sim.CurrentIndex, e.Line, e.PlayerID, e.Action, err)
			}
			t.Fatalf("replay failed at simIndex=%d: %v", sim.CurrentIndex, err)
		}

		ai, ok := item.(notation.ActionItem)
		if !ok || ai.Action == nil {
			continue
		}
		if expIdx >= len(expected) {
			t.Fatalf("ran out of expected states at simIndex=%d action=%T", sim.CurrentIndex, ai.Action)
		}

		want := expected[expIdx]
		gotPlayer := ai.Action.GetPlayerID()
		if gotPlayer != want.PlayerID {
			t.Fatalf("alignment mismatch at expIdx=%d: want player=%s (snellman line %d action=%q), got player=%s action=%T",
				expIdx, want.PlayerID, want.Line, want.Action, gotPlayer, ai.Action)
		}

		p := sim.CurrentState.GetPlayer(gotPlayer)
		if p == nil {
			t.Fatalf("player not found in sim state: %s", gotPlayer)
		}

		if p.VictoryPoints != want.VP ||
			p.Resources.Coins != want.C ||
			p.Resources.Workers != want.W ||
			p.Resources.Priests != want.P ||
			p.Resources.Power.Bowl1 != want.PW1 ||
			p.Resources.Power.Bowl2 != want.PW2 ||
			p.Resources.Power.Bowl3 != want.PW3 ||
			p.CultPositions[0] != want.Fire ||
			p.CultPositions[1] != want.Water ||
			p.CultPositions[2] != want.Earth ||
			p.CultPositions[3] != want.Air {
			t.Fatalf("state mismatch after expIdx=%d simIndex=%d player=%s snellmanLine=%d action=%q\nwant: VP=%d C=%d W=%d P=%d PW=%d/%d/%d cult=%d/%d/%d/%d\ngot:  VP=%d C=%d W=%d P=%d PW=%d/%d/%d cult=%d/%d/%d/%d",
				expIdx, sim.CurrentIndex, gotPlayer, want.Line, want.Action,
				want.VP, want.C, want.W, want.P, want.PW1, want.PW2, want.PW3, want.Fire, want.Water, want.Earth, want.Air,
				p.VictoryPoints, p.Resources.Coins, p.Resources.Workers, p.Resources.Priests, p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3,
				p.CultPositions[0], p.CultPositions[1], p.CultPositions[2], p.CultPositions[3],
			)
		}

		expIdx++
	}

	if expIdx != len(expected) {
		t.Fatalf("expected %d ledger action rows, consumed %d", len(expected), expIdx)
	}
}
