package notation

import (
	"strings"
	"testing"
)

func TestIsSnellmanHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "Snellman ledger table",
			html:     `<html><table id="ledger"><tr><td>test</td></tr></table></html>`,
			expected: true,
		},
		{
			name:     "Snellman URL reference",
			html:     `<html><a href="https://terra.snellman.net/game/123">link</a></html>`,
			expected: true,
		},
		{
			name:     "BGA HTML",
			html:     `<html><div id="gamelogs">test log</div></html>`,
			expected: false,
		},
		{
			name:     "Empty HTML",
			html:     `<html></html>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSnellmanHTML(tt.html)
			if result != tt.expected {
				t.Errorf("IsSnellmanHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSnellmanHTML_Simple(t *testing.T) {
	// Sample HTML mimicking Snellman ledger structure
	html := `<html>
	<table id="ledger">
		<tr>
			<td>engineers</td>
			<td class="ledger-delta">+3</td>
			<td class="ledger-value">101 VP</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 C</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 W</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 P</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">3/0/0 PW</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">2/5/8/1</td>
			<td class="ledger-delta"></td>
			<td class="ledger-delta">+3vp for WATER</td>
		</tr>
	</table>
	</html>`

	result, err := ParseSnellmanHTML(html)
	if err != nil {
		t.Fatalf("ParseSnellmanHTML() error = %v", err)
	}

	// Check that key elements are present
	if !strings.Contains(result, "engineers") {
		t.Error("Expected 'engineers' in output")
	}
	if !strings.Contains(result, "101 VP") {
		t.Error("Expected '101 VP' in output")
	}
	if !strings.Contains(result, "3/0/0 PW") {
		t.Error("Expected '3/0/0 PW' in output")
	}
}

func TestParseSnellmanHTML_SectionHeaders(t *testing.T) {
	// HTML with section headers (colspan rows)
	html := `<html>
	<table id="ledger">
		<tr>
			<td colspan="14">Scoring FIRE cult</td>
			<td><a href="/game/test">show history</a></td>
		</tr>
		<tr>
			<td>cultists</td>
			<td class="ledger-delta">+8</td>
			<td class="ledger-value">125 VP</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 C</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 W</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">0 P</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">5/0/0 PW</td>
			<td class="ledger-delta"></td>
			<td class="ledger-value">10/10/7/8</td>
			<td class="ledger-delta"></td>
			<td class="ledger-delta">+8vp for FIRE</td>
		</tr>
	</table>
	</html>`

	result, err := ParseSnellmanHTML(html)
	if err != nil {
		t.Fatalf("ParseSnellmanHTML() error = %v", err)
	}

	// Check that section header is present
	if !strings.Contains(result, "Scoring FIRE cult") {
		t.Error("Expected 'Scoring FIRE cult' section header in output")
	}

	// Check faction data
	if !strings.Contains(result, "cultists") {
		t.Error("Expected 'cultists' in output")
	}
}

func TestParseSnellmanHTML_NoLedger(t *testing.T) {
	html := `<html><body>No ledger here</body></html>`

	_, err := ParseSnellmanHTML(html)
	if err == nil {
		t.Error("Expected error when ledger table not found")
	}
}

func TestParseSnellmanHTML_SetupHeadersWithShowHistory(t *testing.T) {
	html := `<html>
	<table id="ledger">
		<tr>
			<th colspan="14">Round 1 scoring: SCORE9, TE >> 4</th>
			<th><a href="/game/test">show history</a></th>
		</tr>
		<tr>
			<td colspan="14">Removing tile BON4</td>
			<td><a href="/game/test">show history</a></td>
		</tr>
		<tr>
			<td>cultists</td>
			<td class="ledger-value">20 VP</td>
			<td>setup</td>
		</tr>
	</table>
	</html>`

	result, err := ParseSnellmanHTML(html)
	if err != nil {
		t.Fatalf("ParseSnellmanHTML() error = %v", err)
	}

	if !strings.Contains(result, "Round 1 scoring: SCORE9, TE >> 4") {
		t.Fatalf("expected round scoring line in output, got:\n%s", result)
	}
	if !strings.Contains(result, "Removing tile BON4") {
		t.Fatalf("expected removed tile line in output, got:\n%s", result)
	}
	if strings.Contains(strings.ToLower(result), "show history") {
		t.Fatalf("output should not contain 'show history', got:\n%s", result)
	}
}
