package replay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ParseSnapshot parses a human-readable snapshot string into a GameState
func ParseSnapshot(content string) (*game.GameState, error) {
	lines := strings.Split(content, "\n")
	gs := &game.GameState{
		Players:               make(map[string]*game.Player),
		TurnOrder:             []string{},
		PassOrder:             []string{},
		Map:                   board.NewTerraMysticaMap(), // Initialize empty map
		CultTracks:            game.NewCultTrackState(),
		BonusCards:            game.NewBonusCardState(),
		FavorTiles:            game.NewFavorTileState(),
		TownTiles:             game.NewTownTileState(),
		PowerActions:          game.NewPowerActionState(),
		ScoringTiles:          game.NewScoringTileState(),
		PendingTownFormations: make(map[string][]*game.PendingTownFormation),
	}

	// Parsing state
	currentSection := ""
	currentPlayerID := ""
	stateSubSection := ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for top-level keys
		if strings.HasPrefix(line, "Round:") {
			round, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Round:")))
			if err != nil {
				return nil, fmt.Errorf("invalid round at line %d: %v", i+1, err)
			}
			gs.Round = round
			continue
		}
		if strings.HasPrefix(line, "Phase:") {
			phaseStr := strings.TrimSpace(strings.TrimPrefix(line, "Phase:"))
			gs.Phase = stringToPhase(phaseStr)
			continue
		}
		if strings.HasPrefix(line, "MapType:") {
			// TODO: Handle map type if needed
			continue
		}
		if strings.HasPrefix(line, "Turn:") {
			// Current player turn
			// We'll set this after we know the player IDs
			continue
		}
		if strings.HasPrefix(line, "TurnOrder:") {
			// Parse turn order
			orderStr := strings.TrimSpace(strings.TrimPrefix(line, "TurnOrder:"))
			orderStr = strings.Trim(orderStr, "[]")
			parts := strings.Split(orderStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					// We store faction names as player IDs for now, or need a mapping
					// Assuming playerID == Faction Name for snapshot reconstruction
					gs.TurnOrder = append(gs.TurnOrder, p)
				}
			}
			continue
		}
		if strings.HasPrefix(line, "PassOrder:") {
			orderStr := strings.TrimSpace(strings.TrimPrefix(line, "PassOrder:"))
			orderStr = strings.Trim(orderStr, "[]")
			parts := strings.Split(orderStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					gs.PassOrder = append(gs.PassOrder, p)
				}
			}
			continue
		}

		// Check for Sections
		if line == "Players:" {
			currentSection = "Players"
			stateSubSection = "" // Reset subsection for new section
			continue
		}
		if line == "Map:" {
			currentSection = "Map"
			stateSubSection = "" // Reset subsection for new section
			continue
		}
		if line == "State:" {
			currentSection = "State"
			stateSubSection = "" // Reset subsection for new section
			continue
		}

		// Handle Section Content
		switch currentSection {
		case "Players":
			// Check if it's a player header (indented by 2 spaces, ends with :)
			trimmed := strings.TrimSpace(line)
			// Calculate indent based on leading spaces only
			indent := len(line) - len(strings.TrimLeft(line, " "))

			// Allow 0 indent as fallback (some environments/generators might strip spaces?)
			// Also ensure it's not a known field (which might have similar indentation if headers are 0-indented)
			isField := strings.HasPrefix(trimmed, "VP:") ||
				strings.HasPrefix(trimmed, "Res:") ||
				strings.HasPrefix(trimmed, "Keys:") ||
				strings.HasPrefix(trimmed, "Shipping:") ||
				strings.HasPrefix(trimmed, "Digging:") ||
				strings.HasPrefix(trimmed, "Range:") ||
				strings.HasPrefix(trimmed, "Cult:") ||
				strings.HasPrefix(trimmed, "Map:") ||
				strings.HasPrefix(trimmed, "Bridges:") ||
				strings.HasPrefix(trimmed, "Towns:") ||
				strings.HasPrefix(trimmed, "Bonus:") ||
				strings.HasPrefix(trimmed, "Favor:") ||
				strings.HasPrefix(trimmed, "StrongholdAction:")

			if (indent == 2 || indent == 0) && strings.HasSuffix(trimmed, ":") && !isField {
				// New player block
				factionName := strings.TrimSuffix(trimmed, ":")
				currentPlayerID = factionName

				// Initialize player
				ft := models.FactionTypeFromString(factionName)
				if ft == models.FactionUnknown {
					return nil, fmt.Errorf("unknown faction: %s", factionName)
				}
				f := factions.NewFaction(ft)

				player := &game.Player{
					Faction:       f,
					Resources:     game.NewResourcePool(factions.Resources{}),
					CultPositions: make(map[game.CultTrack]int),
					TownTiles:     []models.TownTileType{},
				}
				gs.Players[currentPlayerID] = player

				// Initialize cult track state for this player
				gs.CultTracks.InitializePlayer(currentPlayerID)
				gs.FavorTiles.InitializePlayer(currentPlayerID)
				gs.BonusCards.InitializePlayer(currentPlayerID)
				continue
			}

			// Player attributes
			if currentPlayerID != "" {
				if strings.HasPrefix(line, "VP:") {
					vp, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "VP:")))
					gs.Players[currentPlayerID].VictoryPoints = vp
				} else if strings.HasPrefix(line, "Res:") {
					parseResources(line, gs.Players[currentPlayerID])
				} else if strings.HasPrefix(line, "Keys:") {
					keys, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Keys:")))
					gs.Players[currentPlayerID].Keys = keys
				} else if strings.HasPrefix(line, "Shipping:") {
					ship, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Shipping:")))
					gs.Players[currentPlayerID].ShippingLevel = ship
				} else if strings.HasPrefix(line, "Digging:") {
					dig, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Digging:")))
					gs.Players[currentPlayerID].DiggingLevel = dig
				} else if strings.HasPrefix(line, "Cult:") {
					parseCultPositions(line, gs, currentPlayerID)
				} else if strings.HasPrefix(line, "Map:") {
					parsePlayerBuildings(line, gs, currentPlayerID)
				} else if strings.HasPrefix(line, "Towns:") {
					parsePlayerTowns(line, gs, currentPlayerID)
				} else if strings.HasPrefix(line, "Bonus:") {
					parsePlayerBonus(line, gs, currentPlayerID)
				} else if strings.HasPrefix(line, "Favor:") {
					parsePlayerFavors(line, gs, currentPlayerID)
				}
			}

		case "Map":
			// Parse terraformed hexes
			// Format: q,r: Color
			content := strings.TrimSpace(line)
			if !strings.Contains(content, ":") {
				continue
			}
			parts := strings.SplitN(content, ":", 2)
			coordsStr := strings.TrimSpace(parts[0])
			terrainStr := strings.TrimSpace(parts[1])

			var q, r int
			_, err := fmt.Sscanf(coordsStr, "%d,%d", &q, &r)
			if err != nil {
				return nil, fmt.Errorf("invalid hex coordinates at line %d: %v", i+1, err)
			}

			hex := board.Hex{Q: q, R: r}
			terrainType := stringToTerrainType(terrainStr)
			if terrainType == models.TerrainTypeUnknown {
				return nil, fmt.Errorf("unknown terrain type '%s' at line %d", terrainStr, i+1)
			}

			// Ensure the hex exists in the map and set its terrain
			if mapHex, ok := gs.Map.Hexes[hex]; ok {
				mapHex.Terrain = terrainType
			} else {
				// If the hex doesn't exist, add it (though NewTerraMysticaMap should pre-populate)
				gs.Map.Hexes[hex] = &board.MapHex{Coord: hex, Terrain: terrainType}
			}

		case "State":
			// Handle lines that start with key (single line lists)
			if strings.HasPrefix(line, "ScoringTiles:") {
				content := strings.TrimSpace(strings.TrimPrefix(line, "ScoringTiles:"))
				content = strings.Trim(content, "[]")
				tileNames := parseQuotedStrings(content)
				allScoringTiles := game.GetAllScoringTiles()
				for _, name := range tileNames {
					tileType := game.ScoringTileTypeFromString(name)
					if tileType != game.ScoringTileUnknown {
						// Find the full tile struct
						for _, t := range allScoringTiles {
							if t.Type == tileType {
								gs.ScoringTiles.Tiles = append(gs.ScoringTiles.Tiles, t)
								break
							}
						}
					}
				}
				stateSubSection = "" // Reset after single-line parse
				continue
			} else if strings.HasPrefix(line, "Bonuses:") {
				stateSubSection = "Bonuses"
				continue
			} else if strings.HasPrefix(line, "Favors:") {
				stateSubSection = "Favors"
				continue
			} else if strings.HasPrefix(line, "Towns:") {
				stateSubSection = "Towns"
				continue
			} else if strings.HasPrefix(line, "PowerActions:") {
				stateSubSection = "PowerActions"
				continue
			} else if strings.HasPrefix(line, "CultBoard:") {
				stateSubSection = "CultBoard"
				continue
			}

			// Content line for subsection
			switch stateSubSection {
			case "Bonuses":
				// "BON1": 2
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					bonusName := strings.Trim(strings.TrimSpace(parts[0]), `"`)
					count, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
					bonusType := game.BonusCardTypeFromString(bonusName)
					if bonusType != game.BonusCardUnknown {
						gs.BonusCards.Available[bonusType] = count
					}
				}
			case "Favors":
				// "FAV1": 1
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					favorName := strings.Trim(strings.TrimSpace(parts[0]), `"`)
					count, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
					favorType := game.FavorTileTypeFromString(favorName)
					if favorType != game.FavorTileUnknown {
						gs.FavorTiles.Available[favorType] = count
					}
				}
			case "Towns":
				// "TW1": 1
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					townName := strings.Trim(strings.TrimSpace(parts[0]), `"`)
					count, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
					townType := models.TownTileTypeFromString(townName)
					if townType != models.TownTileUnknown {
						gs.TownTiles.Available[townType] = count
					}
				}
			case "PowerActions":
				// "bridge": Used
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					actionName := strings.Trim(strings.TrimSpace(parts[0]), `"`)
					status := strings.TrimSpace(parts[1])
					if status == "Used" {
						actionType := game.PowerActionTypeFromString(actionName)
						if actionType != game.PowerActionUnknown {
							gs.PowerActions.MarkUsed(actionType)
						}
					}
				}
			case "CultBoard":
				// Fire: [Player1, Player2]
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cultName := strings.TrimSpace(parts[0])
					playerListStr := strings.Trim(strings.TrimSpace(parts[1]), "[]")
					playerIDs := strings.Split(playerListStr, ",")
					cultTrack := game.CultTrackFromString(cultName)
					if cultTrack != game.CultUnknown {
						for _, pid := range playerIDs {
							pid = strings.TrimSpace(pid)
							if pid != "" {
								if gs.CultTracks.PriestsOnActionSpaces[pid] == nil {
									gs.CultTracks.PriestsOnActionSpaces[pid] = make(map[game.CultTrack]int)
								}
								gs.CultTracks.PriestsOnActionSpaces[pid][cultTrack]++
							}
						}
					}
				}
			}
		}
	}

	return gs, nil
}

// Helpers

func stringToPhase(s string) game.GamePhase {
	switch s {
	case "Setup":
		return game.PhaseSetup
	case "FactionSelection":
		return game.PhaseFactionSelection
	case "Income":
		return game.PhaseIncome
	case "Action":
		return game.PhaseAction
	case "Cleanup":
		return game.PhaseCleanup
	case "End":
		return game.PhaseEnd
	default:
		return game.PhaseAction // Default
	}
}

func stringToTerrainType(s string) models.TerrainType {
	switch s {
	case "Forest":
		return models.TerrainForest
	case "Mountain":
		return models.TerrainMountain
	case "Lake":
		return models.TerrainLake
	case "Desert":
		return models.TerrainDesert
	case "Swamp":
		return models.TerrainSwamp
	case "Plain":
		return models.TerrainPlains
	case "River":
		return models.TerrainRiver
	default:
		return models.TerrainTypeUnknown
	}
}

func parseResources(line string, p *game.Player) {
	// Res: 1w 2p 8c / 2/2/8
	content := strings.TrimSpace(strings.TrimPrefix(line, "Res:"))
	var w, pr, c, b1, b2, b3 int
	// Use Sscanf to parse the complex format
	_, err := fmt.Sscanf(content, "%dw %dp %dc / %d/%d/%d", &w, &pr, &c, &b1, &b2, &b3)
	if err != nil {
		// Log or handle error, for now just proceed with 0s
		return
	}
	p.Resources.Workers = w
	p.Resources.Priests = pr
	p.Resources.Coins = c
	p.Resources.Power.Bowl1 = b1
	p.Resources.Power.Bowl2 = b2
	p.Resources.Power.Bowl3 = b3
}

func parseCultPositions(line string, gs *game.GameState, pid string) {
	// Cult: 0/0/0/0
	content := strings.TrimSpace(strings.TrimPrefix(line, "Cult:"))
	var f, w, e, a int
	_, err := fmt.Sscanf(content, "%d/%d/%d/%d", &f, &w, &e, &a)
	if err != nil {
		return
	}

	// Update player positions
	// We need to update both player struct and CultTrackState
	// Note: CultTrackState.AdvancePlayer logic is complex, we should set directly for snapshot
	gs.CultTracks.PlayerPositions[pid][game.CultFire] = f
	gs.CultTracks.PlayerPositions[pid][game.CultWater] = w
	gs.CultTracks.PlayerPositions[pid][game.CultEarth] = e
	gs.CultTracks.PlayerPositions[pid][game.CultAir] = a

	if p, ok := gs.Players[pid]; ok {
		p.CultPositions[game.CultFire] = f
		p.CultPositions[game.CultWater] = w
		p.CultPositions[game.CultEarth] = e
		p.CultPositions[game.CultAir] = a
	}
}

func parsePlayerBuildings(line string, gs *game.GameState, pid string) {
	// Map: 4,4:D, 5,-2:TP
	content := strings.TrimSpace(strings.TrimPrefix(line, "Map:"))
	if content == "" {
		return
	}

	buildingParts := strings.Split(content, ", ") // Split by ", " to handle "q,r:B" correctly
	for _, part := range buildingParts {
		// part = "4,4:D"
		if !strings.Contains(part, ":") {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		coordsStr := strings.TrimSpace(kv[0])
		bCode := strings.TrimSpace(kv[1])

		var q, r int
		_, err := fmt.Sscanf(coordsStr, "%d,%d", &q, &r)
		if err != nil {
			continue // Skip malformed coordinates
		}

		hex := board.Hex{Q: q, R: r}
		bType := codeToBuildingType(bCode)
		if bType == models.BuildingTypeUnknown {
			continue // Skip unknown building types
		}

		// Ensure the hex exists in the map
		mapHex, ok := gs.Map.Hexes[hex]
		if !ok {
			// If hex doesn't exist, create it (should be pre-populated by NewTerraMysticaMap)
			mapHex = &board.MapHex{Coord: hex, Terrain: models.TerrainTypeUnknown} // Terrain will be set by Map section
			gs.Map.Hexes[hex] = mapHex
		}

		// Set building on the map hex
		mapHex.Building = &models.Building{
			Type:     bType,
			PlayerID: pid,
			Faction:  gs.Players[pid].Faction.GetType(),
		}
		// Set PowerValue
		switch bType {
		case models.BuildingDwelling:
			mapHex.Building.PowerValue = 1
		case models.BuildingTradingHouse, models.BuildingTemple:
			mapHex.Building.PowerValue = 2
		case models.BuildingStronghold, models.BuildingSanctuary:
			mapHex.Building.PowerValue = 3
		}

		// Set terrain to faction's home terrain
		// This assumes the building is on home terrain (which is true unless Sandstorm/etc, but snapshot should handle those via Map section if needed)
		// Actually, for standard factions, building implies home terrain.
		mapHex.Terrain = gs.Players[pid].Faction.GetHomeTerrain()
	}
}

func codeToBuildingType(code string) models.BuildingType {
	switch code {
	case "D":
		return models.BuildingDwelling
	case "TP", "TH": // Trading Post / Trading House
		return models.BuildingTradingHouse
	case "TE": // Temple
		return models.BuildingTemple
	case "SA": // Sanctuary
		return models.BuildingSanctuary
	case "SH": // Stronghold
		return models.BuildingStronghold
	case "BR": // Bridge
		return models.BuildingBridge
	default:
		return models.BuildingTypeUnknown // Fallback
	}
}

func parsePlayerTowns(line string, gs *game.GameState, pid string) {
	// Towns: "5 VP, 6 Coins", "7 VP, 1 Power"
	content := strings.TrimSpace(strings.TrimPrefix(line, "Towns:"))
	if content == "" {
		return
	}

	townStrings := parseQuotedStrings(content)
	if p, ok := gs.Players[pid]; ok {
		for _, townStr := range townStrings {
			townType := models.TownTileTypeFromString(townStr) // This needs to be robust
			if townType != models.TownTileUnknown {
				p.TownTiles = append(p.TownTiles, townType)
			}
		}
	}
}

func parsePlayerBonus(line string, gs *game.GameState, pid string) {
	// Bonus: "BON1"
	content := strings.TrimSpace(strings.TrimPrefix(line, "Bonus:"))
	if content == "" {
		return
	}
	bonusName := strings.Trim(content, `"`)
	bonusType := game.BonusCardTypeFromString(bonusName)
	if bonusType != game.BonusCardUnknown {
		gs.BonusCards.PlayerCards[pid] = bonusType
		gs.BonusCards.PlayerHasCard[pid] = true
	}
}

func parsePlayerFavors(line string, gs *game.GameState, pid string) {
	// Favor: Fire 3, Water 2
	content := strings.TrimSpace(strings.TrimPrefix(line, "Favor:"))
	if content == "" {
		return
	}
	favorParts := strings.Split(content, ", ")
	if _, ok := gs.Players[pid]; ok {
		for _, part := range favorParts {
			subParts := strings.SplitN(part, " ", 2)
			if len(subParts) == 2 {
				favorName := strings.TrimSpace(subParts[0]) + " " + strings.TrimSpace(subParts[1]) // e.g., "Fire 3"
				favorType := game.FavorTileTypeFromString(favorName)
				if favorType != game.FavorTileUnknown {
					gs.FavorTiles.PlayerTiles[pid] = append(gs.FavorTiles.PlayerTiles[pid], favorType)
				}
			}
		}
	}
}

// parseQuotedStrings extracts strings enclosed in double quotes from a comma-separated list.
// Example: `"String 1", "String 2"` -> `["String 1", "String 2"]`
func parseQuotedStrings(s string) []string {
	var result []string
	s = strings.TrimSpace(s)
	if s == "" {
		return result
	}

	// This is a simple parser, might not handle escaped quotes within strings
	parts := strings.Split(s, `",`)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, `"`) {
			trimmed = strings.TrimPrefix(trimmed, `"`)
		}
		if strings.HasSuffix(trimmed, `"`) {
			trimmed = strings.TrimSuffix(trimmed, `"`)
		}
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
