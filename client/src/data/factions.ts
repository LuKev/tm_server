import { FactionType } from '../types/game.types'

export interface FactionData {
    id: FactionType
    type: string
    name: string
    homeTerrain: string
    color: string // Tailwind color name (e.g. 'red', 'blue')
    description: string
    isFanFaction?: boolean
}

export const FACTIONS: FactionData[] = [
    {
        id: FactionType.Nomads,
        type: 'Nomads',
        name: 'Nomads',
        homeTerrain: 'Desert',
        color: 'yellow',
        description: 'Masters of the desert, Nomads can build trading houses cheaply.'
    },
    {
        id: FactionType.Witches,
        type: 'Witches',
        name: 'Witches',
        homeTerrain: 'Forest',
        color: 'green',
        description: 'Dwellers of the forest, Witches can fly to build dwellings in distant lands.'
    },
    {
        id: FactionType.Halflings,
        type: 'Halflings',
        name: 'Halflings',
        homeTerrain: 'Plain',
        color: 'amber',
        description: 'Hardworking farmers who gain victory points for digging.'
    },
    {
        id: FactionType.Mermaids,
        type: 'Mermaids',
        name: 'Mermaids',
        homeTerrain: 'Lake',
        color: 'blue',
        description: 'Water-bound creatures who can skip river spaces to build towns.'
    },
    {
        id: FactionType.Giants,
        type: 'Giants',
        name: 'Giants',
        homeTerrain: 'Wasteland',
        color: 'red',
        description: 'Powerful giants who can terraform any terrain with fixed effort.'
    },
    {
        id: FactionType.ChaosMagicians,
        type: 'ChaosMagicians',
        name: 'Chaos Magicians',
        homeTerrain: 'Wasteland',
        color: 'red',
        description: 'Masters of magic who start with a single dwelling but gain favor tiles easily.'
    },
    {
        id: FactionType.Engineers,
        type: 'Engineers',
        name: 'Engineers',
        homeTerrain: 'Mountain',
        color: 'gray',
        description: 'Builders who construct bridges cheaply and score for them.'
    },
    {
        id: FactionType.Dwarves,
        type: 'Dwarves',
        name: 'Dwarves',
        homeTerrain: 'Mountain',
        color: 'gray',
        description: 'Miners who can tunnel through mountains to reach new lands.'
    },
    {
        id: FactionType.Fakirs,
        type: 'Fakirs',
        name: 'Fakirs',
        homeTerrain: 'Desert',
        color: 'yellow',
        description: 'Mystics who can fly on magic carpets to reach distant lands.'
    },
    {
        id: FactionType.Alchemists,
        type: 'Alchemists',
        name: 'Alchemists',
        homeTerrain: 'Swamp',
        color: 'slate', // black/dark gray
        description: 'Seekers of knowledge who can trade victory points for coins.'
    },
    {
        id: FactionType.Darklings,
        type: 'Darklings',
        name: 'Darklings',
        homeTerrain: 'Swamp',
        color: 'slate',
        description: 'Creatures of the dark who use priests to terraform.'
    },
    {
        id: FactionType.Auren,
        type: 'Auren',
        name: 'Auren',
        homeTerrain: 'Forest',
        color: 'green',
        description: 'Peaceful forest dwellers who gain power when others build near them.'
    },
    {
        id: FactionType.Swarmlings,
        type: 'Swarmlings',
        name: 'Swarmlings',
        homeTerrain: 'Lake',
        color: 'blue',
        description: 'Numerous creatures who can build trading houses on any terrain.'
    },
    {
        id: FactionType.Cultists,
        type: 'Cultists',
        name: 'Cultists',
        homeTerrain: 'Plain',
        color: 'amber',
        description: 'Devout followers who gain power when others refuse to take power.'
    },
    {
        id: FactionType.Architects,
        type: 'Architects',
        name: 'Architects',
        homeTerrain: 'Wasteland',
        color: 'red',
        description: 'Bridge-focused builders who turn connected terrain into cheaper expansions.',
        isFanFaction: true,
    },
    {
        id: FactionType.Archivists,
        type: 'Archivists',
        name: 'Archivists',
        homeTerrain: 'Desert',
        color: 'yellow',
        description: 'Bonus-card specialists with stronger worker income and pass rewards.',
        isFanFaction: true,
    },
    {
        id: FactionType.Atlanteans,
        type: 'Atlanteans',
        name: 'Atlanteans',
        homeTerrain: 'Lake',
        color: 'blue',
        description: 'Begin with a stronghold town and scale rewards as that town grows.',
        isFanFaction: true,
    },
    {
        id: FactionType.ChashDallah,
        type: 'ChashDallah',
        name: 'Chash Dallah',
        homeTerrain: 'Forest',
        color: 'green',
        description: 'Use a separate income track instead of digging upgrades.',
        isFanFaction: true,
    },
    {
        id: FactionType.ChildrenOfTheWyrm,
        type: 'ChildrenOfTheWyrm',
        name: 'Children of the Wyrm',
        homeTerrain: 'Swamp',
        color: 'slate',
        description: 'River-spanning wyrmfolk with cheaper leeching and adjacency-linked power.',
        isFanFaction: true,
    },
    {
        id: FactionType.Conspirators,
        type: 'Conspirators',
        name: 'Conspirators',
        homeTerrain: 'Mountain',
        color: 'gray',
        description: 'Manipulate favor tiles and collect extra coins whenever a favor is gained.',
        isFanFaction: true,
    },
    {
        id: FactionType.Djinni,
        type: 'Djinni',
        name: 'Djinni',
        homeTerrain: 'Desert',
        color: 'yellow',
        description: 'Magic-lamp mystics who rearrange cult positions.',
        isFanFaction: true,
    },
    {
        id: FactionType.DynionGeifr,
        type: 'DynionGeifr',
        name: 'Dynion Geifr',
        homeTerrain: 'Mountain',
        color: 'gray',
        description: 'Goat folk with strong worker economy and high-power structures.',
        isFanFaction: true,
    },
    {
        id: FactionType.Goblins,
        type: 'Goblins',
        name: 'Goblins',
        homeTerrain: 'Swamp',
        color: 'slate',
        description: 'Treasure hoarders who cash tokens in for flexible rewards.',
        isFanFaction: true,
    },
    {
        id: FactionType.Prospectors,
        type: 'Prospectors',
        name: 'Prospectors',
        homeTerrain: 'Plain',
        color: 'amber',
        description: 'Golden-spade specialists who transform with coins instead of normal spades.',
        isFanFaction: true,
    },
    {
        id: FactionType.TheEnlightened,
        type: 'TheEnlightened',
        name: 'The Enlightened',
        homeTerrain: 'Forest',
        color: 'green',
        description: 'Convert and terraform through power, with a strong conversion-focused stronghold.',
        isFanFaction: true,
    },
    {
        id: FactionType.TimeTravelers,
        type: 'TimeTravelers',
        name: 'Time Travelers',
        homeTerrain: 'Plain',
        color: 'amber',
        description: 'Score previous and future round tiles instead of the current one.',
        isFanFaction: true,
    },
    {
        id: FactionType.Treasurers,
        type: 'Treasurers',
        name: 'Treasurers',
        homeTerrain: 'Wasteland',
        color: 'red',
        description: 'Bank income in a treasury now to multiply it at the start of the next round.',
        isFanFaction: true,
    },
    {
        id: FactionType.Wisps,
        type: 'Wisps',
        name: 'Wisps',
        homeTerrain: 'Lake',
        color: 'blue',
        description: 'Trading-post builders who immediately transform adjacent terrain.',
        isFanFaction: true,
    }
]
