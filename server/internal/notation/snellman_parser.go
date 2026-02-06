package notation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// IsSnellmanHTML checks if the HTML content is from Snellman (has <table id="ledger">)
func IsSnellmanHTML(htmlContent string) bool {
	return strings.Contains(htmlContent, `id="ledger"`) ||
		strings.Contains(htmlContent, "terra.snellman.net")
}

// ParseSnellmanHTML parses the raw HTML from Snellman's ledger table and converts it to text format
// The format matches BGA text format that BGAParser already handles
func ParseSnellmanHTML(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the ledger table
	ledger := doc.Find("table#ledger")
	if ledger.Length() == 0 {
		return "", fmt.Errorf("ledger table not found in HTML")
	}

	var lines []string

	// Process each row in the ledger
	ledger.Find("tr").Each(func(i int, row *goquery.Selection) {
		var cells []string

		row.Find("td").Each(func(j int, cell *goquery.Selection) {
			// Get the text content, preserving structure
			text := strings.TrimSpace(cell.Text())

			// Check if this is a header/section row (has colspan)
			if _, hasColspan := cell.Attr("colspan"); hasColspan {
				// This is likely a section header like "Round 1 income" or "Scoring FIRE cult"
				if text != "" && !strings.Contains(text, "Load full log") {
					cells = append(cells, text)
				}
				return
			}

			// Check for ledger-delta class (resource deltas like "+3", "-2")
			if cell.HasClass("ledger-delta") {
				if text != "" {
					cells = append(cells, text)
				}
				return
			}

			// Check for ledger-value class (resource values like "101 VP", "0 C")
			if cell.HasClass("ledger-value") {
				if text != "" {
					cells = append(cells, text)
				}
				return
			}

			// Regular cell (faction name or action)
			if text != "" {
				cells = append(cells, text)
			}
		})

		// Join cells with tabs to match expected format
		if len(cells) > 0 {
			line := strings.Join(cells, "\t")
			// Skip button-only lines but keep section headers
			if strings.Contains(line, "Load full log") {
				return
			}
			lines = append(lines, line)
		}
	})

	if len(lines) == 0 {
		return "", fmt.Errorf("no ledger data found in table")
	}

	// Clean up the output
	result := strings.Join(lines, "\n")

	// Fix common formatting issues
	result = cleanupSnellmanOutput(result)

	return result, nil
}

// cleanupSnellmanOutput cleans up the parsed output to match expected format
func cleanupSnellmanOutput(s string) string {
	// Remove multiple consecutive tabs
	re := regexp.MustCompile(`\t+`)
	s = re.ReplaceAllString(s, "\t")

	// Remove multiple consecutive spaces
	re = regexp.MustCompile(` +`)
	s = re.ReplaceAllString(s, " ")

	return s
}
