package notation

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

type logTableCell struct {
	token         string
	itemIndices   []int
	sourceFaction string
}

type logTableRow []logTableCell

type logTableEvent struct {
	col     int
	faction string
	cell    logTableCell
}

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

	writeStringRow := func(row []string) {
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

	currentRows := make([]logTableRow, 0)
	currentRow := newEmptyLogRow(numCols)
	headerPrinted := false

	var lastAction game.Action
	var lastPlayerID string
	var lastCol int = -1

	// Helper to write header
	writeHeader := func(currentFactions []string) {
		writeStringRow(currentFactions)
		sep := strings.Repeat("-", 15*numCols+3*(numCols-1))
		lines = append(lines, sep)
	}

	finalizeRows := func() {
		if !isLogTableRowEmpty(currentRow) {
			currentRows = append(currentRows, cloneLogTableRow(currentRow))
			currentRow = newEmptyLogRow(numCols)
		}
		if len(currentRows) == 0 {
			return
		}

		events := flattenLogTableEvents(currentRows, factionNames)
		hasAnchoredLeech := false
		for _, ev := range events {
			if isLeechOrDeclineToken(strings.TrimSpace(ev.cell.token)) && strings.TrimSpace(ev.cell.sourceFaction) != "" {
				hasAnchoredLeech = true
				break
			}
		}
		if hasAnchoredLeech {
			events = enforceLeechSourceOrderForEvents(events)
			currentRows = rebuildLogRowsFromEvents(events, factionNames)
		}

		for _, row := range currentRows {
			display := make([]string, len(row))
			for i := range row {
				display[i] = row[i].token
			}
			writeStringRow(display)
			lineIndex := len(lines) - 1
			for col := range row {
				for _, itemIdx := range row[col].itemIndices {
					if itemIdx >= 0 && itemIdx < len(itemLocations) {
						itemLocations[itemIdx] = LogLocation{LineIndex: lineIndex, ColumnIndex: col}
					}
				}
			}
		}

		currentRows = make([]logTableRow, 0)
		currentRow = newEmptyLogRow(numCols)
	}

	for k, item := range items {
		// Default location (e.g. for non-visual items)
		itemLocations[k] = LogLocation{LineIndex: -1, ColumnIndex: -1}

		switch v := item.(type) {
		case GameSettingsItem:
			finalizeRows()
			// Print settings
			itemLocations[k] = LogLocation{LineIndex: len(lines), ColumnIndex: 0}
			for key, val := range v.Settings {
				if !strings.HasPrefix(key, "StartingVP:") {
					lines = append(lines, fmt.Sprintf("%s: %s", key, val))
				}
			}
			// Add StartingVPs line if present
			var vpParts []string
			// We need to iterate over settings again or store them. Map iteration is random.
			// Let's iterate over factionNames if available, but we don't have them yet (they come from RoundStart or Action).
			// Actually, we can just iterate the map and filter.
			for key, val := range v.Settings {
				if strings.HasPrefix(key, "StartingVP:") {
					faction := strings.TrimPrefix(key, "StartingVP:")
					vpParts = append(vpParts, fmt.Sprintf("%s:%s", faction, val))
				}
			}
			if len(vpParts) > 0 {
				lines = append(lines, fmt.Sprintf("StartingVPs: %s", strings.Join(vpParts, ", ")))
			}
			lines = append(lines, "") // Empty line

		case RoundStartItem:
			finalizeRows()

			itemLocations[k] = LogLocation{LineIndex: len(lines), ColumnIndex: 0}

			// Update factions list based on new turn order
			factionNames = v.TurnOrder
			// Rebuild colMap
			colMap = make(map[string]int)
			for i, f := range factionNames {
				colMap[f] = i
			}
			numCols = len(factionNames)
			currentRows = make([]logTableRow, 0)
			currentRow = newEmptyLogRow(numCols)

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
			var homeTerrain = models.TerrainTypeUnknown
			// pID is the faction name (e.g. "Nomads")
			faction := factions.NewFaction(models.FactionTypeFromString(pID))
			if faction != nil {
				homeTerrain = faction.GetHomeTerrain()
			}

			code := generateActionCode(action, homeTerrain)
			leechSource := extractLeechSourceFromAction(action)

			// Check if we should chain with previous action (same player)
			if pID == lastPlayerID && lastAction != nil && shouldChain(lastAction, action) {
				// Append to current cell
				if currentRow[col].token == "" {
					currentRow[col].token = code
				} else {
					currentRow[col].token += "." + code
				}
				currentRow[col].itemIndices = append(currentRow[col].itemIndices, k)
			} else {
				// New cell logic
				// Flush if:
				// 1. Cell is already occupied (and we didn't chain)
				// 2. OR we are backtracking (col <= lastCol)

				mustFlush := false
				if currentRow[col].token != "" {
					mustFlush = true
				} else if col <= lastCol {
					mustFlush = true
				}

				if mustFlush {
					currentRows = append(currentRows, cloneLogTableRow(currentRow))
					currentRow = newEmptyLogRow(numCols)
					lastCol = -1 // Reset lastCol on flush
				}

				currentRow[col] = logTableCell{
					token:         code,
					itemIndices:   []int{k},
					sourceFaction: leechSource,
				}
			}

			lastPlayerID = pID
			lastCol = col
			lastAction = action
		}
	}

	finalizeRows()

	return lines, itemLocations
}

func newEmptyLogRow(numCols int) logTableRow {
	row := make(logTableRow, numCols)
	for i := range row {
		row[i] = logTableCell{}
	}
	return row
}

func cloneLogTableRow(row logTableRow) logTableRow {
	out := make(logTableRow, len(row))
	for i := range row {
		out[i].token = row[i].token
		out[i].sourceFaction = row[i].sourceFaction
		if len(row[i].itemIndices) > 0 {
			out[i].itemIndices = append([]int(nil), row[i].itemIndices...)
		}
	}
	return out
}

func isLogTableRowEmpty(row logTableRow) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell.token) != "" {
			return false
		}
	}
	return true
}

func extractLeechSourceFromAction(action game.Action) string {
	switch a := action.(type) {
	case *LogAcceptLeechAction:
		return strings.TrimSpace(a.FromPlayerID)
	case *LogDeclineLeechAction:
		return strings.TrimSpace(a.FromPlayerID)
	default:
		return ""
	}
}

func flattenLogTableEvents(rows []logTableRow, factions []string) []logTableEvent {
	events := make([]logTableEvent, 0)
	for r := 0; r < len(rows); r++ {
		for c := 0; c < len(factions) && c < len(rows[r]); c++ {
			tok := strings.TrimSpace(rows[r][c].token)
			if tok == "" {
				continue
			}
			events = append(events, logTableEvent{
				col:     c,
				faction: factions[c],
				cell:    rows[r][c],
			})
		}
	}
	return events
}

func enforceLeechSourceOrderForEvents(events []logTableEvent) []logTableEvent {
	if len(events) == 0 {
		return events
	}
	maxIters := len(events)*4 + 16
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		lastReqByReactor := make(map[string]int)
		for i := 0; i < len(events); i++ {
			ev := events[i]
			if !isLeechOrDeclineToken(strings.TrimSpace(ev.cell.token)) {
				continue
			}
			sourceFaction := normalizeFactionNameForMatching(ev.cell.sourceFaction)
			if sourceFaction == "" {
				continue
			}

			req := -1
			for j := i - 1; j >= 0; j-- {
				tok := strings.TrimSpace(events[j].cell.token)
				if tok == "" || isLeechOrDeclineToken(tok) {
					continue
				}
				if normalizeFactionNameForMatching(events[j].faction) == sourceFaction && actionMayTriggerLeech(tok) {
					req = j
					break
				}
			}
			if req < 0 {
				for j := i + 1; j < len(events); j++ {
					tok := strings.TrimSpace(events[j].cell.token)
					if tok == "" || isLeechOrDeclineToken(tok) {
						continue
					}
					if normalizeFactionNameForMatching(events[j].faction) == sourceFaction && actionMayTriggerLeech(tok) {
						req = j
						break
					}
				}
			}
			if req < 0 {
				continue
			}
			if prevReq, ok := lastReqByReactor[ev.faction]; ok && prevReq == req {
				for j := req + 1; j < len(events); j++ {
					tok := strings.TrimSpace(events[j].cell.token)
					if tok == "" || isLeechOrDeclineToken(tok) {
						continue
					}
					if normalizeFactionNameForMatching(events[j].faction) == sourceFaction && actionMayTriggerLeech(tok) {
						req = j
						break
					}
				}
			}

			prev := -1
			for j := i - 1; j >= 0; j-- {
				tok := strings.TrimSpace(events[j].cell.token)
				if tok == "" || isLeechOrDeclineToken(tok) {
					continue
				}
				prev = j
				break
			}
			if prev == req {
				lastReqByReactor[ev.faction] = req
				continue
			}

			moved := ev
			events = append(events[:i], events[i+1:]...)
			insertAt := req + 1
			if i < insertAt {
				insertAt--
			}
			events = append(events, logTableEvent{})
			copy(events[insertAt+1:], events[insertAt:])
			events[insertAt] = moved
			lastReqByReactor[ev.faction] = req
			changed = true
			break
		}
		if !changed {
			return events
		}
	}
	return events
}

func rebuildLogRowsFromEvents(events []logTableEvent, factions []string) []logTableRow {
	if len(events) == 0 {
		return nil
	}
	width := len(factions)
	pos := -1
	rows := make([]logTableRow, 0, len(events))
	for _, ev := range events {
		for {
			pos++
			if pos%width == ev.col {
				break
			}
		}
		rowIdx := pos / width
		for len(rows) <= rowIdx {
			rows = append(rows, newEmptyLogRow(width))
		}
		rows[rowIdx][ev.col] = ev.cell
	}
	compacted := make([]logTableRow, 0, len(rows))
	for _, row := range rows {
		if !isLogTableRowEmpty(row) {
			compacted = append(compacted, row)
		}
	}
	return compacted
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
			return HexToShortString(a.TargetHex)
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
		if a.BonusCard != nil {
			return fmt.Sprintf("PASS-%s", getBonusCardShortCode(*a.BonusCard))
		}
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
	case *LogDeclineLeechAction:
		return "DL"
	case *LogPowerAction:
		return a.ActionCode
	case *game.AdvanceDiggingAction:
		return "+DIG"
	case *LogBurnAction:
		return fmt.Sprintf("BURN%d", a.Amount)
	case *LogDigTransformAction:
		// DIGn-<coord> is an internal token emitted by the Snellman converter to preserve
		// intra-row ordering for "dig" steps (notably when conversions are interleaved).
		return fmt.Sprintf("DIG%d-%s", a.Spades, HexToShortString(a.Target))
	case *LogPostIncomeAction:
		// Post-income wrapper: show inner token prefixed with '^' for debugging/inspection.
		if a.Action == nil {
			return "^<nil>"
		}
		return "^" + generateActionCode(a.Action, homeTerrain)
	case *LogFavorTileAction:
		return a.Tile
	case *LogSpecialAction:
		return a.ActionCode
	case *LogConversionAction:
		return generateConversionCode(a)
	case *LogTownAction:
		return fmt.Sprintf("TW%dVP", a.VP)
	case *LogBonusCardSelectionAction:
		return a.BonusCard // Already in short code format e.g. BON1
	case *LogCompoundAction:
		var parts []string
		for _, subAction := range a.Actions {
			parts = append(parts, generateActionCode(subAction, homeTerrain))
		}
		return strings.Join(parts, ".")
	case *LogHalflingsSpadeAction:
		// Generate T-[Coord]-[Terrain] for each transform (terrain omitted if home terrain)
		var parts []string
		for i, coord := range a.TransformCoords {
			part := "T-" + coord
			// Add terrain code if we have it and it's not home terrain (plains for Halflings)
			if i < len(a.TargetTerrains) && a.TargetTerrains[i] != "" {
				terrainCode := getTerrainCodeFromName(a.TargetTerrains[i])
				if terrainCode != "" && terrainCode != "K" { // K = Plains = Halflings home
					part += "-" + terrainCode
				}
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, ".")
	case *LogCultistAdvanceAction:
		trackCode := "F"
		switch a.Track {
		case game.CultWater:
			trackCode = "W"
		case game.CultEarth:
			trackCode = "E"
		case game.CultAir:
			trackCode = "A"
		}
		return fmt.Sprintf("CULT-%s", trackCode)
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
	case *LogDeclineLeechAction:
		return false
	case *game.DeclinePowerLeechAction:
		return false
	}

	// Don't chain if current action is Leech or Decline Leech
	switch curr.(type) {
	case *LogAcceptLeechAction, *game.AcceptPowerLeechAction:
		return false
	case *LogDeclineLeechAction:
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

func getBonusCardShortCode(t game.BonusCardType) string {
	switch t {
	case game.BonusCardSpade:
		return "BON-SPD"
	case game.BonusCardCultAdvance:
		return "BON-4C"
	case game.BonusCard6Coins:
		return "BON-6C"
	case game.BonusCardShipping:
		return "BON-SHIP"
	case game.BonusCardWorkerPower:
		return "BON-WP"
	case game.BonusCardTradingHouseVP:
		return "BON-TP"
	case game.BonusCardStrongholdSanctuary:
		return "BON-BB"
	case game.BonusCardPriest:
		return "BON-P"
	case game.BonusCardDwellingVP:
		return "BON-DW"
	case game.BonusCardShippingVP:
		return "BON-SHIP-VP"
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

// getTerrainCodeFromName converts terrain name to single-letter code
func getTerrainCodeFromName(name string) string {
	switch strings.ToLower(name) {
	case "plains":
		return "K" // Brown
	case "swamp":
		return "S" // Black
	case "lakes", "lake":
		return "U" // Blue
	case "forest":
		return "G" // Green
	case "mountains", "mountain":
		return "X" // Grey
	case "wasteland":
		return "R" // Red
	case "desert":
		return "Y" // Yellow
	}
	return ""
}
