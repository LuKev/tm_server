// Shared client-side types mirroring Go server models in `server/internal/models`

export enum TerrainType {
  Plains = 0,
  Swamp = 1,
  Lake = 2,
  Forest = 3,
  Mountain = 4,
  Wasteland = 5,
  Desert = 6,
  River = 7,
}

export enum FactionType {
  Unknown = 0,
  Nomads = 1,
  Fakirs = 2,
  ChaosMagicians = 3,
  Giants = 4,
  Swarmlings = 5,
  Mermaids = 6,
  Witches = 7,
  Auren = 8,
  Halflings = 9,
  Cultists = 10,
  Alchemists = 11,
  Darklings = 12,
  Engineers = 13,
  Dwarves = 14,
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

export enum PowerActionType {
  Bridge = 0,
  Priest = 1,
  Workers = 2,
  Coins = 3,
  Spade = 4,
  DoubleSpade = 5,
}

export enum FavorTileType {
  Fire3 = 0,
  Water3 = 1,
  Earth3 = 2,
  Air3 = 3,
  Fire2 = 4,
  Water2 = 5,
  Earth2 = 6,
  Air2 = 7,
  Fire1 = 8,
  Water1 = 9,
  Earth1 = 10,
  Air1 = 11,
}

export enum TownTileId {
  Vp5Coins6 = 0,
  Vp6Power8 = 1,
  Vp7Workers2 = 2,
  Vp4Ship1 = 3,
  Vp8Cult1 = 4,
  Vp9Priest1 = 5,
  Vp11 = 6,
  Vp2Cult2 = 7,
}

export interface Resources {
  coins: number
  workers: number
  priests: number
  power: {
    powerI: number
    powerII: number
    powerIII: number
  }
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
  bridges: Bridge[];
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
  victoryPoints?: number
  VictoryPoints?: number
  Faction?: FactionType | { Type: FactionType }
  specialActionsUsed?: Record<number, boolean>
}

export enum SpecialActionType {
  AurenCultAdvance = 0,
  WitchesRide = 1,
  AlchemistsConvert = 2,
  SwarmlingsUpgrade = 3,
  ChaosMagiciansDoubleTurn = 4,
  GiantsTransform = 5,
  NomadsSandstorm = 6,
  Water2CultAdvance = 7,
  BonusCardSpade = 8,
  BonusCardCultAdvance = 9,
}

export interface CultTrackState {
  playerPositions: Record<string, Record<CultType, number>>
  position10Occupied: Record<CultType, string>
  bonusPositionsClaimed: Record<string, Record<CultType, Record<number, boolean>>>
  priestsOnActionSpaces: Record<string, Record<CultType, number>>
  priestsOnTrack: Record<CultType, Record<number, string[]>>
}

export interface RoundState {
  round: number // 1..6
}

export enum GamePhase {
  Setup = 0,
  FactionSelection = 1,
  Income = 2,
  Action = 3,
  Cleanup = 4,
  End = 5,
}

export enum BonusCardType {
  Priest = 0,
  Shipping = 1,
  DwellingVP = 2,
  WorkerPower = 3,
  Spade = 4,
  TradingHouseVP = 5,
  Coins6 = 6,
  CultAdvance = 7,
  StrongholdSanctuaryVP = 8,
  ShippingVP = 9,
}

export interface BonusCardState {
  available: Record<BonusCardType, number>
  playerCards: Record<string, BonusCardType>
  playerHasCard: Record<string, boolean>
}

export interface ScoringTile {
  type: number
  actionType: number
  actionVP: number
  cultTrack: number
  cultThreshold: number
  cultRewardType: number
  cultRewardAmount: number
}

export interface ScoringTileState {
  tiles: ScoringTile[]
  priestsSent: Record<string, number>
}

export interface FavorTileState {
  available: Record<number, number>
  playerTiles: Record<string, FavorTileType[]>
}

export interface TownTileState {
  available: Record<number, number>
}

export interface GameState {
  id: string
  phase: GamePhase
  players: Record<string, PlayerState>
  turnOrder: string[]
  passOrder?: string[]
  currentTurn: number
  map: MapState
  round: RoundState
  started: boolean
  finished: boolean
  scoringTiles?: ScoringTileState
  townTiles?: TownTileState
  favorTiles?: FavorTileState
  bonusCards?: BonusCardState
  finalScoring?: Record<string, PlayerFinalScore>
  powerActions?: PowerActionState
  cultTracks?: CultTrackState
}

export interface PowerActionState {
  UsedActions: Record<number, boolean>
}

export interface PlayerFinalScore {
  playerId: string
  playerName: string
  baseVp: number
  areaVp: number
  cultVp: number
  resourceVp: number
  totalVp: number
  largestAreaSize: number
  totalResourceValue: number
}
