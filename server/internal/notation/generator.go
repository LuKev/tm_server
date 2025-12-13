package notation

import (
	"fmt"
	"strings"
)

// GenerateConciseLog generates the concise log string from a Log struct
func GenerateConciseLog(log *Log) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Game: %s\n", log.MapName))
	if len(log.ScoringTiles) > 0 {
		sb.WriteString(fmt.Sprintf("ScoringTiles: %s\n", strings.Join(log.ScoringTiles, ", ")))
	}
	if len(log.BonusCards) > 0 {
		sb.WriteString(fmt.Sprintf("BonusCards: %s\n", strings.Join(log.BonusCards, ", ")))
	}
	if len(log.Options) > 0 {
		sb.WriteString(fmt.Sprintf("Options: %s\n", strings.Join(log.Options, ", ")))
	}
	sb.WriteString("\n")

	// Rounds
	for _, round := range log.Rounds {
		sb.WriteString(fmt.Sprintf("Round %d\n", round.RoundNumber))
		sb.WriteString(fmt.Sprintf("TurnOrder: %s\n", strings.Join(round.TurnOrder, ", ")))
		sb.WriteString(strings.Repeat("-", 64) + "\n")

		// Create grid
		// Map[Faction][Row] -> Content
		grid := make(map[string]map[int]string)
		for _, f := range round.TurnOrder {
			grid[f] = make(map[int]string)
		}

		currentRow := 0
		activeFaction := ""
		if len(round.Actions) > 0 {
			activeFaction = round.Actions[0].Faction
		}

		for _, action := range round.Actions {
			isReaction := action.Type == ActionLeech || action.Type == ActionCultReaction

			if !isReaction {
				// If active faction changed, move to next row
				if action.Faction != activeFaction {
					currentRow++
					activeFaction = action.Faction
				} else if _, exists := grid[action.Faction][currentRow]; exists {
					// If same faction already has an entry in this row, move to next row
					currentRow++
				}
			}

			// Add to grid
			content := action.String()
			if existing, ok := grid[action.Faction][currentRow]; ok {
				grid[action.Faction][currentRow] = existing + "; " + content
			} else {
				grid[action.Faction][currentRow] = content
			}
		}

		// Print grid
		// Calculate column widths (min 12 chars)
		colWidths := make(map[string]int)
		for _, f := range round.TurnOrder {
			width := 12
			if len(f) > width {
				width = len(f)
			}
			for _, rowContent := range grid[f] {
				if len(rowContent) > width {
					width = len(rowContent)
				}
			}
			colWidths[f] = width
		}

		// Print Header Row
		headerParts := []string{}
		for _, f := range round.TurnOrder {
			headerParts = append(headerParts, fmt.Sprintf("%-*s", colWidths[f], f))
		}
		sb.WriteString(strings.Join(headerParts, " | ") + "\n")
		sb.WriteString(strings.Repeat("-", 64) + "\n")

		// Print Rows
		for r := 0; r <= currentRow; r++ {
			rowParts := []string{}
			hasContent := false
			for _, f := range round.TurnOrder {
				content := grid[f][r]
				if content != "" {
					hasContent = true
				}
				rowParts = append(rowParts, fmt.Sprintf("%-*s", colWidths[f], content))
			}
			if hasContent {
				sb.WriteString(strings.Join(rowParts, " | ") + "\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
