package notation

import (
	"os"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
)

func TestBGARoundTrip(t *testing.T) {
	// 1. Read BGA Log
	content, err := os.ReadFile("../../../bga_log.txt")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// 2. Parse BGA Log -> Log Items
	bgaParser := NewBGAParser(string(content))
	logItems, err := bgaParser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse BGA log: %v", err)
	}
	t.Logf("Parsed %d log items", len(logItems))

	// Extract actions for comparison
	var originalActions []game.Action
	for _, item := range logItems {
		if actionItem, ok := item.(ActionItem); ok {
			originalActions = append(originalActions, actionItem.Action)
		}
	}
	t.Logf("Extracted %d original actions", len(originalActions))

	// 3. Generate Concise Log -> Text
	conciseText := GenerateConciseLog(logItems)
	t.Logf("Generated Concise Log (first 20 lines):\n%s", firstNLines(conciseText, 20))

	// Save to file for inspection
	if err := os.WriteFile("concise_log.txt", []byte(conciseText), 0644); err != nil {
		t.Fatalf("Failed to write concise log file: %v", err)
	}
	t.Logf("Saved concise log to concise_log.txt")

	// 4. Parse Concise Log -> Reconstructed Actions
	// Note: ParseConciseLog currently returns []game.Action.
	// It needs to be updated to return []LogItem or we just compare actions.
	// For now, let's assume it returns []game.Action and ignores headers.
	reconstructedActions, err := ParseConciseLog(conciseText)
	if err != nil {
		t.Fatalf("Failed to parse concise log: %v", err)
	}
	t.Logf("Parsed %d reconstructed actions", len(reconstructedActions))

	// 5. Compare
	if len(originalActions) != len(reconstructedActions) {
		t.Fatalf("Action count mismatch: Original=%d, Reconstructed=%d", len(originalActions), len(reconstructedActions))
	}

	for i, orig := range originalActions {
		recon := reconstructedActions[i]

		// Compare types
		if orig.GetType() != recon.GetType() {
			t.Errorf("Action %d type mismatch: Original=%v, Reconstructed=%v", i, orig.GetType(), recon.GetType())
			continue
		}

		// Compare PlayerID
		if orig.GetPlayerID() != recon.GetPlayerID() {
			t.Errorf("Action %d player mismatch: Original=%s, Reconstructed=%s", i, orig.GetPlayerID(), recon.GetPlayerID())
		}

		// Compare specifics based on type
		switch o := orig.(type) {
		case *game.SetupDwellingAction:
			r, ok := recon.(*game.SetupDwellingAction)
			// Note: Generator maps SetupDwelling -> [Coord] -> TransformAndBuild(Build=true)
			// So type mismatch might happen if we don't handle ambiguity.
			// Let's see if it fails.
			if !ok {
				// Allow TransformAndBuild if it matches
				if r2, ok2 := recon.(*game.TransformAndBuildAction); ok2 {
					if o.Hex != r2.TargetHex {
						t.Errorf("Action %d hex mismatch: %v vs %v", i, o.Hex, r2.TargetHex)
					}
				} else {
					t.Errorf("Action %d type mismatch for SetupDwelling", i)
				}
			} else {
				if o.Hex != r.Hex {
					t.Errorf("Action %d hex mismatch: %v vs %v", i, o.Hex, r.Hex)
				}
			}
		case *game.TransformAndBuildAction:
			r, ok := recon.(*game.TransformAndBuildAction)
			if !ok {
				t.Errorf("Action %d type cast failed", i)
				continue
			}
			if o.TargetHex != r.TargetHex {
				t.Errorf("Action %d hex mismatch: %v vs %v", i, o.TargetHex, r.TargetHex)
			}
			if o.BuildDwelling != r.BuildDwelling {
				t.Errorf("Action %d BuildDwelling mismatch: %v vs %v", i, o.BuildDwelling, r.BuildDwelling)
			}
			// Add more comparisons as needed
		}
	}
}

func firstNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		return strings.Join(lines[:n], "\n") + "\n..."
	}
	return s
}
