package notation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// ParseConciseLog parses a concise log string into GameActions
func ParseConciseLog(content string) ([]game.Action, error) {
	lines := strings.Split(content, "\n")
	actions := make([]game.Action, 0)

	// State for grid parsing
	colToPlayer := make(map[int]string)
	headerFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(line, "---") {
			continue
		}

		// Skip Game Settings headers (Key: Value)
		if strings.Contains(line, ": ") && !strings.Contains(line, "|") {
			// Could be "Game: Base Game" or "TurnOrder: ..."
			// We skip these for now as we only return []game.Action
			continue
		}

		// Skip Round headers
		if strings.HasPrefix(line, "Round ") {
			continue
		}

		// Check for grid header line
		if strings.Contains(line, "|") {
			// Check if this is a header row or data row
			// Header row usually has faction names. Data row has codes.

			// If we see "Nomads" or "Witches" etc.
			// Let's assume if it doesn't have "S-", "UP-", "->", "PASS", "T-", it's a header?
			// But "PASS" is short.
			// Let's use the fact that headers are usually Title Case words, actions are codes.

			if strings.Contains(line, "Nomads") || strings.Contains(line, "Witches") ||
				strings.Contains(line, "Halflings") || strings.Contains(line, "Darklings") ||
				strings.Contains(line, "Fakirs") || strings.Contains(line, "Giants") ||
				strings.Contains(line, "Chaos Magicians") || strings.Contains(line, "Swarmlings") ||
				strings.Contains(line, "Engineers") || strings.Contains(line, "Dwarves") ||
				strings.Contains(line, "Alchemists") || strings.Contains(line, "Mermaids") ||
				strings.Contains(line, "Auren") || strings.Contains(line, "Cultists") {

				// It's a header
				parts := strings.Split(line, "|")
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
					// fmt.Printf("Warning: No player for col %d in line: %s\n", i, line)
					continue
				}

				// Parse the action code
				// Note: parseActionCode handles chained actions (separated by .) by returning a LogCompoundAction
				action, err := parseActionCode(playerID, part)
				if err != nil {
					fmt.Printf("Error parsing action '%s': %v\n", part, err)
					continue
				}
				actions = append(actions, action)
			}
		} else if !headerFound && strings.Contains(line, ": ") {
			// Fallback for linear format "Player: Code"
			// But we are skipping ": " lines above (Settings).
			// So this block is unreachable if we skip all ": " lines.
			// We should only skip ": " if it's NOT a player action line.
			// Player action line: "Nomads: S-F3"
			// Settings line: "Game: Base Game"
			// We can check if the key is a known faction?
			// Or check if value is an action code?
			// For now, the concise log is GRID only. So we can ignore linear format support or make it stricter.
			// Let's assume Grid only for now.
		}
	}

	return actions, nil
}

func parseActionCode(playerID, code string) (game.Action, error) {
	// Simple parser based on prefixes
	if code == "PASS" {
		var bonusCard *game.BonusCardType
		return game.NewPassAction(playerID, bonusCard), nil
	}
	if code == "+SHIP" {
		return game.NewAdvanceShippingAction(playerID), nil
	}
	if code == "DL" {
		// Return standard action or Log action?
		// Generator uses "DL" -> DeclinePowerLeechAction
		// But standard action requires amount. Log doesn't have amount for DL (usually).
		// Wait, BGAParser emits Decline with amount. Generator prints "DL".
		// So we lose the amount in concise log for Decline.
		// That's fine, usually amount doesn't matter for decline (except for stats).
		return game.NewDeclinePowerLeechAction(playerID, 0), nil
	}

	// Burn: BURN<N>
	if strings.HasPrefix(code, "BURN") {
		amountStr := strings.TrimPrefix(code, "BURN")
		amount, err := strconv.Atoi(amountStr)
		if err == nil {
			return &LogBurnAction{
				PlayerID: playerID,
				Amount:   amount,
			}, nil
		}
	}

	// Favor Tile: FAV-<Code>
	if strings.HasPrefix(code, "FAV-") {
		return &LogFavorTileAction{
			PlayerID: playerID,
			Tile:     code,
		}, nil
	}

	// Special Action: ACT-SH-D-<Coord> (Witches Ride)
	if strings.HasPrefix(code, "ACT-SH-D-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
	}

	// Compound Action: Code.Code
	if strings.Contains(code, ".") {
		parts := strings.Split(code, ".")
		var actions []game.Action
		for _, part := range parts {
			action, err := parseActionCode(playerID, part)
			if err != nil {
				return nil, err
			}
			actions = append(actions, action)
		}
		return &LogCompoundAction{Actions: actions}, nil
	}

	// Auren Stronghold: ACT-SH-<Track>
	if strings.HasPrefix(code, "ACT-SH-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
	}

	// Favor Tile Action: ACT-FAV-<Track>
	if strings.HasPrefix(code, "ACT-FAV-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
	}

	// Bonus Card Spade: ACTS-<Coord>
	if strings.HasPrefix(code, "ACTS-") {
		return &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
	}

	// Digging: +DIG
	if code == "+DIG" {
		return game.NewAdvanceDiggingAction(playerID), nil
	}

	// Leech: L<N> or just L
	if strings.HasPrefix(code, "L") {
		if len(code) > 1 && unicode.IsDigit(rune(code[1])) {
			amount, err := strconv.Atoi(code[1:])
			if err == nil {
				return &LogAcceptLeechAction{
					PlayerID:    playerID,
					PowerAmount: amount,
				}, nil
			}
		}
		// Just "L" -> assume 1 for now
		return &LogAcceptLeechAction{
			PlayerID:    playerID,
			PowerAmount: 1,
		}, nil
	}

	// Conversion: C<Cost>:<Reward>
	if strings.HasPrefix(code, "C") && strings.Contains(code, ":") {
		parts := strings.Split(strings.TrimPrefix(code, "C"), ":")
		if len(parts) == 2 {
			cost := parseResourceString(parts[0])
			reward := parseResourceString(parts[1])
			return &LogConversionAction{
				PlayerID: playerID,
				Cost:     cost,
				Reward:   reward,
			}, nil
		}
	}

	if strings.HasPrefix(code, "ACT") {
		// ACT1, ACT2...
		return &LogPowerAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
	}

	// Town: TW<VP>VP
	if strings.HasPrefix(code, "TW") && strings.HasSuffix(code, "VP") {
		vpStr := strings.TrimSuffix(strings.TrimPrefix(code, "TW"), "VP")
		vp, err := strconv.Atoi(vpStr)
		if err == nil {
			return &LogTownAction{
				PlayerID: playerID,
				VP:       vp,
			}, nil
		}
	}

	if strings.HasPrefix(code, "->") {
		// ->F
		trackCode := strings.TrimPrefix(code, "->")
		track := parseCultShortCode(trackCode)
		return &game.SendPriestToCultAction{
			BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: playerID},
			Track:         track,
			SpacesToClimb: 3, // Default
		}, nil
	}
	if strings.HasPrefix(code, "UP-") {
		// UP-TH-C4
		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid upgrade code")
		}
		buildingType := parseBuildingShortCode(parts[1])
		hex := parseHex(parts[2])
		return game.NewUpgradeBuildingAction(playerID, hex, buildingType), nil
	}
	if strings.HasPrefix(code, "S-") {
		// S-C4
		coord := strings.TrimPrefix(code, "S-")
		hex := parseHex(coord)
		return game.NewSetupDwellingAction(playerID, hex), nil
	}
	if strings.HasPrefix(code, "T-") {
		// T-C4 or T-C4-Y
		parts := strings.Split(code, "-")
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
	if isCoord(code) {
		hex := parseHex(code)
		return game.NewTransformAndBuildAction(playerID, hex, true, models.TerrainTypeUnknown), nil
	}

	return nil, fmt.Errorf("unknown code: %s", code)
}

func parseCultShortCode(s string) game.CultTrack {
	switch s {
	case "F":
		return game.CultFire
	case "W":
		return game.CultWater
	case "E":
		return game.CultEarth
	case "A":
		return game.CultAir
	}
	return game.CultFire
}

func parseBuildingShortCode(s string) models.BuildingType {
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
	switch s {
	case "P", "Br":
		return models.TerrainPlains
	case "S", "Bk":
		return models.TerrainSwamp
	case "L", "Bl":
		return models.TerrainLake
	case "F", "G":
		return models.TerrainForest
	case "M", "Gy":
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
	// Regex to find "N unit" where unit is P, W, PW, VP, C
	re := regexp.MustCompile(`(\d+)(P|W|PW|VP|C)`)
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
