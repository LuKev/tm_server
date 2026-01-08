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

// ParseConciseLog parses a concise log string into LogItems
func ParseConciseLog(content string) ([]LogItem, error) {
	lines := strings.Split(content, "\n")
	items := make([]LogItem, 0)

	// State for grid parsing
	colToPlayer := make(map[int]string)
	headerFound := false
	settings := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
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
			// Check if this is a header row or data row
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
					continue
				}

				// Parse the action code
				action, err := parseActionCode(playerID, part)
				if err != nil {
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

	return items, nil
}

func parseActionCode(playerID, code string) (game.Action, error) {
	// Simple parser based on prefixes
	if strings.HasPrefix(code, "PASS") {
		var bonusCard *game.BonusCardType
		if strings.HasPrefix(code, "PASS-") {
			cardCode := strings.TrimPrefix(code, "PASS-")
			cardType := ParseBonusCardCode(cardCode)
			bonusCard = &cardType
		}
		return game.NewPassAction(playerID, bonusCard), nil
	}
	if strings.HasPrefix(code, "BON") {
		// BON1, BON2 etc.
		return &LogBonusCardSelectionAction{
			PlayerID:  playerID,
			BonusCard: code,
		}, nil
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
				vpCost := amount - 1
				if vpCost < 0 {
					vpCost = 0
				}
				return &LogAcceptLeechAction{
					PlayerID:    playerID,
					PowerAmount: amount,
					VPCost:      vpCost,
				}, nil
			}
		}
		// Just "L" -> assume 1 for now (Cost 0)
		return &LogAcceptLeechAction{
			PlayerID:    playerID,
			PowerAmount: 1,
			VPCost:      0,
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
		// ->F or ->F3
		codeStr := strings.TrimPrefix(code, "->")
		if len(codeStr) == 0 {
			return nil, fmt.Errorf("invalid cult action code")
		}

		// First character is the track
		trackCode := string(codeStr[0])
		track := parseCultShortCode(trackCode)

		// Rest is spaces to climb (default 3)
		spaces := 3
		if len(codeStr) > 1 {
			if s, err := strconv.Atoi(codeStr[1:]); err == nil {
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
