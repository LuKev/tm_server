package models

// FactionType enumerates the 14 base game factions
// Note: Exact abilities to be implemented in the game engine layer

type FactionType int

const (
	FactionNomads FactionType = iota
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
