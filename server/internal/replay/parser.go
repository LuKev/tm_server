package replay

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/models"
)

// LogEntry represents a single line from the game log
type LogEntry struct {
	Faction      models.FactionType
	VP           int
	VPDelta      int
	Coins        int
	CoinsDelta   int
	Workers      int
	WorkersDelta int
	Priests      int
	PriestsDelta int
	PowerBowls   PowerBowls
	CultTracks   CultTracks
	Action       string
	IsComment    bool
	CommentText  string
}

// PowerBowls represents the three power bowls
type PowerBowls struct {
	Bowl1 int
	Bowl2 int
	Bowl3 int
}

// CultTracks represents positions on the four cult tracks
type CultTracks struct {
	Fire  int
	Water int
	Earth int
	Air   int
}

// ParseGameLog parses the entire game log file
func ParseGameLog(filename string) ([]*LogEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	var entries []*LogEntry
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry, err := ParseLogLine(line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %v\nLine: %s", lineNum, err, line)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return entries, nil
}

// ParseLogLine parses a single line from the game log
func ParseLogLine(line string) (*LogEntry, error) {
	// Check if this is a comment/metadata line
	if strings.Contains(line, "show history") ||
	   strings.HasPrefix(line, "Default game") ||
	   strings.HasPrefix(line, "option ") ||
	   strings.HasPrefix(line, "Randomize") ||
	   strings.HasPrefix(line, "Round ") ||
	   strings.HasPrefix(line, "Removing") ||
	   strings.HasPrefix(line, "Player ") ||
	   strings.HasPrefix(line, "Scoring ") ||
	   strings.HasPrefix(line, "Converting") {
		return &LogEntry{
			IsComment:   true,
			CommentText: line,
		}, nil
	}

	// Split by tabs
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return &LogEntry{
			IsComment:   true,
			CommentText: line,
		}, nil
	}

	entry := &LogEntry{}

	// Parse faction (first non-empty part)
	factionStr := strings.TrimSpace(parts[0])
	if factionStr != "" {
		faction, err := ParseFaction(factionStr)
		if err != nil {
			// Not a faction line, treat as comment
			return &LogEntry{
				IsComment:   true,
				CommentText: line,
			}, nil
		}
		entry.Faction = faction
	}

	// Parse remaining fields
	// Format: VP, C, W, P, PW, cult, action
	// Example: 20 VP		10 C		2 W		0 P		3/9/0 PW		0/0/0/0		build E7
	// With deltas: 20 VP	-1	9 C	-1	3 W		0 P	-12	6/0/0 PW		0/0/0/0	1 	burn 6. action ACT6

	idx := 1
	for idx < len(parts) {
		part := strings.TrimSpace(parts[idx])
		if part == "" {
			idx++
			continue
		}

		// Check for VP
		if strings.HasSuffix(part, "VP") {
			vp, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(part, "VP")))
			entry.VP = vp

			// Check for delta
			if idx+1 < len(parts) {
				nextPart := strings.TrimSpace(parts[idx+1])
				if strings.HasPrefix(nextPart, "+") || strings.HasPrefix(nextPart, "-") {
					delta, _ := strconv.Atoi(nextPart)
					entry.VPDelta = delta
					idx++
				}
			}
			idx++
			continue
		}

		// Check for Coins
		if strings.HasSuffix(part, "C") && !strings.Contains(part, "ACT") {
			coins, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(part, "C")))
			entry.Coins = coins

			// Check for delta
			if idx+1 < len(parts) {
				nextPart := strings.TrimSpace(parts[idx+1])
				if strings.HasPrefix(nextPart, "+") || strings.HasPrefix(nextPart, "-") {
					delta, _ := strconv.Atoi(nextPart)
					entry.CoinsDelta = delta
					idx++
				}
			}
			idx++
			continue
		}

		// Check for Workers
		if strings.HasSuffix(part, "W") && len(part) < 5 {
			workers, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(part, "W")))
			entry.Workers = workers

			// Check for delta
			if idx+1 < len(parts) {
				nextPart := strings.TrimSpace(parts[idx+1])
				if strings.HasPrefix(nextPart, "+") || strings.HasPrefix(nextPart, "-") {
					delta, _ := strconv.Atoi(nextPart)
					entry.WorkersDelta = delta
					idx++
				}
			}
			idx++
			continue
		}

		// Check for Priests
		if strings.HasSuffix(part, "P") && len(part) < 5 && !strings.HasSuffix(part, "VP") && !strings.HasSuffix(part, "PW") {
			priests, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(part, "P")))
			entry.Priests = priests

			// Note: Do NOT read delta after priests - the next numeric field is the power delta (Bowl2 change during burns)
			// Priest deltas are not shown in the log format
			idx++
			continue
		}

		// Check for Power Bowls (e.g., "3/9/0 PW")
		if strings.HasSuffix(part, "PW") {
			pwStr := strings.TrimSpace(strings.TrimSuffix(part, "PW"))
			pwParts := strings.Split(pwStr, "/")
			if len(pwParts) == 3 {
				entry.PowerBowls.Bowl1, _ = strconv.Atoi(pwParts[0])
				entry.PowerBowls.Bowl2, _ = strconv.Atoi(pwParts[1])
				entry.PowerBowls.Bowl3, _ = strconv.Atoi(pwParts[2])
			}
			idx++
			continue
		}

		// Check for Cult Tracks (e.g., "0/0/0/0" or "0/1/1/0")
		if strings.Count(part, "/") == 3 && !strings.Contains(part, "PW") {
			cultParts := strings.Split(part, "/")
			if len(cultParts) == 4 {
				entry.CultTracks.Fire, _ = strconv.Atoi(cultParts[0])
				entry.CultTracks.Water, _ = strconv.Atoi(cultParts[1])
				entry.CultTracks.Earth, _ = strconv.Atoi(cultParts[2])
				entry.CultTracks.Air, _ = strconv.Atoi(cultParts[3])
			}
			idx++
			continue
		}

		// Skip numeric-only fields and delta markers (like "2 2" or "+1")
		// These are metadata fields before the actual action
		if _, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			idx++
			continue
		}
		if strings.HasPrefix(part, "+") || strings.HasPrefix(part, "-") {
			if _, err := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(part, "+"), "-")); err == nil {
				idx++
				continue
			}
		}

		// Everything else is part of the action
		actionParts := parts[idx:]
		entry.Action = strings.TrimSpace(strings.Join(actionParts, " "))
		break
	}

	return entry, nil
}

// ParseAction parses the action string into a structured format
func ParseAction(actionStr string) (ActionType, map[string]string, error) {
	actionStr = strings.TrimSpace(actionStr)

	// Remove numeric prefixes (e.g., "1 ", "2 2 ")
	re := regexp.MustCompile(`^[\d\s]+`)
	actionStr = re.ReplaceAllString(actionStr, "")
	actionStr = strings.TrimSpace(actionStr)

	params := make(map[string]string)

	// Parse different action types
	switch {
	case strings.HasPrefix(actionStr, "build "):
		// build E7
		coord := strings.TrimPrefix(actionStr, "build ")
		coord = strings.Fields(coord)[0] // Take first word in case there's more
		params["coord"] = coord
		return ActionBuild, params, nil

	case strings.HasPrefix(actionStr, "upgrade "):
		// upgrade E5 to TP
		// upgrade E5 to TE. +FAV11
		parts := strings.Split(actionStr, ".")
		firstPart := strings.TrimSpace(parts[0])
		fields := strings.Fields(firstPart)
		if len(fields) >= 4 && fields[2] == "to" {
			params["coord"] = fields[1]
			params["building"] = fields[3]

			// Check for favor tile selection
			if len(parts) > 1 {
				for _, part := range parts[1:] {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "+FAV") {
						params["favor_tile"] = strings.TrimPrefix(part, "+")
					}
				}
			}

			return ActionUpgrade, params, nil
		}
		return ActionUnknown, nil, fmt.Errorf("invalid upgrade format: %s", actionStr)

	case strings.HasPrefix(actionStr, "Pass ") || strings.HasPrefix(actionStr, "pass "):
		// Pass BON1
		bonusTile := strings.Fields(actionStr)[1]
		params["bonus"] = bonusTile
		return ActionPass, params, nil

	case strings.Contains(actionStr, "transform") && strings.Contains(actionStr, "build"):
		// transform F2 to gray. build D4
		// dig 1. build G6
		// burn 6. action ACT6. transform F2 to gray. build D4
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "burn ") {
				// Extract burn amount
				fields := strings.Fields(part)
				if len(fields) >= 2 {
					params["burn"] = fields[1]
				}
			} else if strings.HasPrefix(part, "action ") {
				// Extract power action type
				fields := strings.Fields(part)
				if len(fields) >= 2 {
					params["action_type"] = fields[1]
				}
			} else if strings.HasPrefix(part, "transform ") {
				fields := strings.Fields(part)
				if len(fields) >= 4 && fields[2] == "to" {
					params["transform_coord"] = fields[1]
					params["transform_color"] = fields[3]
				}
			} else if strings.HasPrefix(part, "build ") {
				coord := strings.TrimPrefix(part, "build ")
				params["coord"] = strings.Fields(coord)[0]
			} else if strings.HasPrefix(part, "dig ") {
				spades := strings.TrimPrefix(part, "dig ")
				params["spades"] = strings.Fields(spades)[0]
			}
		}
		return ActionTransformAndBuild, params, nil

	case strings.HasPrefix(actionStr, "dig ") && strings.Contains(actionStr, "build"):
		// dig 1. build G6
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "dig ") {
				spades := strings.TrimPrefix(part, "dig ")
				params["spades"] = strings.Fields(spades)[0]
			} else if strings.HasPrefix(part, "build ") {
				coord := strings.TrimPrefix(part, "build ")
				params["coord"] = strings.Fields(coord)[0]
			}
		}
		return ActionTransformAndBuild, params, nil

	case strings.HasPrefix(actionStr, "advance ship"):
		return ActionAdvanceShipping, params, nil

	case strings.HasPrefix(actionStr, "advance dig"):
		return ActionAdvanceDigging, params, nil

	case strings.HasPrefix(actionStr, "send p to "):
		// send p to WATER
		cult := strings.TrimPrefix(actionStr, "send p to ")
		params["cult"] = strings.Fields(cult)[0]
		return ActionSendPriest, params, nil

	case strings.HasPrefix(actionStr, "action "):
		// action ACT6, action BON1, action ACTW
		// action ACT5. build F3
		parts := strings.Split(actionStr, ".")

		// First part is always "action ACTX"
		firstPart := strings.TrimSpace(parts[0])
		actionFields := strings.Fields(firstPart)
		if len(actionFields) >= 2 {
			params["action_type"] = actionFields[1]
		}

		// Check if there are additional parts (e.g., "build F3")
		if len(parts) > 1 {
			for _, part := range parts[1:] {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "build ") {
					coord := strings.TrimPrefix(part, "build ")
					params["coord"] = strings.Fields(coord)[0]
				} else if strings.HasPrefix(part, "transform ") {
					fields := strings.Fields(part)
					if len(fields) >= 4 && fields[2] == "to" {
						params["transform_coord"] = fields[1]
						params["transform_color"] = fields[3]
					}
				}
			}
		}

		return ActionPowerAction, params, nil

	case strings.HasPrefix(actionStr, "burn "):
		// Can be part of compound action: "burn 3. action ACT2" or "burn 6. action ACT6. transform..."
		// Parse all parts of the compound action
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "burn ") {
				// Extract burn amount
				fields := strings.Fields(part)
				if len(fields) >= 2 {
					params["burn"] = fields[1]
				}
			} else if strings.HasPrefix(part, "action ") {
				// Extract power action type (e.g., "action ACT2")
				fields := strings.Fields(part)
				if len(fields) >= 2 {
					params["action_type"] = fields[1]
				}
			}
		}

		// If there's a power action specified, this is a power action with burn
		if _, hasActionType := params["action_type"]; hasActionType {
			return ActionPowerAction, params, nil
		}

		// Otherwise, it's just a burn (not sure if this ever happens alone)
		return ActionBurnPower, params, nil

	case actionStr == "setup":
		return ActionSetup, params, nil

	case actionStr == "pass":
		return ActionPass, params, nil

	case actionStr == "wait":
		return ActionWait, params, nil

	case strings.HasPrefix(actionStr, "Leech ") || strings.HasPrefix(actionStr, "Decline "):
		return ActionLeech, params, nil

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, "upgrade "):
		// Compound action: convert 1W to 1C. upgrade F3 to TE. +FAV9
		// The convert part is a state change (reflected in resource deltas)
		// We only need to execute the upgrade and favor tile actions
		parts := strings.Split(actionStr, ".")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "upgrade ") {
				// Found the upgrade part - parse it and any following favor tile
				// Look for favor tile in the same part or next part
				favorPart := ""
				if strings.Contains(part, "+FAV") {
					favorPart = part
				} else if i+1 < len(parts) && strings.Contains(parts[i+1], "+FAV") {
					favorPart = parts[i+1]
				}

				// Parse the upgrade
				fields := strings.Fields(part)
				if len(fields) >= 4 {
					coord := fields[1]
					buildingType := fields[3] // "TE", "TP", "SH", "SA"
					params["coord"] = coord
					params["building"] = buildingType

					// Parse favor tile if present
					if favorPart != "" {
						favorMatch := regexp.MustCompile(`\+FAV(\d+)`).FindStringSubmatch(favorPart)
						if len(favorMatch) > 1 {
							params["favor_tile"] = "FAV" + favorMatch[1] // No "+" prefix for ParseFavorTile
						}
					}

					// Mark to skip validation since resources are synced by validator first
					params["skip_validation"] = "true"

					return ActionUpgrade, params, nil
				}
			}
		}
		// If we couldn't parse the upgrade, treat as convert only
		return ActionConvert, params, nil

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, "pass "):
		// Compound action: convert 1PW to 1C. pass BON7
		// The convert part is a state change (reflected in resource deltas)
		// We only need to execute the pass action
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "pass ") || strings.HasPrefix(part, "Pass ") {
				bonusTile := strings.Fields(part)[1]
				params["bonus"] = bonusTile
				return ActionPass, params, nil
			}
		}
		// If we couldn't find the pass part, treat as convert only
		return ActionConvert, params, nil

	case strings.HasPrefix(actionStr, "convert "):
		return ActionConvert, params, nil

	case strings.Contains(actionStr, "_income_for_faction"):
		return ActionIncome, params, nil

	case strings.HasPrefix(actionStr, "+"):
		// Cult track advancement like "+FIRE", "+WATER"
		return ActionCultAdvance, params, nil

	default:
		// Unknown or comment
		params["raw"] = actionStr
		return ActionUnknown, params, nil
	}

	return ActionUnknown, params, fmt.Errorf("unrecognized action: %s", actionStr)
}

// ActionType represents different types of actions
type ActionType int

const (
	ActionUnknown ActionType = iota
	ActionSetup
	ActionBuild
	ActionUpgrade
	ActionTransformAndBuild
	ActionPass
	ActionAdvanceShipping
	ActionAdvanceDigging
	ActionSendPriest
	ActionPowerAction
	ActionBurnPower
	ActionLeech
	ActionConvert
	ActionIncome
	ActionCultAdvance
	ActionWait
)
