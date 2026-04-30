import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useWebSocket } from '../services/WebSocketContext'
import { useGameStore } from '../stores/gameStore'
import { DEFAULT_MAP_CATALOG } from '../data/mapCatalog'
import type { CustomMapDefinition, MapSummary } from '../types/map.types'
import { CustomMapEditor } from './CustomMapEditor'
import { buildCustomMapHexes, createEmptyCustomMapDefinition } from '../utils/customMapUtils'
import { HexGridCanvas } from './GameBoard/HexGridCanvas'
import './Lobby.css'

interface GameInfo {
  id: string
  name: string
  host: string
  mapId: string
  enableFanFactions?: boolean
  enableFireIceFactions?: boolean
  fireIceScoring?: 'off' | 'on' | 'random'
  customMap?: CustomMapDefinition
  started?: boolean
  players: string[]
  maxPlayers: number
}

interface LobbyMessage {
  type: string
  payload?: unknown
}

type LobbyErrorPayload = string | {
  error?: string
  gameId?: string
}

function formatLobbyError(payload: LobbyErrorPayload): string {
  if (typeof payload === 'string') {
    switch (payload) {
      case 'not_in_game':
        return 'You are not seated in that game.'
      case 'game_not_found':
        return 'That game no longer exists.'
      default:
        return payload
    }
  }

  switch (payload.error) {
    case 'already_in_game':
      return payload.gameId ? `Leave game ${payload.gameId} before joining another open game.` : 'Leave your current open game before joining another.'
    case 'game_full':
      return 'That game is already full.'
    case 'game_started':
      return 'That game has already started.'
    case 'game_not_found':
      return 'That game no longer exists.'
    case 'not_in_game':
      return 'You are not seated in that game.'
    case 'invalid_map':
      return 'Select a valid map.'
    default:
      return 'Lobby action failed.'
  }
}

export function Lobby(): React.ReactElement {
  const { isConnected, sendMessage, lastMessage, connectionStatus } = useWebSocket()
  const navigate = useNavigate()
  const gameState = useGameStore((state) => state.gameState)
  const storedLocalPlayerId = useGameStore((state) => state.localPlayerId)
  const [playerName, setPlayerName] = useState('')
  const [games, setGames] = useState<GameInfo[]>([])
  const [newGameName, setNewGameName] = useState('')
  const [newGameMaxPlayers, setNewGameMaxPlayers] = useState(5)
  const [availableMaps, setAvailableMaps] = useState<MapSummary[]>(DEFAULT_MAP_CATALOG)
  const [newGameMapId, setNewGameMapId] = useState('base')
  const [customMapDefinition, setCustomMapDefinition] = useState<CustomMapDefinition>(() => createEmptyCustomMapDefinition())
  const [randomizeTurnOrder, setRandomizeTurnOrder] = useState(true)
  const [setupMode, setSetupMode] = useState<'snellman' | 'auction' | 'fast_auction'>('snellman')
  const [turnTimerEnabled, setTurnTimerEnabled] = useState(false)
  const [turnTimerMinutes, setTurnTimerMinutes] = useState(25)
  const [turnTimerIncrementSeconds, setTurnTimerIncrementSeconds] = useState(0)
  const [enableFanFactions, setEnableFanFactions] = useState(false)
  const [enableFireIceFactions, setEnableFireIceFactions] = useState(false)
  const [fireIceScoring, setFireIceScoring] = useState<'off' | 'on' | 'random'>('off')
  const [lobbyError, setLobbyError] = useState<string | null>(null)

  const trimmedPlayerName = playerName.trim()
  const activePlayerName = trimmedPlayerName || storedLocalPlayerId?.trim() || ''
  const joinedGame = useMemo(
    () => games.find((game) => activePlayerName !== '' && game.players.includes(activePlayerName) && !game.started) ?? null,
    [activePlayerName, games],
  )
  const joinedGameId = joinedGame?.id ?? null
  const openGames = useMemo(() => games.filter((game) => !game.started), [games])
  const startedGames = useMemo(() => games.filter((game) => !!game.started), [games])

  useEffect(() => {
    if (gameState?.id && activePlayerName !== '' && gameState.players[activePlayerName] && gameState.started) {
      void navigate(`/game/${gameState.id}`)
    }
  }, [activePlayerName, gameState, navigate])

  useEffect(() => {
    if (lastMessage === null) return

    if (lastMessage && typeof lastMessage === 'object' && 'type' in lastMessage) {
      const msg = lastMessage as LobbyMessage
      if (msg.type === 'lobby_state') {
        setGames(Array.isArray(msg.payload) ? msg.payload as GameInfo[] : [])
        setLobbyError(null)
      } else if (msg.type === 'available_maps') {
        setAvailableMaps(Array.isArray(msg.payload) ? msg.payload as MapSummary[] : DEFAULT_MAP_CATALOG)
      } else if (msg.type === 'error') {
        setLobbyError(formatLobbyError((msg.payload ?? '') as LobbyErrorPayload))
      } else if (msg.type === 'game_left') {
        setLobbyError(null)
      }
    }
  }, [lastMessage])

  useEffect(() => {
    if (isConnected) {
      sendMessage({ type: 'list_games' })
    }
  }, [isConnected, sendMessage])

  const getStatusColorClass = (): string => {
    switch (connectionStatus) {
      case 'connected':
        return 'lobby-status-dot-connected'
      case 'connecting':
        return 'lobby-status-dot-connecting'
      case 'error':
        return 'lobby-status-dot-error'
      default:
        return 'lobby-status-dot-disconnected'
    }
  }

  const handleCreateGame = (overrides?: { maxPlayers?: number }): void => {
    if (!trimmedPlayerName || !newGameName.trim() || joinedGameId) return
    useGameStore.getState().setLocalPlayerId(trimmedPlayerName)
    setLobbyError(null)
    sendMessage({
      type: 'create_game',
      payload: {
        name: newGameName.trim(),
        maxPlayers: overrides?.maxPlayers ?? newGameMaxPlayers,
        creator: trimmedPlayerName,
        mapId: newGameMapId,
        enableFanFactions,
        enableFireIceFactions,
        fireIceScoring,
        customMap: newGameMapId === 'custom' ? customMapDefinition : undefined,
      },
    })
    setNewGameName('')
  }

  const handleJoinGame = (id: string): void => {
    if (!trimmedPlayerName || joinedGameId) return
    useGameStore.getState().setLocalPlayerId(trimmedPlayerName)
    setLobbyError(null)
    sendMessage({ type: 'join_game', payload: { id, name: trimmedPlayerName } })
  }

  const handleLeaveGame = (id: string): void => {
    if (!trimmedPlayerName) return
    setLobbyError(null)
    sendMessage({ type: 'leave_game', payload: { id, name: trimmedPlayerName } })
  }

  const handleSpectateGame = (id: string): void => {
    void navigate(`/game/${id}`)
  }

  return (
    <div className="lobby-page" data-testid="lobby-screen">
      <div className="lobby-shell">
        <div className="lobby-header">
          <p className="lobby-kicker">TM Lobby</p>
          <h1 className="lobby-title">Terra Mystica Online</h1>
          <div className="lobby-status">
            <span className={`lobby-status-dot ${getStatusColorClass()}`}></span>
            <span className="lobby-status-label">{connectionStatus}</span>
          </div>
        </div>

        <div className="lobby-panel">
          <div className="lobby-section">
            <label className="lobby-label" htmlFor="lobby-player-name">Player</label>
            <input
              id="lobby-player-name"
              type="text"
              data-testid="lobby-player-name"
              value={playerName}
              onChange={(e) => { setPlayerName(e.target.value) }}
              className="lobby-input"
              placeholder="Name"
            />
          </div>

          {lobbyError && (
            <div className="lobby-alert" role="alert">
              {lobbyError}
            </div>
          )}

          {joinedGame && (
            <div className="lobby-banner">
              <span className="lobby-banner-title">Current game</span>
              <span>Seated in <strong>{joinedGame.name}</strong>.</span>
              <span>Leave to join or create another game.</span>
            </div>
          )}

          <div className="lobby-section lobby-section-split">
            <div className="lobby-section-heading">
              <div>
                <h2>Create Game</h2>
              </div>
            </div>

            <div className="lobby-create-grid">
              <input
                type="text"
                data-testid="lobby-game-name"
                value={newGameName}
                onChange={(e) => { setNewGameName(e.target.value) }}
                className="lobby-input"
                placeholder="Game Name"
                disabled={!isConnected || joinedGameId !== null}
              />
              <select
                data-testid="lobby-map-id"
                value={newGameMapId}
                onChange={(e) => { setNewGameMapId(e.target.value) }}
                className="lobby-select"
                disabled={!isConnected || joinedGameId !== null}
              >
                {availableMaps.map((map) => (
                  <option key={map.id} value={map.id}>{map.name}</option>
                ))}
              </select>
              <select
                data-testid="lobby-max-players"
                value={newGameMaxPlayers}
                onChange={(e) => { setNewGameMaxPlayers(Number(e.target.value)) }}
                className="lobby-select"
                disabled={!isConnected || joinedGameId !== null}
              >
                <option value={1}>1 player</option>
                <option value={2}>2 players</option>
                <option value={3}>3 players</option>
                <option value={4}>4 players</option>
                <option value={5}>5 players</option>
              </select>
              <button
                data-testid="lobby-create-game"
                onClick={() => { handleCreateGame() }}
                disabled={!isConnected || !trimmedPlayerName || !newGameName.trim() || joinedGameId !== null}
                className="lobby-button lobby-button-primary"
              >
                Create
              </button>
              <button
                data-testid="lobby-refresh-games-top"
                onClick={() => { sendMessage({ type: 'list_games' }) }}
                disabled={!isConnected}
                className="lobby-button lobby-button-secondary"
              >
                Refresh
              </button>
            </div>

            {newGameMapId === 'custom' && (
              <CustomMapEditor
                value={customMapDefinition}
                onChange={setCustomMapDefinition}
                onCreateGame={() => { handleCreateGame() }}
                createGameDisabled={!isConnected || !trimmedPlayerName || !newGameName.trim() || joinedGameId !== null}
                disabled={!isConnected || joinedGameId !== null}
              />
            )}

            <div className="lobby-option-list">
              <label className="lobby-checkbox-row">
                <input
                  type="checkbox"
                  data-testid="lobby-randomize-turn-order"
                  checked={randomizeTurnOrder}
                  onChange={(e) => { setRandomizeTurnOrder(e.target.checked) }}
                />
                <span>Random turn order</span>
              </label>

              <label className="lobby-field-stack">
                <span className="lobby-label">Setup mode</span>
                <select
                  data-testid="lobby-setup-mode"
                  value={setupMode}
                  onChange={(e) => { setSetupMode(e.target.value as 'snellman' | 'auction' | 'fast_auction') }}
                  className="lobby-select"
                  disabled={!isConnected}
                >
                  <option value="snellman">Snellman</option>
                  <option value="auction">Auction</option>
                  <option value="fast_auction">Fast auction</option>
                </select>
              </label>

              <label className="lobby-checkbox-row">
                <input
                  type="checkbox"
                  data-testid="lobby-enable-fire-ice-factions"
                  checked={enableFireIceFactions}
                  onChange={(e) => { setEnableFireIceFactions(e.target.checked) }}
                  disabled={!isConnected || joinedGameId !== null}
                />
                <span>F&amp;I factions</span>
              </label>

              <label className="lobby-checkbox-row">
                <input
                  type="checkbox"
                  data-testid="lobby-enable-fan-factions"
                  checked={enableFanFactions}
                  onChange={(e) => { setEnableFanFactions(e.target.checked) }}
                  disabled={!isConnected || joinedGameId !== null}
                />
                <span>Fan factions</span>
              </label>

              <label className="lobby-field-stack">
                <span className="lobby-label">F&amp;I final scoring</span>
                <select
                  data-testid="lobby-fire-ice-scoring"
                  value={fireIceScoring}
                  onChange={(e) => { setFireIceScoring(e.target.value as 'off' | 'on' | 'random') }}
                  className="lobby-select"
                  disabled={!isConnected || joinedGameId !== null}
                >
                  <option value="off">Off</option>
                  <option value="on">On</option>
                  <option value="random">Random</option>
                </select>
              </label>

              <div className="lobby-timer-box">
                <label className="lobby-checkbox-row">
                  <input
                    type="checkbox"
                    data-testid="lobby-turn-timer-enabled"
                    checked={turnTimerEnabled}
                    onChange={(e) => { setTurnTimerEnabled(e.target.checked) }}
                  />
                  <span>Enable turn timer</span>
                </label>
                {turnTimerEnabled && (
                  <div className="lobby-create-grid">
                    <label className="lobby-field-stack">
                      <span className="lobby-label">Minutes</span>
                      <input
                        type="number"
                        data-testid="lobby-turn-timer-minutes"
                        min={1}
                        step={1}
                        value={turnTimerMinutes}
                        onChange={(e) => {
                          const value = Number(e.target.value)
                          setTurnTimerMinutes(Number.isFinite(value) ? value : 25)
                        }}
                        className="lobby-input"
                      />
                    </label>
                    <label className="lobby-field-stack">
                      <span className="lobby-label">Increment (sec)</span>
                      <input
                        type="number"
                        data-testid="lobby-turn-timer-increment"
                        min={0}
                        step={1}
                        value={turnTimerIncrementSeconds}
                        onChange={(e) => {
                          const value = Number(e.target.value)
                          setTurnTimerIncrementSeconds(Number.isFinite(value) ? value : 0)
                        }}
                        className="lobby-input"
                      />
                    </label>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="lobby-section lobby-section-split">
            <div className="lobby-section-heading">
              <div>
                <h2>Open Games</h2>
              </div>
              <button
                data-testid="lobby-refresh-games-list"
                onClick={() => { sendMessage({ type: 'list_games' }) }}
                disabled={!isConnected}
                className="lobby-button lobby-button-secondary"
              >
                Refresh
              </button>
            </div>

            {openGames.length === 0 ? (
              <p className="lobby-empty">No open games.</p>
            ) : (
              <div className="lobby-games">
                {openGames.map((g) => {
                  const isFull = g.players.length >= g.maxPlayers
                  const isJoined = trimmedPlayerName !== '' && g.players.includes(trimmedPlayerName)
                  const isHost = trimmedPlayerName !== '' && g.host === trimmedPlayerName
                  const joinBlockedByOtherSeat = joinedGameId !== null && joinedGameId !== g.id
                  const displayMapName =
                    g.customMap?.name?.trim()
                    || availableMaps.find((map) => map.id === g.mapId)?.name
                    || (g.mapId === 'custom' ? 'Custom' : g.mapId)
                  return (
                    <div key={g.id} className="lobby-game-card">
                      <div className="lobby-game-meta">
                        <div className="lobby-game-title-row">
                          <div className="lobby-game-title">{g.name}</div>
                          <div className="lobby-tag-row">
                            <span className="lobby-tag">{g.id}</span>
                            <span className="lobby-tag lobby-tag-muted">
                              Map: {displayMapName}
                            </span>
                            <span className="lobby-tag lobby-tag-muted">
                              F&amp;I factions: {g.enableFireIceFactions ? 'On' : 'Off'}
                            </span>
                            <span className="lobby-tag lobby-tag-muted">
                              Fan factions: {g.enableFanFactions ? 'On' : 'Off'}
                            </span>
                            <span className="lobby-tag lobby-tag-muted">
                              F&amp;I scoring: {g.fireIceScoring === 'random' ? 'Random' : g.fireIceScoring === 'on' ? 'On' : 'Off'}
                            </span>
                            {g.host && <span className="lobby-tag lobby-tag-muted">Host: {g.host}</span>}
                          </div>
                        </div>
                        <div className="lobby-player-line">
                          <span>{String(g.players.length)}/{String(g.maxPlayers)} players</span>
                          <span>{g.players.join(', ') || 'No players yet'}</span>
                        </div>
                        {g.mapId === 'custom' && g.customMap && (
                          <div className="lobby-map-preview">
                            <HexGridCanvas
                              testId={`lobby-custom-map-preview-${g.id}`}
                              hexes={buildCustomMapHexes(g.customMap)}
                              showCoords={false}
                              disableHover
                            />
                          </div>
                        )}
                      </div>

                      <div className="lobby-game-actions">
                        {isJoined ? (
                          <button
                            data-testid={`lobby-leave-${g.id}`}
                            onClick={() => { handleLeaveGame(g.id) }}
                            disabled={!isConnected}
                            className="lobby-button lobby-button-danger"
                          >
                            Leave
                          </button>
                        ) : (
                          <button
                            data-testid={`lobby-join-${g.id}`}
                            onClick={() => { handleJoinGame(g.id) }}
                            disabled={!isConnected || !trimmedPlayerName || isFull || joinBlockedByOtherSeat}
                            className="lobby-button lobby-button-accent"
                          >
                            {joinBlockedByOtherSeat ? 'Leave current game' : 'Join'}
                          </button>
                        )}

                        <button
                          data-testid={`lobby-start-${g.id}`}
                          onClick={() => {
                            sendMessage({
                              type: 'start_game',
                              payload: {
                                gameID: g.id,
                                randomizeTurnOrder,
                                setupMode,
                                turnTimerEnabled,
                                turnTimerSeconds: Math.max(1, Math.trunc(turnTimerMinutes * 60)),
                                turnTimerIncrementSeconds: Math.max(0, Math.trunc(turnTimerIncrementSeconds)),
                              },
                            })
                          }}
                          disabled={!isConnected || !isFull || !isHost}
                          className="lobby-button lobby-button-success"
                        >
                          {isFull ? (isHost ? 'Start' : 'Host starts') : `Waiting ${String(g.players.length)}/${String(g.maxPlayers)}`}
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          <div className="lobby-section lobby-section-split">
            <div className="lobby-section-heading">
              <div>
                <h2>Started Games</h2>
              </div>
            </div>

            {startedGames.length === 0 ? (
              <p className="lobby-empty">No started games.</p>
            ) : (
              <div className="lobby-games">
                {startedGames.map((g) => {
                  const isSeated = activePlayerName !== '' && g.players.includes(activePlayerName)
                  const displayMapName =
                    g.customMap?.name?.trim()
                    || availableMaps.find((map) => map.id === g.mapId)?.name
                    || (g.mapId === 'custom' ? 'Custom' : g.mapId)

                  return (
                    <div key={g.id} className="lobby-game-card">
                      <div className="lobby-game-meta">
                        <div className="lobby-game-title-row">
                          <div className="lobby-game-title">{g.name}</div>
                          <div className="lobby-tag-row">
                            <span className="lobby-tag">{g.id}</span>
                            <span className="lobby-tag lobby-tag-muted">Map: {displayMapName}</span>
                            <span className="lobby-tag lobby-tag-muted">
                              F&amp;I factions: {g.enableFireIceFactions ? 'On' : 'Off'}
                            </span>
                            <span className="lobby-tag lobby-tag-muted">
                              Fan factions: {g.enableFanFactions ? 'On' : 'Off'}
                            </span>
                            <span className="lobby-tag lobby-tag-muted">
                              F&amp;I scoring: {g.fireIceScoring === 'random' ? 'Random' : g.fireIceScoring === 'on' ? 'On' : 'Off'}
                            </span>
                            {g.host && <span className="lobby-tag lobby-tag-muted">Host: {g.host}</span>}
                          </div>
                        </div>
                        <div className="lobby-player-line">
                          <span>{String(g.players.length)}/{String(g.maxPlayers)} players</span>
                          <span>{g.players.join(', ') || 'No players listed'}</span>
                        </div>
                        {g.mapId === 'custom' && g.customMap && (
                          <div className="lobby-map-preview">
                            <HexGridCanvas
                              testId={`lobby-custom-map-preview-started-${g.id}`}
                              hexes={buildCustomMapHexes(g.customMap)}
                              showCoords={false}
                              disableHover
                            />
                          </div>
                        )}
                      </div>

                      <div className="lobby-game-actions">
                        <button
                          data-testid={`lobby-spectate-${g.id}`}
                          onClick={() => { handleSpectateGame(g.id) }}
                          className="lobby-button lobby-button-secondary"
                        >
                          {isSeated ? 'Open' : 'Spectate'}
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
