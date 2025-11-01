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
	var pendingDelta *int  // Store delta that comes before a value

	for idx < len(parts) {
		part := strings.TrimSpace(parts[idx])
		if part == "" {
			idx++
			continue
		}

		// Check if this part is a delta (signed number without unit)
		// Deltas come BEFORE values in format: [delta] value unit
		if (strings.HasPrefix(part, "+") || strings.HasPrefix(part, "-")) &&
		   !strings.Contains(part, "/") &&
		   len(part) > 1 {
			delta, err := strconv.Atoi(part)
			if err == nil {
				pendingDelta = &delta
				idx++
				continue
			}
		}

		// Check for VP
		if value, delta, ok := parseResourceField(part, "VP", pendingDelta); ok {
			entry.VP = value
			if ok {
				entry.VPDelta = delta
				pendingDelta = nil
			}
			idx++
			continue
		}

		// Check for Coins
		if strings.HasSuffix(part, "C") && !strings.Contains(part, "ACT") {
			if value, delta, ok := parseResourceField(part, "C", pendingDelta); ok {
				entry.Coins = value
				if ok {
					entry.CoinsDelta = delta
					pendingDelta = nil
				}
				idx++
				continue
			}
		}

		// Check for Workers
		if strings.HasSuffix(part, "W") && len(part) < 5 {
			if value, delta, ok := parseResourceField(part, "W", pendingDelta); ok {
				entry.Workers = value
				if ok {
					entry.WorkersDelta = delta
					pendingDelta = nil
				}
				idx++
				continue
			}
		}

		// Check for Priests
		if strings.HasSuffix(part, "P") && len(part) < 5 && !strings.HasSuffix(part, "VP") && !strings.HasSuffix(part, "PW") {
			if value, delta, ok := parseResourceField(part, "P", pendingDelta); ok {
				entry.Priests = value
				if ok {
					entry.PriestsDelta = delta
					pendingDelta = nil
				}
				idx++
				continue
			}
		}

		// Check for Power Bowls (e.g., "3/9/0 PW")
		if strings.HasSuffix(part, "PW") {
			pwStr := strings.TrimSpace(strings.TrimSuffix(part, "PW"))
			if values, ok := parseSlashSeparatedInts(pwStr, 3); ok {
				entry.PowerBowls.Bowl1 = values[0]
				entry.PowerBowls.Bowl2 = values[1]
				entry.PowerBowls.Bowl3 = values[2]
			}
			// Power delta comes before PW, but we don't store it separately
			// Just reset pendingDelta if it exists
			pendingDelta = nil
			idx++
			continue
		}

		// Check for Cult Tracks (e.g., "0/0/0/0" or "0/1/1/0")
		if !strings.Contains(part, "PW") {
			if values, ok := parseSlashSeparatedInts(part, 4); ok {
				entry.CultTracks.Fire = values[0]
				entry.CultTracks.Water = values[1]
				entry.CultTracks.Earth = values[2]
				entry.CultTracks.Air = values[3]
				// Cult track deltas come before the cult track values, but we don't use them
				// Just reset pendingDelta if it exists
				pendingDelta = nil
				idx++
				continue
			}
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

			// Check for favor tile or town tile selection
			if len(parts) > 1 {
				for _, part := range parts[1:] {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "+FAV") {
						params["favor_tile"] = strings.TrimPrefix(part, "+")
					} else if strings.HasPrefix(part, "+TW") {
						params["town_tile"] = strings.TrimPrefix(part, "+")
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
		parseCompoundActionParts(actionStr, params)
		return ActionTransformAndBuild, params, nil

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, ". action "):
		// Compound action: convert 1PW to 1C. action ACTW. build H4
		// The convert part is a state change (reflected in resource deltas)
		// We only need to execute the action part
		// NOTE: This must be checked BEFORE "convert + build" to avoid misclassification
		parseCompoundActionParts(actionStr, params)
		// Return as power action
		return ActionPowerAction, params, nil

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, "dig ") && strings.Contains(actionStr, "build"):
		// convert 1P to 1W. dig 2. build D5
		parseCompoundActionParts(actionStr, params)
		// Mark this as a compound action with convert
		params["has_convert"] = "true"
		return ActionTransformAndBuild, params, nil

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, "build"):
		// convert 1PW to 1C. build I8
		// convert 1PW to 1C. build I8. wait
		parseCompoundActionParts(actionStr, params)
		return ActionBuild, params, nil

	case strings.HasPrefix(actionStr, "dig ") && strings.Contains(actionStr, "build"):
		// dig 1. build G6
		parseCompoundActionParts(actionStr, params)
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
		// action BON2. +WATER (bonus card cult advance)
		parseCompoundActionParts(actionStr, params)

		// Check for additional special parts (cult track, town tile)
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "+") && !strings.HasPrefix(part, "+TW") && !strings.HasPrefix(part, "+FAV") {
				// Cult track advancement (e.g., "+WATER", "+FIRE")
				cultTrack := strings.TrimPrefix(part, "+")
				params["cult_track"] = strings.TrimSpace(cultTrack)
			} else if strings.HasPrefix(part, "+TW") {
				// Town tile (e.g., "+TW5")
				params["town_tile"] = strings.TrimSpace(part)
			}
		}

		return ActionPowerAction, params, nil

	case strings.HasPrefix(actionStr, "burn "):
		// Can be part of compound action: "burn 3. action ACT2" or "burn 6. action ACT6. transform..."
		// or "burn 4. action ACT5. build B3"
		// Parse all parts of the compound action
		parseCompoundActionParts(actionStr, params)

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

	case strings.HasPrefix(actionStr, "convert ") && strings.Contains(actionStr, "send p to"):
		// Compound action: convert 1PW to 1C. send p to EARTH. convert 1PW to 1C
		// Extract the send priest action
		parts := strings.Split(actionStr, ".")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "send p to ") {
				cult := strings.TrimPrefix(part, "send p to ")
				params["cult"] = strings.Fields(cult)[0]
				return ActionSendPriest, params, nil
			}
		}
		// If we couldn't find the send part, treat as convert only
		return ActionConvert, params, nil

	case strings.HasPrefix(actionStr, "convert "):
		return ActionConvert, params, nil

	case strings.Contains(actionStr, "_income_for_faction"):
		return ActionIncome, params, nil

	case strings.HasPrefix(actionStr, "+"):
		// Cult track advancement like "+FIRE", "+WATER"
		// Can also be compound: "+FIRE. pass BON10"
		if strings.Contains(actionStr, "pass ") {
			// Compound cult advance + pass: extract bonus card from pass action
			parts := strings.Split(actionStr, ".")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "pass ") || strings.HasPrefix(part, "Pass ") {
					bonusTile := strings.Fields(part)[1]
					params["bonus"] = bonusTile
					return ActionPass, params, nil
				}
			}
		}
		// Just cult advancement
		return ActionCultAdvance, params, nil

	default:
		// Unknown or comment
		params["raw"] = actionStr
		return ActionUnknown, params, nil
	}

	return ActionUnknown, params, fmt.Errorf("unrecognized action: %s", actionStr)
}

// parseCompoundActionParts extracts common parts from compound actions (burn, action, build, transform, dig)
func parseCompoundActionParts(actionStr string, params map[string]string) {
	parts := strings.Split(actionStr, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "burn ") {
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				params["burn"] = fields[1]
			}
		} else if strings.HasPrefix(part, "action ") {
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				params["action_type"] = fields[1]
			}
		} else if strings.HasPrefix(part, "build ") {
			coord := strings.TrimPrefix(part, "build ")
			params["coord"] = strings.Fields(coord)[0]
		} else if strings.HasPrefix(part, "transform ") {
			fields := strings.Fields(part)
			if len(fields) >= 4 && fields[2] == "to" {
				params["transform_coord"] = fields[1]
				params["transform_color"] = fields[3]
			}
		} else if strings.HasPrefix(part, "dig ") {
			spades := strings.TrimPrefix(part, "dig ")
			params["spades"] = strings.Fields(spades)[0]
		}
	}
}

// parseResourceField parses a resource field with suffix and optional delta
func parseResourceField(part string, suffix string, pendingDelta *int) (value int, delta int, hasDelta bool) {
	if strings.HasSuffix(part, suffix) {
		numStr := strings.TrimSpace(strings.TrimSuffix(part, suffix))
		parsedValue, err := strconv.Atoi(numStr)
		if err != nil {
			// Not a valid resource field, return false so it can be treated as action
			return 0, 0, false
		}
		value = parsedValue
		if pendingDelta != nil {
			delta = *pendingDelta
			hasDelta = true
		}
		return value, delta, true
	}
	return 0, 0, false
}

// parseSlashSeparatedInts parses a slash-separated string into integers
func parseSlashSeparatedInts(part string, expectedCount int) ([]int, bool) {
	if strings.Count(part, "/") != expectedCount-1 {
		return nil, false
	}
	parts := strings.Split(part, "/")
	if len(parts) != expectedCount {
		return nil, false
	}
	result := make([]int, expectedCount)
	for i, p := range parts {
		result[i], _ = strconv.Atoi(p)
	}
	return result, true
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
	ActionPowerAction // Any action starting with "action" keyword (power/bonus/special)
	ActionBurnPower
	ActionLeech
	ActionConvert
	ActionIncome
	ActionCultAdvance
	ActionWait
)
