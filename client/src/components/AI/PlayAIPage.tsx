import { useEffect, useMemo, useState, type ReactElement } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Bot, Loader2, Play } from 'lucide-react'
import { useWebSocket } from '../../services/WebSocketContext'
import { useGameStore } from '../../stores/gameStore'
import { DEFAULT_MAP_CATALOG } from '../../data/mapCatalog'
import { FACTIONS } from '../../data/factions'
import type { FactionType } from '../../types/game.types'
import type { MapSummary } from '../../types/map.types'
import {
  DEFAULT_HUMAN_FACTION,
  DEFAULT_MODEL_FACTION,
  MODEL_STRENGTHS,
  generatedAIPlayerName,
  type ModelStrength,
} from './modelGame'
import './PlayAIPage.css'

interface LobbyMessage {
  type: string
  payload?: unknown
}

interface StartedGamePayload {
  gameId?: string
  playerId?: string
}

type LobbyErrorPayload = string | {
  error?: string
  gameId?: string
}

function formatAIError(payload: LobbyErrorPayload): string {
  if (typeof payload === 'string') {
    return payload || 'AI game failed.'
  }
  switch (payload.error) {
    case 'already_in_game':
      return 'A previous open game is still holding this player seat. Try again.'
    case 'invalid_map':
      return 'Select a valid map.'
    case 'game_full':
      return 'The AI game filled unexpectedly.'
    default:
      return 'AI game failed.'
  }
}

export function PlayAIPage(): ReactElement {
  const { isConnected, sendMessage, lastMessage, connectionStatus } = useWebSocket()
  const navigate = useNavigate()
  const [availableMaps, setAvailableMaps] = useState<MapSummary[]>(DEFAULT_MAP_CATALOG)
  const [mapId, setMapId] = useState('base')
  const [humanFaction, setHumanFaction] = useState<FactionType>(DEFAULT_HUMAN_FACTION)
  const [modelFaction, setModelFaction] = useState<FactionType>(DEFAULT_MODEL_FACTION)
  const [modelStrength, setModelStrength] = useState<ModelStrength>('balanced')
  const [enableFanFactions, setEnableFanFactions] = useState(false)
  const [enableFireIceFactions, setEnableFireIceFactions] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [starting, setStarting] = useState(false)

  const selectableFactions = useMemo(() => FACTIONS.filter((faction) => {
    if (faction.isFanFaction && !enableFanFactions) return false
    if (faction.isFireIceFaction && !enableFireIceFactions) return false
    return true
  }), [enableFanFactions, enableFireIceFactions])
  const playableMaps = useMemo(() => availableMaps.filter((map) => map.id !== 'custom'), [availableMaps])

  const factionSelectionValid = humanFaction !== modelFaction

  useEffect(() => {
    if (isConnected) {
      sendMessage({ type: 'list_games' })
    }
  }, [isConnected, sendMessage])

  useEffect(() => {
    if (lastMessage === null || typeof lastMessage !== 'object' || !('type' in lastMessage)) return
    const msg = lastMessage as LobbyMessage
    if (msg.type === 'available_maps') {
      setAvailableMaps(Array.isArray(msg.payload) ? msg.payload as MapSummary[] : DEFAULT_MAP_CATALOG)
      return
    }
    if (msg.type === 'error') {
      setStarting(false)
      setError(formatAIError((msg.payload ?? '') as LobbyErrorPayload))
      return
    }
    if (msg.type === 'action_rejected') {
      const payload = (msg.payload ?? {}) as Record<string, unknown>
      setStarting(false)
      setError((payload.message as string | undefined) ?? (payload.error as string | undefined) ?? 'AI game failed.')
      return
    }
    if (msg.type === 'model_game_started') {
      const payload = (msg.payload ?? {}) as StartedGamePayload
      if (payload.playerId && payload.gameId) {
        useGameStore.getState().bindLocalPlayerToGame(payload.gameId, payload.playerId)
      } else if (payload.playerId) {
        useGameStore.getState().setLocalPlayerId(payload.playerId)
      }
      if (payload.gameId) {
        setError(null)
        void navigate(`/game/${payload.gameId}`)
      }
    }
  }, [lastMessage, navigate])

  useEffect(() => {
    if (selectableFactions.length === 0) return
    if (!selectableFactions.some((faction) => faction.id === humanFaction)) {
      setHumanFaction(selectableFactions[0].id)
    }
    if (!selectableFactions.some((faction) => faction.id === modelFaction)) {
      setModelFaction(selectableFactions.find((faction) => faction.id !== humanFaction)?.id ?? selectableFactions[0].id)
    }
  }, [humanFaction, modelFaction, selectableFactions])

  useEffect(() => {
    if (playableMaps.length === 0) return
    if (!playableMaps.some((map) => map.id === mapId)) {
      setMapId(playableMaps[0].id)
    }
  }, [mapId, playableMaps])

  useEffect(() => {
    if (humanFaction !== modelFaction) return
    const replacement = selectableFactions.find((faction) => faction.id !== humanFaction)
    if (replacement) {
      setModelFaction(replacement.id)
    }
  }, [humanFaction, modelFaction, selectableFactions])

  const startGame = (): void => {
    if (!isConnected || starting || !factionSelectionValid) return
    const playerName = generatedAIPlayerName()
    useGameStore.getState().setLocalPlayerId(playerName)
    setStarting(true)
    setError(null)
    sendMessage({
      type: 'create_and_start_model_game',
      payload: {
        name: 'AI Game',
        maxPlayers: 2,
        creator: playerName,
        mapId,
        enableFanFactions,
        enableFireIceFactions,
        fireIceScoring: 'off',
        modelOpponent: {
          enabled: true,
          humanFaction,
          botFaction: modelFaction,
          simulations: MODEL_STRENGTHS[modelStrength].simulations,
          cpuct: 1.5,
          temperature: 0,
          maxDepth: 500,
          moveDelayMs: 350,
        },
      },
    })
  }

  return (
    <main className="play-ai-page" data-testid="play-ai-screen">
      <section className="play-ai-shell">
        <header className="play-ai-header">
          <button className="play-ai-back" onClick={() => { void navigate('/') }}>
            <ArrowLeft size={18} />
            <span>Lobby</span>
          </button>
          <div className="play-ai-title-row">
            <Bot size={34} />
            <h1>Play vs AI</h1>
          </div>
          <div className={`play-ai-status play-ai-status-${connectionStatus}`}>{connectionStatus}</div>
        </header>

        <section className="play-ai-panel">
          <div className="play-ai-grid">
            <label className="play-ai-field">
              <span>Map</span>
              <select
                data-testid="ai-map-id"
                value={mapId}
                onChange={(event) => { setMapId(event.target.value) }}
                disabled={!isConnected || starting}
              >
                {playableMaps.map((map) => (
                  <option key={map.id} value={map.id}>{map.name}</option>
                ))}
              </select>
            </label>

            <label className="play-ai-field">
              <span>Your faction</span>
              <select
                data-testid="ai-human-faction"
                value={humanFaction}
                onChange={(event) => { setHumanFaction(Number(event.target.value) as FactionType) }}
                disabled={!isConnected || starting}
              >
                {selectableFactions.map((faction) => (
                  <option key={faction.id} value={faction.id}>{faction.name}</option>
                ))}
              </select>
            </label>

            <label className="play-ai-field">
              <span>AI faction</span>
              <select
                data-testid="ai-model-faction"
                value={modelFaction}
                onChange={(event) => { setModelFaction(Number(event.target.value) as FactionType) }}
                disabled={!isConnected || starting}
              >
                {selectableFactions.map((faction) => (
                  <option key={faction.id} value={faction.id} disabled={faction.id === humanFaction}>{faction.name}</option>
                ))}
              </select>
            </label>

            <label className="play-ai-field">
              <span>Search budget</span>
              <select
                data-testid="ai-model-strength"
                value={modelStrength}
                onChange={(event) => { setModelStrength(event.target.value as ModelStrength) }}
                disabled={!isConnected || starting}
              >
                {(Object.entries(MODEL_STRENGTHS) as [ModelStrength, { label: string; simulations: number }][]).map(([value, config]) => (
                  <option key={value} value={value}>{config.label}</option>
                ))}
              </select>
            </label>
          </div>

          <div className="play-ai-options">
            <label>
              <input
                type="checkbox"
                data-testid="ai-enable-fire-ice-factions"
                checked={enableFireIceFactions}
                onChange={(event) => { setEnableFireIceFactions(event.target.checked) }}
                disabled={!isConnected || starting}
              />
              <span>F&amp;I factions</span>
            </label>
            <label>
              <input
                type="checkbox"
                data-testid="ai-enable-fan-factions"
                checked={enableFanFactions}
                onChange={(event) => { setEnableFanFactions(event.target.checked) }}
                disabled={!isConnected || starting}
              />
              <span>Fan factions</span>
            </label>
          </div>

          {error !== null && <div className="play-ai-error" role="alert">{error}</div>}
          {!factionSelectionValid && <div className="play-ai-error" role="alert">Choose two different factions.</div>}

          <button
            className="play-ai-start"
            data-testid="ai-start-game"
            onClick={startGame}
            disabled={!isConnected || starting || !factionSelectionValid}
          >
            {starting ? <Loader2 size={20} className="play-ai-spin" /> : <Play size={20} />}
            <span>{starting ? 'Starting' : 'Start Game'}</span>
          </button>
        </section>
      </section>
    </main>
  )
}
