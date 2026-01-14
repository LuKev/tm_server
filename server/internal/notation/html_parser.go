package notation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseBGAHTML parses the raw HTML from BGA game logs and converts it to the concise text format
func ParseBGAHTML(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Helper to replace amount divs
	replaceAmount := func(className, unitName string) {
		doc.Find("div." + className).Each(func(i int, s *goquery.Selection) {
			amount := strings.TrimSpace(s.Text())
			text := amount
			if unitName != "" {
				text = fmt.Sprintf("%s %s", amount, unitName)
			}

			parent := s.Parent()
			if parent.HasClass("tmlogs_icon") {
				parent.ReplaceWithHtml(text)
			} else {
				s.ReplaceWithHtml(text)
			}
		})
	}

	// Replace specific amounts
	replaceAmount("workers_amount", "workers")
	replaceAmount("coins_amount", "coins")
	replaceAmount("power_amount", "power")
	replaceAmount("spade_amount", "spade(s)")
	replaceAmount("vp_amount", "VP")
	replaceAmount("priests_amount", "Priests")
	replaceAmount("cult_p_amount", "priest(s)")

	// Cult track amounts (no unit needed)
	replaceAmount("earth_amount", "")
	replaceAmount("fire_amount", "")
	replaceAmount("water_amount", "")
	replaceAmount("air_amount", "")

	// Replace terrain icons
	terrainMap := map[string]string{
		"trans_mountains": "mountains",
		"trans_forest":    "forest",
		"trans_lakes":     "lakes",
		"trans_swamp":     "swamp",
		"trans_desert":    "desert",
		"trans_plains":    "plains",
		"trans_wasteland": "wasteland",
	}

	for cls, name := range terrainMap {
		doc.Find("div." + cls).Each(func(i int, s *goquery.Selection) {
			parent := s.Parent()
			if parent.HasClass("tmlogs_icon") {
				parent.ReplaceWithHtml(name)
			} else {
				s.ReplaceWithHtml(name)
			}
		})
	}

	// General cleanup of tmlogs_icon if any remain (using title as fallback)
	doc.Find("div.tmlogs_icon").Each(func(i int, s *goquery.Selection) {
		title, exists := s.Attr("title")
		if exists {
			s.ReplaceWithHtml(title)
		}
	})

	// Extract logs
	var logs []string
	doc.Find("div.gamelogreview").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Clean up whitespace
		re := regexp.MustCompile(`\s+`)
		text = re.ReplaceAllString(text, " ")

		// Fix conversion parenthesis: "... collects: ... ) ... ..." -> "... collects: ... ... ... )"
		if strings.Contains(text, "Conversions") && strings.Contains(text, "collects:") {
			parts := strings.Split(text, "collects:")
			if len(parts) > 1 {
				suffix := parts[1]
				if strings.Contains(suffix, ")") {
					splitSuffix := strings.SplitN(suffix, ")", 2)
					preParen := splitSuffix[0]
					postParen := splitSuffix[1]

					// Check if post_paren contains resources (digits)
					if regexp.MustCompile(`\d+`).MatchString(postParen) {
						// Move parenthesis to the end
						newSuffix := preParen + postParen + ")"
						text = parts[0] + "collects:" + newSuffix
					}
				}
			}
		}

		logs = append(logs, text)
	})

	if len(logs) == 0 {
		// Fallback if no gamelogreview divs found (maybe just text?)
		return strings.TrimSpace(doc.Text()), nil
	}

	return strings.Join(logs, "\n"), nil
}
