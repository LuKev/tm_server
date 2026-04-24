package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
	"github.com/lukev/tm_server/internal/replay"
	"gopkg.in/yaml.v3"
)

// GameConfig holds the game setup configuration
type GameConfig struct {
	ScoringTiles             []string                     `yaml:"scoring_tiles"`
	BonusCards               []string                     `yaml:"bonus_cards"`
	FireIceFinalScoringTile  string                       `yaml:"fire_ice_final_scoring_tile"`
	StartingCultChoices      map[string]string            `yaml:"starting_cult_choices"`
	BonusCardSelections      map[string]map[string]string `yaml:"bonus_card_selections"`
	ExtraBonusCardSelections map[string]map[string]string `yaml:"extra_bonus_card_selections"`
	ConspiratorsSwapReturns  map[string][]string          `yaml:"conspirators_swap_returns"`
	AcolytesCultTracks       map[string][]string          `yaml:"acolytes_cult_tracks"`
	RiverBuildHexes          map[string][]string          `yaml:"river_build_hexes"`
}

func main() {
	// Flags
	urlFlag := flag.String("url", "", "BGA table URL or table ID (e.g., 555795328 or https://boardgamearena.com/table?table=555795328)")
	fileFlag := flag.String("file", "", "Path to pre-downloaded log file (text format after HTML parsing)")
	configFlag := flag.String("config", "", "Path to YAML config file with scoring tiles, bonus cards, and selections")
	scoringFlag := flag.String("scoring", "", "Comma-separated scoring tiles (e.g., SCORE1,SCORE2,SCORE3,SCORE4,SCORE5,SCORE6)")
	bonusFlag := flag.String("bonus", "", "Comma-separated bonus cards (e.g., BON-SPD,BON-6C,BON-TP,BON-BB,BON-P,BON-DW,BON-SHIP-VP)")
	helpFlag := flag.Bool("help", false, "Show usage")
	verboseFlag := flag.Bool("v", false, "Verbose output")

	flag.Parse()

	if *helpFlag || (*urlFlag == "" && *fileFlag == "") {
		printUsage()
		os.Exit(0)
	}

	// Load config if provided
	var config *GameConfig
	if *configFlag != "" {
		var err error
		config, err = loadConfig(*configFlag)
		if err != nil {
			fmt.Printf("❌ Failed to load config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📋 Loaded config from %s\n", *configFlag)
	}

	var logContent string
	var err error
	var gameID string

	// Get log content from URL or file
	if *urlFlag != "" {
		gameID = extractTableID(*urlFlag)
		if gameID == "" {
			fmt.Println("❌ Invalid URL or table ID")
			os.Exit(1)
		}
		fmt.Printf("📡 Fetching log for game %s...\n", gameID)
		logContent, err = fetchBGALog(gameID)
		if err != nil {
			fmt.Printf("❌ Failed to fetch log: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Log fetched successfully")
	} else {
		// Read from file
		content, err := os.ReadFile(*fileFlag)
		if err != nil {
			fmt.Printf("❌ Failed to read file: %v\n", err)
			os.Exit(1)
		}
		logContent = string(content)
		gameID = filepath.Base(*fileFlag)
		fmt.Printf("📄 Loaded log from %s\n", *fileFlag)
	}

	// Parse the log
	fmt.Println("🔍 Parsing log...")
	parser := notation.NewBGAParser(logContent)
	items, err := parser.Parse()
	if err != nil {
		fmt.Printf("❌ Failed to parse log: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Parsed %d log items\n", len(items))

	// Inject settings from config or flags
	scoringStr := *scoringFlag
	bonusStr := *bonusFlag
	if config != nil {
		if len(config.ScoringTiles) > 0 {
			scoringStr = strings.Join(config.ScoringTiles, ",")
		}
		if len(config.BonusCards) > 0 {
			bonusStr = strings.Join(config.BonusCards, ",")
		}
	}

	if scoringStr != "" || bonusStr != "" {
		items = injectSettings(items, scoringStr, bonusStr)
	}
	if config != nil && len(config.StartingCultChoices) > 0 {
		items = injectStartingCultChoices(items, config.StartingCultChoices)
	}

	// Inject bonus card selections from config
	if config != nil && (len(config.BonusCardSelections) > 0 || len(config.ExtraBonusCardSelections) > 0) {
		items = injectBonusCardSelections(items, config.BonusCardSelections, config.ExtraBonusCardSelections)
	}
	if config != nil && len(config.ConspiratorsSwapReturns) > 0 {
		items = injectConspiratorsSwapReturns(items, config.ConspiratorsSwapReturns)
	}

	// Create initial game state
	initialState := replay.CreateInitialState(items)
	if config != nil && strings.TrimSpace(config.FireIceFinalScoringTile) != "" {
		if err := applyFireIceFinalScoringTileConfig(initialState, config.FireIceFinalScoringTile); err != nil {
			fmt.Printf("❌ Failed to apply Fire & Ice final scoring tile config: %v\n", err)
			os.Exit(1)
		}
	}
	if config != nil && len(config.AcolytesCultTracks) > 0 {
		if err := applyAcolytesCultTrackConfig(initialState, config.AcolytesCultTracks); err != nil {
			fmt.Printf("❌ Failed to apply Acolytes cult-track config: %v\n", err)
			os.Exit(1)
		}
	}
	if config != nil && len(config.RiverBuildHexes) > 0 {
		if err := applyRiverBuildHexConfig(initialState, config.RiverBuildHexes); err != nil {
			fmt.Printf("❌ Failed to apply replay river-build config: %v\n", err)
			os.Exit(1)
		}
	}

	// Create simulator
	simulator := replay.NewGameSimulator(initialState, items)

	// Run simulation
	fmt.Println("\n🎮 Running simulation...")
	successCount := 0
	totalActions := len(items)

	for simulator.CurrentIndex < totalActions {
		currentItem := items[simulator.CurrentIndex]
		var verbosePlayerID string
		if *verboseFlag {
			fmt.Printf("  Executing action %d/%d: %T\n", simulator.CurrentIndex+1, totalActions, currentItem)
			if actionItem, ok := currentItem.(notation.ActionItem); ok && actionItem.Action != nil {
				fmt.Printf("    Action detail: %#v\n", actionItem.Action)
				playerID := actionItem.Action.GetPlayerID()
				verbosePlayerID = playerID
				if riverAction, ok := actionItem.Action.(*notation.LogRiverBuildAction); ok {
					printRiverBuildDebug(simulator.GetState(), riverAction)
				}
				if player := simulator.GetState().GetPlayer(playerID); player != nil && player.Resources != nil && player.Resources.Power != nil {
					fmt.Printf(
						"    Pre-state %s: VP=%d C=%d W=%d P=%d PW=%d/%d/%d Cult=%d/%d/%d/%d\n",
						playerID,
						player.VictoryPoints,
						player.Resources.Coins,
						player.Resources.Workers,
						player.Resources.Priests,
						player.Resources.Power.Bowl1,
						player.Resources.Power.Bowl2,
						player.Resources.Power.Bowl3,
						player.CultPositions[game.CultFire],
						player.CultPositions[game.CultWater],
						player.CultPositions[game.CultEarth],
						player.CultPositions[game.CultAir],
					)
					fmt.Printf(
						"    Pre-context %s: Tiles=%s Buildings=%s Bonus=%s\n",
						playerID,
						formatFavorTiles(simulator.GetState(), playerID),
						formatBuildingCounts(simulator.GetState(), playerID),
						formatBonusCards(simulator.GetState(), playerID),
					)
				}
				if pendingCult := simulator.GetState().PendingCultRewardSpades; len(pendingCult) > 0 {
					fmt.Printf("    Pending cult spades: %+v\n", pendingCult)
				}
				if pendingLeech := simulator.GetState().PendingLeechOffers; len(pendingLeech) > 0 {
					fmt.Printf("    Pending leech offers:")
					printedAny := false
					for pendingPlayerID, offers := range pendingLeech {
						if len(offers) == 0 {
							continue
						}
						printedAny = true
						fmt.Printf(" %s=[", pendingPlayerID)
						for i, offer := range offers {
							if i > 0 {
								fmt.Print(", ")
							}
							if offer == nil {
								fmt.Print("nil")
								continue
							}
							fmt.Printf("%s:%d(vp=%d,event=%d)", offer.FromPlayerID, offer.Amount, offer.VPCost, offer.EventID)
						}
						fmt.Print("]")
					}
					if !printedAny {
						fmt.Print(" none")
					}
					fmt.Println()
				}
			}
		}

		err := simulator.StepForward()
		if err != nil {
			fmt.Printf("\n❌ Simulation failed at action %d/%d\n", simulator.CurrentIndex, totalActions)
			fmt.Printf("   Error: %v\n", err)

			// Show context
			if simulator.CurrentIndex < len(items) {
				fmt.Printf("   Action: %T\n", items[simulator.CurrentIndex])
				if actionItem, ok := items[simulator.CurrentIndex].(notation.ActionItem); ok {
					actionJSON, _ := json.MarshalIndent(actionItem.Action, "   ", "  ")
					fmt.Printf("   Details: %s\n", actionJSON)
				}
			}

			// Check for missing info
			if missingErr, ok := err.(*game.MissingInfoError); ok {
				fmt.Printf("\n💡 This error indicates missing game setup info:\n")
				fmt.Printf("   Type: %s\n", missingErr.Type)
				fmt.Printf("   Round: %d\n", missingErr.Round)
				fmt.Printf("   Players: %v\n", missingErr.Players)
				fmt.Println("\n   Use -config flag to provide a YAML config file with bonus card selections")
			}

			os.Exit(1)
		}
		if *verboseFlag {
			if verbosePlayerID != "" {
				if player := simulator.GetState().GetPlayer(verbosePlayerID); player != nil && player.Resources != nil && player.Resources.Power != nil {
					fmt.Printf(
						"    Post-state %s: VP=%d C=%d W=%d P=%d PW=%d/%d/%d Cult=%d/%d/%d/%d\n",
						verbosePlayerID,
						player.VictoryPoints,
						player.Resources.Coins,
						player.Resources.Workers,
						player.Resources.Priests,
						player.Resources.Power.Bowl1,
						player.Resources.Power.Bowl2,
						player.Resources.Power.Bowl3,
						player.CultPositions[game.CultFire],
						player.CultPositions[game.CultWater],
						player.CultPositions[game.CultEarth],
						player.CultPositions[game.CultAir],
					)
					fmt.Printf(
						"    Post-context %s: Tiles=%s Buildings=%s Bonus=%s\n",
						verbosePlayerID,
						formatFavorTiles(simulator.GetState(), verbosePlayerID),
						formatBuildingCounts(simulator.GetState(), verbosePlayerID),
						formatBonusCards(simulator.GetState(), verbosePlayerID),
					)
				}
			} else if roundStart, ok := currentItem.(notation.RoundStartItem); ok {
				fmt.Printf(
					"    Round state after round %d start marker: phase=%v incomePending=%t\n",
					roundStart.Round,
					simulator.GetState().Phase,
					simulator.GetState().Phase == game.PhaseIncome,
				)
			}
		}
		successCount++
	}

	fmt.Printf("\n✅ Simulation completed successfully!\n")
	fmt.Printf("   Executed %d actions\n", successCount)

	// Calculate final scoring
	finalScores := simulator.GetState().CalculateFinalScoring()
	rankedScores := game.GetRankedPlayers(finalScores)

	fmt.Printf("\n📊 Final Scores:\n")
	for i, score := range rankedScores {
		fireIcePart := ""
		if score.FireIceVP != 0 || score.FireIceMetricValue != 0 {
			fireIcePart = fmt.Sprintf(", F&I: %d/%d", score.FireIceVP, score.FireIceMetricValue)
		}
		fmt.Printf("   %d. %s: %d VP (Base: %d, Area: %d%s, Cult: %d, Res: %d)\n",
			i+1, score.PlayerName, score.TotalVP, score.BaseVP, score.AreaVP, fireIcePart, score.CultVP, score.ResourceVP)
	}
}

func applyFireIceFinalScoringTileConfig(gs *game.GameState, tileName string) error {
	if gs == nil {
		return fmt.Errorf("game state is nil")
	}
	normalized := strings.ToLower(strings.TrimSpace(tileName))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "", "none", "off":
		gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOff
		gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileNone
	case "distance", "greatest_distance":
		gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOn
		gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileGreatestDistance
	case "stronghold_sanctuary", "sh_sa":
		gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOn
		gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileStrongholdSanctuary
	case "edge", "outposts", "outpost":
		gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOn
		gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileOutposts
	case "cluster", "settlements", "connected_settlements":
		gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOn
		gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileSettlements
	default:
		return fmt.Errorf("unknown Fire & Ice final scoring tile %q", tileName)
	}
	return nil
}

func loadConfig(path string) (*GameConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config GameConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func applyAcolytesCultTrackConfig(gs *game.GameState, configuredTracks map[string][]string) error {
	if gs == nil {
		return fmt.Errorf("game state is nil")
	}
	if gs.ReplayAcolytesCultTracks == nil {
		gs.ReplayAcolytesCultTracks = make(map[string][]game.CultTrack)
	}
	if gs.ReplayAcolytesCultTrackIndex == nil {
		gs.ReplayAcolytesCultTrackIndex = make(map[string]int)
	}

	for playerID, trackCodes := range configuredTracks {
		queue := make([]game.CultTrack, 0, len(trackCodes))
		for _, code := range trackCodes {
			track, err := parseCultTrackCode(code)
			if err != nil {
				return fmt.Errorf("%s: %w", playerID, err)
			}
			queue = append(queue, track)
		}
		gs.ReplayAcolytesCultTracks[playerID] = queue
		gs.ReplayAcolytesCultTrackIndex[playerID] = 0
	}

	return nil
}

func applyRiverBuildHexConfig(gs *game.GameState, configuredHexes map[string][]string) error {
	if gs == nil {
		return fmt.Errorf("game state is nil")
	}
	if gs.ReplayRiverBuildHexes == nil {
		gs.ReplayRiverBuildHexes = make(map[string][]board.Hex)
	}
	if gs.ReplayRiverBuildHexIndex == nil {
		gs.ReplayRiverBuildHexIndex = make(map[string]int)
	}

	for playerID, rawHexes := range configuredHexes {
		queue := make([]board.Hex, 0, len(rawHexes))
		for _, rawHex := range rawHexes {
			hex, err := parseReplayHex(rawHex)
			if err != nil {
				return fmt.Errorf("%s: %w", playerID, err)
			}
			if gs.Map != nil {
				mapHex := gs.Map.GetHex(hex)
				if mapHex == nil || mapHex.Terrain != models.TerrainRiver {
					return fmt.Errorf("%s: replay river-build hex %q is not a river space", playerID, rawHex)
				}
			}
			queue = append(queue, hex)
		}
		gs.ReplayRiverBuildHexes[playerID] = queue
		gs.ReplayRiverBuildHexIndex[playerID] = 0
	}

	return nil
}

func parseCultTrackCode(code string) (game.CultTrack, error) {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "F", "FIRE":
		return game.CultFire, nil
	case "W", "WATER":
		return game.CultWater, nil
	case "E", "EARTH":
		return game.CultEarth, nil
	case "A", "AIR":
		return game.CultAir, nil
	default:
		return game.CultFire, fmt.Errorf("unknown cult track code %q", code)
	}
}

func parseReplayHex(raw string) (board.Hex, error) {
	parts := strings.Split(strings.TrimSpace(raw), "_")
	if len(parts) != 2 {
		return board.Hex{}, fmt.Errorf("invalid replay hex %q (expected Q_R)", raw)
	}
	q, err := strconv.Atoi(parts[0])
	if err != nil {
		return board.Hex{}, fmt.Errorf("invalid replay hex q coordinate %q: %w", raw, err)
	}
	r, err := strconv.Atoi(parts[1])
	if err != nil {
		return board.Hex{}, fmt.Errorf("invalid replay hex r coordinate %q: %w", raw, err)
	}
	return board.NewHex(q, r), nil
}

func printRiverBuildDebug(gs *game.GameState, action *notation.LogRiverBuildAction) {
	if gs == nil || gs.Map == nil || action == nil {
		return
	}

	fmt.Printf("    River build analysis for %s %s\n", action.PlayerID, action.CoordToken)
	if configuredQueue := gs.ReplayRiverBuildHexes[action.PlayerID]; len(configuredQueue) > 0 {
		index := gs.ReplayRiverBuildHexIndex[action.PlayerID]
		if index < len(configuredQueue) {
			fmt.Printf("    Configured replay hex: %s\n", formatHexWithDisplay(gs.Map, configuredQueue[index]))
		}
	}

	token := strings.TrimSpace(action.CoordToken)
	if !strings.HasPrefix(strings.ToUpper(token), "R~") {
		fmt.Printf("    Non-river token; no river analysis needed\n")
		return
	}

	landDisplay := strings.TrimSpace(token[2:])
	landHex, ok := gs.Map.HexForDisplayCoordinate(landDisplay)
	if !ok {
		fmt.Printf("    Could not resolve land anchor %q on map %s\n", landDisplay, gs.Map.ID)
		return
	}

	fmt.Printf("    Land anchor: %s -> %s\n", landDisplay, formatHexWithDisplay(gs.Map, landHex))

	seen := make(map[board.Hex]bool)
	candidates := make([]board.Hex, 0, 6)
	addCandidate := func(hex board.Hex) {
		if seen[hex] {
			return
		}
		seen[hex] = true
		candidates = append(candidates, hex)
	}

	if defaultHex, err := notation.ConvertRiverCoordToAxialForMap(gs.Map.ID, token); err == nil {
		addCandidate(defaultHex)
		fmt.Printf("    Default mapped candidate: %s\n", formatHexWithDisplay(gs.Map, defaultHex))
	}
	if rowCountHex, err := convertRiverCoordByRowCount(gs.Map.ID, token); err == nil {
		if !seen[rowCountHex] {
			fmt.Printf("    Row-count candidate: %s\n", formatHexWithDisplay(gs.Map, rowCountHex))
		}
		addCandidate(rowCountHex)
	}
	for _, neighbor := range landHex.Neighbors() {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain != models.TerrainRiver {
			continue
		}
		addCandidate(neighbor)
	}

	for _, candidate := range candidates {
		clone := gs.CloneForUndo()
		err := game.NewTransformAndBuildAction(action.PlayerID, candidate, true, models.TerrainTypeUnknown).Execute(clone)
		neighbors := describeCandidateNeighbors(gs.Map, candidate)
		if err != nil {
			fmt.Printf("    Candidate %s invalid: %v | neighbors=%s\n", formatHexWithDisplay(gs.Map, candidate), err, neighbors)
			continue
		}
		fmt.Printf(
			"    Candidate %s valid | neighbors=%s | leech=%s\n",
			formatHexWithDisplay(gs.Map, candidate),
			neighbors,
			formatPendingLeechOffers(clone.PendingLeechOffers),
		)
	}
}

func describeCandidateNeighbors(m *board.TerraMysticaMap, riverHex board.Hex) string {
	parts := make([]string, 0, 6)
	for _, neighbor := range m.GetDirectNeighbors(riverHex) {
		mapHex := m.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain == models.TerrainRiver {
			continue
		}
		label := formatHexWithDisplay(m, neighbor)
		if mapHex.Building == nil {
			parts = append(parts, fmt.Sprintf("%s=empty/%s", label, mapHex.Terrain))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s/%s/%s", label, mapHex.Building.PlayerID, mapHex.Building.Type, mapHex.Terrain))
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func formatPendingLeechOffers(offers map[string][]*game.PowerLeechOffer) string {
	if len(offers) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(offers))
	for playerID, playerOffers := range offers {
		if len(playerOffers) == 0 {
			continue
		}
		desc := make([]string, 0, len(playerOffers))
		for _, offer := range playerOffers {
			if offer == nil {
				desc = append(desc, "nil")
				continue
			}
			desc = append(desc, fmt.Sprintf("%s:%d(vp=%d)", offer.FromPlayerID, offer.Amount, offer.VPCost))
		}
		parts = append(parts, fmt.Sprintf("%s=[%s]", playerID, strings.Join(desc, ", ")))
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}

func formatHexWithDisplay(m *board.TerraMysticaMap, hex board.Hex) string {
	if display, ok := m.DisplayCoordinateForHex(hex); ok {
		return fmt.Sprintf("%s/%d_%d", display, hex.Q, hex.R)
	}
	return fmt.Sprintf("%d_%d", hex.Q, hex.R)
}

func convertRiverCoordByRowCount(mapID board.MapID, riverCoord string) (board.Hex, error) {
	riverCoord = strings.ToUpper(strings.TrimSpace(riverCoord))
	if !strings.HasPrefix(riverCoord, "R~") {
		return board.Hex{}, fmt.Errorf("invalid river coordinate format: %s", riverCoord)
	}

	coord := strings.TrimSpace(riverCoord[2:])
	if len(coord) < 2 {
		return board.Hex{}, fmt.Errorf("invalid river coordinate: %s", riverCoord)
	}

	row := int(coord[0] - 'A')
	if row < 0 || row > 25 {
		return board.Hex{}, fmt.Errorf("invalid river row: %q", coord)
	}

	var riverNum int
	if _, err := fmt.Sscanf(coord[1:], "%d", &riverNum); err != nil {
		return board.Hex{}, fmt.Errorf("invalid river number in %s: %w", riverCoord, err)
	}
	if riverNum < 1 {
		return board.Hex{}, fmt.Errorf("river number must be >= 1, got %d", riverNum)
	}

	layout, err := board.LayoutForMap(mapID)
	if err != nil {
		return board.Hex{}, err
	}

	startQ := 0
	foundRow := false
	for candidate := range layout {
		if candidate.R != row {
			continue
		}
		foundRow = true
		if candidate.Q < startQ {
			startQ = candidate.Q
		}
	}
	if !foundRow {
		return board.Hex{}, fmt.Errorf("row %d not found on map %s", row, mapID)
	}

	count := 0
	for q := startQ; ; q++ {
		hex := board.NewHex(q, row)
		terrain, exists := layout[hex]
		if !exists {
			break
		}
		if terrain != models.TerrainRiver {
			continue
		}
		count++
		if count == riverNum {
			return hex, nil
		}
	}

	return board.Hex{}, fmt.Errorf("river %s not found in row-count scan", riverCoord)
}

func printUsage() {
	fmt.Println("BGA Replay Tester - Test if a BGA game can be replayed to completion")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  bga_test -url <table_id_or_url> [options]")
	fmt.Println("  bga_test -file <log_file.txt> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -url string     BGA table URL or table ID")
	fmt.Println("  -file string    Path to pre-downloaded log file")
	fmt.Println("  -config string  Path to YAML config file with setup info")
	fmt.Println("  -scoring string Comma-separated scoring tiles (e.g., SCORE1,SCORE2,...)")
	fmt.Println("  -bonus string   Comma-separated bonus cards (e.g., BON-SPD,BON-6C,...)")
	fmt.Println("  -v              Verbose output")
	fmt.Println("  -help           Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  bga_test -url 555795328")
	fmt.Println("  bga_test -file game.txt -config game_config.yaml")
}

// injectBonusCardSelections updates PassAction bonus cards from config.
// For Archivists after building the stronghold, an extra bonus-card follow-up action
// can be injected immediately after the pass.
func injectBonusCardSelections(items []notation.LogItem, selections map[string]map[string]string, extraSelections map[string]map[string]string) []notation.LogItem {
	// For round 0 (initial bonus cards), we need to find GameSettingsItem
	if round0, ok := selections["0"]; ok {
		for i, item := range items {
			if s, ok := item.(notation.GameSettingsItem); ok {
				for playerID, cardCode := range round0 {
					s.Settings["InitialBonusCard:"+playerID] = cardCode
				}
				items[i] = s
				break
			}
		}
	}

	// For rounds 1-5, update PassAction items
	currentRound := 0
	for i := 0; i < len(items); i++ {
		item := items[i]
		// Track round changes
		if _, ok := item.(notation.RoundStartItem); ok {
			currentRound++
		}

		// Update PassAction bonus cards
		if actionItem, ok := item.(notation.ActionItem); ok {
			if passAction, ok := actionItem.Action.(*game.PassAction); ok {
				roundKey := strconv.Itoa(currentRound)
				if roundSelections, ok := selections[roundKey]; ok {
					if cardCode, ok := roundSelections[passAction.PlayerID]; ok {
						cardType := notation.ParseBonusCardCode(cardCode)
						if cardType != game.BonusCardUnknown {
							passAction.BonusCard = &cardType
							items[i] = notation.ActionItem{Action: passAction}
							fmt.Printf("  Injected %s for %s in round %d\n", cardCode, passAction.PlayerID, currentRound)
						}
					}
				}

				if roundSelections, ok := extraSelections[roundKey]; ok {
					if cardCode, ok := roundSelections[passAction.PlayerID]; ok {
						cardType := notation.ParseBonusCardCode(cardCode)
						if cardType != game.BonusCardUnknown {
							extraAction := notation.ActionItem{
								Action: game.NewSelectArchivistsBonusCardAction(passAction.PlayerID, cardType),
							}
							items = append(items[:i+1], append([]notation.LogItem{extraAction}, items[i+1:]...)...)
							i++
							fmt.Printf("  Injected extra %s for %s in round %d\n", cardCode, passAction.PlayerID, currentRound)
						}
					}
				}
			}
		}
	}

	return items
}

func injectStartingCultChoices(items []notation.LogItem, choices map[string]string) []notation.LogItem {
	if len(choices) == 0 {
		return items
	}

	inserted := make([]notation.LogItem, 0, len(choices))
	playerIDs := make([]string, 0, len(choices))
	for playerID := range choices {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)

	for _, playerID := range playerIDs {
		cultName := choices[playerID]
		cultTrack, ok := parseCultTrackChoice(cultName)
		if !ok {
			fmt.Printf("  Warning: unknown starting cult choice %q for %s\n", cultName, playerID)
			continue
		}
		inserted = append(inserted, notation.ActionItem{
			Action: game.NewSelectDjinniStartingCultTrackAction(playerID, cultTrack),
		})
		fmt.Printf("  Injected Djinni starting cult %s for %s\n", strings.ToUpper(strings.TrimSpace(cultName)), playerID)
	}
	if len(inserted) == 0 {
		return items
	}

	for i, item := range items {
		if _, ok := item.(notation.GameSettingsItem); ok {
			return append(items[:i+1], append(inserted, items[i+1:]...)...)
		}
	}

	return append(inserted, items...)
}

func injectConspiratorsSwapReturns(items []notation.LogItem, returnsByPlayer map[string][]string) []notation.LogItem {
	seenByPlayer := make(map[string]int)
	for i, item := range items {
		actionItem, ok := item.(notation.ActionItem)
		if !ok {
			continue
		}
		swapAction, ok := actionItem.Action.(*notation.LogConspiratorsSwapFavorAction)
		if !ok {
			continue
		}

		returns := returnsByPlayer[swapAction.PlayerID]
		if len(returns) == 0 {
			continue
		}
		idx := seenByPlayer[swapAction.PlayerID]
		seenByPlayer[swapAction.PlayerID] = idx + 1
		if idx >= len(returns) {
			continue
		}

		swapAction.ReturnedTile = returns[idx]
		items[i] = notation.ActionItem{Action: swapAction}
		fmt.Printf("  Injected Conspirators return %s for %s swap #%d\n", returns[idx], swapAction.PlayerID, idx+1)
	}
	return items
}

func parseCultTrackChoice(raw string) (game.CultTrack, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "F", "FIRE":
		return game.CultFire, true
	case "W", "WATER":
		return game.CultWater, true
	case "E", "EARTH":
		return game.CultEarth, true
	case "A", "AIR":
		return game.CultAir, true
	default:
		return game.CultFire, false
	}
}

func extractTableID(input string) string {
	// Check if it's just a number
	if _, err := strconv.Atoi(input); err == nil {
		return input
	}

	// Extract from URL (table=XXXXXX)
	if strings.Contains(input, "table=") {
		parts := strings.Split(input, "table=")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "&")[0]
			if _, err := strconv.Atoi(id); err == nil {
				return id
			}
		}
	}

	return ""
}

func formatFavorTiles(gs *game.GameState, playerID string) string {
	tiles := gs.FavorTiles.GetPlayerTiles(playerID)
	if len(tiles) == 0 {
		return "[]"
	}
	names := make([]string, 0, len(tiles))
	allTiles := game.GetAllFavorTiles()
	for _, tile := range tiles {
		if info, ok := allTiles[tile]; ok {
			names = append(names, info.Name)
			continue
		}
		names = append(names, fmt.Sprintf("Favor(%d)", tile))
	}
	sort.Strings(names)
	return "[" + strings.Join(names, ",") + "]"
}

func formatBonusCards(gs *game.GameState, playerID string) string {
	cards := gs.BonusCards.GetPlayerCards(playerID)
	if len(cards) == 0 {
		return "[]"
	}
	names := make([]string, 0, len(cards))
	allCards := game.GetAllBonusCards()
	for _, card := range cards {
		if info, ok := allCards[card]; ok {
			names = append(names, info.Name)
			continue
		}
		names = append(names, fmt.Sprintf("Bonus(%d)", card))
	}
	sort.Strings(names)
	return "[" + strings.Join(names, ",") + "]"
}

func formatBuildingCounts(gs *game.GameState, playerID string) string {
	counts := map[models.BuildingType]int{}
	for _, mapHex := range gs.Map.Hexes {
		if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
			continue
		}
		counts[mapHex.Building.Type]++
	}
	parts := []string{
		fmt.Sprintf("D=%d", counts[models.BuildingDwelling]),
		fmt.Sprintf("TP=%d", counts[models.BuildingTradingHouse]),
		fmt.Sprintf("TE=%d", counts[models.BuildingTemple]),
		fmt.Sprintf("SA=%d", counts[models.BuildingSanctuary]),
		fmt.Sprintf("SH=%d", counts[models.BuildingStronghold]),
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func createInitialState(items []notation.LogItem) *game.GameState {
	return replay.CreateInitialState(items)
}

func fetchBGALog(tableID string) (string, error) {
	// Get script directory
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	scriptDir := filepath.Join(filepath.Dir(execPath), "..", "..", "scripts")

	// Try relative path from source
	if _, err := os.Stat(filepath.Join(scriptDir, "fetch_bga_log.py")); os.IsNotExist(err) {
		// Try from current working directory
		cwd, _ := os.Getwd()
		scriptDir = filepath.Join(cwd, "scripts")
	}

	scriptPath := filepath.Join(scriptDir, "fetch_bga_log.py")
	outputPath := filepath.Join(scriptDir, fmt.Sprintf("game_%s.txt", tableID))

	// Check if log already exists
	if content, err := os.ReadFile(outputPath); err == nil {
		fmt.Println("  (Using cached log file)")
		return string(content), nil
	}

	// Fetch using Python script
	cmd := exec.Command("python3", scriptPath, tableID, "--output", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("script failed: %w, output: %s", err, output)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}
	return string(content), nil
}

func injectSettings(items []notation.LogItem, scoringStr, bonusStr string) []notation.LogItem {
	// Find existing settings or create new
	var settingsItem *notation.GameSettingsItem
	settingsIndex := -1

	for i, item := range items {
		if s, ok := item.(notation.GameSettingsItem); ok {
			settingsItem = &s
			settingsIndex = i
			break
		}
	}

	if settingsItem == nil {
		settingsItem = &notation.GameSettingsItem{
			Settings: make(map[string]string),
		}
	}

	if scoringStr != "" {
		settingsItem.Settings["ScoringTiles"] = scoringStr
	}
	if bonusStr != "" {
		settingsItem.Settings["BonusCards"] = bonusStr
	}

	if settingsIndex >= 0 {
		items[settingsIndex] = *settingsItem
	} else {
		items = append([]notation.LogItem{*settingsItem}, items...)
	}

	return items
}
