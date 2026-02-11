package replay

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/notation"
)

func TestParseReplayLogContent_AutoDetectsConcise(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	concise := `Game: Base
ScoringTiles: SCORE1, SCORE2, SCORE3, SCORE4, SCORE5, SCORE6
BonusCards: BON-SPD, BON-4C, BON-6C, BON-SHIP, BON-WP, BON-BB, BON-TP
StartingVPs: Cultists:20, Engineers:20

Round 1
TurnOrder: Cultists, Engineers
------------------------------------------------------------
Cultists     | Engineers
------------------------------------------------------------
UP-TH-E6.+E  | L`

	items, canonical, err := manager.parseReplayLogContent(concise, ReplayLogFormatAuto)
	if err != nil {
		t.Fatalf("parseReplayLogContent(auto concise) error = %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("parseReplayLogContent(auto concise) returned no items")
	}
	if strings.TrimSpace(canonical) != strings.TrimSpace(concise) {
		t.Fatalf("canonical concise log should match input")
	}
}

func TestParseReplayLogContent_SnellmanConvertsToConcise(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	snellman := strings.Join([]string{
		"option strict-leech\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t12 C\t-1\t2 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
	}, "\n")

	items, canonical, err := manager.parseReplayLogContent(snellman, ReplayLogFormatSnellman)
	if err != nil {
		t.Fatalf("parseReplayLogContent(snellman) error = %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("parseReplayLogContent(snellman) returned no items")
	}
	if !strings.Contains(canonical, "Game: Base") {
		t.Fatalf("snellman canonical log should be concise output, got:\n%s", canonical)
	}
}

func TestParseReplayLogContent_SnellmanExtractsSetupScoringAndBonuses(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	snellman := strings.Join([]string{
		"Randomize setup\tshow history",
		"round 1 scoring: score9, TE >> 4\tshow history",
		"Round 2 scoring: SCORE3, D >> 2\tshow history",
		"Round 3 scoring: SCORE8, TP >> 3\tshow history",
		"Round 4 scoring: SCORE5, D >> 2\tshow history",
		"Round 5 scoring: SCORE6, TP >> 3\tshow history",
		"Round 6 scoring: SCORE4, SA/SH >> 5\tshow history",
		"Removing tile BON4\tshow history",
		"Removing tile BON5\tshow history",
		"Removing tile BON7\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t12 C\t-1\t2 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
	}, "\n")

	_, canonical, err := manager.parseReplayLogContent(snellman, ReplayLogFormatSnellman)
	if err != nil {
		t.Fatalf("parseReplayLogContent(snellman) error = %v", err)
	}

	if !strings.Contains(canonical, "ScoringTiles: SCORE9, SCORE3, SCORE8, SCORE5, SCORE6, SCORE4") {
		t.Fatalf("expected scoring tiles in canonical output, got:\n%s", canonical)
	}
	if !strings.Contains(canonical, "BonusCards:") {
		t.Fatalf("expected bonus cards in canonical output, got:\n%s", canonical)
	}

	var bonusLine string
	for _, line := range strings.Split(canonical, "\n") {
		if strings.HasPrefix(line, "BonusCards:") {
			bonusLine = strings.TrimSpace(strings.TrimPrefix(line, "BonusCards:"))
			break
		}
	}
	if bonusLine == "" {
		t.Fatalf("failed to extract bonus card line from canonical output:\n%s", canonical)
	}

	removed := map[string]bool{
		"BON-SHIP": true,
		"BON-WP":   true,
		"BON-TP":   true,
	}
	for _, token := range strings.Split(bonusLine, ",") {
		card := strings.TrimSpace(token)
		if removed[card] {
			t.Fatalf("canonical bonus cards should exclude removed Snellman bonuses, got:\n%s", canonical)
		}
	}
}

func TestParseReplayLogContent_SnellmanExtractsSetupVariantHeaderPhrases(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	snellman := strings.Join([]string{
		"Randomize setup\tshow history",
		"Round 1 scoring tile: SCORE9, TE >> 4\tshow history",
		"Round 2 scoring tile: SCORE3, D >> 2\tshow history",
		"Round 3 scoring tile: SCORE8, TP >> 3\tshow history",
		"Round 4 scoring tile: SCORE5, D >> 2\tshow history",
		"Round 5 scoring tile: SCORE6, TP >> 3\tshow history",
		"Round 6 scoring tile: SCORE4, SA/SH >> 5\tshow history",
		"Removing bonus tile BON4\tshow history",
		"Removing bonus tile BON5\tshow history",
		"Removing bonus tile BON7\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t12 C\t-1\t2 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
	}, "\n")

	_, canonical, err := manager.parseReplayLogContent(snellman, ReplayLogFormatSnellman)
	if err != nil {
		t.Fatalf("parseReplayLogContent(snellman variant headers) error = %v", err)
	}

	if !strings.Contains(canonical, "ScoringTiles: SCORE9, SCORE3, SCORE8, SCORE5, SCORE6, SCORE4") {
		t.Fatalf("expected scoring tiles in canonical output, got:\n%s", canonical)
	}
	if !strings.Contains(canonical, "BonusCards:") {
		t.Fatalf("expected bonus cards in canonical output, got:\n%s", canonical)
	}
}

func TestParseReplayLogContent_ConciseStrictErrors(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	badConcise := `Game: Base
ScoringTiles: SCORE1, SCORE2, SCORE3, SCORE4, SCORE5, SCORE6
BonusCards: BON-SPD, BON-4C, BON-6C, BON-SHIP, BON-WP, BON-BB, BON-TP
StartingVPs: Cultists:20, Engineers:20

Round 1
TurnOrder: Cultists, Engineers
------------------------------------------------------------
Cultists     | Engineers
------------------------------------------------------------
BADTOKEN     |`

	_, _, err := manager.parseReplayLogContent(badConcise, ReplayLogFormatConcise)
	if err == nil {
		t.Fatalf("parseReplayLogContent(concise strict) expected error")
	}
	if _, ok := err.(*notation.ConciseParseError); !ok {
		t.Fatalf("error type = %T, want *notation.ConciseParseError", err)
	}
}
