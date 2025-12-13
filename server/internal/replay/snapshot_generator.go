package replay

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// GenerateSnapshot creates a human-readable snapshot of the current game state
func GenerateSnapshot(gs *game.GameState) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Round: %d\n", gs.Round))
	sb.WriteString(fmt.Sprintf("Phase: %s\n", phaseToString(gs.Phase)))
	sb.WriteString("MapType: base\n") // Currently only base map supported

	// Turn info
	if gs.CurrentPlayerIndex >= 0 && gs.CurrentPlayerIndex < len(gs.TurnOrder) {
		currentPlayer := gs.TurnOrder[gs.CurrentPlayerIndex]
		sb.WriteString(fmt.Sprintf("Turn: %s\n", getFactionName(gs, currentPlayer)))
	}

	// Turn Order
	sb.WriteString("TurnOrder: [")
	for i, pid := range gs.TurnOrder {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(getFactionName(gs, pid))
	}
	sb.WriteString("]\n")

	// Pass Order
	sb.WriteString("PassOrder: [")
	for i, pid := range gs.PassOrder {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(getFactionName(gs, pid))
	}
	sb.WriteString("]\n\n")

	// Players
	sb.WriteString("Players:\n")
	// Sort players by turn order for consistent output, or just iterate map?
	// Iterating turn order is better for readability usually, but map iteration is random.
	// Let's use a consistent order: Faction name alphabetical or just TurnOrder?
	// The prompt example used specific faction blocks. Let's iterate through all players.
	// To keep it deterministic, let's sort by Player Name (Faction).
	playerIDs := make([]string, 0, len(gs.Players))
	for pid := range gs.Players {
		playerIDs = append(playerIDs, pid)
	}
	sort.Slice(playerIDs, func(i, j int) bool {
		return getFactionName(gs, playerIDs[i]) < getFactionName(gs, playerIDs[j])
	})

	for _, pid := range playerIDs {
		p := gs.Players[pid]
		factionName := p.Faction.GetType().String()
		sb.WriteString(fmt.Sprintf("  %s:\n", factionName))
		sb.WriteString(fmt.Sprintf("    VP: %d\n", p.VictoryPoints))

		// Resources
		// Resources: [W]w [P]p [C]c / [P1]/[P2]/[P3]
		sb.WriteString(fmt.Sprintf("    Res: %dw %dp %dc / %d/%d/%d\n",
			p.Resources.Workers,
			p.Resources.Priests,
			p.Resources.Coins,
			p.Resources.Power.Bowl1,
			p.Resources.Power.Bowl2,
			p.Resources.Power.Bowl3))

		sb.WriteString(fmt.Sprintf("    Keys: %d\n", p.Keys))
		sb.WriteString(fmt.Sprintf("    Shipping: %d\n", p.ShippingLevel))
		sb.WriteString(fmt.Sprintf("    Digging: %d\n", p.DiggingLevel))

		if p.Faction.GetType() == models.FactionFakirs {
			sb.WriteString(fmt.Sprintf("    Range: %d\n", p.ShippingLevel)) // Reusing shipping level for range as per standard logic
		}

		// Cult: F/W/E/A
		cult := gs.CultTracks
		sb.WriteString(fmt.Sprintf("    Cult: %d/%d/%d/%d\n",
			cult.GetPosition(pid, game.CultFire),
			cult.GetPosition(pid, game.CultWater),
			cult.GetPosition(pid, game.CultEarth),
			cult.GetPosition(pid, game.CultAir)))

		// Map: q,r:B ...
		sb.WriteString("    Map: ")
		buildings := []string{}
		// Scan map for player's buildings
		// We need a deterministic order for hexes. q then r.
		sortedHexes := getSortedHexes(gs.Map)
		for _, h := range sortedHexes {
			mh := gs.Map.GetHex(h)
			if mh.Building != nil && mh.Building.PlayerID == pid {
				bType := buildingTypeToCode(mh.Building.Type)
				buildings = append(buildings, fmt.Sprintf("%d,%d:%s", h.Q, h.R, bType))
			}
		}
		sb.WriteString(strings.Join(buildings, ", "))
		sb.WriteString("\n")

		// Bridges
		sb.WriteString("    Bridges: ")
		playerBridges := []string{}
		// In the current model, bridges don't have explicit owners stored in the map.
		// We infer ownership if the bridge connects two of the player's buildings,
		// or if we can't determine, we might list them in a global section.
		// However, for the snapshot to be player-centric as requested:
		// We'll iterate all bridges and check if this player has buildings at both ends.
		// This is a heuristic and might miss bridges connecting to empty hexes (expansion).
		// Given the limitation, we'll list bridges that connect at least one of the player's buildings.
		// Or better, we should probably move Bridges to the Map/State section if ownership is ambiguous.
		// But let's try to find bridges connected to this player's buildings.
		for bridgeKey := range gs.Map.Bridges {
			h1 := gs.Map.GetHex(bridgeKey.H1)
			h2 := gs.Map.GetHex(bridgeKey.H2)
			isConnected := false
			if h1 != nil && h1.Building != nil && h1.Building.PlayerID == pid {
				isConnected = true
			}
			if h2 != nil && h2.Building != nil && h2.Building.PlayerID == pid {
				isConnected = true
			}

			// If connected to player's building, list it.
			// Note: This might list the same bridge for two players if they share it (which shouldn't happen in TM rules usually)
			// But bridges are usually exclusive.
			if isConnected {
				// Format: q1,r1-q2,r2
				bridgeStr := fmt.Sprintf("%d,%d-%d,%d", bridgeKey.H1.Q, bridgeKey.H1.R, bridgeKey.H2.Q, bridgeKey.H2.R)
				playerBridges = append(playerBridges, bridgeStr)
			}
		}
		sort.Strings(playerBridges)
		sb.WriteString(strings.Join(playerBridges, ", "))
		sb.WriteString("\n")

		// Towns
		sb.WriteString("    Towns: ")
		towns := []string{}
		for _, t := range p.TownTiles {
			towns = append(towns, fmt.Sprintf("\"%s\"", townTileToString(t)))
		}
		sb.WriteString(strings.Join(towns, ", "))
		sb.WriteString("\n")

		// Bonus
		if bonus, ok := gs.BonusCards.GetPlayerCard(pid); ok {
			// Check if used (placeholder logic, assuming unused for now unless tracked)
			used := " (Unused)"
			// TODO: Check actual usage state if available

			// Get card details to print name
			allCards := game.GetAllBonusCards()
			card := allCards[bonus]
			sb.WriteString(fmt.Sprintf("    Bonus: \"%s\"%s\n", bonusCardToString(card), used))
		} else {
			sb.WriteString("    Bonus: \n")
		}

		// Favor
		sb.WriteString("    Favor: ")
		favors := []string{}
		for _, tile := range gs.FavorTiles.GetPlayerTiles(pid) {
			favors = append(favors, favorTileToString(tile))
		}
		sb.WriteString(strings.Join(favors, ", "))
		sb.WriteString("\n")

		// Stronghold Action
		sb.WriteString("    StrongholdAction: ")
		// Check if faction has SH action and if used
		// We need to map faction to its stronghold action type
		shAction := getStrongholdActionType(p.Faction.GetType())
		if shAction != game.SpecialActionType(-1) {
			if p.SpecialActionsUsed[shAction] {
				sb.WriteString("Used")
			} else if p.HasStrongholdAbility { // Assuming HasStrongholdAbility tracks if SH is built/ability unlocked
				sb.WriteString("Available")
			} else {
				sb.WriteString("None")
			}
		} else {
			sb.WriteString("None")
		}
		sb.WriteString("\n\n")
	}

	// Map (Terraformed hexes without buildings or special tokens)
	sb.WriteString("Map:\n")
	// List terraformed hexes
	// We compare current terrain with base terrain
	baseLayout := board.BaseGameTerrainLayout()
	sortedHexes := getSortedHexes(gs.Map)
	for _, h := range sortedHexes {
		mh := gs.Map.GetHex(h)
		baseTerrain, ok := baseLayout[h]
		// If terrain is different from base, OR if it's a base hex but we want to be explicit?
		// Usually snapshots only record changes.
		// Also check if there is NO building (buildings are listed under players)
		if ok && mh.Terrain != baseTerrain && mh.Building == nil {
			sb.WriteString(fmt.Sprintf("  %d,%d: %s\n", h.Q, h.R, mh.Terrain.String()))
		}
	}
	sb.WriteString("\n")

	// State
	sb.WriteString("State:\n")

	// Scoring Tiles
	sb.WriteString("  ScoringTiles: ")
	if gs.ScoringTiles != nil {
		st := []string{}
		for _, tile := range gs.ScoringTiles.Tiles {
			st = append(st, fmt.Sprintf("\"%s\"", scoringTileToString(tile.Type)))
		}
		sb.WriteString(strings.Join(st, ", "))
	}
	sb.WriteString("\n")

	// Bonuses
	sb.WriteString("  Bonuses:\n")
	// List available bonus cards
	// Iterate gs.BonusCards.Available
	allCards := game.GetAllBonusCards()
	// Sort for deterministic output
	availableCards := make([]game.BonusCardType, 0, len(gs.BonusCards.Available))
	for c := range gs.BonusCards.Available {
		availableCards = append(availableCards, c)
	}
	sort.Slice(availableCards, func(i, j int) bool {
		return int(availableCards[i]) < int(availableCards[j])
	})

	for _, c := range availableCards {
		coins := gs.BonusCards.Available[c]
		card := allCards[c]
		sb.WriteString(fmt.Sprintf("    \"%s\": %d\n", bonusCardToString(card), coins))
	}

	// Favors
	sb.WriteString("  Favors:\n")
	// Count available favors
	// We need to iterate through all favor types and check availability
	allFavors := game.GetAllFavorTiles()
	// Sort keys for deterministic output
	favorTypes := make([]int, 0, len(allFavors))
	for t := range allFavors {
		favorTypes = append(favorTypes, int(t))
	}
	sort.Ints(favorTypes)

	for _, tInt := range favorTypes {
		t := game.FavorTileType(tInt)
		// Check availability in gs.FavorTiles.Available
		// We need to access the map directly or use a method if available.
		// GameState struct has FavorTiles *FavorTileState
		// FavorTileState has Available map[FavorTileType]int
		if count, ok := gs.FavorTiles.Available[t]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("    \"%s\": %d\n", favorTileToString(t), count))
		}
	}

	// Towns
	sb.WriteString("  Towns:\n")
	// Count available towns
	// GameState has TownTiles *TownTileState
	// TownTileState has Available map[models.TownTileType]int
	allTowns := []models.TownTileType{
		models.TownTile5Points, models.TownTile6Points, models.TownTile7Points,
		models.TownTile4Points, models.TownTile8Points, models.TownTile9Points,
		models.TownTile11Points, models.TownTile2Points,
	}
	// Sort or iterate in fixed order? The slice above is fixed order.
	for _, t := range allTowns {
		if count, ok := gs.TownTiles.Available[t]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("    \"%s\": %d\n", townTileToString(t), count))
		}
	}

	// PowerActions
	sb.WriteString("  PowerActions:\n")
	// Iterate all power actions
	allPowerActions := []game.PowerActionType{
		game.PowerActionBridge, game.PowerActionPriest, game.PowerActionWorkers,
		game.PowerActionCoins, game.PowerActionSpade1, game.PowerActionSpade2,
	}
	for _, act := range allPowerActions {
		status := "Available"
		if !gs.PowerActions.IsAvailable(act) {
			status = "Used"
		}
		sb.WriteString(fmt.Sprintf("    \"%s\": %s\n", powerActionToString(act), status))
	}

	// CultBoard
	sb.WriteString("  CultBoard:\n")
	// List players occupying priest spots
	// PriestsOnActionSpaces map[string]map[CultTrack]int
	// We need to invert this: Track -> List of Players
	tracks := []game.CultTrack{game.CultFire, game.CultWater, game.CultEarth, game.CultAir}
	for _, track := range tracks {
		playersOnTrack := []string{}
		// Iterate all players to find who has priests on this track
		// Sort player IDs for deterministic output
		pids := make([]string, 0, len(gs.Players))
		for pid := range gs.Players {
			pids = append(pids, pid)
		}
		sort.Strings(pids)

		for _, pid := range pids {
			if counts, ok := gs.CultTracks.PriestsOnActionSpaces[pid]; ok {
				count := counts[track]
				// Add player name 'count' times
				factionName := getFactionName(gs, pid)
				for i := 0; i < count; i++ {
					playersOnTrack = append(playersOnTrack, factionName)
				}
			}
		}

		trackName := cultTrackToString(track)
		sb.WriteString(fmt.Sprintf("    %s: [%s]\n", trackName, strings.Join(playersOnTrack, ", ")))
	}

	return sb.String()
}

// Helpers

func phaseToString(p game.GamePhase) string {
	switch p {
	case game.PhaseSetup:
		return "Setup"
	case game.PhaseFactionSelection:
		return "FactionSelection"
	case game.PhaseIncome:
		return "Income"
	case game.PhaseAction:
		return "Action"
	case game.PhaseCleanup:
		return "Cleanup"
	case game.PhaseEnd:
		return "End"
	default:
		return "Unknown"
	}
}

func getFactionName(gs *game.GameState, pid string) string {
	if p, ok := gs.Players[pid]; ok {
		return p.Faction.GetType().String()
	}
	return pid
}

func buildingTypeToCode(t models.BuildingType) string {
	switch t {
	case models.BuildingDwelling:
		return "D"
	case models.BuildingTradingHouse:
		return "TP"
	case models.BuildingTemple:
		return "TE"
	case models.BuildingSanctuary:
		return "SA"
	case models.BuildingStronghold:
		return "SH"
	default:
		return "?"
	}
}

func getSortedHexes(m *board.TerraMysticaMap) []board.Hex {
	hexes := make([]board.Hex, 0, len(m.Hexes))
	for _, h := range m.Hexes {
		hexes = append(hexes, h.Coord)
	}
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].Q != hexes[j].Q {
			return hexes[i].Q < hexes[j].Q
		}
		return hexes[i].R < hexes[j].R
	})
	return hexes
}

// String converters

func townTileToString(t models.TownTileType) string {
	switch t {
	case models.TownTile5Points:
		return "5 VP, 6 Coins" // TW1
	case models.TownTile6Points:
		return "6 VP, 8 Power" // TW2
	case models.TownTile7Points:
		return "7 VP, 2 Workers" // TW3
	case models.TownTile4Points:
		return "4 VP, Ship" // TW7
	case models.TownTile8Points:
		return "8 VP, Cult" // TW4
	case models.TownTile9Points:
		return "9 VP, Priest" // TW5
	case models.TownTile11Points:
		return "11 VP" // TW6
	case models.TownTile2Points:
		return "2 VP, 2 Cult" // TW8
	default:
		return fmt.Sprintf("Unknown Town %d", t)
	}
}

func bonusCardToString(b game.BonusCard) string {
	// Use the name from the card definition if available, or map type to name
	// The BonusCard struct has a Name field.
	if b.Name != "" {
		return b.Name
	}
	// Fallback if struct is empty (shouldn't happen if loaded correctly)
	return fmt.Sprintf("BON%d", b.Type)
}

func favorTileToString(f game.FavorTileType) string {
	switch f {
	case game.FavorFire3:
		return "Fire +3"
	case game.FavorWater3:
		return "Water +3"
	case game.FavorEarth3:
		return "Earth +3"
	case game.FavorAir3:
		return "Air +3"
	case game.FavorFire2:
		return "Fire +2: Town Power"
	case game.FavorWater2:
		return "Water +2: Cult Advance"
	case game.FavorEarth2:
		return "Earth +2: Worker Income"
	case game.FavorAir2:
		return "Air +2: Power Income"
	case game.FavorFire1:
		return "Fire +1: Coin Income"
	case game.FavorWater1:
		return "Water +1: Trading House VP"
	case game.FavorEarth1:
		return "Earth +1: Dwelling VP"
	case game.FavorAir1:
		return "Air +1: Trading House Pass VP"
	default:
		return fmt.Sprintf("FAV%d", f)
	}
}

func scoringTileToString(s game.ScoringTileType) string {
	switch s {
	case game.ScoringDwellingWater:
		return "Dwelling / Water" // SCORE1
	case game.ScoringDwellingFire:
		return "Dwelling / Fire" // SCORE2
	case game.ScoringTradingHouseWater:
		return "Trading House / Water" // SCORE3
	case game.ScoringTradingHouseAir:
		return "Trading House / Air" // SCORE4
	case game.ScoringTemplePriest:
		return "Temple / Priest" // SCORE5
	case game.ScoringStrongholdFire:
		return "Stronghold / Fire" // SCORE6
	case game.ScoringStrongholdAir:
		return "Stronghold / Air" // SCORE7
	case game.ScoringSpades:
		return "Spades / Earth" // SCORE8
	case game.ScoringTown:
		return "Town / Earth" // SCORE9
	default:
		return fmt.Sprintf("SCORE%d", s)
	}
}

func powerActionToString(p game.PowerActionType) string {
	switch p {
	case game.PowerActionBridge:
		return "Bridge"
	case game.PowerActionPriest:
		return "Priest"
	case game.PowerActionWorkers:
		return "2 Workers"
	case game.PowerActionCoins:
		return "7 Coins"
	case game.PowerActionSpade1:
		return "Spade"
	case game.PowerActionSpade2:
		return "2 Spades"
	default:
		return fmt.Sprintf("ACT%d", p)
	}
}

func cultTrackToString(c game.CultTrack) string {
	switch c {
	case game.CultFire:
		return "Fire"
	case game.CultWater:
		return "Water"
	case game.CultEarth:
		return "Earth"
	case game.CultAir:
		return "Air"
	default:
		return "Unknown"
	}
}

func getStrongholdActionType(f models.FactionType) game.SpecialActionType {
	switch f {
	case models.FactionWitches:
		return game.SpecialActionWitchesRide
	case models.FactionAuren:
		return game.SpecialActionAurenCultAdvance
	case models.FactionSwarmlings:
		return game.SpecialActionSwarmlingsUpgrade
	case models.FactionChaosMagicians:
		return game.SpecialActionChaosMagiciansDoubleTurn
	case models.FactionGiants:
		return game.SpecialActionGiantsTransform
	case models.FactionNomads:
		return game.SpecialActionNomadsSandstorm
	default:
		return game.SpecialActionType(-1) // Unknown/None
	}
}
