package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
	"github.com/lukev/tm_server/internal/replay"
	"gopkg.in/yaml.v3"
)

// GameConfig holds the game setup configuration
type GameConfig struct {
	ScoringTiles        []string                     `yaml:"scoring_tiles"`
	BonusCards          []string                     `yaml:"bonus_cards"`
	BonusCardSelections map[string]map[string]string `yaml:"bonus_card_selections"`
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

	// Inject bonus card selections from config
	if config != nil && len(config.BonusCardSelections) > 0 {
		items = injectBonusCardSelections(items, config.BonusCardSelections)
	}

	// Create initial game state
	initialState := createInitialState(items)

	// Create simulator
	simulator := replay.NewGameSimulator(initialState, items)

	// Run simulation
	fmt.Println("\n🎮 Running simulation...")
	successCount := 0
	totalActions := len(items)

	for simulator.CurrentIndex < totalActions {
		if *verboseFlag {
			fmt.Printf("  Executing action %d/%d: %T\n", simulator.CurrentIndex+1, totalActions, items[simulator.CurrentIndex])
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
		successCount++
	}

	fmt.Printf("\n✅ Simulation completed successfully!\n")
	fmt.Printf("   Executed %d actions\n", successCount)

	// Print final scores
	fmt.Println("\n📊 Final Scores:")
	for playerID, player := range simulator.GetState().Players {
		fmt.Printf("   %s: %d VP\n", playerID, player.VictoryPoints)
	}
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

// injectBonusCardSelections updates PassAction bonus cards from config
func injectBonusCardSelections(items []notation.LogItem, selections map[string]map[string]string) []notation.LogItem {
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
	for i, item := range items {
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
			}
		}
	}

	return items
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

func createInitialState(items []notation.LogItem) *game.GameState {
	initialState := game.NewGameState()

	// Pre-populate players and settings from GameSettingsItem if present
	for _, item := range items {
		if s, ok := item.(notation.GameSettingsItem); ok {
			// First pass: create players
			for k, v := range s.Settings {
				if strings.HasPrefix(k, "Player:") {
					factionName := v
					factionType := models.FactionTypeFromString(factionName)
					faction := factions.NewFaction(factionType)
					initialState.AddPlayer(factionName, faction)

					// Set player name
					playerName := strings.TrimPrefix(k, "Player:")
					if p, exists := initialState.Players[factionName]; exists {
						p.Name = playerName
					}

					// Set starting VPs if specified
					if vpStr, ok := s.Settings["StartingVP:"+factionName]; ok {
						if vp, err := strconv.Atoi(vpStr); err == nil {
							if p, exists := initialState.Players[factionName]; exists {
								p.VictoryPoints = vp
							}
						}
					}
				} else if k == "BonusCards" {
					// Parse bonus cards
					cards := strings.Split(v, ",")
					availableCards := make([]game.BonusCardType, 0)
					for _, cardCode := range cards {
						parts := strings.Split(cardCode, " ")
						code := parts[0]
						cardType := notation.ParseBonusCardCode(code)
						if cardType != game.BonusCardUnknown {
							availableCards = append(availableCards, cardType)
						}
					}
					initialState.BonusCards.SetAvailableBonusCards(availableCards)
				} else if k == "ScoringTiles" {
					// Parse scoring tiles
					tiles := strings.Split(v, ",")
					initialState.ScoringTiles = game.NewScoringTileState()
					for i, tileCode := range tiles {
						parts := strings.Split(tileCode, " ")
						code := parts[0]
						tile, err := parseScoringTile(code)
						if err != nil {
							fmt.Printf("Warning: failed to parse scoring tile %s: %v\n", code, err)
							continue
						}
						if i < 6 {
							initialState.ScoringTiles.Tiles = append(initialState.ScoringTiles.Tiles, tile)
						}
					}
				}
			}

			// Second pass: assign initial bonus cards (must happen after players and bonus cards are set up)
			for k, v := range s.Settings {
				if strings.HasPrefix(k, "InitialBonusCard:") {
					playerID := strings.TrimPrefix(k, "InitialBonusCard:")
					cardType := notation.ParseBonusCardCode(v)
					if cardType != game.BonusCardUnknown {
						// Assign the card to the player
						_, err := initialState.BonusCards.TakeBonusCard(playerID, cardType)
						if err != nil {
							fmt.Printf("Warning: failed to assign initial bonus card %s to %s: %v\n", v, playerID, err)
						} else {
							fmt.Printf("  Assigned initial bonus card %s to %s\n", v, playerID)
						}
					}
				}
			}
			break
		}
	}

	return initialState
}

func parseScoringTile(code string) (game.ScoringTile, error) {
	allTiles := game.GetAllScoringTiles()

	// Map of scoring codes to tile types
	scoreMap := map[string]game.ScoringTileType{
		"SCORE1": game.ScoringSpades,
		"SCORE2": game.ScoringTown,
		"SCORE3": game.ScoringDwellingWater,
		"SCORE4": game.ScoringStrongholdFire,
		"SCORE5": game.ScoringDwellingFire,
		"SCORE6": game.ScoringTradingHouseWater,
		"SCORE7": game.ScoringStrongholdAir,
		"SCORE8": game.ScoringTradingHouseAir,
		"SCORE9": game.ScoringTemplePriest,
	}

	tileType, ok := scoreMap[code]
	if !ok {
		return game.ScoringTile{}, fmt.Errorf("unknown scoring tile code: %s", code)
	}

	// Find the matching tile from all tiles
	for _, tile := range allTiles {
		if tile.Type == tileType {
			return tile, nil
		}
	}

	return game.ScoringTile{}, fmt.Errorf("scoring tile not found: %s", code)
}
