import { FactionType } from '../types/game.types';

export interface BuildingSlot {
    cost: {
        workers?: number;
        coins?: number;
    };
    income: {
        workers?: number;
        coins?: number;
        priests?: number;
        power?: number;
        powerTokens?: number; // For gaining power tokens (bowl 1)
        cult?: number; // Generic cult step
    } | null; // null means no income (e.g. last dwelling often has none, or just reveals nothing)
}

export interface FactionBoardLayout {
    dwellings: BuildingSlot[];     // 8 slots
    tradingHouses: BuildingSlot[]; // 4 slots
    temples: BuildingSlot[];       // 3 slots
    sanctuary: BuildingSlot;       // 1 slot
    stronghold: BuildingSlot;      // 1 slot
}

const DEFAULT_DWELLING_COST = { workers: 1, coins: 2 };
const DEFAULT_TRADING_HOUSE_COST = { workers: 2, coins: 3 }; // 6 coins if neighbor
const DEFAULT_TEMPLE_COST = { workers: 2, coins: 5 };
const DEFAULT_STRONGHOLD_COST = { workers: 4, coins: 6 };
const DEFAULT_SANCTUARY_COST = { workers: 4, coins: 6 };

// Standard income (approximate for most factions)
const STANDARD_DWELLINGS: BuildingSlot[] = [
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: { workers: 1 } },
    { cost: DEFAULT_DWELLING_COST, income: null }, // 8th dwelling usually gives no additional income
];

const STANDARD_TRADING_HOUSES: BuildingSlot[] = [
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { power: 2, coins: 2 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { power: 2, coins: 2 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { power: 2, coins: 2 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { power: 2, coins: 2 } },
];

const STANDARD_TEMPLES: BuildingSlot[] = [
    { cost: DEFAULT_TEMPLE_COST, income: { priests: 1 } },
    { cost: DEFAULT_TEMPLE_COST, income: { priests: 1 } },
    { cost: DEFAULT_TEMPLE_COST, income: { priests: 1 } },
];

const STANDARD_SANCTUARY: BuildingSlot = {
    cost: DEFAULT_SANCTUARY_COST,
    income: { priests: 1 },
};

const STANDARD_STRONGHOLD: BuildingSlot = {
    cost: DEFAULT_STRONGHOLD_COST,
    income: { power: 2 }, // Varies wildly
};

// Chaos Magicians specific
// Dwellings: 1W, 2C. Income: 1W (x7), -
// Trading Houses: 2W, 3/6C. Income: 2C, 1P -> 2C, 1P -> 2C, 2P -> 2C, 2P
// Temples: 2W, 5C. Income: 1 Priest
// Sanctuary: 4W, 6C. Income: 1 Priest
// Stronghold: 4W, 6C. Income: 2 Power (Double Action ability)

const CHAOS_MAGICIAN_TRADING_HOUSES: BuildingSlot[] = [
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 1 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 1 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 2 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 2 } },
];

// --- Faction Specific Costs ---

// Auren
const AUREN_SANCTUARY_COST = { workers: 4, coins: 8 };

// Chaos Magicians
const CHAOS_SANCTUARY_COST = { workers: 4, coins: 8 };
const CHAOS_STRONGHOLD_COST = { workers: 4, coins: 4 };

// Cultists
const CULTISTS_SANCTUARY_COST = { workers: 4, coins: 8 };
const CULTISTS_STRONGHOLD_COST = { workers: 4, coins: 8 };

// Darklings
const DARKLINGS_SANCTUARY_COST = { workers: 4, coins: 10 };

// Engineers
const ENGINEERS_DWELLING_COST = { workers: 1, coins: 1 };
const ENGINEERS_TRADING_HOUSE_COST = { workers: 1, coins: 4 };
const ENGINEERS_TEMPLE_COST = { workers: 1, coins: 4 };
const ENGINEERS_SANCTUARY_COST = { workers: 3, coins: 6 };
const ENGINEERS_STRONGHOLD_COST = { workers: 3, coins: 6 };

// Fakirs
const FAKIRS_STRONGHOLD_COST = { workers: 4, coins: 10 };

// Halflings
const HALFLINGS_STRONGHOLD_COST = { workers: 4, coins: 8 };

// Mermaids
const MERMAIDS_SANCTUARY_COST = { workers: 4, coins: 8 };

// Nomads
const NOMADS_STRONGHOLD_COST = { workers: 4, coins: 8 };

// Swarmlings
const SWARMLINGS_DWELLING_COST = { workers: 2, coins: 3 };
const SWARMLINGS_TRADING_HOUSE_COST = { workers: 3, coins: 8 };
const SWARMLINGS_TEMPLE_COST = { workers: 3, coins: 6 };
const SWARMLINGS_SANCTUARY_COST = { workers: 5, coins: 8 };
const SWARMLINGS_STRONGHOLD_COST = { workers: 5, coins: 8 };


// --- Faction Specific Boards ---

const AUREN_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: { cost: AUREN_SANCTUARY_COST, income: { priests: 1 } },
    stronghold: STANDARD_STRONGHOLD,
};

const CHAOS_MAGICIAN_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: CHAOS_MAGICIAN_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: { cost: CHAOS_SANCTUARY_COST, income: { priests: 1 } },
    stronghold: { cost: CHAOS_STRONGHOLD_COST, income: { workers: 2 } }, // Income: 2 Workers
};

const CULTISTS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: { cost: CULTISTS_SANCTUARY_COST, income: { priests: 1 } },
    stronghold: { cost: CULTISTS_STRONGHOLD_COST, income: { power: 2 } },
};

const DARKLINGS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: { cost: DARKLINGS_SANCTUARY_COST, income: { priests: 2 } }, // Income: 2 Priests
    stronghold: STANDARD_STRONGHOLD,
};

// Engineers Income Logic
// Dwellings: 1, 2, 4, 5, 7, 8 give 1 Worker. 3 and 6 give nothing.
const ENGINEERS_DWELLINGS = STANDARD_DWELLINGS.map((d, i) => {
    const slotIndex = i + 1;
    if (slotIndex === 3 || slotIndex === 6) {
        return { ...d, cost: ENGINEERS_DWELLING_COST, income: null };
    }
    return { ...d, cost: ENGINEERS_DWELLING_COST };
});

// Temples: 1st and 3rd give 1 Priest. 2nd gives 5 Power.
const ENGINEERS_TEMPLES = STANDARD_TEMPLES.map((d, i) => {
    const slotIndex = i + 1;
    if (slotIndex === 2) {
        return { ...d, cost: ENGINEERS_TEMPLE_COST, income: { power: 5 } };
    }
    return { ...d, cost: ENGINEERS_TEMPLE_COST };
});

const ENGINEERS_TRADING_HOUSES = STANDARD_TRADING_HOUSES.map(d => ({ ...d, cost: ENGINEERS_TRADING_HOUSE_COST }));

const ENGINEERS_BOARD: FactionBoardLayout = {
    dwellings: ENGINEERS_DWELLINGS,
    tradingHouses: ENGINEERS_TRADING_HOUSES,
    temples: ENGINEERS_TEMPLES,
    sanctuary: { cost: ENGINEERS_SANCTUARY_COST, income: { priests: 1 } },
    stronghold: { cost: ENGINEERS_STRONGHOLD_COST, income: { power: 2 } },
};

const FAKIRS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: STANDARD_SANCTUARY,
    stronghold: { cost: FAKIRS_STRONGHOLD_COST, income: { priests: 1 } }, // Income: 1 Priest
};

const GIANTS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: STANDARD_SANCTUARY,
    stronghold: { cost: DEFAULT_STRONGHOLD_COST, income: { power: 4 } }, // Income: 4 Power
};

const HALFLINGS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: STANDARD_SANCTUARY,
    stronghold: { cost: HALFLINGS_STRONGHOLD_COST, income: { power: 2 } },
};

const MERMAIDS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: STANDARD_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: { cost: MERMAIDS_SANCTUARY_COST, income: { priests: 1 } },
    stronghold: { cost: DEFAULT_STRONGHOLD_COST, income: { power: 4 } }, // Income: 4 Power
};

// Nomads Trading Houses: 1-2: 2C+1PW, 3: 3C+1PW, 4: 4C+1PW
const NOMADS_TRADING_HOUSES: BuildingSlot[] = [
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 1 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 2, power: 1 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 3, power: 1 } },
    { cost: DEFAULT_TRADING_HOUSE_COST, income: { coins: 4, power: 1 } },
];

const NOMADS_BOARD: FactionBoardLayout = {
    dwellings: STANDARD_DWELLINGS,
    tradingHouses: NOMADS_TRADING_HOUSES,
    temples: STANDARD_TEMPLES,
    sanctuary: STANDARD_SANCTUARY,
    stronghold: { cost: NOMADS_STRONGHOLD_COST, income: { power: 2 } },
};

// Swarmlings Trading Houses: 1-3: 2C+2PW, 4: 3C+2PW
const SWARMLINGS_TRADING_HOUSES_DATA: BuildingSlot[] = [
    { cost: SWARMLINGS_TRADING_HOUSE_COST, income: { coins: 2, power: 2 } },
    { cost: SWARMLINGS_TRADING_HOUSE_COST, income: { coins: 2, power: 2 } },
    { cost: SWARMLINGS_TRADING_HOUSE_COST, income: { coins: 2, power: 2 } },
    { cost: SWARMLINGS_TRADING_HOUSE_COST, income: { coins: 3, power: 2 } },
];

const SWARMLINGS_DWELLINGS = STANDARD_DWELLINGS.map(d => ({ ...d, cost: SWARMLINGS_DWELLING_COST }));
const SWARMLINGS_TEMPLES = STANDARD_TEMPLES.map(d => ({ ...d, cost: SWARMLINGS_TEMPLE_COST }));

const SWARMLINGS_BOARD: FactionBoardLayout = {
    dwellings: SWARMLINGS_DWELLINGS,
    tradingHouses: SWARMLINGS_TRADING_HOUSES_DATA,
    temples: SWARMLINGS_TEMPLES,
    sanctuary: { cost: SWARMLINGS_SANCTUARY_COST, income: { priests: 2 } }, // Income: 2 Priests
    stronghold: { cost: SWARMLINGS_STRONGHOLD_COST, income: { power: 4 } }, // Income: 4 Power
};


export const FACTION_BOARDS: Record<FactionType, FactionBoardLayout> = {
    [FactionType.ChaosMagicians]: CHAOS_MAGICIAN_BOARD,
    [FactionType.Darklings]: DARKLINGS_BOARD,
    [FactionType.Auren]: AUREN_BOARD,
    [FactionType.Cultists]: CULTISTS_BOARD,
    [FactionType.Engineers]: ENGINEERS_BOARD,
    [FactionType.Fakirs]: FAKIRS_BOARD,
    [FactionType.Giants]: GIANTS_BOARD,
    [FactionType.Halflings]: HALFLINGS_BOARD,
    [FactionType.Mermaids]: MERMAIDS_BOARD,
    [FactionType.Nomads]: NOMADS_BOARD,
    [FactionType.Swarmlings]: SWARMLINGS_BOARD,

    // Defaults for others
    [FactionType.Witches]: { dwellings: STANDARD_DWELLINGS, tradingHouses: STANDARD_TRADING_HOUSES, temples: STANDARD_TEMPLES, sanctuary: STANDARD_SANCTUARY, stronghold: STANDARD_STRONGHOLD },
    [FactionType.Dwarves]: { dwellings: STANDARD_DWELLINGS, tradingHouses: STANDARD_TRADING_HOUSES, temples: STANDARD_TEMPLES, sanctuary: STANDARD_SANCTUARY, stronghold: STANDARD_STRONGHOLD },
    [FactionType.Alchemists]: { dwellings: STANDARD_DWELLINGS, tradingHouses: STANDARD_TRADING_HOUSES, temples: STANDARD_TEMPLES, sanctuary: STANDARD_SANCTUARY, stronghold: STANDARD_STRONGHOLD },
};
