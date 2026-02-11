package notation

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

type snellmanRoundData struct {
	Number    int
	Rows      []map[string]string // chronological rows; faction -> action in that row
	TurnOrder []string
	PassOrder []string // Order factions passed (for next round's turn order)
}

// IsSnellmanTextFormat checks if the content is Snellman's tab-delimited ledger format
func IsSnellmanTextFormat(content string) bool {
	lines := strings.Split(content, "\n")
	if len(lines) < 5 {
		return false
	}

	// Check for characteristic Snellman patterns
	hasSnellmanHeaders := false
	hasFactionLines := false
	factionNames := []string{"engineers", "darklings", "cultists", "witches", "halflings", "auren",
		"alchemists", "chaos magicians", "nomads", "fakirs", "giants", "dwarves",
		"mermaids", "swarmlings"}

	for _, line := range lines[:min(20, len(lines))] {
		lower := strings.ToLower(line)

		// Check for Snellman-specific headers
		if strings.Contains(lower, "option strict-leech") ||
			strings.Contains(lower, "default game options") ||
			strings.Contains(lower, "randomize setup") {
			hasSnellmanHeaders = true
		}

		// Check for faction lines with tab-separated data
		for _, faction := range factionNames {
			if strings.HasPrefix(lower, faction+"\t") && strings.Contains(line, "VP") {
				hasFactionLines = true
				break
			}
		}
	}

	return hasSnellmanHeaders || hasFactionLines
}

// ConvertSnellmanToConcise converts Snellman's tab-delimited ledger format to Concise Notation format
func ConvertSnellmanToConcise(content string) (string, error) {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	// State tracking
	var scoringTiles []string
	var removedBonusCards []string
	var factions []string
	factionVPs := make(map[string]int)
	maintainPlayerOrder := false
	variableTurnOrder := false

	// Round tracking
	var rounds []*snellmanRoundData
	var currentRound *snellmanRoundData
	roundLeechAnchors := make(map[*snellmanRoundData]map[int]map[string]string)

	inSetupPhase := true
	setupActions := make(map[string][]string) // faction -> setup dwelling placements
	setupPassOrder := []string{}              // Track setup phase pass order for Round 1
	actionsAddedThisTurn := make(map[string]bool)
	lastMainActionRow := make(map[string]int)
	lastLeechSourceRow := make(map[string]int)
	leechAnchorSource := make(map[int]map[string]string) // row -> reactor faction -> source faction
	lastEventRow := make(map[string]int)
	rowLockedForMain := make(map[int]bool)
	turnActionRow := -1
	turnActionCol := -1
	turnBaseRow := 0
	cultLeechPattern := regexp.MustCompile(`^\+(EARTH|WATER|FIRE|AIR)\.\s*Leech`)
	cultPassPattern := regexp.MustCompile(`(?i)^\+(EARTH|WATER|FIRE|AIR)\.\s*(pass\s+.+)$`)
	cultPrefixPattern := regexp.MustCompile(`(?i)^\+(EARTH|WATER|FIRE|AIR)\.?\s*(.*)$`)
	scoringPattern := regexp.MustCompile(`(?i)^round\s+\d+\s+scoring\b`)
	scoreCodePattern := regexp.MustCompile(`(?i)\b(SCORE\d+)\b`)
	removedBonusPattern := regexp.MustCompile(`(?i)^removing\s+(?:tile|bonus\s+tile)\s+(BON\d+)\b`)
	seenScoringTiles := make(map[string]bool)

	var savedLine string

	for {
		var line string
		if savedLine != "" {
			line = savedLine
			savedLine = ""
		} else {
			if !scanner.Scan() {
				break
			}
			line = scanner.Text()
		}

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Parse scoring tiles
		if strings.HasPrefix(strings.ToLower(line), "option ") {
			l := strings.ToLower(line)
			if strings.Contains(l, "maintain-player-order") {
				maintainPlayerOrder = true
			}
			if strings.Contains(l, "variable-turn-order") {
				variableTurnOrder = true
			}
		}

		// Parse scoring tiles.
		if scoringPattern.MatchString(line) {
			matches := scoreCodePattern.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) > 1 {
					code := strings.ToUpper(m[1])
					if !seenScoringTiles[code] {
						seenScoringTiles[code] = true
						scoringTiles = append(scoringTiles, code)
					}
				}
			}
			continue
		}

		// Parse removed bonus cards.
		if m := removedBonusPattern.FindStringSubmatch(line); len(m) > 1 {
			removedBonusCards = append(removedBonusCards, strings.ToUpper(m[1]))
			continue
		}

		// Round detection (Income)
		if strings.HasPrefix(line, "Round ") && strings.Contains(line, "income") {
			inSetupPhase = false
			re := regexp.MustCompile(`Round (\d+)`)
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				roundNum, _ := parseRound(m[1])

				// Start new round if needed
				if currentRound == nil || currentRound.Number != roundNum {
					turnOrder := computeRoundTurnOrder(rounds, factions, maintainPlayerOrder, variableTurnOrder)

					currentRound = &snellmanRoundData{
						Number:    roundNum,
						Rows:      []map[string]string{},
						TurnOrder: make([]string, len(turnOrder)),
						PassOrder: []string{},
					}
					copy(currentRound.TurnOrder, turnOrder)
					rounds = append(rounds, currentRound)
					actionsAddedThisTurn = make(map[string]bool)
					lastMainActionRow = make(map[string]int)
					lastLeechSourceRow = make(map[string]int)
					leechAnchorSource = make(map[int]map[string]string)
					roundLeechAnchors[currentRound] = leechAnchorSource
					lastEventRow = make(map[string]int)
					rowLockedForMain = make(map[int]bool)
					turnActionRow = -1
					turnActionCol = -1
					turnBaseRow = len(currentRound.Rows)
				}
			}
			inSetupPhase = false

			// Skip income lines - they are informational only, simulator calculates income
			for scanner.Scan() {
				incomeLine := scanner.Text()
				incomeLine = strings.TrimSpace(incomeLine)

				if incomeLine == "" || strings.HasPrefix(incomeLine, "Round ") || strings.HasPrefix(incomeLine, "Turn ") {
					// End of income block, save line for next iteration
					savedLine = incomeLine
					break
				}
				// Income lines are skipped - simulator handles income calculation
			}
			continue
		}

		if strings.HasPrefix(line, "Round ") && strings.Contains(line, "turn") {
			re := regexp.MustCompile(`Round (\d+)`)
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				roundNum, _ := parseRound(m[1])

				// Start new round if needed
				if currentRound == nil || currentRound.Number != roundNum {
					turnOrder := computeRoundTurnOrder(rounds, factions, maintainPlayerOrder, variableTurnOrder)

					currentRound = &snellmanRoundData{
						Number:    roundNum,
						Rows:      []map[string]string{},
						TurnOrder: make([]string, len(turnOrder)),
						PassOrder: []string{},
					}
					copy(currentRound.TurnOrder, turnOrder)
					rounds = append(rounds, currentRound)
					actionsAddedThisTurn = make(map[string]bool)
					lastMainActionRow = make(map[string]int)
					lastLeechSourceRow = make(map[string]int)
					leechAnchorSource = make(map[int]map[string]string)
					roundLeechAnchors[currentRound] = leechAnchorSource
					lastEventRow = make(map[string]int)
					rowLockedForMain = make(map[int]bool)
					turnActionRow = -1
					turnActionCol = -1
					turnBaseRow = len(currentRound.Rows)
				} else {
					// Same round, new turn: reset per-turn merge tracking.
					actionsAddedThisTurn = make(map[string]bool)
					turnActionRow = -1
					turnActionCol = -1
					turnBaseRow = len(currentRound.Rows)
				}
			}
			inSetupPhase = false
			continue
		}

		// Parse faction action lines
		parts := strings.Split(line, "\t")

		if len(parts) < 2 {
			continue
		}

		factionName := strings.ToLower(strings.TrimSpace(parts[0]))

		if !isKnownFaction(factionName) {
			continue
		}

		// Track faction if not seen
		factionSeen := false
		for _, f := range factions {
			if f == factionName {
				factionSeen = true
				break
			}
		}
		if !factionSeen {
			factions = append(factions, factionName)
		}

		// Extract action
		action := extractSnellmanAction(parts)
		if action == "" {
			continue
		}

		// SPECIAL: Handle cult bonus + leech pattern (e.g., "+EARTH. Leech 1 from engineers")
		// This is Cultists' faction power - the cult advance should be merged with their last main action,
		// while the leech should be a separate row
		if m := cultPrefixPattern.FindStringSubmatch(action); len(m) > 2 && currentRound != nil && !inSetupPhase && factionName == "cultists" {
			cultBonus := "+" + trackToShort(m[1])
			if rowIdx, ok := lastMainActionRow[factionName]; ok &&
				rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
				currentRound.Rows[rowIdx][factionName] != "" {
				currentRound.Rows[rowIdx][factionName] += "." + cultBonus
			}
			remainder := strings.TrimSpace(m[2])
			if remainder == "" {
				continue
			}
			action = remainder
		}
		if m := cultLeechPattern.FindStringSubmatch(action); len(m) > 1 && currentRound != nil && !inSetupPhase {
			cultBonus := "+" + trackToShort(m[1])

			// Backtrack to the faction's latest main action row and add the cult bonus there.
			if rowIdx, ok := lastMainActionRow[factionName]; ok &&
				rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
				currentRound.Rows[rowIdx][factionName] != "" {
				currentRound.Rows[rowIdx][factionName] += "." + cultBonus
			}

			// Preserve the source faction for row placement by keeping only the leech part.
			action = extractTrailingLeechAction(action)
		}
		// SPECIAL: Handle delayed cult bonus + pass pattern (e.g., "+EARTH. pass BON9")
		// The cult step belongs to Cultists' prior triggering action, not to the pass itself.
		if m := cultPassPattern.FindStringSubmatch(action); len(m) > 2 && currentRound != nil && !inSetupPhase {
			cultBonus := "+" + trackToShort(m[1])
			if rowIdx, ok := lastMainActionRow[factionName]; ok &&
				rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
				currentRound.Rows[rowIdx][factionName] != "" {
				currentRound.Rows[rowIdx][factionName] += "." + cultBonus
			}
			action = m[2]
		}

		// Convert action
		cultDelta := extractCultDelta(parts)
		conciseAction := convertActionToConcise(action, factionName, inSetupPhase, cultDelta)
		if conciseAction == "" {
			continue
		}

		// Store action
		if inSetupPhase {
			setupActions[factionName] = append(setupActions[factionName], conciseAction)
			// Track setup pass order (BON-* during setup = passes)
			if strings.HasPrefix(conciseAction, "BON-") {
				alreadyPassed := false
				for _, p := range setupPassOrder {
					if p == factionName {
						alreadyPassed = true
						break
					}
				}
				if !alreadyPassed {
					setupPassOrder = append(setupPassOrder, factionName)
				}
			}
		} else if currentRound != nil {
			// Check if this is a leech action - leeches should be their own row, not merged
			isLeechAction := conciseAction == "L" || conciseAction == "DL"

			if isLeechAction {
				targetRow := len(currentRound.Rows)
				sourceFaction := extractLeechSourceFaction(action)
				if sourceFaction == "" {
					return "", fmt.Errorf("unable to parse leech source faction from action %q", action)
				}
				sourceRow := findLatestEligibleLeechSourceRow(currentRound, sourceFaction, len(currentRound.Rows)-1)
				if sourceRow < 0 {
					return "", fmt.Errorf("unable to resolve leech source row for faction %q from action %q", sourceFaction, action)
				}
				targetRow = sourceRow
				cols := roundColumns(currentRound, factions)
				sourceCol := factionColumnIndex(cols, sourceFaction)
				reactorCol := factionColumnIndex(cols, factionName)
				// Preserve left-to-right chronology within a row:
				// a reaction shown left of the source action would look like it happened first.
				// In that case, force the reaction to a later row.
				if sourceCol >= 0 && reactorCol >= 0 && reactorCol < sourceCol {
					targetRow = sourceRow + 1
				}
				// If compaction placed unrelated main actions between source and leech, move them out
				// so the leech remains semantically tied to the correct source action.
				relocateInterferingActionsForLeech(
					currentRound,
					sourceRow,
					sourceFaction,
					factionName,
					rowLockedForMain,
					lastMainActionRow,
					lastLeechSourceRow,
					leechAnchorSource,
					factions,
				)
				if newSourceRow, ok := lastLeechSourceRow[sourceFaction]; ok {
					sourceRow = newSourceRow
					targetRow = sourceRow
					sourceCol = factionColumnIndex(cols, sourceFaction)
					reactorCol = factionColumnIndex(cols, factionName)
					if sourceCol >= 0 && reactorCol >= 0 && reactorCol < sourceCol {
						targetRow = sourceRow + 1
					}
				}
				// Ensure the rendered previous non-leech token for this leech is exactly its source action.
				// If not, insert a row directly after source to restore correspondence semantics.
				if sourceCol >= 0 && reactorCol >= 0 && !previousNonLeechIsSource(currentRound, cols, targetRow, reactorCol, sourceRow, sourceCol) {
					insertRoundRowAt(
						currentRound,
						sourceRow+1,
						lastMainActionRow,
						lastLeechSourceRow,
						lastEventRow,
						rowLockedForMain,
						leechAnchorSource,
					)
					targetRow = sourceRow + 1
				}
				if targetRow >= 0 && targetRow < len(currentRound.Rows) && currentRound.Rows[targetRow][factionName] != "" {
					existing := currentRound.Rows[targetRow][factionName]
					existingSource := ""
					if anchors, ok := leechAnchorSource[targetRow]; ok {
						existingSource = anchors[factionName]
					}
					// If two different source factions collide on the same reactor cell,
					// split the current source action into its own row to preserve attribution.
					if sourceRow >= 0 &&
						(existing == "L" || existing == "DL") &&
						existingSource != "" && sourceFaction != "" && existingSource != sourceFaction {
						// Only move the source action if the collision is on the exact inline source row.
						// For delayed rows (sourceRow+1 etc), just place leech on the next row.
						if targetRow == sourceRow {
							movedSourceRow := moveSourceActionAndAnchoredLeeches(
								currentRound,
								sourceFaction,
								sourceRow,
								targetRow+1,
								rowLockedForMain,
								lastMainActionRow,
								lastLeechSourceRow,
								leechAnchorSource,
							)
							targetRow = movedSourceRow
							cols := roundColumns(currentRound, factions)
							sourceCol := factionColumnIndex(cols, sourceFaction)
							reactorCol := factionColumnIndex(cols, factionName)
							if sourceCol >= 0 && reactorCol >= 0 && reactorCol < sourceCol {
								targetRow = movedSourceRow + 1
							}
						} else {
							targetRow++
						}
					} else {
						targetRow++
					}
				}
				rowIdx := placeRoundAction(currentRound, factionName, conciseAction, targetRow)
				if sourceFaction != "" {
					if _, ok := leechAnchorSource[rowIdx]; !ok {
						leechAnchorSource[rowIdx] = make(map[string]string)
					}
					leechAnchorSource[rowIdx][factionName] = sourceFaction
				}
				// Final guard: ensure row-major correspondence semantics hold in rendered output.
				// If the previous non-leech token is not the resolved source action, re-home leech
				// directly after source action.
				if sourceCol >= 0 && reactorCol >= 0 && !previousNonLeechIsSource(currentRound, cols, rowIdx, reactorCol, sourceRow, sourceCol) {
					currentRound.Rows[rowIdx][factionName] = ""
					if anchors, ok := leechAnchorSource[rowIdx]; ok {
						delete(anchors, factionName)
						if len(anchors) == 0 {
							delete(leechAnchorSource, rowIdx)
						}
					}
					insertRoundRowAt(
						currentRound,
						sourceRow+1,
						lastMainActionRow,
						lastLeechSourceRow,
						lastEventRow,
						rowLockedForMain,
						leechAnchorSource,
					)
					rowIdx = sourceRow + 1
					currentRound.Rows[rowIdx][factionName] = conciseAction
					if _, ok := leechAnchorSource[rowIdx]; !ok {
						leechAnchorSource[rowIdx] = make(map[string]string)
					}
					leechAnchorSource[rowIdx][factionName] = sourceFaction
				}
				lastEventRow[factionName] = maxInt(lastEventRow[factionName], rowIdx)
			} else if actionsAddedThisTurn[factionName] && !strings.HasPrefix(conciseAction, "PASS") {
				// Merge follow-up non-leech actions in the same turn with faction's main action.
				if rowIdx, ok := lastMainActionRow[factionName]; ok &&
					rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
					currentRound.Rows[rowIdx][factionName] != "" {
					currentRound.Rows[rowIdx][factionName] += "." + conciseAction
					lastEventRow[factionName] = maxInt(lastEventRow[factionName], rowIdx)
					turnActionRow = rowIdx
					turnActionCol = factionColumnIndex(roundColumns(currentRound, factions), factionName)
					if factionLikelyDelayedLeechSource(factionName, currentRound.Rows[rowIdx][factionName]) {
						rowLockedForMain[rowIdx] = true
					}
				} else {
					rowIdx := placeRoundAction(currentRound, factionName, conciseAction, len(currentRound.Rows))
					lastMainActionRow[factionName] = rowIdx
					if actionMayTriggerLeech(currentRound.Rows[rowIdx][factionName]) {
						lastLeechSourceRow[factionName] = rowIdx
					}
					lastEventRow[factionName] = maxInt(lastEventRow[factionName], rowIdx)
					turnActionRow = rowIdx
					turnActionCol = factionColumnIndex(roundColumns(currentRound, factions), factionName)
					if factionLikelyDelayedLeechSource(factionName, currentRound.Rows[rowIdx][factionName]) {
						rowLockedForMain[rowIdx] = true
					}
				}
			} else {
				startRow := turnBaseRow
				if prevRow, ok := lastEventRow[factionName]; ok {
					candidate := prevRow + 1
					if candidate > startRow {
						startRow = candidate
					}
				}
				if startRow < turnBaseRow {
					startRow = turnBaseRow
				}
				colIdx := factionColumnIndex(roundColumns(currentRound, factions), factionName)
				if turnActionRow >= 0 {
					if startRow < turnActionRow {
						startRow = turnActionRow
					}
					if startRow == turnActionRow && turnActionCol >= 0 && colIdx >= 0 && colIdx < turnActionCol {
						startRow = turnActionRow + 1
					}
				}
				rowIdx := placeMainAction(currentRound, factionName, conciseAction, startRow, rowLockedForMain, roundColumns(currentRound, factions))
				lastMainActionRow[factionName] = rowIdx
				if actionMayTriggerLeech(currentRound.Rows[rowIdx][factionName]) {
					lastLeechSourceRow[factionName] = rowIdx
				}
				lastEventRow[factionName] = maxInt(lastEventRow[factionName], rowIdx)
				actionsAddedThisTurn[factionName] = true
				turnActionRow = rowIdx
				turnActionCol = colIdx
				if factionLikelyDelayedLeechSource(factionName, currentRound.Rows[rowIdx][factionName]) {
					rowLockedForMain[rowIdx] = true
				}
			}

			// Track pass order: when a faction passes, add to PassOrder (if not already).
			// Snellman logs can use either "pass BONx" -> PASS-BON-* or plain "pass" -> PASS.
			if conciseAction == "PASS" || strings.Contains(conciseAction, "PASS-") {
				alreadyPassed := false
				for _, p := range currentRound.PassOrder {
					if p == factionName {
						alreadyPassed = true
						break
					}
				}
				if !alreadyPassed {
					currentRound.PassOrder = append(currentRound.PassOrder, factionName)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading content: %w", err)
	}

	// Build output
	result = append(result, "Game: Base")

	// Scoring tiles
	if len(scoringTiles) > 0 {
		result = append(result, fmt.Sprintf("ScoringTiles: %s", strings.Join(scoringTiles, ", ")))
	}

	// Bonus cards
	bonusCards := getBonusCardsMinusRemoved(removedBonusCards)
	if len(bonusCards) > 0 {
		result = append(result, fmt.Sprintf("BonusCards: %s", strings.Join(bonusCards, ", ")))
	}

	// Starting VPs
	if len(factions) > 0 {
		var vpParts []string
		for _, f := range factions {
			vp := factionVPs[f]
			if vp == 0 {
				vp = 20 // default
			}
			vpParts = append(vpParts, fmt.Sprintf("%s:%d", strings.Title(f), vp))
		}
		result = append(result, fmt.Sprintf("StartingVPs: %s", strings.Join(vpParts, ", ")))
	}

	result = append(result, "")

	// Setup phase
	if len(factions) > 0 {
		result = append(result, "Setup")
		result = append(result, fmt.Sprintf("TurnOrder: %s", formatFactionList(factions)))
		result = append(result, strings.Repeat("-", 60))
		result = append(result, formatFactionHeader(factions))
		result = append(result, strings.Repeat("-", 60))

		maxSetupRows := 0
		for _, f := range factions {
			if len(setupActions[f]) > maxSetupRows {
				maxSetupRows = len(setupActions[f])
			}
		}

		for i := 0; i < maxSetupRows; i++ {
			var row []string
			for _, f := range factions {
				if i < len(setupActions[f]) {
					row = append(row, setupActions[f][i])
				} else {
					row = append(row, "")
				}
			}
			result = append(result, formatTableRow(row))
		}
	}

	// Rounds
	for _, round := range rounds {
		result = append(result, "")
		result = append(result, fmt.Sprintf("Round %d", round.Number))

		if len(round.TurnOrder) > 0 {
			// Keep round turn-order identifiers in the same canonical player-ID casing
			// used by setup/header rows so replay action player IDs and turn order match.
			result = append(result, fmt.Sprintf("TurnOrder: %s", formatFactionList(round.TurnOrder)))
		}

		columns := factions
		if len(round.TurnOrder) > 0 {
			columns = round.TurnOrder
		}
		if err := enforceLeechSourceOrder(round, columns, roundLeechAnchors[round]); err != nil {
			return "", err
		}
		if err := stabilizeLeechBindings(round, columns, roundLeechAnchors[round]); err != nil {
			return "", err
		}
		resolveDuplicateLeechBindings(round, columns, roundLeechAnchors[round])
		resolveAnchoredDuplicateSources(round, columns, roundLeechAnchors[round])
		forceLeechesOffNonTriggers(round, columns, roundLeechAnchors[round])
		breakRemainingDuplicateLeeches(round, columns, roundLeechAnchors[round])
		compactCrossRowBlankRuns(round, columns)

		result = append(result, strings.Repeat("-", 60))
		result = append(result, formatFactionHeader(columns))
		result = append(result, strings.Repeat("-", 60))

		// Output chronological rows.
		for i := 0; i < len(round.Rows); i++ {
			empty := true
			for _, f := range columns {
				if strings.TrimSpace(round.Rows[i][f]) != "" {
					empty = false
					break
				}
			}
			if empty {
				continue
			}
			var row []string
			for _, f := range columns {
				row = append(row, round.Rows[i][f])
			}
			result = append(result, formatTableRow(row))
		}
	}

	return strings.Join(result, "\n"), nil
}

func placeRoundAction(round *snellmanRoundData, faction, action string, startRow int) int {
	if startRow < 0 {
		startRow = 0
	}
	for len(round.Rows) <= startRow {
		round.Rows = append(round.Rows, make(map[string]string))
	}
	for row := startRow; ; row++ {
		if row >= len(round.Rows) {
			round.Rows = append(round.Rows, make(map[string]string))
		}
		if round.Rows[row][faction] == "" {
			round.Rows[row][faction] = action
			return row
		}
	}
}

func compactCrossRowBlankRuns(round *snellmanRoundData, columns []string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return
	}
	round.Rows = stripEmptyRows(round.Rows, columns)
	if len(round.Rows) == 0 {
		return
	}
	width := len(columns)
	flat := make([]string, 0, len(round.Rows)*width)
	for _, row := range round.Rows {
		for _, c := range columns {
			flat = append(flat, strings.TrimSpace(row[c]))
		}
	}
	compacted := make([]string, 0, len(flat))
	for i := 0; i < len(flat); {
		if flat[i] != "" {
			compacted = append(compacted, flat[i])
			i++
			continue
		}
		j := i
		for j < len(flat) && flat[j] == "" {
			j++
		}
		keep := (j - i) % width
		for k := 0; k < keep; k++ {
			compacted = append(compacted, "")
		}
		i = j
	}
	newRows := make([]map[string]string, 0, (len(compacted)+width-1)/width)
	for idx, tok := range compacted {
		if idx%width == 0 {
			newRows = append(newRows, make(map[string]string))
		}
		if tok != "" {
			row := idx / width
			col := idx % width
			newRows[row][columns[col]] = tok
		}
	}
	round.Rows = stripEmptyRows(newRows, columns)
}

func stripEmptyRows(rows []map[string]string, columns []string) []map[string]string {
	if len(rows) == 0 {
		return rows
	}
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		empty := true
		for _, c := range columns {
			if strings.TrimSpace(row[c]) != "" {
				empty = false
				break
			}
		}
		if !empty {
			out = append(out, row)
		}
	}
	return out
}

type anchoredEvent struct {
	col           int
	faction       string
	token         string
	sourceFaction string
}

func enforceLeechSourceOrder(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) error {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return nil
	}
	events := flattenAnchoredEvents(round, columns, anchors)
	if len(events) == 0 {
		return nil
	}
	lastReqByReactor := make(map[string]int)
	for i := 0; i < len(events); i++ {
		ev := events[i]
		if ev.token != "L" && ev.token != "DL" {
			continue
		}
		if ev.sourceFaction == "" {
			continue
		}
		req := -1
		for j := i - 1; j >= 0; j-- {
			if events[j].token == "L" || events[j].token == "DL" {
				continue
			}
			if normalizeFactionNameForMatching(events[j].faction) == normalizeFactionNameForMatching(ev.sourceFaction) && actionMayTriggerLeech(events[j].token) {
				req = j
				break
			}
		}
		// Some Snellman logs can be out-of-order; if no prior eligible source is found,
		// allow binding to the next eligible source action by the same faction.
		if req < 0 {
			for j := i + 1; j < len(events); j++ {
				if events[j].token == "L" || events[j].token == "DL" {
					continue
				}
				if normalizeFactionNameForMatching(events[j].faction) == normalizeFactionNameForMatching(ev.sourceFaction) && actionMayTriggerLeech(events[j].token) {
					req = j
					break
				}
			}
		}
		if req < 0 {
			return fmt.Errorf("unable to resolve leech source ordering for source faction %q token %q", ev.sourceFaction, ev.token)
		}
		// If this reactor already leeched from the same resolved source action,
		// force binding to the next eligible action by the same source faction.
		if prevReq, ok := lastReqByReactor[ev.faction]; ok && prevReq == req {
			for j := req + 1; j < len(events); j++ {
				if events[j].token == "L" || events[j].token == "DL" {
					continue
				}
				if normalizeFactionNameForMatching(events[j].faction) == normalizeFactionNameForMatching(ev.sourceFaction) && actionMayTriggerLeech(events[j].token) {
					req = j
					break
				}
			}
		}
		prev := -1
		for j := i - 1; j >= 0; j-- {
			if events[j].token != "L" && events[j].token != "DL" {
				prev = j
				break
			}
		}
		if prev == req {
			continue
		}
		moved := ev
		events = append(events[:i], events[i+1:]...)
		insertAt := req + 1
		if i < insertAt {
			insertAt--
		}
		events = append(events, anchoredEvent{})
		copy(events[insertAt+1:], events[insertAt:])
		events[insertAt] = moved
		lastReqByReactor[ev.faction] = req
	}
	rebuildFromAnchoredEvents(round, columns, events, anchors)
	return nil
}

func flattenAnchoredEvents(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) []anchoredEvent {
	out := make([]anchoredEvent, 0)
	for r := 0; r < len(round.Rows); r++ {
		for c, faction := range columns {
			tok := strings.TrimSpace(round.Rows[r][faction])
			if tok == "" {
				continue
			}
			source := ""
			if tok == "L" || tok == "DL" {
				if m, ok := anchors[r]; ok {
					source = m[faction]
				}
			}
			out = append(out, anchoredEvent{
				col:           c,
				faction:       faction,
				token:         tok,
				sourceFaction: source,
			})
		}
	}
	return out
}

func rebuildFromAnchoredEvents(round *snellmanRoundData, columns []string, events []anchoredEvent, anchors map[int]map[string]string) {
	for k := range anchors {
		delete(anchors, k)
	}
	if len(events) == 0 {
		round.Rows = nil
		return
	}
	width := len(columns)
	pos := -1
	type placed struct {
		row int
		ev  anchoredEvent
	}
	placedEvents := make([]placed, 0, len(events))
	for _, ev := range events {
		for {
			pos++
			if pos%width == ev.col {
				break
			}
		}
		placedEvents = append(placedEvents, placed{
			row: pos / width,
			ev:  ev,
		})
	}
	rows := make([]map[string]string, 0, (pos/width)+1)
	for i := 0; i <= pos/width; i++ {
		rows = append(rows, make(map[string]string))
	}
	for _, p := range placedEvents {
		rows[p.row][p.ev.faction] = p.ev.token
		if (p.ev.token == "L" || p.ev.token == "DL") && p.ev.sourceFaction != "" {
			if _, ok := anchors[p.row]; !ok {
				anchors[p.row] = make(map[string]string)
			}
			anchors[p.row][p.ev.faction] = p.ev.sourceFaction
		}
	}
	round.Rows = stripEmptyRows(rows, columns)
}

func stabilizeLeechBindings(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) error {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return nil
	}
	type prevPos struct {
		row int
		col int
	}
	maxIters := len(round.Rows)*len(columns)*8 + 64
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		lastPrevByReactor := make(map[string]prevPos)
		for r := 0; r < len(round.Rows); r++ {
			for c, reactorFaction := range columns {
				tok := strings.TrimSpace(round.Rows[r][reactorFaction])
				if tok != "L" && tok != "DL" {
					continue
				}
				pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, c)
				if !ok {
					continue
				}
				sourceFaction := ""
				if m, ok := anchors[r]; ok {
					sourceFaction = normalizeFactionNameForMatching(m[reactorFaction])
				}
				needsMove := false
				duplicatePrev := false
				if !actionMayTriggerLeech(pTok) {
					needsMove = true
				}
				if sourceFaction != "" && normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
					needsMove = true
				}
				if prev, ok := lastPrevByReactor[reactorFaction]; ok && prev.row == pRow && prev.col == pCol {
					needsMove = true
					duplicatePrev = true
				}
				if !needsMove {
					lastPrevByReactor[reactorFaction] = prevPos{row: pRow, col: pCol}
					continue
				}
				lastPrevRow, lastPrevCol := -1, -1
				if prev, ok := lastPrevByReactor[reactorFaction]; ok {
					lastPrevRow, lastPrevCol = prev.row, prev.col
				}
				sourceForSearch := sourceFaction
				if duplicatePrev {
					sourceForSearch = ""
				}
				if !actionMayTriggerLeech(pTok) {
					// If currently bound to a definitely invalid trigger, prefer rebinding to
					// the nearest valid trigger even if source-faction anchoring is noisy.
					sourceForSearch = ""
				}
				if hasFutureValidLeechBinding(round, columns, r, c, sourceForSearch, lastPrevRow, lastPrevCol) {
					moveLeechTokenDown(round, anchors, reactorFaction, r, r+1)
					changed = true
					break
				}
				if hasPastValidLeechBinding(round, columns, r, c, sourceForSearch, lastPrevRow, lastPrevCol) {
					if moveLeechTokenUp(round, anchors, reactorFaction, r, r-1) {
						changed = true
						break
					}
				}
				lastPrevByReactor[reactorFaction] = prevPos{row: pRow, col: pCol}
				continue
			}
			if changed {
				break
			}
		}
		if !changed {
			return nil
		}
	}
	return nil
}

func moveLeechTokenDown(
	round *snellmanRoundData,
	anchors map[int]map[string]string,
	reactorFaction string,
	fromRow int,
	targetRow int,
) {
	_ = moveLeechTokenToRow(round, anchors, reactorFaction, fromRow, targetRow, true)
}

func moveLeechTokenUp(
	round *snellmanRoundData,
	anchors map[int]map[string]string,
	reactorFaction string,
	fromRow int,
	targetRow int,
) bool {
	return moveLeechTokenToRow(round, anchors, reactorFaction, fromRow, targetRow, false)
}

func moveLeechTokenToRow(
	round *snellmanRoundData,
	anchors map[int]map[string]string,
	reactorFaction string,
	fromRow int,
	targetRow int,
	allowGrowDown bool,
) bool {
	if round == nil {
		return false
	}
	if fromRow < 0 || fromRow >= len(round.Rows) {
		return false
	}
	if targetRow == fromRow {
		return false
	}
	if targetRow > fromRow {
		for len(round.Rows) <= targetRow {
			round.Rows = append(round.Rows, make(map[string]string))
		}
		for strings.TrimSpace(round.Rows[targetRow][reactorFaction]) != "" {
			targetRow++
			if !allowGrowDown {
				return false
			}
			for len(round.Rows) <= targetRow {
				round.Rows = append(round.Rows, make(map[string]string))
			}
		}
	} else {
		for targetRow >= 0 && strings.TrimSpace(round.Rows[targetRow][reactorFaction]) != "" {
			targetRow--
		}
		if targetRow < 0 {
			return false
		}
	}
	tok := strings.TrimSpace(round.Rows[fromRow][reactorFaction])
	if tok == "" {
		return false
	}
	round.Rows[targetRow][reactorFaction] = tok
	delete(round.Rows[fromRow], reactorFaction)
	if m, ok := anchors[fromRow]; ok {
		if src, ok := m[reactorFaction]; ok {
			if _, ok := anchors[targetRow]; !ok {
				anchors[targetRow] = make(map[string]string)
			}
			anchors[targetRow][reactorFaction] = src
			delete(m, reactorFaction)
			if len(m) == 0 {
				delete(anchors, fromRow)
			}
		}
	}
	return true
}

func insertRoundRowAtSimple(round *snellmanRoundData, anchors map[int]map[string]string, at int) {
	if round == nil {
		return
	}
	if at < 0 {
		at = 0
	}
	if at > len(round.Rows) {
		at = len(round.Rows)
	}
	round.Rows = append(round.Rows, nil)
	copy(round.Rows[at+1:], round.Rows[at:])
	round.Rows[at] = make(map[string]string)
	if len(anchors) == 0 {
		return
	}
	shifted := make(map[int]map[string]string, len(anchors))
	for row, m := range anchors {
		if row >= at {
			shifted[row+1] = m
		} else {
			shifted[row] = m
		}
	}
	for k := range anchors {
		delete(anchors, k)
	}
	for k, v := range shifted {
		anchors[k] = v
	}
}

func hasPastValidLeechBinding(
	round *snellmanRoundData,
	columns []string,
	row, col int,
	sourceFaction string,
	lastPrevRow, lastPrevCol int,
) bool {
	if round == nil || len(columns) == 0 || row <= 0 || col < 0 || col >= len(columns) {
		return false
	}
	for r := row - 1; r >= 0; r-- {
		if strings.TrimSpace(round.Rows[r][columns[col]]) != "" {
			continue
		}
		pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, col)
		if !ok {
			continue
		}
		if !actionMayTriggerLeech(pTok) {
			continue
		}
		if sourceFaction != "" && normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
			continue
		}
		if lastPrevRow >= 0 && lastPrevCol >= 0 && lastPrevRow == pRow && lastPrevCol == pCol {
			continue
		}
		return true
	}
	return false
}

func findPreviousNonLeech(round *snellmanRoundData, columns []string, row, col int) (int, int, string, bool) {
	if round == nil || row < 0 || row >= len(round.Rows) || col < 0 || col >= len(columns) {
		return 0, 0, "", false
	}
	for r := row; r >= 0; r-- {
		startCol := len(columns) - 1
		if r == row {
			startCol = col - 1
		}
		for c := startCol; c >= 0; c-- {
			tok := strings.TrimSpace(round.Rows[r][columns[c]])
			if tok == "" || tok == "L" || tok == "DL" {
				continue
			}
			return r, c, tok, true
		}
	}
	return 0, 0, "", false
}

func findPreviousNonLeechAtPosition(round *snellmanRoundData, columns []string, row, col int) (int, int, string, bool) {
	if round == nil || len(columns) == 0 || col < 0 || col >= len(columns) {
		return 0, 0, "", false
	}
	if row < 0 {
		return 0, 0, "", false
	}
	if row >= len(round.Rows) {
		row = len(round.Rows)
	}
	if row == 0 {
		return 0, 0, "", false
	}
	for r := row; r >= 0; r-- {
		startCol := len(columns) - 1
		if r == row {
			if r >= len(round.Rows) {
				startCol = len(columns) - 1
			} else {
				startCol = col - 1
			}
		}
		if r >= len(round.Rows) {
			continue
		}
		for c := startCol; c >= 0; c-- {
			tok := strings.TrimSpace(round.Rows[r][columns[c]])
			if tok == "" || tok == "L" || tok == "DL" {
				continue
			}
			return r, c, tok, true
		}
	}
	return 0, 0, "", false
}

func hasFutureValidLeechBinding(
	round *snellmanRoundData,
	columns []string,
	row, col int,
	sourceFaction string,
	lastPrevRow, lastPrevCol int,
) bool {
	if round == nil || len(columns) == 0 || row < 0 || col < 0 || col >= len(columns) {
		return false
	}
	for r := row + 1; r <= len(round.Rows); r++ {
		pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, col)
		if !ok {
			continue
		}
		if !actionMayTriggerLeech(pTok) {
			continue
		}
		if sourceFaction != "" && normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
			continue
		}
		if lastPrevRow >= 0 && lastPrevCol >= 0 && lastPrevRow == pRow && lastPrevCol == pCol {
			continue
		}
		return true
	}
	return false
}

func findFutureLeechPlacementRow(
	round *snellmanRoundData,
	columns []string,
	reactorFaction string,
	row, col int,
	sourceFaction string,
	lastPrevRow, lastPrevCol int,
) int {
	if round == nil || len(columns) == 0 || row < 0 || col < 0 || col >= len(columns) {
		return -1
	}
	for r := row + 1; r <= len(round.Rows); r++ {
		if r < len(round.Rows) && strings.TrimSpace(round.Rows[r][reactorFaction]) != "" {
			continue
		}
		pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, col)
		if !ok {
			continue
		}
		if !actionMayTriggerLeech(pTok) {
			continue
		}
		if sourceFaction != "" && normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
			continue
		}
		if lastPrevRow >= 0 && lastPrevCol >= 0 && lastPrevRow == pRow && lastPrevCol == pCol {
			continue
		}
		return r
	}
	return -1
}

func findPastLeechPlacementRow(
	round *snellmanRoundData,
	columns []string,
	reactorFaction string,
	row, col int,
	sourceFaction string,
	lastPrevRow, lastPrevCol int,
) int {
	if round == nil || len(columns) == 0 || row < 0 || col < 0 || col >= len(columns) {
		return -1
	}
	for r := row - 1; r >= 0; r-- {
		if strings.TrimSpace(round.Rows[r][reactorFaction]) != "" {
			continue
		}
		pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, col)
		if !ok {
			continue
		}
		if !actionMayTriggerLeech(pTok) {
			continue
		}
		if sourceFaction != "" && normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
			continue
		}
		if lastPrevRow >= 0 && lastPrevCol >= 0 && lastPrevRow == pRow && lastPrevCol == pCol {
			continue
		}
		return r
	}
	return -1
}

func forceLeechesOffNonTriggers(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return
	}
	maxIters := len(round.Rows)*len(columns)*6 + 32
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		for r := 0; r < len(round.Rows); r++ {
			for c, faction := range columns {
				tok := strings.TrimSpace(round.Rows[r][faction])
				if tok != "L" && tok != "DL" {
					continue
				}
				_, _, pTok, ok := findPreviousNonLeech(round, columns, r, c)
				if !ok || actionMayTriggerLeech(pTok) {
					continue
				}
				lastLeechPrevRow, lastLeechPrevCol := -1, -1
				for pr := r - 1; pr >= 0; pr-- {
					pt := strings.TrimSpace(round.Rows[pr][faction])
					if pt != "L" && pt != "DL" {
						continue
					}
					lr, lc, _, lok := findPreviousNonLeech(round, columns, pr, c)
					if lok {
						lastLeechPrevRow, lastLeechPrevCol = lr, lc
					}
					break
				}
				for u := r - 1; u >= 0; u-- {
					if strings.TrimSpace(round.Rows[u][faction]) != "" {
						continue
					}
					p2r, p2c, p2Tok, ok2 := findPreviousNonLeech(round, columns, u, c)
					if !ok2 || !actionMayTriggerLeech(p2Tok) {
						continue
					}
					if lastLeechPrevRow >= 0 && lastLeechPrevCol >= 0 && p2r == lastLeechPrevRow && p2c == lastLeechPrevCol {
						continue
					}
					if moveLeechTokenUp(round, anchors, faction, r, u) {
						changed = true
					}
					break
				}
				if changed {
					break
				}
			}
			if changed {
				break
			}
		}
		if !changed {
			return
		}
	}
}

func alignLeechesToAnchoredSource(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 || len(anchors) == 0 {
		return
	}
	maxIters := len(round.Rows)*len(columns)*8 + 64
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		for r := 0; r < len(round.Rows); r++ {
			for c, reactorFaction := range columns {
				tok := strings.TrimSpace(round.Rows[r][reactorFaction])
				if tok != "L" && tok != "DL" {
					continue
				}
				sourceFaction := ""
				if m, ok := anchors[r]; ok {
					sourceFaction = normalizeFactionNameForMatching(m[reactorFaction])
				}
				if sourceFaction == "" {
					continue
				}
				_, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, c)
				if ok && actionMayTriggerLeech(pTok) &&
					normalizeFactionNameForMatching(columns[pCol]) == sourceFaction {
					continue
				}
				target := findSourceBindingTargetRow(round, columns, reactorFaction, sourceFaction, r, c)
				if target < 0 {
					insertRoundRowAtSimple(round, anchors, r+1)
					target = r + 1
				}
				if target == r {
					continue
				}
				if target < r {
					if !moveLeechTokenUp(round, anchors, reactorFaction, r, target) {
						continue
					}
				} else {
					moveLeechTokenDown(round, anchors, reactorFaction, r, target)
				}
				changed = true
				break
			}
			if changed {
				break
			}
		}
		if !changed {
			return
		}
	}
}

func findSourceBindingTargetRow(
	round *snellmanRoundData,
	columns []string,
	reactorFaction string,
	sourceFaction string,
	currentRow int,
	col int,
) int {
	if round == nil || len(columns) == 0 || col < 0 || col >= len(columns) {
		return -1
	}
	best := -1
	bestDist := 1 << 30
	for cand := 0; cand <= len(round.Rows); cand++ {
		if cand != len(round.Rows) && strings.TrimSpace(round.Rows[cand][reactorFaction]) != "" {
			continue
		}
		_, pCol, pTok, ok := findPreviousNonLeechAtPosition(round, columns, cand, col)
		if !ok || !actionMayTriggerLeech(pTok) {
			continue
		}
		if normalizeFactionNameForMatching(columns[pCol]) != sourceFaction {
			continue
		}
		dist := cand - currentRow
		if dist < 0 {
			dist = -dist
		}
		if dist < bestDist {
			bestDist = dist
			best = cand
		}
	}
	return best
}

func resolveDuplicateLeechBindings(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return
	}
	maxIters := len(round.Rows)*len(columns)*8 + 64
	for iter := 0; iter < maxIters; iter++ {
		moved := false
		for c, reactorFaction := range columns {
			lastPrevRow, lastPrevCol := -1, -1
			lastLeechRow := -1
			lastLeechSource := ""
			for r := 0; r < len(round.Rows); r++ {
				tok := strings.TrimSpace(round.Rows[r][reactorFaction])
				if tok != "L" && tok != "DL" {
					continue
				}
				source := ""
				if m, ok := anchors[r]; ok {
					source = normalizeFactionNameForMatching(m[reactorFaction])
				}
				pRow, pCol, _, ok := findPreviousNonLeech(round, columns, r, c)
				if !ok {
					continue
				}
				if lastPrevRow == pRow && lastPrevCol == pCol {
					// Generic duplicate breaker: move the previous leech upward to the nearest
					// row that yields a different valid trigger binding.
					if lastLeechRow >= 0 {
						prevTarget := findPastLeechPlacementRow(round, columns, reactorFaction, lastLeechRow, c, lastLeechSource, pRow, pCol)
						if prevTarget >= 0 {
							if moveLeechTokenUp(round, anchors, reactorFaction, lastLeechRow, prevTarget) {
								moved = true
								break
							}
						}
					}
					// If two consecutive leeches currently bind to the same prior action but
					// come from different source factions, place the current one on a row that
					// binds to its own source faction action.
					if lastLeechRow >= 0 && source != "" && lastLeechSource != "" && source != lastLeechSource {
						targetRow := findFutureLeechPlacementRow(round, columns, reactorFaction, r, c, source, pRow, pCol)
						if targetRow < 0 {
							targetRow = findPastLeechPlacementRow(round, columns, reactorFaction, r, c, source, pRow, pCol)
						}
						if targetRow >= 0 {
							if targetRow < r {
								moveLeechTokenUp(round, anchors, reactorFaction, r, targetRow)
							} else {
								moveLeechTokenDown(round, anchors, reactorFaction, r, targetRow)
							}
							moved = true
							break
						}
						// Fall back to inserting one row and moving down; this often makes the
						// previous non-leech in row-major order become the intended source.
						insertRoundRowAtSimple(round, anchors, r+1)
						moveLeechTokenDown(round, anchors, reactorFaction, r, r+1)
						moved = true
						break
					}
					// Contradictory duplicate decisions for the same source action in the same
					// reactor column can appear in noisy logs (e.g. DL followed by L). Keep the
					// latest decision and drop the earlier one.
					if lastLeechRow >= 0 && source != "" && source == lastLeechSource {
						delete(round.Rows[lastLeechRow], reactorFaction)
						if m, ok := anchors[lastLeechRow]; ok {
							delete(m, reactorFaction)
							if len(m) == 0 {
								delete(anchors, lastLeechRow)
							}
						}
						moved = true
						break
					}
					targetRow := -1
					// Prefer preserving explicit source-faction anchor if available.
					if source != "" && hasFutureValidLeechBinding(round, columns, r, c, source, pRow, pCol) {
						targetRow = findFutureLeechPlacementRow(round, columns, reactorFaction, r, c, source, pRow, pCol)
					}
					// Fallback: any future valid trigger different from the duplicated one.
					if targetRow < 0 && hasFutureValidLeechBinding(round, columns, r, c, "", pRow, pCol) {
						targetRow = findFutureLeechPlacementRow(round, columns, reactorFaction, r, c, "", pRow, pCol)
					}
					if targetRow < 0 {
						targetRow = findPastLeechPlacementRow(round, columns, reactorFaction, r, c, source, pRow, pCol)
					}
					if targetRow < 0 {
						insertRoundRowAtSimple(round, anchors, r+1)
						moveLeechTokenDown(round, anchors, reactorFaction, r, r+1)
						moved = true
						break
						lastLeechRow = r
						lastLeechSource = source
						continue
					}
					if targetRow < r {
						moveLeechTokenUp(round, anchors, reactorFaction, r, targetRow)
					} else {
						moveLeechTokenDown(round, anchors, reactorFaction, r, targetRow)
					}
					moved = true
					break
				}
				lastPrevRow, lastPrevCol = pRow, pCol
				lastLeechRow = r
				lastLeechSource = source
			}
			if moved {
				break
			}
		}
		if !moved {
			return
		}
	}
}

func resolveAnchoredDuplicateSources(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 || len(anchors) == 0 {
		return
	}
	maxIters := len(round.Rows)*len(columns)*6 + 32
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		for c, reactorFaction := range columns {
			lastLeechRow := -1
			lastPrevRow, lastPrevCol := -1, -1
			lastSource := ""
			for r := 0; r < len(round.Rows); r++ {
				tok := strings.TrimSpace(round.Rows[r][reactorFaction])
				if tok != "L" && tok != "DL" {
					continue
				}
				source := ""
				if m, ok := anchors[r]; ok {
					source = normalizeFactionNameForMatching(m[reactorFaction])
				}
				pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, c)
				if !ok || !actionMayTriggerLeech(pTok) {
					lastLeechRow = r
					lastPrevRow, lastPrevCol = pRow, pCol
					lastSource = source
					continue
				}
				if lastLeechRow >= 0 && pRow == lastPrevRow && pCol == lastPrevCol &&
					source != "" && lastSource != "" && source != lastSource {
					target := findSourceBindingTargetRow(round, columns, reactorFaction, source, r, c)
					if target < 0 || target == r {
						insertRoundRowAtSimple(round, anchors, r+1)
						target = r + 1
					}
					if target < r {
						moveLeechTokenUp(round, anchors, reactorFaction, r, target)
					} else {
						moveLeechTokenDown(round, anchors, reactorFaction, r, target)
					}
					changed = true
					break
				}
				lastLeechRow = r
				lastPrevRow, lastPrevCol = pRow, pCol
				lastSource = source
			}
			if changed {
				break
			}
		}
		if !changed {
			return
		}
	}
}

func breakRemainingDuplicateLeeches(round *snellmanRoundData, columns []string, anchors map[int]map[string]string) {
	if round == nil || len(columns) == 0 || len(round.Rows) == 0 {
		return
	}
	maxIters := len(round.Rows)*len(columns)*6 + 32
	for iter := 0; iter < maxIters; iter++ {
		changed := false
		for c, reactorFaction := range columns {
			lastPrevRow, lastPrevCol := -1, -1
			for r := 0; r < len(round.Rows); r++ {
				tok := strings.TrimSpace(round.Rows[r][reactorFaction])
				if tok != "L" && tok != "DL" {
					continue
				}
				pRow, pCol, pTok, ok := findPreviousNonLeech(round, columns, r, c)
				if !ok || !actionMayTriggerLeech(pTok) {
					lastPrevRow, lastPrevCol = pRow, pCol
					continue
				}
				if lastPrevRow == pRow && lastPrevCol == pCol {
					target := findPastLeechPlacementRow(round, columns, reactorFaction, r, c, "", pRow, pCol)
					if target < 0 {
						target = findFutureLeechPlacementRow(round, columns, reactorFaction, r, c, "", pRow, pCol)
					}
					if target < 0 || target == r {
						insertRoundRowAtSimple(round, anchors, r+1)
						target = r + 1
					}
					if target < r {
						moveLeechTokenUp(round, anchors, reactorFaction, r, target)
					} else {
						moveLeechTokenDown(round, anchors, reactorFaction, r, target)
					}
					changed = true
					break
				}
				lastPrevRow, lastPrevCol = pRow, pCol
			}
			if changed {
				break
			}
		}
		if !changed {
			return
		}
	}
}

func previousNonLeechIsSource(round *snellmanRoundData, columns []string, row, col, sourceRow, sourceCol int) bool {
	if round == nil || row < 0 || row >= len(round.Rows) || col < 0 || col >= len(columns) || sourceRow < 0 || sourceCol < 0 {
		return false
	}
	for r := row; r >= 0; r-- {
		cStart := len(columns) - 1
		if r == row {
			cStart = col - 1
		}
		for c := cStart; c >= 0; c-- {
			tok := strings.TrimSpace(round.Rows[r][columns[c]])
			if tok == "" || tok == "L" || tok == "DL" {
				continue
			}
			return r == sourceRow && c == sourceCol
		}
	}
	return false
}

func insertRoundRowAt(
	round *snellmanRoundData,
	insertAt int,
	lastMainActionRow map[string]int,
	lastLeechSourceRow map[string]int,
	lastEventRow map[string]int,
	rowLockedForMain map[int]bool,
	leechAnchorSource map[int]map[string]string,
) {
	if round == nil {
		return
	}
	if insertAt < 0 {
		insertAt = 0
	}
	if insertAt > len(round.Rows) {
		insertAt = len(round.Rows)
	}
	round.Rows = append(round.Rows, nil)
	copy(round.Rows[insertAt+1:], round.Rows[insertAt:])
	round.Rows[insertAt] = make(map[string]string)

	for f, r := range lastMainActionRow {
		if r >= insertAt {
			lastMainActionRow[f] = r + 1
		}
	}
	for f, r := range lastLeechSourceRow {
		if r >= insertAt {
			lastLeechSourceRow[f] = r + 1
		}
	}
	for f, r := range lastEventRow {
		if r >= insertAt {
			lastEventRow[f] = r + 1
		}
	}
	if len(rowLockedForMain) > 0 {
		shifted := make(map[int]bool, len(rowLockedForMain))
		for r, v := range rowLockedForMain {
			if r >= insertAt {
				shifted[r+1] = v
			} else {
				shifted[r] = v
			}
		}
		for k := range rowLockedForMain {
			delete(rowLockedForMain, k)
		}
		for k, v := range shifted {
			rowLockedForMain[k] = v
		}
	}
	if len(leechAnchorSource) > 0 {
		shifted := make(map[int]map[string]string, len(leechAnchorSource))
		for r, v := range leechAnchorSource {
			if r >= insertAt {
				shifted[r+1] = v
			} else {
				shifted[r] = v
			}
		}
		for k := range leechAnchorSource {
			delete(leechAnchorSource, k)
		}
		for k, v := range shifted {
			leechAnchorSource[k] = v
		}
	}
}

func trailingBlankCount(row map[string]string, columns []string) int {
	count := 0
	for i := len(columns) - 1; i >= 0; i-- {
		if strings.TrimSpace(row[columns[i]]) == "" {
			count++
		} else {
			break
		}
	}
	return count
}

func leadingBlankCount(row map[string]string, columns []string) int {
	count := 0
	for i := 0; i < len(columns); i++ {
		if strings.TrimSpace(row[columns[i]]) == "" {
			count++
		} else {
			break
		}
	}
	return count
}

func firstNonEmptyCell(row map[string]string, columns []string) (string, string, int, bool) {
	for i := 0; i < len(columns); i++ {
		tok := strings.TrimSpace(row[columns[i]])
		if tok != "" {
			return columns[i], tok, i, true
		}
	}
	return "", "", -1, false
}

func hasOccupiedColumnGreaterThan(row map[string]string, columns []string, col int) bool {
	if row == nil {
		return false
	}
	for f, tok := range row {
		if strings.TrimSpace(tok) == "" {
			continue
		}
		c := factionColumnIndex(columns, f)
		if c > col {
			return true
		}
	}
	return false
}

func placeMainAction(round *snellmanRoundData, faction, action string, startRow int, blockedRows map[int]bool, columns []string) int {
	if startRow < 0 {
		startRow = 0
	}
	for row := startRow; ; row++ {
		if row >= len(round.Rows) {
			round.Rows = append(round.Rows, make(map[string]string))
		}
		if blockedRows[row] {
			continue
		}
		if !rowCanAcceptMainAction(round.Rows[row], faction, columns) {
			continue
		}
		if round.Rows[row][faction] == "" {
			round.Rows[row][faction] = action
			return row
		}
	}
}

func rowCanAcceptMainAction(row map[string]string, faction string, columns []string) bool {
	if row == nil {
		return true
	}
	newCol := factionColumnIndex(columns, faction)
	if newCol < 0 {
		return true
	}
	for existingFaction, tok := range row {
		if tok != "L" && tok != "DL" {
			continue
		}
		existingCol := factionColumnIndex(columns, existingFaction)
		if existingCol > newCol {
			// Do not place a main action before an existing leech marker in the same row.
			// This can make the leech appear to correspond to the wrong action.
			return false
		}
	}

	return true
}

func moveSourceActionAndAnchoredLeeches(
	round *snellmanRoundData,
	sourceFaction string,
	fromRow int,
	startRow int,
	blockedRows map[int]bool,
	lastMainActionRow map[string]int,
	lastLeechSourceRow map[string]int,
	leechAnchorSource map[int]map[string]string,
) int {
	if round == nil || fromRow < 0 || fromRow >= len(round.Rows) {
		return fromRow
	}
	sourceAction := round.Rows[fromRow][sourceFaction]
	if sourceAction == "" {
		return fromRow
	}

	if startRow < fromRow+1 {
		startRow = fromRow + 1
	}
	toRow := placeMainAction(round, sourceFaction, sourceAction, startRow, blockedRows, roundColumns(round, nil))
	round.Rows[fromRow][sourceFaction] = ""

	if lastMainActionRow[sourceFaction] == fromRow {
		lastMainActionRow[sourceFaction] = toRow
	}
	if lastLeechSourceRow[sourceFaction] == fromRow {
		lastLeechSourceRow[sourceFaction] = toRow
	}

	anchors := leechAnchorSource[fromRow]
	for reactor, src := range anchors {
		if src != sourceFaction {
			continue
		}
		leechAction := round.Rows[fromRow][reactor]
		if leechAction == "" {
			delete(anchors, reactor)
			continue
		}
		newRow := placeRoundAction(round, reactor, leechAction, toRow)
		round.Rows[fromRow][reactor] = ""
		delete(anchors, reactor)
		if _, ok := leechAnchorSource[newRow]; !ok {
			leechAnchorSource[newRow] = make(map[string]string)
		}
		leechAnchorSource[newRow][reactor] = sourceFaction
	}
	if len(anchors) == 0 {
		delete(leechAnchorSource, fromRow)
	} else {
		leechAnchorSource[fromRow] = anchors
	}

	return toRow
}

func relocateInterferingActionsForLeech(
	round *snellmanRoundData,
	sourceRow int,
	sourceFaction string,
	reactorFaction string,
	blockedRows map[int]bool,
	lastMainActionRow map[string]int,
	lastLeechSourceRow map[string]int,
	leechAnchorSource map[int]map[string]string,
	fallbackCols []string,
) {
	if round == nil || sourceRow < 0 || sourceRow >= len(round.Rows) {
		return
	}
	cols := roundColumns(round, fallbackCols)
	sourceCol := factionColumnIndex(cols, sourceFaction)
	reactorCol := factionColumnIndex(cols, reactorFaction)
	if sourceCol < 0 || reactorCol < 0 {
		return
	}

	var moveCols []int
	if reactorCol > sourceCol {
		// Inline leech on the right: nothing should sit between source and reactor columns.
		for c := sourceCol + 1; c < reactorCol; c++ {
			moveCols = append(moveCols, c)
		}
	} else if reactorCol < sourceCol {
		// Leech moved to next row for left-side reactor: no trailing actions to the right of source.
		for c := sourceCol + 1; c < len(cols); c++ {
			moveCols = append(moveCols, c)
		}
	} else {
		return
	}

	for _, col := range moveCols {
		f := cols[col]
		if f == "" || f == sourceFaction || f == reactorFaction {
			continue
		}
		a := strings.TrimSpace(round.Rows[sourceRow][f])
		if a == "" || a == "L" || a == "DL" {
			continue
		}
		moveSourceActionAndAnchoredLeeches(
			round,
			f,
			sourceRow,
			sourceRow+1,
			blockedRows,
			lastMainActionRow,
			lastLeechSourceRow,
			leechAnchorSource,
		)
	}
}

func extractTrailingLeechAction(action string) string {
	re := regexp.MustCompile(`(?i)(Leech\s+\d+\s+from\s+.+|Decline\s+\d+\s+from\s+.+)$`)
	if m := re.FindStringSubmatch(strings.TrimSpace(action)); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return action
}

func extractLeechSourceFaction(action string) string {
	re := regexp.MustCompile(`(?i)(?:Leech|Decline)\s+\d+\s+from\s+([a-z ]+)$`)
	m := re.FindStringSubmatch(strings.TrimSpace(action))
	if len(m) < 2 {
		return ""
	}
	source := normalizeFactionNameForMatching(m[1])
	if isKnownFaction(source) {
		return source
	}
	return ""
}

func factionColumnIndex(factions []string, faction string) int {
	for i, f := range factions {
		if f == faction {
			return i
		}
	}
	return -1
}

func roundColumns(round *snellmanRoundData, fallback []string) []string {
	if round != nil && len(round.TurnOrder) > 0 {
		return round.TurnOrder
	}
	return fallback
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func computeRoundTurnOrder(rounds []*snellmanRoundData, factions []string, _ bool, variableTurnOrder bool) []string {
	if len(rounds) == 0 {
		return append([]string(nil), factions...)
	}
	prevRound := rounds[len(rounds)-1]
	if len(prevRound.PassOrder) != len(factions) {
		return append([]string(nil), factions...)
	}
	// Policy: if variable-turn-order is present, use full pass order.
	// Otherwise always use maintain/cyclic order (ignore maintain-player-order option).
	if variableTurnOrder {
		return append([]string(nil), prevRound.PassOrder...)
	}
	if len(prevRound.PassOrder) > 0 {
		return rotateToFirst(factions, prevRound.PassOrder[0])
	}
	return append([]string(nil), factions...)
}

func rotateToFirst(order []string, first string) []string {
	idx := -1
	for i, f := range order {
		if f == first {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return append([]string(nil), order...)
	}
	rot := append([]string(nil), order[idx:]...)
	rot = append(rot, order[:idx]...)
	return rot
}

func actionMayTriggerLeech(action string) bool {
	if action == "" {
		return false
	}
	parts := strings.Split(action, ".")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "UP-") {
			return true
		}
		// Any explicit build coordinate can trigger leech.
		if regexp.MustCompile(`^[A-L][0-9]{1,2}$`).MatchString(p) {
			return true
		}
		// Witches stronghold action places a dwelling directly.
		if strings.HasPrefix(p, "ACT-SH-D-") {
			return true
		}
	}
	return false
}

func findLatestEligibleLeechSourceRow(round *snellmanRoundData, sourceFaction string, beforeRow int) int {
	if round == nil || sourceFaction == "" || len(round.Rows) == 0 {
		return -1
	}
	if beforeRow >= len(round.Rows) {
		beforeRow = len(round.Rows) - 1
	}
	for row := beforeRow; row >= 0; row-- {
		tok := strings.TrimSpace(round.Rows[row][sourceFaction])
		if tok == "" || tok == "L" || tok == "DL" {
			continue
		}
		if actionMayTriggerLeech(tok) {
			return row
		}
	}
	return -1
}

func extractSnellmanAction(parts []string) string {
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		lowerPart := strings.ToLower(part)
		if strings.Contains(lowerPart, "for network") ||
			strings.Contains(lowerPart, "for cult") ||
			strings.Contains(lowerPart, "for resources") ||
			strings.Contains(lowerPart, "for passing") {
			continue
		}
		// Preserve explicit +TRACK action strings (e.g. "+EARTH. pass BON9")
		// so deferred Cultists bonuses can be backtracked correctly.
		if strings.HasPrefix(part, "+") && regexp.MustCompile(`^\+[A-Za-z0-9]+(\.|$)`).MatchString(part) {
			if i > 0 {
				prev := strings.TrimSpace(parts[i-1])
				prevLower := strings.ToLower(prev)
				if strings.HasPrefix(prevLower, "action bon2") || regexp.MustCompile(`(?i)^action\s+fav\d+`).MatchString(prev) {
					if strings.HasSuffix(prev, ".") {
						return prev + " " + part
					}
					return prev + ". " + part
				}
			}
			return part
		}

		// Check if it's clearly an action
		lower := strings.ToLower(part)
		if strings.Contains(lower, "action") ||
			strings.Contains(lower, "pass") ||
			strings.Contains(lower, "convert") ||
			strings.Contains(lower, "build") ||
			strings.Contains(lower, "upgrade") ||
			strings.Contains(lower, "transform") ||
			strings.Contains(lower, "burn") ||
			strings.Contains(lower, "dig") ||
			strings.Contains(lower, "send p to") ||
			strings.Contains(lower, "advance") {

			// Clean up any preceding resources (e.g. "1/2/4/0 transform...")
			re := regexp.MustCompile(`(?i)(action|pass|convert|build|upgrade|transform|burn|dig|send p to|advance).*`)
			if match := re.FindString(part); match != "" {
				return match
			}
			return part
		}

		// Skip resource columns
		if strings.Contains(strings.ToUpper(part), "VP") || strings.Contains(strings.ToUpper(part), "PW") ||
			strings.Contains(part, "/") || part == "C" || part == "W" || part == "P" {
			continue
		}
		// Skip numeric deltas
		if isNumericWithDelta(part) {
			continue
		}
		return part
	}
	return ""
}

func convertActionToConcise(action, faction string, isSetup bool, cultDelta int) string {
	action = strings.TrimSpace(action)
	lowerAction := strings.ToLower(action)

	// Skip non-actions
	if action == "setup" || action == "other_income_for_faction" ||
		action == "score_resources" || strings.Contains(action, "[opponent") ||
		strings.Contains(lowerAction, "for network") || strings.Contains(lowerAction, "for cult") {
		return ""
	}

	// Build dwelling: "build E7" -> "S-E7" in setup, "E7" in game
	if strings.HasPrefix(lowerAction, "build ") {
		if strings.Contains(action, ".") {
			return convertCompoundActionToConcise(action, faction, cultDelta)
		}
		coord := strings.TrimSpace(action[len("build "):])
		coord = strings.ToUpper(strings.TrimSpace(coord))
		if isSetup {
			return fmt.Sprintf("S-%s", coord)
		}
		return coord
	}

	// Upgrade: "upgrade E5 to TP" -> "UP-TH-E5" (note: TP -> TH for Trading House)
	if strings.HasPrefix(lowerAction, "upgrade ") {
		re := regexp.MustCompile(`(?i)upgrade (\w+) to (\w+)`)
		if m := re.FindStringSubmatch(action); len(m) > 2 {
			coord := strings.ToUpper(m[1])
			building := snellmanBuildingToConscise(strings.ToUpper(m[2]))
			result := fmt.Sprintf("UP-%s-%s", building, coord)

			// Check for favor tile: "+FAV11" -> ".FAV-E1" (FAV11 = Earth +1)
			if strings.Contains(action, "+FAV") {
				favRe := regexp.MustCompile(`\+FAV(\d+)`)
				if fm := favRe.FindStringSubmatch(action); len(fm) > 1 {
					favCode := snellmanFavToConscise(fm[1])
					result += fmt.Sprintf(".%s", favCode)
				}
			}
			return result
		}
	}

	// Pass: "Pass BON1" -> "BON-SPD" during setup, "PASS-BON-SPD" during rounds
	if strings.EqualFold(action, "pass") {
		return "PASS"
	}
	if strings.HasPrefix(lowerAction, "pass ") {
		bonusCard := strings.TrimSpace(action[len("pass "):])
		bonusCode := snellmanBonToConscise(strings.ToUpper(bonusCard))
		if isSetup {
			// During setup, this is just bonus card selection (not a pass action)
			return bonusCode
		}
		return fmt.Sprintf("PASS-%s", bonusCode)
	}

	// Leech: "Leech 1 from engineers" -> "L"
	if strings.HasPrefix(lowerAction, "leech ") {
		return "L"
	}

	// Decline: "Decline X from Y" -> "DL"
	if strings.HasPrefix(lowerAction, "decline ") {
		return "DL"
	}

	// Power actions: "action ACT6" -> "ACT6"
	if strings.HasPrefix(lowerAction, "action ") {
		return convertPowerActionToConcise(action, faction, cultDelta)
	}

	// Burn + action: "burn 6. action ACT6. transform F2 to gray. build D4"
	if strings.HasPrefix(lowerAction, "burn ") {
		return convertCompoundActionToConcise(action, faction, cultDelta)
	}

	// Send priest: "send p to WATER" -> "->W"
	if strings.HasPrefix(lowerAction, "send p to ") {
		if strings.Contains(action, ".") {
			return convertCompoundActionToConcise(action, faction, cultDelta)
		}
		return convertSendPriestToConcise(action, cultDelta)
	}

	// Digging: "dig 1. build G6" -> "G6" (implicit dig)
	if strings.HasPrefix(lowerAction, "dig ") {
		converted := convertCompoundActionToConcise(action, faction, cultDelta)
		if converted != "" {
			return converted
		}
		// Keep +DIG fallback for factions where a standalone dig action may be meaningful.
		if strings.ToLower(faction) == "darklings" {
			return ""
		}
		return "+DIG"
	}

	// Transform: "transform G2 to yellow"
	if strings.HasPrefix(lowerAction, "transform ") {
		return convertCompoundActionToConcise(action, faction, cultDelta)
	}

	// Advance shipping: "advance ship" -> "+SHIP"
	if strings.HasPrefix(lowerAction, "advance ship") {
		return "+SHIP"
	}

	// Advance digging: "advance dig" -> "+DIG"
	if strings.HasPrefix(lowerAction, "advance dig") {
		return "+DIG"
	}

	// Convert: "convert 1PW to 1C"
	if strings.HasPrefix(lowerAction, "convert ") {
		return convertCompoundActionToConcise(action, faction, cultDelta)
	}

	// Cult advance: "+EARTH. Leech 1..." -> usually a side effect, but check if it's compound
	if strings.HasPrefix(action, "+") {
		return convertCompoundActionToConcise(action, faction, cultDelta)
	}

	return ""
}

func convertPowerActionToConcise(action, faction string, cultDelta int) string {
	// "action ACT6" -> "ACT6"
	// "action BON2. +WATER" -> "ACT-BON2.+W"
	// "action ACT5. build F3" -> "ACT5.F3"
	return convertCompoundActionToConcise(action, faction, cultDelta)
}

func convertCompoundActionToConcise(action, faction string, cultDelta int) string {
	// "burn 6. action ACT6. transform F2 to gray. build D4"
	parts := strings.Split(action, ".")
	var resultParts []string

	// Check for Bonus Card Cult Action (BON2 + Track)
	var hasBon2 bool
	var cultTrack string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		partLower := strings.ToLower(part)
		if strings.HasPrefix(partLower, "action ") {
			act := strings.ToUpper(strings.TrimSpace(part[len("action "):]))
			if act == "BON2" {
				hasBon2 = true
			}
		}
		if strings.HasPrefix(part, "+") && !strings.HasPrefix(strings.ToUpper(part), "+SHIP") && !strings.HasPrefix(strings.ToUpper(part), "+FAV") {
			cultTrack = trackToShort(strings.TrimPrefix(part, "+"))
		}
	}

	for i, part := range parts {
		part = strings.TrimSpace(part)
		partLower := strings.ToLower(part)

		// Burn
		if strings.HasPrefix(partLower, "burn ") {
			re := regexp.MustCompile(`(?i)burn (\d+)`)
			if m := re.FindStringSubmatch(part); len(m) > 1 {
				resultParts = append(resultParts, fmt.Sprintf("BURN%s", m[1]))
			}
		}

		// Action
		if strings.HasPrefix(partLower, "action ") {
			actType := strings.TrimSpace(part[len("action "):])
			actType = strings.ToUpper(actType)

			// Auren SH: ACTA +2TRACK -> ACT-SH-<track>
			if actType == "ACTA" {
				merged := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					re := regexp.MustCompile(`^\+2(FIRE|WATER|EARTH|AIR)$`)
					if m := re.FindStringSubmatch(strings.ToUpper(nextPart)); len(m) > 1 {
						resultParts = append(resultParts, fmt.Sprintf("ACT-SH-%s", trackToShort(m[1])))
						parts[j] = ""
						merged = true
						break
					}
				}
				if !merged {
					resultParts = append(resultParts, "ACTA")
				}
				continue
			}

			// Favor action: action FAV6. +FIRE -> ACT-FAV-F
			if favActionRe := regexp.MustCompile(`(?i)^FAV(\d+)$`); favActionRe.MatchString(actType) {
				merged := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					re := regexp.MustCompile(`^\+(FIRE|WATER|EARTH|AIR)$`)
					if m := re.FindStringSubmatch(strings.ToUpper(nextPart)); len(m) > 1 {
						resultParts = append(resultParts, fmt.Sprintf("ACT-FAV-%s", trackToShort(m[1])))
						parts[j] = ""
						merged = true
						break
					}
				}
				if !merged {
					if m := favActionRe.FindStringSubmatch(actType); len(m) > 1 {
						resultParts = append(resultParts, snellmanFavToConscise(m[1]))
					}
				}
				continue
			}

			// Witches Ride: ACTW + Build -> ACT-SH-D-COORD
			if actType == "ACTW" {
				// Look ahead for build/transform
				mergeFound := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					if strings.HasPrefix(strings.ToLower(nextPart), "build ") {
						coord := strings.TrimSpace(nextPart[len("build "):])
						coord = strings.ToUpper(strings.TrimSpace(coord))
						resultParts = append(resultParts, fmt.Sprintf("ACT-SH-D-%s", coord))
						parts[j] = "" // Consume the build part
						mergeFound = true
						break
					}
				}
				if !mergeFound {
					resultParts = append(resultParts, actType)
				}
				continue
			}

			// Giants Stronghold: ACTG + Transform -> ACT-SH-S-COORD
			if actType == "ACTG" {
				mergeFound := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					if strings.HasPrefix(strings.ToLower(nextPart), "transform ") {
						// transform X to Y
						re := regexp.MustCompile(`(?i)transform (\w+) to (\w+)`)
						if m := re.FindStringSubmatch(nextPart); len(m) > 2 {
							coord := strings.ToUpper(m[1])
							resultParts = append(resultParts, fmt.Sprintf("ACT-SH-S-%s", coord))
							parts[j] = "" // Consume transform part
							mergeFound = true
							break
						}
					}
				}
				if !mergeFound {
					resultParts = append(resultParts, actType)
				}
				continue
			}

			// Nomads Stronghold (Sandstorm):
			// ACTN + Transform -> ACT-SH-T-COORD
			// ACTN + Build -> ACT-SH-T-COORD.COORD
			if actType == "ACTN" {
				mergeFound := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					if strings.HasPrefix(strings.ToLower(nextPart), "transform ") {
						re := regexp.MustCompile(`(?i)transform (\w+) to (\w+)`)
						if m := re.FindStringSubmatch(nextPart); len(m) > 2 {
							coord := strings.ToUpper(m[1])
							resultParts = append(resultParts, fmt.Sprintf("ACT-SH-T-%s", coord))
							parts[j] = "" // consume transform
							// If a build follows on the same coord, fold it into the same token.
							for k := j + 1; k < len(parts); k++ {
								buildPart := strings.TrimSpace(parts[k])
								if strings.HasPrefix(strings.ToLower(buildPart), "build ") {
									buildCoord := strings.ToUpper(strings.TrimSpace(buildPart[len("build "):]))
									resultParts[len(resultParts)-1] = resultParts[len(resultParts)-1] + "." + buildCoord
									parts[k] = "" // consume build
									break
								}
							}
							mergeFound = true
							break
						}
					}
					if strings.HasPrefix(strings.ToLower(nextPart), "build ") {
						coord := strings.ToUpper(strings.TrimSpace(nextPart[len("build "):]))
						resultParts = append(resultParts, fmt.Sprintf("ACT-SH-T-%s.%s", coord, coord))
						parts[j] = "" // consume build
						mergeFound = true
						break
					}
				}
				if !mergeFound {
					// Keep legacy token if the source log is incomplete.
					resultParts = append(resultParts, actType)
				}
				continue
			}

			if actType == "BON1" {
				// Bonus spade action can be represented directly as ACTS-<coord>
				// when followed by a transform/build target in the same compound action.
				// If BON1 directly leads to a build, include the build token as well:
				// ACTS-<coord>.<coord>
				merged := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					if strings.HasPrefix(strings.ToLower(nextPart), "transform ") {
						re := regexp.MustCompile(`(?i)transform (\w+) to (\w+)`)
						if m := re.FindStringSubmatch(nextPart); len(m) > 2 {
							coord := strings.ToUpper(m[1])
							resultParts = append(resultParts, fmt.Sprintf("ACTS-%s", coord))
							parts[j] = "" // consume transform part
							merged = true
							break
						}
					}
					if strings.HasPrefix(strings.ToLower(nextPart), "build ") {
						coord := strings.TrimSpace(nextPart[len("build "):])
						coord = strings.ToUpper(strings.TrimSpace(coord))
						resultParts = append(resultParts, fmt.Sprintf("ACTS-%s", coord))
						resultParts = append(resultParts, coord)
						parts[j] = "" // consume build part
						merged = true
						break
					}
				}
				if !merged {
					resultParts = append(resultParts, "ACT-BON-SPD")
				}
			} else if actType == "BON2" && cultTrack != "" {
				resultParts = append(resultParts, fmt.Sprintf("ACT-BON-%s", cultTrack))
			} else {
				resultParts = append(resultParts, actType)
			}
		}

		// Cult track advance (e.g. +WATER)
		if strings.HasPrefix(part, "+") {
			// Skip if it was merged into BON2
			if hasBon2 && cultTrack != "" && !strings.HasPrefix(strings.ToUpper(part), "+SHIP") && !strings.HasPrefix(strings.ToUpper(part), "+FAV") {
				continue
			}

			track := strings.TrimPrefix(part, "+")
			upperTrack := strings.ToUpper(strings.TrimSpace(track))
			if regexp.MustCompile(`^\d`).MatchString(upperTrack) {
				// VP/resource delta strings like +15vp... are not concise actions.
				continue
			}
			if townRe := regexp.MustCompile(`^TW(\d+)$`); townRe.MatchString(upperTrack) {
				if m := townRe.FindStringSubmatch(upperTrack); len(m) > 1 {
					resultParts = append(resultParts, snellmanTownToConcise(m[1]))
				}
				continue
			}
			if upperTrack == "SHIP" {
				resultParts = append(resultParts, "+SHIP")
			} else if strings.HasPrefix(strings.ToUpper(track), "FAV") {
				// Handle +FAV9 -> FAV-F1
				favRe := regexp.MustCompile(`(?i)FAV(\d+)`)
				if fm := favRe.FindStringSubmatch(track); len(fm) > 1 {
					favCode := snellmanFavToConscise(fm[1])
					resultParts = append(resultParts, favCode)
				}
			} else {
				// Keep explicit +TRACK actions (e.g. +EARTH from leech)
				shortTrack := trackToShort(track)
				if shortTrack != "" {
					resultParts = append(resultParts, fmt.Sprintf("+%s", shortTrack))
				}
			}
		}

		// Transform
		if strings.HasPrefix(partLower, "transform ") {
			re := regexp.MustCompile(`(?i)transform (\w+) to (\w+)`)
			if m := re.FindStringSubmatch(part); len(m) > 2 {
				coord := strings.ToUpper(m[1])
				color := snellmanColorToShort(m[2])
				if strings.EqualFold(color, factionHomeColorShort(faction)) {
					resultParts = append(resultParts, fmt.Sprintf("T-%s", coord))
				} else {
					resultParts = append(resultParts, fmt.Sprintf("T-%s-%s", coord, color))
				}
			}
		}

		// Build
		if strings.HasPrefix(partLower, "build ") {
			coord := strings.TrimSpace(part[len("build "):])
			resultParts = append(resultParts, strings.ToUpper(strings.TrimSpace(coord)))
		}

		// Upgrade
		if strings.HasPrefix(partLower, "upgrade ") {
			re := regexp.MustCompile(`(?i)upgrade (\w+) to (\w+)`)
			if m := re.FindStringSubmatch(part); len(m) > 2 {
				coord := strings.ToUpper(m[1])
				building := snellmanBuildingToConscise(strings.ToUpper(m[2]))
				res := fmt.Sprintf("UP-%s-%s", building, coord)

				// Check for favor tile in same part
				if strings.Contains(part, "+FAV") {
					favRe := regexp.MustCompile(`\+FAV(\d+)`)
					if fm := favRe.FindStringSubmatch(part); len(fm) > 1 {
						favCode := snellmanFavToConscise(fm[1])
						res += fmt.Sprintf(".%s", favCode)
					}
				}
				resultParts = append(resultParts, res)
			}
		}

		// Convert
		if strings.HasPrefix(partLower, "convert ") {
			// convert 1PW to 1C -> C1PW:1C
			re := regexp.MustCompile(`(?i)convert (.*) to (.*)`)
			if m := re.FindStringSubmatch(part); len(m) > 2 {
				cost := parseSnellmanResources(m[1])
				reward := parseSnellmanResources(m[2])
				resultParts = append(resultParts, fmt.Sprintf("C%s:%s", cost, reward))
			}
		}

		// Advance shipping / digging can appear inside compound rows:
		// e.g. "burn 3. convert 5PW to 1P. advance ship"
		if strings.HasPrefix(partLower, "advance ship") {
			resultParts = append(resultParts, "+SHIP")
		}
		if strings.HasPrefix(partLower, "advance dig") {
			resultParts = append(resultParts, "+DIG")
		}

		// Send priest in compound action
		if strings.HasPrefix(strings.ToLower(part), "send p to ") {
			resultParts = append(resultParts, convertSendPriestToConcise(part, cultDelta))
		}

		// Pass
		if strings.EqualFold(part, "pass") {
			resultParts = append(resultParts, "PASS")
		}
		if strings.HasPrefix(partLower, "pass ") {
			bonusCard := strings.TrimSpace(part[len("pass "):])
			bonusCode := snellmanBonToConscise(strings.ToUpper(bonusCard))
			resultParts = append(resultParts, fmt.Sprintf("PASS-%s", bonusCode))
		}
	}

	return strings.Join(resultParts, ".")
}

func extractCultDelta(parts []string) int {
	cultPattern := regexp.MustCompile(`^\d+/\d+/\d+/\d+$`)
	deltaPattern := regexp.MustCompile(`^[+-]\d+$`)
	for i, part := range parts {
		p := strings.TrimSpace(part)
		if !cultPattern.MatchString(p) {
			continue
		}
		if i == 0 {
			return 0
		}
		prev := strings.TrimSpace(parts[i-1])
		if !deltaPattern.MatchString(prev) {
			return 0
		}
		var delta int
		if _, err := fmt.Sscanf(prev, "%d", &delta); err != nil {
			return 0
		}
		if delta < 0 {
			delta = -delta
		}
		return delta
	}
	return 0
}

func convertSendPriestToConcise(action string, cultDelta int) string {
	track := strings.TrimSpace(action)
	track = strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(track), "send p to "), "send p to ")
	track = strings.ToUpper(strings.TrimSpace(track))
	if strings.Contains(track, ".") {
		track = strings.Split(track, ".")[0]
	}
	short := trackToShort(track)
	if cultDelta >= 1 && cultDelta <= 3 {
		return fmt.Sprintf("->%s%d", short, cultDelta)
	}
	return fmt.Sprintf("->%s", short)
}

func factionHomeColorShort(faction string) string {
	switch strings.ToLower(faction) {
	case "engineers", "dwarves":
		return "Gy"
	case "darklings", "alchemists":
		return "Bk"
	case "cultists", "halflings", "nomads":
		return "Br"
	case "auren", "witches":
		return "G"
	case "chaos magicians", "giants":
		return "R"
	case "fakirs":
		return "Y"
	case "mermaids", "swarmlings":
		return "Bl"
	default:
		return ""
	}
}

func parseSnellmanResources(s string) string {
	// 1PW -> 1PW usually, but sometimes 1PW means 1W (Worker) in Snellman logs?
	// Heuristic: 1PW -> 1W REMOVED. 1PW should mean 1 Power.
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	return s
}

func trackToShort(track string) string {
	switch strings.ToUpper(track) {
	case "FIRE":
		return "F"
	case "WATER":
		return "W"
	case "EARTH":
		return "E"
	case "AIR":
		return "A"
	default:
		return track[:1]
	}
}

func isKnownFaction(name string) bool {
	name = normalizeFactionNameForMatching(name)
	known := map[string]bool{
		"engineers": true, "darklings": true, "cultists": true, "witches": true,
		"halflings": true, "auren": true, "alchemists": true, "chaosmagicians": true,
		"nomads": true, "fakirs": true, "giants": true, "dwarves": true,
		"mermaids": true, "swarmlings": true,
	}
	return known[name]
}

func normalizeFactionNameForMatching(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, " ", "")
	return n
}

func getBonusCardsMinusRemoved(removed []string) []string {
	// Map Snellman codes to concise notation codes
	allMap := map[string]string{
		"BON1":  "BON-SPD",
		"BON2":  "BON-4C",
		"BON3":  "BON-6C",
		"BON4":  "BON-SHIP",
		"BON5":  "BON-WP",
		"BON6":  "BON-BB",
		"BON7":  "BON-TP",
		"BON8":  "BON-P",
		"BON9":  "BON-DW",
		"BON10": "BON-SHIP-VP",
	}

	removedSet := make(map[string]bool)
	for _, r := range removed {
		removedSet[r] = true
	}

	// Ordered list to maintain consistent output
	allOrdered := []string{"BON1", "BON2", "BON3", "BON4", "BON5", "BON6", "BON7", "BON8", "BON9", "BON10"}

	var result []string
	for _, b := range allOrdered {
		if !removedSet[b] {
			result = append(result, allMap[b])
		}
	}
	return result
}

func formatFactionList(factions []string) string {
	var titled []string
	for _, f := range factions {
		titled = append(titled, strings.Title(f))
	}
	return strings.Join(titled, ", ")
}

func formatFactionHeader(factions []string) string {
	var parts []string
	for _, f := range factions {
		parts = append(parts, fmt.Sprintf("%-12s", strings.Title(f)))
	}
	return strings.Join(parts, " | ")
}

func formatTableRow(actions []string) string {
	var parts []string
	for _, a := range actions {
		parts = append(parts, fmt.Sprintf("%-12s", a))
	}
	return strings.Join(parts, " | ")
}

func isNumericWithDelta(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '+' && c != '-' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func factionLikelyDelayedLeechSource(faction, action string) bool {
	if strings.ToLower(faction) != "cultists" {
		return false
	}
	parts := strings.Split(action, ".")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "UP-") {
			return true
		}
		if regexp.MustCompile(`^[A-I][0-9]{1,2}$`).MatchString(p) {
			return true
		}
	}
	return false
}

// snellmanBonToConscise converts Snellman bonus card codes (BON1-BON10) to concise notation codes
func snellmanBonToConscise(code string) string {
	mapping := map[string]string{
		"BON1":  "BON-SPD",     // Spade
		"BON2":  "BON-4C",      // +4 Cult
		"BON3":  "BON-6C",      // 6 Coins
		"BON4":  "BON-SHIP",    // Shipping
		"BON5":  "BON-WP",      // Worker + Power
		"BON6":  "BON-BB",      // SH/SA VP
		"BON7":  "BON-TP",      // TP VP
		"BON8":  "BON-P",       // Priest
		"BON9":  "BON-DW",      // Dwelling VP
		"BON10": "BON-SHIP-VP", // Shipping VP
	}
	if result, ok := mapping[code]; ok {
		return result
	}
	return code // Return original if not found
}

// snellmanBuildingToConscise converts Snellman building codes to concise notation codes
// TP (Trading Post) -> TH (Trading House), others stay the same
func snellmanBuildingToConscise(code string) string {
	if code == "TP" {
		return "TH"
	}
	return code
}

// snellmanFavToConscise converts Snellman favor tile codes (1-12) to concise notation codes
// Based on notation.go:
// FAV 1,5,9 = Fire (3,2,1 step), FAV 2,6,10 = Water, FAV 3,7,11 = Earth, FAV 4,8,12 = Air
func snellmanFavToConscise(num string) string {
	mapping := map[string]string{
		// Fire tiles
		"1": "FAV-F3", "5": "FAV-F2", "9": "FAV-F1",
		// Water tiles
		"2": "FAV-W3", "6": "FAV-W2", "10": "FAV-W1",
		// Earth tiles
		"3": "FAV-E3", "7": "FAV-E2", "11": "FAV-E1",
		// Air tiles
		"4": "FAV-A3", "8": "FAV-A2", "12": "FAV-A1",
	}
	if result, ok := mapping[num]; ok {
		return result
	}
	return "FAV-F1" // Default
}

// snellmanTownToConcise converts Snellman town tile ids (TW1-TW8) to concise VP-coded town tokens.
func snellmanTownToConcise(num string) string {
	mapping := map[string]string{
		"1": "TW5VP",
		"2": "TW6VP",
		"3": "TW7VP",
		"4": "TW8VP",
		"5": "TW9VP",
		"6": "TW11VP",
		"7": "TW2VP",
		"8": "TW4VP",
	}
	if result, ok := mapping[num]; ok {
		return result
	}
	return "TW5VP"
}

func parseRound(s string) (int, error) {
	var r int
	_, err := fmt.Sscanf(s, "%d", &r)
	return r, err
}

func snellmanColorToShort(color string) string {
	switch strings.ToLower(color) {
	case "brown":
		return "Br"
	case "black":
		return "Bk"
	case "blue":
		return "Bl"
	case "green":
		return "G"
	case "gray":
		return "Gy"
	case "red":
		return "R"
	case "yellow":
		return "Y"
	}
	return "Y" // Default/Unknown
}
