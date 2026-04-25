package models

// TerrainType represents the different terrain types in Terra Mystica
type TerrainType int

const (
	TerrainPlains TerrainType = iota
	TerrainSwamp
	TerrainLake
	TerrainForest
	TerrainMountain
	TerrainWasteland
	TerrainDesert
	TerrainRiver                   // River hexes - cannot be built on, used for shipping
	TerrainIce                     // Fire & Ice expansion terrain
	TerrainVolcano                 // Fire & Ice expansion terrain
	TerrainTypeUnknown TerrainType = -1
)

func (t TerrainType) String() string {
	switch t {
	case TerrainDesert:
		return "Desert"
	case TerrainPlains:
		return "Plains"
	case TerrainSwamp:
		return "Swamp"
	case TerrainLake:
		return "Lake"
	case TerrainForest:
		return "Forest"
	case TerrainMountain:
		return "Mountain"
	case TerrainWasteland:
		return "Wasteland"
	case TerrainRiver:
		return "River"
	case TerrainIce:
		return "Ice"
	case TerrainVolcano:
		return "Volcano"
	default:
		return "Unknown"
	}
}
