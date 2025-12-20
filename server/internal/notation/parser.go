package notation

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/replay"
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

				// Handle chained actions (e.g. "S-C4.S-D5")
				codes := strings.Split(part, ".")
				for _, code := range codes {
					code = strings.TrimSpace(code)
					if code == "" {
						continue
					}

					action, err := parseActionCode(playerID, code)
					if err != nil {
						// If we fail to parse, maybe it was a header we missed?
						// But we should have caught it above.
						return nil, fmt.Errorf("failed to parse code '%s' for player '%s': %w", code, playerID, err)
					}
					actions = append(actions, action)
				}
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

	if strings.HasPrefix(code, "ACT") {
		// ACT1, ACT2...
		return &LogPowerAction{
			PlayerID:   playerID,
			ActionCode: code,
		}, nil
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
		// T-C4
		coord := strings.TrimPrefix(code, "T-")
		hex := parseHex(coord)
		return game.NewTransformAndBuildAction(playerID, hex, false), nil
	}

	// Default: Build Dwelling (e.g. "C4")
	if isCoord(code) {
		hex := parseHex(code)
		return game.NewTransformAndBuildAction(playerID, hex, true), nil
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
	h, err := replay.ConvertLogCoordToAxial(s)
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
	_, err := replay.ConvertLogCoordToAxial(s)
	return err == nil
}
