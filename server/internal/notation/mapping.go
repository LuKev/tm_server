package notation

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

// ParseFavorTileCode converts a concise notation code (e.g., "FAV-F1") to a FavorTileType
func ParseFavorTileCode(code string) (game.FavorTileType, error) {
	switch code {
	case "FAV-F3":
		return game.FavorFire3, nil
	case "FAV-W3":
		return game.FavorWater3, nil
	case "FAV-E3":
		return game.FavorEarth3, nil
	case "FAV-A3":
		return game.FavorAir3, nil
	case "FAV-F2":
		return game.FavorFire2, nil
	case "FAV-W2":
		return game.FavorWater2, nil
	case "FAV-E2":
		return game.FavorEarth2, nil
	case "FAV-A2":
		return game.FavorAir2, nil
	case "FAV-F1":
		return game.FavorFire1, nil
	case "FAV-W1":
		return game.FavorWater1, nil
	case "FAV-E1":
		return game.FavorEarth1, nil
	case "FAV-A1":
		return game.FavorAir1, nil
	default:
		return game.FavorTileUnknown, fmt.Errorf("unknown favor tile code: %s", code)
	}
}

// GetTownTileFromVP converts a VP amount to a TownTileType
// Assumes standard town tiles where VP values are unique
func GetTownTileFromVP(vp int) (models.TownTileType, error) {
	switch vp {
	case 2:
		return models.TownTile2Points, nil
	case 4:
		return models.TownTile4Points, nil
	case 5:
		return models.TownTile5Points, nil
	case 6:
		return models.TownTile6Points, nil
	case 7:
		return models.TownTile7Points, nil
	case 8:
		return models.TownTile8Points, nil
	case 9:
		return models.TownTile9Points, nil
	case 11:
		return models.TownTile11Points, nil
	default:
		return models.TownTileUnknown, fmt.Errorf("unknown town tile VP: %d", vp)
	}
}

// ParsePowerActionCode converts a power action code (e.g., "ACT1") to a PowerActionType
// Returns PowerActionUnknown if unknown
func ParsePowerActionCode(code string) game.PowerActionType {
	switch code {
	case "ACT1":
		return game.PowerActionBridge
	case "ACT2":
		return game.PowerActionPriest
	case "ACT3":
		return game.PowerActionWorkers
	case "ACT4":
		return game.PowerActionCoins
	case "ACT5":
		return game.PowerActionSpade1
	case "ACT6":
		return game.PowerActionSpade2
	default:
		return game.PowerActionUnknown
	}
}
