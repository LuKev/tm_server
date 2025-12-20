package notation

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// GenerateConciseLog generates a concise log string from a list of LogItems
func GenerateConciseLog(items []LogItem) string {
	var sb strings.Builder

	// Initial factions (will be updated by RoundStartItem)
	factions := make([]string, 0)
	factionSet := make(map[string]bool)

	// First pass: find all factions involved to establish initial set
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			pID := actionItem.Action.GetPlayerID()
			if pID != "" && !factionSet[pID] {
				factionSet[pID] = true
				factions = append(factions, pID)
			}
		} else if roundItem, ok := item.(RoundStartItem); ok {
			for _, f := range roundItem.TurnOrder {
				if !factionSet[f] {
					factionSet[f] = true
					factions = append(factions, f)
				}
			}
		}
	}

	// Map faction -> column index
	colMap := make(map[string]int)
	for i, f := range factions {
		colMap[f] = i
	}

	numCols := len(factions)
	if numCols == 0 {
		return ""
	}

	// Helper to write row
	writeRow := func(row []string, currentFactions []string) {
		for i, cell := range row {
			// Pad cell to column width (e.g. 15 chars)
			sb.WriteString(padRight(cell, 15))
			if i < len(row)-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString("\n")
	}

	// Process items
	currentRow := make([]string, numCols)
	headerPrinted := false
	// Helper to flush current row
	flush := func() {
		writeRow(currentRow, factions)
		currentRow = make([]string, numCols)
	}

	var lastAction game.Action
	var lastPlayerID string
	var lastCol int = -1

	// Helper to write header
	writeHeader := func(factions []string) {
		writeRow(factions, factions)
		sep := strings.Repeat("-", 15*numCols+3*(numCols-1))
		sb.WriteString(sep + "\n")
	}

	for _, item := range items {
		switch v := item.(type) {
		case GameSettingsItem:
			// Print settings
			for k, val := range v.Settings {
				sb.WriteString(fmt.Sprintf("%s: %s\n", k, val))
			}
			sb.WriteString("\n")

		case RoundStartItem:
			// Flush any pending row from previous round
			if !isRowEmpty(currentRow) {
				flush()
			}

			// Update factions list based on new turn order
			factions = v.TurnOrder
			// Rebuild colMap
			colMap = make(map[string]int)
			for i, f := range factions {
				colMap[f] = i
			}
			numCols = len(factions)
			currentRow = make([]string, numCols)

			sb.WriteString(fmt.Sprintf("Round %d\n", v.Round))
			sb.WriteString(fmt.Sprintf("TurnOrder: %s\n", strings.Join(v.TurnOrder, ", ")))
			writeHeader(factions)
			headerPrinted = true // Header is now printed for this round

			// Reset last state for new round
			lastAction = nil
			lastPlayerID = ""
			lastCol = -1

		case ActionItem:
			action := v.Action
			pID := action.GetPlayerID()

			// Ensure we have a header printed (for Setup phase)
			if !headerPrinted {
				writeHeader(factions)
				headerPrinted = true
			}

			col, ok := colMap[pID]
			if !ok {
				// Should not happen if factions map is correct
				continue
			}

			code := generateActionCode(action)

			// Check if we should chain with previous action (same player)
			if pID == lastPlayerID && lastAction != nil && shouldChain(lastAction, action) {
				// Append to current cell
				if currentRow[col] == "" {
					currentRow[col] = code
				} else {
					currentRow[col] += "." + code
				}
			} else {
				// New cell logic
				// Flush if:
				// 1. Cell is already occupied (and we didn't chain)
				// 2. OR we are backtracking (col <= lastCol)

				mustFlush := false
				if currentRow[col] != "" {
					mustFlush = true
				} else if col <= lastCol {
					mustFlush = true
				}

				if mustFlush {
					flush()
					lastCol = -1 // Reset lastCol on flush
				}

				currentRow[col] = code
			}

			lastPlayerID = pID
			lastCol = col
			lastAction = action
		}
	}

	// Flush final row
	if !isRowEmpty(currentRow) {
		writeRow(currentRow, factions)
	}

	return sb.String()
}

func isRowEmpty(row []string) bool {
	for _, s := range row {
		if s != "" {
			return false
		}
	}
	return true
}

func padRight(s string, width int) string {
	if len(s) < width {
		return s + strings.Repeat(" ", width-len(s))
	}
	return s // Or truncate?
}

func generateActionCode(action game.Action) string {
	switch a := action.(type) {
	case *game.SetupDwellingAction:
		return fmt.Sprintf("S-%s", HexToShortString(a.Hex))
	case *game.TransformAndBuildAction:
		// Simple representation: "Transform [Hex]" or "Build [Hex]"
		if a.BuildDwelling {
			return fmt.Sprintf("%s", HexToShortString(a.TargetHex))
		}
		return fmt.Sprintf("T-%s", HexToShortString(a.TargetHex))
	case *game.UpgradeBuildingAction:
		// UP-TH-C4
		shortType := getBuildingShortCode(a.NewBuildingType)
		return fmt.Sprintf("UP-%s-%s", shortType, HexToShortString(a.TargetHex))
	case *game.PassAction:
		return "PASS"
	case *game.SendPriestToCultAction:
		// ->F
		trackCode := getCultShortCode(a.Track)
		return fmt.Sprintf("->%s", trackCode)
	case *game.AdvanceShippingAction:
		return "+SHIP"
	case *game.AcceptPowerLeechAction:
		return "L" // Standard action doesn't have amount
	case *game.DeclinePowerLeechAction:
		return "DL"
	case *LogAcceptLeechAction:
		return "L" // Just "L", amount is implicit or lost in concise notation
	case *LogPowerAction:
		return a.ActionCode
	case *game.AdvanceDiggingAction:
		return "+DIG"
	case *LogBurnAction:
		return fmt.Sprintf("BURN%d", a.Amount)
	case *LogFavorTileAction:
		return a.Tile
	case *LogSpecialAction:
		return a.ActionCode
	default:
		return fmt.Sprintf("UNKNOWN(%T)", action)
	}
}

func shouldChain(prev, curr game.Action) bool {
	// Don't chain if previous action was Leech or Decline Leech
	// User wants Leech to be a separate action
	switch prev.(type) {
	case *LogAcceptLeechAction, *game.AcceptPowerLeechAction:
		return false
	case *game.DeclinePowerLeechAction:
		return false
	}

	// Don't chain if current action is Leech or Decline Leech
	switch curr.(type) {
	case *LogAcceptLeechAction, *game.AcceptPowerLeechAction:
		return false
	case *game.DeclinePowerLeechAction:
		return false
	}

	// Allow chaining for Burn actions (e.g., B3.ACT2)
	// Default chaining logic for other actions
	return true
}

// HexToShortString converts Hex{Q:3, R:5} to "F3"
// Uses logic reversed from replay.ConvertLogCoordToAxial
func HexToShortString(h board.Hex) string {
	// Row: 0=A, 1=B...
	rowChar := rune('A' + h.R)

	// Get terrain layout to skip river hexes
	layout := board.BaseGameTerrainLayout()

	// Start Q for this row
	startQ := -h.R / 2

	count := 0
	// Iterate from start of row up to our hex
	for q := startQ; q <= h.Q; q++ {
		curr := board.NewHex(q, h.R)
		terrain, exists := layout[curr]
		if !exists {
			// Should not happen if h is valid
			break
		}
		if terrain != models.TerrainRiver {
			count++
		}
	}

	return fmt.Sprintf("%c%d", rowChar, count)
}

func getBuildingShortCode(t models.BuildingType) string {
	switch t {
	case models.BuildingDwelling:
		return "D"
	case models.BuildingTradingHouse:
		return "TH"
	case models.BuildingTemple:
		return "TE"
	case models.BuildingSanctuary:
		return "SA"
	case models.BuildingStronghold:
		return "SH"
	}
	return "?"
}

func getCultShortCode(t game.CultTrack) string {
	switch t {
	case game.CultFire:
		return "F"
	case game.CultWater:
		return "W"
	case game.CultEarth:
		return "E"
	case game.CultAir:
		return "A"
	}
	return "?"
}
