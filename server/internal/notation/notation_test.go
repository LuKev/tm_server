package notation

import (
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	originalLog := &Log{
		MapName:      "Base",
		ScoringTiles: []string{"SCORE1", "SCORE2"},
		BonusCards:   []string{"BON1", "BON2"},
		Options:      []string{"OPT1"},
		Rounds: []*RoundLog{
			{
				RoundNumber: 1,
				TurnOrder:   []string{"Witches", "Nomads", "Cultists"},
				Actions: []*GameAction{
					NewGameAction("Witches", ActionBuild, map[string]string{"coord": "C4"}),
					NewGameAction("Nomads", ActionLeech, nil),
					NewGameAction("Cultists", ActionLeech, map[string]string{"decline": "true"}),
					NewGameAction("Cultists", ActionCultReaction, map[string]string{"track": "F"}),
					NewGameAction("Nomads", ActionDigBuild, map[string]string{"spades": "D", "coord": "F5"}),
				},
			},
		},
	}

	// Generate string
	generated := GenerateConciseLog(originalLog)
	t.Logf("Generated Log:\n%s", generated)

	// Parse back
	parsedLog, err := ParseConciseLog(generated)
	if err != nil {
		t.Fatalf("Failed to parse generated log: %v", err)
	}

	// Compare
	if parsedLog.MapName != originalLog.MapName {
		t.Errorf("MapName mismatch: got %s, want %s", parsedLog.MapName, originalLog.MapName)
	}
	if !reflect.DeepEqual(parsedLog.ScoringTiles, originalLog.ScoringTiles) {
		t.Errorf("ScoringTiles mismatch: got %v, want %v", parsedLog.ScoringTiles, originalLog.ScoringTiles)
	}

	if len(parsedLog.Rounds) != len(originalLog.Rounds) {
		t.Fatalf("Round count mismatch: got %d, want %d", len(parsedLog.Rounds), len(originalLog.Rounds))
	}

	for i, round := range originalLog.Rounds {
		parsedRound := parsedLog.Rounds[i]
		if parsedRound.RoundNumber != round.RoundNumber {
			t.Errorf("Round %d number mismatch: got %d, want %d", i, parsedRound.RoundNumber, round.RoundNumber)
		}
		if !reflect.DeepEqual(parsedRound.TurnOrder, round.TurnOrder) {
			t.Errorf("Round %d TurnOrder mismatch: got %v, want %v", i, parsedRound.TurnOrder, round.TurnOrder)
		}
		if len(parsedRound.Actions) != len(round.Actions) {
			t.Fatalf("Round %d Action count mismatch: got %d, want %d", i, len(parsedRound.Actions), len(round.Actions))
		}
		for j, action := range round.Actions {
			parsedAction := parsedRound.Actions[j]
			if parsedAction.Faction != action.Faction {
				t.Errorf("Round %d Action %d Faction mismatch: got %s, want %s", i, j, parsedAction.Faction, action.Faction)
			}
			if parsedAction.Type != action.Type {
				t.Errorf("Round %d Action %d Type mismatch: got %s, want %s", i, j, parsedAction.Type, action.Type)
			}
			// Params comparison might be tricky due to map equality, but reflect.DeepEqual handles it
			if !reflect.DeepEqual(parsedAction.Params, action.Params) {
				t.Errorf("Round %d Action %d Params mismatch: got %v, want %v", i, j, parsedAction.Params, action.Params)
			}
		}
	}
}

func TestParseActionString(t *testing.T) {
	tests := []struct {
		input      string
		wantType   ActionType
		wantParams map[string]string
	}{
		{"C4", ActionBuild, map[string]string{"coord": "C4"}},
		{"TP-C4", ActionUpgrade, map[string]string{"building": "TP", "coord": "C4"}},
		{"D-C4", ActionDigBuild, map[string]string{"spades": "D", "coord": "C4"}},
		{"ACT1-C4-C5", ActionPower, map[string]string{"code": "ACT1", "args": "C4-C5"}},
		{"Pass-BON1", ActionPass, map[string]string{"bonus": "BON1"}},
		{"L", ActionLeech, nil},
		{"DL", ActionLeech, map[string]string{"decline": "true"}},
		{"CULT-F", ActionCultReaction, map[string]string{"track": "F"}},
		{"->F", ActionSendPriest, map[string]string{"target": "F"}},
		{"B3", ActionBurn, map[string]string{"amount": "3"}},
		{"C3PW:1W", ActionConvert, map[string]string{"in": "3PW", "out": "1W"}},
		{"ACT-SH-D-C4", ActionSpecial, map[string]string{"code": "ACT-SH-D-C4"}},
	}

	for _, tt := range tests {
		got, err := ParseActionString("Faction", tt.input)
		if err != nil {
			t.Errorf("ParseActionString(%q) error = %v", tt.input, err)
			continue
		}
		if got.Type != tt.wantType {
			t.Errorf("ParseActionString(%q) Type = %v, want %v", tt.input, got.Type, tt.wantType)
		}
		// Check params subset
		for k, v := range tt.wantParams {
			if got.Params[k] != v {
				t.Errorf("ParseActionString(%q) Param[%s] = %v, want %v", tt.input, k, got.Params[k], v)
			}
		}
	}
}
