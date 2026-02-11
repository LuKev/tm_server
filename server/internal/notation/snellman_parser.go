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
		// Setup/event rows are often rendered as single header cells with colspan.
		row.Find("th[colspan], td[colspan]").EachWithBreak(func(_ int, cell *goquery.Selection) bool {
			text := normalizeSnellmanCellText(cell.Text())
			if text != "" {
				lines = append(lines, text)
			}
			return false
		})

		var cells []string

		row.Find("th, td").Each(func(j int, cell *goquery.Selection) {
			// Section/header rows are handled above.
			if _, hasColspan := cell.Attr("colspan"); hasColspan {
				return
			}

			// Get the text content, preserving structure
			text := normalizeSnellmanCellText(cell.Text())
			if text == "" {
				return
			}

			// Check for ledger-delta class (resource deltas like "+3", "-2")
			if cell.HasClass("ledger-delta") {
				cells = append(cells, text)
				return
			}

			// Check for ledger-value class (resource values like "101 VP", "0 C")
			if cell.HasClass("ledger-value") {
				cells = append(cells, text)
				return
			}

			// Regular cell (faction name or action)
			cells = append(cells, text)
		})

		// Join cells with tabs to match expected format
		if len(cells) > 0 {
			line := strings.Join(cells, "\t")
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

func normalizeSnellmanCellText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}

	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")

	// Snellman rows often include a trailing "show history" control.
	showHistory := regexp.MustCompile(`(?i)\s*show history\s*$`)
	text = showHistory.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	if strings.EqualFold(text, "Load full log") {
		return ""
	}
	return text
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
