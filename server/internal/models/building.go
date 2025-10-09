package models

// BuildingType represents the different buildings in Terra Mystica
// Dwelling, Trading House, Temple, Sanctuary, Stronghold

type BuildingType int

const (
	BuildingDwelling BuildingType = iota
	BuildingTradingHouse
	BuildingTemple
	BuildingSanctuary
	BuildingStronghold
)

func (b BuildingType) String() string {
	switch b {
	case BuildingDwelling:
		return "Dwelling"
	case BuildingTradingHouse:
		return "TradingHouse"
	case BuildingTemple:
		return "Temple"
	case BuildingSanctuary:
		return "Sanctuary"
	case BuildingStronghold:
		return "Stronghold"
	default:
		return "Unknown"
	}
}
