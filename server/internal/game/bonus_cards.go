package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Bonus Card System Implementation
//
// There are 9 different bonus cards (depicted as scrolls)
// Main purpose: Additional income in Phase I
// Valid for a single round only (returned after passing)
// Each player selects one bonus card when passing
//
// Coins accumulate on unused bonus cards:
// - At end of each round, add 1 coin to each leftover bonus card
// - When a player takes a card with coins, they get those coins

// BonusCardType represents the type of bonus card
type BonusCardType int

const (
	BonusCardPriest BonusCardType = iota // +1 Priest income
	BonusCardShipping                     // +3 Power income, +1 Shipping for the round
	BonusCardDwellingVP                   // +2 Coins income, VP for Dwellings when passing
	BonusCardWorkerPower                  // +1 Worker, +3 Power income
	BonusCardSpade                        // +2 Coins income, Special action: 1 free spade
	BonusCardTradingHouseVP               // +1 Worker income, VP for Trading Houses when passing
	BonusCard6Coins                       // +6 Coins income
	BonusCardCultAdvance                  // +4 Coins income, Special action: Advance 1 on cult track
	BonusCardStrongholdSanctuary          // +2 Workers income, VP for Stronghold/Sanctuary when passing
	BonusCardShippingVP                   // +3 Power income, VP based on shipping level when passing
)

// BonusCard represents a bonus card with its properties
type BonusCard struct {
	Type        BonusCardType
	Name        string
	Description string
	// Income bonuses
	Coins   int
	Workers int
	Priests int
	Power   int
	// Special abilities
	HasSpecialAction bool
	ShippingBonus    int // Temporary shipping level increase for the round
	// Pass VP bonuses
	PassVPType string // "dwelling", "trading_house", "stronghold_sanctuary", "shipping"
}

// GetAllBonusCards returns all bonus cards with their properties
func GetAllBonusCards() map[BonusCardType]BonusCard {
	return map[BonusCardType]BonusCard{
		BonusCardPriest: {
			Type:             BonusCardPriest,
			Name:             "Priest Income",
			Description:      "Income: +1 Priest",
			Coins:            0,
			Workers:          0,
			Priests:          1,
			Power:            0,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "",
		},
		BonusCardShipping: {
			Type:             BonusCardShipping,
			Name:             "Shipping Bonus",
			Description:      "Income: +3 Power. Shipping +1 for the round (not Dwarves/Fakirs)",
			Coins:            0,
			Workers:          0,
			Priests:          0,
			Power:            3,
			HasSpecialAction: false,
			ShippingBonus:    1,
			PassVPType:       "",
		},
		BonusCardDwellingVP: {
			Type:             BonusCardDwellingVP,
			Name:             "Dwelling VP",
			Description:      "Income: +2 Coins. Pass: +1 VP per Dwelling",
			Coins:            2,
			Workers:          0,
			Priests:          0,
			Power:            0,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "dwelling",
		},
		BonusCardWorkerPower: {
			Type:             BonusCardWorkerPower,
			Name:             "Worker & Power",
			Description:      "Income: +1 Worker, +3 Power",
			Coins:            0,
			Workers:          1,
			Priests:          0,
			Power:            3,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "",
		},
		BonusCardSpade: {
			Type:             BonusCardSpade,
			Name:             "Free Spade",
			Description:      "Income: +2 Coins. Special action: 1 free spade (once per round)",
			Coins:            2,
			Workers:          0,
			Priests:          0,
			Power:            0,
			HasSpecialAction: true,
			ShippingBonus:    0,
			PassVPType:       "",
		},
		BonusCardTradingHouseVP: {
			Type:             BonusCardTradingHouseVP,
			Name:             "Trading House VP",
			Description:      "Income: +1 Worker. Pass: +2 VP per Trading House",
			Coins:            0,
			Workers:          1,
			Priests:          0,
			Power:            0,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "trading_house",
		},
		BonusCard6Coins: {
			Type:             BonusCard6Coins,
			Name:             "6 Coins",
			Description:      "Income: +6 Coins",
			Coins:            6,
			Workers:          0,
			Priests:          0,
			Power:            0,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "",
		},
		BonusCardCultAdvance: {
			Type:             BonusCardCultAdvance,
			Name:             "Cult Advance",
			Description:      "Income: +4 Coins. Special action: Advance 1 on any cult track (once per round)",
			Coins:            4,
			Workers:          0,
			Priests:          0,
			Power:            0,
			HasSpecialAction: true,
			ShippingBonus:    0,
			PassVPType:       "",
		},
		BonusCardStrongholdSanctuary: {
			Type:             BonusCardStrongholdSanctuary,
			Name:             "Stronghold/Sanctuary VP",
			Description:      "Income: +2 Workers. Pass: +4 VP if Stronghold built, +4 VP if Sanctuary built",
			Coins:            0,
			Workers:          2,
			Priests:          0,
			Power:            0,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "stronghold_sanctuary",
		},
		BonusCardShippingVP: {
			Type:             BonusCardShippingVP,
			Name:             "Shipping VP",
			Description:      "Income: +3 Power. Pass: +3 VP per shipping level (not Dwarves/Fakirs)",
			Coins:            0,
			Workers:          0,
			Priests:          0,
			Power:            3,
			HasSpecialAction: false,
			ShippingBonus:    0,
			PassVPType:       "shipping",
		},
	}
}

// BonusCardState tracks available bonus cards and player selections
type BonusCardState struct {
	// Available cards (type -> coins accumulated on the card)
	Available map[BonusCardType]int

	// Player selections (playerID -> bonus card type for this round)
	PlayerCards map[string]BonusCardType

	// Track which players have selected a card this round
	PlayerHasCard map[string]bool
}

// NewBonusCardState creates a new bonus card state
// Note: Cards should be selected via SelectRandomBonusCards during setup
func NewBonusCardState() *BonusCardState {
	return &BonusCardState{
		Available:     make(map[BonusCardType]int),
		PlayerCards:   make(map[string]BonusCardType),
		PlayerHasCard: make(map[string]bool),
	}
}

// SelectRandomBonusCards randomly selects (playerCount + 3) bonus cards for the game
// This should be called during game setup
// Returns the selected card types
func (bcs *BonusCardState) SelectRandomBonusCards(playerCount int) []BonusCardType {
	// Get all 10 bonus cards
	allCards := GetAllBonusCards()
	allCardTypes := make([]BonusCardType, 0, len(allCards))
	for cardType := range allCards {
		allCardTypes = append(allCardTypes, cardType)
	}

	// Randomly shuffle the cards (Fisher-Yates shuffle)
	// Note: In production, use crypto/rand for better randomness
	// For now, using a simple shuffle
	for i := len(allCardTypes) - 1; i > 0; i-- {
		// In a real implementation, use a proper random source
		// For now, this is a placeholder that should be replaced with actual random selection
		j := i // Placeholder - should be: rand.Intn(i + 1)
		allCardTypes[i], allCardTypes[j] = allCardTypes[j], allCardTypes[i]
	}

	// Select the first (playerCount + 3) cards
	numCards := playerCount + 3
	if numCards > len(allCardTypes) {
		numCards = len(allCardTypes)
	}

	selectedCards := allCardTypes[:numCards]

	// Add selected cards to available pool with 0 coins
	for _, cardType := range selectedCards {
		bcs.Available[cardType] = 0
	}

	return selectedCards
}

// SetAvailableBonusCards manually sets which bonus cards are available
// Useful for testing or custom game setup
func (bcs *BonusCardState) SetAvailableBonusCards(cardTypes []BonusCardType) {
	bcs.Available = make(map[BonusCardType]int)
	for _, cardType := range cardTypes {
		bcs.Available[cardType] = 0
	}
}

// InitializePlayer initializes bonus card tracking for a player
func (bcs *BonusCardState) InitializePlayer(playerID string) {
	bcs.PlayerHasCard[playerID] = false
}

// IsAvailable checks if a bonus card is available to take
func (bcs *BonusCardState) IsAvailable(cardType BonusCardType) bool {
	_, exists := bcs.Available[cardType]
	return exists
}

// GetCoinsOnCard returns the number of coins accumulated on a bonus card
func (bcs *BonusCardState) GetCoinsOnCard(cardType BonusCardType) int {
	return bcs.Available[cardType]
}

// TakeBonusCard assigns a bonus card to a player when they pass
// Returns the number of coins that were on the card
func (bcs *BonusCardState) TakeBonusCard(playerID string, cardType BonusCardType) (int, error) {
	// Check if available
	if !bcs.IsAvailable(cardType) {
		return 0, fmt.Errorf("bonus card %v is not available", cardType)
	}

	// Check if player already has a card this round
	if bcs.PlayerHasCard[playerID] {
		return 0, fmt.Errorf("player already has a bonus card this round")
	}

	// Get coins from the card
	coins := bcs.Available[cardType]

	// Remove from available and assign to player
	delete(bcs.Available, cardType)
	bcs.PlayerCards[playerID] = cardType
	bcs.PlayerHasCard[playerID] = true

	return coins, nil
}

// ReturnBonusCard returns a player's bonus card at the end of the round
func (bcs *BonusCardState) ReturnBonusCard(playerID string) {
	if cardType, ok := bcs.PlayerCards[playerID]; ok {
		// Return card to available pool with 0 coins
		bcs.Available[cardType] = 0
		delete(bcs.PlayerCards, playerID)
		bcs.PlayerHasCard[playerID] = false
	}
}

// AddCoinsToLeftoverCards adds 1 coin to each leftover (unselected) bonus card
// Called during cleanup phase
func (bcs *BonusCardState) AddCoinsToLeftoverCards() {
	for cardType := range bcs.Available {
		bcs.Available[cardType]++
	}
}

// GetPlayerCard returns the bonus card a player has this round
func (bcs *BonusCardState) GetPlayerCard(playerID string) (BonusCardType, bool) {
	cardType, ok := bcs.PlayerCards[playerID]
	return cardType, ok
}

// GetBonusCardIncomeBonus returns the income bonus from a player's bonus card
func GetBonusCardIncomeBonus(cardType BonusCardType) (coins int, workers int, priests int, power int) {
	allCards := GetAllBonusCards()
	card, ok := allCards[cardType]
	if !ok {
		return 0, 0, 0, 0
	}

	return card.Coins, card.Workers, card.Priests, card.Power
}

// GetBonusCardShippingBonus returns the shipping bonus from a player's bonus card
// Returns 0 if player's faction doesn't benefit (Dwarves, Fakirs)
func GetBonusCardShippingBonus(cardType BonusCardType, factionType models.FactionType) int {
	allCards := GetAllBonusCards()
	card, ok := allCards[cardType]
	if !ok {
		return 0
	}

	// Dwarves and Fakirs don't benefit from shipping bonus
	if factionType == models.FactionDwarves || factionType == models.FactionFakirs {
		return 0
	}

	return card.ShippingBonus
}

// GetBonusCardPassVP returns VP gained when passing based on the bonus card
func GetBonusCardPassVP(cardType BonusCardType, gs *GameState, playerID string) int {
	allCards := GetAllBonusCards()
	card, ok := allCards[cardType]
	if !ok {
		return 0
	}

	player := gs.GetPlayer(playerID)
	if player == nil {
		return 0
	}

	switch card.PassVPType {
	case "dwelling":
		// Count dwellings on the map
		count := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil &&
				mapHex.Building.PlayerID == playerID &&
				mapHex.Building.Type == models.BuildingDwelling {
				count++
			}
		}
		return count * 1 // 1 VP per dwelling

	case "trading_house":
		// Count trading houses on the map
		count := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil &&
				mapHex.Building.PlayerID == playerID &&
				mapHex.Building.Type == models.BuildingTradingHouse {
				count++
			}
		}
		return count * 2 // 2 VP per trading house

	case "stronghold_sanctuary":
		vp := 0
		// Check for stronghold
		hasStronghold := false
		hasSanctuary := false
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
				if mapHex.Building.Type == models.BuildingStronghold {
					hasStronghold = true
				}
				if mapHex.Building.Type == models.BuildingSanctuary {
					hasSanctuary = true
				}
			}
		}
		if hasStronghold {
			vp += 4
		}
		if hasSanctuary {
			vp += 4
		}
		return vp

	case "shipping":
		// Dwarves and Fakirs don't benefit
		if player.Faction.GetType() == models.FactionDwarves ||
			player.Faction.GetType() == models.FactionFakirs {
			return 0
		}
		return player.ShippingLevel * 3 // 3 VP per shipping level

	default:
		return 0
	}
}

// HasBonusCardSpecialAction checks if a bonus card provides a special action
func HasBonusCardSpecialAction(cardType BonusCardType) bool {
	allCards := GetAllBonusCards()
	card, ok := allCards[cardType]
	if !ok {
		return false
	}
	return card.HasSpecialAction
}
