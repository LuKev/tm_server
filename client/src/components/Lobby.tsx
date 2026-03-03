import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useWebSocket } from '../services/WebSocketContext'
import { useGameStore } from '../stores/gameStore'
import type { GameState } from '../types/game.types'

interface GameInfo {
  id: string
  name: string
  players: string[]
  maxPlayers: number
}

interface LobbyMessage {
  type: string
  payload?: unknown
}

export function Lobby(): React.ReactElement {
  const { isConnected, sendMessage, lastMessage, connectionStatus } = useWebSocket()
  const navigate = useNavigate()
  const [playerName, setPlayerName] = useState('')
  const [games, setGames] = useState<GameInfo[]>([])
  const [newGameName, setNewGameName] = useState('')
  const [newGameMaxPlayers, setNewGameMaxPlayers] = useState(5)
  const [randomizeTurnOrder, setRandomizeTurnOrder] = useState(true)
  const [setupMode, setSetupMode] = useState<'snellman' | 'auction' | 'fast_auction'>('snellman')

  useEffect(() => {
    if (lastMessage === null) return

    // Handle lobby messages
    if (lastMessage && typeof lastMessage === 'object' && 'type' in lastMessage) {
      const msg = lastMessage as LobbyMessage
      if (msg.type === 'lobby_state') {
        setGames(Array.isArray(msg.payload) ? msg.payload as GameInfo[] : [])
      } else if (msg.type === 'game_state_update') {
        const gameState = msg.payload as GameState | undefined
        // If we are in this game AND it has started, navigate to it
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (gameState?.id && gameState.players[playerName] && gameState.started) {
          void navigate(`/game/${gameState.id}`)
        }
      }
    }
  }, [lastMessage, navigate, playerName])

  // Request games list on connect
  useEffect(() => {
    if (isConnected) {
      sendMessage({ type: 'list_games' })
    }
  }, [isConnected, sendMessage])

  const getStatusColor = (): string => {
    switch (connectionStatus) {
      case 'connected': return 'bg-green-500'
      case 'connecting': return 'bg-yellow-500'
      case 'error': return 'bg-red-500'
      default: return 'bg-gray-500'
    }
  }

  const handleCreateGame = (): void => {
    if (!playerName.trim() || !newGameName.trim()) return
    useGameStore.getState().setLocalPlayerId(playerName.trim())
    sendMessage({
      type: 'create_game',
      payload: { name: newGameName.trim(), maxPlayers: newGameMaxPlayers, creator: playerName.trim() },
    })
    setNewGameName('')
  }

  const handleJoinGame = (id: string): void => {
    if (!playerName.trim()) return
    useGameStore.getState().setLocalPlayerId(playerName.trim())
    sendMessage({ type: 'join_game', payload: { id, name: playerName.trim() } })
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4" data-testid="lobby-screen">
      <div className="max-w-4xl w-full">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-5xl font-bold text-white mb-2">Terra Mystica Online</h1>
          <p className="text-gray-300">Multiplayer Strategy Board Game</p>

          {/* Connection Status */}
          <div className="mt-4 flex items-center justify-center gap-2">
            <div className={`w-3 h-3 rounded-full ${getStatusColor()} animate-pulse`}></div>
            <span className="text-sm text-gray-300 capitalize">{connectionStatus}</span>
          </div>
        </div>

        {/* Main Content */}
        <div className="bg-white/10 backdrop-blur-lg rounded-lg shadow-2xl p-8 border border-white/20">
          <div className="space-y-6">
            {/* Player Name */}
            <div>
              <label className="block text-sm font-medium text-gray-200 mb-2">Player Name</label>
              <input
                type="text"
                data-testid="lobby-player-name"
                value={playerName}
                onChange={(e) => { setPlayerName(e.target.value); }}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                placeholder="Enter your name"
              />
            </div>

            {/* Create Game */}
            <div className="border-t border-white/20 pt-6">
              <div className="flex items-center justify-between mb-2">
                <h2 className="text-xl font-semibold text-white">Create Game</h2>
                <span className="text-xs text-gray-300">IDs are generated automatically</span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                <input
                  type="text"
                  data-testid="lobby-game-name"
                  value={newGameName}
                  onChange={(e) => { setNewGameName(e.target.value); }}
                  className="px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  placeholder="Game Name"
                  disabled={!isConnected}
                />
                <select
                  data-testid="lobby-max-players"
                  value={newGameMaxPlayers}
                  onChange={(e) => { setNewGameMaxPlayers(Number(e.target.value)); }}
                  className="px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  disabled={!isConnected}
                >
                  <option value={1}>1 player</option>
                  <option value={2}>2 players</option>
                  <option value={3}>3 players</option>
                  <option value={4}>4 players</option>
                  <option value={5}>5 players</option>
                </select>
                <button
                  data-testid="lobby-create-game"
                  onClick={handleCreateGame}
                  disabled={!isConnected || !playerName.trim() || !newGameName.trim()}
                  className="px-6 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-colors"
                >
                  Create
                </button>
                <button
                  data-testid="lobby-refresh-games-top"
                  onClick={() => { sendMessage({ type: 'list_games' }); }}
                  disabled={!isConnected}
                  className="px-6 py-2 bg-white/10 hover:bg-white/20 border border-white/20 text-white rounded-lg font-medium transition-colors"
                >
                  Refresh
                </button>
              </div>
              <label className="mt-3 inline-flex items-center gap-2 text-sm text-gray-200">
                <input
                  type="checkbox"
                  data-testid="lobby-randomize-turn-order"
                  checked={randomizeTurnOrder}
                  onChange={(e) => { setRandomizeTurnOrder(e.target.checked); }}
                  className="rounded border-white/30 bg-white/10"
                />
                Randomize turn order on start
              </label>
              <div className="mt-3">
                <label className="block text-sm text-gray-200 mb-1">Setup mode</label>
                <select
                  data-testid="lobby-setup-mode"
                  value={setupMode}
                  onChange={(e) => { setSetupMode(e.target.value as 'snellman' | 'auction' | 'fast_auction') }}
                  className="px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  disabled={!isConnected}
                >
                  <option value="snellman">Snellman (Pick Factions)</option>
                  <option value="auction">Auction</option>
                  <option value="fast_auction">Fast Auction</option>
                </select>
              </div>
            </div>

            {/* Games List */}
            <div className="border-t border-white/20 pt-6">
              <div className="flex items-center justify-between mb-3">
                <h2 className="text-xl font-semibold text-white">Open Games</h2>
                <button
                  data-testid="lobby-refresh-games-list"
                  onClick={() => { sendMessage({ type: 'list_games' }); }}
                  disabled={!isConnected}
                  className="px-3 py-1 text-sm bg-white/20 hover:bg-white/30 text-white rounded-md"
                >Refresh</button>
              </div>
              {games.length === 0 ? (
                <p className="text-gray-400 text-sm">No open games. Create one above.</p>
              ) : (
                <div className="space-y-2">
                  {games.map((g) => {
                    const isFull = g.players.length >= g.maxPlayers
                    return (
                      <div key={g.id} className="flex items-center justify-between bg-white/10 border border-white/20 rounded-md p-3">
                        <div>
                          <div className="text-white font-medium">{g.name} <span className="text-xs text-gray-300">({g.id})</span></div>
                          <span className="text-sm text-gray-500">Players: {String(g.players.length)}/{String(g.maxPlayers)}</span>
                        </div>
                        <div className="flex gap-2">
                          <button
                            data-testid={`lobby-join-${g.id}`}
                            onClick={() => { handleJoinGame(g.id); }}
                            disabled={!isConnected || !playerName.trim() || isFull}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium"
                          >Join</button>
                          <button
                            data-testid={`lobby-start-${g.id}`}
                            onClick={() => {
                              sendMessage({
                                type: 'start_game',
                                payload: {
                                  gameID: g.id,
                                  randomizeTurnOrder,
                                  setupMode,
                                },
                              })
                            }}
                            disabled={!isConnected || !isFull}
                            className={`px-4 py-2 ${isFull
                              ? 'bg-green-600 hover:bg-green-700'
                              : 'bg-gray-600 cursor-not-allowed'
                              } disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium`}
                          >
                            {isFull ? 'Start' : `Waiting (${String(g.players.length)}/${String(g.maxPlayers)})`}
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

        {/* Info Footer */}
        <div className="mt-6 text-center text-sm text-gray-400">
          <p>Phase 2: Lobby wired to server (list/create/join)</p>
          <p className="mt-1">Server connected</p>
        </div>
      </div>
    </div>
  )
}
