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

	for i, line := range lines[:min(20, len(lines))] {
		lower := strings.ToLower(line)
		fmt.Printf("DEBUG IsSnellman: Checking line %d: '%s'\n", i, line)

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

	// Round tracking
	var rounds []*snellmanRoundData
	var currentRound *snellmanRoundData

	inSetupPhase := true
	setupActions := make(map[string][]string) // faction -> setup dwelling placements
	setupPassOrder := []string{}              // Track setup phase pass order for Round 1
	actionsAddedThisTurn := make(map[string]bool)
	lastMainActionRow := make(map[string]int)

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

		fmt.Printf("DEBUG: Processing line: %s\n", line)

		// Parse scoring tiles
		if strings.HasPrefix(line, "Round ") && strings.Contains(line, "scoring:") {
			re := regexp.MustCompile(`Round \d+ scoring: (SCORE\d+)`)
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				scoringTiles = append(scoringTiles, m[1])
			}
			continue
		}

		// Parse removed bonus cards
		if strings.HasPrefix(line, "Removing tile") {
			re := regexp.MustCompile(`Removing tile (BON\d+)`)
			if m := re.FindStringSubmatch(line); len(m) > 1 {
				removedBonusCards = append(removedBonusCards, m[1])
			}
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
					// Determine turn order from previous round's pass order
					var turnOrder []string
					if len(rounds) > 0 {
						prevRound := rounds[len(rounds)-1]
						if len(prevRound.PassOrder) == len(factions) {
							turnOrder = prevRound.PassOrder
						} else {
							turnOrder = factions // Fallback to setup order
						}
					} else if len(setupPassOrder) == len(factions) {
						// Round 1: use setup phase pass order
						turnOrder = setupPassOrder
					} else {
						turnOrder = factions // Fallback to setup order
					}

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
					// Determine turn order from previous round's pass order
					var turnOrder []string
					if len(rounds) > 0 {
						prevRound := rounds[len(rounds)-1]
						if len(prevRound.PassOrder) == len(factions) {
							turnOrder = prevRound.PassOrder
						} else {
							turnOrder = factions // Fallback to setup order
						}
					} else if len(setupPassOrder) == len(factions) {
						// Round 1: use setup phase pass order
						turnOrder = setupPassOrder
					} else {
						turnOrder = factions // Fallback to setup order
					}

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
				} else {
					// Same round, new turn: reset per-turn merge tracking.
					actionsAddedThisTurn = make(map[string]bool)
				}
			}
			inSetupPhase = false
			continue
		}

		// Parse faction action lines
		parts := strings.Split(line, "\t")

		if strings.Contains(line, "pass BON8") {
			fmt.Printf("DEBUG: Found pass BON8 line: '%s'\n", line)
			fmt.Printf("DEBUG: Parts len: %d\n", len(parts))
			if len(parts) > 0 {
				fmt.Printf("DEBUG: Faction: '%s', IsKnown: %v\n", strings.ToLower(strings.TrimSpace(parts[0])), isKnownFaction(strings.ToLower(strings.TrimSpace(parts[0]))))
			}
		}

		if len(parts) < 2 {
			continue
		}

		factionName := strings.ToLower(strings.TrimSpace(parts[0]))
		if factionName == "engineers" && strings.Contains(line, "pass") {
			fmt.Printf("DEBUG: Processing Engineers pass line: %s\n", line)
		}

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
		cultLeechPattern := regexp.MustCompile(`^\+(EARTH|WATER|FIRE|AIR)\.\s*Leech`)
		if m := cultLeechPattern.FindStringSubmatch(action); len(m) > 1 && currentRound != nil && !inSetupPhase {
			cultBonus := "+" + trackToShort(m[1])

			// Backtrack to the faction's latest main action row and add the cult bonus there.
			if rowIdx, ok := lastMainActionRow[factionName]; ok &&
				rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
				currentRound.Rows[rowIdx][factionName] != "" {
				currentRound.Rows[rowIdx][factionName] += "." + cultBonus
				fmt.Printf("DEBUG: Backtracked cult bonus '%s' to action at row %d -> %s\n", cultBonus, rowIdx, currentRound.Rows[rowIdx][factionName])
			}

			// Preserve the source faction for row placement by keeping only the leech part.
			action = extractTrailingLeechAction(action)
		}

		// Convert action
		conciseAction := convertActionToConcise(action, factionName, inSetupPhase)
		if conciseAction == "" {
			continue
		}

		// DEBUG
		if factionName == "engineers" && strings.Contains(action, "pass") {
			fmt.Printf("DEBUG: Converted Engineers pass: '%s' -> '%s'\n", action, conciseAction)
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
					fmt.Printf("DEBUG: Setup faction %s passed (order: %d)\n", factionName, len(setupPassOrder))
				}
			}
		} else if currentRound != nil {
			// Check if this is a leech action - leeches should be their own row, not merged
			isLeechAction := conciseAction == "L" || conciseAction == "DL"

			if isLeechAction {
				targetRow := len(currentRound.Rows)
				if sourceFaction := extractLeechSourceFaction(action); sourceFaction != "" {
					if rowIdx, ok := lastMainActionRow[sourceFaction]; ok {
						targetRow = rowIdx
					}
				}
				rowIdx := placeRoundAction(currentRound, factionName, conciseAction, targetRow)
				fmt.Printf("DEBUG: Placed leech '%s' for %s in Round %d at row %d\n", conciseAction, factionName, currentRound.Number, rowIdx)
			} else if actionsAddedThisTurn[factionName] {
				// Merge follow-up non-leech actions in the same turn with faction's main action.
				if rowIdx, ok := lastMainActionRow[factionName]; ok &&
					rowIdx >= 0 && rowIdx < len(currentRound.Rows) &&
					currentRound.Rows[rowIdx][factionName] != "" {
					currentRound.Rows[rowIdx][factionName] += "." + conciseAction
					fmt.Printf("DEBUG: Merged Action '%s' for %s in Round %d at row %d -> %s\n", conciseAction, factionName, currentRound.Number, rowIdx, currentRound.Rows[rowIdx][factionName])
				} else {
					rowIdx := placeRoundAction(currentRound, factionName, conciseAction, len(currentRound.Rows))
					lastMainActionRow[factionName] = rowIdx
					fmt.Printf("DEBUG: Appended Action '%s' for %s in Round %d at row %d\n", conciseAction, factionName, currentRound.Number, rowIdx)
				}
			} else {
				rowIdx := placeRoundAction(currentRound, factionName, conciseAction, len(currentRound.Rows))
				lastMainActionRow[factionName] = rowIdx
				actionsAddedThisTurn[factionName] = true
				fmt.Printf("DEBUG: Appended Action '%s' for %s in Round %d at row %d\n", conciseAction, factionName, currentRound.Number, rowIdx)
			}

			// Track pass order: when a faction passes, add to PassOrder (if not already)
			if strings.Contains(conciseAction, "PASS-") {
				alreadyPassed := false
				for _, p := range currentRound.PassOrder {
					if p == factionName {
						alreadyPassed = true
						break
					}
				}
				if !alreadyPassed {
					currentRound.PassOrder = append(currentRound.PassOrder, factionName)
					fmt.Printf("DEBUG: Faction %s passed (order: %d) in Round %d\n", factionName, len(currentRound.PassOrder), currentRound.Number)
				}
			}
		} else {
			fmt.Printf("DEBUG: Dropped action because currentRound is nil: %s\n", conciseAction)
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
			result = append(result, fmt.Sprintf("TurnOrder: %s", strings.Join(round.TurnOrder, ", ")))
		}

		result = append(result, strings.Repeat("-", 60))
		result = append(result, formatFactionHeader(factions))
		result = append(result, strings.Repeat("-", 60))

		// Output chronological rows.
		for i := 0; i < len(round.Rows); i++ {
			var row []string
			for _, f := range factions {
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
	source := strings.ToLower(strings.TrimSpace(m[1]))
	if isKnownFaction(source) {
		return source
	}
	return ""
}

func extractSnellmanAction(parts []string) string {
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
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
			strings.Contains(lower, "advance") {

			// Clean up any preceding resources (e.g. "1/2/4/0 transform...")
			re := regexp.MustCompile(`(?i)(action|pass|convert|build|upgrade|transform|burn|dig|advance).*`)
			if match := re.FindString(part); match != "" {
				return match
			}
			return part
		}

		// Skip resource columns
		if strings.Contains(part, "VP") || strings.Contains(part, "PW") ||
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

func convertActionToConcise(action, faction string, isSetup bool) string {
	action = strings.TrimSpace(action)

	// Skip non-actions
	if action == "setup" || action == "other_income_for_faction" ||
		action == "score_resources" || strings.Contains(action, "[opponent") {
		return ""
	}

	// Build dwelling: "build E7" -> "S-E7" in setup, "E7" in game
	if strings.HasPrefix(action, "build ") {
		coord := strings.TrimPrefix(action, "build ")
		coord = strings.ToUpper(strings.TrimSpace(coord))
		if isSetup {
			return fmt.Sprintf("S-%s", coord)
		}
		return coord
	}

	// Upgrade: "upgrade E5 to TP" -> "UP-TH-E5" (note: TP -> TH for Trading House)
	if strings.HasPrefix(action, "upgrade ") {
		re := regexp.MustCompile(`upgrade (\w+) to (\w+)`)
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
	if strings.HasPrefix(action, "Pass ") || strings.HasPrefix(action, "pass ") {
		bonusCard := strings.TrimPrefix(strings.TrimPrefix(action, "Pass "), "pass ")
		bonusCode := snellmanBonToConscise(strings.ToUpper(bonusCard))
		if isSetup {
			// During setup, this is just bonus card selection (not a pass action)
			return bonusCode
		}
		return fmt.Sprintf("PASS-%s", bonusCode)
	}

	// Leech: "Leech 1 from engineers" -> "L"
	if strings.HasPrefix(action, "Leech ") {
		return "L"
	}

	// Decline: "Decline X from Y" -> "DL"
	if strings.HasPrefix(action, "Decline ") {
		return "DL"
	}

	// Power actions: "action ACT6" -> "ACT6"
	if strings.HasPrefix(action, "action ") {
		return convertPowerActionToConcise(action)
	}

	// Burn + action: "burn 6. action ACT6. transform F2 to gray. build D4"
	if strings.HasPrefix(action, "burn ") {
		return convertCompoundActionToConcise(action)
	}

	// Send priest: "send p to WATER" -> "->W"
	if strings.HasPrefix(action, "send p to ") {
		track := strings.TrimPrefix(action, "send p to ")
		track = strings.ToUpper(strings.TrimSpace(track))
		if strings.Contains(track, ".") {
			track = strings.Split(track, ".")[0]
		}
		return fmt.Sprintf("->%s", trackToShort(track))
	}

	// Digging: "dig 1. build G6" -> "G6" (implicit dig)
	if strings.HasPrefix(action, "dig ") {
		// Extract build part if present
		if strings.Contains(action, "build ") {
			parts := strings.Split(action, ".")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "build ") {
					coord := strings.TrimPrefix(part, "build ")
					return strings.ToUpper(strings.TrimSpace(coord))
				}
			}
		}
		return "+DIG" // Fallback if no build
	}

	// Transform: "transform G2 to yellow"
	if strings.HasPrefix(action, "transform ") {
		return convertCompoundActionToConcise(action)
	}

	// Advance shipping: "advance ship" -> "+SHIP"
	if strings.HasPrefix(action, "advance ship") {
		return "+SHIP"
	}

	// Advance digging: "advance dig" -> "+DIG"
	if strings.HasPrefix(action, "advance dig") {
		return "+DIG"
	}

	// Convert: "convert 1PW to 1C"
	if strings.HasPrefix(action, "convert ") {
		return convertCompoundActionToConcise(action)
	}

	// Cult advance: "+EARTH. Leech 1..." -> usually a side effect, but check if it's compound
	if strings.HasPrefix(action, "+") {
		return convertCompoundActionToConcise(action)
	}

	return ""
}

func convertPowerActionToConcise(action string) string {
	// "action ACT6" -> "ACT6"
	// "action BON2. +WATER" -> "ACT-BON2.+W"
	// "action ACT5. build F3" -> "ACT5.F3"
	return convertCompoundActionToConcise(action)
}

func convertCompoundActionToConcise(action string) string {
	// "burn 6. action ACT6. transform F2 to gray. build D4"
	parts := strings.Split(action, ".")
	var resultParts []string

	if strings.Contains(action, "+FAV") {
		fmt.Printf("DEBUG CONVERT: Parsing compound action with FAV: '%s'\n", action)
	}

	// Check for Bonus Card Cult Action (BON2 + Track)
	var hasBon2 bool
	var cultTrack string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "action ") {
			act := strings.ToUpper(strings.TrimPrefix(part, "action "))
			if act == "BON2" {
				hasBon2 = true
			}
		}
		if strings.HasPrefix(part, "+") && !strings.HasPrefix(part, "+SHIP") && !strings.HasPrefix(part, "+FAV") {
			cultTrack = trackToShort(strings.TrimPrefix(part, "+"))
		}
	}

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// Burn
		if strings.HasPrefix(part, "burn ") {
			re := regexp.MustCompile(`burn (\d+)`)
			if m := re.FindStringSubmatch(part); len(m) > 1 {
				resultParts = append(resultParts, fmt.Sprintf("BURN%s", m[1]))
			}
		}

		// Action
		if strings.HasPrefix(part, "action ") {
			actType := strings.TrimPrefix(part, "action ")
			actType = strings.ToUpper(actType)

			// Witches Ride: ACTW + Build -> ACT-SH-D-COORD
			if actType == "ACTW" {
				// Look ahead for build/transform
				mergeFound := false
				for j := i + 1; j < len(parts); j++ {
					nextPart := strings.TrimSpace(parts[j])
					if strings.HasPrefix(nextPart, "build ") {
						coord := strings.TrimPrefix(nextPart, "build ")
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
					if strings.HasPrefix(nextPart, "transform ") {
						// transform X to Y
						re := regexp.MustCompile(`transform (\w+) to (\w+)`)
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

			if actType == "BON1" {
				resultParts = append(resultParts, "ACT-BON-SPD")
			} else if actType == "BON2" && cultTrack != "" {
				resultParts = append(resultParts, fmt.Sprintf("ACT-BON-%s", cultTrack))
			} else {
				resultParts = append(resultParts, actType)
			}
		}

		// Cult track advance (e.g. +WATER)
		if strings.HasPrefix(part, "+") {
			// Skip if it was merged into BON2
			if hasBon2 && cultTrack != "" && !strings.HasPrefix(part, "+SHIP") && !strings.HasPrefix(part, "+FAV") {
				continue
			}

			track := strings.TrimPrefix(part, "+")
			if track == "SHIP" {
				resultParts = append(resultParts, "+SHIP")
			} else if strings.HasPrefix(track, "FAV") {
				// Handle +FAV9 -> FAV-F1
				favRe := regexp.MustCompile(`FAV(\d+)`)
				if fm := favRe.FindStringSubmatch(track); len(fm) > 1 {
					favCode := snellmanFavToConscise(fm[1])
					fmt.Printf("DEBUG CONVERT: Found FAV code %s -> %s\n", fm[1], favCode)
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
		if strings.HasPrefix(part, "transform ") {
			re := regexp.MustCompile(`transform (\w+) to (\w+)`)
			if m := re.FindStringSubmatch(part); len(m) > 2 {
				coord := strings.ToUpper(m[1])
				color := snellmanColorToShort(m[2])
				resultParts = append(resultParts, fmt.Sprintf("T-%s-%s", coord, color))
			}
		}

		// Build
		if strings.HasPrefix(part, "build ") {
			coord := strings.TrimPrefix(part, "build ")
			resultParts = append(resultParts, strings.ToUpper(strings.TrimSpace(coord)))
		}

		// Upgrade
		if strings.HasPrefix(part, "upgrade ") {
			re := regexp.MustCompile(`upgrade (\w+) to (\w+)`)
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
		if strings.HasPrefix(part, "convert ") {
			// convert 1PW to 1C -> C1PW:1C
			re := regexp.MustCompile(`convert (.*) to (.*)`)
			if m := re.FindStringSubmatch(part); len(m) > 2 {
				cost := parseSnellmanResources(m[1])
				reward := parseSnellmanResources(m[2])
				resultParts = append(resultParts, fmt.Sprintf("C%s:%s", cost, reward))
			}
		}

		// Pass
		if strings.HasPrefix(part, "pass ") {
			bonusCard := strings.TrimPrefix(part, "pass ")
			bonusCode := snellmanBonToConscise(strings.ToUpper(bonusCard))
			resultParts = append(resultParts, fmt.Sprintf("PASS-%s", bonusCode))
		}
	}

	return strings.Join(resultParts, ".")
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
	known := map[string]bool{
		"engineers": true, "darklings": true, "cultists": true, "witches": true,
		"halflings": true, "auren": true, "alchemists": true, "chaos magicians": true,
		"nomads": true, "fakirs": true, "giants": true, "dwarves": true,
		"mermaids": true, "swarmlings": true,
	}
	return known[name]
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
