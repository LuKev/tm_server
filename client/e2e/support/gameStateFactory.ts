import { BASE_GAME_MAP } from '../../src/data/baseGameMap'
import {
  BonusCardType,
  BuildingType,
  CultType,
  FactionType,
  GamePhase,
  type GameState,
  type MapHex,
  type PlayerState,
  TerrainType,
} from '../../src/types/game.types'

type BuildingSeed = {
  q: number
  r: number
  ownerPlayerId: string
  faction: FactionType
  type: BuildingType
  terrain?: TerrainType
}

const defaultPlayer = (
  id: string,
  name: string,
  faction: FactionType,
): PlayerState => ({
  id,
  name,
  faction,
  resources: {
    coins: 20,
    workers: 10,
    priests: 3,
    power: {
      powerI: 5,
      powerII: 7,
      powerIII: 3,
    },
  },
  shipping: 0,
  digging: 0,
  cults: {
    [CultType.Fire]: 0,
    [CultType.Water]: 0,
    [CultType.Earth]: 0,
    [CultType.Air]: 0,
  },
  buildings: {},
  hasPassed: false,
  hasStrongholdAbility: false,
  specialActionsUsed: {},
  victoryPoints: 20,
  name,
  options: {
    autoLeechMode: 'off',
    autoConvertOnPass: false,
    confirmActions: true,
    showIncomePreview: false,
  },
})

const buildBaseMapHexes = (): Record<string, MapHex> => {
  const out: Record<string, MapHex> = {}
  BASE_GAME_MAP.forEach((hex) => {
    const key = `${String(hex.coord.q)},${String(hex.coord.r)}`
    out[key] = {
      coord: { ...hex.coord },
      terrain: hex.terrain,
    }
  })
  return out
}

export const withBuildings = (
  state: GameState,
  buildings: BuildingSeed[],
): GameState => {
  const next = structuredClone(state)
  buildings.forEach((building) => {
    const key = `${String(building.q)},${String(building.r)}`
    if (!next.map.hexes[key]) {
      next.map.hexes[key] = {
        coord: { q: building.q, r: building.r },
        terrain: building.terrain ?? TerrainType.Plains,
      }
    }
    if (building.terrain !== undefined) {
      next.map.hexes[key].terrain = building.terrain
    }
    next.map.hexes[key].building = {
      ownerPlayerId: building.ownerPlayerId,
      faction: building.faction,
      type: building.type,
    }
  })
  return next
}

export const makeBaseGameState = (overrides: Partial<GameState> = {}): GameState => {
  const players = overrides.players ?? {
    p1: defaultPlayer('p1', 'p1', FactionType.Nomads),
    p2: defaultPlayer('p2', 'p2', FactionType.Darklings),
    p3: defaultPlayer('p3', 'p3', FactionType.Mermaids),
    p4: defaultPlayer('p4', 'p4', FactionType.Witches),
  }

  const state: GameState = {
    id: 'test-game',
    revision: 1,
    phase: GamePhase.Action,
    setupMode: 'snellman',
    players,
    turnOrder: ['p1', 'p2', 'p3', 'p4'],
    currentTurn: 0,
    map: {
      hexes: buildBaseMapHexes(),
      bridges: [],
    },
    round: {
      round: 1,
    },
    started: true,
    finished: false,
    scoringTiles: {
      tiles: [],
      priestsSent: {},
    },
    townTiles: {
      available: {
        0: 2,
        1: 2,
        2: 2,
        3: 2,
        4: 2,
        5: 2,
        6: 1,
        7: 1,
      },
    },
    favorTiles: {
      available: {
        0: 1,
        1: 1,
        2: 1,
        3: 1,
        4: 3,
        5: 3,
        6: 3,
        7: 3,
        8: 2,
        9: 2,
        10: 2,
        11: 2,
      },
      playerTiles: {
        p1: [],
        p2: [],
        p3: [],
        p4: [],
      },
    },
    bonusCards: {
      available: {
        [BonusCardType.Priest]: 0,
        [BonusCardType.Shipping]: 0,
        [BonusCardType.DwellingVP]: 0,
        [BonusCardType.WorkerPower]: 0,
        [BonusCardType.Spade]: 0,
        [BonusCardType.TradingHouseVP]: 0,
        [BonusCardType.Coins6]: 0,
        [BonusCardType.CultAdvance]: 0,
        [BonusCardType.StrongholdSanctuaryVP]: 0,
        [BonusCardType.ShippingVP]: 0,
      },
      playerCards: {},
      playerHasCard: {},
    },
    powerActions: {
      UsedActions: {},
    },
    cultTracks: {
      playerPositions: {},
      position10Occupied: {
        [CultType.Fire]: '',
        [CultType.Water]: '',
        [CultType.Earth]: '',
        [CultType.Air]: '',
      },
      bonusPositionsClaimed: {},
      priestsOnActionSpaces: {},
      priestsOnTrack: {
        [CultType.Fire]: { 1: [], 2: [], 3: [] },
        [CultType.Water]: { 1: [], 2: [], 3: [] },
        [CultType.Earth]: { 1: [], 2: [], 3: [] },
        [CultType.Air]: { 1: [], 2: [], 3: [] },
      },
    },
    pendingDecision: null,
    auctionState: null,
  }

  const merged = {
    ...state,
    ...overrides,
    map: {
      ...state.map,
      ...(overrides.map ?? {}),
      hexes: {
        ...state.map.hexes,
        ...(overrides.map?.hexes ?? {}),
      },
      bridges: overrides.map?.bridges ?? state.map.bridges,
    },
    players,
    turnOrder: overrides.turnOrder ?? state.turnOrder,
  }

  return merged
}
