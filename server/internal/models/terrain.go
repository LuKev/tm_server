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
	TerrainRiver // River hexes - cannot be built on, used for shipping
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
	default:
		return "Unknown"
	}
}
