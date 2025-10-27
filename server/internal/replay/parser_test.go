package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestParseLogLine_Setup(t *testing.T) {
	line := "engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("Failed to parse line: %v", err)
	}

	if entry.Faction != models.FactionEngineers {
		t.Errorf("Expected faction Engineers, got %v", entry.Faction)
	}
	if entry.VP != 20 {
		t.Errorf("Expected VP 20, got %d", entry.VP)
	}
	if entry.Coins != 10 {
		t.Errorf("Expected Coins 10, got %d", entry.Coins)
	}
	if entry.Workers != 2 {
		t.Errorf("Expected Workers 2, got %d", entry.Workers)
	}
	if entry.Priests != 0 {
		t.Errorf("Expected Priests 0, got %d", entry.Priests)
	}
	if entry.PowerBowls.Bowl1 != 3 || entry.PowerBowls.Bowl2 != 9 || entry.PowerBowls.Bowl3 != 0 {
		t.Errorf("Expected PowerBowls 3/9/0, got %d/%d/%d",
			entry.PowerBowls.Bowl1, entry.PowerBowls.Bowl2, entry.PowerBowls.Bowl3)
	}
	if entry.Action != "setup" {
		t.Errorf("Expected action 'setup', got '%s'", entry.Action)
	}
}

func TestParseLogLine_Build(t *testing.T) {
	line := "engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tbuild E7"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("Failed to parse line: %v", err)
	}

	if entry.Faction != models.FactionEngineers {
		t.Errorf("Expected faction Engineers, got %v", entry.Faction)
	}
	if entry.Action != "build E7" {
		t.Errorf("Expected action 'build E7', got '%s'", entry.Action)
	}
}

func TestParseLogLine_WithDeltas(t *testing.T) {
	line := "engineers\t\t20 VP\t-1\t9 C\t-1\t3 W\t\t0 P\t-12\t6/0/0 PW\t\t0/0/0/0\t1 \tburn 6. action ACT6. transform F2 to gray. build D4"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("Failed to parse line: %v", err)
	}

	if entry.Faction != models.FactionEngineers {
		t.Errorf("Expected faction Engineers, got %v", entry.Faction)
	}
	if entry.VP != 20 || entry.VPDelta != -1 {
		t.Errorf("Expected VP 20 delta -1, got %d delta %d", entry.VP, entry.VPDelta)
	}
	if entry.Coins != 9 || entry.CoinsDelta != -1 {
		t.Errorf("Expected Coins 9 delta -1, got %d delta %d", entry.Coins, entry.CoinsDelta)
	}
	if entry.Workers != 3 {
		t.Errorf("Expected Workers 3, got %d", entry.Workers)
	}
	if entry.PowerBowls.Bowl1 != 6 {
		t.Errorf("Expected Bowl1 6, got %d", entry.PowerBowls.Bowl1)
	}
}

func TestParseLogLine_Cultists(t *testing.T) {
	line := "cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("Failed to parse line: %v", err)
	}

	if entry.Faction != models.FactionCultists {
		t.Errorf("Expected faction Cultists, got %v", entry.Faction)
	}
	if entry.CultTracks.Fire != 1 {
		t.Errorf("Expected Fire cult 1, got %d", entry.CultTracks.Fire)
	}
	if entry.CultTracks.Earth != 1 {
		t.Errorf("Expected Earth cult 1, got %d", entry.CultTracks.Earth)
	}
	if entry.CultTracks.Water != 0 || entry.CultTracks.Air != 0 {
		t.Errorf("Expected Water/Air cult 0, got %d/%d", entry.CultTracks.Water, entry.CultTracks.Air)
	}
}

func TestParseLogLine_Comment(t *testing.T) {
	line := "Round 1 scoring: SCORE2, TOWN >> 5\tshow history"
	entry, err := ParseLogLine(line)
	if err != nil {
		t.Fatalf("Failed to parse line: %v", err)
	}

	if !entry.IsComment {
		t.Errorf("Expected comment line")
	}
}

func TestParseAction_Build(t *testing.T) {
	actionType, params, err := ParseAction("build E7")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionBuild {
		t.Errorf("Expected ActionBuild, got %v", actionType)
	}
	if params["coord"] != "E7" {
		t.Errorf("Expected coord E7, got %s", params["coord"])
	}
}

func TestParseAction_Upgrade(t *testing.T) {
	actionType, params, err := ParseAction("upgrade E5 to TP")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionUpgrade {
		t.Errorf("Expected ActionUpgrade, got %v", actionType)
	}
	if params["coord"] != "E5" {
		t.Errorf("Expected coord E5, got %s", params["coord"])
	}
	if params["building"] != "TP" {
		t.Errorf("Expected building TP, got %s", params["building"])
	}
}

func TestParseAction_Pass(t *testing.T) {
	actionType, params, err := ParseAction("Pass BON1")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionPass {
		t.Errorf("Expected ActionPass, got %v", actionType)
	}
	if params["bonus"] != "BON1" {
		t.Errorf("Expected bonus BON1, got %s", params["bonus"])
	}
}

func TestParseAction_TransformAndBuild(t *testing.T) {
	actionType, params, err := ParseAction("1  burn 6. action ACT6. transform F2 to gray. build D4")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionTransformAndBuild {
		t.Errorf("Expected ActionTransformAndBuild, got %v", actionType)
	}
	if params["transform_coord"] != "F2" {
		t.Errorf("Expected transform_coord F2, got %s", params["transform_coord"])
	}
	if params["transform_color"] != "gray" {
		t.Errorf("Expected transform_color gray, got %s", params["transform_color"])
	}
	if params["coord"] != "D4" {
		t.Errorf("Expected coord D4, got %s", params["coord"])
	}
}

func TestParseAction_DigAndBuild(t *testing.T) {
	actionType, params, err := ParseAction("dig 1. build G6")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionTransformAndBuild {
		t.Errorf("Expected ActionTransformAndBuild, got %v", actionType)
	}
	if params["spades"] != "1" {
		t.Errorf("Expected spades 1, got %s", params["spades"])
	}
	if params["coord"] != "G6" {
		t.Errorf("Expected coord G6, got %s", params["coord"])
	}
}

func TestParseAction_SendPriest(t *testing.T) {
	actionType, params, err := ParseAction("send p to WATER")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionSendPriest {
		t.Errorf("Expected ActionSendPriest, got %v", actionType)
	}
	if params["cult"] != "WATER" {
		t.Errorf("Expected cult WATER, got %s", params["cult"])
	}
}

func TestParseAction_PowerAction(t *testing.T) {
	actionType, params, err := ParseAction("action ACT6")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionPowerAction {
		t.Errorf("Expected ActionPowerAction, got %v", actionType)
	}
	if params["action_type"] != "ACT6" {
		t.Errorf("Expected action_type ACT6, got %s", params["action_type"])
	}
}

func TestParseAction_AdvanceShipping(t *testing.T) {
	actionType, _, err := ParseAction("advance ship")
	if err != nil {
		t.Fatalf("Failed to parse action: %v", err)
	}

	if actionType != ActionAdvanceShipping {
		t.Errorf("Expected ActionAdvanceShipping, got %v", actionType)
	}
}
