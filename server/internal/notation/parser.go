package notation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// ConciseParseError includes source location for concise parsing failures.
type ConciseParseError struct {
	Line     int
	Column   int
	PlayerID string
	Token    string
	Cause    error
}

func (e *ConciseParseError) Error() string {
	if e == nil {
		return "concise parse error"
	}
	if e.PlayerID != "" {
		return fmt.Sprintf("line %d, column %d, player %q, token %q: %v", e.Line, e.Column, e.PlayerID, e.Token, e.Cause)
	}
	return fmt.Sprintf("line %d, column %d, token %q: %v", e.Line, e.Column, e.Token, e.Cause)
}

// ParseConciseLog parses a concise log string into LogItems
func ParseConciseLog(content string) ([]LogItem, error) {
	return parseConciseLog(content, false)
}

// ParseConciseLogStrict parses concise logs and fails fast with line/column context.
func ParseConciseLogStrict(content string) ([]LogItem, error) {
	return parseConciseLog(content, true)
}

func parseConciseLog(content string, strict bool) ([]LogItem, error) {
	lines := strings.Split(content, "\n")
	items := make([]LogItem, 0)

	// State for grid parsing
	colToPlayer := make(map[int]string)
	headerFound := false
	settings := make(map[string]string)

	for lineIdx, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(line, "---") {
			continue
		}

		// Parse Game Settings
		if strings.HasPrefix(line, "Game:") || strings.HasPrefix(line, "MiniExpansions:") || strings.HasPrefix(line, "ScoringTiles:") || strings.HasPrefix(line, "BonusCards:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key == "ScoringTiles" {
					// fmt.Printf("DEBUG: Parsed ScoringTiles: %s\n", value)
				}
				settings[key] = value
			}
			continue
		}

		if strings.HasPrefix(line, "StartingVPs:") {
			// StartingVPs: Halflings:20, Auren:20
			// We extract players from here
			value := strings.TrimPrefix(line, "StartingVPs:")
			parts := strings.Split(value, ",")
			for _, part := range parts {
				// part is "Halflings:20"
				pParts := strings.Split(part, ":")
				if len(pParts) >= 1 {
					playerName := strings.TrimSpace(pParts[0])
					// Add Player setting: Player:Halflings -> Halflings
					settings["Player:"+playerName] = playerName
					// fmt.Printf("DEBUG: Parsed player from StartingVPs: %s\n", playerName)

					// Add StartingVP setting
					if len(pParts) >= 2 {
						settings["StartingVP:"+playerName] = strings.TrimSpace(pParts[1])
					}
				}
			}
			continue
		}

		// Emit settings if we have them and haven't emitted yet
		if len(settings) > 0 && len(items) == 0 {
			items = append(items, GameSettingsItem{Settings: settings})
			// Keep settings map for future additions if needed, but usually they are at top
		}

		// Parse Round headers
		if strings.HasPrefix(line, "Round ") {
			roundStr := strings.TrimPrefix(line, "Round ")
			round, err := strconv.Atoi(roundStr)
			if err == nil {
				items = append(items, RoundStartItem{Round: round})
			}
			continue
		}

		// Parse TurnOrder
		if strings.HasPrefix(line, "TurnOrder:") {
			orderStr := strings.TrimPrefix(line, "TurnOrder:")
			players := strings.Split(orderStr, ",")
			turnOrder := make([]string, 0)
			for _, p := range players {
				turnOrder = append(turnOrder, strings.TrimSpace(p))
			}
			// Update the last RoundStartItem if it exists
			if len(items) > 0 {
				if rs, ok := items[len(items)-1].(RoundStartItem); ok {
					rs.TurnOrder = turnOrder
					items[len(items)-1] = rs
				}
			}
			continue
		}

		// Check for grid header line
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			nonEmpty := 0
			matched := 0
			for _, part := range parts {
				playerID := strings.TrimSpace(part)
				if playerID == "" {
					continue
				}
				nonEmpty++
				if isFactionHeaderName(playerID) {
					matched++
				}
			}
			// Header rows consist solely of faction display names.
			if nonEmpty >= 2 && matched == nonEmpty {
				colToPlayer = make(map[int]string) // Reset for new block
				for i, part := range parts {
					playerID := strings.TrimSpace(part)
					if playerID != "" {
						colToPlayer[i] = playerID
					}
				}
				headerFound = true
				continue
			}
		}

		// Parse data line
		if headerFound && strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				playerID, ok := colToPlayer[i]
				if !ok {
					continue
				}

				// Parse the action code
				action, err := parseActionCode(playerID, part)
				if err != nil {
					if strict {
						return nil, &ConciseParseError{
							Line:     lineIdx + 1,
							Column:   i + 1,
							PlayerID: playerID,
							Token:    part,
							Cause:    err,
						}
					}
					fmt.Printf("Error parsing action '%s': %v\n", part, err)
					continue
				}
				// Create action item
				item := ActionItem{
					Action: action,
				}
				items = append(items, item)
			}
		}
	}

	// Header-only concise logs should still emit settings.
	if len(settings) > 0 && len(items) == 0 {
		items = append(items, GameSettingsItem{Settings: settings})
	}

	return items, nil
}

func isFactionHeaderName(playerID string) bool {
	switch strings.ToLower(strings.TrimSpace(playerID)) {
	case "nomads",
		"fakirs",
		"chaos magicians",
		"giants",
		"swarmlings",
		"mermaids",
		"witches",
		"auren",
		"halflings",
		"cultists",
		"alchemists",
		"darklings",
		"engineers",
		"dwarves":
		return true
	default:
		return false
	}
}

func parseActionCode(playerID, code string) (game.Action, error) {
	return parseActionCodeWithContext(playerID, code, false)
}

func parseActionCodeWithContext(playerID, code string, inCompound bool) (game.Action, error) {
	code = strings.TrimSpace(code)
	if strings.HasPrefix(code, "@") {
		inner := strings.TrimSpace(strings.TrimPrefix(code, "@"))
		if inner == "" {
			return nil, fmt.Errorf("empty pre-income action")
		}
		action, err := parseActionCodeWithContext(playerID, inner, inCompound)
		if err != nil {
			return nil, err
		}
		return &LogPreIncomeAction{Action: action}, nil
	}
	upperCode := strings.ToUpper(code)

	// Compound Action: Code.Code
	if strings.Contains(code, ".") {
		parts := strings.Split(code, ".")
		var actions []game.Action

		// Pre-process to merge T-X + X patterns into single TransformAndBuild
		// Example: T-A7.A7 becomes single action with buildDwelling=true
		mergedParts := mergeTransformAndBuildTokens(parts)

		for _, part := range mergedParts {
			action, err := parseActionCodeWithContext(playerID, part, true)
			if err != nil {
				return nil, err
			}
			actions = append(actions, action)
		}
		return &LogCompoundAction{Actions: actions}, nil
	}

	// Simple parser based on prefixes
	if strings.HasPrefix(upperCode, "PASS") {
		var bonusCard *game.BonusCardType
		if strings.HasPrefix(upperCode, "PASS-") {
			cardCode := strings.TrimPrefix(upperCode, "PASS-")
			cardType := ParseBonusCardCode(cardCode)
			bonusCard = &cardType
		}
		return game.NewPassAction(playerID, bonusCard), nil
	}
	if strings.HasPrefix(upperCode, "BON") {
		// BON1, BON2 etc.
		return &LogBonusCardSelectionAction{
			PlayerID:  playerID,
			BonusCard: upperCode,
		}, nil
	}
	if upperCode == "+SHIP" {
		return game.NewAdvanceShippingAction(playerID), nil
	}
	if len(upperCode) == 2 && strings.HasPrefix(upperCode, "+") {
		if track, ok := parseCultShortCodeOk(string(upperCode[1])); ok {
			return &LogCultistAdvanceAction{
				PlayerID: playerID,
				Track:    track,
			}, nil
		}
	}

	// Decline leech: DL, DL2, DL-Witches, DL2-Witches
	if m := regexp.MustCompile(`(?i)^DL(\d+)?(?:-(.+))?$`).FindStringSubmatch(code); len(m) > 0 {
		from := ""
		if len(m) > 2 {
			from = strings.TrimSpace(m[2])
		}
		return &LogDeclineLeechAction{PlayerID: playerID, FromPlayerID: from}, nil
	}

	// Burn: BURN<N>
	if strings.HasPrefix(upperCode, "BURN") {
		amountStr := strings.TrimPrefix(upperCode, "BURN")
		amount, err := strconv.Atoi(amountStr)
		if err == nil {
			return &LogBurnAction{
				PlayerID: playerID,
				Amount:   amount,
			}, nil
		}
	}

	// Favor Tile: FAV-<Code>
	if strings.HasPrefix(upperCode, "FAV-") {
		return &LogFavorTileAction{
			PlayerID: playerID,
			Tile:     upperCode,
		}, nil
	}

	// Special Action: ACT-SH-D-<Coord> (Witches Ride)
	if strings.HasPrefix(upperCode, "ACT-SH-D-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Auren Stronghold: ACT-SH-<Track>
	if strings.HasPrefix(upperCode, "ACT-SH-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Favor Tile Action: ACT-FAV-<Track>
	if strings.HasPrefix(upperCode, "ACT-FAV-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Bonus Card cult action, Mermaids town action, Engineers bridge action
	if strings.HasPrefix(upperCode, "ACT-BON-") || strings.HasPrefix(upperCode, "ACT-TOWN-") || strings.HasPrefix(upperCode, "ACT-BR-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Bonus Card Spade: ACTS-<Coord>
	if strings.HasPrefix(upperCode, "ACTS-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Digging: +DIG
	if upperCode == "+DIG" {
		return game.NewAdvanceDiggingAction(playerID), nil
	}

	// Accept leech: L, L2, L-Witches, L2-Witches
	if m := regexp.MustCompile(`(?i)^L(\d+)?(?:-(.+))?$`).FindStringSubmatch(code); len(m) > 0 {
		amount := 1
		explicit := false
		if m[1] != "" {
			if n, err := strconv.Atoi(m[1]); err == nil {
				amount = n
				explicit = true
			}
		}
		from := ""
		if len(m) > 2 {
			from = strings.TrimSpace(m[2])
		}
		vpCost := amount - 1
		if vpCost < 0 {
			vpCost = 0
		}
		return &LogAcceptLeechAction{
			PlayerID:     playerID,
			FromPlayerID: from,
			PowerAmount:  amount,
			VPCost:       vpCost,
			Explicit:     explicit,
		}, nil
	}

	// Conversion: C<Cost>:<Reward>
	if strings.HasPrefix(upperCode, "C") && strings.Contains(upperCode, ":") {
		parts := strings.Split(strings.TrimPrefix(upperCode, "C"), ":")
		if len(parts) == 2 {
			if !inCompound {
				// In strict replay notation, conversions must be chained with a main action
				// (e.g. "C1PW:1C.PASS-..."), not represented as a standalone turn action.
				return nil, fmt.Errorf("standalone conversion is not a legal main action; chain it with a main action in the same token")
			}
			return &LogConversionAction{
				PlayerID: playerID,
				Cost:     parseResourceString(parts[0]),
				Reward:   parseResourceString(parts[1]),
			}, nil
		}
	}

	if strings.HasPrefix(upperCode, "ACT") {
		if ParsePowerActionCode(upperCode) == game.PowerActionUnknown {
			return nil, fmt.Errorf("unknown ACT code: %s", code)
		}
		// ACT1, ACT2...
		return &LogPowerAction{
			PlayerID:   playerID,
			ActionCode: upperCode,
		}, nil
	}

	// Town: TW<VP>VP
	if strings.HasPrefix(upperCode, "TW") && strings.HasSuffix(upperCode, "VP") {
		vpStr := strings.TrimSuffix(strings.TrimPrefix(upperCode, "TW"), "VP")
		vp, err := strconv.Atoi(vpStr)
		if err == nil {
			return &LogTownAction{
				PlayerID: playerID,
				VP:       vp,
			}, nil
		}
	}

	if strings.HasPrefix(code, "->") {
		// ->F or ->F3
		codeStr := strings.TrimPrefix(code, "->")
		if len(codeStr) == 0 {
			return nil, fmt.Errorf("invalid cult action code")
		}

		// First character is the track
		trackCode := string(strings.ToUpper(codeStr)[0])
		track := parseCultShortCode(trackCode)

		// Rest is spaces to climb (default 3)
		spaces := 3
		if len(codeStr) > 1 {
			if s, err := strconv.Atoi(strings.TrimSpace(codeStr[1:])); err == nil {
				spaces = s
			}
		}

		// fmt.Printf("DEBUG: Parsed Cult Action %s -> Track: %v, Spaces: %d\n", code, track, spaces)

		return &game.SendPriestToCultAction{
			BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: playerID},
			Track:         track,
			SpacesToClimb: spaces,
		}, nil
	}
	if strings.HasPrefix(upperCode, "UP-") {
		// UP-TH-C4
		parts := strings.Split(upperCode, "-")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid upgrade code")
		}
		buildingType := parseBuildingShortCode(parts[1])
		hex := parseHex(parts[2])
		return game.NewUpgradeBuildingAction(playerID, hex, buildingType), nil
	}
	if strings.HasPrefix(upperCode, "S-") {
		// S-C4
		coord := strings.TrimPrefix(upperCode, "S-")
		hex := parseHex(coord)
		return game.NewSetupDwellingAction(playerID, hex), nil
	}

	// DIGn- prefix: Terraform by n spades on a specific hex (no building).
	// Used to preserve Snellman intra-row ordering for interleaved "dig" + conversions.
	if strings.HasPrefix(upperCode, "DIG") {
		re := regexp.MustCompile(`^DIG(\d+)-([A-I]\d{1,2})$`)
		if m := re.FindStringSubmatch(upperCode); len(m) > 2 {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, fmt.Errorf("invalid DIG spade count: %q", m[1])
			}
			hex := parseHex(m[2])
			return &LogDigTransformAction{PlayerID: playerID, Spades: n, Target: hex}, nil
		}
		return nil, fmt.Errorf("invalid DIG code: %s", code)
	}

	// TB- prefix: Transform AND Build (merged from T-X + X pattern)
	// This is an internal notation created by mergeTransformAndBuildTokens
	if strings.HasPrefix(upperCode, "TB-") {
		// TB-C4 or TB-C4-Y
		parts := strings.Split(upperCode, "-")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid transform+build code: %s", code)
		}
		coord := parts[1]
		hex := parseHex(coord)

		targetTerrain := models.TerrainTypeUnknown
		if len(parts) > 2 {
			targetTerrain = parseTerrainShortCode(parts[2])
		}

		// buildDwelling=true for merged transform+build
		return game.NewTransformAndBuildAction(playerID, hex, true, targetTerrain), nil
	}

	if strings.HasPrefix(upperCode, "T-") {
		// T-C4 or T-C4-Y
		parts := strings.Split(upperCode, "-")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid transform code: %s", code)
		}
		coord := parts[1]
		hex := parseHex(coord)

		targetTerrain := models.TerrainTypeUnknown
		if len(parts) > 2 {
			targetTerrain = parseTerrainShortCode(parts[2])
		}

		return game.NewTransformAndBuildAction(playerID, hex, false, targetTerrain), nil
	}

	// Default: Build Dwelling (e.g. "C4")
	if isCoord(upperCode) {
		hex := parseHex(upperCode)
		return game.NewTransformAndBuildAction(playerID, hex, true, models.TerrainTypeUnknown), nil
	}

	return nil, fmt.Errorf("unknown code: %s", code)
}

func parseCultShortCode(s string) game.CultTrack {
	if track, ok := parseCultShortCodeOk(s); ok {
		return track
	}
	return game.CultFire
}

func parseCultShortCodeOk(s string) (game.CultTrack, bool) {
	s = strings.ToUpper(s)
	switch s {
	case "F":
		return game.CultFire, true
	case "W":
		return game.CultWater, true
	case "E":
		return game.CultEarth, true
	case "A":
		return game.CultAir, true
	}
	return game.CultFire, false
}

func parseBuildingShortCode(s string) models.BuildingType {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "D":
		return models.BuildingDwelling
	case "TH":
		return models.BuildingTradingHouse
	case "TE":
		return models.BuildingTemple
	case "SA":
		return models.BuildingSanctuary
	case "SH":
		return models.BuildingStronghold
	}
	return models.BuildingDwelling
}

func parseTerrainShortCode(s string) models.TerrainType {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "P", "BR":
		return models.TerrainPlains
	case "S", "BK":
		return models.TerrainSwamp
	case "L", "BL":
		return models.TerrainLake
	case "F", "G":
		return models.TerrainForest
	case "M", "GY":
		return models.TerrainMountain
	case "W", "R":
		return models.TerrainWasteland
	case "D", "Y":
		return models.TerrainDesert
	}
	return models.TerrainTypeUnknown
}

func parseHex(s string) board.Hex {
	s = strings.Trim(s, "()")
	// Try parsing as (Q,R) first (legacy/internal format)
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		q, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		r, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return board.Hex{Q: q, R: r}
		}
	}

	// Try parsing as Log Coord (e.g. F3)
	h, err := ConvertLogCoordToAxial(s)
	if err != nil {
		fmt.Printf("Error parsing hex %s: %v\n", s, err)
		return board.Hex{}
	}
	return h
}

func isCoord(s string) bool {
	// Check for (Q,R) format
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return true
	}
	// Check for Log format (e.g. F3)
	_, err := ConvertLogCoordToAxial(s)
	return err == nil
}
func parseResourceString(s string) map[models.ResourceType]int {
	res := make(map[models.ResourceType]int)
	s = strings.ToUpper(strings.TrimSpace(s))
	// Regex to find "N unit" where unit is P, W, PW, VP, C
	re := regexp.MustCompile(`(\d+)(PW|VP|P|W|C)`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		amount, _ := strconv.Atoi(match[1])
		unit := match[2]
		switch unit {
		case "PW":
			res[models.ResourcePower] += amount
		case "P":
			res[models.ResourcePriest] += amount
		case "W":
			res[models.ResourceWorker] += amount
		case "C":
			res[models.ResourceCoin] += amount
		case "VP":
			res[models.ResourceVictoryPoint] += amount
		}
	}
	return res
}

// mergeTransformAndBuildTokens merges T-X + X patterns into single tokens
// Example: ["BURN2", "ACT6", "T-A7", "T-B2", "A7"] -> ["BURN2", "ACT6", "TB-A7", "T-B2"]
// Where TB-X means transform AND build (with buildDwelling=true)
func mergeTransformAndBuildTokens(tokens []string) []string {
	result := make([]string, 0, len(tokens))

	// First pass: identify which transforms have matching builds
	// Key = coordinate, Value = index in tokens array
	transformCoords := make(map[string]int)
	for i, token := range tokens {
		if strings.HasPrefix(strings.ToUpper(token), "T-") {
			// Extract coordinate (e.g., T-A7 -> A7, T-A7-Y -> A7)
			parts := strings.Split(token, "-")
			if len(parts) >= 2 {
				coord := strings.ToUpper(parts[1])
				transformCoords[coord] = i
			}
		}
	}

	// Track which builds have been merged into their transforms
	buildMerged := make(map[int]bool)

	// Check each token to see if it's a bare coordinate that matches a transform
	for i, token := range tokens {
		// Check if this is a bare coordinate (e.g., "A7") that should be merged
		if isCoord(token) {
			tokenCoord := strings.ToUpper(token)
			if transformIdx, exists := transformCoords[tokenCoord]; exists && transformIdx < i {
				// This build should be merged with the earlier transform
				buildMerged[i] = true
			}
		}
	}

	// Second pass: emit tokens, converting T-X to TB-X where builds were merged
	for i, token := range tokens {
		// Skip builds that were merged
		if buildMerged[i] {
			continue
		}

		// Check if this is a transform that needs to be upgraded to transform+build
		if strings.HasPrefix(strings.ToUpper(token), "T-") {
			parts := strings.Split(token, "-")
			if len(parts) >= 2 {
				coord := strings.ToUpper(parts[1])
				// Check if there's a merged build for this coord
				hasBuildMerge := false
				for buildIdx := range buildMerged {
					if buildMerged[buildIdx] && strings.EqualFold(tokens[buildIdx], coord) {
						hasBuildMerge = true
						break
					}
				}
				if hasBuildMerge {
					// Emit as TB-X (transform and build)
					// If there's a terrain suffix (T-A7-Y), preserve it (TB-A7-Y)
					result = append(result, "TB"+token[1:])
					continue
				}
			}
		}

		result = append(result, token)
	}

	return result
}
