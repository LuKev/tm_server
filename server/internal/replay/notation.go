package replay

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

// BonusTileType represents the different bonus tiles
type BonusTileType string

const (
	BON1  BonusTileType = "BON1"  // Spade
	BON2  BonusTileType = "BON2"  // +4 C
	BON3  BonusTileType = "BON3"  // 6 coins
	BON4  BonusTileType = "BON4"  // [4c] +3 PW, 1 ship
	BON5  BonusTileType = "BON5"  // [2c] +1 W, +3 PW
	BON6  BonusTileType = "BON6"  // pass-vp:SH*4, pass-vp:SA*4, +2 W
	BON7  BonusTileType = "BON7"  // pass-vp:TP*2, +1 W
	BON8  BonusTileType = "BON8"  // [1c] +1 P
	BON9  BonusTileType = "BON9"  // pass-vp:D*1, +2 C
	BON10 BonusTileType = "BON10" // Shipping VP scoring
)

// FavorTileType represents the different favor tiles
type FavorTileType string

const (
	// 3-step favor tiles (Fire/Water/Air/Earth)
	FAV1 FavorTileType = "FAV1" // Fire 3-step
	FAV2 FavorTileType = "FAV2" // Water 3-step
	FAV3 FavorTileType = "FAV3" // Air 3-step
	FAV4 FavorTileType = "FAV4" // Earth 3-step

	// 2-step favor tiles
	FAV5  FavorTileType = "FAV5"  // 2-step
	FAV6  FavorTileType = "FAV6"  // 2-step
	FAV7  FavorTileType = "FAV7"  // 2-step
	FAV8  FavorTileType = "FAV8"  // 2-step
	FAV9  FavorTileType = "FAV9"  // 1-step
	FAV10 FavorTileType = "FAV10" // 1-step
	FAV11 FavorTileType = "FAV11" // 1-step
	FAV12 FavorTileType = "FAV12" // 1-step
)

// TownTileType represents the different town tiles
type TownTileType string

const (
	TW1 TownTileType = "TW1" // Coin town
	TW2 TownTileType = "TW2" // Workers town
	TW3 TownTileType = "TW3" // Priest town
	TW4 TownTileType = "TW4" // Power town
	TW5 TownTileType = "TW5" // 8 VP cult town
	TW6 TownTileType = "TW6" // Two key town
	TW7 TownTileType = "TW7" // Shipping/range upgrade town
	TW8 TownTileType = "TW8" // 11 point town
)

// ParseTownTile converts a town tile string to the internal TownTileType
func ParseTownTile(townStr string) (game.TownTileType, error) {
	switch townStr {
	case "TW1":
		return game.TownTile5Points, nil
	case "TW2":
		return game.TownTile7Points, nil // 7 VP + 2 workers + 1 key
	case "TW3":
		return game.TownTile9Points, nil // 9 VP + 1 priest + 1 key
	case "TW4":
		return game.TownTile6Points, nil
	case "TW5":
		return game.TownTile8Points, nil
	case "TW6":
		return game.TownTile2Points, nil
	case "TW7":
		return game.TownTile4Points, nil
	case "TW8":
		return game.TownTile11Points, nil
	default:
		return 0, fmt.Errorf("unknown town tile: %s", townStr)
	}
}

// PowerActionType represents the different power actions
type PowerActionType string

const (
	ACT1 PowerActionType = "ACT1" // Bridge
	ACT2 PowerActionType = "ACT2" // Priest
	ACT3 PowerActionType = "ACT3" // Workers
	ACT4 PowerActionType = "ACT4" // Coins
	ACT5 PowerActionType = "ACT5" // Single spade
	ACT6 PowerActionType = "ACT6" // Double spade
)

// SpecialActionType represents faction-specific special actions
type SpecialActionType string

const (
	ACTW SpecialActionType = "ACTW" // Witches special action
	ACTN SpecialActionType = "ACTN" // Nomads special action
	ACTG SpecialActionType = "ACTG" // Giants special action
	ACTC SpecialActionType = "ACTC" // Chaos Magicians special action
	ACTA SpecialActionType = "ACTA" // Auren special action
	ACTS SpecialActionType = "ACTS" // Swarmlings special action
)

// ScoringTileType represents the different scoring tiles
type ScoringTileType string

const (
	SCORE1 ScoringTileType = "SCORE1"
	SCORE2 ScoringTileType = "SCORE2"
	SCORE3 ScoringTileType = "SCORE3"
	SCORE4 ScoringTileType = "SCORE4"
	SCORE5 ScoringTileType = "SCORE5"
	SCORE6 ScoringTileType = "SCORE6"
	SCORE7 ScoringTileType = "SCORE7"
	SCORE8 ScoringTileType = "SCORE8"
)

// ParseFaction converts a faction string to the internal FactionType
func ParseFaction(factionStr string) (models.FactionType, error) {
	switch factionStr {
	case "witches":
		return models.FactionWitches, nil
	case "nomads":
		return models.FactionNomads, nil
	case "halflings":
		return models.FactionHalflings, nil
	case "cultists":
		return models.FactionCultists, nil
	case "alchemists":
		return models.FactionAlchemists, nil
	case "darklings":
		return models.FactionDarklings, nil
	case "mermaids":
		return models.FactionMermaids, nil
	case "swarmlings":
		return models.FactionSwarmlings, nil
	case "engineers":
		return models.FactionEngineers, nil
	case "chaosmagicians":
		return models.FactionChaosMagicians, nil
	case "giants":
		return models.FactionGiants, nil
	case "fakirs":
		return models.FactionFakirs, nil
	case "dwarves":
		return models.FactionDwarves, nil
	case "auren":
		return models.FactionAuren, nil
	default:
		return 0, fmt.Errorf("unknown faction: %s", factionStr)
	}
}

// ParseBuildingType converts a building string to the internal BuildingType
func ParseBuildingType(buildingStr string) (models.BuildingType, error) {
	switch buildingStr {
	case "D":
		return models.BuildingDwelling, nil
	case "TP":
		return models.BuildingTradingHouse, nil
	case "TE":
		return models.BuildingTemple, nil
	case "SH":
		return models.BuildingStronghold, nil
	case "SA":
		return models.BuildingSanctuary, nil
	default:
		return 0, fmt.Errorf("unknown building type: %s", buildingStr)
	}
}

// ParseTerrainColor converts a color string to the internal TerrainType
func ParseTerrainColor(color string) (models.TerrainType, error) {
	switch color {
	case "gray", "grey":
		return models.TerrainMountain, nil // Gray = Mountain (Engineers, Dwarves)
	case "blue":
		return models.TerrainLake, nil // Blue = Lake (Mermaids)
	case "brown":
		return models.TerrainPlains, nil // Brown = Plains (Halflings)
	case "green":
		return models.TerrainForest, nil // Green = Forest (Witches)
	case "yellow":
		return models.TerrainDesert, nil // Yellow = Desert (Fakirs, Nomads)
	case "red":
		return models.TerrainWasteland, nil // Red = Wasteland (Giants, Chaos Magicians)
	case "black":
		return models.TerrainSwamp, nil // Black = Swamp (Darklings, Alchemists)
	default:
		return 0, fmt.Errorf("unknown terrain color: %s", color)
	}
}

// ParseCultTrack converts a cult track string to the internal CultType
func ParseCultTrack(cultStr string) (models.CultType, error) {
	// Convert to uppercase for case-insensitive matching
	cultStr = strings.ToUpper(cultStr)
	switch cultStr {
	case "FIRE":
		return models.CultFire, nil
	case "WATER":
		return models.CultWater, nil
	case "EARTH":
		return models.CultEarth, nil
	case "AIR":
		return models.CultAir, nil
	default:
		return 0, fmt.Errorf("unknown cult track: %s", cultStr)
	}
}

// ParseBonusCard converts a bonus card string to the internal BonusCardType
func ParseBonusCard(bonusStr string) (game.BonusCardType, error) {
	switch bonusStr {
	case "BON1": // Spade
		return game.BonusCardSpade, nil
	case "BON2": // +4 C
		return game.BonusCardCultAdvance, nil
	case "BON3": // 6 coins
		return game.BonusCard6Coins, nil
	case "BON4": // +3 PW, 1 ship
		return game.BonusCardShipping, nil
	case "BON5": // +1 W, +3 PW
		return game.BonusCardWorkerPower, nil
	case "BON6": // pass-vp:SH/SA, +2 W
		return game.BonusCardStrongholdSanctuary, nil
	case "BON7": // pass-vp:TP, +1 W
		return game.BonusCardTradingHouseVP, nil
	case "BON8": // +1 P
		return game.BonusCardPriest, nil
	case "BON9": // pass-vp:D, +2 C
		return game.BonusCardDwellingVP, nil
	case "BON10": // Shipping VP
		return game.BonusCardShippingVP, nil
	default:
		return 0, fmt.Errorf("unknown bonus card: %s", bonusStr)
	}
}

// ParsePowerActionType parses a power action string (ACT1-ACT6) to game.PowerActionType
func ParsePowerActionType(actionStr string) (game.PowerActionType, error) {
	switch actionStr {
	case "ACT1": // 3 PW: Build a bridge
		return game.PowerActionBridge, nil
	case "ACT2": // 3 PW: Gain 1 priest
		return game.PowerActionPriest, nil
	case "ACT3": // 4 PW: Gain 2 workers
		return game.PowerActionWorkers, nil
	case "ACT4": // 4 PW: Gain 7 coins
		return game.PowerActionCoins, nil
	case "ACT5": // 4 PW: 1 free spade for transform
		return game.PowerActionSpade1, nil
	case "ACT6": // 6 PW: 2 free spades for transform
		return game.PowerActionSpade2, nil
	default:
		return 0, fmt.Errorf("unknown power action: %s", actionStr)
	}
}

// ParseFavorTile parses a favor tile string (FAV1-FAV12) to game.FavorTileType
func ParseFavorTile(tileStr string) (game.FavorTileType, error) {
	switch tileStr {
	// +3 cult advancement tiles (1 of each)
	case "FAV1": // Fire +3
		return game.FavorFire3, nil
	case "FAV2": // Water +3
		return game.FavorWater3, nil
	case "FAV3": // Earth +3
		return game.FavorEarth3, nil
	case "FAV4": // Air +3
		return game.FavorAir3, nil
	// +2 cult advancement tiles with abilities (3 of each)
	case "FAV5": // Fire +2 (town power requirement 6)
		return game.FavorFire2, nil
	case "FAV6": // Water +2 (cult track action)
		return game.FavorWater2, nil
	case "FAV7": // Earth +2 (+1 worker, +1 power income)
		return game.FavorEarth2, nil
	case "FAV8": // Air +2 (+4 power income)
		return game.FavorAir2, nil
	// +1 cult advancement tiles with abilities (3 of each)
	case "FAV9": // Fire +1 (+3 coins income)
		return game.FavorFire1, nil
	case "FAV10": // Water +1 (+3 VP per DW->TP upgrade)
		return game.FavorWater1, nil
	case "FAV11": // Earth +1 (+2 VP per dwelling)
		return game.FavorEarth1, nil
	case "FAV12": // Air +1 (VP at pass based on trading houses)
		return game.FavorAir1, nil
	default:
		return 0, fmt.Errorf("unknown favor tile: %s", tileStr)
	}
}
