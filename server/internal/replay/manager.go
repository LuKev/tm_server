package replay

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

// ReplayManager handles the lifecycle of replay sessions
type ReplayManager struct {
	mu        sync.RWMutex
	sessions  map[string]*ReplaySession
	scriptDir string
}

// NewReplayManager creates a new ReplayManager
func NewReplayManager(scriptDir string) *ReplayManager {
	return &ReplayManager{
		sessions:  make(map[string]*ReplaySession),
		scriptDir: scriptDir,
	}
}

// ReplaySession represents an active replay session
type ReplaySession struct {
	GameID       string
	Simulator    *GameSimulator
	MissingInfo  *MissingGameInfo
	LogStrings   []string
	LogLocations []notation.LogLocation
}

// MissingGameInfo contains information that couldn't be parsed from the log
type MissingGameInfo struct {
	// Global setup info
	GlobalBonusCards   bool // True if the set of 10 bonus cards is unknown
	GlobalScoringTiles bool // True if the set of 6 scoring tiles is unknown

	// Round-specific info
	// Round 0 = Setup (initial bonus card selection)
	// Round 1-5 = End of round bonus card selection
	// Key: Round Number -> Player ID -> true (missing)
	BonusCardSelections map[int]map[string]bool

	// Player info
	PlayerFactions map[string]bool // Player -> true if faction is unknown/ambiguous
}

// ProvidedGameInfo contains the information provided by the user
type ProvidedGameInfo struct {
	ScoringTiles        []string                     `json:"scoringTiles"`
	BonusCards          []string                     `json:"bonusCards"`
	BonusCardSelections map[string]map[string]string `json:"bonusCardSelections"` // Round -> PlayerID -> BonusCard
}

// StartReplay fetches the log for a game and initializes a session
func (m *ReplayManager) StartReplay(gameID string, restart bool) (*ReplaySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	if session, ok := m.sessions[gameID]; ok {
		if !restart {
			return session, nil
		}
		// If restarting, delete the existing session so it gets recreated
		delete(m.sessions, gameID)
	}

	// Fetch log
	logContent, err := m.fetchLog(gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch log: %w", err)
	}

	// Parse log
	var items []notation.LogItem
	if gameID == "local" {
		items, err = notation.ParseConciseLog(logContent)
	} else {
		parser := notation.NewBGAParser(logContent)
		items, err = parser.Parse()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse log: %w", err)
	}

	// Special case for local testing: Inject hardcoded settings
	if gameID == "local" {
		// Remove any existing GameSettingsItem
		newItems := make([]notation.LogItem, 0)
		for _, item := range items {
			if _, ok := item.(notation.GameSettingsItem); !ok {
				newItems = append(newItems, item)
			}
		}

		// Inject hardcoded settings
		// Scoring Tiles: SCORE5, SCORE8, SCORE4, SCORE1, SCORE6, SCORE7
		// Bonus Cards: All except BON2, BON4, BON5 -> BON1, BON3, BON6, BON7, BON8, BON9, BON10
		settings := notation.GameSettingsItem{
			Settings: map[string]string{
				"ScoringTiles": "SCORE5,SCORE8,SCORE4,SCORE1,SCORE6,SCORE7",
				"BonusCards":   "BON-SPD,BON-6C,BON-TP,BON-BB,BON-P,BON-DW,BON-SHIP-VP",
			},
		}

		// Preserve player mappings if they existed in the original settings (which we removed)
		// Actually, we should check if there was an existing settings item and copy player mappings
		for _, item := range items {
			if s, ok := item.(notation.GameSettingsItem); ok {
				for k, v := range s.Settings {
					if strings.HasPrefix(k, "Player:") || strings.HasPrefix(k, "StartingVP:") {
						settings.Settings[k] = v
					}
				}
			}
		}

		// Prepend settings
		items = append([]notation.LogItem{settings}, newItems...)
	}

	// Create simulator
	initialState := createInitialState(items)

	simulator := NewGameSimulator(initialState, items)

	// Create session
	logStrings, logLocations := notation.GenerateConciseLog(items)

	session := &ReplaySession{
		GameID:       gameID,
		Simulator:    simulator,
		MissingInfo:  detectMissingInfo(items),
		LogStrings:   logStrings,
		LogLocations: logLocations,
	}

	m.sessions[gameID] = session
	return session, nil
}

func (m *ReplayManager) fetchLog(gameID string) (string, error) {
	// Special case for local testing
	if gameID == "local" {
		// Try absolute path to concise_log.txt
		absPath := "/Users/kevin/projects/tm_server/server/internal/notation/concise_log.txt"
		content, err := os.ReadFile(absPath)
		if err != nil {
			// Fallback to relative path
			content, err = os.ReadFile("internal/notation/concise_log.txt")
			if err != nil {
				return "", fmt.Errorf("failed to read local log: %w", err)
			}
		}
		return string(content), nil
	}

	scriptPath := filepath.Join(m.scriptDir, "fetch_bga_log.py")
	outputPath := filepath.Join(m.scriptDir, fmt.Sprintf("game_%s.txt", gameID))

	cmd := exec.Command("python3", scriptPath, gameID, "--output", outputPath)
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

// ImportLog imports a log from raw HTML content
func (m *ReplayManager) ImportLog(gameID string, htmlContent string) error {
	// Parse HTML to text
	logContent, err := notation.ParseBGAHTML(htmlContent)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Save to file
	outputPath := filepath.Join(m.scriptDir, fmt.Sprintf("game_%s.txt", gameID))
	if err := os.WriteFile(outputPath, []byte(logContent), 0644); err != nil {
		return fmt.Errorf("failed to save log file: %w", err)
	}

	// Start replay (force restart)
	_, err = m.StartReplay(gameID, true)
	return err
}

// ProvideInfo updates the session with missing information
func (m *ReplayManager) ProvideInfo(gameID string, info *ProvidedGameInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[gameID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// 1. Update GameSettingsItem
	// Find existing or create new
	var settingsItem *notation.GameSettingsItem
	settingsIndex := -1

	for i, item := range session.Simulator.Actions {
		if s, ok := item.(notation.GameSettingsItem); ok {
			settingsItem = &s
			settingsIndex = i
			break
		}
	}

	if settingsItem == nil {
		// Create new settings item at the beginning
		settingsItem = &notation.GameSettingsItem{
			Settings: make(map[string]string),
		}
		// Insert at index 0
		session.Simulator.Actions = append([]notation.LogItem{*settingsItem}, session.Simulator.Actions...)
		settingsIndex = 0
	}

	// Update settings
	if len(info.ScoringTiles) > 0 {
		// Parse "SCORE1 (Desc)" -> "SCORE1"
		tiles := make([]string, len(info.ScoringTiles))
		for i, t := range info.ScoringTiles {
			parts := strings.Split(t, " ")
			tiles[i] = parts[0]
		}
		settingsItem.Settings["ScoringTiles"] = strings.Join(tiles, ",")
	}

	if len(info.BonusCards) > 0 {
		// Parse "BON1 (Desc)" -> "BON1"
		cards := make([]string, len(info.BonusCards))
		for i, c := range info.BonusCards {
			parts := strings.Split(c, " ")
			cards[i] = parts[0]
		}
		settingsItem.Settings["BonusCards"] = strings.Join(cards, ",")
	}

	// Update the item in the slice
	session.Simulator.Actions[settingsIndex] = *settingsItem

	// 1.5 Inject Bonus Card Selections (Round 0)
	// We need to insert these actions AFTER settings but BEFORE Round 1
	// Find index of Round 1 start
	round1Index := -1
	for i, item := range session.Simulator.Actions {
		if rs, ok := item.(notation.RoundStartItem); ok && rs.Round == 1 {
			round1Index = i
			break
		}
	}

	if round1Index == -1 {
		// If Round 1 not found, append to end? Or just after settings?
		round1Index = settingsIndex + 1
	}

	// Check if Round 0 bonus card selections already exist (e.g., from a previous ProvideInfo call)
	hasRound0BonusCards := false
	for _, item := range session.Simulator.Actions {
		if actionItem, ok := item.(notation.ActionItem); ok {
			if _, ok := actionItem.Action.(*notation.LogBonusCardSelectionAction); ok {
				hasRound0BonusCards = true
				break
			}
		}
	}

	// Only insert Round 0 bonus card selections if they don't already exist
	if !hasRound0BonusCards {
		if roundSelections, ok := info.BonusCardSelections["0"]; ok && len(roundSelections) > 0 {
			newActions := make([]notation.LogItem, 0)
			for playerID, cardStr := range roundSelections {
				// Parse "BON1 (Desc)" -> "BON1"
				parts := strings.Split(cardStr, " ")
				cardCode := parts[0]

				action := &notation.LogBonusCardSelectionAction{
					PlayerID:  playerID,
					BonusCard: cardCode,
				}
				newActions = append(newActions, notation.ActionItem{Action: action})
			}

			// Insert newActions at round1Index
			// We need to be careful with slice manipulation
			// Actions = [0...round1Index-1] + newActions + [round1Index...]
			if round1Index >= len(session.Simulator.Actions) {
				session.Simulator.Actions = append(session.Simulator.Actions, newActions...)
			} else {
				session.Simulator.Actions = append(session.Simulator.Actions[:round1Index], append(newActions, session.Simulator.Actions[round1Index:]...)...)
			}
		}
	}

	// 1.6 Update Pass Actions (Runtime)
	if len(info.BonusCardSelections) > 0 {
		// If we are providing info for a specific round/player (runtime),
		// we might need to update an existing PassAction.
		// Iterate through actions to find the matching PassAction.
		// We don't have exact index, but we can infer from context or just scan.
		// Actually, simpler: if we are at runtime, the "missing info" error came from a specific action.
		// But we don't know which one here easily without passing index.
		// However, we can just scan for PassActions with nil BonusCard and see if we have info for them.

		// Let's track current round in the loop
		currentRound := 0 // Round 0 is setup, then 1-5
		for i, item := range session.Simulator.Actions {
			if rs, ok := item.(notation.RoundStartItem); ok {
				currentRound = rs.Round
			}

			if actionItem, ok := item.(notation.ActionItem); ok {
				if pass, ok := actionItem.Action.(*game.PassAction); ok {
					if pass.BonusCard == nil {
						// Check if we have info for this
						roundStr := fmt.Sprintf("%d", currentRound)
						if roundSelections, ok := info.BonusCardSelections[roundStr]; ok {
							if cardStr, ok := roundSelections[pass.PlayerID]; ok {
								// We found a match! Update it.
								// Parse card code
								parts := strings.Split(cardStr, " ")
								cardCode := parts[0]
								cardType := notation.ParseBonusCardCode(cardCode)

								// We need to update the action in the slice.
								// Since actionItem is a copy (value receiver), we need to update the pointer in the slice if possible.
								// But ActionItem holds Action interface.
								// The underlying *PassAction is a pointer.
								// So modifying 'pass' modifies the underlying struct.
								pass.BonusCard = &cardType
								fmt.Printf("Updated PassAction for %s with card %s at index %d (Round %d)\n", pass.PlayerID, cardCode, i, currentRound)
							}
						}
					}
				}
			}
		}
	}

	// 2. Re-simulate
	// Instead of resetting to 0, we want to re-simulate up to the current point.
	// But since we modified the log (potentially in the past), we MUST reset state.
	// However, we can fast-forward to the previous index.
	targetIndex := session.Simulator.CurrentIndex
	session.Simulator.CurrentIndex = 0
	session.Simulator.CurrentState = createInitialState(session.Simulator.Actions)
	session.Simulator.History = make([]*game.GameState, 0)

	// Re-detect missing info (global only)
	session.MissingInfo = detectMissingInfo(session.Simulator.Actions)

	// Regenerate concise log strings
	logStrings, logLocations := notation.GenerateConciseLog(session.Simulator.Actions)
	session.LogStrings = logStrings
	session.LogLocations = logLocations

	// Fast-forward
	// We loop until we reach targetIndex OR we hit an error (e.g. another missing info)
	for session.Simulator.CurrentIndex < targetIndex {
		if err := session.Simulator.StepForward(); err != nil {
			// If we hit an error during fast-forward, stop there.
			// This is expected if we hit another missing info.
			fmt.Printf("Fast-forward stopped at %d: %v\n", session.Simulator.CurrentIndex, err)
			break
		}
	}

	return nil
}

// GetSession returns an active replay session
func (m *ReplayManager) GetSession(gameID string) *ReplaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[gameID]
}

// JumpTo moves the replay to the specified index
func (m *ReplayManager) JumpTo(gameID string, targetIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[gameID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// If jumping backwards, we need to reset the simulator
	if targetIndex < session.Simulator.CurrentIndex {
		fmt.Printf("JumpTo: Backwards jump detected (%d -> %d). Resetting simulator.\n", session.Simulator.CurrentIndex, targetIndex)
		initialState := createInitialState(session.Simulator.Actions)
		session.Simulator = NewGameSimulator(initialState, session.Simulator.Actions)
	}

	// Fast forward to target
	return session.Simulator.JumpTo(targetIndex)
}

func createInitialState(items []notation.LogItem) *game.GameState {
	initialState := game.NewGameState()

	// Pre-populate players and settings from GameSettingsItem if present
	for _, item := range items {
		if s, ok := item.(notation.GameSettingsItem); ok {
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
						// Parse "BON1 (Desc)" -> "BON1"
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
						// Parse "SCORE1 (Desc)" -> "SCORE1"
						parts := strings.Split(tileCode, " ")
						code := parts[0]
						tile, err := parseScoringTile(code)
						if err != nil {
							fmt.Printf("Warning: failed to parse scoring tile %s: %v\n", code, err)
							continue
						}
						// Ensure we don't add more than 6
						if i < 6 {
							initialState.ScoringTiles.Tiles = append(initialState.ScoringTiles.Tiles, tile)
						}
					}
				}
			}
			break // Only need the first settings item
		}
	}
	// Debug logging
	fmt.Printf("DEBUG: createInitialState - BonusCards Available: %d\n", len(initialState.BonusCards.Available))
	for k, v := range initialState.BonusCards.Available {
		fmt.Printf("  Card: %v, Coins: %d\n", k, v)
	}

	return initialState
}

func detectMissingInfo(items []notation.LogItem) *MissingGameInfo {
	missing := &MissingGameInfo{
		BonusCardSelections: make(map[int]map[string]bool),
		PlayerFactions:      make(map[string]bool),
	}
	hasMissingInfo := false

	fmt.Println("DEBUG: Starting detectMissingInfo")

	// 1. Check Game Settings
	settingsFound := false
	var players []string
	for _, item := range items {
		if s, ok := item.(notation.GameSettingsItem); ok {
			settingsFound = true
			if len(s.Settings["BonusCards"]) == 0 {
				missing.GlobalBonusCards = true
				hasMissingInfo = true
			}
			if len(s.Settings["ScoringTiles"]) == 0 {
				missing.GlobalScoringTiles = true
				hasMissingInfo = true
			}
			// Extract players to check Round 0 bonus cards
			for k, v := range s.Settings {
				if strings.HasPrefix(k, "Player:") {
					players = append(players, v) // v is faction/playerID
				}
			}
			break
		}
	}
	if !settingsFound {
		missing.GlobalBonusCards = true
		missing.GlobalScoringTiles = true
		hasMissingInfo = true
	}

	// 2. Check Round 0 Bonus Card Selections - MOVED TO RUNTIME CHECK
	// We do not block start on this anymore.

	// 3. Scan for missing bonus card selections in Pass actions - MOVED TO RUNTIME CHECK
	// We do not block start on this anymore.

	if !hasMissingInfo {
		fmt.Println("DEBUG: No global missing info detected")
		return nil
	}

	fmt.Printf("DEBUG: Global Missing Info Detected: GlobalBonusCards=%v, GlobalScoringTiles=%v\n",
		missing.GlobalBonusCards, missing.GlobalScoringTiles)
	return missing
}
