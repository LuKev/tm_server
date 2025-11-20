package models

// TownTileType represents the type of town tile bonus
type TownTileType int

const (
	TownTile5Points TownTileType = iota // +5 VP, +6 coins, +1 key (immediate)
	TownTile6Points                     // +6 VP, +8 power, +1 key (immediate)
	TownTile7Points                     // +7 VP, +2 workers, +1 key (immediate)
	TownTile4Points                     // +4 VP, +1 shipping/range, +1 key (TW7)
	TownTile8Points                     // +8 VP, +1 on all cult tracks, +1 key
	TownTile9Points                     // +9 VP, +1 priest, +1 key (immediate)
	TownTile11Points                    // +11 VP, +1 key
	TownTile2Points                     // +2 VP, +2 on all cult tracks, +2 keys
)
