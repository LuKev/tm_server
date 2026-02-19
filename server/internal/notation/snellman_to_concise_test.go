package notation

import (
	"strings"
	"testing"
)

func TestConvertSnellmanToConcise_SpecialActions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Witches Ride ACTW",
			input:    "action ACTW. build F3",
			expected: "ACT-SH-D-F3",
		},
		{
			name:     "Giants Stronghold ACTG",
			input:    "action ACTG. transform G4 to red",
			expected: "ACT-SH-S-G4",
		},
		{
			name:     "Nomads Stronghold ACTN transform",
			input:    "action ACTN. transform E8 to yellow",
			expected: "ACT-SH-T-E8",
		},
		{
			name:     "Nomads Stronghold ACTN build",
			input:    "action ACTN. build H5",
			expected: "ACT-SH-T-H5.H5",
		},
		{
			name:     "Nomads Stronghold ACTN build keeps interleaved conversion order",
			input:    "action ACTN. convert 2w to 2c. build g3",
			expected: "ACT-SH-T-G3.C2W:2C.G3",
		},
		{
			name:     "Upgrade internal part of compound",
			input:    "convert 2PW to 2C. upgrade E9 to SH",
			expected: "C2PW:2C.UP-SH-E9",
		},
		{
			name:     "Combined Witches Ride with Conversion",
			input:    "convert 1PW to 1C. action ACTW. build H4",
			expected: "C1PW:1C.ACT-SH-D-H4",
		},
		{
			name:     "Lowercase single-unit conversion reward",
			input:    "convert 3pw to w. build c2",
			expected: "C3PW:1W.C2",
		},
		{
			name:     "Pass with Cult Advance Prefix",
			input:    "+FIRE. pass BON10",
			expected: "+F.PASS-BON-SHIP-VP",
		},
		{
			name:     "Favor action without inline track remains action token",
			input:    "action FAV6. convert 1pw to 1c",
			expected: "ACT-FAV.C1PW:1C",
		},
		{
			name:     "Dig plus immediate transform emits only transform",
			input:    "dig 1. transform I11 to gray",
			expected: "T-I11",
		},
		{
			name:     "Dig with interleaved conversion keeps DIG token",
			input:    "dig 1. convert 1PW to 1C. transform I11 to gray",
			expected: "DIG1-I11.C1PW:1C.T-I11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faction := "cultists"
			if strings.Contains(tt.name, "Dig ") {
				faction = "dwarves"
			}
			got := convertCompoundActionToConcise(tt.input, faction, 0)
			if got != tt.expected {
				t.Errorf("convertCompoundActionToConcise() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConvertSnellmanToConcise_LegacySpecialActionMappings(t *testing.T) {
	t.Run("Chaos Magicians ACTC maps to ACT-SH-2X", func(t *testing.T) {
		got := convertCompoundActionToConcise("action ACTC. advance dig. build F2", "chaosmagicians", 0)
		want := "ACT-SH-2X.+DIG.F2"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("Engineers ACTE with bridge maps to ACT-BR", func(t *testing.T) {
		got := convertCompoundActionToConcise("action ACTE. bridge D6:C5", "engineers", 0)
		want := "ACT-BR-D6-C5"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("Swarmlings ACTS with TP upgrade maps to ACT-SH-TP", func(t *testing.T) {
		got := convertCompoundActionToConcise("action ACTS. upgrade C3 to TP", "swarmlings", 0)
		want := "ACT-SH-TP-C3"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})
}

func TestConvertSnellmanToConcise_CompoundSendPriestAndBon1SpadeTransform(t *testing.T) {
	t.Run("Compound send priest is preserved", func(t *testing.T) {
		input := "burn 3. convert 5PW to 1P. send p to EARTH"
		got := convertCompoundActionToConcise(input, "engineers", 3)
		want := "BURN3.C5PW:1P.->E3"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("Compound advance ship is preserved", func(t *testing.T) {
		input := "burn 3. convert 5PW to 1P. advance ship"
		got := convertCompoundActionToConcise(input, "witches", 0)
		want := "BURN3.C5PW:1P.+SHIP"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("Compound advance dig is preserved", func(t *testing.T) {
		input := "burn 2. convert 3PW to 1W. advance dig"
		got := convertCompoundActionToConcise(input, "halflings", 0)
		want := "BURN2.C3PW:1W.+DIG"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("BON1 transform is ACTS", func(t *testing.T) {
		input := "action BON1. transform C5 to green"
		got := convertCompoundActionToConcise(input, "auren", 0)
		want := "ACTS-C5"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("BON1 build includes build token", func(t *testing.T) {
		input := "action BON1. build E6"
		got := convertCompoundActionToConcise(input, "fakirs", 0)
		want := "ACTS-E6.E6"
		if got != want {
			t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
		}
	})

	t.Run("Send priest then conversion stays compound", func(t *testing.T) {
		input := "send p to AIR. convert 1PW to 1C"
		got := convertActionToConcise(input, "cultists", false, 2)
		want := "->A2.C1PW:1C"
		if got != want {
			t.Fatalf("convertActionToConcise() = %v, want %v", got, want)
		}
	})
}

func TestExtractSnellmanAction_PreservesLeadingSendPriestInCompound(t *testing.T) {
	parts := []string{
		"cultists",
		"+1",
		"49 VP",
		"send p to AIR. convert 1PW to 1C",
	}
	got := extractSnellmanAction(parts)
	want := "send p to AIR. convert 1PW to 1C"
	if got != want {
		t.Fatalf("extractSnellmanAction() = %q, want %q", got, want)
	}
}

func TestConvertSnellmanToConcise_PlainPassParses(t *testing.T) {
	got := convertActionToConcise("pass", "cultists", false, 0)
	if got != "PASS" {
		t.Fatalf("convertActionToConcise(pass) = %v, want PASS", got)
	}
}

func TestConvertSnellmanToConcise_OmitsTransformColorForHomeTerrain(t *testing.T) {
	got := convertCompoundActionToConcise("burn 3. action ACT6. transform E5 to brown. transform H5 to brown. build E5", "cultists", 0)
	want := "BURN3.ACT6.T-E5.T-H5.E5"
	if got != want {
		t.Fatalf("convertCompoundActionToConcise() = %v, want %v", got, want)
	}
}

func TestConvertSnellmanToConcise_SendPriestInfersCultSpotFromDelta(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 2 income\tshow history",
		"Round 2, turn 1\tshow history",
		"cultists\t+4\t21 VP\t-2\t14 C\t-1\t3 W\t\t1 P\t-8\t4/5/3 PW\t\t1/0/4/0\t3\taction ACT5. build G4",
		"darklings\t-2\t13 VP\t\t7 C\t\t3 W\t\t2 P\t+3\t0/2/3 PW\t\t0/1/5/0\t\tLeech 3 from cultists",
		"darklings\t\t13 VP\t+7\t14 C\t\t3 W\t\t2 P\t-8\t4/0/0 PW\t\t0/1/5/0\t\tburn 1. action ACT4",
		"engineers\t\t15 VP\t\t6 C\t\t3 W\t-1\t0 P\t+1\t0/6/1 PW\t+3\t0/0/5/3\t\tsend p to AIR",
		"auren\t\t24 VP\t-3\t6 C\t\t5 W\t\t1 P\t\t5/0/0 PW\t\t0/1/1/1\t\tconvert 3PW to 3C",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "BURN1.ACT4   | ->A3         | C3") {
		t.Fatalf("expected compact row with explicit send-priest spot:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_FavorActionTownAndAurenStrongholdCodes(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 6 income\tshow history",
		"Round 6, turn 7\tshow history",
		"darklings\t\t130 VP\t\t0 C\t\t0 W\t\t0 P\t+2\t0/3/1 PW\t+1\t5/7/7/2\t\taction FAV6. +FIRE",
		"auren\t\t89 VP\t\t3 C\t\t6 W\t\t1 P\t\t0/4/1 PW\t+2\t9/8/1/8\t\taction ACTA. +2FIRE",
		"engineers\t+6\t90 VP\t-1\t4 C\t-1\t1 W\t\t2 P\t+8\t0/3/4 PW\t\t3/6/7/10\t\tbuild F2. +TW4",
		"auren\t+13\t78 VP\t-1\t16 C\t-1\t10 W\t\t1 P\t-2\t1/0/4 PW\t\t4/8/1/8\t2\tconvert 1pw to 1c. build g2. +tw8",
		"darklings\t+3\t143 VP\t\t0 C\t\t0 W\t\t0 P\t\t2/1/0 PW\t\t5/7/7/2\t\t+3vp for network",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "ACT-FAV-F") {
		t.Fatalf("expected favor action to map to ACT-FAV-F:\n%s", got)
	}
	if !strings.Contains(got, "ACT-SH-F") {
		t.Fatalf("expected Auren stronghold action to map to ACT-SH-F:\n%s", got)
	}
	if !strings.Contains(got, "F2.TW6VP") {
		t.Fatalf("expected +TW4 to map to TW6VP:\n%s", got)
	}
	if !strings.Contains(got, "C1PW:1C.G2.TW11VP") {
		t.Fatalf("expected lowercase +tw8 to map to TW11VP:\n%s", got)
	}
	if strings.Contains(got, "+3.+") || strings.Contains(got, "for network") {
		t.Fatalf("expected scoring deltas to be ignored, got:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_MermaidsConnectMultipleTownsPreservesOrder(t *testing.T) {
	// Regression test for a subtle ordering bug:
	// In a compound Snellman row like:
	//   upgrade ... +FAV.. +TW2 convert ... +TW4 convert ...
	// the conversions must remain between the town selections, otherwise the power
	// bowls diverge from the Snellman ledger (because spending power returns it to bowl 1).
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 6, turn 2\tshow history",
		"mermaids\t+23\t93 VP\t-3\t9 C\t\t8 W\t\t2 P\t+6\t1/0/5 PW\t+2\t7/5/1/0\t2\tupgrade C1 to TE. +FAV5. connect r1. +TW2. connect r10. convert 1PW to 1C. +TW4. convert 1PW to 1C",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	// Expect the concise token sequence to preserve:
	//   UP-TE-C1, FAV-F2, TW7VP, C1PW:1C, TW6VP, C1PW:1C
	// (Favor tile 5 is Fire+2 => FAV-F2; TW2 => TW7VP; TW4 => TW6VP.)
	want := "UP-TE-C1.FAV-F2.TW7VP.C1PW:1C.TW6VP.C1PW:1C"
	if !strings.Contains(got, want) {
		t.Fatalf("expected Mermaids connect/town/conversion order to be preserved, missing %q in:\n%s", want, got)
	}
}

func TestConvertSnellmanToConcise_UpgradeSanctuaryFavorThenTown(t *testing.T) {
	// Sanctuary/Temple upgrades create a pending favor selection first.
	// Town formation is checked only after the favor selection is completed,
	// so concise logs must select FAV before selecting the TW* token.
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 5, turn 1\tshow history",
		"cultists\t+10\t75 VP\t\t13 C\t-4\t7 W\t\t2 P\t-1\t0/1/4 PW\t+1\t7/8/7/10\t3 4 2\tconvert 2PW to 2C. upgrade E6 to SA. +TW1. +FAV12",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	// TW1 -> TW5VP; FAV12 -> FAV-A1. We need FAV ahead of TW.
	want := "C2PW:2C.UP-SA-E6.FAV-A1.TW5VP"
	if !strings.Contains(got, want) {
		t.Fatalf("expected upgrade->favor->town ordering, missing %q in:\n%s", want, got)
	}
}

func TestConvertSnellmanToConcise_CompactsAdjacentNonLeechRows(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 2 income\tshow history",
		"Round 2, turn 3\tshow history",
		"darklings\t\t20 VP\t-2\t8 C\t-1\t1 W\t\t1 P\t\t4/0/0 PW\t\t0/1/5/0\t\tbuild I10",
		"engineers\t\t20 VP\t-5\t5 C\t-2\t1 W\t\t0 P\t\t0/4/3 PW\t+1\t0/1/6/3\t2 1\tupgrade E7 to TE. +FAV10",
		"auren\t\t20 VP\t\t5 C\t\t3 W\t\t0 P\t+1\t0/5/0 PW\t\t0/4/1/1\t\tLeech 1 from engineers",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "I10          | UP-TE-E7.FAV-W1 | L") {
		t.Fatalf("expected compaction of adjacent non-leech rows:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_CultistsLeadingCultStepBacktracksToPriorAction(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 2 income\tshow history",
		"Round 2, turn 4\tshow history",
		"cultists\t+2\t29 VP\t-2\t6 C\t-1\t1 W\t\t0 P\t\t4/5/0 PW\t\t1/0/5/0\t1\tbuild H5",
		"darklings\t+6\t27 VP\t-2\t8 C\t-1\t0 W\t-1\t0 P\t\t4/0/0 PW\t\t0/1/5/0\t\tdig 1. build D8",
		"engineers\t\t14 VP\t\t1 C\t\t1 W\t\t0 P\t\t0/4/3 PW\t+1\t0/1/6/3\t\taction BON2. +EARTH",
		"auren\t+2\t29 VP\t-2\t3 C\t-1\t2 W\t\t0 P\t\t0/5/0 PW\t\t0/4/1/1\t\tbuild A10",
		"Round 2, turn 5\tshow history",
		"cultists\t+2\t31 VP\t-2\t4 C\t-1\t0 W\t\t0 P\t\t4/5/0 PW\t\t1/0/6/0\t2\t+EARTH. build F3",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "H5.+E") {
		t.Fatalf("expected +EARTH to backtrack to prior Cultists action:\n%s", got)
	}
	if strings.Contains(got, "+E.F3") {
		t.Fatalf("did not expect leading +EARTH to remain chained to new action:\n%s", got)
	}
	if !strings.Contains(got, "ACT-BON-E") {
		t.Fatalf("expected BON2 +EARTH to parse as ACT-BON-E:\n%s", got)
	}
}

func TestConvertSnellmanToConciseForReplay_CultistsBumpsBacktrackAndLeechesStayStandalone(t *testing.T) {
	input := strings.Join([]string{
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
		"cultists\t\t20 VP\t\t16 C\t\t4 W\t\t0 P\t\t5/7/0 PW\t+1\t1/0/1/0\t\t+EARTH. Leech 2 from darklings. upgrade F7 to TE. +FAV9",
		"mermaids\t-1\t19 VP\t\t14 C\t\t4 W\t\t0 P\t+2\t3/9/0 PW\t\t0/2/0/0\t\tLeech 2 from darklings",
		"cultists\t\t20 VP\t\t16 C\t\t4 W\t\t0 P\t\t5/7/0 PW\t\t1/0/2/0\t\t+EARTH",
	}, "\n")

	got, err := ConvertSnellmanToConciseForReplay(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConciseForReplay() error = %v", err)
	}

	if !strings.Contains(got, "UP-TH-F7.+E") {
		t.Fatalf("expected first cultists trigger action to contain chained +E:\n%s", got)
	}
	if !strings.Contains(got, "UP-TE-F7.FAV-F1.+E") {
		t.Fatalf("expected second cultists trigger action to contain chained +E:\n%s", got)
	}

	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		for _, cell := range cells {
			tok := strings.TrimSpace(cell)
			if tok == "" {
				continue
			}
			if strings.Contains(tok, ".L") || strings.Contains(tok, ".DL") || strings.HasPrefix(tok, "L.") || strings.HasPrefix(tok, "DL.") {
				t.Fatalf("leech action must be standalone, found chained token %q in:\n%s", tok, got)
			}
			if tok == "+E" || tok == "+W" || tok == "+F" || tok == "+A" {
				t.Fatalf("standalone cult bump token is illegal in replay concise output, found %q in:\n%s", tok, got)
			}
		}
	}
}

func TestExtractSnellmanAction_CombinesSplitBon2TrackParts(t *testing.T) {
	parts := []string{"engineers", "foo", "action BON2", "+AIR"}
	got := extractSnellmanAction(parts)
	want := "action BON2. +AIR"
	if got != want {
		t.Fatalf("extractSnellmanAction() = %q, want %q", got, want)
	}
}

func TestConvertCompoundActionToConcise_BON2ConsumesOneMatchingTrack(t *testing.T) {
	got := convertCompoundActionToConcise("+WATER. action BON2. +WATER", "cultists", 0)
	want := "ACT-BON-W.+W"
	if got != want {
		t.Fatalf("convertCompoundActionToConcise() = %q, want %q", got, want)
	}
}

func TestConvertSnellmanToConcise_ParsesMixedCaseConvertUpgradeFav(t *testing.T) {
	input := "auren\t+5\t37 VP\t-5\t0 C\t-4\t1 W\t\t0 P\t-2\t4/1/0 PW\t+1\t4/4/1/2\t1 \tConvert 1pw to 1c. upgrade C3 to SH. +FAV9"
	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "C1PW:1C.UP-SH-C3.FAV-F1") {
		t.Fatalf("expected mixed-case compound parsing with favor tile:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_PreservesAdvanceShipWithTrailingConvert(t *testing.T) {
	input := "engineers\t+2\t70 VP\t-4\t10 C\t-1\t3 W\t\t2 P\t-1\t3/4/1 PW\t\t0/2/1/4\t\tadvance ship. convert 1PW to 1C"
	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "+SHIP.C1PW:1C") {
		t.Fatalf("expected advance ship compound to keep conversion:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_ParsesGreyTerrainSpelling(t *testing.T) {
	input := "chaosmagicians\t+4\t28 VP\t-2\t11 C\t-1\t4 W\t\t1 P\t-12\t6/0/2 PW\t\t6/0/6/2\t\taction ACT6. transform I6 to grey. build G1"
	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "ACT6.T-I6-Gy.G1") {
		t.Fatalf("expected grey to map to Gy:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_LeechFromEngineersStaysBeforeAurenPassInRound3Order(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"Round 2 income\tshow history",
		"Round 2, turn 6\tshow history",
		"darklings\t\t15 VP\t\t11 C\t\t2 W\t\t1 P\t\t0/8/0 PW\t\t0/1/4/0\t\tpass BON1",
		"auren\t\t15 VP\t\t11 C\t\t2 W\t\t1 P\t\t0/8/0 PW\t\t0/1/4/0\t\tpass BON2",
		"engineers\t\t15 VP\t\t11 C\t\t2 W\t\t1 P\t\t0/8/0 PW\t\t0/1/4/0\t\tpass BON3",
		"cultists\t\t15 VP\t\t11 C\t\t2 W\t\t1 P\t\t0/8/0 PW\t\t0/1/4/0\t\tpass BON4",
		"Round 3 income\tshow history",
		"Round 3, turn 6\tshow history",
		"engineers\t\t18 VP\t-1\t1 C\t-1\t1 W\t\t0 P\t-12\t6/1/0 PW\t\t0/1/6/5\t1 \taction ACT6. build E4",
		"cultists\t\t49 VP\t\t2 C\t\t2 W\t\t0 P\t+1\t3/3/0 PW\t\t1/1/7/2\t\tLeech 1 from engineers",
		"auren\t\t37 VP\t\t0 C\t\t1 W\t\t0 P\t\t3/2/0 PW\t\t4/4/1/4\t\tPass bon1",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	actIdx := strings.Index(got, "ACT6.E4")
	leechIdx := strings.Index(got, "ACT6.E4      |              | L")
	if leechIdx == -1 {
		leechIdx = strings.Index(got, "ACT6.E4      | L")
	}
	passIdx := -1
	if actIdx >= 0 {
		if rel := strings.Index(got[actIdx:], "PASS-BON-SPD"); rel >= 0 {
			passIdx = actIdx + rel
		}
	}
	if actIdx == -1 || leechIdx == -1 || passIdx == -1 {
		t.Fatalf("expected ACT6, corresponding leech, and pass in output:\n%s", got)
	}
	if !(actIdx <= leechIdx && leechIdx < passIdx) {
		t.Fatalf("expected cultists leech from engineers to appear before auren pass:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DistinctLeechSourcesDoNotShareMergedSourceRow(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"halflings\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"dwarves\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"mermaids\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 2\tshow history",
		"halflings\t\t23 VP\t-5\t13 C\t-2\t2 W\t\t0 P\t\t2/10/0 PW\t+1\t0/0/2/1\t1 1\tupgrade F5 to TE. +FAV11",
		"dwarves\t\t23 VP\t\t12 C\t\t5 W\t\t0 P\t+1\t0/11/1 PW\t\t0/0/2/0\t\tLeech 1 from halflings",
		"darklings\t\t21 VP\t\t17 C\t\t3 W\t+1\t1 P\t-6\t4/5/0 PW\t\t0/1/1/0\t\tburn 3. action ACT2",
		"dwarves\t\t23 VP\t-5\t7 C\t-2\t3 W\t\t0 P\t+1\t0/10/2 PW\t+1\t0/0/3/0\t1\tupgrade E7 to TE. +FAV11",
		"mermaids\t\t23 VP\t\t14 C\t\t4 W\t\t0 P\t-12\t6/0/0 PW\t\t0/2/0/0\t\tburn 6. action ACT6. transform C3 to blue. transform C4 to blue",
		"Round 1, turn 3\tshow history",
		"halflings\t+3\t26 VP\t-2\t11 C\t-1\t1 W\t\t0 P\t-8\t6/2/0 PW\t\t0/0/2/1\t1\tburn 4. action ACT5. build E10",
		"darklings\t+3\t24 VP\t-3\t14 C\t-2\t1 W\t\t1 P\t\t4/5/0 PW\t\t0/1/1/0\t2 1 2\tupgrade G5 to TP",
		"dwarves\t\t23 VP\t\t7 C\t\t3 W\t\t0 P\t+1\t0/9/3 PW\t\t0/0/3/0\t\tLeech 1 from halflings",
		"dwarves\t\t23 VP\t\t7 C\t\t3 W\t\t0 P\t+1\t0/8/4 PW\t\t0/0/3/0\t\tLeech 1 from darklings",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "BURN4.ACT5.E10 |              | L-Halflings  |") {
		t.Fatalf("expected dwarves leech from halflings on halflings action row:\n%s", got)
	}
	if !strings.Contains(got, "             | UP-TH-G5     | L-Darklings  |") {
		t.Fatalf("expected dwarves leech from darklings on darklings action row:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_PassOnLaterLineDoesNotMergeIntoPriorAction(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"mermaids\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/2/0/0\t\tsetup",
		"dwarves\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/2/0\t\tsetup",
		"halflings\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/2/1\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 7\tshow history",
		"darklings\t\t22 VP\t-5\t9 C\t-2\t1 W\t\t0 P\t+2\t2/6/0 PW\t+2\t0/1/7/0\t2 1 2\tupgrade G5 to TE. +FAV7",
		"mermaids\t-1\t24 VP\t\t5 C\t\t0 W\t\t0 P\t+2\t0/6/0 PW\t\t0/2/1/0\t\tLeech 2 from darklings",
		"mermaids\t\t24 VP\t\t5 C\t\t0 W\t\t0 P\t\t0/6/0 PW\t\t0/2/1/0\t\tpass BON3",
		"dwarves\t\t28 VP\t\t6 C\t\t0 W\t\t0 P\t+1\t0/5/7 PW\t\t0/0/3/0\t\tLeech 1 from darklings",
		"darklings\t\t22 VP\t\t9 C\t\t1 W\t\t0 P\t\t2/6/0 PW\t\t0/1/7/0\t\tpass BON1",
		"halflings\t-1\t26 VP\t\t9 C\t\t0 W\t\t0 P\t+2\t1/7/0 PW\t\t0/0/2/1\t\tLeech 2 from darklings",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if strings.Contains(got, "UP-TE-G5.FAV-E2.PASS-BON-SPD") {
		t.Fatalf("expected darklings pass to be on a later row, not merged:\n%s", got)
	}
	if !strings.Contains(got, "UP-TE-G5.FAV-E2") || !strings.Contains(got, "PASS-BON-SPD") {
		t.Fatalf("expected both upgrade and later pass entries:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DelayedLeechAnchorsToSourceRow(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tupgrade G5 to TP",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tLeech 1 from cultists",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tLeech 1 from cultists",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "UP-TH-E6     |              | L-Cultists   | L-Cultists") {
		t.Fatalf("expected cultists row with delayed leeches from engineers+auren:\n%s", got)
	}
	if !strings.Contains(got, "             | UP-TH-G5     |              |") {
		t.Fatalf("expected darklings action on following row:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DelayedCultBonusFromPassBacktracks(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 2\tshow history",
		"cultists\t\t19 VP\t-5\t13 C\t-2\t2 W\t\t0 P\t+1\t0/12/0 PW\t+1\t1/0/3/0\t2 1\tupgrade E6 to TE. +FAV11",
		"darklings\t\t18 VP\t\t16 C\t+2\t4 W\t\t1 P\t-8\t6/2/0 PW\t\t0/1/1/0\t\tburn 4. action ACT3",
		"cultists\t\t19 VP\t\t13 C\t\t2 W\t\t0 P\t\t0/12/0 PW\t+1\t1/0/3/0\t\t[opponent accepted power]",
		"auren\t-1\t19 VP\t\t14 C\t\t4 W\t\t0 P\t+2\t1/11/0 PW\t\t0/1/0/1\t\tLeech 2 from cultists",
		"Round 1, turn 3\tshow history",
		"cultists\t\t19 VP\t+1\t14 C\t\t2 W\t\t0 P\t\t0/12/0 PW\t\t1/0/4/0\t\t+EARTH. pass BON9",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "UP-TE-E6.FAV-E1.+E") {
		t.Fatalf("expected delayed cult bonus to backtrack to upgrade action:\n%s", got)
	}
	if !strings.Contains(got, "PASS-BON-DW") {
		t.Fatalf("expected pass action to remain after extracting delayed cult bonus:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_StandaloneCultBonusAfterPassBacktracksToLeechSource(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 2 income\tshow history",
		"Round 2, turn 7\tshow history",
		"cultists\t+2\t36 VP\t-2\t1 C\t-1\t0 W\t\t0 P\t\t5/3/0 PW\t\t2/4/3/0\t\tbuild A7",
		"cultists\t+9\t45 VP\t+1\t2 C\t\t0 W\t\t0 P\t\t5/3/0 PW\t\t2/4/3/0\t\tpass BON6",
		"cultists\t\t45 VP\t\t2 C\t\t0 W\t\t0 P\t\t5/3/0 PW\t+1\t2/4/3/0\t\t[opponent accepted power]",
		"engineers\t\t35 VP\t\t2 C\t\t0 W\t\t0 P\t+1\t2/2/3 PW\t\t0/0/3/0\t\tLeech 1 from cultists",
		"cultists\t\t45 VP\t\t2 C\t\t0 W\t\t0 P\t\t5/3/0 PW\t\t2/4/3/1\t\t+AIR",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "A7.+A") {
		t.Fatalf("expected standalone cult bonus to backtrack to leech-source action:\n%s", got)
	}
	if strings.Contains(got, "PASS-BON-BB.+A") {
		t.Fatalf("unexpected cult bonus attached to pass token:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_LeechFromLeftColumnsMovesToLaterRow(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tupgrade G5 to TP",
		"engineers\t\t20 VP\t-2\t8 C\t-1\t4 W\t\t0 P\t\t0/10/2 PW\t\t0/0/0/0\t\tupgrade F6 to TP",
		"darklings\t-2\t18 VP\t\t16 C\t\t2 W\t\t1 P\t+3\t2/10/0 PW\t\t0/1/1/0\t\tLeech 3 from engineers",
		"cultists\t\t20 VP\t\t18 C\t\t4 W\t\t0 P\t+1\t3/9/0 PW\t\t1/0/2/0\t\tLeech 1 from engineers",
		"auren\t\t20 VP\t\t17 C\t\t6 W\t\t0 P\t+1\t3/9/0 PW\t\t0/1/0/1\t\tLeech 1 from engineers",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "UP-TH-F6") {
		t.Fatalf("expected engineers source action to be present:\n%s", got)
	}
	if !strings.Contains(got, "| UP-TH-F6") {
		t.Fatalf("expected source action row to remain present:\n%s", got)
	}
	if !strings.Contains(got, "L-Engineers  | L3-Engineers |              |") {
		t.Fatalf("expected left-side leeches to move to later row:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_CompactsEarlyRoundRows(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t18 C\t-2\t4 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
		"darklings\t\t20 VP\t-3\t16 C\t-2\t2 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tupgrade G5 to TP",
		"cultists\t\t20 VP\t\t18 C\t\t4 W\t\t0 P\t+1\t4/8/0 PW\t\t1/0/1/0\t\tLeech 1 from darklings",
		"engineers\t\t20 VP\t\t10 C\t\t5 W\t\t0 P\t+1\t0/11/1 PW\t\t0/0/0/0\t\tLeech 1 from cultists",
		"engineers\t\t20 VP\t\t10 C\t\t5 W\t\t0 P\t+1\t0/10/2 PW\t\t0/0/0/0\t\tLeech 1 from darklings",
		"engineers\t\t20 VP\t-2\t8 C\t-1\t4 W\t\t0 P\t\t0/10/2 PW\t\t0/0/0/0\t\tupgrade F6 to TP",
		"darklings\t-2\t18 VP\t\t16 C\t\t2 W\t\t1 P\t+3\t2/10/0 PW\t\t0/1/1/0\t\tLeech 3 from engineers",
		"cultists\t\t20 VP\t\t18 C\t\t4 W\t\t0 P\t+1\t3/9/0 PW\t\t1/0/2/0\t\tLeech 1 from engineers",
		"auren\t\t20 VP\t\t17 C\t\t6 W\t\t0 P\t+1\t3/9/0 PW\t\t0/1/0/1\t\tLeech 1 from engineers",
		"auren\t\t20 VP\t-3\t14 C\t-2\t4 W\t\t0 P\t\t3/9/0 PW\t\t0/1/0/1\t\tupgrade F4 to TP",
		"engineers\t\t20 VP\t\t8 C\t\t4 W\t\t0 P\t+1\t0/9/3 PW\t\t0/0/0/0\t\tLeech 1 from auren",
		"cultists\t-1\t19 VP\t\t18 C\t\t4 W\t\t0 P\t+2\t1/11/0 PW\t\t1/0/2/0\t\tLeech 2 from auren",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "L-Darklings  |              | UP-TH-F6     | L-Engineers") {
		t.Fatalf("expected compact row containing cultists leech plus engineers action and right-side leech:\n%s", got)
	}
	if !strings.Contains(got, "UP-TH-F4") {
		t.Fatalf("expected auren action to be preserved:\n%s", got)
	}
	if !strings.Contains(got, "L2-Auren     |              | L-Auren      |") {
		t.Fatalf("expected left-side delayed leeches to remain after source actions:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DoesNotDropActionsBetweenIncomeBlocks(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"witches\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/2\t\tsetup",
		"Round 6 income\tshow history",
		"witches\t\t70 VP\t\t10 C\t\t3 W\t\t1 P\t\t4/0/1 PW\t\t1/1/0/8\t\tcult_income_for_faction",
		"cultists\t\t65 VP\t\t4 C\t\t3 W\t\t0 P\t\t1/4/0 PW\t\t7/8/7/9\t\tcult_income_for_faction",
		"engineers\t\t91 VP\t\t3 C\t\t0 W\t\t0 P\t\t6/1/0 PW\t\t0/4/5/0\t\tcult_income_for_faction",
		"darklings\t\t89 VP\t\t5 C\t\t0 W\t\t0 P\t\t3/0/1 PW\t\t3/10/10/2\t\tcult_income_for_faction",
		"engineers\t\t91 VP\t\t3 C\t\t0 W\t\t0 P\t\t6/1/0 PW\t\t0/4/5/0\t\ttransform I10 to green",
		"Round 6 income\tshow history",
		"engineers\t\t91 VP\t+10\t13 C\t+4\t4 W\t+2\t2 P\t+12\t0/1/6 PW\t\t0/4/5/0\t\tother_income_for_faction",
		"Round 6, turn 1\tshow history",
		"engineers\t\t105 VP\t-1\t13 C\t-1\t2 W\t\t2 P\t\t3/1/3 PW\t\t2/4/5/0\t\taction BON1. transform I10 to gray. build I10",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "T-I10-G") {
		t.Fatalf("expected transform between income blocks to be preserved:\n%s", got)
	}
	if !strings.Contains(got, "ACTS-I10.I10") {
		t.Fatalf("expected BON1 transform/build to preserve transformed home terrain action:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_PreservesTurnOrderWhenCompacting(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 2\tshow history",
		"cultists\t\t19 VP\t-5\t13 C\t-2\t2 W\t\t0 P\t+1\t0/12/0 PW\t+1\t1/0/3/0\t2 1\tupgrade E6 to TE. +FAV11",
		"darklings\t\t18 VP\t\t16 C\t+2\t4 W\t\t1 P\t-8\t6/2/0 PW\t\t0/1/1/0\t\tburn 4. action ACT3",
		"auren\t-1\t19 VP\t\t14 C\t\t4 W\t\t0 P\t+2\t1/11/0 PW\t\t0/1/0/1\t\tLeech 2 from cultists",
		"Round 1, turn 3\tshow history",
		"cultists\t\t19 VP\t+1\t14 C\t\t2 W\t\t0 P\t\t0/12/0 PW\t\t1/0/4/0\t\t+EARTH. pass BON9",
		"engineers\t\t20 VP\t-1\t7 C\t-1\t3 W\t\t0 P\t-12\t6/4/0 PW\t\t0/0/0/0\t2\tburn 2. action ACT6. build G6",
		"auren\t\t19 VP\t\t14 C\t\t4 W\t+1\t1 P\t-6\t4/5/0 PW\t\t0/1/0/1\t\tburn 3. action ACT2",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	upIdx := strings.Index(got, "UP-TE-E6.FAV-E1")
	burn4Idx := strings.Index(got, "BURN4.ACT3")
	passIdx := strings.Index(got, "PASS-BON-DW")
	burn2Idx := strings.Index(got, "BURN2.ACT6.G6")
	burn3Idx := strings.Index(got, "BURN3.ACT2")
	if upIdx == -1 || burn4Idx == -1 || passIdx == -1 || burn2Idx == -1 || burn3Idx == -1 {
		t.Fatalf("expected all key actions in output:\n%s", got)
	}
	if burn4Idx < upIdx {
		t.Fatalf("darklings turn-2 action should not appear before cultists turn-2 action:\n%s", got)
	}
	if burn2Idx < passIdx || burn3Idx < passIdx {
		t.Fatalf("turn-3 actions should appear after cultists pass in turn 3:\n%s", got)
	}
	if strings.Contains(got, "UP-TE-E6.FAV-E1.+E | BURN4.ACT3") {
		t.Fatalf("main actions should not share a row:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_CompactsNonLeechMainActionsOnSameRow(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 1 income\tshow history",
		"Round 1, turn 2\tshow history",
		"cultists\t\t19 VP\t-5\t13 C\t-2\t2 W\t\t0 P\t+1\t0/12/0 PW\t+1\t1/0/3/0\t2 1\tupgrade E6 to TE. +FAV11",
		"darklings\t\t18 VP\t\t16 C\t+2\t4 W\t\t1 P\t-8\t6/2/0 PW\t\t0/1/1/0\t\tburn 4. action ACT3",
		"cultists\t\t19 VP\t\t13 C\t\t2 W\t\t0 P\t\t0/12/0 PW\t+1\t1/0/3/0\t\t[opponent accepted power]",
		"engineers\t\t20 VP\t\t8 C\t\t4 W\t\t0 P\t+1\t0/8/4 PW\t\t0/0/0/0\t\tLeech 1 from cultists",
		"engineers\t\t20 VP\t-1\t7 C\t-1\t3 W\t\t0 P\t-12\t6/4/0 PW\t\t0/0/0/0\t2\tburn 2. action ACT6. build G6",
		"Round 1, turn 3\tshow history",
		"cultists\t\t19 VP\t+1\t14 C\t\t2 W\t\t0 P\t\t0/12/0 PW\t\t1/0/4/0\t\t+EARTH. pass BON9",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "BURN4.ACT3   | BURN2.ACT6.G6") {
		t.Fatalf("expected darklings and engineers main actions to share row when not leech-anchored:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_UsesRoundTurnOrderForColumns(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tPass BON1",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tPass BON5",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tPass BON2",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tPass BON3",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"auren\t\t20 VP\t-3\t12 C\t-2\t2 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tupgrade F4 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "Cultists     | Darklings    | Engineers    | Auren") {
		t.Fatalf("expected round columns in round turn-order order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_Round1TurnOrderIgnoresSetupPassOrder(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tPass BON1",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tPass BON5",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tPass BON2",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tPass BON3",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"cultists\t\t20 VP\t-3\t18 C\t-2\t4 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tupgrade E6 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if !strings.Contains(got, "TurnOrder: Cultists, Darklings, Engineers, Auren") {
		t.Fatalf("expected round 1 turn order to use setup faction order, not setup pass order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_MaintainPlayerOrderUsesCyclicRotation(t *testing.T) {
	input := strings.Join([]string{
		"option maintain-player-order\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tpass BON1",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tpass BON3",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tpass BON2",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tpass BON4",
		"Round 2 income\tshow history",
		"Round 2, turn 1\tshow history",
		"auren\t\t20 VP\t-3\t12 C\t-2\t2 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tupgrade F4 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "Round 2\nTurnOrder: Auren, Cultists, Darklings, Engineers") {
		t.Fatalf("expected cyclic turn order anchored to first passer under maintain-player-order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_VariableTurnOrderUsesPassOrder(t *testing.T) {
	input := strings.Join([]string{
		"option variable-turn-order\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tpass BON1",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tpass BON3",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tpass BON2",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tpass BON4",
		"Round 2 income\tshow history",
		"Round 2, turn 1\tshow history",
		"auren\t\t20 VP\t-3\t12 C\t-2\t2 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tupgrade F4 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "Round 2\nTurnOrder: Auren, Darklings, Cultists, Engineers") {
		t.Fatalf("expected variable turn order to use full pass order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_VariableTurnOrderUsesPassOrderWithPlainPass(t *testing.T) {
	input := strings.Join([]string{
		"option variable-turn-order\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 6\tshow history",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tpass",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tpass",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tpass",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tpass",
		"Round 2 income\tshow history",
		"Round 2, turn 1\tshow history",
		"engineers\t\t20 VP\t-3\t7 C\t-2\t0 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tupgrade F4 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "Round 2\nTurnOrder: Engineers, Cultists, Auren, Darklings") {
		t.Fatalf("expected variable turn order to follow full plain-pass order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_VariableTurnOrderOverridesMaintainWhenBothPresent(t *testing.T) {
	input := strings.Join([]string{
		"option maintain-player-order\tshow history",
		"option variable-turn-order\tshow history",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tsetup",
		"Round 1 income\tshow history",
		"Round 1, turn 1\tshow history",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tpass BON3",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tpass BON2",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tpass BON4",
		"auren\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/1/0/1\t\tpass BON1",
		"Round 2 income\tshow history",
		"Round 2, turn 1\tshow history",
		"darklings\t\t20 VP\t-3\t12 C\t-2\t0 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tupgrade E5 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "Round 2\nTurnOrder: Darklings, Cultists, Engineers, Auren") {
		t.Fatalf("expected variable-turn-order to override maintain-player-order:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_CompactsActionIntoLeechRowWhenToRight(t *testing.T) {
	input := strings.Join([]string{
		"option variable-turn-order\tshow history",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/1/1/0\t\tsetup",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t1/0/1/0\t\tsetup",
		"engineers\t\t20 VP\t\t10 C\t\t2 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tsetup",
		"witches\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/2\t\tsetup",
		"Round 6 income\tshow history",
		"Round 6, turn 1\tshow history",
		"darklings\t\t20 VP\t-6\t9 C\t-4\t0 W\t+4\t5 P\t\t5/7/0 PW\t\t0/1/1/0\t\tupgrade B5 to SH",
		"witches\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t+1\t4/8/0 PW\t\t0/0/0/2\t\tLeech 1 from darklings",
		"cultists\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t-8\t4/5/3 PW\t\t1/0/1/0\t\taction ACT4",
		"engineers\t\t20 VP\t-2\t8 C\t-1\t1 W\t\t0 P\t\t3/9/0 PW\t\t0/0/0/0\t\tupgrade E3 to TP",
		"darklings\t\t20 VP\t\t9 C\t\t0 W\t\t5 P\t+1\t4/8/0 PW\t\t0/1/1/0\t\tLeech 1 from engineers",
		"witches\t\t20 VP\t-3\t12 C\t-2\t1 W\t\t0 P\t\t4/8/0 PW\t\t0/0/0/2\t\tupgrade H8 to TP",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if strings.Contains(got, "L            |              |              |             \n             |              |              | UP-TH-H8") {
		t.Fatalf("did not expect an extra uncompacted blank-gap row before witches action:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DarklingsDigTransformIsNotPlusDig(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 3 income\tshow history",
		"Round 3, turn 4\tshow history",
		"darklings\t+2\t63 VP\t\t1 C\t\t6 W\t-1\t3 P\t\t0/3/1 PW\t\t1/8/10/1\t\tdig 1. transform E8 to brown",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if strings.Contains(got, "+DIG") {
		t.Fatalf("did not expect +DIG for darklings dig+transform action:\n%s", got)
	}
	if !strings.Contains(got, "T-E8-Br") {
		t.Fatalf("expected darklings dig+transform to parse as transform:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DelayedDeclineAnchorsToLastLeechSourceAction(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"halflings\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"darklings\t\t20 VP\t\t15 C\t\t1 W\t\t1 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"dwarves\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"mermaids\t\t20 VP\t\t15 C\t\t3 W\t\t0 P\t\t5/7/0 PW\t\t0/0/0/0\t\tsetup",
		"Round 4 income\tshow history",
		"Round 4, turn 6\tshow history",
		"darklings\t\t71 VP\t-2\t0 C\t-2\t1 W\t\t0 P\t\t1/3/0 PW\t\t2/4/7/2\t\tupgrade G7 to TE",
		"mermaids\t\t43 VP\t\t0 C\t\t3 W\t\t0 P\t+1\t4/1/0 PW\t\t4/4/1/8\t\tLeech 1 from darklings",
		"dwarves\t\t77 VP\t\t10 C\t\t3 W\t\t0 P\t\t0/7/0 PW\t\t7/0/5/0\t\tDecline 2 from darklings",
		"darklings\t\t71 VP\t\t1 C\t\t2 W\t\t2 P\t\t1/3/0 PW\t\t2/4/7/2\t\tpass BON9",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}
	if !strings.Contains(got, "UP-TE-G7") {
		t.Fatalf("expected source upgrade action:\n%s", got)
	}
	if !strings.Contains(got, "UP-TE-G7     | DL2-Darklings | L-Darklings") {
		t.Fatalf("expected delayed decline to anchor to source upgrade row:\n%s", got)
	}
	if strings.Contains(got, "PASS-BON-DW  | DL") {
		t.Fatalf("did not expect decline to anchor to pass row:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DelayedLeechDoesNotGetHiddenByCompactedInterveningAction(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 5 income\tshow history",
		"Round 5, turn 3\tshow history",
		"halflings\t\t73 VP\t-5\t1 C\t-2\t8 W\t\t1 P\t\t4/1/0 PW\t+1\t0/1/2/1\t1 \tupgrade A12 to TE. +FAV10",
		"darklings\t+3\t61 VP\t-4\t1 C\t\t6 W\t-1\t4 P\t\t2/2/0 PW\t\t1/8/9/0\t\tadvance ship",
		"mermaids\t\t51 VP\t\t9 C\t\t8 W\t\t2 P\t+1\t0/1/5 PW\t\t4/4/7/3\t\tLeech 1 from halflings",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if strings.Contains(got, "UP-TE-A12.FAV-W1 | +SHIP        |") && strings.Contains(got, "| L") {
		t.Fatalf("expected delayed leech to remain visually tied to halflings source, not darklings row:\n%s", got)
	}
	if !strings.Contains(got, "UP-TE-A12.FAV-W1") || !strings.Contains(got, "L") {
		t.Fatalf("expected both source upgrade and leech:\n%s", got)
	}
}

func TestConvertSnellmanToConcise_DelayedLeftLeechDoesNotFollowUnrelatedRightAction(t *testing.T) {
	input := strings.Join([]string{
		"option strict-leech\tshow history",
		"Round 3 income\tshow history",
		"Round 3, turn 2\tshow history",
		"halflings\t+6\t29 VP\t-1\t10 C\t-2\t11 W\t-1\t1 P\t\t4/2/0 PW\t\t0/0/2/1\t\tadvance dig",
		"darklings\t+8\t30 VP\t-2\t3 C\t-1\t4 W\t-2\t2 P\t\t3/1/0 PW\t\t0/3/8/0\t2 \tdig 2. build G4",
		"halflings\t-1\t28 VP\t\t10 C\t\t11 W\t\t1 P\t+2\t2/4/0 PW\t\t0/0/2/1\t\tLeech 2 from darklings",
		"dwarves\t+2\t39 VP\t-2\t4 C\t-1\t2 W\t\t1 P\t\t1/9/0 PW\t\t3/0/5/0\t2 \tbuild D7",
		"mermaids\t\t21 VP\t+7\t19 C\t\t3 W\t\t1 P\t-8\t4/2/0 PW\t\t0/4/6/0\t\taction ACT4",
		"halflings\t-1\t27 VP\t\t10 C\t\t11 W\t\t1 P\t+2\t0/6/0 PW\t\t0/0/2/1\t\tLeech 2 from dwarves",
	}, "\n")

	got, err := ConvertSnellmanToConcise(input)
	if err != nil {
		t.Fatalf("ConvertSnellmanToConcise() error = %v", err)
	}

	if strings.Contains(got, "G4           |              | ACT4") && strings.Contains(got, "L            |              |              |") {
		t.Fatalf("expected left-side delayed leech to avoid appearing after unrelated right-side action:\n%s", got)
	}
	if !strings.Contains(got, "G4") || !strings.Contains(got, "D7") || !strings.Contains(got, "L") {
		t.Fatalf("expected key actions and leeches present:\n%s", got)
	}
}
