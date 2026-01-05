package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Favor Tile System Implementation
//
// Players gain favor tiles when building Temples or Sanctuaries
// Each favor tile provides:
// 1. Immediate cult track advancement (1, 2, or 3 spaces)
// 2. Optional ongoing ability or bonus
//
// There are 12 different favor tiles:
// - 4 tiles with +3 cult advancement only (one per cult track)
// - 8 tiles with +1 or +2 cult advancement plus special abilities (3 of each type)

// FavorTileType represents the type of favor tile
type FavorTileType int

const (
	// +3 Cult advancement only (no special ability)
	FavorFire3  FavorTileType = iota // Fire +3
	FavorWater3                      // Water +3
	FavorEarth3                      // Earth +3
	FavorAir3                        // Air +3

	// +2 Cult advancement with special ability
	FavorFire2  // Fire +2: Town power requirement reduced to 6
	FavorWater2 // Water +2: Special action to advance 1 on any cult track
	FavorEarth2 // Earth +2: Income +1 worker, +1 power
	FavorAir2   // Air +2: Income +4 power

	// +1 Cult advancement with special ability
	FavorFire1                     // Fire +1: Income +3 coins
	FavorWater1                    // Water +1: +3 VP when upgrading Dwelling→Trading House
	FavorEarth1                    // Earth +1: +2 VP when building Dwelling
	FavorAir1                      // Air +1: VP at pass based on Trading Houses (2/3/3/4 for 1/2/3/4 TH)
	FavorTileUnknown FavorTileType = -1
)

func FavorTileTypeFromString(s string) FavorTileType {
	switch s {
	case "Fire +3":
		return FavorFire3
	case "Water +3":
		return FavorWater3
	case "Earth +3":
		return FavorEarth3
	case "Air +3":
		return FavorAir3
	case "Fire +2: Town Power":
		return FavorFire2
	case "Water +2: Cult Advance":
		return FavorWater2
	case "Earth +2: Worker Income":
		return FavorEarth2
	case "Air +2: Power Income":
		return FavorAir2
	case "Fire +1: Coin Income":
		return FavorFire1
	case "Water +1: Trading House VP":
		return FavorWater1
	case "Earth +1: Dwelling VP":
		return FavorEarth1
	case "Air +1: Trading House Pass VP":
		return FavorAir1
	default:
		return FavorTileUnknown
	}
}

// FavorTile represents a favor tile with its properties
type FavorTile struct {
	Type         FavorTileType
	CultTrack    CultTrack
	CultAdvance  int    // How many spaces to advance on the cult track
	Name         string // Human-readable name
	Description  string // Description of the special ability
	HasAbility   bool   // Whether this tile has an ongoing ability
	AvailableQty int    // How many of this tile are available (1 for +3, 3 for others)
}

// GetAllFavorTiles returns all favor tiles with their properties
func GetAllFavorTiles() map[FavorTileType]FavorTile {
	return map[FavorTileType]FavorTile{
		// +3 Cult advancement only (1 of each)
		FavorFire3: {
			Type:         FavorFire3,
			CultTrack:    CultFire,
			CultAdvance:  3,
			Name:         "Fire +3",
			Description:  "Advance 3 spaces on Fire cult track",
			HasAbility:   false,
			AvailableQty: 1,
		},
		FavorWater3: {
			Type:         FavorWater3,
			CultTrack:    CultWater,
			CultAdvance:  3,
			Name:         "Water +3",
			Description:  "Advance 3 spaces on Water cult track",
			HasAbility:   false,
			AvailableQty: 1,
		},
		FavorEarth3: {
			Type:         FavorEarth3,
			CultTrack:    CultEarth,
			CultAdvance:  3,
			Name:         "Earth +3",
			Description:  "Advance 3 spaces on Earth cult track",
			HasAbility:   false,
			AvailableQty: 1,
		},
		FavorAir3: {
			Type:         FavorAir3,
			CultTrack:    CultAir,
			CultAdvance:  3,
			Name:         "Air +3",
			Description:  "Advance 3 spaces on Air cult track",
			HasAbility:   false,
			AvailableQty: 1,
		},

		// +2 Cult advancement with special ability (3 of each)
		FavorFire2: {
			Type:         FavorFire2,
			CultTrack:    CultFire,
			CultAdvance:  2,
			Name:         "Fire +2: Town Power",
			Description:  "Town power requirement reduced to 6 (from 7)",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorWater2: {
			Type:         FavorWater2,
			CultTrack:    CultWater,
			CultAdvance:  2,
			Name:         "Water +2: Cult Advance",
			Description:  "Special action: Advance 1 space on any cult track (once per round)",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorEarth2: {
			Type:         FavorEarth2,
			CultTrack:    CultEarth,
			CultAdvance:  2,
			Name:         "Earth +2: Worker Income",
			Description:  "Income: +1 worker, +1 power",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorAir2: {
			Type:         FavorAir2,
			CultTrack:    CultAir,
			CultAdvance:  2,
			Name:         "Air +2: Power Income",
			Description:  "Income: +4 power",
			HasAbility:   true,
			AvailableQty: 3,
		},

		// +1 Cult advancement with special ability (3 of each)
		FavorFire1: {
			Type:         FavorFire1,
			CultTrack:    CultFire,
			CultAdvance:  1,
			Name:         "Fire +1: Coin Income",
			Description:  "Income: +3 coins",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorWater1: {
			Type:         FavorWater1,
			CultTrack:    CultWater,
			CultAdvance:  1,
			Name:         "Water +1: Trading House VP",
			Description:  "+3 VP when upgrading Dwelling to Trading House",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorEarth1: {
			Type:         FavorEarth1,
			CultTrack:    CultEarth,
			CultAdvance:  1,
			Name:         "Earth +1: Dwelling VP",
			Description:  "+2 VP when building Dwelling",
			HasAbility:   true,
			AvailableQty: 3,
		},
		FavorAir1: {
			Type:         FavorAir1,
			CultTrack:    CultAir,
			CultAdvance:  1,
			Name:         "Air +1: Trading House Pass VP",
			Description:  "VP when passing: 2/3/3/4 for 1/2/3/4 Trading Houses",
			HasAbility:   true,
			AvailableQty: 3,
		},
	}
}

// FavorTileState tracks available favor tiles and which players have which tiles
type FavorTileState struct {
	// Available tiles (type -> remaining quantity)
	Available map[FavorTileType]int `json:"available"`

	// Player tiles (playerID -> list of favor tiles)
	PlayerTiles map[string][]FavorTileType `json:"playerTiles"`
}

// NewFavorTileState creates a new favor tile state with all tiles available
func NewFavorTileState() *FavorTileState {
	available := make(map[FavorTileType]int)
	allTiles := GetAllFavorTiles()

	for tileType, tile := range allTiles {
		available[tileType] = tile.AvailableQty
	}

	return &FavorTileState{
		Available:   available,
		PlayerTiles: make(map[string][]FavorTileType),
	}
}

// InitializePlayer initializes favor tiles for a player
func (fts *FavorTileState) InitializePlayer(playerID string) {
	fts.PlayerTiles[playerID] = []FavorTileType{}
}

// IsAvailable checks if a favor tile is available to take
func (fts *FavorTileState) IsAvailable(tileType FavorTileType) bool {
	return fts.Available[tileType] > 0
}

// HasTileType checks if a player already has a tile of this type
// (Players can only have one tile of each type)
func (fts *FavorTileState) HasTileType(playerID string, tileType FavorTileType) bool {
	tiles, ok := fts.PlayerTiles[playerID]
	if !ok {
		return false
	}

	for _, t := range tiles {
		if t == tileType {
			return true
		}
	}
	return false
}

// TakeFavorTile gives a favor tile to a player
// Returns error if tile is not available or player already has this type
func (fts *FavorTileState) TakeFavorTile(playerID string, tileType FavorTileType) error {
	// Check if available
	if !fts.IsAvailable(tileType) {
		return fmt.Errorf("favor tile %v is not available", tileType)
	}

	// Check if player already has this type
	if fts.HasTileType(playerID, tileType) {
		return fmt.Errorf("player already has favor tile type %v", tileType)
	}

	// Take the tile
	fts.Available[tileType]--
	fts.PlayerTiles[playerID] = append(fts.PlayerTiles[playerID], tileType)

	return nil
}

// GetPlayerTiles returns all favor tiles a player has
func (fts *FavorTileState) GetPlayerTiles(playerID string) []FavorTileType {
	return fts.PlayerTiles[playerID]
}

// ApplyFavorTileImmediate applies the immediate effects of taking a favor tile
// (cult track advancement and one-time bonuses)
func ApplyFavorTileImmediate(gs *GameState, playerID string, tileType FavorTileType) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	allTiles := GetAllFavorTiles()
	tile, ok := allTiles[tileType]
	if !ok {
		return fmt.Errorf("invalid favor tile type: %v", tileType)
	}

	// Apply cult track advancement
	if tile.CultAdvance > 0 {
		// Use AdvanceCultTrack (not CultTracks.AdvancePlayer) to properly sync player.CultPositions
		gs.AdvanceCultTrack(playerID, tile.CultTrack, tile.CultAdvance)
	}

	// Note: Ongoing abilities are applied during income phase or relevant actions
	// No additional immediate effects for these tiles

	return nil
}

// GetFavorTileIncomeBonus returns the income bonus from a player's favor tiles
func GetFavorTileIncomeBonus(playerTiles []FavorTileType) (coins int, workers int, power int) {
	for _, tileType := range playerTiles {
		switch tileType {
		case FavorFire1:
			coins += 3
		case FavorEarth2:
			workers += 1
			power += 1
		case FavorAir2:
			power += 4
		}
	}
	return coins, workers, power
}

// HasFavorTile checks if a player has a specific favor tile
func HasFavorTile(playerTiles []FavorTileType, tileType FavorTileType) bool {
	for _, t := range playerTiles {
		if t == tileType {
			return true
		}
	}
	return false
}

// GetTownPowerRequirement returns the power requirement for founding a town
// (7 normally, 6 if player has Fire +2 favor tile)
func GetTownPowerRequirement(playerTiles []FavorTileType) int {
	if HasFavorTile(playerTiles, FavorFire2) {
		return 6
	}
	return 7
}

// GetAir1PassVP returns VP gained when passing based on Trading House count
// Only applies if player has Air +1 favor tile
func GetAir1PassVP(playerTiles []FavorTileType, tradingHouseCount int) int {
	if !HasFavorTile(playerTiles, FavorAir1) {
		return 0
	}

	switch tradingHouseCount {
	case 0:
		return 0
	case 1:
		return 2
	case 2:
		return 3
	case 3:
		return 3
	case 4:
		return 4
	default:
		return 4 // Max 4 trading houses
	}
}

// GetFavorTileCount returns how many favor tiles a player should receive
// when building a Temple or Sanctuary
// Chaos Magicians get 2 tiles, all other factions get 1
func GetFavorTileCount(factionType models.FactionType) int {
	if factionType == models.FactionChaosMagicians {
		return 2
	}
	return 1
}
