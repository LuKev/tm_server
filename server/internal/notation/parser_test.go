package notation

import (
	"errors"
	"testing"
)

func TestParseActionCode_RecognizesSpecialACTCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{name: "bonus card cult action", code: "ACT-BON-E"},
		{name: "mermaids town action", code: "ACT-TOWN-2_-3"},
		{name: "engineers bridge action", code: "ACT-BR-C2-D4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := parseActionCode("Cultists", tt.code)
			if err != nil {
				t.Fatalf("parseActionCode(%q) error = %v", tt.code, err)
			}
			if _, ok := action.(*LogSpecialAction); !ok {
				t.Fatalf("parseActionCode(%q) type = %T, want *LogSpecialAction", tt.code, action)
			}
		})
	}
}

func TestParseActionCode_ParsesCultShorthandInCompound(t *testing.T) {
	action, err := parseActionCode("Cultists", "UP-TH-E6.+E")
	if err != nil {
		t.Fatalf("parseActionCode(compound) error = %v", err)
	}

	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(compound) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 2 {
		t.Fatalf("compound action count = %d, want 2", len(compound.Actions))
	}
	if _, ok := compound.Actions[1].(*LogCultistAdvanceAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogCultistAdvanceAction", compound.Actions[1])
	}
}

func TestParseActionCode_RejectsStandaloneConversion(t *testing.T) {
	_, err := parseActionCode("Cultists", "C5PW:1P")
	if err == nil {
		t.Fatalf("parseActionCode(standalone conversion) expected error, got nil")
	}
}

func TestParseConciseLogStrict_ReturnsLocationForInvalidToken(t *testing.T) {
	input := `Game: Base
ScoringTiles: SCORE1, SCORE2, SCORE3, SCORE4, SCORE5, SCORE6
BonusCards: BON-SPD, BON-4C, BON-6C, BON-SHIP, BON-WP, BON-BB, BON-TP
StartingVPs: Cultists:20, Engineers:20

Round 1
TurnOrder: Cultists, Engineers
------------------------------------------------------------
Cultists     | Engineers
------------------------------------------------------------
BADTOKEN     |`

	_, err := ParseConciseLogStrict(input)
	if err == nil {
		t.Fatalf("ParseConciseLogStrict() expected error, got nil")
	}

	var parseErr *ConciseParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type = %T, want *ConciseParseError", err)
	}
	if parseErr.Token != "BADTOKEN" {
		t.Fatalf("parseErr.Token = %q, want BADTOKEN", parseErr.Token)
	}
	if parseErr.PlayerID != "Cultists" {
		t.Fatalf("parseErr.PlayerID = %q, want Cultists", parseErr.PlayerID)
	}
	if parseErr.Line <= 0 || parseErr.Column <= 0 {
		t.Fatalf("invalid parse error location: line=%d column=%d", parseErr.Line, parseErr.Column)
	}
}
