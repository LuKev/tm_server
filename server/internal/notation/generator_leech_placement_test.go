package notation

import (
	"strings"
	"testing"
)

func TestGenerateConciseLog_ReanchorsDistinctLeechSourcesWithPlainLeechTokens(t *testing.T) {
	input := `Game: Base
ScoringTiles: SCORE1, SCORE2, SCORE3, SCORE4, SCORE5, SCORE6
BonusCards: BON-SPD, BON-4C, BON-6C, BON-SHIP-VP, BON-WP, BON-BB, BON-TP
StartingVPs: Cultists:20, Witches:20, Darklings:20, Mermaids:20

Round 1
TurnOrder: Cultists, Witches, Darklings, Mermaids
------------------------------------------------------------
Cultists     | Witches      | Darklings    | Mermaids
------------------------------------------------------------
UP-TH-F7     |              |              |
             | BURN4.ACT5.C5|              |
             |              | UP-TH-G5     |
             |              |              | L-Cultists
             |              |              | L-Darklings
             |              | UP-TH-G6     |`

	items, err := ParseConciseLogStrict(input)
	if err != nil {
		t.Fatalf("ParseConciseLogStrict() error = %v", err)
	}

	logStrings, logLocations := GenerateConciseLog(items)
	columns := []string{"Cultists", "Witches", "Darklings", "Mermaids"}

	checked := 0
	for i, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		leech, ok := actionItem.Action.(*LogAcceptLeechAction)
		if !ok || strings.TrimSpace(leech.FromPlayerID) == "" {
			continue
		}
		loc := logLocations[i]
		if loc.LineIndex < 0 || loc.ColumnIndex < 0 {
			t.Fatalf("expected concrete location for leech action index %d", i)
		}
		token := tokenAt(logStrings, loc.LineIndex, loc.ColumnIndex, len(columns))
		if token != "L" {
			t.Fatalf("expected display token L at index %d, got %q\n%s", i, token, strings.Join(logStrings, "\n"))
		}

		prevFaction, ok := previousNonLeechFaction(logStrings, loc.LineIndex, loc.ColumnIndex, columns)
		if !ok {
			t.Fatalf("no previous non-leech token for leech %q at index %d\n%s", leech.FromPlayerID, i, strings.Join(logStrings, "\n"))
		}
		if normalizeFactionNameForMatching(prevFaction) != normalizeFactionNameForMatching(leech.FromPlayerID) {
			t.Fatalf("leech source mismatch at index %d: previous non-leech=%q expected=%q\n%s", i, prevFaction, leech.FromPlayerID, strings.Join(logStrings, "\n"))
		}
		checked++
	}

	if checked < 2 {
		t.Fatalf("expected to validate at least two anchored leeches, validated %d", checked)
	}
}

func TestGenerateConciseLog_SnellmanRound1_MermaidsLeechesAnchorToCultistsThenDarklings(t *testing.T) {
	snellman := strings.Join([]string{
		"option strict-leech\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"witches\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/2\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"mermaids\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/2/0/0\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t16 C\t-2\t4 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t1\tupgrade F7 to TP",
		"witches\t\t20 VP\t-2\t13 C\t-1\t5 W\t\t0 P\t-8\t6/2/0 PW\t\t0/0/0/2\t\tburn 4. action ACT5. build C5",
		"darklings\t\t20 VP\t-3\t12 C\t-2\t2 W\t\t2 P\t\t5/7/0 PW\t\t0/1/1/0\t1\tupgrade G5 to TP",
		"mermaids\t\t20 VP\t\t17 C\t\t6 W\t\t0 P\t+1\t2/10/0 PW\t\t0/2/0/0\t\t[opponent accepted power]",
		"mermaids\t\t20 VP\t\t17 C\t\t6 W\t\t0 P\t+1\t1/11/0 PW\t\t0/2/0/0\t\tLeech 1 from cultists",
		"mermaids\t\t20 VP\t-3\t14 C\t-2\t4 W\t\t0 P\t\t1/11/0 PW\t\t0/2/0/0\t\tLeech 1 from darklings",
		"darklings\t-1\t19 VP\t\t12 C\t\t2 W\t\t2 P\t+2\t3/9/0 PW\t\t0/1/1/0\t2 2\tupgrade G6 to TP",
	}, "\n")

	concise, err := ConvertSnellmanToConciseForReplay(snellman)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConciseForReplay() error = %v", err)
	}
	items, err := ParseConciseLogStrict(concise)
	if err != nil {
		t.Fatalf("ParseConciseLogStrict() error = %v\nconcise:\n%s", err, concise)
	}

	logStrings, logLocations := GenerateConciseLog(items)
	columns := []string{"Cultists", "Witches", "Darklings", "Mermaids"}

	checked := 0
	for i, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		leech, ok := actionItem.Action.(*LogAcceptLeechAction)
		if !ok || strings.TrimSpace(leech.FromPlayerID) == "" {
			continue
		}
		loc := logLocations[i]
		prevFaction, ok := previousNonLeechFaction(logStrings, loc.LineIndex, loc.ColumnIndex, columns)
		if !ok {
			t.Fatalf("no previous non-leech token for leech %q at index %d\n%s", leech.FromPlayerID, i, strings.Join(logStrings, "\n"))
		}
		if normalizeFactionNameForMatching(prevFaction) != normalizeFactionNameForMatching(leech.FromPlayerID) {
			t.Fatalf("leech source mismatch at index %d: previous non-leech=%q expected=%q\n%s", i, prevFaction, leech.FromPlayerID, strings.Join(logStrings, "\n"))
		}
		if tok := tokenAt(logStrings, loc.LineIndex, loc.ColumnIndex, len(columns)); tok != "L" {
			t.Fatalf("expected display token L, got %q\n%s", tok, strings.Join(logStrings, "\n"))
		}
		checked++
	}
	if checked < 2 {
		t.Fatalf("expected at least two source-anchored leeches, validated %d", checked)
	}
}

func tokenAt(lines []string, row, col, width int) string {
	if row < 0 || row >= len(lines) {
		return ""
	}
	cells := splitLogRow(lines[row], width)
	if col < 0 || col >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[col])
}

func previousNonLeechFaction(lines []string, row, col int, columns []string) (string, bool) {
	width := len(columns)
	for r := row; r >= 0; r-- {
		line := strings.TrimSpace(lines[r])
		if line == "" || strings.HasPrefix(line, "Round ") || strings.HasPrefix(line, "TurnOrder:") || strings.HasPrefix(line, "---") {
			continue
		}
		cells := splitLogRow(lines[r], width)
		startCol := width - 1
		if r == row {
			startCol = col - 1
		}
		for c := startCol; c >= 0; c-- {
			tok := strings.TrimSpace(cells[c])
			if tok == "" || isLeechOrDeclineToken(tok) {
				continue
			}
			return columns[c], true
		}
	}
	return "", false
}

func splitLogRow(line string, width int) []string {
	parts := strings.Split(line, "|")
	out := make([]string, width)
	for i := 0; i < width && i < len(parts); i++ {
		out[i] = parts[i]
	}
	return out
}
