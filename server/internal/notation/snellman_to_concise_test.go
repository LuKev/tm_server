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
			name:     "Upgrade internal part of compound",
			input:    "convert 2PW to 2C. upgrade E9 to SH",
			expected: "C2PW:2C.UP-SH-E9",
		},
		{
			name:     "Combined Witches Ride with Conversion",
			input:    "convert 1PW to 1C. action ACTW. build H4",
			expected: "C1W:1C.ACT-SH-D-H4",
		},
		{
			name:     "Pass with Cult Advance Prefix",
			input:    "+FIRE. pass BON10",
			expected: "PASS-BON-SHIP-VP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertCompoundActionToConcise(tt.input)
			if got != tt.expected {
				t.Errorf("convertCompoundActionToConcise() = %v, want %v", got, tt.expected)
			}
		})
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

	if !strings.Contains(got, "UP-TH-E6     |              | L            | L") {
		t.Fatalf("expected cultists row with delayed leeches from engineers+auren:\n%s", got)
	}
	if !strings.Contains(got, "             | UP-TH-G5     |              |") {
		t.Fatalf("expected darklings action on following row:\n%s", got)
	}
}
