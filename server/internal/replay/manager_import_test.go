package replay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
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

func TestParseReplayLogContent_SnellmanReanchorsLeechAfterSourceAction(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	manager.SetSourceAnchoredLeechOrdering(true)

	fixturePath := filepath.Join("testdata", "snellman_batch", "4pLeague_S69_D1L1_G6.txt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	items, _, err := manager.parseReplayLogContent(string(content), ReplayLogFormatSnellman)
	if err != nil {
		t.Fatalf("parseReplayLogContent(snellman): %v", err)
	}

	indexCultistsUpF7 := findActionIndex(items, func(action game.Action) bool {
		return actionContainsUpgrade(action, "Cultists", "F7", models.BuildingTradingHouse)
	})
	indexMermaidsLeechCultists := findActionIndex(items, func(action game.Action) bool {
		leech, ok := action.(*notation.LogAcceptLeechAction)
		return ok &&
			leech.PlayerID == "Mermaids" &&
			strings.EqualFold(strings.TrimSpace(leech.FromPlayerID), "Cultists")
	})
	indexWitchesBurnAct5C5 := findActionIndex(items, func(action game.Action) bool {
		compound, ok := action.(*notation.LogCompoundAction)
		if !ok || compound.GetPlayerID() != "Witches" {
			return false
		}
		hasBurn4 := false
		hasAct5 := false
		hasBuildC5 := false
		for _, sub := range compound.Actions {
			switch v := sub.(type) {
			case *notation.LogBurnAction:
				hasBurn4 = hasBurn4 || v.Amount == 4
			case *notation.LogPowerAction:
				hasAct5 = hasAct5 || strings.EqualFold(v.ActionCode, "ACT5")
			case *game.TransformAndBuildAction:
				hasBuildC5 = hasBuildC5 || (v.BuildDwelling && notation.HexToShortString(v.TargetHex) == "C5")
			}
		}
		return hasBurn4 && hasAct5 && hasBuildC5
	})
	indexDarklingsUpG5 := findActionIndex(items, func(action game.Action) bool {
		return actionContainsUpgrade(action, "Darklings", "G5", models.BuildingTradingHouse)
	})
	indexMermaidsLeechDarklings := findActionIndex(items, func(action game.Action) bool {
		leech, ok := action.(*notation.LogAcceptLeechAction)
		return ok &&
			leech.PlayerID == "Mermaids" &&
			strings.EqualFold(strings.TrimSpace(leech.FromPlayerID), "Darklings")
	})
	indexMermaidsUpG6 := findActionIndex(items, func(action game.Action) bool {
		return actionContainsUpgrade(action, "Mermaids", "G6", models.BuildingTradingHouse)
	})

	indices := map[string]int{
		"Cultists UP-TH-F7":     indexCultistsUpF7,
		"Mermaids L-Cultists":   indexMermaidsLeechCultists,
		"Witches BURN4.ACT5.C5": indexWitchesBurnAct5C5,
		"Darklings UP-TH-G5":    indexDarklingsUpG5,
		"Mermaids L-Darklings":  indexMermaidsLeechDarklings,
		"Mermaids UP-TH-G6":     indexMermaidsUpG6,
	}
	for name, idx := range indices {
		if idx < 0 {
			t.Fatalf("failed to find action index for %s", name)
		}
	}

	if !(indexCultistsUpF7 < indexMermaidsLeechCultists &&
		indexMermaidsLeechCultists < indexWitchesBurnAct5C5 &&
		indexWitchesBurnAct5C5 < indexDarklingsUpG5 &&
		indexDarklingsUpG5 < indexMermaidsLeechDarklings &&
		indexMermaidsLeechDarklings < indexMermaidsUpG6) {
		t.Fatalf(
			"unexpected ordering: CultistsUP=%d MermaidsL-Cultists=%d WitchesBURN=%d DarklingsUP=%d MermaidsL-Darklings=%d MermaidsUP=%d",
			indexCultistsUpF7,
			indexMermaidsLeechCultists,
			indexWitchesBurnAct5C5,
			indexDarklingsUpG5,
			indexMermaidsLeechDarklings,
			indexMermaidsUpG6,
		)
	}
}

func TestReplayWithSourceAnchoredLeechOrdering_S69G6Completes(t *testing.T) {
	manager := NewReplayManager(t.TempDir())
	manager.SetSourceAnchoredLeechOrdering(true)

	fixturePath := filepath.Join("testdata", "snellman_batch", "4pLeague_S69_D1L1_G6.txt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gameID := "reanchor_s69_g6"
	if err := manager.ImportText(gameID, string(content), "snellman"); err != nil {
		t.Fatalf("ImportText: %v", err)
	}

	session, err := manager.StartReplay(gameID, true)
	if err != nil {
		t.Fatalf("StartReplay: %v", err)
	}

	totalActions := len(session.Simulator.Actions)
	if err := manager.JumpTo(gameID, totalActions); err != nil {
		t.Fatalf("JumpTo(%d): %v", totalActions, err)
	}
}

func findActionIndex(items []notation.LogItem, predicate func(game.Action) bool) int {
	for i, item := range items {
		actionItem, ok := item.(notation.ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if predicate(actionItem.Action) {
			return i
		}
	}
	return -1
}

func actionContainsUpgrade(action game.Action, playerID, coord string, building models.BuildingType) bool {
	switch v := action.(type) {
	case *notation.LogPreIncomeAction:
		if v == nil {
			return false
		}
		return actionContainsUpgrade(v.Action, playerID, coord, building)
	case *notation.LogPostIncomeAction:
		if v == nil {
			return false
		}
		return actionContainsUpgrade(v.Action, playerID, coord, building)
	case *notation.LogCompoundAction:
		for _, sub := range v.Actions {
			if actionContainsUpgrade(sub, playerID, coord, building) {
				return true
			}
		}
		return false
	case *game.UpgradeBuildingAction:
		return v.PlayerID == playerID &&
			v.NewBuildingType == building &&
			notation.HexToShortString(v.TargetHex) == coord
	default:
		return false
	}
}
