// Shared client-side types mirroring Go server models in `server/internal/models`

export enum TerrainType {
  Desert = 0,
  Plains = 1,
  Swamp = 2,
  Lake = 3,
  Forest = 4,
  Mountain = 5,
  Wasteland = 6,
}

export enum FactionType {
  Nomads = 0,
  Fakirs = 1,
  ChaosMagicians = 2,
  Giants = 3,
  Swarmlings = 4,
  Mermaids = 5,
  Witches = 6,
  Auren = 7,
  Halflings = 8,
  Cultists = 9,
  Alchemists = 10,
  Darklings = 11,
  Engineers = 12,
  Dwarves = 13,
}

export enum BuildingType {
  Dwelling = 0,
  TradingHouse = 1,
  Temple = 2,
  Sanctuary = 3,
  Stronghold = 4,
}

export enum CultType {
  Fire = 0,
  Water = 1,
  Earth = 2,
  Air = 3,
}

export interface Resources {
  coins: number
  workers: number
  priests: number
  powerI: number
  powerII: number
  powerIII: number
}

export interface HexCoord {
  q: number
  r: number
}

export interface Building {
  ownerPlayerId: string
  faction: FactionType
  type: BuildingType
}

export interface Bridge {
  ownerPlayerId: string
  faction: FactionType
  fromCoord: HexCoord
  toCoord: HexCoord
}

export interface MapHex {
  coord: HexCoord
  terrain: TerrainType
  building?: Building
}

export interface MapState {
  // key: "q,r" string
  hexes: Record<string, MapHex>
}

export interface PlayerState {
  id: string
  name: string
  faction: FactionType
  resources: Resources
  shipping: number
  digging: number
  cults: Partial<Record<CultType, number>>
  buildings: Record<string, Building>
}

export interface RoundState {
  round: number // 1..6
}

export enum GamePhase {
  Setup = 0,
  FactionSelection = 1,
  Game = 2,
  End = 3,
}

export interface GameState {
  id: string
  phase: GamePhase
  players: Record<string, PlayerState>
  order: string[]
  activeIndex: number
  map: MapState
  round: RoundState
  started: boolean
  finished: boolean
}
