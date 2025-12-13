package notation

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseConciseLog parses the concise log string into a Log struct
func ParseConciseLog(content string) (*Log, error) {
	log := &Log{
		Rounds: []*RoundLog{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentRound *RoundLog
	var turnOrder []string

	inGrid := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Header parsing
		if strings.HasPrefix(line, "Game: ") {
			log.MapName = strings.TrimPrefix(line, "Game: ")
			continue
		}
		if strings.HasPrefix(line, "ScoringTiles: ") {
			log.ScoringTiles = strings.Split(strings.TrimPrefix(line, "ScoringTiles: "), ", ")
			continue
		}
		if strings.HasPrefix(line, "BonusCards: ") {
			log.BonusCards = strings.Split(strings.TrimPrefix(line, "BonusCards: "), ", ")
			continue
		}
		if strings.HasPrefix(line, "Options: ") {
			log.Options = strings.Split(strings.TrimPrefix(line, "Options: "), ", ")
			continue
		}

		// Round parsing
		if strings.HasPrefix(line, "Round ") {
			roundNumStr := strings.TrimPrefix(line, "Round ")
			roundNum, err := strconv.Atoi(roundNumStr)
			if err != nil {
				return nil, fmt.Errorf("invalid round number: %s", roundNumStr)
			}
			currentRound = &RoundLog{
				RoundNumber: roundNum,
				Actions:     []*GameAction{},
			}
			log.Rounds = append(log.Rounds, currentRound)
			inGrid = false
			continue
		}

		if strings.HasPrefix(line, "TurnOrder: ") {
			if currentRound == nil {
				return nil, fmt.Errorf("TurnOrder found outside of round")
			}
			orderStr := strings.TrimPrefix(line, "TurnOrder: ")
			turnOrder = strings.Split(orderStr, ", ")
			currentRound.TurnOrder = turnOrder
			continue
		}

		// Grid parsing
		if strings.HasPrefix(line, "---") {
			// Separator line
			continue
		}

		if strings.Contains(line, "|") {
			// Grid row
			parts := strings.Split(line, "|")

			// Check if this is the header row (contains faction names)
			cleanParts := []string{}
			for _, p := range parts {
				cleanParts = append(cleanParts, strings.TrimSpace(p))
			}

			// If parts match turn order, it's the header
			if len(cleanParts) == len(turnOrder) {
				match := true
				for i, p := range cleanParts {
					if p != turnOrder[i] {
						match = false
						break
					}
				}
				if match {
					inGrid = true
					continue
				}
			}

			if !inGrid {
				// Maybe header row didn't match exactly or something?
				// But we should assume we are in grid if we see pipes after TurnOrder
				// Let's assume if it's not the header, it's data
			}

			// Parse actions from cells
			for i, part := range parts {
				if i >= len(turnOrder) {
					break
				}
				actionStr := strings.TrimSpace(part)
				if actionStr == "" {
					continue
				}

				// Split multiple actions in one cell
				subActions := strings.Split(actionStr, ";")
				for _, sub := range subActions {
					sub = strings.TrimSpace(sub)
					if sub == "" {
						continue
					}

					action, err := ParseActionString(turnOrder[i], sub)
					if err != nil {
						return nil, fmt.Errorf("failed to parse action '%s': %v", sub, err)
					}
					currentRound.Actions = append(currentRound.Actions, action)
				}
			}
		}
	}

	return log, nil
}

// ParseActionString parses a single action string into a GameAction
func ParseActionString(faction, actionStr string) (*GameAction, error) {
	params := make(map[string]string)
	var actionType ActionType

	// Basic matching logic
	// TODO: Use regex for robust matching

	if strings.HasPrefix(actionStr, "Pass-") {
		actionType = ActionPass
		params["bonus"] = strings.TrimPrefix(actionStr, "Pass-")
	} else if strings.HasPrefix(actionStr, "L") || actionStr == "DL" {
		actionType = ActionLeech
		if actionStr == "DL" {
			params["decline"] = "true"
		}
	} else if strings.HasPrefix(actionStr, "CULT-") {
		actionType = ActionCultReaction
		params["track"] = strings.TrimPrefix(actionStr, "CULT-")
	} else if strings.HasPrefix(actionStr, "ORD-") {
		actionType = ActionDarklingsOrdination
		params["amount"] = strings.TrimPrefix(actionStr, "ORD-")
	} else if strings.HasPrefix(actionStr, "ACT-") {
		// Faction special actions
		actionType = ActionSpecial
		params["code"] = actionStr
	} else if strings.HasPrefix(actionStr, "ACT") {
		// Power actions ACT1-6
		actionType = ActionPower
		// Check for params like ACT1-C4-C5
		parts := strings.Split(actionStr, "-")
		params["code"] = parts[0]
		if len(parts) > 1 {
			params["args"] = strings.Join(parts[1:], "-")
		}
		if parts[0] == "ACTS" {
			actionType = ActionBonusSpade
		}
	} else if strings.HasPrefix(actionStr, "->") {
		actionType = ActionSendPriest
		params["target"] = strings.TrimPrefix(actionStr, "->")
	} else if strings.HasPrefix(actionStr, "B") && len(actionStr) > 1 && actionStr[1] >= '0' && actionStr[1] <= '9' {
		actionType = ActionBurn
		params["amount"] = strings.TrimPrefix(actionStr, "B")
	} else if strings.HasPrefix(actionStr, "C") && strings.Contains(actionStr, ":") {
		actionType = ActionConvert
		parts := strings.Split(actionStr, ":")
		params["in"] = strings.TrimPrefix(parts[0], "C")
		params["out"] = parts[1]
	} else if strings.HasPrefix(actionStr, "+") {
		actionType = ActionAdvance
		params["track"] = strings.TrimPrefix(actionStr, "+")
	} else if strings.Contains(actionStr, "-") {
		// Upgrade or Dig
		parts := strings.Split(actionStr, "-")
		if parts[0] == "D" || parts[0] == "DD" || parts[0] == "DDD" {
			actionType = ActionDigBuild
			params["spades"] = parts[0]
			params["coord"] = parts[1]
			if len(parts) > 2 && parts[2] == "T" {
				actionType = ActionTransform
			}
		} else {
			actionType = ActionUpgrade
			params["building"] = parts[0]
			params["coord"] = parts[1]
		}
	} else {
		// Build (Coordinate only)
		// Assume coordinate like C4, F5
		if regexp.MustCompile(`^[A-Z][0-9]+$`).MatchString(actionStr) {
			actionType = ActionBuild
			params["coord"] = actionStr
		} else {
			// Fallback
			return nil, fmt.Errorf("unknown action format: %s", actionStr)
		}
	}

	return &GameAction{
		Faction:  faction,
		Type:     actionType,
		Params:   params,
		Original: actionStr,
	}, nil
}
