package notation

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseActionCode_RecognizesSpecialACTCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{name: "bonus card cult action", code: "ACT-BON-E"},
		{name: "favor action placeholder", code: "ACT-FAV"},
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

func TestParseActionCode_ParsesCultDecreaseShorthand(t *testing.T) {
	action, err := parseActionCode("Cultists", "-W")
	if err != nil {
		t.Fatalf("parseActionCode(-W) error = %v", err)
	}
	if _, ok := action.(*LogCultTrackDecreaseAction); !ok {
		t.Fatalf("parseActionCode(-W) type = %T, want *LogCultTrackDecreaseAction", action)
	}
}

func TestParseActionCode_ParsesMultipleCultDecreaseSelectorsWithTown(t *testing.T) {
	action, err := parseActionCode("Witches", "-F.-W.-E.TW8VP")
	if err != nil {
		t.Fatalf("parseActionCode(-F.-W.-E.TW8VP) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(-F.-W.-E.TW8VP) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 4 {
		t.Fatalf("compound action count = %d, want 4", len(compound.Actions))
	}
	if _, ok := compound.Actions[0].(*LogCultTrackDecreaseAction); !ok {
		t.Fatalf("compound first action type = %T, want *LogCultTrackDecreaseAction", compound.Actions[0])
	}
	if _, ok := compound.Actions[1].(*LogCultTrackDecreaseAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogCultTrackDecreaseAction", compound.Actions[1])
	}
	if _, ok := compound.Actions[2].(*LogCultTrackDecreaseAction); !ok {
		t.Fatalf("compound third action type = %T, want *LogCultTrackDecreaseAction", compound.Actions[2])
	}
	if _, ok := compound.Actions[3].(*LogTownAction); !ok {
		t.Fatalf("compound fourth action type = %T, want *LogTownAction", compound.Actions[3])
	}
}

func TestParseActionCode_RejectsStandaloneConversion(t *testing.T) {
	_, err := parseActionCode("Cultists", "C5PW:1P")
	if err == nil {
		t.Fatalf("parseActionCode(standalone conversion) expected error, got nil")
	}
}

func TestParseActionCode_DLParsesAsLogDecline(t *testing.T) {
	action, err := parseActionCode("Cultists", "DL")
	if err != nil {
		t.Fatalf("parseActionCode(DL) error = %v", err)
	}
	if _, ok := action.(*LogDeclineLeechAction); !ok {
		t.Fatalf("parseActionCode(DL) type = %T, want *LogDeclineLeechAction", action)
	}
}

func TestParseActionCode_AllowsConversionInsideCompound(t *testing.T) {
	action, err := parseActionCode("Witches", "BURN3.C5PW:1P.+SHIP")
	if err != nil {
		t.Fatalf("parseActionCode(compound conversion) error = %v", err)
	}

	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(compound conversion) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 3 {
		t.Fatalf("compound action count = %d, want 3", len(compound.Actions))
	}
	if _, ok := compound.Actions[0].(*LogBurnAction); !ok {
		t.Fatalf("compound first action type = %T, want *LogBurnAction", compound.Actions[0])
	}
	if _, ok := compound.Actions[1].(*LogConversionAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogConversionAction", compound.Actions[1])
	}
	if gotType := fmt.Sprintf("%T", compound.Actions[2]); gotType != "*game.AdvanceShippingAction" {
		t.Fatalf("compound third action type = %s, want *game.AdvanceShippingAction", gotType)
	}
}

func TestParseActionCode_IsCaseInsensitiveForCompoundTokens(t *testing.T) {
	action, err := parseActionCode("Witches", "burn3.c5pw:1p.+ship")
	if err != nil {
		t.Fatalf("parseActionCode(lowercase compound) error = %v", err)
	}

	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(lowercase compound) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 3 {
		t.Fatalf("compound action count = %d, want 3", len(compound.Actions))
	}
	if _, ok := compound.Actions[0].(*LogBurnAction); !ok {
		t.Fatalf("compound first action type = %T, want *LogBurnAction", compound.Actions[0])
	}
	if _, ok := compound.Actions[1].(*LogConversionAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogConversionAction", compound.Actions[1])
	}
	if gotType := fmt.Sprintf("%T", compound.Actions[2]); !strings.HasSuffix(gotType, ".AdvanceShippingAction") {
		t.Fatalf("compound third action type = %s, want *game.AdvanceShippingAction", gotType)
	}
}

func TestParseActionCode_ParsesPassWithTrailingCultBonusAsCompound(t *testing.T) {
	action, err := parseActionCode("Cultists", "PASS-BON-BB.+A")
	if err != nil {
		t.Fatalf("parseActionCode(pass+cult bonus compound) error = %v", err)
	}

	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(pass+cult bonus compound) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 2 {
		t.Fatalf("compound action count = %d, want 2", len(compound.Actions))
	}
	if gotType := fmt.Sprintf("%T", compound.Actions[0]); !strings.HasSuffix(gotType, ".PassAction") {
		t.Fatalf("compound first action type = %s, want *game.PassAction", gotType)
	}
	if _, ok := compound.Actions[1].(*LogCultistAdvanceAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogCultistAdvanceAction", compound.Actions[1])
	}
}

func TestParseActionCode_KeepsBonusSpadeBuildCombined(t *testing.T) {
	action, err := parseActionCode("Dwarves", "ACTS-G2.G2")
	if err != nil {
		t.Fatalf("parseActionCode(ACTS-G2.G2) error = %v", err)
	}
	if _, ok := action.(*LogSpecialAction); !ok {
		t.Fatalf("parseActionCode(ACTS-G2.G2) type = %T, want *LogSpecialAction", action)
	}
}

func TestParseActionCode_KeepsNomadsSandstormBuildCombined(t *testing.T) {
	action, err := parseActionCode("Nomads", "ACT-SH-T-F4.F4")
	if err != nil {
		t.Fatalf("parseActionCode(ACT-SH-T-F4.F4) error = %v", err)
	}
	if _, ok := action.(*LogSpecialAction); !ok {
		t.Fatalf("parseActionCode(ACT-SH-T-F4.F4) type = %T, want *LogSpecialAction", action)
	}
}

func TestParseActionCode_ParsesCombinedSpecialThenConversion(t *testing.T) {
	action, err := parseActionCode("Cultists", "ACTS-B5.B5.C1PW:1C")
	if err != nil {
		t.Fatalf("parseActionCode(ACTS-B5.B5.C1PW:1C) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(ACTS-B5.B5.C1PW:1C) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 2 {
		t.Fatalf("compound action count = %d, want 2", len(compound.Actions))
	}
	if _, ok := compound.Actions[0].(*LogSpecialAction); !ok {
		t.Fatalf("compound first action type = %T, want *LogSpecialAction", compound.Actions[0])
	}
	if _, ok := compound.Actions[1].(*LogConversionAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogConversionAction", compound.Actions[1])
	}
}

func TestParseActionCode_DoesNotMergeSpecialAcrossConversionToken(t *testing.T) {
	action, err := parseActionCode("Nomads", "ACTS-E3.C2PW:2C.E3")
	if err != nil {
		t.Fatalf("parseActionCode(ACTS-E3.C2PW:2C.E3) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parseActionCode(ACTS-E3.C2PW:2C.E3) type = %T, want *LogCompoundAction", action)
	}
	if len(compound.Actions) != 3 {
		t.Fatalf("compound action count = %d, want 3", len(compound.Actions))
	}
	if _, ok := compound.Actions[0].(*LogSpecialAction); !ok {
		t.Fatalf("compound first action type = %T, want *LogSpecialAction", compound.Actions[0])
	}
	if _, ok := compound.Actions[1].(*LogConversionAction); !ok {
		t.Fatalf("compound second action type = %T, want *LogConversionAction", compound.Actions[1])
	}
	if gotType := fmt.Sprintf("%T", compound.Actions[2]); !strings.HasSuffix(gotType, ".TransformAndBuildAction") {
		t.Fatalf("compound third action type = %s, want *game.TransformAndBuildAction", gotType)
	}
}

func TestIsCoord_RejectsConversionLikeTokens(t *testing.T) {
	if isCoord("C2PW:2C") {
		t.Fatalf("isCoord(C2PW:2C) = true, want false")
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
