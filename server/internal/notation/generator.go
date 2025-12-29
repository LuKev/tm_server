package notation

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// GenerateConciseLog generates a concise log string from a list of LogItems
// Returns the log as a list of strings (lines) and a mapping from item index to LogLocation
func GenerateConciseLog(items []LogItem) ([]string, []LogLocation) {
	lines := make([]string, 0)
	itemLocations := make([]LogLocation, len(items))

	// Initial factions (will be updated by RoundStartItem)
	factionNames := make([]string, 0)
	factionSet := make(map[string]bool)

	// First pass: find all factions involved to establish initial set
	for _, item := range items {
		if actionItem, ok := item.(ActionItem); ok {
			pID := actionItem.Action.GetPlayerID()
			if pID != "" && !factionSet[pID] {
				factionSet[pID] = true
				factionNames = append(factionNames, pID)
			}
		} else if roundItem, ok := item.(RoundStartItem); ok {
			for _, f := range roundItem.TurnOrder {
				if !factionSet[f] {
					factionSet[f] = true
					factionNames = append(factionNames, f)
				}
			}
		}
	}

	// Map faction -> column index
	colMap := make(map[string]int)
	for i, f := range factionNames {
		colMap[f] = i
	}

	numCols := len(factionNames)
	if numCols == 0 {
		return []string{}, []LogLocation{}
	}

	// Helper to write row
	writeRow := func(row []string, currentFactions []string) {
		var sb strings.Builder
		for i, cell := range row {
			// Pad cell to column width (e.g. 15 chars)
			sb.WriteString(padRight(cell, 15))
			if i < len(row)-1 {
				sb.WriteString(" | ")
			}
		}
		lines = append(lines, sb.String())
	}

	// Process items
	currentRow := make([]string, numCols)
	headerPrinted := false
	// Helper to flush current row
	flush := func() {
		writeRow(currentRow, factionNames)
		currentRow = make([]string, numCols)
	}

	var lastAction game.Action
	var lastPlayerID string
	var lastCol int = -1

	// Helper to write header
	writeHeader := func(currentFactions []string) {
		writeRow(currentFactions, currentFactions)
		sep := strings.Repeat("-", 15*numCols+3*(numCols-1))
		lines = append(lines, sep)
	}

	for k, item := range items {
		// Default location (e.g. for non-visual items)
		itemLocations[k] = LogLocation{LineIndex: -1, ColumnIndex: -1}

		switch v := item.(type) {
		case GameSettingsItem:
			// Print settings
			itemLocations[k] = LogLocation{LineIndex: len(lines), ColumnIndex: 0}
			for key, val := range v.Settings {
				lines = append(lines, fmt.Sprintf("%s: %s", key, val))
			}
			lines = append(lines, "") // Empty line

		case RoundStartItem:
			// Flush any pending row from previous round
			if !isRowEmpty(currentRow) {
				flush()
			}

			itemLocations[k] = LogLocation{LineIndex: len(lines), ColumnIndex: 0}

			// Update factions list based on new turn order
			factionNames = v.TurnOrder
			// Rebuild colMap
			colMap = make(map[string]int)
			for i, f := range factionNames {
				colMap[f] = i
			}
			numCols = len(factionNames)
			currentRow = make([]string, numCols)

			lines = append(lines, fmt.Sprintf("Round %d", v.Round))
			lines = append(lines, fmt.Sprintf("TurnOrder: %s", strings.Join(v.TurnOrder, ", ")))
			writeHeader(factionNames)
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
				writeHeader(factionNames)
				headerPrinted = true
			}

			col, ok := colMap[pID]
			if !ok {
				// Should not happen if factions map is correct
				continue
			}

			// Get player's home terrain if available
			var homeTerrain models.TerrainType = models.TerrainTypeUnknown
			// pID is the faction name (e.g. "Nomads")
			faction := factions.NewFaction(models.FactionTypeFromString(pID))
			if faction != nil {
				homeTerrain = faction.GetHomeTerrain()
			}

			code := generateActionCode(action, homeTerrain)

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

			// Record location
			// The action is in currentRow, which will be written at len(lines)
			itemLocations[k] = LogLocation{LineIndex: len(lines), ColumnIndex: col}

			lastPlayerID = pID
			lastCol = col
			lastAction = action
		}
	}

	// Flush final row
	if !isRowEmpty(currentRow) {
		writeRow(currentRow, factionNames)
	}

	return lines, itemLocations
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

func generateActionCode(action game.Action, homeTerrain models.TerrainType) string {
	switch a := action.(type) {
	case *game.SetupDwellingAction:
		return fmt.Sprintf("S-%s", HexToShortString(a.Hex))
	case *game.TransformAndBuildAction:
		// Simple representation: "Transform [Hex]" or "Build [Hex]"
		if a.BuildDwelling {
			return fmt.Sprintf("%s", HexToShortString(a.TargetHex))
		}
		if a.TargetTerrain != models.TerrainTypeUnknown {
			// If target terrain is same as home terrain, omit the code
			if a.TargetTerrain == homeTerrain {
				return fmt.Sprintf("T-%s", HexToShortString(a.TargetHex))
			}
			terrainCode := getTerrainShortCode(a.TargetTerrain)
			return fmt.Sprintf("T-%s-%s", HexToShortString(a.TargetHex), terrainCode)
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
		if a.SpacesToClimb > 0 {
			return fmt.Sprintf("->%s%d", trackCode, a.SpacesToClimb)
		}
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
	case *LogConversionAction:
		return generateConversionCode(a)
	case *LogTownAction:
		return fmt.Sprintf("TW%dVP", a.VP)
	case *LogCompoundAction:
		var parts []string
		for _, subAction := range a.Actions {
			parts = append(parts, generateActionCode(subAction, homeTerrain))
		}
		return strings.Join(parts, ".")
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
func getTerrainShortCode(t models.TerrainType) string {
	switch t {
	case models.TerrainPlains:
		return "Br" // Brown
	case models.TerrainSwamp:
		return "Bk" // Black
	case models.TerrainLake:
		return "Bl" // Blue
	case models.TerrainForest:
		return "G" // Green
	case models.TerrainMountain:
		return "Gy" // Gray
	case models.TerrainWasteland:
		return "R" // Red
	case models.TerrainDesert:
		return "Y" // Yellow
	}
	return "?"
}
func generateConversionCode(a *LogConversionAction) string {
	// Format: C[Cost]:[Reward]
	// Order: P, W, PW, VP, C
	costStr := formatResources(a.Cost)
	rewardStr := formatResources(a.Reward)
	return fmt.Sprintf("C%s:%s", costStr, rewardStr)
}

func formatResources(res map[models.ResourceType]int) string {
	var parts []string
	// Strict order: P, W, PW, VP, C
	if amount, ok := res[models.ResourcePriest]; ok && amount > 0 {
		parts = append(parts, fmt.Sprintf("%dP", amount))
	}
	if amount, ok := res[models.ResourceWorker]; ok && amount > 0 {
		parts = append(parts, fmt.Sprintf("%dW", amount))
	}
	if amount, ok := res[models.ResourcePower]; ok && amount > 0 {
		parts = append(parts, fmt.Sprintf("%dPW", amount))
	}
	if amount, ok := res[models.ResourceVictoryPoint]; ok && amount > 0 {
		parts = append(parts, fmt.Sprintf("%dVP", amount))
	}
	if amount, ok := res[models.ResourceCoin]; ok && amount > 0 {
		parts = append(parts, fmt.Sprintf("%dC", amount))
	}
	return strings.Join(parts, "")
}
