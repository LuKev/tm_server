package models

// FactionType enumerates the base game factions plus supported fan factions.
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
	FactionArchitects
	FactionArchivists
	FactionAtlanteans
	FactionChashDallah
	FactionChildrenOfTheWyrm
	FactionConspirators
	FactionDjinni
	FactionDynionGeifr
	FactionGoblins
	FactionProspectors
	FactionTheEnlightened
	FactionTimeTravelers
	FactionTreasurers
	FactionWisps
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
	case FactionArchitects:
		return "Architects"
	case FactionArchivists:
		return "Archivists"
	case FactionAtlanteans:
		return "Atlanteans"
	case FactionChashDallah:
		return "ChashDallah"
	case FactionChildrenOfTheWyrm:
		return "ChildrenOfTheWyrm"
	case FactionConspirators:
		return "Conspirators"
	case FactionDjinni:
		return "Djinni"
	case FactionDynionGeifr:
		return "DynionGeifr"
	case FactionGoblins:
		return "Goblins"
	case FactionProspectors:
		return "Prospectors"
	case FactionTheEnlightened:
		return "TheEnlightened"
	case FactionTimeTravelers:
		return "TimeTravelers"
	case FactionTreasurers:
		return "Treasurers"
	case FactionWisps:
		return "Wisps"
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
	case FactionArchitects, FactionTreasurers:
		return ColorRed // Wasteland
	case FactionArchivists, FactionDjinni:
		return ColorYellow // Desert
	case FactionAtlanteans, FactionWisps:
		return ColorBlue // Lake
	case FactionChashDallah, FactionTheEnlightened:
		return ColorGreen // Forest
	case FactionChildrenOfTheWyrm, FactionGoblins:
		return ColorBlack // Swamp
	case FactionConspirators, FactionDynionGeifr:
		return ColorGray // Mountain
	case FactionProspectors, FactionTimeTravelers:
		return ColorBrown // Plains
	default:
		return ColorYellow // Default
	}
}

func (f FactionType) IsFanFaction() bool {
	switch f {
	case FactionArchitects,
		FactionArchivists,
		FactionAtlanteans,
		FactionChashDallah,
		FactionChildrenOfTheWyrm,
		FactionConspirators,
		FactionDjinni,
		FactionDynionGeifr,
		FactionGoblins,
		FactionProspectors,
		FactionTheEnlightened,
		FactionTimeTravelers,
		FactionTreasurers,
		FactionWisps:
		return true
	default:
		return false
	}
}

// FactionTypeFromString converts a string representation to FactionType
func FactionTypeFromString(s string) FactionType {
	switch s {
	case "Nomads":
		return FactionNomads
	case "Fakirs":
		return FactionFakirs
	case "ChaosMagicians", "Chaos Magicians":
		return FactionChaosMagicians
	case "Giants":
		return FactionGiants
	case "Swarmlings":
		return FactionSwarmlings
	case "Mermaids":
		return FactionMermaids
	case "Witches":
		return FactionWitches
	case "Auren":
		return FactionAuren
	case "Halflings":
		return FactionHalflings
	case "Cultists":
		return FactionCultists
	case "Alchemists":
		return FactionAlchemists
	case "Darklings":
		return FactionDarklings
	case "Engineers":
		return FactionEngineers
	case "Dwarves":
		return FactionDwarves
	case "Architects":
		return FactionArchitects
	case "Archivists":
		return FactionArchivists
	case "Atlanteans":
		return FactionAtlanteans
	case "ChashDallah", "Chash Dallah", "CashDallah", "Cash Dallah":
		return FactionChashDallah
	case "ChildrenOfTheWyrm", "Children of the Wyrm", "Children Of The Wyrm":
		return FactionChildrenOfTheWyrm
	case "Conspirators":
		return FactionConspirators
	case "Djinni", "Djinn":
		return FactionDjinni
	case "DynionGeifr", "Dynion Geifr":
		return FactionDynionGeifr
	case "Goblins":
		return FactionGoblins
	case "Prospectors", "GoldDiggers", "Gold Diggers":
		return FactionProspectors
	case "TheEnlightened", "The Enlightened":
		return FactionTheEnlightened
	case "TimeTravelers", "Time Travelers", "TimeTravellers", "Time Travellers":
		return FactionTimeTravelers
	case "Treasurers":
		return FactionTreasurers
	case "Wisps":
		return FactionWisps
	default:
		return FactionUnknown
	}
}
