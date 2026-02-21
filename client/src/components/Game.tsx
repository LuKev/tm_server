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

const ResponsiveGridLayout = WidthProvider(Responsive)

type ConfirmDialog = {
  title: string
  message: string
  onConfirm: () => void
}

type PendingPowerMode =
  | { type: 'power_spade'; actionType: PowerActionType }
  | { type: 'power_bridge'; source: 'power' | 'engineers'; firstHex: { q: number; r: number } | null }
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
  send_priest: '{\n  "cultTrack": 0,\n  "spacesToClimb": 1\n}',
  power_action_claim: '{\n  "actionType": 1\n}',
  special_action_use: '{\n  "specialActionType": 0\n}',
  pass: '{\n  "bonusCard": 0\n}',
  conversion: '{\n  "conversionType": "worker_to_coin",\n  "amount": 1\n}',
  burn_power: '{\n  "amount": 1\n}',
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
    if (isConnected && gameId && !gameState) {
      sendMessage({ type: 'get_game_state', payload: { gameID: gameId, playerID: localPlayerId } })
    }
  }, [isConnected, gameId, gameState, localPlayerId, sendMessage])

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

  const queueConfirm = (title: string, message: string, onConfirm: () => void): void => {
    setConfirmDialog({ title, message, onConfirm })
  }

  const performAction = (type: string, params: Record<string, unknown> = {}): void => {
    if (!gameId) return
    submitAction(gameId, type, params)
  }

  const currentPlayerId = gameState?.turnOrder?.[gameState.currentTurn]
  const isMyTurn = !!localPlayerId && currentPlayerId === localPlayerId

  const localPlayer = useMemo(() => {
    if (!localPlayerId || !gameState?.players) return null
    return gameState.players[localPlayerId] ?? null
  }, [gameState?.players, localPlayerId])

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
  const hasPendingDecisionForMe = !!localPlayerId && pendingDecisionPlayerId === localPlayerId
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

  useEffect(() => {
    if (!(hasPendingDecisionForMe && pendingDecisionType === 'town_cult_top_choice')) {
      setSelectedTownCultTracks([])
      return
    }
    setSelectedTownCultTracks((current) =>
      current.filter((track) => pendingTownCultTopCandidates.includes(track)),
    )
  }, [hasPendingDecisionForMe, pendingDecisionType, pendingTownCultTopCandidates])

  const setupDwellingPlayerId = useMemo(() => {
    if (!gameState?.setupDwellingOrder) return null
    const idx = gameState.setupDwellingIndex ?? -1
    if (idx < 0 || idx >= gameState.setupDwellingOrder.length) return null
    return gameState.setupDwellingOrder[idx] ?? null
  }, [gameState?.setupDwellingOrder, gameState?.setupDwellingIndex])

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
    const cards = Object.entries(gameState?.bonusCards?.available ?? {})
      .map(([k]) => Number(k))
      .filter((card) => Number.isInteger(card) && card >= 0)
    return cards.sort((a, b) => a - b)
  }, [gameState?.bonusCards?.available])

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
      queueConfirm('Confirm Setup Dwelling', `Place setup dwelling at (${String(q)},${String(r)})?`, () => {
        submitSetupDwelling(localPlayerId, q, r, gameId)
        setConfirmDialog(null)
      })
      return
    }

    if (hasPendingDecisionForMe && pendingDecisionType === 'halflings_spades') {
      setPendingHex({ q, r })
      setHexActionMode('transform_only')
      setSelectedTerrain(localHomeTerrain)
      return
    }

    if (hasPendingCultSpadesForMe > 0 && isMyTurn) {
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
        `Build bridge from (${String(from.q)},${String(from.r)}) to (${String(to.q)},${String(to.r)})?`,
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
        queueConfirm('Confirm Witches Ride', `Use Witches Ride on (${String(q)},${String(r)})?`, () => {
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
        queueConfirm('Confirm Swarmlings Upgrade', `Use Swarmlings free upgrade at (${String(q)},${String(r)})?`, () => {
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
        queueConfirm('Confirm Mermaids Connect', `Connect river town at (${String(q)},${String(r)})?`, () => {
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

    if (!isMyTurn || !isActionPhase(gameState.phase) || hasPendingDecisionForMe) return

    const hex = gameState.map?.hexes?.[`${String(q)},${String(r)}`]
    if (!hex) return

    if (hex.building?.ownerPlayerId === localPlayerId) {
      setUpgradeHex({ q, r })
      return
    }

    openHexActionModal(q, r)
  }

  const handleBridgeEdgeClick = (from: { q: number; r: number }, to: { q: number; r: number }): void => {
    if (powerMode?.type !== 'power_bridge') return

    queueConfirm(
      'Confirm Bridge',
      `Build bridge from (${String(from.q)},${String(from.r)}) to (${String(to.q)},${String(to.r)})?`,
      () => {
        performAction(powerMode.source === 'engineers' ? 'engineers_bridge' : 'power_bridge_place', { bridgeHex1: from, bridgeHex2: to })
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

  const handlePowerActionClick = (action: PowerActionType): void => {
    if (!isMyTurn || hasPendingDecisionForMe || !isActionPhase(gameState?.phase)) return

    if (action === PowerActionType.Bridge) {
      setPowerMode({ type: 'power_bridge', source: 'power', firstHex: null })
      return
    }

    if (action === PowerActionType.Spade || action === PowerActionType.DoubleSpade) {
      setPowerMode({ type: 'power_spade', actionType: action })
      return
    }

    queueConfirm('Confirm Power Action', `Use power action ${PowerActionType[action]}?`, () => {
      performAction('power_action_claim', { actionType: action })
      setConfirmDialog(null)
    })
  }

  const handleCultSpotClick = (cult: CultType, tileIndex: number): void => {
    if (!isMyTurn || hasPendingDecisionForMe) return

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
    if (!gameState || playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe) return

    queueConfirm('Confirm Conversion', `Execute conversion: ${conversionType}?`, () => {
      performAction('conversion', { conversionType, amount: 1 })
      setConfirmDialog(null)
    })
  }

  const handleBurnPower = (playerId: string, amount: number): void => {
    if (!gameState || playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe) return

    queueConfirm('Confirm Burn Power', `Burn ${String(amount * 2)} power from Bowl II to gain ${String(amount)} power in Bowl III?`, () => {
      performAction('burn_power', { amount })
      setConfirmDialog(null)
    })
  }

  const handleAdvanceShipping = (playerId: string): void => {
    if (playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe) return

    queueConfirm('Confirm Shipping Upgrade', 'Upgrade shipping track?', () => {
      performAction('advance_shipping')
      setConfirmDialog(null)
    })
  }

  const handleAdvanceDigging = (playerId: string): void => {
    if (playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe) return

    queueConfirm('Confirm Digging Upgrade', 'Upgrade digging track?', () => {
      performAction('advance_digging')
      setConfirmDialog(null)
    })
  }

  const handleEngineersBridgeAction = (playerId: string): void => {
    if (playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe || !isActionPhase(gameState?.phase)) return
    setPowerMode({ type: 'power_bridge', source: 'engineers', firstHex: null })
  }

  const handleMermaidsConnectAction = (playerId: string): void => {
    if (playerId !== localPlayerId || !isMyTurn || hasPendingDecisionForMe || !isActionPhase(gameState?.phase)) return
    setPowerMode({ type: 'special_action_target', actionType: SpecialActionType.MermaidsRiverTown })
  }

  const handleStrongholdAction = (_playerId: string, actionType: SpecialActionType): void => {
    if (!isMyTurn || hasPendingDecisionForMe) return

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

    queueConfirm('Confirm Special Action', `Use special action ${SpecialActionType[actionType]}?`, () => {
      performAction('special_action_use', { specialActionType: actionType })
      setConfirmDialog(null)
    })
  }

  const handleWater2Action = (_playerId: string): void => {
    if (!isMyTurn || hasPendingDecisionForMe) return
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

    if (isOwnedByMe && isMyTurn && isActionPhase(gameState.phase)) {
      if (cardType === BonusCardType.Spade) {
        setPowerMode({ type: 'special_action_target', actionType: SpecialActionType.BonusCardSpade })
        return
      }
      if (cardType === BonusCardType.CultAdvance) {
        setCultChoiceContext('bonus_cult')
      }
      return
    }

    if (!isMyTurn || !isActionPhase(gameState.phase) || hasPendingDecisionForMe) return
    if (owner) return

    const warning = hasUnspentOptionalActions
      ? ' You still have optional special actions or pending spades available.'
      : ''
    queueConfirm('Confirm Pass', `Pass and take ${bonusCardLabel(cardType)}?${warning}`, () => {
      performAction('pass', { bonusCard: cardType })
      setConfirmDialog(null)
    })
  }

  const isPassingCardClickable = (cardType: BonusCardType): boolean => {
    if (!gameState || !localPlayerId) return false
    const owner = bonusCardOwners[String(cardType)]

    if (pendingDecisionType === 'setup_bonus_card') {
      return hasPendingDecisionForMe && !owner
    }

    if (owner === localPlayerId && isMyTurn && isActionPhase(gameState.phase)) {
      if (cardType === BonusCardType.Spade || cardType === BonusCardType.CultAdvance) {
        return true
      }
    }

    if (!isMyTurn || !isActionPhase(gameState.phase) || hasPendingDecisionForMe) return false
    return !owner
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

  const handleTownTileClick = (tileId: TownTileId): void => {
    if (!(hasPendingDecisionForMe && pendingDecisionType === 'town_tile_selection')) return

    queueConfirm('Confirm Town Tile', `Take ${townTileLabel(tileId)}?`, () => {
      performAction('select_town_tile', { tileType: tileId })
      setConfirmDialog(null)
    })
  }

  const isTownTileClickable = (_tileId: TownTileId, count: number): boolean => {
    if (count <= 0) return false
    return hasPendingDecisionForMe && pendingDecisionType === 'town_tile_selection' && !!localPlayerId
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

  const closeHexModal = (): void => {
    setPendingHex(null)
    setHexActionMode(null)
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
      performAction('use_cult_spade', { hex: pendingHex })
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
    <div className="min-h-screen p-4 bg-gray-100">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-3xl font-bold text-gray-800">Terra Mystica - Game {gameId}</h1>
          <div className="flex gap-2">
            <button
              onClick={() => { setIsLayoutLocked(!isLayoutLocked) }}
              className={`px-4 py-2 rounded text-sm font-medium transition-colors ${isLayoutLocked
                ? 'bg-blue-100 text-blue-700 hover:bg-blue-200'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
            >
              {isLayoutLocked ? 'Unlock Layout' : 'Lock Layout'}
            </button>
            <button
              onClick={resetLayout}
              className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-sm font-medium text-gray-700 transition-colors"
            >
              Reset Layout
            </button>
          </div>
        </div>

        {errorMessage && (
          <div className="mb-4 rounded border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800">
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
            Bridge mode: first endpoint selected at ({String(powerMode.firstHex.q)},{String(powerMode.firstHex.r)}). Click the second endpoint.
          </div>
        )}

        {(powerMode?.type === 'power_spade' || powerMode?.type === 'special_action_target') && (
          <div className="mb-3 rounded border border-blue-300 bg-blue-50 px-4 py-2 text-sm text-blue-800">
            Click a target hex.
          </div>
        )}

        {(hasPendingSpadesForMe > 0 && isMyTurn) && (
          <div className="mb-3 flex items-center justify-between rounded border border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900">
            <span>
              Pending spade follow-up: {String(hasPendingSpadesForMe)} spade(s) remaining.
              {canBuildWithPendingSpadeForMe ? '' : ' Dwelling build on this follow-up is not allowed.'}
            </span>
            <button
              className="rounded bg-amber-700 px-3 py-1 text-white"
              onClick={() => { performAction('discard_pending_spade', { count: 1 }) }}
            >
              Discard 1 Spade
            </button>
          </div>
        )}

        {(hasPendingCultSpadesForMe > 0 && isMyTurn) && (
          <div className="mb-3 rounded border border-emerald-300 bg-emerald-50 px-4 py-2 text-sm text-emerald-900">
            Pending cult reward spade: {String(hasPendingCultSpadesForMe)} remaining. Select a hex to transform (no dwelling build).
          </div>
        )}

        {/* Faction Selector */}
        {gameState?.phase === GamePhase.FactionSelection && (
          <FactionSelector
            selectedFactions={selectedFactionsMap}
            onSelect={handleFactionSelect}
            isMyTurn={isMyTurn}
            currentPlayerPosition={currentPlayerPosition}
          />
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
              {gameState && <PlayerSummaryBar gameState={gameState} />}
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
                onConversion={handleConversion}
                onBurnPower={handleBurnPower}
                onAdvanceShipping={handleAdvanceShipping}
                onAdvanceDigging={handleAdvanceDigging}
                onStrongholdAction={handleStrongholdAction}
                onEngineersBridgeAction={handleEngineersBridgeAction}
                onMermaidsConnectAction={handleMermaidsConnectAction}
                onWater2Action={handleWater2Action}
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
        isOpen={confirmDialog !== null}
        onClose={() => { setConfirmDialog(null) }}
        title={confirmDialog?.title ?? 'Confirm Action'}
        footer={(
          <>
            <button className="px-4 py-2 rounded bg-gray-200 text-gray-800" onClick={() => { setConfirmDialog(null) }}>Cancel</button>
            <button
              className="px-4 py-2 rounded bg-blue-600 text-white"
              onClick={() => {
                if (!confirmDialog) return
                confirmDialog.onConfirm()
              }}
            >
              Confirm
            </button>
          </>
        )}
      >
        <p>{confirmDialog?.message}</p>
      </Modal>

      <Modal
        isOpen={pendingHex !== null}
        onClose={() => {
          closeHexModal()
          if (powerMode?.type === 'power_spade' || powerMode?.type === 'special_action_target') {
            setPowerMode(null)
          }
        }}
        title={isHalflingsSpadeDecision ? 'Apply Halflings Spade' : isCultSpadeDecision ? 'Use Cult Spade' : 'Hex Action'}
        footer={(
          <>
            <button className="px-4 py-2 rounded bg-gray-200 text-gray-800" onClick={closeHexModal}>Cancel</button>
            <button className="px-4 py-2 rounded bg-blue-600 text-white" onClick={submitHexModalAction}>Submit</button>
          </>
        )}
      >
        <div className="space-y-3">
          <p>
            Selected hex: {pendingHex ? `(${String(pendingHex.q)}, ${String(pendingHex.r)})` : ''}
          </p>

          {isHalflingsSpadeDecision && (
            <>
              <p>Choose terrain to transform to.</p>
              <select
                value={selectedTerrain}
                onChange={(e) => { setSelectedTerrain(Number(e.target.value) as TerrainType) }}
                className="w-full rounded border px-2 py-1"
              >
                {TERRAIN_CHOICES.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            </>
          )}

          {isCultSpadeDecision && (
            <p>This action uses one cult reward spade and cannot build a dwelling.</p>
          )}

          {!isHalflingsSpadeDecision && !isCultSpadeDecision && (
            <>
              {isPendingSpadeDecision && !canBuildWithPendingSpadeForMe ? (
                <div className="rounded border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
                  This follow-up spade can only transform terrain (no dwelling build allowed).
                </div>
              ) : (
                <div className="space-y-2">
                  <label className="block text-sm font-medium">Action</label>
                  <select
                    value={hexActionMode ?? 'build'}
                    onChange={(e) => { setHexActionMode(e.target.value as 'build' | 'transform_build' | 'transform_only') }}
                    className="w-full rounded border px-2 py-1"
                  >
                    <option value="build">Build dwelling</option>
                    <option value="transform_build">Transform and build</option>
                    <option value="transform_only">Transform only</option>
                  </select>
                </div>
              )}

              {(hexActionMode === 'transform_build' || hexActionMode === 'transform_only') && (
                <div className="space-y-2">
                  <label className="block text-sm font-medium">Target terrain</label>
                  <select
                    value={selectedTerrain}
                    onChange={(e) => { setSelectedTerrain(Number(e.target.value) as TerrainType) }}
                    className="w-full rounded border px-2 py-1"
                  >
                    {TERRAIN_CHOICES.map((t) => (
                      <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>
              )}
            </>
          )}
        </div>
      </Modal>

      <Modal
        isOpen={upgradeHex !== null}
        onClose={() => { setUpgradeHex(null) }}
        title="Upgrade Building"
      >
        <div className="space-y-2">
          {upgradeOptions.length === 0 && <p>No legal upgrades for this building.</p>}
          {upgradeOptions.map((opt) => (
            <button
              key={opt.type}
              type="button"
              className="w-full rounded border px-3 py-2 text-left hover:bg-gray-100"
              onClick={() => { selectUpgrade(opt.type) }}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </Modal>

      <Modal
        isOpen={cultChoiceContext !== null}
        onClose={() => { setCultChoiceContext(null) }}
        title="Choose Cult Track"
      >
        <div className="grid grid-cols-2 gap-2">
          {CULT_CHOICES.map((choice) => (
            <button
              key={choice.track}
              type="button"
              className="rounded border px-3 py-2 hover:bg-gray-100"
              onClick={() => { submitCultChoice(choice.track) }}
            >
              {choice.label}
            </button>
          ))}
        </div>
      </Modal>

      <Modal
        isOpen={chaosModalOpen}
        onClose={() => { setChaosModalOpen(false) }}
        title="Chaos Magicians Double Turn"
        footer={(
          <>
            <button className="px-4 py-2 rounded bg-gray-200 text-gray-800" onClick={() => { setChaosModalOpen(false) }}>Cancel</button>
            <button
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
                className="rounded bg-gray-200 px-2 py-1 text-xs text-gray-800"
                onClick={() => { applyChaosTemplate('first', chaosFirstType) }}
              >
                Use Template
              </button>
              {chaosFirstParamsError && (
                <span className="text-xs text-red-700">{chaosFirstParamsError}</span>
              )}
            </div>
            <textarea className="w-full rounded border px-2 py-1 font-mono text-xs" rows={4} value={chaosFirstParams} onChange={(e) => { setChaosFirstParams(e.target.value) }} />
          </div>
          <div className="space-y-2">
            <label className="block text-sm font-medium">Second action</label>
            <select
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
                className="rounded bg-gray-200 px-2 py-1 text-xs text-gray-800"
                onClick={() => { applyChaosTemplate('second', chaosSecondType) }}
              >
                Use Template
              </button>
              {chaosSecondParamsError && (
                <span className="text-xs text-red-700">{chaosSecondParamsError}</span>
              )}
            </div>
            <textarea className="w-full rounded border px-2 py-1 font-mono text-xs" rows={4} value={chaosSecondParams} onChange={(e) => { setChaosSecondParams(e.target.value) }} />
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'town_cult_top_choice'}
        onClose={() => { }}
        title="Choose Cults To Top"
      >
        <div className="space-y-3">
          <p>
            Choose {String(pendingTownCultTopMaxSelections)} track(s) to advance to 10.
          </p>
          <div className="grid grid-cols-2 gap-2">
            {pendingTownCultTopCandidates.map((track) => (
              <button
                key={track}
                type="button"
                className={`rounded border px-3 py-2 text-left ${selectedTownCultTracks.includes(track) ? 'bg-blue-100 border-blue-400' : 'hover:bg-gray-100'}`}
                onClick={() => { toggleTownCultTrack(track) }}
              >
                {CULT_CHOICES.find((choice) => choice.track === track)?.label ?? `Cult ${String(track)}`}
              </button>
            ))}
          </div>
          <button
            type="button"
            className="rounded bg-blue-600 px-3 py-2 text-white disabled:bg-blue-300"
            disabled={selectedTownCultTracks.length !== pendingTownCultTopMaxSelections}
            onClick={() => {
              performAction('select_town_cult_top', { tracks: selectedTownCultTracks })
            }}
          >
            Confirm selection
          </button>
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'leech_offer'}
        onClose={() => { }}
        title="Leech Offer"
      >
        <div className="space-y-3">
          {((pendingDecision?.offers as Array<Record<string, unknown>> | undefined) ?? []).map((offer, idx) => {
            const amount = Number(offer.Amount ?? offer.amount ?? 0)
            const vpCost = Math.max(0, amount - 1)
            return (
              <div key={idx} className="rounded border p-3">
                <p>Accept {String(amount)} power for {String(vpCost)} VP?</p>
                <div className="mt-2 flex gap-2">
                  <button className="rounded bg-green-600 px-3 py-1 text-white" onClick={() => { performAction('accept_leech', { offerIndex: idx }) }}>Accept</button>
                  <button className="rounded bg-gray-600 px-3 py-1 text-white" onClick={() => { performAction('decline_leech', { offerIndex: idx }) }}>Decline</button>
                </div>
              </div>
            )
          })}
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'setup_bonus_card'}
        onClose={() => { }}
        title="Choose Setup Bonus Card"
      >
        <div className="grid grid-cols-2 gap-2">
          {setupBonusCards.map((cardId) => (
            <button
              key={cardId}
              type="button"
              className="rounded border px-3 py-2 hover:bg-gray-100"
              onClick={() => { performAction('setup_bonus_card', { bonusCard: cardId }) }}
            >
              {bonusCardLabel(cardId)}
            </button>
          ))}
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'darklings_ordination'}
        onClose={() => { }}
        title="Darklings Ordination"
      >
        <div className="grid grid-cols-4 gap-2">
          {[0, 1, 2, 3].map((count) => (
            <button
              key={count}
              type="button"
              className="rounded border px-3 py-2 hover:bg-gray-100"
              onClick={() => { performAction('darklings_ordination', { workersToConvert: count }) }}
            >
              {count}
            </button>
          ))}
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'halflings_spades' && Number((gameState?.pendingHalflingsSpades as Record<string, unknown> | undefined)?.spadesRemaining ?? 0) === 0}
        onClose={() => { }}
        title="Halflings Optional Dwelling"
      >
        <div className="space-y-3">
          <p>All spades used. Build one dwelling on a transformed hex or skip.</p>
          <div className="grid grid-cols-2 gap-2">
            {transformedHalflingsHexes.map((h) => (
              <button
                key={`${String(h.q)},${String(h.r)}`}
                type="button"
                className="rounded border px-3 py-2 hover:bg-gray-100"
                onClick={() => { performAction('halflings_build_dwelling', { targetHex: h }) }}
              >
                Build at ({String(h.q)},{String(h.r)})
              </button>
            ))}
          </div>
          <button className="rounded bg-gray-600 px-3 py-2 text-white" onClick={() => { performAction('halflings_skip_dwelling') }}>
            Skip dwelling
          </button>
        </div>
      </Modal>

      <Modal
        isOpen={hasPendingDecisionForMe && pendingDecisionType === 'cultists_cult_choice'}
        onClose={() => { }}
        title="Cultists: Choose Cult Track"
      >
        <div className="grid grid-cols-2 gap-2">
          {CULT_CHOICES.map((choice) => (
            <button
              key={choice.track}
              type="button"
              className="rounded border px-3 py-2 hover:bg-gray-100"
              onClick={() => { performAction('select_cultists_track', { cultTrack: choice.track }) }}
            >
              {choice.label}
            </button>
          ))}
        </div>
      </Modal>
    </div>
  )
}
