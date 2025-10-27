package replay

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// SetupGame initializes the game state from log entries
func (v *GameValidator) SetupGame() error {
	// Extract game setup information from the log
	setupInfo, err := v.extractSetupInfo()
	if err != nil {
		return fmt.Errorf("failed to extract setup info: %v", err)
	}

	// Create game state
	v.GameState = game.NewGameState()

	// Add players with their factions
	for _, factionType := range setupInfo.Factions {
		// Use faction name as player ID for consistent mapping
		playerID := factionType.String()

		// Create faction instance
		faction, err := createFaction(factionType)
		if err != nil {
			return fmt.Errorf("failed to create faction %v: %v", factionType, err)
		}

		// Create player
		player := &game.Player{
			ID:                   playerID,
			Faction:              faction,
			Resources:            game.NewResourcePool(faction.GetStartingResources()),
			ShippingLevel:        0, // Default starting shipping
			DiggingLevel:         0, // Default starting digging
			CultPositions:        make(map[game.CultTrack]int),
			SpecialActionsUsed:   make(map[game.SpecialActionType]bool),
			HasPassed:            false,
			VictoryPoints:        20, // Starting VP
			Keys:                 0,
			TownsFormed:          0,
			TownTiles:            make([]game.TownTileType, 0),
			HasStrongholdAbility: false,
		}

		// Initialize cult positions from faction
		cultPositions := faction.GetStartingCultPositions()
		player.CultPositions[game.CultFire] = cultPositions.Fire
		player.CultPositions[game.CultWater] = cultPositions.Water
		player.CultPositions[game.CultEarth] = cultPositions.Earth
		player.CultPositions[game.CultAir] = cultPositions.Air

		v.GameState.Players[playerID] = player
		v.GameState.TurnOrder = append(v.GameState.TurnOrder, playerID)
	}

	// Set up scoring tiles
	if err := v.setupScoringTiles(setupInfo); err != nil {
		return fmt.Errorf("failed to setup scoring tiles: %v", err)
	}

	// Set up bonus cards
	if err := v.setupBonusCards(setupInfo); err != nil {
		return fmt.Errorf("failed to setup bonus cards: %v", err)
	}

	v.GameState.Phase = game.PhaseSetup
	v.GameState.Round = 0

	return nil
}

// GameSetupInfo contains information extracted from the game log
type GameSetupInfo struct {
	Factions        []models.FactionType
	ScoringTiles    map[int]string          // Round -> Scoring tile
	RemovedBonuses  []string                // Removed bonus tiles
	BonusCards      []game.BonusCardType    // Bonus cards used in the game
	PlayerNames     map[int]string          // Player number -> name
}

func (v *GameValidator) extractSetupInfo() (*GameSetupInfo, error) {
	info := &GameSetupInfo{
		Factions:       make([]models.FactionType, 0),
		ScoringTiles:   make(map[int]string),
		RemovedBonuses: make([]string, 0),
		PlayerNames:    make(map[int]string),
	}

	// Look through early entries to find setup information
	for _, entry := range v.LogEntries {
		if !entry.IsComment {
			continue
		}

		text := entry.CommentText

		// Parse scoring tiles
		// Format: "Round 1 scoring: SCORE2, TOWN >> 5"
		if len(text) > 6 && text[:5] == "Round" {
			var round int
			var score string
			_, err := fmt.Sscanf(text, "Round %d scoring: %s", &round, &score)
			if err == nil {
				info.ScoringTiles[round] = score
			}
		}

		// Parse removed bonus tiles
		// Format: "Removing tile BON9"
		if len(text) > 13 && text[:13] == "Removing tile" {
			var tile string
			_, err := fmt.Sscanf(text, "Removing tile %s", &tile)
			if err == nil {
				info.RemovedBonuses = append(info.RemovedBonuses, tile)
			}
		}

		// Parse player names
		// Format: "Player 1: GeorgeShortwell"
		if len(text) > 7 && text[:6] == "Player" {
			var playerNum int
			var playerName string
			_, err := fmt.Sscanf(text, "Player %d: %s", &playerNum, &playerName)
			if err == nil {
				info.PlayerNames[playerNum] = playerName
			}
		}
	}

	// Extract factions from first non-comment entries with "setup" action
	for _, entry := range v.LogEntries {
		if entry.IsComment || entry.Faction == 0 {
			continue
		}

		if entry.Action == "setup" {
			info.Factions = append(info.Factions, entry.Faction)
		}

		// Stop after we've seen all setup entries
		if len(info.Factions) > 0 && entry.Action != "setup" && entry.Action != "" {
			break
		}
	}

	if len(info.Factions) == 0 {
		return nil, fmt.Errorf("no factions found in log")
	}

	return info, nil
}

func (v *GameValidator) setupScoringTiles(info *GameSetupInfo) error {
	// TODO: Parse and set scoring tiles
	// For now, just use defaults
	return nil
}

func (v *GameValidator) setupBonusCards(info *GameSetupInfo) error {
	// All 10 bonus cards in BON1-BON10 order
	allBonusCards := []game.BonusCardType{
		game.BonusCardSpade,              // BON1 - Spade special action
		game.BonusCardCultAdvance,        // BON2 - +4 C, cult advance action
		game.BonusCard6Coins,             // BON3 - 6 coins income
		game.BonusCardShipping,           // BON4 - +3 PW, 1 ship for round
		game.BonusCardWorkerPower,        // BON5 - +1 W, +3 PW
		game.BonusCardStrongholdSanctuary, // BON6 - +2 W, pass-vp:SH/SA
		game.BonusCardTradingHouseVP,     // BON7 - +1 W, pass-vp:TP
		game.BonusCardPriest,             // BON8 - +1 P income
		game.BonusCardDwellingVP,         // BON9 - +2 C, pass-vp:D
		game.BonusCardShippingVP,         // BON10 - +3 PW, pass-vp:shipping level
	}

	// If no bonus cards were removed, use all
	if len(info.RemovedBonuses) == 0 {
		v.GameState.BonusCards.SetAvailableBonusCards(allBonusCards)
		return nil
	}

	// Parse removed bonus card strings to types
	removedTypes := make(map[game.BonusCardType]bool)
	for _, bonusStr := range info.RemovedBonuses {
		bonusType, err := ParseBonusCard(bonusStr)
		if err != nil {
			return fmt.Errorf("invalid removed bonus card %s: %v", bonusStr, err)
		}
		removedTypes[bonusType] = true
	}

	// Filter out removed bonus cards
	availableCards := make([]game.BonusCardType, 0)
	for _, card := range allBonusCards {
		if !removedTypes[card] {
			availableCards = append(availableCards, card)
		}
	}

	// Set available bonus cards in game state
	v.GameState.BonusCards.SetAvailableBonusCards(availableCards)
	return nil
}

// applySetupState applies the initial state from setup log entries
func (v *GameValidator) applySetupState(info *GameSetupInfo) error {
	// Find setup entries in the log and apply their state
	for _, entry := range v.LogEntries {
		if entry.Action != "setup" {
			continue
		}

		// Find the player for this faction
		playerID := entry.Faction.String()
		player, ok := v.GameState.Players[playerID]
		if !ok {
			return fmt.Errorf("player not found for faction %v during setup", entry.Faction)
		}

		// Apply cult positions from the setup entry
		player.CultPositions[game.CultFire] = entry.CultTracks.Fire
		player.CultPositions[game.CultWater] = entry.CultTracks.Water
		player.CultPositions[game.CultEarth] = entry.CultTracks.Earth
		player.CultPositions[game.CultAir] = entry.CultTracks.Air
	}

	return nil
}

func createFaction(factionType models.FactionType) (factions.Faction, error) {
	switch factionType {
	case models.FactionWitches:
		return factions.NewWitches(), nil
	case models.FactionEngineers:
		return factions.NewEngineers(), nil
	case models.FactionDarklings:
		return factions.NewDarklings(), nil
	case models.FactionCultists:
		return factions.NewCultists(), nil
	case models.FactionNomads:
		return factions.NewNomads(), nil
	case models.FactionHalflings:
		return factions.NewHalflings(), nil
	case models.FactionAlchemists:
		return factions.NewAlchemists(), nil
	case models.FactionMermaids:
		return factions.NewMermaids(), nil
	case models.FactionSwarmlings:
		return factions.NewSwarmlings(), nil
	case models.FactionChaosMagicians:
		return factions.NewChaosMagicians(), nil
	case models.FactionGiants:
		return factions.NewGiants(), nil
	case models.FactionFakirs:
		return factions.NewFakirs(), nil
	case models.FactionDwarves:
		return factions.NewDwarves(), nil
	case models.FactionAuren:
		return factions.NewAuren(), nil
	default:
		return nil, fmt.Errorf("unknown faction type: %v", factionType)
	}
}
