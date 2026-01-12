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
)

// BGAParser parses a BGA game log into a sequence of GameActions
// BGAParser parses a BGA game log into a sequence of LogItems
type BGAParser struct {
	lines       []string
	currentLine int
	items       []LogItem
	// State tracking
	currentRound  int
	players       map[string]string // Name -> Faction
	passOrder     []string          // Tracks who passed in current round to determine next round order
	townPending   map[string]bool   // Name -> bool, tracks if player just founded a town
	consumedLines map[int]bool      // Line indices that have been consumed by another action
}

func NewBGAParser(content string) *BGAParser {
	// Split content into lines
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return &BGAParser{
		lines:         lines,
		currentLine:   0,
		items:         make([]LogItem, 0),
		players:       make(map[string]string),
		passOrder:     make([]string, 0),
		townPending:   make(map[string]bool),
		consumedLines: make(map[int]bool),
	}
}

func (p *BGAParser) Parse() ([]LogItem, error) {
	// Regex patterns
	reMove := regexp.MustCompile(`^Move (\d+) :`)
	reFactionSelection := regexp.MustCompile(`(.*) is playing the (.*) Faction(?: \(with (\d+) VP Starting VPs\))?`)
	reFactionSelection2 := regexp.MustCompile(`(.*) selected the faction (.*) on`)
	reGameBoard := regexp.MustCompile(`Game board: (.*)`)
	reMiniExpansions := regexp.MustCompile(`Mini-expansions: (.*)`)

	// Setup patterns
	reScoringTile := regexp.MustCompile(`Round (\d+) scoring: (.*)`)
	reRemovedBonus := regexp.MustCompile(`Removing tile (.*)`)

	// Action Start Patterns (Prefixes)
	reBuildDwellingSetup := regexp.MustCompile(`(.*) places a Dwelling \[(.*)\]`) // Setup is single line
	reBuildDwellingGameStart := regexp.MustCompile(`(.*) builds a Dwelling for`)
	reUpgradeStart := regexp.MustCompile(`(.*) upgrades a (.*) to a (.*) for`)
	reTransformStart := regexp.MustCompile(`(.*) transforms a Terrain space(?:.* → (.*))? for`)

	// Single-line Action Starts (Updated)
	rePowerAction := regexp.MustCompile(`(.*) spends (\d+) power to (?:get|collect) (.*) \(Power action\)`)
	reLeechGets := regexp.MustCompile(`(.*) gets (\d+) power via Structures \[(.*)\]`)
	reLeechPays := regexp.MustCompile(`(.*) pays (\d+) VP and gets (\d+) power via Structures \[(.*)\]`)
	reBurn := regexp.MustCompile(`(.*) sacrificed (\d+) power in Bowl 2 to get (\d+) power from Bowl 2 to Bowl 3`)
	reFavorTileStart := regexp.MustCompile(`(.*) takes a Favor tile`)
	reWitchesRide := regexp.MustCompile(`(.*) builds a Dwelling for free \(Witches Ride\) \[(.*)\]`)
	reExchangeTrack := regexp.MustCompile(`(.*) advances on the Exchange Track for (.*) and earns (\d+) VP`)
	reTownFound := regexp.MustCompile(`(.*) founds a Town \[(.*)\]`)
	reMermaidsRiverTown := regexp.MustCompile(`(.*) founds a Town on a River space \(Mermaids Ability\) \[(R~.*)\]`)
	reTownVP := regexp.MustCompile(`(.*) gets (\d+) VP (?:Victory points )?.*`)

	// Faction-specific stronghold actions
	reGiantsStronghold := regexp.MustCompile(`(.*) transforms a Terrain space.* \(Giants Stronghold\) \[(.*)\]`)
	reSwarmlingStronghold := regexp.MustCompile(`(.*) upgrades a Dwelling to a Trading house for free \(Swarmlings Stronghold\) \[(.*)\]`)
	reNomadsStronghold := regexp.MustCompile(`(.*) transforms a Terrain space.* for free \(Nomads Stronghold\).*\[(.*)\]`)
	reCMDoubleTurn := regexp.MustCompile(`(.*) takes a double-turn \(Chaos Magicians Stronghold\)`)
	reHalflingsSpades := regexp.MustCompile(`(.*) gets 3 Spades to Transform and Build \(Halflings Stronghold\)`)

	rePass := regexp.MustCompile(`(.*) passes`)
	rePriest := regexp.MustCompile(`(.*) sends a Priest to the Order of the Cult of (.*)\. Forever!`)
	reShipping := regexp.MustCompile(`(.*) advances on the Shipping track`)
	reDeclineLeech := regexp.MustCompile(`(.*) declines getting Power via Structures \[(.*)\]`)

	// Round detection
	reActionPhase := regexp.MustCompile(`~ Action phase ~`)
	reFinalScoring := regexp.MustCompile(`~ Final scoring ~`)

	settings := make(map[string]string)
	auctionOver := false
	scoringTiles := make(map[int]string)
	removedBonuses := make([]string, 0)

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
		if matches := reScoringTile.FindStringSubmatch(line); len(matches) > 2 {
			round, _ := strconv.Atoi(matches[1])
			// Extract just the score code (e.g. "SCORE2, TOWN >> 5" -> "SCORE2")
			parts := strings.Split(matches[2], ",")
			if len(parts) > 0 {
				scoringTiles[round] = strings.TrimSpace(parts[0])
			}
		}
		if matches := reRemovedBonus.FindStringSubmatch(line); len(matches) > 1 {
			removedBonuses = append(removedBonuses, strings.TrimSpace(matches[1]))
		}

		if strings.Contains(line, "The Factions auction is over") {
			auctionOver = true
			fmt.Println("Found auction over line")
			break
		}
		// Fallback: if we see "Every player has chosen a Faction", stop here (auction might be skipped or log different)
		if strings.Contains(line, "Every player has chosen a Faction") {
			fmt.Println("Found faction setup over line (in header scan)")
			p.currentLine--
			break
		}
		if reFactionSelection.MatchString(line) {
			p.currentLine--
			break
		}
		if reFactionSelection2.MatchString(line) {
			p.currentLine--
			break
		}
	}

	if !auctionOver {
		fmt.Println("Warning: Did not find auction over line")
	}

	// Parse player factions from setup summary
	var setupOrder []string
	for p.currentLine < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.currentLine])
		p.currentLine++

		if strings.Contains(line, "Every player has chosen a Faction") {
			break
		}
		if matches := reFactionSelection.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			factionName := strings.TrimSpace(matches[2])
			p.players[playerName] = factionName
			if !contains(setupOrder, factionName) {
				setupOrder = append(setupOrder, factionName)
			}
			// Check for starting VPs (group 3)
			if len(matches) > 3 && matches[3] != "" {
				settings["StartingVP:"+factionName] = matches[3]
			}
		} else if matches := reFactionSelection2.FindStringSubmatch(line); len(matches) > 2 {
			playerName := strings.TrimSpace(matches[1])
			factionName := strings.TrimSpace(matches[2])
			p.players[playerName] = factionName
			if !contains(setupOrder, factionName) {
				setupOrder = append(setupOrder, factionName)
			}
		}
	}

	// Add settings item with players
	for player, faction := range p.players {
		settings["Player:"+player] = faction
	}

	// Add ScoringTiles setting
	if len(scoringTiles) > 0 {
		var tiles []string
		for i := 1; i <= 6; i++ {
			if t, ok := scoringTiles[i]; ok {
				tiles = append(tiles, t)
			}
		}
		settings["ScoringTiles"] = strings.Join(tiles, ",")
	}

	// Add BonusCards setting ONLY if we detected removed bonuses
	// If no removed bonuses were detected, leave BonusCards empty so the user is prompted to select
	// This is because BGA logs don't reliably include which bonus cards were removed
	if len(removedBonuses) > 0 {
		allBonusCodes := []string{
			"BON-SPD", "BON-4C", "BON-6C", "BON-SHIP", "BON-WP",
			"BON-BB", "BON-TP", "BON-P", "BON-DW", "BON-SHIP-VP",
		}
		// Map BGA codes (BON1..BON10) to internal codes
		bgaToInternal := map[string]string{
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
		for _, rb := range removedBonuses {
			if internal, ok := bgaToInternal[rb]; ok {
				removedSet[internal] = true
			}
		}

		var availableBonuses []string
		for _, code := range allBonusCodes {
			if !removedSet[code] {
				availableBonuses = append(availableBonuses, code)
			}
		}
		settings["BonusCards"] = strings.Join(availableBonuses, ",")
	}
	// If removedBonuses is empty, BonusCards stays empty -> triggers user prompt

	p.items = append(p.items, GameSettingsItem{Settings: settings})

	// Main parsing loop
	var currentMove int
	_ = currentMove

	// fmt.Println("Starting main parsing loop...")
	for p.currentLine < len(p.lines) {
		lineIndex := p.currentLine
		line := p.lines[p.currentLine]
		// // fmt.Printf("Line %d: %s\n", p.currentLine, line)
		p.currentLine++

		// Skip lines that were already consumed by another action (e.g., dwelling build merged with bonus card spade)
		if p.consumedLines[lineIndex] {
			continue
		}

		if line == "" {
			continue
		}

		// Stop at Final Scoring
		if reFinalScoring.MatchString(line) {
			fmt.Println("Found Final Scoring, stopping parse.")
			break
		}

		// Check for Mermaids River Town (must be before regular town check)
		if matches := reMermaidsRiverTown.FindStringSubmatch(line); len(matches) > 2 {
			playerID := p.getPlayerID(matches[1])
			riverCoord := matches[2] // e.g., "R~D5"
			p.handleMermaidsRiverTown(playerID, riverCoord)
			continue
		}

		// Check for Town Found (regular)
		if matches := reTownFound.FindStringSubmatch(line); len(matches) > 2 {
			p.handleTownFound(matches[1])
			continue
		}

		// Check for Town VP (to merge)
		if matches := reTownVP.FindStringSubmatch(line); len(matches) > 2 {
			p.handleTownVP(matches[1], matches[2])
			continue
		}

		// Check for Move header
		if matches := reMove.FindStringSubmatch(line); len(matches) > 1 {
			moveNum, _ := strconv.Atoi(matches[1])
			currentMove = moveNum
			// // fmt.Printf("Processing Move %d\n", currentMove)
			continue
		}

		// Check for Round Start
		if reActionPhase.MatchString(line) {
			p.currentRound++
			// // fmt.Printf("Found Round %d Start\n", p.currentRound)

			// Determine turn order
			var turnOrder []string
			if p.currentRound == 1 {
				turnOrder = setupOrder
			} else {
				// Use pass order from previous round directly
				if len(p.passOrder) > 0 {
					turnOrder = make([]string, len(p.passOrder))
					copy(turnOrder, p.passOrder)
				} else {
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
		// Check specific actions first to avoid being consumed by general ones

		if matches := reConversion.FindStringSubmatch(line); len(matches) > 3 {
			playerID := p.getPlayerID(matches[1])
			p.handleConversion(playerID, matches[2], matches[3])

		} else if matches := reAlchemistsVP.FindStringSubmatch(line); len(matches) > 3 {
			playerID := p.getPlayerID(matches[1])
			p.handleAlchemistsVP(playerID, matches[2], matches[3])

		} else if matches := reBonusCardCult.FindStringSubmatch(line); len(matches) > 2 {
			// Bonus card cult advance: "gains 1 on the Cult of Y track (Bonus card action)"
			playerID := p.getPlayerID(matches[1])
			track := matches[2]
			p.handleBonusCardCult(playerID, track)

		} else if matches := reBonusCardSpade.FindStringSubmatch(line); len(matches) > 1 {
			playerID := p.getPlayerID(matches[1])
			p.handleBonusCardSpade(playerID, line)

		} else if matches := reReclaimPriest.FindStringSubmatch(line); len(matches) > 1 {
			playerID := p.getPlayerID(matches[1])
			p.handleReclaimPriest(playerID, line)

		} else if matches := reAurenStronghold.FindStringSubmatch(line); len(matches) > 1 {
			playerID := p.getPlayerID(matches[1])
			p.handleAurenStronghold(playerID, line)

		} else if matches := reFavorTileAction.FindStringSubmatch(line); len(matches) > 1 {
			playerID := p.getPlayerID(matches[1])
			p.handleFavorTileAction(playerID, line)

		} else if matches := reBridgePower.FindStringSubmatch(line); len(matches) > 1 {
			playerID := p.getPlayerID(matches[1])
			p.handleBridgePower(playerID, line)

		} else if matches := reBuildDwellingSetup.FindStringSubmatch(line); len(matches) > 2 {
			// fmt.Printf("Matched BuildDwelling (Setup): %s\n", line)
			p.handleBuildDwelling(matches[1], matches[2], true)

		} else if matches := reBuildDwellingGameStart.FindStringSubmatch(line); len(matches) > 1 {
			// fmt.Printf("Matched BuildDwelling Start (Game): %s\n", line)
			coordStr := p.extractCoord(line)
			if coordStr == "" {
				coordStr = p.consumeUntilCoord()
			}
			if coordStr != "" {
				p.handleBuildDwelling(matches[1], coordStr, false)
			}

		} else if matches := reGiantsStronghold.FindStringSubmatch(line); len(matches) > 2 {
			// Giants Stronghold: transforms terrain for free
			playerID := p.getPlayerID(matches[1])
			coordStr := matches[2]
			p.handleGiantsStronghold(playerID, coordStr)

		} else if matches := reSwarmlingStronghold.FindStringSubmatch(line); len(matches) > 2 {
			// Swarmlings Stronghold: free Trading House upgrade
			playerID := p.getPlayerID(matches[1])
			coordStr := matches[2]
			p.handleSwarmlingStronghold(playerID, coordStr)

		} else if matches := reNomadsStronghold.FindStringSubmatch(line); len(matches) > 2 {
			// Nomads Stronghold (Sandstorm): transform any hex to desert for free
			playerID := p.getPlayerID(matches[1])
			coordStr := matches[2]
			p.handleNomadsStronghold(playerID, coordStr)

		} else if matches := reCMDoubleTurn.FindStringSubmatch(line); len(matches) > 1 {
			// Chaos Magicians Stronghold: double-turn
			playerID := p.getPlayerID(matches[1])
			p.handleCMDoubleTurn(playerID)

		} else if matches := reHalflingsSpades.FindStringSubmatch(line); len(matches) > 1 {
			// Halflings Stronghold: 3 spades for transform
			playerID := p.getPlayerID(matches[1])
			p.handleHalflingsStrongholdSpades(playerID)

		} else if matches := reUpgradeStart.FindStringSubmatch(line); len(matches) > 3 {
			// fmt.Printf("Matched Upgrade Start: %s\n", line)
			coordStr := p.extractCoord(line)
			if coordStr == "" {
				coordStr = p.consumeUntilCoord()
			}
			if coordStr != "" {
				p.handleUpgrade(matches[1], matches[2], matches[3], coordStr)
			}

		} else if matches := reTransformStart.FindStringSubmatch(line); len(matches) > 1 {
			// fmt.Printf("Matched Transform Start: %s\n", line)
			coordStr := p.extractCoord(line)
			if coordStr == "" {
				coordStr = p.consumeUntilCoord()
			}
			targetTerrain := ""
			if len(matches) > 2 {
				targetTerrain = strings.TrimSpace(matches[2])
			}
			if coordStr != "" {
				p.handleTransform(matches[1], coordStr, targetTerrain)
			}

		} else if matches := rePowerAction.FindStringSubmatch(line); len(matches) > 3 {
			// Single-line Power Action:
			// Player spends X power to get/collect Y Reward (Power action)
			playerName := matches[1]
			cost, _ := strconv.Atoi(matches[2])
			rewardStr := matches[3]

			// Parse reward amount from string (e.g. "1 spade(s)" -> 1)
			reward := p.parseAmount(rewardStr)

			// fmt.Printf("Matched PowerAction: %s spends %d to get %d\n", playerName, cost, reward)
			p.handlePowerAction(playerName, cost, reward)

		} else if matches := reBurn.FindStringSubmatch(line); len(matches) > 3 {
			// Single-line Burn:
			// Player sacrificed X power in Bowl 2 to get Y power from Bowl 2 to Bowl 3
			playerName := matches[1]
			amount, _ := strconv.Atoi(matches[2])

			// fmt.Printf("Matched Burn: %s sacrificed %d\n", playerName, amount)
			p.handleBurn(playerName, amount)

		} else if matches := reFavorTileStart.FindStringSubmatch(line); len(matches) > 1 {
			// Favor Tile:
			// Player takes a Favor tile
			// Player gains X on the Cult of Y track (Favor tile)
			playerName := matches[1]
			// fmt.Printf("Matched Favor Tile Start: %s\n", playerName)
			p.handleFavorTile(playerName)

		} else if matches := reWitchesRide.FindStringSubmatch(line); len(matches) > 2 {
			playerName := matches[1]
			coordStr := matches[2]
			// fmt.Printf("Matched Witches Ride: %s at %s\n", playerName, coordStr)
			p.handleWitchesRide(playerName, coordStr)

		} else if matches := reLeechPays.FindStringSubmatch(line); len(matches) > 4 {
			// Single-line Leech (Pays):
			// Player pays X VP and gets Y power via Structures [Coord]
			playerName := matches[1]
			cost, _ := strconv.Atoi(matches[2])
			amount, _ := strconv.Atoi(matches[3])
			coordStr := matches[4]

			// fmt.Printf("Matched Leech Pays: %s pays %d gets %d at %s\n", playerName, cost, amount, coordStr)
			p.handleLeech(playerName, coordStr, amount, cost, true)

		} else if matches := reLeechGets.FindStringSubmatch(line); len(matches) > 3 {
			// Single-line Leech (Gets):
			// Player gets X power via Structures [Coord]
			playerName := matches[1]
			amount, _ := strconv.Atoi(matches[2])
			coordStr := matches[3]

			// fmt.Printf("Matched Leech Gets: %s gets %d at %s\n", playerName, amount, coordStr)
			p.handleLeech(playerName, coordStr, amount, 0, true)

		} else if matches := rePass.FindStringSubmatch(line); len(matches) > 1 {
			// fmt.Printf("Matched Pass: %s\n", line)
			p.handlePass(matches[1])

		} else if matches := rePriest.FindStringSubmatch(line); len(matches) > 2 {
			// fmt.Printf("Matched Priest: %s\n", line)
			p.handleSendPriest(matches[1], matches[2])

		} else if matches := reShipping.FindStringSubmatch(line); len(matches) > 1 {
			// fmt.Printf("Matched Shipping: %s\n", line)
			p.handleAdvanceShipping(matches[1])

		} else if matches := reExchangeTrack.FindStringSubmatch(line); len(matches) > 3 {
			// Single-line Digging:
			// Player advances on the Exchange Track for [Cost String] and earns X VP
			playerName := matches[1]
			// fmt.Printf("Matched Exchange Track (Digging): %s\n", playerName)
			p.handleAdvanceDigging(playerName)

		} else if matches := reDeclineLeech.FindStringSubmatch(line); len(matches) > 2 {
			// fmt.Printf("Matched Decline Leech: %s\n", line)
			p.handleLeech(matches[1], matches[2], 0, 0, false)
		}
	}

	return p.items, nil
}

func (p *BGAParser) handleBuildDwelling(playerName, coordStr string, isSetup bool) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)

	var action game.Action
	if isSetup {
		action = game.NewSetupDwellingAction(playerID, hex)
		p.items = append(p.items, ActionItem{Action: action})
	} else {
		action = game.NewTransformAndBuildAction(playerID, hex, true, models.TerrainTypeUnknown)

		// Check for Cultists ability
		if cultCode := p.checkForCultistAbility(playerID); cultCode != "" {
			cultAction := &LogCultistAdvanceAction{
				PlayerID: playerID,
				Track:    GetCultTrackFromCode(cultCode),
			}
			compound := &LogCompoundAction{
				Actions: []game.Action{action, cultAction},
			}
			p.items = append(p.items, ActionItem{Action: compound})
		} else {
			p.items = append(p.items, ActionItem{Action: action})
		}
	}
}

func (p *BGAParser) handleUpgrade(playerName, from, to, coordStr string) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)
	newType := parseBuildingType(to)

	action := game.NewUpgradeBuildingAction(playerID, hex, newType)

	// Check for Cultists ability
	cultCode := p.checkForCultistAbility(playerID)

	// Check for Favor Tile (Temple/Sanctuary upgrade)
	// This might be chained after Cultist ability
	var favorAction *LogFavorTileAction
	if newType == models.BuildingTemple || newType == models.BuildingSanctuary {
		// Look ahead for "takes a Favor tile"
		// We need to be careful not to consume lines that checkForCultistAbility might have skipped over if we didn't use it
		// But checkForCultistAbility already consumed the cult line if found.

		// Simple lookahead for favor tile
		reFavor := regexp.MustCompile(fmt.Sprintf(`^%s takes a Favor tile`, regexp.QuoteMeta(playerName)))
		for lookAhead := 0; lookAhead < 20 && p.currentLine+lookAhead < len(p.lines); lookAhead++ {
			lineIndex := p.currentLine + lookAhead
			if p.consumedLines[lineIndex] {
				continue
			}
			line := p.lines[lineIndex]
			if reFavor.MatchString(line) {
				p.consumedLines[lineIndex] = true
				// The next line should be the favor tile selection
				// "Player gains X on the Cult of Y track (Favor tile)"
				// We need to parse that too.
				// Let's use a helper or just parse it here.
				// Actually, handleFavorTileAction does this.
				// But we want to merge it.

				// Look for the favor tile selection line
				for subLookAhead := 1; subLookAhead < 3 && lineIndex+subLookAhead < len(p.lines); subLookAhead++ {
					subLineIndex := lineIndex + subLookAhead
					subLine := p.lines[subLineIndex]
					if matches := reFavorTileAction.FindStringSubmatch(subLine); len(matches) > 3 {
						if p.getPlayerID(matches[1]) == playerID {
							p.consumedLines[subLineIndex] = true
							track := matches[3]
							amount, _ := strconv.Atoi(matches[2])

							// Map track name to code
							trackCode := "F"
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

							favorAction = &LogFavorTileAction{
								PlayerID: playerID,
								Tile:     fmt.Sprintf("FAV-%s%d", trackCode, amount),
							}
							break
						}
					}
				}
				break
			}
		}
	}

	if cultCode != "" || favorAction != nil {
		actions := []game.Action{action}

		if cultCode != "" {
			cultAction := &LogCultistAdvanceAction{
				PlayerID: playerID,
				Track:    GetCultTrackFromCode(cultCode),
			}
			actions = append(actions, cultAction)
		}

		if favorAction != nil {
			actions = append(actions, favorAction)
		}

		compound := &LogCompoundAction{
			Actions: actions,
		}
		p.items = append(p.items, ActionItem{Action: compound})
	} else {
		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleTransform(playerName, coordStr, targetTerrainStr string) {
	hex := parseCoord(coordStr)
	playerID := p.getPlayerID(playerName)

	var targetTerrain models.TerrainType
	if targetTerrainStr != "" {
		switch strings.ToLower(targetTerrainStr) {
		case "plains":
			targetTerrain = models.TerrainPlains
		case "swamp":
			targetTerrain = models.TerrainSwamp
		case "lakes":
			targetTerrain = models.TerrainLake
		case "forest":
			targetTerrain = models.TerrainForest
		case "mountains":
			targetTerrain = models.TerrainMountain
		case "wasteland":
			targetTerrain = models.TerrainWasteland
		case "desert":
			targetTerrain = models.TerrainDesert
		}
	}

	// NOTE: We do NOT merge transform + build here anymore.
	// Each transform is a separate action (T-X), and the build (X) stays separate.
	// This preserves the correct notation format: T-I7.BURN1.C1W1PW:2C.I7
	// The carpet flight / tunneling should only be charged ONCE even with separate actions
	// because the game state tracks SkipAbilityUsedThisAction per hex.

	// Check for Cultists ability
	// NOTE: We do NOT check for Cultists ability here because it's usually triggered by the Build action
	// which follows the Transform action.

	action := game.NewTransformAndBuildAction(playerID, hex, false, targetTerrain)
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

func (p *BGAParser) handleSendPriest(playerName, track string) {
	playerID := p.getPlayerID(playerName)
	if playerID == "" {
		return
	}

	cultTrack := game.CultFire // default
	switch track {
	case "Fire":
		cultTrack = game.CultFire
	case "Water":
		cultTrack = game.CultWater
	case "Earth":
		cultTrack = game.CultEarth
	case "Air":
		cultTrack = game.CultAir
	}

	// Look ahead for "gains X on the Cult of Y track" to determine the spot
	spacesToClimb := 1 // Default to 1 (sacrifice) if not found
	// However, "Forever!" implies it's an action space (2 or 3).
	// If we default to 1, it won't be shown on the track.
	// We should try hard to find the amount.

	// Peek at next few lines
	// Regex to find "gains X" related to cult
	// Examples: "gains 3 on the Cult of Fire track", "gains 2 on the Cult of Water track"
	reGain := regexp.MustCompile(`gains (\d+) on the Cult of`)

	for i := 0; i < 5; i++ {
		if p.currentLine+i >= len(p.lines) {
			break
		}
		line := p.lines[p.currentLine+i]
		if matches := reGain.FindStringSubmatch(line); len(matches) > 1 {
			amount, _ := strconv.Atoi(matches[1])
			spacesToClimb = amount
			break
		}
		// Stop if we hit another action (but be careful not to stop too early)
		// "sends a Priest" or "builds a" usually start new actions
		if strings.Contains(line, "sends a Priest") || strings.Contains(line, "builds a") {
			break
		}
	}

	action := &game.SendPriestToCultAction{
		BaseAction: game.BaseAction{
			Type:     game.ActionSendPriestToCult,
			PlayerID: playerID,
		},
		Track:         cultTrack,
		SpacesToClimb: spacesToClimb,
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
	playerName = strings.TrimSpace(playerName)
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
	// Consume up to 5 lines to find the track info
	var amount int
	var track string

	// Regex for single-line gain: "Player gains X on the Cult of Y track (Favor tile)"
	reFavorTileGain := regexp.MustCompile(`gains (\d+) on the Cult of (.*) track \(Favor tile\)`)

	for i := 0; i < 5; i++ {
		line := p.consumeLine()

		// Check for single-line gain
		if matches := reFavorTileGain.FindStringSubmatch(line); len(matches) > 2 {
			amount, _ = strconv.Atoi(matches[1])
			track = matches[2]
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
				case *game.UpgradeBuildingAction:
					// Check for Auren Stronghold upgrade
					if upgrade, ok := actItem.Action.(*game.UpgradeBuildingAction); ok {
						if upgrade.NewBuildingType == models.BuildingStronghold {
							// Check if player is Auren
							if p.players[playerName] == "Auren" {
								// Merge into compound action
								compoundAction := &LogCompoundAction{
									Actions: []game.Action{upgrade, action},
								}
								p.items[i] = ActionItem{Action: compoundAction}
								inserted = true
								break
							}
						}
					}
					// Fallthrough for other upgrades
					p.items = append(p.items[:i+1], append([]LogItem{newItem}, p.items[i+1:]...)...)
					inserted = true
				case *game.TransformAndBuildAction, *game.SetupDwellingAction:
					// Insert after
					p.items = append(p.items[:i+1], append([]LogItem{newItem}, p.items[i+1:]...)...)
					inserted = true
				}
				if inserted {
					break
				}
			}
		}
	}

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
	// Digging action is ActionAdvanceDigging
	// The cost and reward are now in the single line matched by reExchangeTrack
	// We don't need to consume lines anymore.

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
	h, err := ConvertLogCoordToAxial(s)
	if err != nil {
		// fmt.Printf("Error parsing coord %s: %v\n", s, err)
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

func (p *BGAParser) extractCoord(line string) string {
	if strings.Contains(line, "[") && strings.Contains(line, "]") {
		// Extract content between brackets
		start := strings.Index(line, "[")
		end := strings.Index(line, "]")
		if start != -1 && end != -1 && end > start {
			return line[start+1 : end]
		}
	}
	return ""
}

func (p *BGAParser) consumeUntilCoord() string {
	// Consume lines until we find one that looks like a coord [A1]
	// Limit to avoid infinite loop
	for i := 0; i < 5; i++ {
		line := p.consumeLine()
		return p.extractCoord(line)
	}
	return ""
}

func (p *BGAParser) peekLine() string {
	if p.currentLine >= len(p.lines) {
		return ""
	}
	return strings.TrimSpace(p.lines[p.currentLine])
}

func (p *BGAParser) parseAmount(s string) int {
	// Extract first number from string
	// e.g. "1 spade(s)" -> 1
	// "2 workers" -> 2
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		val, _ := strconv.Atoi(matches[1])
		return val
	}
	return 0
}
func (p *BGAParser) handleReclaimPriest(playerID, line string) {
	matches := reReclaimPriest.FindStringSubmatch(line)
	if len(matches) > 2 {
		cultName := matches[2]
		track := parseCultTrack(cultName)
		// Reclaiming implies placing on the "1" spot (order of the cult)
		action := &game.SendPriestToCultAction{
			BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: playerID},
			Track:         track,
			SpacesToClimb: 1,
		}
		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleAurenStronghold(playerID, line string) {
	matches := reAurenStronghold.FindStringSubmatch(line)
	if len(matches) > 2 {
		cultName := matches[2]
		track := parseCultTrack(cultName)
		// Auren Stronghold action: ACT-SH-[Track]
		// We'll represent this as a SpecialAction
		action := &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: fmt.Sprintf("ACT-SH-%s", getCultShortCode(track)),
		}
		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleFavorTileAction(playerID, line string) {
	matches := reFavorTileAction.FindStringSubmatch(line)
	if len(matches) > 3 {
		// amountStr := matches[2] // Usually 1
		// amount, _ := strconv.Atoi(amountStr)
		cultName := matches[3]
		track := parseCultTrack(cultName)
		trackCode := getCultShortCode(track)
		// Favor tile action: ACT-FAV-[Track]
		// This is a special action (using the tile), not taking a tile
		action := &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: fmt.Sprintf("ACT-FAV-%s", trackCode),
		}

		// Check if previous action was Auren Stronghold upgrade (UP-TH-C3)
		// If so, merge this action into it
		if len(p.items) > 0 {
			lastItem := p.items[len(p.items)-1]
			if actionItem, ok := lastItem.(ActionItem); ok {
				if upgradeAction, ok := actionItem.Action.(*game.UpgradeBuildingAction); ok {
					if upgradeAction.NewBuildingType == models.BuildingStronghold {
						// Merge!
						compoundAction := &LogCompoundAction{
							Actions: []game.Action{upgradeAction, action},
						}
						p.items[len(p.items)-1] = ActionItem{Action: compoundAction}
						return
					}
				}
			}
		}

		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleBonusCardCult(playerID, track string) {
	// Bonus card cult action: +1 cult step from BON2
	// Format: ACT-BON-[Track]
	cultTrack := parseCultTrack(track)
	trackCode := getCultShortCode(cultTrack)

	action := &LogSpecialAction{
		PlayerID:   playerID,
		ActionCode: fmt.Sprintf("ACT-BON-%s", trackCode),
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleBonusCardSpade(playerID, line string) {
	// Look for coordinate in current line or next
	coord := p.extractCoord(line)
	if coord == "" {
		coord = p.consumeUntilCoord()
	}
	if coord != "" {
		// Check if next lines include a dwelling build at the same coordinate
		// BGA log often has: "transforms ... [G3]" followed by "builds a Dwelling ... [G3]"
		// If so, we should combine them into ACTS-G3.G3 (transform + build)
		buildDwelling := false

		// Look ahead for "builds a Dwelling" at the same coordinate
		reDwellingBuild := regexp.MustCompile(`builds a Dwelling`)
		dwellingLineIndex := -1
		for lookAhead := 0; lookAhead < 5 && p.currentLine+lookAhead < len(p.lines); lookAhead++ {
			lineIndex := p.currentLine + lookAhead
			nextLine := p.lines[lineIndex]
			if reDwellingBuild.MatchString(nextLine) {
				// Check if it's at the same coordinate
				nextCoord := p.extractCoordFromLine(nextLine)
				if nextCoord == coord {
					buildDwelling = true
					dwellingLineIndex = lineIndex
					break
				}
			}
			// If we hit another action start (like "passes", "upgrades", etc.), stop looking
			if strings.Contains(nextLine, " passes") ||
				strings.Contains(nextLine, " upgrades") ||
				strings.Contains(nextLine, " transforms") ||
				strings.Contains(nextLine, " sends a Priest") {
				break
			}
		}

		// Mark the dwelling build line as consumed if we found one
		if dwellingLineIndex >= 0 {
			p.consumedLines[dwellingLineIndex] = true
		}

		var actionCode string
		if buildDwelling {
			// Combined action: transform + build at same hex
			actionCode = fmt.Sprintf("ACTS-%s.%s", coord, strings.ToLower(coord))
		} else {
			// Just transform
			actionCode = fmt.Sprintf("ACTS-%s", coord)

			// Try to parse target terrain from the line
			// "transforms a Terrain space lakes → swamp for"
			reTransform := regexp.MustCompile(`transforms a Terrain space .* → (.*) for`)
			if matches := reTransform.FindStringSubmatch(line); len(matches) > 1 {
				targetTerrainName := strings.TrimSpace(matches[1])
				var terrainCode string
				switch strings.ToLower(targetTerrainName) {
				case "plains":
					terrainCode = "Br"
				case "swamp":
					terrainCode = "Bk"
				case "lakes", "lake":
					terrainCode = "Bl"
				case "forest":
					terrainCode = "G"
				case "mountains", "mountain":
					terrainCode = "Gy"
				case "wasteland":
					terrainCode = "R"
				case "desert":
					terrainCode = "Y"
				}

				if terrainCode != "" {
					actionCode += "-" + terrainCode
				}
			}
		}

		action := &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: actionCode,
		}

		// Check for Cultists ability
		if cultCode := p.checkForCultistAbility(playerID); cultCode != "" {
			cultAction := &LogCultistAdvanceAction{
				PlayerID: playerID,
				Track:    GetCultTrackFromCode(cultCode),
			}
			compound := &LogCompoundAction{
				Actions: []game.Action{action, cultAction},
			}
			p.items = append(p.items, ActionItem{Action: compound})
		} else {
			p.items = append(p.items, ActionItem{Action: action})
		}
	}
}

// extractCoordFromLine extracts coordinate from a specific line without consuming anything
func (p *BGAParser) extractCoordFromLine(line string) string {
	reCoord := regexp.MustCompile(`\[([A-Z]\d+)\]`)
	matches := reCoord.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractTransformInfoFromLine extracts coord and target terrain from a transform line
func (p *BGAParser) extractTransformInfoFromLine(line string) (coord string, targetTerrain string) {
	// Extract coord
	reCoord := regexp.MustCompile(`\[([A-Z]\d+)\]`)
	coordMatches := reCoord.FindStringSubmatch(line)
	if len(coordMatches) > 1 {
		coord = coordMatches[1]
	}

	// Extract target terrain from "→ [terrain]" pattern
	reTransform := regexp.MustCompile(`→\s*(\w+)`)
	terrainMatches := reTransform.FindStringSubmatch(line)
	if len(terrainMatches) > 1 {
		targetTerrain = strings.TrimSpace(terrainMatches[1])
	}

	return coord, targetTerrain
}

func (p *BGAParser) handleBridgePower(playerID, line string) {
	// Look ahead for coordinate [C2-D4]
	// consumeUntilCoord might not work for bridge coords like [C2-D4]
	// We need a custom consumer or check the next line
	// Usually coords are on the next line or at end of line
	// The log says "[C2-D4]"
	reBridgeCoords := regexp.MustCompile(`\[([A-Z]\d+)-([A-Z]\d+)\]`)

	// Check current line first
	matches := reBridgeCoords.FindStringSubmatch(line)
	if len(matches) == 0 {
		// Check next line
		if p.currentLine < len(p.lines) {
			nextLine := p.lines[p.currentLine]
			matches = reBridgeCoords.FindStringSubmatch(nextLine)
			if len(matches) > 0 {
				p.currentLine++
			}
		}
	}

	if len(matches) > 2 {
		// Found coords - output ACT1-C2-D4 format
		coord1 := matches[1] // e.g., "B3"
		coord2 := matches[2] // e.g., "C3"
		action := &LogPowerAction{
			PlayerID:   playerID,
			ActionCode: fmt.Sprintf("ACT1-%s-%s", coord1, coord2),
		}
		p.items = append(p.items, ActionItem{Action: action})
	} else {
		// No coords found - output ACT1 (fallback)
		action := &LogPowerAction{
			PlayerID:   playerID,
			ActionCode: "ACT1",
		}
		p.items = append(p.items, ActionItem{Action: action})
	}
}

func (p *BGAParser) handleMermaidsRiverTown(playerID, riverCoord string) {
	// Parse river coordinate (e.g., "R~D5") to axial
	riverHex, err := ConvertRiverCoordToAxial(riverCoord)
	if err != nil {
		// Log error but continue parsing
		return
	}

	// Create ACT-TOWN action with river hex coordinates
	// Format: ACT-TOWN-Q_R where Q and R are the axial coordinates
	actionCode := fmt.Sprintf("ACT-TOWN-%d_%d", riverHex.Q, riverHex.R)

	action := &LogSpecialAction{
		PlayerID:   playerID,
		ActionCode: actionCode,
	}

	// Append the action
	p.items = append(p.items, ActionItem{Action: action})

	// Set pending flag for town VP merge
	// The players map is Name -> Faction, so we need to find the name for this faction/playerID
	for name, faction := range p.players {
		if faction == playerID {
			p.townPending[name] = true
			break
		}
	}
}

func (p *BGAParser) handleTownFound(playerName string) {
	// Set pending flag
	p.townPending[playerName] = true
}

func (p *BGAParser) handleTownVP(playerName, vpStr string) {
	// Check if town is pending
	if !p.townPending[playerName] {
		return
	}
	// Reset flag
	p.townPending[playerName] = false

	playerID := p.getPlayerID(playerName)
	if playerID == "" {
		return
	}

	vp, err := strconv.Atoi(vpStr)
	if err != nil {
		return
	}

	// Create LogTownAction
	townAction := &LogTownAction{
		PlayerID: playerID,
		VP:       vp,
	}

	// Merge with previous action
	// We look backwards for the last action by this player
	// Similar to Favor Tile merge logic
	inserted := false
	for i := len(p.items) - 1; i >= 0; i-- {
		item := p.items[i]
		if actItem, ok := item.(ActionItem); ok {
			// Check if this action belongs to the player
			if actItem.Action.GetPlayerID() == playerID {
				// Merge!
				// Check if it's already a compound action
				if compound, ok := actItem.Action.(*LogCompoundAction); ok {
					compound.Actions = append(compound.Actions, townAction)
					p.items[i] = ActionItem{Action: compound}
				} else {
					// Create new compound action
					compound := &LogCompoundAction{
						Actions: []game.Action{actItem.Action, townAction},
					}
					p.items[i] = ActionItem{Action: compound}
				}
				inserted = true
				break
			}
		}
	}

	if !inserted {
		// If no previous action found (unlikely), append as new item
		p.items = append(p.items, ActionItem{Action: townAction})
	}
}

func (p *BGAParser) handleConversion(playerName, spent, collects string) {
	// spentStr: "1 power 0 Priests 0 workers"
	// collectsStr: "0 Priests 0 workers 1 coins"

	playerID := p.getPlayerID(playerName)
	cost := p.parseResourceMap(spent)
	reward := p.parseResourceMap(collects)

	// Filter out zero values
	for k, v := range cost {
		if v == 0 {
			delete(cost, k)
		}
	}
	for k, v := range reward {
		if v == 0 {
			delete(reward, k)
		}
	}

	// Simplify conversion by subtracting common resources
	// e.g. Cost: 3 Power, 2 Workers -> Reward: 3 Workers, 2 Coins
	// Becomes: Cost: 3 Power -> Reward: 1 Worker, 2 Coins
	for rType, costAmount := range cost {
		if rewardAmount, ok := reward[rType]; ok {
			if costAmount > rewardAmount {
				cost[rType] -= rewardAmount
				delete(reward, rType)
			} else if costAmount < rewardAmount {
				reward[rType] -= costAmount
				delete(cost, rType)
			} else {
				// Equal amounts, remove from both
				delete(cost, rType)
				delete(reward, rType)
			}
		}
	}

	if len(cost) == 0 && len(reward) == 0 {
		return
	}

	action := &LogConversionAction{
		PlayerID: playerID,
		Cost:     cost,
		Reward:   reward,
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) handleAlchemistsVP(playerID, vpStr, coinsStr string) {
	vp, _ := strconv.Atoi(vpStr)
	coins, _ := strconv.Atoi(coinsStr)

	cost := map[models.ResourceType]int{models.ResourceVictoryPoint: vp}
	reward := map[models.ResourceType]int{models.ResourceCoin: coins}

	action := &LogConversionAction{
		PlayerID: playerID,
		Cost:     cost,
		Reward:   reward,
	}
	p.items = append(p.items, ActionItem{Action: action})
}

func (p *BGAParser) parseResourceMap(s string) map[models.ResourceType]int {
	res := make(map[models.ResourceType]int)
	// Regex to find "N unit"
	re := regexp.MustCompile(`(\d+) (\w+)`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		amount, _ := strconv.Atoi(match[1])
		unit := strings.ToLower(match[2])
		switch unit {
		case "power":
			res[models.ResourcePower] += amount
		case "priests", "priest":
			res[models.ResourcePriest] += amount
		case "workers", "worker":
			res[models.ResourceWorker] += amount
		case "coins", "coin":
			res[models.ResourceCoin] += amount
		case "vp":
			res[models.ResourceVictoryPoint] += amount
		}
	}
	return res
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (p *BGAParser) handleGiantsStronghold(playerID, coordStr string) {
	// Giants Stronghold: transform 2 spades for free to home terrain
	// Emit as ACT-SH-S-[coord] (S = Spade, as expected by executeStrongholdAction)
	p.items = append(p.items, ActionItem{
		Action: &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: "ACT-SH-S-" + coordStr,
		},
	})
}

func (p *BGAParser) handleSwarmlingStronghold(playerID, coordStr string) {
	// Swarmlings Stronghold: free Dwelling -> Trading House upgrade
	// Emit as ACT-SH-TP-[coord] (TP = Trading Post, as expected by executeStrongholdAction)
	p.items = append(p.items, ActionItem{
		Action: &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: "ACT-SH-TP-" + coordStr,
		},
	})
}

func (p *BGAParser) handleNomadsStronghold(playerID, coordStr string) {
	// Nomads Stronghold (Sandstorm): transform any hex to desert for free
	// Format: ACT-SH-T-[coord] (transform only) or ACT-SH-T-[coord].[coord] (transform + build)

	// Look ahead for dwelling build at the same coordinate and consume it
	buildDwelling := false
	reDwellingBuild := regexp.MustCompile(`builds a Dwelling`)
	for lookAhead := 0; lookAhead < 5 && p.currentLine+lookAhead < len(p.lines); lookAhead++ {
		lineIndex := p.currentLine + lookAhead
		nextLine := p.lines[lineIndex]
		if reDwellingBuild.MatchString(nextLine) {
			// Check if it's at the same coordinate
			nextCoord := p.extractCoordFromLine(nextLine)
			if nextCoord == coordStr {
				buildDwelling = true
				// Mark this line as consumed so it's not parsed again
				p.consumedLines[lineIndex] = true
				break
			}
		}
		// If we hit another action start, stop looking
		if strings.Contains(nextLine, " passes") ||
			strings.Contains(nextLine, " upgrades") ||
			strings.Contains(nextLine, " transforms") ||
			strings.Contains(nextLine, " sends a Priest") {
			break
		}
	}

	var actionCode string
	if buildDwelling {
		actionCode = fmt.Sprintf("ACT-SH-T-%s.%s", coordStr, strings.ToLower(coordStr))
	} else {
		actionCode = fmt.Sprintf("ACT-SH-T-%s", coordStr)
	}

	p.items = append(p.items, ActionItem{
		Action: &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: actionCode,
		},
	})
}

func (p *BGAParser) handleCMDoubleTurn(playerID string) {
	// Chaos Magicians Stronghold: double-turn
	// Emit as ACT-SH-2X
	p.items = append(p.items, ActionItem{
		Action: &LogSpecialAction{
			PlayerID:   playerID,
			ActionCode: "ACT-SH-2X",
		},
	})
}

func (p *BGAParser) handleHalflingsStrongholdSpades(playerID string) {
	// Halflings Stronghold: 3 spades for transform
	// This is triggered after building the stronghold
	// Look ahead for transform actions from this player and merge them with the previous upgrade action
	// NOTE: Transforms may be in a DIFFERENT Move, so we look past Move boundaries

	// Look ahead for transform lines
	reTransform := regexp.MustCompile(`transforms a Terrain space`)
	reDecline := regexp.MustCompile(`declines building`)
	playerName := p.getPlayerName(playerID)

	var transformCoords []string
	var targetTerrains []string

	for lookAhead := 0; lookAhead < 50 && p.currentLine+lookAhead < len(p.lines); lookAhead++ {
		lineIndex := p.currentLine + lookAhead
		nextLine := p.lines[lineIndex]

		// Stop at "declines building" from this player
		if reDecline.MatchString(nextLine) && strings.Contains(nextLine, playerName) {
			p.consumedLines[lineIndex] = true
			break
		}

		// Check if it's the same player's transform
		if reTransform.MatchString(nextLine) && strings.Contains(nextLine, playerName) {
			coord, terrain := p.extractTransformInfoFromLine(nextLine)
			if coord != "" {
				transformCoords = append(transformCoords, coord)
				targetTerrains = append(targetTerrains, terrain)
				p.consumedLines[lineIndex] = true
			}
			// Stop after collecting 3 transforms (the stronghold gives exactly 3 spades)
			if len(transformCoords) >= 3 {
				break
			}
		}

		// Skip Move headers - don't break, continue looking
		// Move boundaries don't stop the spade application
	}

	// If we found transform actions, merge them with the previous upgrade action
	if len(transformCoords) > 0 && len(p.items) > 0 {
		// Find the last action for this player (the stronghold upgrade)
		// It might not be the very last item if there was a leech in between
		var targetIndex int = -1
		for i := len(p.items) - 1; i >= 0; i-- {
			item := p.items[i]
			if actionItem, ok := item.(ActionItem); ok {
				if actionItem.Action.GetPlayerID() == playerID {
					targetIndex = i
					break
				}
			}
		}

		if targetIndex >= 0 {
			actionItem := p.items[targetIndex].(ActionItem)
			// Create a compound action by making LogHalflingsSpadeAction
			halflingsAction := &LogHalflingsSpadeAction{
				PlayerID:        playerID,
				TransformCoords: transformCoords,
				TargetTerrains:  targetTerrains,
			}
			compound := &LogCompoundAction{
				Actions: []game.Action{actionItem.Action, halflingsAction},
			}
			p.items[targetIndex] = ActionItem{Action: compound}
		}
	}
}

func (p *BGAParser) getPlayerName(playerID string) string {
	// Get player name from ID (reverse of getPlayerID)
	for name, id := range p.players {
		if id == playerID {
			return name
		}
	}
	return playerID
}

// checkForCultistAbility looks ahead for "gains X on the Cult of Y track (Cultists ability)"
// and returns the cult track code (F, W, E, A) if found.
// It consumes the line if found.
func (p *BGAParser) checkForCultistAbility(playerID string) string {
	reCultistAbility := regexp.MustCompile(`(.*) gains \d+ on the Cult of (\w+) track \(Cultists ability\)`)
	reLeech := regexp.MustCompile(`gets \d+ power via Structures`)
	reDecline := regexp.MustCompile(`declines doing Conversions`)
	rePowerCap := regexp.MustCompile(`Power gain via Structures is capped`)
	reAutoAccept := regexp.MustCompile(`You have enabled automatic acceptance of Power`)
	reVPGain := regexp.MustCompile(`gets \d+ VP`)

	// Look ahead a reasonable number of lines (e.g., 10) to skip over leeches
	for lookAhead := 0; lookAhead < 15 && p.currentLine+lookAhead < len(p.lines); lookAhead++ {
		lineIndex := p.currentLine + lookAhead
		line := p.lines[lineIndex]

		// Skip already consumed lines
		if p.consumedLines[lineIndex] {
			continue
		}

		// Check for Cultists ability
		if matches := reCultistAbility.FindStringSubmatch(line); len(matches) > 2 {
			// Verify it's the same player
			if p.getPlayerID(matches[1]) == playerID {
				trackName := matches[2]
				p.consumedLines[lineIndex] = true

				switch trackName {
				case "Fire":
					return "F"
				case "Water":
					return "W"
				case "Earth":
					return "E"
				case "Air":
					return "A"
				}
				return "F" // Default fallback
			}
		}

		// Allow skipping over:
		// 1. Leech actions (by anyone)
		// 2. Decline conversions (by anyone)
		// 3. Power cap messages
		// 4. Auto accept messages
		// 5. VP gain messages (e.g. Scoring tile bonus)
		if reLeech.MatchString(line) ||
			reDecline.MatchString(line) ||
			rePowerCap.MatchString(line) ||
			reAutoAccept.MatchString(line) ||
			reVPGain.MatchString(line) {
			continue
		}

		// Stop if we hit anything else (e.g. another player's move, start of round, etc.)
		// But be careful not to stop on "Move X :" or timestamps which are skipped in main loop
		if strings.Contains(line, "Move ") || strings.Contains(line, " AM") || strings.Contains(line, " PM") {
			continue
		}

		// If we see a player name at the start of the line doing something else, stop
		// Simple heuristic: if line starts with a known player name and it's not one of the skipped actions
		for name := range p.players {
			if strings.HasPrefix(line, name) {
				return ""
			}
		}

		// If we hit a phase change
		if strings.HasPrefix(line, "~") {
			continue // Skip phase messages? Or stop? Usually phase messages are fine to skip e.g. auto accept
		}
	}
	return ""
}
