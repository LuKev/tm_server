package models

// TownTileType represents the type of town tile bonus
type TownTileType int

const (
	TownTile5Points  TownTileType = iota // +5 VP, +6 coins, +1 key (immediate)
	TownTile6Points                      // +6 VP, +8 power, +1 key (immediate)
	TownTile7Points                      // +7 VP, +2 workers, +1 key (immediate)
	TownTile4Points                      // +4 VP, +1 shipping/range, +1 key (TW7)
	TownTile8Points                      // +8 VP, +1 on all cult tracks, +1 key
	TownTile9Points                      // +9 VP, +1 priest, +1 key (immediate)
	TownTile11Points                     // +11 VP, +1 key
	TownTile2Points                      // +2 VP, +2 on all cult tracks, +2 keys
	TownTileUnknown  TownTileType = -1
)

func TownTileTypeFromString(s string) TownTileType {
	switch s {
	case "5 VP, 6 Coins":
		return TownTile5Points
	case "6 VP, 8 Power":
		return TownTile6Points
	case "7 VP, 2 Workers":
		return TownTile7Points
	case "4 VP, Shipping":
		return TownTile4Points
	case "8 VP, Cult":
		return TownTile8Points
	case "9 VP, Priest":
		return TownTile9Points
	case "11 VP":
		return TownTile11Points
	case "2 VP, Cult":
		return TownTile2Points
	default:
		return TownTileUnknown
	}
}
