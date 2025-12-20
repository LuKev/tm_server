package notation

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/replay"
)

// BGAParser parses a BGA game log into a sequence of GameActions
// BGAParser parses a BGA game log into a sequence of LogItems
type BGAParser struct {
	lines       []string
	currentLine int
	items       []LogItem
	// State tracking
	currentRound int
	players      map[string]string // Name -> Faction
	passOrder    []string          // Tracks who passed in current round to determine next round order
}

func NewBGAParser(content string) *BGAParser {
	// Split content into lines
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return &BGAParser{
		lines:     lines,
		items:     make([]LogItem, 0),
		players:   make(map[string]string),
		passOrder: make([]string, 0),
	}
}

func (p *BGAParser) Parse() ([]LogItem, error) {
	// Regex patterns
	reMove := regexp.MustCompile(`^Move (\d+) :`)
	reFactionSelection := regexp.MustCompile(`(.*) is playing the (.*) Faction`)
	reGameBoard := regexp.MustCompile(`Game board: (.*)`)
	reMiniExpansions := regexp.MustCompile(`Mini-expansions: (.*)`)

	// Action Start Patterns (Prefixes)
	reBuildDwellingSetup := regexp.MustCompile(`(.*) places a Dwelling \[(.*)\]`) // Setup is single line
	reBuildDwellingGameStart := regexp.MustCompile(`(.*) builds a Dwelling for`)
	reUpgradeStart := regexp.MustCompile(`(.*) upgrades a (.*) to a (.*) for`)
	reTransformStart := regexp.MustCompile(`(.*) transforms a Terrain space`)

	// Multi-line Action Starts
	rePowerActionStart := regexp.MustCompile(`(.*) spends$`)
	reLeechGetsStart := regexp.MustCompile(`(.*) gets$`)
	reLeechPaysStart := regexp.MustCompile(`(.*) pays$`)
	reBurnStart := regexp.MustCompile(`(.*) sacrificed$`)
	reFavorTileStart := regexp.MustCompile(`(.*) takes a Favor tile`)
	reWitchesRide := regexp.MustCompile(`(.*) builds a Dwelling for free \(Witches Ride\) \[(.*)\]`)
	reExchangeTrack := regexp.MustCompile(`(.*) advances on the Exchange Track`)

	rePass := regexp.MustCompile(`(.*) passes`)
	rePriest := regexp.MustCompile(`(.*) sends a Priest to the Order of the Cult of (.*)\. Forever!`)
	reShipping := regexp.MustCompile(`(.*) advances on the Shipping track`)
	reDeclineLeech := regexp.MustCompile(`(.*) declines getting Power via Structures \[(.*)\]`)

	// Round detection
	reActionPhase := regexp.MustCompile(`~ Action phase ~`)
	reFinalScoring := regexp.MustCompile(`~ Final scoring ~`)

	settings := make(map[string]string)
	auctionOver := false

	// Parse header and setup
	for p.currentLine < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.currentLine])
		p.currentLine++

		if matches := reGameBoard.FindStringSubmatch(line); len(matches) > 1 {
			settings["Game"] = matches[1]
		}
		if matches := reMiniExpansions.FindStringSubmatch(line); len(matches) > 1 {
			settings["MiniExpansions"] = matches[1]
		}

		if strings.Contains(line, "The Factions auction is over") {
			auctionOver = true
			fmt.Println("Found auction over line")
			break
		}
	}

	if !auctionOver {
		fmt.Println("Warning: Did not find auction over line")
	}

	// Add settings item
	p.items = append(p.items, GameSettingsItem{Settings: settings})

	// Parse player factions from setup summary
	var setupOrder []string
	for p.currentLine < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.currentLine])
		p.currentLine++

		if strings.Contains(line, "Every player has chosen a Faction") {
			fmt.Println("Found faction setup over line")
			break
		}
		if matches := reFactionSelection.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			factionName := strings.TrimSpace(matches[2])
			p.players[playerName] = factionName
			setupOrder = append(setupOrder, factionName)
			fmt.Printf("Found player %s playing %s\n", playerName, factionName)
		}
	}

	// Main parsing loop
	var currentMove int
	_ = currentMove

	fmt.Println("Starting main parsing loop...")
	for p.currentLine < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.currentLine])
		p.currentLine++

		if line == "" {
			continue
		}

		// Stop at Final Scoring
		if reFinalScoring.MatchString(line) {
			fmt.Println("Found Final Scoring, stopping parse.")
			break
		}

		// Check for Move header
		if matches := reMove.FindStringSubmatch(line); len(matches) > 1 {
			moveNum, _ := strconv.Atoi(matches[1])
			currentMove = moveNum
			fmt.Printf("Processing Move %d\n", currentMove)
			continue
		}

		// Check for Round Start
		if reActionPhase.MatchString(line) {
			p.currentRound++
			fmt.Printf("Found Round %d Start\n", p.currentRound)

			// Determine turn order
			var turnOrder []string
			if p.currentRound == 1 {
				turnOrder = setupOrder
			} else {
				// Use pass order from previous round directly
				// In TM/AOI, the turn order for the next round is exactly the pass order
				if len(p.passOrder) > 0 {
					// We might need to ensure all players are in passOrder?
					// If log is complete, they should be.
					// If log is partial, we use what we have.
					// But we should append any players who haven't passed yet?
					// Usually round doesn't end until everyone passes.
					turnOrder = make([]string, len(p.passOrder))
					copy(turnOrder, p.passOrder)
				} else {
					// Fallback if no pass order found (shouldn't happen)
					turnOrder = setupOrder
				}
				// Reset pass order for new round
				p.passOrder = make([]string, 0)
			}

			p.items = append(p.items, RoundStartItem{
				Round:     p.currentRound,
				TurnOrder: turnOrder,
			})
		}

		// Skip timestamps
		if strings.Contains(line, " AM") || strings.Contains(line, " PM") {
			continue
		}

		// Try to match actions
		if matches := reBuildDwellingSetup.FindStringSubmatch(line); len(matches) > 2 {
			fmt.Printf("Matched BuildDwelling (Setup): %s\n", line)
			p.handleBuildDwelling(matches[1], matches[2], true)

		} else if matches := reBuildDwellingGameStart.FindStringSubmatch(line); len(matches) > 1 {
			fmt.Printf("Matched BuildDwelling Start (Game): %s\n", line)
			coordStr := p.consumeUntilCoord()
			if coordStr != "" {
				p.handleBuildDwelling(matches[1], coordStr, false)
			}

		} else if matches := reUpgradeStart.FindStringSubmatch(line); len(matches) > 3 {
			fmt.Printf("Matched Upgrade Start: %s\n", line)
			coordStr := p.consumeUntilCoord()
			if coordStr != "" {
				p.handleUpgrade(matches[1], matches[2], matches[3], coordStr)
			}

		} else if matches := reTransformStart.FindStringSubmatch(line); len(matches) > 1 {
			fmt.Printf("Matched Transform Start: %s\n", line)
			coordStr := p.consumeUntilCoord()
			if coordStr != "" {
				p.handleTransform(matches[1], coordStr)
			}

		} else if matches := rePowerActionStart.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Power Action:
			// Player spends
			// Cost
			// to collect/get
			// Reward
			// (Power action)
			playerName := matches[1]
			cost, _ := p.consumeInt()
			_ = p.consumeLine() // "to collect" or "to get"
			reward, _ := p.consumeInt()
			_ = p.consumeLine() // "(Power action)"

			fmt.Printf("Matched PowerAction: %s spends %d to get %d\n", playerName, cost, reward)
			p.handlePowerAction(playerName, cost, reward)

		} else if matches := reBurnStart.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Burn:
			// Player sacrificed
			// Amount
			// in Bowl 2 to get
			// Amount
			// from Bowl 2 to Bowl 3
			playerName := matches[1]
			amount, _ := p.consumeInt()
			// Consume rest of lines
			_ = p.consumeLine()   // "in Bowl 2 to get"
			_, _ = p.consumeInt() // Amount again
			_ = p.consumeLine()   // "from Bowl 2 to Bowl 3"

			fmt.Printf("Matched Burn: %s sacrificed %d\n", playerName, amount)
			p.handleBurn(playerName, amount)

		} else if matches := reFavorTileStart.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Favor Tile:
			// Player takes a Favor tile
			// Player gains X on the Cult of Y track (Favor tile)
			playerName := matches[1]
			fmt.Printf("Matched Favor Tile Start: %s\n", playerName)
			p.handleFavorTile(playerName)

		} else if matches := reWitchesRide.FindStringSubmatch(line); len(matches) > 2 {
			// Single-line Witches Ride:
			// Player builds a Dwelling for free (Witches Ride) [Coord]
			playerName := matches[1]
			coordStr := matches[2]
			fmt.Printf("Matched Witches Ride: %s at %s\n", playerName, coordStr)
			p.handleWitchesRide(playerName, coordStr)

		} else if matches := reLeechPaysStart.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Leech (Pays):
			// Player pays
			// Cost
			// and gets
			// Amount
			// via Structures [Coord]
			playerName := matches[1]
			cost, _ := p.consumeInt()
			_ = p.consumeLine() // "and gets"
			amount, _ := p.consumeInt()
			// Consume until via Structures
			line := p.consumeLine()
			if strings.Contains(line, "via Structures") {
				fmt.Printf("Matched Leech Pays: %s pays %d gets %d\n", playerName, cost, amount)
				p.handleLeech(playerName, "", amount, cost, true)
			}

		} else if matches := reLeechGetsStart.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Leech (Gets):
			// Player gets
			// Amount
			// via Structures [Coord]
			playerName := matches[1]
			amount, _ := p.consumeInt()
			line := p.consumeLine()
			if strings.Contains(line, "via Structures") {
				fmt.Printf("Matched Leech Gets: %s gets %d\n", playerName, amount)
				p.handleLeech(playerName, "", amount, 0, true)
			}

		} else if matches := rePass.FindStringSubmatch(line); len(matches) > 1 {
			fmt.Printf("Matched Pass: %s\n", line)
			p.handlePass(matches[1])

		} else if matches := rePriest.FindStringSubmatch(line); len(matches) > 2 {
			fmt.Printf("Matched Priest: %s\n", line)
			p.handleSendPriest(matches[1], matches[2])

		} else if matches := reShipping.FindStringSubmatch(line); len(matches) > 1 {
			fmt.Printf("Matched Shipping: %s\n", line)
			p.handleAdvanceShipping(matches[1])

		} else if matches := reExchangeTrack.FindStringSubmatch(line); len(matches) > 1 {
			// Multi-line Digging:
			// Player advances on the Exchange Track for
			// Cost
			// 5
			// 1
			// and earns
			// 6
			playerName := matches[1]
			fmt.Printf("Matched Exchange Track (Digging): %s\n", playerName)
			p.handleAdvanceDigging(playerName)

		} else if matches := reDeclineLeech.FindStringSubmatch(line); len(matches) > 2 {
			fmt.Printf("Matched Decline Leech: %s\n", line)
			p.handleLeech(matches[1], matches[2], 0, 0, false)
		}
	}

	return p.items, nil
}

// getClockwiseOrder returns the faction order starting from startFaction, following seatOrder
func (p *BGAParser) getClockwiseOrder(startFaction string, seatOrder []string) []string {
	startIndex := -1
	for i, f := range seatOrder {
		if f == startFaction {
			startIndex = i
			break
		}
	}
	if startIndex == -1 {
		return seatOrder // Should not happen
	}

	result := make([]string, 0, len(seatOrder))
	// Add from startIndex to end
	result = append(result, seatOrder[startIndex:]...)
	// Add from start to startIndex
	result = append(result, seatOrder[:startIndex]...)
	return result
}

func (p *BGAParser) handleBuildDwelling(playerName, coordStr string, isSetup bool) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)

	var action game.Action
	if isSetup {
		action = game.NewSetupDwellingAction(playerID, hex)
	} else {
		action = game.NewTransformAndBuildAction(playerID, hex, true)
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleUpgrade(playerName, from, to, coordStr string) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)
	newType := parseBuildingType(to)

	action := game.NewUpgradeBuildingAction(playerID, hex, newType)
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleTransform(playerName, coordStr string) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)
	action := game.NewTransformAndBuildAction(playerID, hex, false)
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handlePowerAction(playerName string, cost, reward int) {
	playerID := p.getPlayerID(playerName)

	// Map to Action Code
	code := "ACT?"
	if cost == 3 {
		if reward == 1 {
			code = "ACT2" // Priest (3pw -> 1 Priest)
		} else {
			code = "ACT1" // Bridge (3pw -> Bridge) - unlikely to match "collect N"
		}
	} else if cost == 4 {
		if reward == 2 {
			code = "ACT3" // Workers (4pw -> 2 Workers)
		} else if reward == 7 {
			code = "ACT4" // Coins (4pw -> 7 Coins)
		} else if reward == 1 {
			code = "ACT5" // Spade (4pw -> 1 Spade)
		}
	} else if cost == 6 {
		if reward == 2 {
			code = "ACT6" // 2 Spades (6pw -> 2 Spades)
		}
	}

	action := &LogPowerAction{
		PlayerID:   playerID,
		ActionCode: code,
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handlePass(playerName string) {
	playerID := p.getPlayerID(playerName) // This is the Faction Name
	p.passOrder = append(p.passOrder, playerID)

	// Bonus card is usually in next line, but PassAction requires it or nil
	var bonusCard *game.BonusCardType
	action := game.NewPassAction(playerID, bonusCard)
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleSendPriest(playerName, cultColor string) {
	playerID := p.getPlayerID(playerName)
	track := parseCultTrack(cultColor)

	action := &game.SendPriestToCultAction{
		BaseAction: game.BaseAction{
			Type:     game.ActionSendPriestToCult,
			PlayerID: playerID,
		},
		Track:         track,
		SpacesToClimb: 3, // Placeholder
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleAdvanceShipping(playerName string) {
	playerID := p.getPlayerID(playerName)
	action := game.NewAdvanceShippingAction(playerID)
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleLeech(playerName, coordStr string, amount int, cost int, accepted bool) {
	playerID := p.getPlayerID(playerName)
	_ = coordStr

	if accepted {
		action := &LogAcceptLeechAction{
			PlayerID:    playerID,
			PowerAmount: amount,
			VPCost:      cost,
		}
		p.items = append(p.items, ActionItem{Action: action})
	} else {
		// For decline, we use the standard action (it's simpler) or we could make a LogDecline
		// But standard DeclinePowerLeechAction doesn't have extra fields we need for log?
		// Actually standard DeclinePowerLeechAction just needs PlayerID.
		// But wait, NewDeclinePowerLeechAction takes (playerID, amount).
		// We have amount.
		action := game.NewDeclinePowerLeechAction(playerID, amount)
		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleBurn(playerName string, amount int) {
	playerID := p.getPlayerID(playerName)
	action := &LogBurnAction{
		PlayerID: playerID,
		Amount:   amount,
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleFavorTile(playerName string) {
	playerID := p.getPlayerID(playerName)
	if playerID == "" {
		return
	}

	// Parse the Favor Tile details
	// Example:
	// haligh takes a Favor tile
	// haligh gains
	// 1
	// on the Cult of Fire track (Favor tile)

	var tileCode string
	reFavorTileGain := regexp.MustCompile(`on the Cult of (.*) track \(Favor tile\)`)

	// Consume up to 5 lines to find the track info
	var amount int
	var track string

	for i := 0; i < 5; i++ {
		line := p.consumeLine()

		// Check for amount (just a number)
		if val, err := strconv.Atoi(line); err == nil {
			amount = val
			continue
		}

		// Check for track info
		if matches := reFavorTileGain.FindStringSubmatch(line); len(matches) > 1 {
			track = matches[1]
			break
		}

		// Stop if we hit a new move or timestamp
		if strings.HasPrefix(line, "Move ") || strings.Contains(line, " AM") || strings.Contains(line, " PM") {
			break
		}
	}

	if track != "" && amount > 0 {
		trackCode := ""
		switch track {
		case "Fire":
			trackCode = "F"
		case "Water":
			trackCode = "W"
		case "Earth":
			trackCode = "E"
		case "Air":
			trackCode = "A"
		}

		if trackCode != "" {
			tileCode = fmt.Sprintf("FAV-%s%d", trackCode, amount)
		}
	}

	if tileCode == "" {
		tileCode = "FAV-UNKNOWN"
	}

	action := &LogFavorTileAction{
		PlayerID: playerID,
		Tile:     tileCode,
	}
	newItem := ActionItem{Action: action}

	// Hoist the action: Find the last "Structure" action by this player and insert after it
	// Structure actions: Upgrade, Build (Setup or Game)
	// We search backwards from the end
	inserted := false
	for i := len(p.items) - 1; i >= 0; i-- {
		item := p.items[i]
		if actItem, ok := item.(ActionItem); ok {
			if actItem.Action.GetPlayerID() == playerID {
				// Check if it's a relevant action
				switch actItem.Action.(type) {
				case *game.UpgradeBuildingAction, *game.TransformAndBuildAction, *game.SetupDwellingAction:
					// Found it! Insert after this index
					// Insert at i+1
					p.items = append(p.items[:i+1], append([]LogItem{newItem}, p.items[i+1:]...)...)
					inserted = true
					goto Done
				}
			}
		}
	}

Done:
	if !inserted {
		// Fallback: append to end
		p.items = append(p.items, newItem)
	}
}


func (p *BGAParser) handleWitchesRide(playerName string, coordStr string) {
	playerID := p.getPlayerID(playerName)
	if playerID == "" {
		return
	}

	actionCode := fmt.Sprintf("ACT-SH-D-%s", coordStr)

	p.items = append(p.items, ActionItem{
		Action: &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: actionCode,
		},
	})
}

func (p *BGAParser) handleAdvanceDigging(playerName string) {
	playerID := p.getPlayerID(playerName)
	if playerID == "" {
		return
	}

	// Digging action is ActionAdvanceDigging
	// But we need to consume the cost lines?
	// The log says:
	// felipebart advances on the Exchange Track for
	// 2
	// 5
	// 1
	// and earns
	// 6

	// We should consume these lines to keep the parser sync.
	// We can consume until we hit "and earns" + 1 line?
	// Or just consume until we hit a non-number?
	// The cost lines are just numbers.

	// Consume cost lines
	for {
		line := p.peekLine()
		if line == "" {
			break
		}
		// If it's a number, consume it
		if _, err := strconv.Atoi(line); err == nil {
			p.consumeLine()
			continue
		}
		// If it starts with "and earns", consume it and the next line (reward)
		if strings.HasPrefix(line, "and earns") {
			p.consumeLine() // "and earns"
			p.consumeLine() // Reward amount
			break
		}
		// Otherwise stop
		break
	}

	// Create action
	// We don't have a specific LogAction for Digging, we can use the game action directly?
	// ActionAdvanceDigging exists in game package.
	// But does it have fields we need?
	// type AdvanceDiggingAction struct { BaseAction }
	// Yes.

	action := game.NewAdvanceDiggingAction(playerID)
	p.items = append(p.items, ActionItem{Action: action})
}

// Helper functions

func (p *BGAParser) getPlayerID(name string) string {
	if faction, ok := p.players[name]; ok {
		return faction
	}
	return name
}

func parseCoord(s string) board.Hex {
	// Remove brackets if present
	s = strings.Trim(s, "[]")
	h, err := replay.ConvertLogCoordToAxial(s)
	if err != nil {
		fmt.Printf("Error parsing coord %s: %v\n", s, err)
		return board.Hex{}
	}
	return h
}

func parseBuildingType(s string) models.BuildingType {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "dwelling"):
		return models.BuildingDwelling
	case strings.Contains(s, "trading house"):
		return models.BuildingTradingHouse
	case strings.Contains(s, "temple"):
		return models.BuildingTemple
	case strings.Contains(s, "sanctuary"):
		return models.BuildingSanctuary
	case strings.Contains(s, "stronghold"):
		return models.BuildingStronghold
	}
	return models.BuildingDwelling
}

func parseCultTrack(s string) game.CultTrack {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "fire"):
		return game.CultFire
	case strings.Contains(s, "water"):
		return game.CultWater
	case strings.Contains(s, "earth"):
		return game.CultEarth
	case strings.Contains(s, "air"):
		return game.CultAir
	}
	return game.CultFire // Default
}

func (p *BGAParser) consumeLine() string {
	if p.currentLine >= len(p.lines) {
		return ""
	}
	line := strings.TrimSpace(p.lines[p.currentLine])
	p.currentLine++
	return line
}

func (p *BGAParser) consumeInt() (int, error) {
	line := p.consumeLine()
	return strconv.Atoi(line)
}

func (p *BGAParser) consumeUntilCoord() string {
	// Consume lines until we find one that looks like a coord [A1]
	// Limit to avoid infinite loop
	for i := 0; i < 5; i++ {
		line := p.consumeLine()
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			// Extract content between brackets
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 && end > start {
				return line[start+1 : end]
			}
		}
	}
	return ""
}

func (p *BGAParser) peekLine() string {
	if p.currentLine >= len(p.lines) {
		return ""
	}
	return strings.TrimSpace(p.lines[p.currentLine])
}
