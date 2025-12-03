import { FactionType } from '../types/game.types'

export interface FactionData {
    id: FactionType
    type: string
    name: string
    homeTerrain: string
    color: string // Tailwind color name (e.g. 'red', 'blue')
    description: string
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
    }
]
