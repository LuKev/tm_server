package models

// FactionType enumerates the 14 base game factions
// Note: Exact abilities to be implemented in the game engine layer

type FactionType int

const (
	FactionUnknown FactionType = iota // 0 = unknown/uninitialized
	FactionNomads
	FactionFakirs
	FactionChaosMagicians
	FactionGiants
	FactionSwarmlings
	FactionMermaids
	FactionWitches
	FactionAuren
	FactionHalflings
	FactionCultists
	FactionAlchemists
	FactionDarklings
	FactionEngineers
	FactionDwarves
)

func (f FactionType) String() string {
	switch f {
	case FactionUnknown:
		return "Unknown"
	case FactionNomads:
		return "Nomads"
	case FactionFakirs:
		return "Fakirs"
	case FactionChaosMagicians:
		return "ChaosMagicians"
	case FactionGiants:
		return "Giants"
	case FactionSwarmlings:
		return "Swarmlings"
	case FactionMermaids:
		return "Mermaids"
	case FactionWitches:
		return "Witches"
	case FactionAuren:
		return "Auren"
	case FactionHalflings:
		return "Halflings"
	case FactionCultists:
		return "Cultists"
	case FactionAlchemists:
		return "Alchemists"
	case FactionDarklings:
		return "Darklings"
	case FactionEngineers:
		return "Engineers"
	case FactionDwarves:
		return "Dwarves"
	default:
		return "Unknown"
	}
}

// FactionColor represents the terrain color of a faction
type FactionColor int

const (
	ColorYellow FactionColor = iota // Desert
	ColorRed                        // Wasteland
	ColorBlue                       // Lake
	ColorGreen                      // Forest
	ColorBrown                      // Plains
	ColorBlack                      // Swamp
	ColorGray                       // Mountain
)

// GetFactionColor returns the color/terrain type of a faction
func (f FactionType) GetFactionColor() FactionColor {
	switch f {
	case FactionNomads, FactionFakirs:
		return ColorYellow // Desert
	case FactionChaosMagicians, FactionGiants:
		return ColorRed // Wasteland
	case FactionSwarmlings, FactionMermaids:
		return ColorBlue // Lake
	case FactionWitches, FactionAuren:
		return ColorGreen // Forest
	case FactionHalflings, FactionCultists:
		return ColorBrown // Plains
	case FactionAlchemists, FactionDarklings:
		return ColorBlack // Swamp
	case FactionEngineers, FactionDwarves:
		return ColorGray // Mountain
	default:
		return ColorYellow // Default
	}
}
