package models

// TerrainType represents the seven terrain types in Terra Mystica
// Desert, Plains, Swamp, Lake, Forest, Mountain, Wasteland
// We use iota for stable ordinal mapping; JSON uses string names via Marshal/Unmarshal helpers if needed later

type TerrainType int

const (
	TerrainDesert TerrainType = iota
	TerrainPlains
	TerrainSwamp
	TerrainLake
	TerrainForest
	TerrainMountain
	TerrainWasteland
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
	default:
		return "Unknown"
	}
}
