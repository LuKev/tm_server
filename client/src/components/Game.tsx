import { useParams } from 'react-router-dom'
import { useMemo, useEffect, useState } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { ScoringTiles } from './GameBoard/ScoringTiles'
import { TownTiles } from './GameBoard/TownTiles'
import { FavorTiles } from './GameBoard/FavorTiles'
import { PassingTiles } from './GameBoard/PassingTiles'
import { PlayerBoards } from './GameBoard/PlayerBoards'
import { PlayerSummaryBar } from './GameBoard/PlayerSummaryBar'
import { CultTracks, type PriestSpot } from './CultTracks/CultTracks'
import { FactionSelector } from './FactionSelector'
import { FACTIONS } from '../data/factions'
import { useGameStore } from '../stores/gameStore'
import { useActionService } from '../services/actionService'
import {
  CultType,
  GamePhase,
  PowerActionType,
  SpecialActionType,
  BuildingType,
  TerrainType,
  BonusCardType,
  FactionType,
  type LeechAutoMode,
  type PlayerOptions,
  type FavorTileType,
  type TownTileId,
} from '../types/game.types'
import { useWebSocket } from '../services/WebSocketContext'
import { Responsive, WidthProvider } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'
import { getCultPositions, resolveFaction } from '../utils/gameUtils'
import { useGameLayout } from '../hooks/useGameLayout'
import { Modal } from './shared/Modal'
import { buildDisplayCoordinateMap, formatDisplayCoordinate } from '../utils/hexUtils'

const ResponsiveGridLayout = WidthProvider(Responsive)

type ConfirmDialog = {
  title: string
  message: string
  onConfirm?: () => void
  actions?: Array<{
    label: string
    onClick: () => void
    testId?: string
    className?: string
  }>
}

const DEFAULT_PLAYER_OPTIONS: PlayerOptions = {
  autoLeechMode: 'off',
  autoConvertOnPass: false,
  confirmActions: true,
  showIncomePreview: false,
}

const LEECH_AUTO_OPTIONS: Array<{ value: LeechAutoMode; label: string }> = [
  { value: 'off', label: 'Auto accept power: Off' },
  { value: 'accept_1', label: 'Auto accept up to 1 power (0 VP)' },
  { value: 'accept_2', label: 'Auto accept up to 2 power (1 VP)' },
  { value: 'accept_3', label: 'Auto accept up to 3 power (2 VP)' },
  { value: 'accept_4', label: 'Auto accept up to 4 power (3 VP)' },
  { value: 'decline_vp', label: 'Auto decline VP-cost leech (accept only 1/0)' },
]

const STRONGHOLD_TARGET_ACTION_TYPES: SpecialActionType[] = [
  SpecialActionType.WitchesRide,
  SpecialActionType.SwarmlingsUpgrade,
  SpecialActionType.GiantsTransform,
  SpecialActionType.NomadsSandstorm,
  SpecialActionType.MermaidsRiverTown,
]

type PendingPowerMode =
  | { type: 'power_spade'; actionType: PowerActionType; useCoins?: boolean }
  | { type: 'power_bridge'; source: 'power' | 'engineers'; firstHex: { q: number; r: number } | null; useCoins?: boolean }
  | {
    type: 'special_action_target'
    actionType: SpecialActionType
  }

const TERRAIN_CHOICES: Array<{ id: TerrainType; name: string }> = [
  { id: TerrainType.Plains, name: 'Plains' },
  { id: TerrainType.Swamp, name: 'Swamp' },
  { id: TerrainType.Lake, name: 'Lake' },
  { id: TerrainType.Forest, name: 'Forest' },
  { id: TerrainType.Mountain, name: 'Mountain' },
  { id: TerrainType.Wasteland, name: 'Wasteland' },
  { id: TerrainType.Desert, name: 'Desert' },
]

const CULT_CHOICES: Array<{ track: CultType; label: string }> = [
  { track: CultType.Fire, label: 'Fire' },
  { track: CultType.Water, label: 'Water' },
  { track: CultType.Earth, label: 'Earth' },
  { track: CultType.Air, label: 'Air' },
]

const CHAOS_ACTION_TYPES = [
  'transform_build',
  'upgrade_building',
  'advance_shipping',
  'advance_digging',
  'advance_chash_track',
  'send_priest',
  'power_action_claim',
  'special_action_use',
  'pass',
  'conversion',
  'burn_power',
]

const CHAOS_PARAM_TEMPLATES: Record<string, string> = {
  transform_build: '{\n  "targetHex": { "q": 0, "r": 0 },\n  "buildDwelling": false,\n  "targetTerrain": 0\n}',
  upgrade_building: '{\n  "targetHex": { "q": 0, "r": 0 },\n  "newBuildingType": 2\n}',
  advance_shipping: '{}',
  advance_digging: '{}',
  advance_chash_track: '{}',
  send_priest: '{\n  "cultTrack": 0,\n  "spacesToClimb": 1\n}',
  power_action_claim: '{\n  "actionType": 1\n}',
  special_action_use: '{\n  "specialActionType": 0\n}',
  pass: '{\n  "bonusCard": 0\n}',
  conversion: '{\n  "conversionType": "worker_to_coin",\n  "amount": 1\n}',
  burn_power: '{\n  "amount": 1\n}',
}

const getPowerActionCost = (action: PowerActionType): number => {
  switch (action) {
    case PowerActionType.Bridge:
    case PowerActionType.Priest:
      return 3
    case PowerActionType.Workers:
    case PowerActionType.Coins:
    case PowerActionType.Spade:
      return 4
    case PowerActionType.DoubleSpade:
      return 6
    default:
      return 0
  }
}

const canPayPowerActionWithPower = (player: PlayerState | null, action: PowerActionType): boolean => {
  if (!player) return false
  const cost = getPowerActionCost(action)
  const bowl3 = player.resources.power.powerIII ?? 0
  const bowl2 = player.resources.power.powerII ?? 0
  return bowl3 + Math.floor(bowl2 / 2) >= cost
}

const shouldAutoUseChashCoins = (player: PlayerState | null, action: PowerActionType): boolean => {
  if (!player || player.faction !== FactionType.ChashDallah || !player.hasStrongholdAbility) return false
  const cost = getPowerActionCost(action)
  return !canPayPowerActionWithPower(player, action) && (player.resources.coins ?? 0) >= cost
}

const canPayPowerActionWithCoins = (player: PlayerState | null, action: PowerActionType): boolean => {
  if (!player || player.faction !== FactionType.ChashDallah || !player.hasStrongholdAbility) return false
  return (player.resources.coins ?? 0) >= getPowerActionCost(action)
}

const townTileLabel = (id: number): string => {
  const map: Record<number, string> = {
    0: '5 VP + 6 coins',
    1: '6 VP + 8 power',
    2: '7 VP + 2 workers',
    3: '4 VP + shipping',
    4: '8 VP + 1 cult all',
    5: '9 VP + 1 priest',
    6: '11 VP',
    7: '2 VP + 2 cult all',
  }
  return map[id] ?? `Town ${id}`
}

const favorTileLabel = (id: number): string => {
  const map: Record<number, string> = {
    0: 'Fire +3',
    1: 'Water +3',
    2: 'Earth +3',
    3: 'Air +3',
    4: 'Fire +2',
    5: 'Water +2',
    6: 'Earth +2',
    7: 'Air +2',
    8: 'Fire +1',
    9: 'Water +1',
    10: 'Earth +1',
    11: 'Air +1',
  }
  return map[id] ?? `Favor ${id}`
}

const bonusCardLabel = (id: number): string => {
  const map: Record<number, string> = {
    0: 'Priest Income',
    1: 'Shipping Bonus',
    2: 'Dwelling VP',
    3: 'Worker + Power',
    4: 'Spade',
    5: 'Trading House VP',
    6: '6 Coins',
    7: 'Cult Advance',
    8: 'Stronghold/Sanctuary VP',
    9: 'Shipping VP',
  }
  return map[id] ?? `Bonus ${id}`
}

const isActionPhase = (phase?: GamePhase): boolean => phase === GamePhase.Action

const formatActorList = (
  playerIds: string[],
  players: Record<string, { name?: string } | undefined> | undefined,
  localPlayerId: string | null | undefined,
): string => {
  const uniqueIds = playerIds.filter((playerId, index) => playerId && playerIds.indexOf(playerId) === index)
  const labels = uniqueIds.map((playerId) => {
    if (playerId === localPlayerId) return 'You'
    const player = players?.[playerId]
    const displayName = typeof player?.name === 'string' ? player.name.trim() : ''
    return displayName || playerId
  })

  if (labels.length === 0) return 'Unknown player'
  if (labels.length === 1) return labels[0]
  if (labels.length === 2) return `${labels[0]} and ${labels[1]}`
  return `${labels.slice(0, -1).join(', ')}, and ${labels[labels.length - 1]}`
}

export const Game = () => {
  const { gameId } = useParams()
  const { isConnected, sendMessage, lastMessage } = useWebSocket()
  const gameState = useGameStore((state) => state.gameState)
  const localPlayerId = useGameStore((state) => state.localPlayerId)

  const { submitAction, submitSetupDwelling, submitSelectFaction } = useActionService()

  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialog | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [powerMode, setPowerMode] = useState<PendingPowerMode | null>(null)

  const [pendingHex, setPendingHex] = useState<{ q: number; r: number } | null>(null)
  const [hexActionMode, setHexActionMode] = useState<'build' | 'transform_build' | 'transform_only' | null>(null)
  const [selectedTerrain, setSelectedTerrain] = useState<TerrainType>(TerrainType.Plains)

  const [upgradeHex, setUpgradeHex] = useState<{ q: number; r: number } | null>(null)
  const [chaosModalOpen, setChaosModalOpen] = useState(false)
  const [chaosFirstType, setChaosFirstType] = useState<string>('transform_build')
  const [chaosSecondType, setChaosSecondType] = useState<string>('transform_build')
  const [chaosFirstParams, setChaosFirstParams] = useState<string>('{}')
  const [chaosSecondParams, setChaosSecondParams] = useState<string>('{}')
  const [selectedTownCultTracks, setSelectedTownCultTracks] = useState<CultType[]>([])
  const [auctionBidInputs, setAuctionBidInputs] = useState<Record<string, number>>({})
  const [fastAuctionBidInputs, setFastAuctionBidInputs] = useState<Record<string, number>>({})
  const [conspiratorsSwapModalOpen, setConspiratorsSwapModalOpen] = useState(false)
  const [conspiratorsReturnTile, setConspiratorsReturnTile] = useState<number | ''>('')
  const [conspiratorsNewTile, setConspiratorsNewTile] = useState<number | ''>('')
  const [cultChoiceContext, setCultChoiceContext] = useState<
    | 'cultists'
    | 'water2'
    | 'auren_sh'
    | 'bonus_cult'
    | null
  >(null)

  const chaosFirstParamsError = useMemo(() => {
    try {
      JSON.parse(chaosFirstParams)
      return null
    } catch {
      return 'Invalid JSON'
    }
  }, [chaosFirstParams])

  const chaosSecondParamsError = useMemo(() => {
    try {
      JSON.parse(chaosSecondParams)
      return null
    } catch {
      return 'Invalid JSON'
    }
  }, [chaosSecondParams])

  const applyChaosTemplate = (slot: 'first' | 'second', actionType: string): void => {
    const template = CHAOS_PARAM_TEMPLATES[actionType] ?? '{}'
    if (slot === 'first') {
      setChaosFirstParams(template)
      return
    }
    setChaosSecondParams(template)
  }

  const numCards = useMemo(() => {
    if (!gameState?.bonusCards) return 9
    const available = Object.keys(gameState.bonusCards.available ?? {}).length
    const taken = Object.keys(gameState.bonusCards.playerCards ?? {}).length
    return available + taken
  }, [gameState?.bonusCards])

  const {
    layouts,
    rowHeight,
    handleWidthChange,
    handleLayoutChange,
    isLayoutLocked,
    setIsLayoutLocked,
    resetLayout,
  } = useGameLayout(gameState, numCards, 'game')

  useEffect(() => {
    if (isConnected && gameId && (!gameState || gameState.id !== gameId)) {
      sendMessage({ type: 'get_game_state', payload: { gameID: gameId, playerID: localPlayerId } })
    }
  }, [isConnected, gameId, gameState, localPlayerId, sendMessage])

  useEffect(() => {
    if (!import.meta.env.DEV || typeof window === 'undefined') return
    const testWindow = window as Window & {
      __TM_TEST_GET_REVISION__?: () => number | null
      __TM_TEST_GET_LOCAL_PLAYER_ID__?: () => string | null
      __TM_TEST_SET_LOCAL_PLAYER_ID__?: (playerId: string) => void
    }
    testWindow.__TM_TEST_GET_REVISION__ = () => {
      const revision = useGameStore.getState().gameState?.revision
      return typeof revision === 'number' ? revision : null
    }
    testWindow.__TM_TEST_GET_LOCAL_PLAYER_ID__ = () => useGameStore.getState().localPlayerId
    testWindow.__TM_TEST_SET_LOCAL_PLAYER_ID__ = (playerId: string) => {
      useGameStore.getState().setLocalPlayerId(playerId)
      localStorage.setItem('tm-game-storage', JSON.stringify({ state: { localPlayerId: playerId }, version: 0 }))
    }
    return () => {
      testWindow.__TM_TEST_GET_REVISION__ = undefined
      testWindow.__TM_TEST_GET_LOCAL_PLAYER_ID__ = undefined
      testWindow.__TM_TEST_SET_LOCAL_PLAYER_ID__ = undefined
    }
  }, [])

  useEffect(() => {
    if (!lastMessage || typeof lastMessage !== 'object' || !('type' in lastMessage)) return

    const msg = lastMessage as { type: string; payload?: unknown }
    if (msg.type === 'action_rejected') {
      const payload = (msg.payload ?? {}) as Record<string, unknown>
      const text = (payload.message as string | undefined) ?? (payload.error as string | undefined) ?? 'Action rejected'
      setErrorMessage(text)
      return
    }

    if (msg.type === 'error') {
      const text = typeof msg.payload === 'string' ? msg.payload : 'Server error'
      setErrorMessage(text)
      return
    }

    if (msg.type === 'action_accepted') {
      setErrorMessage(null)
    }
  }, [lastMessage])

  useEffect(() => {
    if (!errorMessage) return
    const t = setTimeout(() => {
      setErrorMessage(null)
    }, 4500)
    return () => {
      clearTimeout(t)
    }
  }, [errorMessage])

  const queueConfirm = (
    title: string,
    message: string,
    onConfirm: () => void,
    scope: 'interaction' | 'turn_end' = 'interaction',
  ): void => {
    const needsConfirmation = localPlayerOptions.confirmActions && scope === 'turn_end'
    if (!needsConfirmation) {
      onConfirm()
      return
    }
    setConfirmDialog({ title, message, onConfirm })
  }

  const openChoiceDialog = (
    title: string,
    message: string,
    actions: ConfirmDialog['actions'],
  ): void => {
    setConfirmDialog({ title, message, actions })
  }

  const performAction = (type: string, params: Record<string, unknown> = {}): void => {
    if (!gameId) return
    submitAction(gameId, type, params)
  }

  const currentPlayerId = gameState?.turnOrder?.[gameState.currentTurn]
  const isMyTurn = !!localPlayerId && currentPlayerId === localPlayerId
  const setupMode = (gameState?.setupMode ?? 'snellman') as 'snellman' | 'auction' | 'fast_auction'
  const auctionState = (gameState?.auctionState ?? null) as
    | {
      nominationOrder?: string[]
      currentBids?: Record<string, number>
      factionHolders?: Record<string, string>
      currentBidder?: string
      fastSubmitted?: Record<string, boolean>
      fastBids?: Record<string, Record<string, number>>
    }
    | null

  const localPlayer = useMemo(() => {
    if (!localPlayerId || !gameState?.players) return null
    return gameState.players[localPlayerId] ?? null
  }, [gameState?.players, localPlayerId])
  const isSpectator = gameState != null && localPlayer == null
  const localPlayerFavorTiles = useMemo(() => {
    if (!localPlayerId) return [] as FavorTileType[]
    return (gameState?.favorTiles?.playerTiles?.[localPlayerId] ?? []) as FavorTileType[]
  }, [gameState?.favorTiles?.playerTiles, localPlayerId])
  const availableFavorTileTypes = useMemo(() => {
    return Object.entries(gameState?.favorTiles?.available ?? {})
      .map(([tileId, count]) => ({ tileId: Number(tileId), count }))
      .filter(({ tileId, count }) => Number.isInteger(tileId) && count > 0)
      .map(({ tileId }) => tileId as FavorTileType)
      .sort((a, b) => a - b)
  }, [gameState?.favorTiles?.available])

  const localPlayerOptions = useMemo((): PlayerOptions => {
    if (!localPlayer?.options) return DEFAULT_PLAYER_OPTIONS
    return {
      autoLeechMode: localPlayer.options.autoLeechMode ?? DEFAULT_PLAYER_OPTIONS.autoLeechMode,
      autoConvertOnPass: localPlayer.options.autoConvertOnPass ?? DEFAULT_PLAYER_OPTIONS.autoConvertOnPass,
      confirmActions: localPlayer.options.confirmActions ?? DEFAULT_PLAYER_OPTIONS.confirmActions,
      showIncomePreview: localPlayer.options.showIncomePreview ?? DEFAULT_PLAYER_OPTIONS.showIncomePreview,
    }
  }, [localPlayer])

  const updatePlayerOptions = (patch: Partial<PlayerOptions>): void => {
    performAction('set_player_options', patch as Record<string, unknown>)
  }

  useEffect(() => {
    if (!localPlayerOptions.confirmActions) {
      setConfirmDialog(null)
    }
  }, [localPlayerOptions.confirmActions])

  const localFactionType = useMemo(() => {
    if (!localPlayer) return FactionType.Unknown
    const raw = (localPlayer.faction ?? localPlayer.Faction) as unknown
    return resolveFaction(raw)
  }, [localPlayer])

  const localHomeTerrain = useMemo(() => {
    const faction = FACTIONS.find((f) => f.id === localFactionType)
    if (!faction) return TerrainType.Plains
    const terrain = faction.homeTerrain.toLowerCase()
    if (terrain.startsWith('plain')) return TerrainType.Plains
    if (terrain.startsWith('swamp')) return TerrainType.Swamp
    if (terrain.startsWith('lake')) return TerrainType.Lake
    if (terrain.startsWith('forest')) return TerrainType.Forest
    if (terrain.startsWith('mountain')) return TerrainType.Mountain
    if (terrain.startsWith('wasteland')) return TerrainType.Wasteland
    if (terrain.startsWith('desert')) return TerrainType.Desert
    return TerrainType.Plains
  }, [localFactionType])

  const pendingDecision = (gameState?.pendingDecision ?? null) as Record<string, unknown> | null
  const pendingDecisionType = (pendingDecision?.type as string | undefined) ?? null
  const pendingDecisionPlayerId = (pendingDecision?.playerId as string | undefined) ?? null
  const pendingDecisionPlayerIds = useMemo(() => {
    const raw = (pendingDecision?.playerIds as unknown[]) ?? []
    return raw
      .map((value) => (typeof value === 'string' ? value : ''))
      .filter((value) => value.length > 0)
  }, [pendingDecision])
  const pendingAuctionFactions = useMemo(() => {
    const raw = (pendingDecision?.nominatedFactions as unknown[]) ?? []
    return raw
      .map((value) => (typeof value === 'string' ? value : ''))
      .filter((value) => value.length > 0)
  }, [pendingDecision])
  const hasPendingDecisionForMe = !!localPlayerId
    && (pendingDecisionPlayerId === localPlayerId || pendingDecisionPlayerIds.includes(localPlayerId))
  const isFastAuctionBidDecisionForMe = pendingDecisionType === 'fast_auction_bid_matrix'
    && !!localPlayerId
    && !(auctionState?.fastSubmitted?.[localPlayerId] ?? false)
  const isOtherPlayerTurnConfirmationWindow = !!pendingDecisionPlayerId
    && pendingDecisionPlayerId !== localPlayerId
    && (pendingDecisionType === 'post_action_free_actions' || pendingDecisionType === 'turn_confirmation')
  const isBlockingPendingDecisionForMe = hasPendingDecisionForMe || isOtherPlayerTurnConfirmationWindow
  const displayCoordinates = useMemo(() => {
    const hexes = Object.values(gameState?.map?.hexes ?? {}).map((hex) => ({
      coord: hex.coord,
      isRiver: hex.terrain === TerrainType.River,
      displayCoord: hex.displayCoord,
    }))
    return buildDisplayCoordinateMap(hexes)
  }, [gameState?.map?.hexes])
  const formatHexCoord = (coord: { q: number; r: number }): string => formatDisplayCoordinate(coord, displayCoordinates)
  const canInitiateTurnAction = isMyTurn && isActionPhase(gameState?.phase) && !isBlockingPendingDecisionForMe
  const pendingTownCultTopCandidates = useMemo(() => {
    if (pendingDecisionType !== 'town_cult_top_choice') return [] as CultType[]
    const raw = (pendingDecision?.candidateTracks as unknown[]) ?? []
    return raw
      .map((v) => Number(v))
      .filter((v): v is CultType => Number.isInteger(v) && v >= 0 && v <= 3)
  }, [pendingDecision, pendingDecisionType])
  const pendingTownCultTopMaxSelections = useMemo(() => {
    if (pendingDecisionType !== 'town_cult_top_choice') return 0
    return Number(pendingDecision?.maxSelections ?? 0)
  }, [pendingDecision, pendingDecisionType])
  const pendingLeechOffersForMe = useMemo(() => {
    if (!localPlayerId) return [] as Array<Record<string, unknown>>
    const pending = gameState?.pendingLeechOffers ?? {}
    const offers = pending[localPlayerId]
    if (Array.isArray(offers) && offers.length > 0) {
      return offers as Array<Record<string, unknown>>
    }
    if (hasPendingDecisionForMe && pendingDecisionType === 'leech_offer') {
      return ((pendingDecision?.offers as Array<Record<string, unknown>> | undefined) ?? [])
    }
    return [] as Array<Record<string, unknown>>
  }, [gameState?.pendingLeechOffers, hasPendingDecisionForMe, localPlayerId, pendingDecision, pendingDecisionType])
  const pendingLeechResponders = useMemo(() => {
    const pending = gameState?.pendingLeechOffers ?? {}
    return Object.entries(pending)
      .filter(([, offers]) => Array.isArray(offers) && offers.length > 0)
      .map(([playerId]) => playerId)
  }, [gameState?.pendingLeechOffers])
  const waitingOnOtherLeechResponses = useMemo(() => {
    if (!localPlayerId || pendingLeechResponders.length === 0) return 0
    if (!isMyTurn) return 0
    if (pendingLeechResponders.includes(localPlayerId)) return 0
    return pendingLeechResponders.filter((pid) => pid !== localPlayerId).length
  }, [isMyTurn, localPlayerId, pendingLeechResponders])
  const setupDwellingPlayerId = useMemo(() => {
    if (!gameState?.setupDwellingOrder) return null
    const idx = gameState.setupDwellingIndex ?? -1
    if (idx < 0 || idx >= gameState.setupDwellingOrder.length) return null
    return gameState.setupDwellingOrder[idx] ?? null
  }, [gameState?.setupDwellingOrder, gameState?.setupDwellingIndex])
  const orderedPendingLeechResponders = useMemo(() => {
    if (pendingLeechResponders.length === 0) return [] as string[]
    const turnOrder = gameState?.turnOrder ?? []
    const pendingSet = new Set(pendingLeechResponders)
    const ordered = turnOrder.filter((playerId) => pendingSet.has(playerId))
    pendingLeechResponders.forEach((playerId) => {
      if (!ordered.includes(playerId)) {
        ordered.push(playerId)
      }
    })
    return ordered
  }, [gameState?.turnOrder, pendingLeechResponders])
  const pendingLeechResponderList = useMemo(
    () => formatActorList(orderedPendingLeechResponders, gameState?.players, localPlayerId),
    [gameState?.players, localPlayerId, orderedPendingLeechResponders],
  )
  const decisionStripStatus = useMemo(() => {
    const actorText = (playerIds: string[], singularAction: string, pluralAction = singularAction): string => {
      const uniqueIds = playerIds.filter((playerId, index) => playerId && playerIds.indexOf(playerId) === index)
      if (uniqueIds.length === 0) return ''
      return `${formatActorList(uniqueIds, gameState?.players, localPlayerId)} ${uniqueIds.length > 1 ? pluralAction : singularAction}`
    }

    if (orderedPendingLeechResponders.length > 0) {
      return actorText(orderedPendingLeechResponders, 'must make a leech decision.', 'must make leech decisions.')
    }

    switch (pendingDecisionType) {
      case 'cult_reward_spade':
        return actorText([pendingDecisionPlayerId ?? ''], 'must use cult spades.')
      case 'spade_followup':
        return actorText([pendingDecisionPlayerId ?? ''], 'must resolve pending spades.')
      case 'post_action_free_actions':
        return actorText([pendingDecisionPlayerId ?? ''], 'must finish free actions or confirm the turn.')
      case 'turn_confirmation':
        return actorText([pendingDecisionPlayerId ?? ''], 'must confirm or undo the turn.')
      case 'favor_tile_selection':
        return actorText([pendingDecisionPlayerId ?? ''], 'must select a favor tile.')
      case 'town_tile_selection':
        return actorText([pendingDecisionPlayerId ?? ''], 'must select a town tile.')
      case 'town_cult_top_choice':
        return actorText([pendingDecisionPlayerId ?? ''], 'must choose cult tracks to top.')
      case 'setup_bonus_card':
        return actorText([pendingDecisionPlayerId ?? ''], 'must choose a setup bonus card.')
      case 'darklings_ordination':
        return actorText([pendingDecisionPlayerId ?? ''], 'must choose a Darklings ordination.')
      case 'cultists_cult_choice':
        return actorText([pendingDecisionPlayerId ?? ''], 'must choose a cult track.')
      case 'halflings_spades': {
        const remaining = Number((gameState?.pendingHalflingsSpades as Record<string, unknown> | undefined)?.spadesRemaining ?? 0)
        return actorText([pendingDecisionPlayerId ?? ''], remaining === 0 ? 'must decide whether to build a dwelling.' : 'must use Halflings spades.')
      }
      case 'wisps_stronghold_dwelling':
        return actorText([pendingDecisionPlayerId ?? ''], 'must place the free Wisps dwelling.')
      case 'auction_nomination':
        return actorText([pendingDecisionPlayerId ?? auctionState?.currentBidder ?? ''], 'must nominate a faction.')
      case 'auction_bid':
        return actorText([pendingDecisionPlayerId ?? auctionState?.currentBidder ?? ''], 'must place a bid.')
      case 'fast_auction_bid_matrix':
        return actorText(pendingDecisionPlayerIds, 'must submit bids.', 'must submit bids.')
      default:
        break
    }

    if (gameState?.phase === GamePhase.FactionSelection) {
      if (setupMode === 'snellman' && currentPlayerId) {
        return actorText([currentPlayerId], 'must choose a faction.')
      }
      if (setupMode === 'auction' && (pendingDecisionPlayerId ?? auctionState?.currentBidder)) {
        return actorText([pendingDecisionPlayerId ?? auctionState?.currentBidder ?? ''], 'must continue the auction.')
      }
      if (setupMode === 'fast_auction' && pendingDecisionPlayerIds.length > 0) {
        return actorText(pendingDecisionPlayerIds, 'must submit bids.', 'must submit bids.')
      }
    }

    if (gameState?.phase === GamePhase.Setup && setupDwellingPlayerId) {
      return actorText([setupDwellingPlayerId], 'must place a dwelling.')
    }

    if (isActionPhase(gameState?.phase) && currentPlayerId) {
      return actorText([currentPlayerId], 'must take an action.')
    }

    return 'Waiting for the next required action.'
  }, [
    auctionState?.currentBidder,
    currentPlayerId,
    gameState?.pendingHalflingsSpades,
    gameState?.phase,
    gameState?.players,
    gameState?.turnOrder,
    localPlayerId,
    orderedPendingLeechResponders,
    pendingDecisionPlayerId,
    pendingDecisionPlayerIds,
    pendingDecisionType,
    setupDwellingPlayerId,
    setupMode,
  ])

  useEffect(() => {
    if (!(hasPendingDecisionForMe && pendingDecisionType === 'town_cult_top_choice')) {
      setSelectedTownCultTracks([])
      return
    }
    setSelectedTownCultTracks((current) =>
      current.filter((track) => pendingTownCultTopCandidates.includes(track)),
    )
  }, [hasPendingDecisionForMe, pendingDecisionType, pendingTownCultTopCandidates])

  useEffect(() => {
    if (pendingDecisionType !== 'auction_bid') return
    if (pendingAuctionFactions.length === 0) return
    setAuctionBidInputs((current) => {
      const next = { ...current }
      pendingAuctionFactions.forEach((faction) => {
        if (next[faction] === undefined) {
          next[faction] = 0
        }
      })
      return next
    })
  }, [auctionState?.currentBids, pendingAuctionFactions, pendingDecisionType])

  useEffect(() => {
    if (!isFastAuctionBidDecisionForMe) return
    if (pendingAuctionFactions.length === 0) return
    setFastAuctionBidInputs((current) => {
      const next = { ...current }
      pendingAuctionFactions.forEach((faction) => {
        if (next[faction] === undefined) {
          next[faction] = 0
        }
      })
      return next
    })
  }, [isFastAuctionBidDecisionForMe, pendingAuctionFactions])

  const hasPendingSpadesForMe = useMemo(() => {
    if (!localPlayerId) return 0
    const count = gameState?.pendingSpades?.[localPlayerId]
    return typeof count === 'number' ? count : 0
  }, [gameState?.pendingSpades, localPlayerId])

  const canBuildWithPendingSpadeForMe = useMemo(() => {
    if (!localPlayerId) return true
    const allowed = gameState?.pendingSpadeBuildAllowed?.[localPlayerId]
    return allowed !== false
  }, [gameState?.pendingSpadeBuildAllowed, localPlayerId])

  const hasPendingCultSpadesForMe = useMemo(() => {
    if (!localPlayerId) return 0
    const count = gameState?.pendingCultRewardSpades?.[localPlayerId]
    return typeof count === 'number' ? count : 0
  }, [gameState?.pendingCultRewardSpades, localPlayerId])

  useEffect(() => {
    if (hasPendingSpadesForMe > 0 && !canBuildWithPendingSpadeForMe && hexActionMode !== 'transform_only') {
      setHexActionMode('transform_only')
    }
  }, [canBuildWithPendingSpadeForMe, hasPendingSpadesForMe, hexActionMode])

  const hasUnspentOptionalActions = useMemo(() => {
    if (!localPlayer || !localPlayerId) return false
    const used = localPlayer.specialActionsUsed ?? {}
    const card = gameState?.bonusCards?.playerCards?.[localPlayerId]
    const hasUnusedBonusSpade = card === BonusCardType.Spade && !used[SpecialActionType.BonusCardSpade]
    const hasUnusedBonusCult = card === BonusCardType.CultAdvance && !used[SpecialActionType.BonusCardCultAdvance]

    let hasUnusedStronghold = false
    switch (localFactionType) {
      case FactionType.Auren:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.AurenCultAdvance]
        break
      case FactionType.Witches:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.WitchesRide]
        break
      case FactionType.Swarmlings:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.SwarmlingsUpgrade]
        break
      case FactionType.ChaosMagicians:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.ChaosMagiciansDoubleTurn]
        break
      case FactionType.Giants:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.GiantsTransform]
        break
      case FactionType.Nomads:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.NomadsSandstorm]
        break
      case FactionType.TheEnlightened:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.EnlightenedGainPower]
        break
      case FactionType.Conspirators:
        hasUnusedStronghold = !!localPlayer.hasStrongholdAbility && !used[SpecialActionType.ConspiratorsSwapFavor]
        break
      default:
        hasUnusedStronghold = false
        break
    }

    return hasUnusedStronghold || hasUnusedBonusSpade || hasUnusedBonusCult || hasPendingSpadesForMe > 0 || hasPendingCultSpadesForMe > 0
  }, [gameState?.bonusCards?.playerCards, hasPendingCultSpadesForMe, hasPendingSpadesForMe, localFactionType, localPlayer, localPlayerId])

  const bonusCardOwners = useMemo(() => {
    const out: Record<string, string> = {}
    const playerCards = gameState?.bonusCards?.playerCards ?? {}
    Object.entries(playerCards).forEach(([playerID, card]) => {
      out[String(card)] = playerID
    })
    return out
  }, [gameState?.bonusCards?.playerCards])

  const availableCards = useMemo(() => {
    const cardSet = new Set<number>()

    Object.entries(gameState?.bonusCards?.available ?? {}).forEach(([k]) => {
      const card = Number(k)
      if (Number.isInteger(card) && card >= 0) {
        cardSet.add(card)
      }
    })

    Object.values(gameState?.bonusCards?.playerCards ?? {}).forEach((cardRaw) => {
      const card = Number(cardRaw)
      if (Number.isInteger(card) && card >= 0) {
        cardSet.add(card)
      }
    })

    return [...cardSet].sort((a, b) => a - b)
  }, [gameState?.bonusCards?.available, gameState?.bonusCards?.playerCards])

  const passedPlayers = useMemo(() => {
    const passed = new Set<string>()
    const players = gameState?.players ?? {}
    Object.entries(players).forEach(([id, player]) => {
      if (player.hasPassed) {
        passed.add(id)
      }
    })
    return passed
  }, [gameState?.players])

  const selectedFactionsMap = useMemo(() => {
    const map = new Map<string, { playerNumber: number; vp: number }>()

    if (!gameState?.players || !gameState.turnOrder) return map

    gameState.turnOrder.forEach((playerId: string, index: number) => {
      const player = gameState.players[playerId]
      if (!player) return

      const factionRaw = player.faction ?? player.Faction
      if (factionRaw !== undefined) {
        const factionId = resolveFaction(factionRaw)
        const factionType = FACTIONS.find((f) => f.id === factionId)?.type
        if (factionType) {
          map.set(factionType, {
            playerNumber: index + 1,
            vp: player.victoryPoints ?? player.VictoryPoints ?? 20,
          })
        }
      }
    })

    return map
  }, [gameState])

  const availableAuctionNominationFactions = useMemo(() => {
    if (setupMode === 'snellman') return [] as string[]
    const nominated = new Set((auctionState?.nominationOrder ?? []).map((value) => String(value)))
    const nominatedColors = new Set(
      (auctionState?.nominationOrder ?? [])
        .map((factionType) => FACTIONS.find((f) => f.type === factionType)?.color)
        .filter((value): value is string => !!value),
    )

    return FACTIONS
      .filter((faction) => (gameState?.enableFanFactions ?? false) || !faction.isFanFaction)
      .filter((faction) => !nominated.has(faction.type) && !nominatedColors.has(faction.color))
      .map((faction) => faction.type)
  }, [auctionState?.nominationOrder, gameState?.enableFanFactions, setupMode])

  const currentPlayerPosition = useMemo(() => {
    if (!gameState?.turnOrder || !localPlayerId) return 1
    const index = gameState.turnOrder.indexOf(localPlayerId)
    return index !== -1 ? index + 1 : 1
  }, [gameState, localPlayerId])

  const cultPositions = useMemo(() => {
    if (gameState?.phase === GamePhase.FactionSelection) {
      return new Map([
        [CultType.Fire, []],
        [CultType.Water, []],
        [CultType.Earth, []],
        [CultType.Air, []],
      ])
    }
    return getCultPositions(gameState)
  }, [gameState])

  const priestSpots = useMemo(() => {
    const out = new Map<CultType, PriestSpot[]>()
    const tracks = gameState?.cultTracks?.priestsOnTrack ?? {}
    const players = gameState?.players ?? {}

    const resolveSpotFaction = (playerID?: string): FactionType | undefined => {
      if (!playerID || !players[playerID]) return undefined
      return resolveFaction((players[playerID].faction ?? players[playerID].Faction) as unknown)
    }

    CULT_CHOICES.forEach(({ track }) => {
      const list: PriestSpot[] = [{}, {}, {}, {}, {}]
      const spot3 = (tracks as Record<number, Record<number, string[]>>)[track]?.[3] ?? []
      if (spot3[0]) {
        list[0] = { priests: 1, faction: resolveSpotFaction(spot3[0]) }
      }

      const spot2 = (tracks as Record<number, Record<number, string[]>>)[track]?.[2] ?? []
      for (let i = 0; i < 3; i++) {
        if (spot2[i]) {
          list[i + 1] = { priests: 1, faction: resolveSpotFaction(spot2[i]) }
        }
      }

      const spot1 = (tracks as Record<number, Record<number, string[]>>)[track]?.[1] ?? []
      if (spot1[0]) {
        list[4] = { priests: 1, faction: resolveSpotFaction(spot1[0]) }
      }

      out.set(track, list)
    })

    return out
  }, [gameState?.cultTracks?.priestsOnTrack, gameState?.players])

  const openHexActionModal = (q: number, r: number): void => {
    const mapHex = gameState?.map?.hexes?.[`${String(q)},${String(r)}`]
    if (!mapHex) return

    const isHome = mapHex.terrain === localHomeTerrain
    setPendingHex({ q, r })
    setHexActionMode(isHome ? 'build' : 'transform_build')
    setSelectedTerrain(localHomeTerrain)
  }

  const handleHexClick = (q: number, r: number): void => {
    if (!localPlayerId || !gameState) return

    if (gameState.phase === GamePhase.Setup && gameState.setupSubphase === 'dwellings') {
      if (setupDwellingPlayerId !== localPlayerId) return
      queueConfirm('Confirm Setup Dwelling', `Place setup dwelling at ${formatHexCoord({ q, r })}?`, () => {
        submitSetupDwelling(localPlayerId, q, r, gameId)
        setConfirmDialog(null)
      })
      return
    }

    if (isOtherPlayerTurnConfirmationWindow) return

    if (hasPendingDecisionForMe && pendingDecisionType === 'halflings_spades') {
      setPendingHex({ q, r })
      setHexActionMode('transform_only')
      setSelectedTerrain(localHomeTerrain)
      return
    }

    if (hasPendingDecisionForMe && pendingDecisionType === 'wisps_stronghold_dwelling') {
      queueConfirm('Confirm Wisps Dwelling', `Place the free Wisps dwelling at ${formatHexCoord({ q, r })}?`, () => {
        performAction('wisps_stronghold_dwelling', { targetHex: { q, r } })
        setConfirmDialog(null)
      })
      return
    }

    if (hasPendingCultSpadesForMe > 0) {
      setPendingHex({ q, r })
      setHexActionMode('transform_only')
      setSelectedTerrain(localHomeTerrain)
      return
    }

    if (hasPendingSpadesForMe > 0 && isMyTurn) {
      setPendingHex({ q, r })
      setHexActionMode(canBuildWithPendingSpadeForMe ? 'transform_build' : 'transform_only')
      setSelectedTerrain(localHomeTerrain)
      return
    }

    if (powerMode?.type === 'power_spade') {
      setPendingHex({ q, r })
      setHexActionMode('transform_build')
      setSelectedTerrain(localHomeTerrain)
      return
    }

    if (powerMode?.type === 'power_bridge') {
      if (powerMode.firstHex == null) {
        setPowerMode({ ...powerMode, firstHex: { q, r } })
        return
      }
      const from = powerMode.firstHex
      const to = { q, r }
      queueConfirm(
        'Confirm Bridge',
        `Build bridge from ${formatHexCoord(from)} to ${formatHexCoord(to)}?`,
        () => {
          performAction(powerMode.source === 'engineers' ? 'engineers_bridge' : 'power_bridge_place', { bridgeHex1: from, bridgeHex2: to })
          setPowerMode(null)
          setConfirmDialog(null)
        },
      )
      return
    }

    if (powerMode?.type === 'special_action_target') {
      const actionType = powerMode.actionType
      if (actionType === SpecialActionType.WitchesRide) {
        queueConfirm('Confirm Witches Ride', `Use Witches Ride on ${formatHexCoord({ q, r })}?`, () => {
          performAction('special_action_use', {
            specialActionType: actionType,
            targetHex: { q, r },
          })
          setPowerMode(null)
          setConfirmDialog(null)
        })
        return
      }

      if (actionType === SpecialActionType.SwarmlingsUpgrade) {
        queueConfirm('Confirm Swarmlings Upgrade', `Use Swarmlings free upgrade at ${formatHexCoord({ q, r })}?`, () => {
          performAction('special_action_use', {
            specialActionType: actionType,
            upgradeHex: { q, r },
          })
          setPowerMode(null)
          setConfirmDialog(null)
        })
        return
      }

      if (actionType === SpecialActionType.GiantsTransform || actionType === SpecialActionType.NomadsSandstorm || actionType === SpecialActionType.BonusCardSpade) {
        setPendingHex({ q, r })
        setHexActionMode('transform_build')
        setSelectedTerrain(localHomeTerrain)
        return
      }

      if (actionType === SpecialActionType.MermaidsRiverTown) {
        queueConfirm('Confirm Mermaids Connect', `Connect river town at ${formatHexCoord({ q, r })}?`, () => {
          performAction('special_action_use', {
            specialActionType: actionType,
            targetHex: { q, r },
          })
          setPowerMode(null)
          setConfirmDialog(null)
        })
      }
      return
    }

    if (!canInitiateTurnAction) return

    const hex = gameState.map?.hexes?.[`${String(q)},${String(r)}`]
    if (!hex) return

    if (hex.building?.ownerPlayerId === localPlayerId) {
      setUpgradeHex({ q, r })
      return
    }

    openHexActionModal(q, r)
  }

  const handleBridgeEdgeClick = (from: { q: number; r: number }, to: { q: number; r: number }): void => {
    if (isOtherPlayerTurnConfirmationWindow) return
    if (powerMode?.type !== 'power_bridge') return

    queueConfirm(
      'Confirm Bridge',
      `Build bridge from ${formatHexCoord(from)} to ${formatHexCoord(to)}?`,
      () => {
        performAction(powerMode.source === 'engineers' ? 'engineers_bridge' : 'power_bridge_place', {
          bridgeHex1: from,
          bridgeHex2: to,
          useCoins: powerMode.useCoins,
        })
        setPowerMode(null)
        setConfirmDialog(null)
      },
    )
  }

  const handleFactionSelect = (factionType: string): void => {
    if (!localPlayerId || !gameId) return
    queueConfirm('Confirm Faction', `Choose ${factionType} as your faction?`, () => {
      submitSelectFaction(localPlayerId, factionType, gameId)
      setConfirmDialog(null)
    })
  }

  const handleAuctionNominate = (factionType: string): void => {
    queueConfirm('Confirm Nomination', `Nominate ${factionType} for auction?`, () => {
      performAction('auction_nominate', { faction: factionType })
      setConfirmDialog(null)
    })
  }

  const handleAuctionBid = (factionType: string): void => {
    const additionalReduction = Math.max(0, Math.min(40, Math.trunc(Number(auctionBidInputs[factionType] ?? 0))))
    const currentReduction = Number(auctionState?.currentBids?.[factionType] ?? 0)
    const vpReduction = Math.max(0, Math.min(40, currentReduction + additionalReduction))
    queueConfirm('Confirm Auction Bid', `Increase ${factionType} to ${String(vpReduction)} VP reduction?`, () => {
      performAction('auction_bid', { faction: factionType, vpReduction })
      setConfirmDialog(null)
    })
  }

  const handleFastAuctionSubmit = (): void => {
    const bids: Record<string, number> = {}
    pendingAuctionFactions.forEach((faction) => {
      bids[faction] = Math.max(0, Math.min(40, Math.trunc(Number(fastAuctionBidInputs[faction] ?? 0))))
    })
    queueConfirm('Confirm Fast Auction Bids', 'Submit your fast auction bid matrix?', () => {
      performAction('fast_auction_submit_bids', { bids })
      setConfirmDialog(null)
    })
  }

  const handlePowerActionClick = (action: PowerActionType): void => {
    if (!canInitiateTurnAction) return
    const canUseCoins = canPayPowerActionWithCoins(localPlayer, action)
    const canUsePower = canPayPowerActionWithPower(localPlayer, action)
    const choosePayment = (useCoins: boolean): void => {
      if (action === PowerActionType.Bridge) {
        setPowerMode({ type: 'power_bridge', source: 'power', firstHex: null, useCoins })
        return
      }

      if (action === PowerActionType.Spade || action === PowerActionType.DoubleSpade) {
        setPowerMode({ type: 'power_spade', actionType: action, useCoins })
        return
      }

      performAction('power_action_claim', { actionType: action, useCoins })
    }

    if (canUseCoins && canUsePower) {
      openChoiceDialog(
        'Choose Payment',
        `Pay for ${PowerActionType[action]} with coins or power?`,
        [
          {
            label: 'Coins',
            testId: 'confirm-action-choice-coins',
            className: 'rounded bg-amber-500 px-3 py-1 text-sm text-slate-950',
            onClick: () => {
              setConfirmDialog(null)
              choosePayment(true)
            },
          },
          {
            label: 'Power',
            testId: 'confirm-action-choice-power',
            className: 'rounded bg-blue-600 px-3 py-1 text-sm text-white',
            onClick: () => {
              setConfirmDialog(null)
              choosePayment(false)
            },
          },
        ],
      )
      return
    }

    choosePayment(shouldAutoUseChashCoins(localPlayer, action))
  }

  const handleCultSpotClick = (cult: CultType, tileIndex: number): void => {
    if (!canInitiateTurnAction) return

    const spaces = tileIndex === 0 ? 3 : tileIndex === 4 ? 1 : 2
    queueConfirm(
      'Confirm Priest Send',
      `Send priest to ${CULT_CHOICES.find((c) => c.track === cult)?.label ?? 'cult'} (${String(spaces)} step)?`,
      () => {
        performAction('send_priest', { cultTrack: cult, spacesToClimb: spaces })
        setConfirmDialog(null)
      },
    )
  }

  const handleConversion = (playerId: string, conversionType: string): void => {
    if (!gameState || playerId !== localPlayerId || !canUseConversionWindow) return

    queueConfirm('Confirm Conversion', `Execute conversion: ${conversionType}?`, () => {
      performAction('conversion', { conversionType, amount: 1 })
      setConfirmDialog(null)
    })
  }

  const handleBurnPower = (playerId: string, amount: number): void => {
    if (!gameState || playerId !== localPlayerId || !canUseConversionWindow) return

    queueConfirm('Confirm Burn Power', `Burn ${String(amount * 2)} power from Bowl II to gain ${String(amount)} power in Bowl III?`, () => {
      performAction('burn_power', { amount })
      setConfirmDialog(null)
    })
  }

  const handleAdvanceShipping = (playerId: string): void => {
    if (playerId !== localPlayerId || !canInitiateTurnAction) return

    queueConfirm('Confirm Shipping Upgrade', 'Upgrade shipping track?', () => {
      performAction('advance_shipping')
      setConfirmDialog(null)
    })
  }

  const handleAdvanceDigging = (playerId: string): void => {
    if (playerId !== localPlayerId || !canInitiateTurnAction) return

    queueConfirm('Confirm Digging Upgrade', 'Upgrade digging track?', () => {
      performAction('advance_digging')
      setConfirmDialog(null)
    })
  }

  const handleAdvanceChashTrack = (playerId: string): void => {
    if (playerId !== localPlayerId || !canInitiateTurnAction) return

    queueConfirm('Confirm Chash Track', 'Advance the Chash Dallah income track for 2 workers and 2 coins?', () => {
      performAction('advance_chash_track')
      setConfirmDialog(null)
    })
  }

  const handleEngineersBridgeAction = (playerId: string): void => {
    if (playerId !== localPlayerId || !canInitiateTurnAction) return
    setPowerMode({ type: 'power_bridge', source: 'engineers', firstHex: null, useCoins: false })
  }

  const handleMermaidsConnectAction = (playerId: string): void => {
    if (playerId !== localPlayerId || !canInitiateTurnAction) return
    setPowerMode({ type: 'special_action_target', actionType: SpecialActionType.MermaidsRiverTown })
  }

  const closeConspiratorsSwapModal = (): void => {
    setConspiratorsSwapModalOpen(false)
    setConspiratorsReturnTile('')
    setConspiratorsNewTile('')
  }

  const handleStrongholdAction = (_playerId: string, actionType: SpecialActionType): void => {
    if (!canInitiateTurnAction) return

    if (actionType === SpecialActionType.AurenCultAdvance) {
      setCultChoiceContext('auren_sh')
      return
    }

    if (
      actionType === SpecialActionType.GiantsTransform
      || actionType === SpecialActionType.NomadsSandstorm
      || actionType === SpecialActionType.WitchesRide
      || actionType === SpecialActionType.SwarmlingsUpgrade
      || actionType === SpecialActionType.MermaidsRiverTown
    ) {
      setPowerMode({ type: 'special_action_target', actionType })
      return
    }

    if (actionType === SpecialActionType.ChaosMagiciansDoubleTurn) {
      setChaosModalOpen(true)
      return
    }

    if (actionType === SpecialActionType.ConspiratorsSwapFavor) {
      const returnTile = localPlayerFavorTiles[0]
      const newTile = availableFavorTileTypes.find((tile) => tile !== returnTile && !localPlayerFavorTiles.includes(tile))
      setConspiratorsReturnTile(returnTile ?? '')
      setConspiratorsNewTile(newTile ?? '')
      setConspiratorsSwapModalOpen(true)
      return
    }

    queueConfirm('Confirm Special Action', `Use special action ${SpecialActionType[actionType]}?`, () => {
      performAction('special_action_use', { specialActionType: actionType })
      setConfirmDialog(null)
    })
  }

  const conspiratorsNewTileOptions = useMemo(() => {
    if (conspiratorsReturnTile === '') return [] as FavorTileType[]
    return availableFavorTileTypes.filter((tile) => tile !== conspiratorsReturnTile && !localPlayerFavorTiles.includes(tile))
  }, [availableFavorTileTypes, conspiratorsReturnTile, localPlayerFavorTiles])

  useEffect(() => {
    if (!conspiratorsSwapModalOpen) return
    if (conspiratorsReturnTile === '' || !localPlayerFavorTiles.includes(conspiratorsReturnTile as FavorTileType)) {
      const nextReturn = localPlayerFavorTiles[0]
      setConspiratorsReturnTile(nextReturn ?? '')
      return
    }
    if (
      conspiratorsNewTile === ''
      || !conspiratorsNewTileOptions.includes(conspiratorsNewTile as FavorTileType)
    ) {
      setConspiratorsNewTile(conspiratorsNewTileOptions[0] ?? '')
    }
  }, [
    conspiratorsNewTile,
    conspiratorsNewTileOptions,
    conspiratorsReturnTile,
    conspiratorsSwapModalOpen,
    localPlayerFavorTiles,
  ])

  const submitConspiratorsSwap = (): void => {
    if (conspiratorsReturnTile === '' || conspiratorsNewTile === '') return
    performAction('special_action_use', {
      specialActionType: SpecialActionType.ConspiratorsSwapFavor,
      returnTile: conspiratorsReturnTile,
      newTile: conspiratorsNewTile,
    })
    closeConspiratorsSwapModal()
  }

  const handleWater2Action = (_playerId: string): void => {
    if (!canInitiateTurnAction) return
    setCultChoiceContext('water2')
  }

  const handlePassingTileClick = (cardType: BonusCardType): void => {
    if (!gameState || !localPlayerId) return

    const owner = bonusCardOwners[String(cardType)]
    const isOwnedByMe = owner === localPlayerId

    if (pendingDecisionType === 'setup_bonus_card' && hasPendingDecisionForMe) {
      queueConfirm('Confirm Setup Bonus Card', `Take ${bonusCardLabel(cardType)}?`, () => {
        performAction('setup_bonus_card', { bonusCard: cardType })
        setConfirmDialog(null)
      })
      return
    }

    if (isOwnedByMe && canInitiateTurnAction) {
      if (cardType === BonusCardType.Spade) {
        setPowerMode({ type: 'special_action_target', actionType: SpecialActionType.BonusCardSpade })
        return
      }
      if (cardType === BonusCardType.CultAdvance) {
        setCultChoiceContext('bonus_cult')
      }
      return
    }

    if (!canInitiateTurnAction) return
    if (owner) return

    const warning = hasUnspentOptionalActions
      ? ' You still have optional special actions or pending spades available.'
      : ''
    queueConfirm('Confirm Pass', `Pass and take ${bonusCardLabel(cardType)}?${warning}`, () => {
      performAction('pass', { bonusCard: cardType })
      setConfirmDialog(null)
    }, 'turn_end')
  }

  const isPassingCardClickable = (cardType: BonusCardType): boolean => {
    if (!gameState || !localPlayerId) return false
    const owner = bonusCardOwners[String(cardType)]

    if (pendingDecisionType === 'setup_bonus_card') {
      return hasPendingDecisionForMe && !owner
    }

    if (owner === localPlayerId && canInitiateTurnAction) {
      if (cardType === BonusCardType.Spade || cardType === BonusCardType.CultAdvance) {
        return true
      }
    }

    if (!canInitiateTurnAction) return false
    return !owner
  }

  const handlePassWithoutCard = (): void => {
    if (!canInitiateTurnAction) return
    const warning = hasUnspentOptionalActions
      ? ' You still have optional special actions or pending spades available.'
      : ''
    queueConfirm('Confirm Pass', `Pass this round without selecting a bonus card?${warning}`, () => {
      performAction('pass', {})
      setConfirmDialog(null)
    }, 'turn_end')
  }

  const handleFavorTileClick = (tileType: FavorTileType): void => {
    if (!(hasPendingDecisionForMe && pendingDecisionType === 'favor_tile_selection')) return

    queueConfirm('Confirm Favor Tile', `Take ${favorTileLabel(tileType)}?`, () => {
      performAction('select_favor_tile', { tileType })
      setConfirmDialog(null)
    })
  }

  const isFavorTileClickable = (_tileType: FavorTileType, availableCount: number): boolean => {
    if (availableCount <= 0) return false
    return hasPendingDecisionForMe && pendingDecisionType === 'favor_tile_selection' && !!localPlayerId
  }

  const hasPendingTownFormationForMe = useMemo(() => {
    if (!localPlayerId) return false
    const pending = gameState?.pendingTownFormations as Record<string, unknown[] | undefined> | undefined
    const list = pending?.[localPlayerId] ?? []
    return list.length > 0
  }, [gameState?.pendingTownFormations, localPlayerId])

  const handleTownTileClick = (tileId: TownTileId): void => {
    const isMandatoryTownSelection = hasPendingDecisionForMe && pendingDecisionType === 'town_tile_selection'
    const isOptionalDelayedTownSelection = !!localPlayerId && hasPendingTownFormationForMe
    if (!isMandatoryTownSelection && !isOptionalDelayedTownSelection) return

    queueConfirm('Confirm Town Tile', `Take ${townTileLabel(tileId)}?`, () => {
      performAction('select_town_tile', { tileType: tileId })
      setConfirmDialog(null)
    })
  }

  const isTownTileClickable = (_tileId: TownTileId, count: number): boolean => {
    if (count <= 0) return false
    const isMandatoryTownSelection = hasPendingDecisionForMe && pendingDecisionType === 'town_tile_selection'
    const isOptionalDelayedTownSelection = !!localPlayerId && hasPendingTownFormationForMe
    return isMandatoryTownSelection || isOptionalDelayedTownSelection
  }

  const submitCultChoice = (track: CultType): void => {
    if (cultChoiceContext === 'cultists') {
      performAction('select_cultists_track', { cultTrack: track })
      setCultChoiceContext(null)
      return
    }

    if (cultChoiceContext === 'water2') {
      performAction('special_action_use', {
        specialActionType: SpecialActionType.Water2CultAdvance,
        cultTrack: track,
      })
      setCultChoiceContext(null)
      return
    }

    if (cultChoiceContext === 'auren_sh') {
      performAction('special_action_use', {
        specialActionType: SpecialActionType.AurenCultAdvance,
        cultTrack: track,
      })
      setCultChoiceContext(null)
      return
    }

    if (cultChoiceContext === 'bonus_cult') {
      performAction('special_action_use', {
        specialActionType: SpecialActionType.BonusCardCultAdvance,
        cultTrack: track,
      })
      setCultChoiceContext(null)
    }
  }

  const submitChaosDoubleTurn = (): void => {
    let firstParamsObj: Record<string, unknown>
    let secondParamsObj: Record<string, unknown>
    try {
      firstParamsObj = JSON.parse(chaosFirstParams) as Record<string, unknown>
      secondParamsObj = JSON.parse(chaosSecondParams) as Record<string, unknown>
    } catch {
      setErrorMessage('Invalid JSON in Chaos double-turn params.')
      return
    }

    performAction('special_action_use', {
      specialActionType: SpecialActionType.ChaosMagiciansDoubleTurn,
      firstAction: {
        type: chaosFirstType,
        params: firstParamsObj,
      },
      secondAction: {
        type: chaosSecondType,
        params: secondParamsObj,
      },
    })
    setChaosModalOpen(false)
  }

  const toggleTownCultTrack = (track: CultType): void => {
    setSelectedTownCultTracks((current) => {
      if (current.includes(track)) {
        return current.filter((t) => t !== track)
      }
      if (current.length >= pendingTownCultTopMaxSelections) {
        return current
      }
      return [...current, track]
    })
  }

  const setupBonusCards = useMemo(() => {
    return Object.entries(gameState?.bonusCards?.available ?? {})
      .map(([k]) => Number(k))
      .filter((card) => Number.isInteger(card) && card >= 0)
      .sort((a, b) => a - b)
  }, [gameState?.bonusCards?.available])

  const transformedHalflingsHexes = useMemo(() => {
    const pending = gameState?.pendingHalflingsSpades as unknown as { transformedHexes?: Array<{ Q?: number; R?: number; q?: number; r?: number }> }
    const items = pending?.transformedHexes ?? []
    return items
      .map((h) => ({
        q: h.q ?? h.Q,
        r: h.r ?? h.R,
      }))
      .filter((h): h is { q: number; r: number } => typeof h.q === 'number' && typeof h.r === 'number')
  }, [gameState?.pendingHalflingsSpades])

  const isHalflingsSpadeDecision = hasPendingDecisionForMe && pendingDecisionType === 'halflings_spades'
  const isCultSpadeDecision = hasPendingCultSpadesForMe > 0
  const isPendingSpadeDecision = hasPendingSpadesForMe > 0
  const isPostActionFreeWindowForMe = hasPendingDecisionForMe && pendingDecisionType === 'post_action_free_actions'
  const isTurnConfirmationWindowForMe = hasPendingDecisionForMe && (pendingDecisionType === 'post_action_free_actions' || pendingDecisionType === 'turn_confirmation')
  const canUseConversionWindow = canInitiateTurnAction || isPostActionFreeWindowForMe
  const hasFavorSelectionForMe = hasPendingDecisionForMe && pendingDecisionType === 'favor_tile_selection'
  const hasTownSelectionForMe = (hasPendingDecisionForMe && pendingDecisionType === 'town_tile_selection')
    || (!!localPlayerId && hasPendingTownFormationForMe)
  const activeStrongholdActionType = useMemo((): SpecialActionType | null => {
    if (powerMode?.type === 'special_action_target' && STRONGHOLD_TARGET_ACTION_TYPES.includes(powerMode.actionType)) {
      return powerMode.actionType
    }
    if (cultChoiceContext === 'auren_sh') return SpecialActionType.AurenCultAdvance
    if (chaosModalOpen) return SpecialActionType.ChaosMagiciansDoubleTurn
    if (conspiratorsSwapModalOpen) return SpecialActionType.ConspiratorsSwapFavor
    return null
  }, [chaosModalOpen, conspiratorsSwapModalOpen, cultChoiceContext, powerMode])
  const activeBonusCardActionType = useMemo((): SpecialActionType | null => {
    if (powerMode?.type === 'special_action_target' && powerMode.actionType === SpecialActionType.BonusCardSpade) {
      return SpecialActionType.BonusCardSpade
    }
    if (cultChoiceContext === 'bonus_cult') return SpecialActionType.BonusCardCultAdvance
    return null
  }, [cultChoiceContext, powerMode])
  const activeWater2Action = cultChoiceContext === 'water2'

  const closeHexModal = (): void => {
    setPendingHex(null)
    setHexActionMode(null)
  }

  const cancelHexAction = (): void => {
    closeHexModal()
    if (powerMode?.type === 'power_spade' || powerMode?.type === 'special_action_target') {
      setPowerMode(null)
    }
  }

  const submitHexModalAction = (): void => {
    if (!pendingHex || !hexActionMode) return

    if (isHalflingsSpadeDecision) {
      const remaining = Number((gameState?.pendingHalflingsSpades as Record<string, unknown> | undefined)?.spadesRemaining ?? 0)
      if (remaining > 0) {
        performAction('halflings_apply_spade', {
          hex: pendingHex,
          targetTerrain: selectedTerrain,
        })
      }
      closeHexModal()
      return
    }

    if (hasPendingCultSpadesForMe > 0) {
      performAction('use_cult_spade', {
        hex: pendingHex,
        targetTerrain: selectedTerrain,
      })
      closeHexModal()
      return
    }

    if (hasPendingSpadesForMe > 0) {
      performAction('transform_build', {
        hex: pendingHex,
        buildDwelling: hexActionMode !== 'transform_only',
        targetTerrain: selectedTerrain,
      })
      closeHexModal()
      return
    }

    if (powerMode?.type === 'power_spade') {
      const buildDwelling = hexActionMode !== 'transform_only'
      performAction('power_action_claim', {
        actionType: powerMode.actionType,
        targetHex: pendingHex,
        buildDwelling,
        targetTerrain: selectedTerrain,
        useCoins: powerMode.useCoins,
      })
      setPowerMode(null)
      closeHexModal()
      return
    }

    if (powerMode?.type === 'special_action_target') {
      const actionType = powerMode.actionType
      const buildDwelling = hexActionMode !== 'transform_only'
      performAction('special_action_use', {
        specialActionType: actionType,
        targetHex: pendingHex,
        buildDwelling,
        targetTerrain: selectedTerrain,
      })
      setPowerMode(null)
      closeHexModal()
      return
    }

    if (hexActionMode === 'build') {
      performAction('transform_build', {
        hex: pendingHex,
        buildDwelling: true,
      })
      closeHexModal()
      return
    }

    if (hexActionMode === 'transform_build') {
      performAction('transform_build', {
        hex: pendingHex,
        buildDwelling: true,
        targetTerrain: selectedTerrain,
      })
      closeHexModal()
      return
    }

    if (hexActionMode === 'transform_only') {
      performAction('transform_build', {
        hex: pendingHex,
        buildDwelling: false,
        targetTerrain: selectedTerrain,
      })
      closeHexModal()
    }
  }

  const upgradeOptions = useMemo(() => {
    if (!upgradeHex || !gameState) return [] as Array<{ type: BuildingType; label: string }>

    const key = `${String(upgradeHex.q)},${String(upgradeHex.r)}`
    const building = gameState.map.hexes[key]?.building
    if (!building || building.ownerPlayerId !== localPlayerId) return []

    switch (building.type) {
      case BuildingType.Dwelling:
        return [{ type: BuildingType.TradingHouse, label: 'Upgrade to Trading House' }]
      case BuildingType.TradingHouse:
        return [
          { type: BuildingType.Temple, label: 'Upgrade to Temple' },
          { type: BuildingType.Stronghold, label: 'Upgrade to Stronghold' },
        ]
      case BuildingType.Temple:
        return [{ type: BuildingType.Sanctuary, label: 'Upgrade to Sanctuary' }]
      default:
        return []
    }
  }, [gameState, localPlayerId, upgradeHex])

  const selectUpgrade = (newType: BuildingType): void => {
    if (!upgradeHex) return
    performAction('upgrade_building', {
      targetHex: upgradeHex,
      newBuildingType: newType,
    })
    setUpgradeHex(null)
  }

  return (
    <div className="min-h-screen bg-white p-4 text-gray-900" data-testid="game-screen">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-3xl font-bold text-gray-800">Terra Mystica - Game {gameId}</h1>
          <div className="flex gap-2">
            <button
              data-testid="layout-lock-toggle"
              onClick={() => { setIsLayoutLocked(!isLayoutLocked) }}
              className={`px-4 py-2 rounded text-sm font-medium transition-colors ${isLayoutLocked
                ? 'bg-blue-100 text-blue-700 hover:bg-blue-200'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
            >
              {isLayoutLocked ? 'Unlock Layout' : 'Lock Layout'}
            </button>
            <button
              data-testid="layout-reset"
              onClick={resetLayout}
              className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-sm font-medium text-gray-700 transition-colors"
            >
              Reset Layout
            </button>
          </div>
        </div>

        {isSpectator && (
          <div className="mb-3 rounded border border-sky-300 bg-sky-50 px-4 py-2 text-sm text-sky-900" data-testid="spectator-banner">
            Spectator mode. You can watch this game, but you cannot take actions.
          </div>
        )}

        {localPlayer && (
          <div className="mb-3 rounded border border-slate-300 bg-white px-3 py-2" data-testid="player-options-panel">
            <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-600">Player Options</div>
            <div className="flex flex-wrap items-center gap-3">
              <label className="flex items-center gap-2 text-sm text-slate-800">
                <span>Auto Leech</span>
                <select
                  data-testid="option-auto-leech-mode"
                  className="rounded border border-slate-300 px-2 py-1 text-sm"
                  value={localPlayerOptions.autoLeechMode}
                  onChange={(e) => { updatePlayerOptions({ autoLeechMode: e.target.value as LeechAutoMode }) }}
                >
                  {LEECH_AUTO_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </label>

              <label className="flex items-center gap-2 text-sm text-slate-800">
                <input
                  data-testid="option-auto-convert-pass"
                  type="checkbox"
                  checked={localPlayerOptions.autoConvertOnPass}
                  onChange={(e) => { updatePlayerOptions({ autoConvertOnPass: e.target.checked }) }}
                />
                <span>Auto convert on pass</span>
              </label>

              <label className="flex items-center gap-2 text-sm text-slate-800">
                <input
                  data-testid="option-confirm-actions"
                  type="checkbox"
                  checked={localPlayerOptions.confirmActions}
                  onChange={(e) => { updatePlayerOptions({ confirmActions: e.target.checked }) }}
                />
                <span>Confirm Turn End</span>
              </label>

              <label className="flex items-center gap-2 text-sm text-slate-800">
                <input
                  data-testid="option-show-income-preview"
                  type="checkbox"
                  checked={localPlayerOptions.showIncomePreview}
                  onChange={(e) => { updatePlayerOptions({ showIncomePreview: e.target.checked }) }}
                />
                <span>Show Next Income</span>
              </label>
            </div>
          </div>
        )}

        <div className="mb-3 sticky top-2 z-40 rounded border border-slate-300 bg-white px-4 py-3 min-h-[72px] shadow-sm" data-testid="game-decision-strip">
          <div className="mb-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Required Action</div>
            <div className="text-sm font-semibold text-slate-900">{decisionStripStatus}</div>
          </div>
          {confirmDialog ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">{confirmDialog.title}</div>
                <div className="text-sm text-slate-700">{confirmDialog.message}</div>
              </div>
              <div className="flex items-center gap-2">
                <button data-testid="confirm-action-cancel" className="rounded bg-gray-200 px-3 py-1 text-sm text-gray-800" onClick={() => { setConfirmDialog(null) }}>Cancel</button>
                {confirmDialog.actions ? confirmDialog.actions.map((action) => (
                  <button
                    key={action.testId ?? action.label}
                    data-testid={action.testId}
                    className={action.className ?? 'rounded bg-blue-600 px-3 py-1 text-sm text-white'}
                    onClick={() => {
                      action.onClick()
                    }}
                  >
                    {action.label}
                  </button>
                )) : (
                  <button
                    data-testid="confirm-action-confirm"
                    className="rounded bg-blue-600 px-3 py-1 text-sm text-white"
                    onClick={() => {
                      if (!confirmDialog?.onConfirm) return
                      confirmDialog.onConfirm()
                    }}
                  >
                    Confirm
                  </button>
                )}
              </div>
            </div>
          ) : pendingHex ? (
            <div className="space-y-3">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold text-slate-900">
                    {isHalflingsSpadeDecision ? 'Apply Halflings Spade' : isCultSpadeDecision ? 'Use Cult Spade' : 'Hex Action'}
                  </div>
                  <div className="text-sm text-slate-700">Selected hex: {formatHexCoord(pendingHex)}</div>
                </div>
                <div className="flex items-center gap-2">
                  <button data-testid="hex-action-cancel" className="rounded bg-gray-200 px-3 py-1 text-sm text-gray-800" onClick={cancelHexAction}>Cancel</button>
                  <button data-testid="hex-action-submit" className="rounded bg-blue-600 px-3 py-1 text-sm text-white" onClick={submitHexModalAction}>Submit</button>
                </div>
              </div>

              {isHalflingsSpadeDecision && (
                <div className="flex flex-wrap items-center gap-3">
                  <span className="text-sm text-slate-700">Choose terrain to transform to.</span>
                  <select
                    data-testid="hex-action-terrain-halflings"
                    value={selectedTerrain}
                    onChange={(e) => { setSelectedTerrain(Number(e.target.value) as TerrainType) }}
                    className="rounded border px-2 py-1 text-sm"
                  >
                    {TERRAIN_CHOICES.map((t) => (
                      <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>
              )}

              {isCultSpadeDecision && (
                <div className="flex flex-wrap items-end gap-4">
                  <div className="rounded border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900">
                    This action uses one cult reward spade and cannot build a dwelling.
                  </div>
                  <label className="flex flex-col gap-1 text-sm text-slate-800">
                    <span className="font-medium">Target terrain</span>
                    <select
                      data-testid="hex-action-target-terrain"
                      value={selectedTerrain}
                      onChange={(e) => { setSelectedTerrain(Number(e.target.value) as TerrainType) }}
                      className="rounded border px-2 py-1"
                    >
                      {TERRAIN_CHOICES.map((t) => (
                        <option key={t.id} value={t.id}>{t.name}</option>
                      ))}
                    </select>
                  </label>
                </div>
              )}

              {!isHalflingsSpadeDecision && !isCultSpadeDecision && (
                <div className="flex flex-wrap items-end gap-4">
                  {isPendingSpadeDecision && !canBuildWithPendingSpadeForMe ? (
                    <div className="rounded border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
                      This follow-up spade can only transform terrain (no dwelling build allowed).
                    </div>
                  ) : (
                    <label className="flex flex-col gap-1 text-sm text-slate-800">
                      <span className="font-medium">Action</span>
                      <select
                        data-testid="hex-action-mode"
                        value={hexActionMode ?? 'build'}
                        onChange={(e) => { setHexActionMode(e.target.value as 'build' | 'transform_build' | 'transform_only') }}
                        className="rounded border px-2 py-1"
                      >
                        <option value="build">Build dwelling</option>
                        <option value="transform_build">Transform and build</option>
                        <option value="transform_only">Transform only</option>
                      </select>
                    </label>
                  )}

                  {(hexActionMode === 'transform_build' || hexActionMode === 'transform_only') && (
                    <label className="flex flex-col gap-1 text-sm text-slate-800">
                      <span className="font-medium">Target terrain</span>
                      <select
                        data-testid="hex-action-target-terrain"
                        value={selectedTerrain}
                        onChange={(e) => { setSelectedTerrain(Number(e.target.value) as TerrainType) }}
                        className="rounded border px-2 py-1"
                      >
                        {TERRAIN_CHOICES.map((t) => (
                          <option key={t.id} value={t.id}>{t.name}</option>
                        ))}
                      </select>
                    </label>
                  )}
                </div>
              )}
            </div>
          ) : upgradeHex ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Upgrade Building</div>
                <div className="text-sm text-slate-700">Select an upgrade for {formatHexCoord(upgradeHex)}.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {upgradeOptions.length === 0 && <span className="text-sm text-slate-700">No legal upgrades for this building.</span>}
                {upgradeOptions.map((opt) => (
                  <button
                    key={opt.type}
                    type="button"
                    data-testid={`upgrade-option-${String(opt.type)}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { selectUpgrade(opt.type) }}
                  >
                    {opt.label}
                  </button>
                ))}
                <button
                  type="button"
                  data-testid="upgrade-building-cancel"
                  className="rounded bg-gray-200 px-3 py-2 text-sm text-gray-800"
                  onClick={() => { setUpgradeHex(null) }}
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : cultChoiceContext !== null ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Choose Cult Track</div>
                <div className="text-sm text-slate-700">Select the cult track for this action.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {CULT_CHOICES.map((choice) => (
                  <button
                    key={choice.track}
                    type="button"
                    data-testid={`cult-choice-${String(choice.track)}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { submitCultChoice(choice.track) }}
                  >
                    {choice.label}
                  </button>
                ))}
                <button
                  type="button"
                  data-testid="cult-choice-cancel"
                  className="rounded bg-gray-200 px-3 py-2 text-sm text-gray-800"
                  onClick={() => { setCultChoiceContext(null) }}
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : hasPendingDecisionForMe && pendingDecisionType === 'town_cult_top_choice' ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Choose Cults To Top</div>
                <div className="text-sm text-slate-700">Choose {String(pendingTownCultTopMaxSelections)} track(s) to advance to 10.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {pendingTownCultTopCandidates.map((track) => (
                  <button
                    key={track}
                    type="button"
                    data-testid={`town-cult-top-choice-${String(track)}`}
                    className={`rounded border px-3 py-2 text-sm ${selectedTownCultTracks.includes(track) ? 'border-blue-400 bg-blue-100 text-blue-900' : 'border-slate-300 text-slate-800 hover:bg-slate-100'}`}
                    onClick={() => { toggleTownCultTrack(track) }}
                  >
                    {CULT_CHOICES.find((choice) => choice.track === track)?.label ?? `Cult ${String(track)}`}
                  </button>
                ))}
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  data-testid="town-cult-top-choice-confirm"
                  className="rounded bg-blue-600 px-3 py-2 text-sm text-white disabled:bg-blue-300"
                  disabled={selectedTownCultTracks.length !== pendingTownCultTopMaxSelections}
                  onClick={() => {
                    performAction('select_town_cult_top', { tracks: selectedTownCultTracks })
                  }}
                >
                  Confirm selection
                </button>
              </div>
            </div>
          ) : hasPendingDecisionForMe && pendingDecisionType === 'setup_bonus_card' ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Choose Setup Bonus Card</div>
                <div className="text-sm text-slate-700">Select your starting bonus card.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {setupBonusCards.map((cardId) => (
                  <button
                    key={cardId}
                    type="button"
                    data-testid={`setup-bonus-card-${String(cardId)}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { performAction('setup_bonus_card', { bonusCard: cardId }) }}
                  >
                    {bonusCardLabel(cardId)}
                  </button>
                ))}
              </div>
            </div>
          ) : hasPendingDecisionForMe && pendingDecisionType === 'darklings_ordination' ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Darklings Ordination</div>
                <div className="text-sm text-slate-700">
                  Darklings can only convert workers to priests when a player just upgraded to a stronghold.
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {[0, 1, 2, 3].map((count) => (
                  <button
                    key={count}
                    type="button"
                    data-testid={`darklings-ordination-${count}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { performAction('darklings_ordination', { workersToConvert: count }) }}
                  >
                    {count}
                  </button>
                ))}
              </div>
            </div>
          ) : hasPendingDecisionForMe && pendingDecisionType === 'cultists_cult_choice' ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Cultists: Choose Cult Track</div>
                <div className="text-sm text-slate-700">Select the cult track to advance.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {CULT_CHOICES.map((choice) => (
                  <button
                    key={choice.track}
                    type="button"
                    data-testid={`cultists-cult-choice-${String(choice.track)}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { performAction('select_cultists_track', { cultTrack: choice.track }) }}
                  >
                    {choice.label}
                  </button>
                ))}
              </div>
            </div>
          ) : hasPendingDecisionForMe && pendingDecisionType === 'halflings_spades' && Number((gameState?.pendingHalflingsSpades as Record<string, unknown> | undefined)?.spadesRemaining ?? 0) === 0 ? (
            <div className="space-y-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Halflings Optional Dwelling</div>
                <div className="text-sm text-slate-700">All spades used. Build one dwelling on a transformed hex or skip.</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {transformedHalflingsHexes.map((h) => (
                  <button
                    key={`${String(h.q)},${String(h.r)}`}
                    type="button"
                    data-testid={`halflings-build-${String(h.q)}-${String(h.r)}`}
                    className="rounded border border-slate-300 px-3 py-2 text-sm text-slate-800 hover:bg-slate-100"
                    onClick={() => { performAction('halflings_build_dwelling', { targetHex: h }) }}
                  >
                    Build at {formatHexCoord(h)}
                  </button>
                ))}
                <button
                  data-testid="halflings-skip-dwelling"
                  className="rounded bg-gray-600 px-3 py-2 text-sm text-white"
                  onClick={() => { performAction('halflings_skip_dwelling') }}
                >
                  Skip dwelling
                </button>
              </div>
            </div>
          ) : pendingLeechOffersForMe.length > 0 ? (
            <div className="space-y-2">
              <div className="text-sm font-semibold text-slate-900">Leech Offer</div>
              {pendingLeechOffersForMe.map((offer, idx) => {
                const amount = Number(offer.Amount ?? offer.amount ?? 0)
                const vpCost = Math.max(0, amount - 1)
                return (
                  <div key={idx} className="flex flex-wrap items-center justify-between gap-2 rounded border border-slate-200 bg-slate-50 px-3 py-2">
                    <span className="text-sm text-slate-800">Accept {String(amount)} power for {String(vpCost)} VP?</span>
                    <div className="flex items-center gap-2">
                      <button data-testid={`leech-offer-${idx}-accept`} className="rounded bg-green-600 px-3 py-1 text-sm text-white" onClick={() => { performAction('accept_leech', { offerIndex: idx }) }}>Accept</button>
                      <button data-testid={`leech-offer-${idx}-decline`} className="rounded bg-gray-600 px-3 py-1 text-sm text-white" onClick={() => { performAction('decline_leech', { offerIndex: idx }) }}>Decline</button>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : waitingOnOtherLeechResponses > 0 ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Waiting On Leech Responses</div>
                <div className="text-sm text-slate-700">
                  Waiting for {pendingLeechResponderList} to accept or decline leech.
                </div>
              </div>
            </div>
          ) : hasFavorSelectionForMe ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Please select a favor tile</div>
                <div className="text-sm text-slate-700">A favor tile is required before your turn can continue.</div>
              </div>
            </div>
          ) : hasTownSelectionForMe ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Please select a town tile</div>
                <div className="text-sm text-slate-700">A town tile selection is required.</div>
              </div>
            </div>
          ) : isTurnConfirmationWindowForMe ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Confirm Turn</div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  data-testid="turn-end-undo"
                  className="rounded bg-amber-600 px-3 py-1 text-sm text-white"
                  onClick={() => { performAction('undo_turn') }}
                >
                  Undo
                </button>
                <button
                  data-testid="turn-end-confirm"
                  className="rounded bg-slate-700 px-3 py-1 text-sm text-white"
                  onClick={() => { performAction('confirm_turn') }}
                >
                  Confirm turn
                </button>
              </div>
            </div>
          ) : (
            <div className="text-sm text-slate-500">No pending follow-up controls.</div>
          )}
        </div>

        {errorMessage && (
          <div className="mb-4 rounded border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800" data-testid="action-error-message">
            {errorMessage}
          </div>
        )}

        {(powerMode?.type === 'power_bridge' && powerMode.firstHex == null) && (
          <div className="mb-3 rounded border border-orange-300 bg-orange-50 px-4 py-2 text-sm text-orange-800">
            Bridge mode: click a bridge edge on the board (or click two endpoints).
          </div>
        )}

        {(powerMode?.type === 'power_bridge' && powerMode.firstHex != null) && (
          <div className="mb-3 rounded border border-orange-300 bg-orange-50 px-4 py-2 text-sm text-orange-800">
            Bridge mode: first endpoint selected at {formatHexCoord(powerMode.firstHex)}. Click the second endpoint.
          </div>
        )}

        {(powerMode?.type === 'power_spade' || powerMode?.type === 'special_action_target') && (
          <div className="mb-3 rounded border border-blue-300 bg-blue-50 px-4 py-2 text-sm text-blue-800">
            {powerMode?.type === 'special_action_target' && STRONGHOLD_TARGET_ACTION_TYPES.includes(powerMode.actionType)
              ? 'Stronghold action active: click a target hex.'
              : 'Click a target hex.'}
          </div>
        )}

        {(hasPendingSpadesForMe > 0 && isMyTurn) && (
          <div className="mb-3 flex items-center justify-between rounded border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900" data-testid="pending-spades-banner">
            <span>
              Pending spade follow-up: {String(hasPendingSpadesForMe)} spade(s) remaining.
              {canBuildWithPendingSpadeForMe ? '' : ' Dwelling build on this follow-up is not allowed.'}
            </span>
            <button
              data-testid="discard-pending-spade"
              className="rounded bg-amber-700 px-3 py-1 text-white"
              onClick={() => { performAction('discard_pending_spade', { count: 1 }) }}
            >
              Discard 1 Spade
            </button>
          </div>
        )}

        {(hasPendingCultSpadesForMe > 0) && (
          <div className="mb-3 rounded border border-emerald-300 bg-emerald-50 px-4 py-2 text-sm text-emerald-900">
            Pending cult reward spade: {String(hasPendingCultSpadesForMe)} remaining. Select a hex to transform (no dwelling build).
          </div>
        )}

        {canInitiateTurnAction && (gameState?.round?.round ?? 0) >= 6 && (
          <div className="mb-3 flex items-center justify-end">
            <button
              data-testid="pass-without-card"
              className="rounded bg-slate-700 px-3 py-2 text-sm font-medium text-white hover:bg-slate-800"
              onClick={handlePassWithoutCard}
            >
              Pass (Final Round)
            </button>
          </div>
        )}

        {/* Faction Selector */}
        {gameState?.phase === GamePhase.FactionSelection && setupMode === 'snellman' && (
          <FactionSelector
            selectedFactions={selectedFactionsMap}
            onSelect={handleFactionSelect}
            isMyTurn={isMyTurn}
            currentPlayerPosition={currentPlayerPosition}
            enableFanFactions={gameState?.enableFanFactions ?? false}
          />
        )}

        {gameState?.phase === GamePhase.FactionSelection && setupMode !== 'snellman' && (
          <div className="mb-4 rounded border border-slate-300 bg-white p-4" data-testid="auction-setup-panel">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-slate-900">
                {setupMode === 'auction' ? 'Auction Setup' : 'Fast Auction Setup'}
              </h2>
              <span className="text-sm text-slate-600">
                {setupMode === 'fast_auction'
                  ? `Waiting: ${pendingDecisionPlayerIds.length > 0 ? pendingDecisionPlayerIds.join(', ') : 'none'}`
                  : `Current: ${(pendingDecisionPlayerId ?? auctionState?.currentBidder ?? '').toString() || 'n/a'}`}
              </span>
            </div>

            <div className="mb-3 text-sm text-slate-700">
              Nominated factions: {(auctionState?.nominationOrder ?? []).length > 0
                ? (auctionState?.nominationOrder ?? []).join(', ')
                : 'none yet'}
            </div>

            {(auctionState?.nominationOrder ?? []).length > 0 && (
              <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-2">
                {(auctionState?.nominationOrder ?? []).map((faction) => (
                  <div key={faction} className="rounded border border-slate-200 px-3 py-2 text-sm text-slate-700">
                    <div className="font-medium">{faction}</div>
                    <div>Current bid: {String(auctionState?.currentBids?.[faction] ?? 0)}</div>
                    <div>Holder: {(auctionState?.factionHolders?.[faction] ?? 'none').toString()}</div>
                  </div>
                ))}
              </div>
            )}

            {(hasPendingDecisionForMe && pendingDecisionType === 'auction_nomination') && (
              <div className="space-y-2">
                <p className="text-sm text-slate-700">Choose one faction to nominate:</p>
                <div className="flex flex-wrap gap-2">
                  {availableAuctionNominationFactions.map((factionType) => (
                    <button
                      key={factionType}
                      data-testid={`auction-nominate-${factionType}`}
                      onClick={() => { handleAuctionNominate(factionType) }}
                      className="rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                      {factionType}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {(hasPendingDecisionForMe && pendingDecisionType === 'auction_bid') && (
              <div className="space-y-2">
                <p className="text-sm text-slate-700">Submit one additional bid amount on a nominated faction:</p>
                {pendingAuctionFactions.map((faction) => (
                  <div key={faction} className="flex items-center gap-2">
                    <span className="w-32 text-sm text-slate-800">{faction}</span>
                    <span className="w-32 text-xs text-slate-500">Current {String(auctionState?.currentBids?.[faction] ?? 0)}</span>
                    <input
                      type="number"
                      data-testid={`auction-bid-input-${faction}`}
                      min={0}
                      max={40}
                      value={auctionBidInputs[faction] ?? 0}
                      onChange={(e) => {
                        const value = Number(e.target.value)
                        setAuctionBidInputs((current) => ({ ...current, [faction]: Number.isFinite(value) ? value : 0 }))
                      }}
                      className="w-24 rounded border border-slate-300 px-2 py-1 text-sm"
                    />
                    <button
                      data-testid={`auction-bid-submit-${faction}`}
                      onClick={() => { handleAuctionBid(faction) }}
                      className="rounded bg-indigo-600 px-3 py-1 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                      Bid
                    </button>
                  </div>
                ))}
              </div>
            )}

            {(isFastAuctionBidDecisionForMe) && (
              <div className="space-y-2">
                <p className="text-sm text-slate-700">Set VP-reduction bids for all nominated factions:</p>
                {pendingAuctionFactions.map((faction) => (
                  <div key={faction} className="flex items-center gap-2">
                    <span className="w-32 text-sm text-slate-800">{faction}</span>
                    <input
                      type="number"
                      data-testid={`fast-auction-bid-input-${faction}`}
                      min={0}
                      max={40}
                      value={fastAuctionBidInputs[faction] ?? 0}
                      onChange={(e) => {
                        const value = Number(e.target.value)
                        setFastAuctionBidInputs((current) => ({ ...current, [faction]: Number.isFinite(value) ? value : 0 }))
                      }}
                      className="w-24 rounded border border-slate-300 px-2 py-1 text-sm"
                    />
                  </div>
                ))}
                <button
                  data-testid="fast-auction-submit"
                  onClick={handleFastAuctionSubmit}
                  className="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                >
                  Submit Fast Bids
                </button>
              </div>
            )}
          </div>
        )}

        <ResponsiveGridLayout
          className={`layout ${isLayoutLocked ? 'layout-locked' : ''}`}
          layouts={layouts}
          breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
          cols={{ lg: 24, md: 20, sm: 12, xs: 8, xxs: 4 }}
          rowHeight={rowHeight}
          onLayoutChange={handleLayoutChange}
          onWidthChange={handleWidthChange}
          isDraggable={!isLayoutLocked}
          isResizable={!isLayoutLocked}
          resizeHandles={['e']}
          draggableHandle=".drag-handle"
        >
          <div
            key="summary"
            style={{
              backgroundColor: '#ffffff',
              borderRadius: '0.5rem',
              boxShadow: '0 0.25rem 0.75rem rgba(0,0,0,0.08)',
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
            }}
          >
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div
              style={{
                flex: '1 1 auto',
                minHeight: 0,
                overflow: 'hidden',
                padding: '0.25rem',
                display: 'flex',
              }}
            >
              {gameState && (
                <PlayerSummaryBar
                  gameState={gameState}
                  localPlayerId={localPlayerId}
                  showIncomePreview={localPlayerOptions.showIncomePreview}
                />
              )}
            </div>
          </div>

          <div key="scoring" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto">
              <ScoringTiles
                tiles={gameState?.scoringTiles?.tiles.map((t) => t.type) || []}
                currentRound={gameState?.round?.round || 1}
              />
            </div>
          </div>

          <div key="board" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto p-4 flex items-center justify-center bg-gray-50">
              <GameBoard
                onHexClick={handleHexClick}
                onBridgeEdgeClick={handleBridgeEdgeClick}
                bridgeEdgeSelectionEnabled={powerMode?.type === 'power_bridge'}
                onPowerActionClick={handlePowerActionClick}
                disablePowerActions={!canInitiateTurnAction}
              />
            </div>
          </div>

          <div key="cult" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto p-2">
              <CultTracks
                cultPositions={cultPositions}
                bonusTiles={priestSpots}
                onBonusTileClick={handleCultSpotClick}
                priestsOnTrack={gameState?.cultTracks?.priestsOnTrack}
                players={gameState?.players as Record<string, { faction: FactionType }>}
              />
            </div>
          </div>

          <div key="towns" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto">
              <TownTiles
                availableTiles={
                  gameState?.townTiles?.available
                    ? Object.entries(gameState.townTiles.available).flatMap(([id, count]) => Array.from({ length: count }, () => Number(id)))
                    : []
                }
                onTileClick={handleTownTileClick}
                isTileClickable={isTownTileClickable}
              />
            </div>
          </div>

          <div key="favor" className="bg-white rounded-lg shadow-md overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto" style={{ flex: 1 }}>
              <FavorTiles onTileClick={handleFavorTileClick} isTileClickable={isFavorTileClickable} />
            </div>
          </div>

          <div key="playerBoards" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-hidden">
              <PlayerBoards
                canUseTurnActions={canInitiateTurnAction}
                canUseConversions={canUseConversionWindow}
                onConversion={handleConversion}
                onBurnPower={handleBurnPower}
                onAdvanceShipping={handleAdvanceShipping}
                onAdvanceDigging={handleAdvanceDigging}
                onAdvanceChashTrack={handleAdvanceChashTrack}
                onStrongholdAction={handleStrongholdAction}
                onEngineersBridgeAction={handleEngineersBridgeAction}
                onMermaidsConnectAction={handleMermaidsConnectAction}
                onWater2Action={handleWater2Action}
                activeStrongholdActionType={activeStrongholdActionType}
                isEngineersBridgeActive={powerMode?.type === 'power_bridge' && powerMode.source === 'engineers'}
                isMermaidsConnectActive={powerMode?.type === 'special_action_target' && powerMode.actionType === SpecialActionType.MermaidsRiverTown}
                isWater2Active={activeWater2Action}
              />
            </div>
          </div>

          <div key="passing" className="bg-white rounded-lg shadow-md overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto" style={{ flex: 1 }}>
              <PassingTiles
                availableCards={availableCards}
                bonusCardCoins={(gameState?.bonusCards?.available ?? {}) as Record<string, number>}
                bonusCardOwners={bonusCardOwners}
                players={gameState?.players}
                passedPlayers={passedPlayers}
                onCardClick={handlePassingTileClick}
                isCardClickable={isPassingCardClickable}
                activeSpecialCardActionType={activeBonusCardActionType}
              />
            </div>
          </div>
        </ResponsiveGridLayout>

        <details className="mt-8 p-4 bg-gray-200 rounded">
          <summary className="font-bold cursor-pointer">Debug: Game State Players</summary>
          <pre className="mt-2 text-xs overflow-auto max-h-96">
            {JSON.stringify(gameState?.players, null, 2)}
          </pre>
        </details>
      </div>

      <Modal
        isOpen={chaosModalOpen}
        onClose={() => { setChaosModalOpen(false) }}
        title="Chaos Magicians Double Turn"
        testId="chaos-double-turn-modal"
        footer={(
          <>
            <button data-testid="chaos-double-turn-cancel" className="px-4 py-2 rounded bg-gray-200 text-gray-800" onClick={() => { setChaosModalOpen(false) }}>Cancel</button>
            <button
              data-testid="chaos-double-turn-submit"
              className="px-4 py-2 rounded bg-blue-600 text-white disabled:bg-blue-300"
              onClick={submitChaosDoubleTurn}
              disabled={chaosFirstParamsError !== null || chaosSecondParamsError !== null}
            >
              Submit
            </button>
          </>
        )}
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-700">Configure two sequential actions. Use JSON params for each action payload.</p>
          <div className="space-y-2">
            <label className="block text-sm font-medium">First action</label>
            <select
              data-testid="chaos-double-turn-first-type"
              className="w-full rounded border px-2 py-1"
              value={chaosFirstType}
              onChange={(e) => {
                const next = e.target.value
                setChaosFirstType(next)
                if (chaosFirstParams.trim() === '{}') {
                  applyChaosTemplate('first', next)
                }
              }}
            >
              {CHAOS_ACTION_TYPES.map((type) => (
                <option key={type} value={type}>{type}</option>
              ))}
            </select>
            <div className="flex items-center justify-between">
              <button
                type="button"
                data-testid="chaos-double-turn-first-template"
                className="rounded bg-gray-200 px-2 py-1 text-xs text-gray-800"
                onClick={() => { applyChaosTemplate('first', chaosFirstType) }}
              >
                Use Template
              </button>
              {chaosFirstParamsError && (
                <span className="text-xs text-red-700">{chaosFirstParamsError}</span>
              )}
            </div>
            <textarea data-testid="chaos-double-turn-first-params" className="w-full rounded border px-2 py-1 font-mono text-xs" rows={4} value={chaosFirstParams} onChange={(e) => { setChaosFirstParams(e.target.value) }} />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium">Second action</label>
            <select
              data-testid="chaos-double-turn-second-type"
              className="w-full rounded border px-2 py-1"
              value={chaosSecondType}
              onChange={(e) => {
                const next = e.target.value
                setChaosSecondType(next)
                if (chaosSecondParams.trim() === '{}') {
                  applyChaosTemplate('second', next)
                }
              }}
            >
              {CHAOS_ACTION_TYPES.map((type) => (
                <option key={type} value={type}>{type}</option>
              ))}
            </select>
            <div className="flex items-center justify-between">
              <button
                type="button"
                data-testid="chaos-double-turn-second-template"
                className="rounded bg-gray-200 px-2 py-1 text-xs text-gray-800"
                onClick={() => { applyChaosTemplate('second', chaosSecondType) }}
              >
                Use Template
              </button>
              {chaosSecondParamsError && (
                <span className="text-xs text-red-700">{chaosSecondParamsError}</span>
              )}
            </div>
            <textarea data-testid="chaos-double-turn-second-params" className="w-full rounded border px-2 py-1 font-mono text-xs" rows={4} value={chaosSecondParams} onChange={(e) => { setChaosSecondParams(e.target.value) }} />
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={conspiratorsSwapModalOpen}
        onClose={closeConspiratorsSwapModal}
        title="Conspirators Swap Favor"
        testId="conspirators-swap-favor-modal"
        footer={(
          <>
            <button
              data-testid="conspirators-swap-favor-cancel"
              className="px-4 py-2 rounded bg-gray-200 text-gray-800"
              onClick={closeConspiratorsSwapModal}
            >
              Cancel
            </button>
            <button
              data-testid="conspirators-swap-favor-submit"
              className="px-4 py-2 rounded bg-blue-600 text-white disabled:bg-blue-300"
              onClick={submitConspiratorsSwap}
              disabled={conspiratorsReturnTile === '' || conspiratorsNewTile === ''}
            >
              Swap
            </button>
          </>
        )}
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-700">Return one of your favor tiles to take a different favor tile from the supply.</p>
          <div className="space-y-2">
            <label className="block text-sm font-medium" htmlFor="conspirators-return-tile">Return tile</label>
            <select
              id="conspirators-return-tile"
              data-testid="conspirators-return-tile"
              className="w-full rounded border px-2 py-1"
              value={conspiratorsReturnTile}
              onChange={(e) => { setConspiratorsReturnTile(e.target.value === '' ? '' : Number(e.target.value)) }}
            >
              <option value="">Select a favor tile</option>
              {localPlayerFavorTiles.map((tile) => (
                <option key={tile} value={tile}>{favorTileLabel(tile)}</option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium" htmlFor="conspirators-new-tile">Take tile</label>
            <select
              id="conspirators-new-tile"
              data-testid="conspirators-new-tile"
              className="w-full rounded border px-2 py-1"
              value={conspiratorsNewTile}
              onChange={(e) => { setConspiratorsNewTile(e.target.value === '' ? '' : Number(e.target.value)) }}
            >
              <option value="">Select a favor tile</option>
              {conspiratorsNewTileOptions.map((tile) => (
                <option key={tile} value={tile}>{favorTileLabel(tile)}</option>
              ))}
            </select>
          </div>
        </div>
      </Modal>

    </div>
  )
}
