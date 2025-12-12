import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useWebSocket } from '../services/WebSocketContext'
import { useGameStore } from '../stores/gameStore'

export function Lobby() {
  const { isConnected, sendMessage, lastMessage, connectionStatus } = useWebSocket()
  const navigate = useNavigate()
  const [playerName, setPlayerName] = useState('')
  const [testMessage, setTestMessage] = useState('')
  const [messages, setMessages] = useState<string[]>([])
  const [games, setGames] = useState<{ id: string; name: string; players: string[]; maxPlayers: number }[]>([])
  const [newGameName, setNewGameName] = useState('')
  const [newGameMaxPlayers, setNewGameMaxPlayers] = useState(5)

  useEffect(() => {
    if (lastMessage == null) return

    // Collect raw messages for debugging
    setMessages(prev => [
      ...prev,
      typeof lastMessage === 'string' ? lastMessage : JSON.stringify(lastMessage),
    ])

    // Handle lobby messages
    if (typeof lastMessage === 'object' && lastMessage !== null && 'type' in lastMessage) {
      const msg = lastMessage as { type: string; payload?: any }
      if (msg.type === 'lobby_state') {
        setGames(Array.isArray(msg.payload) ? msg.payload : [])
      } else if (msg.type === 'game_joined') {
        // Don't navigate immediately, wait for start
        console.log('Joined game:', msg.payload.gameId)
      } else if (msg.type === 'game_created') {
        // Don't navigate immediately, wait for start
        console.log('Created game:', msg.payload.gameId)
      } else if (msg.type === 'game_state_update') {
        const gameState = msg.payload
        // If we are in this game AND it has started, navigate to it
        if (gameState?.id && gameState.players?.[playerName] && gameState.started) {
          navigate(`/game/${gameState.id}`)
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

  const handleSendTest = () => {
    if (testMessage.trim()) {
      sendMessage(testMessage)
      setTestMessage('')
    }
  }

  const getStatusColor = () => {
    switch (connectionStatus) {
      case 'connected': return 'bg-green-500'
      case 'connecting': return 'bg-yellow-500'
      case 'error': return 'bg-red-500'
      default: return 'bg-gray-500'
    }
  }

  const handleCreateGame = () => {
    if (!playerName.trim() || !newGameName.trim()) return
    useGameStore.getState().setLocalPlayerId(playerName.trim())
    sendMessage({
      type: 'create_game',
      payload: { name: newGameName.trim(), maxPlayers: newGameMaxPlayers, creator: playerName.trim() },
    })
    setNewGameName('')
  }

  const handleJoinGame = (id: string) => {
    if (!playerName.trim()) return
    useGameStore.getState().setLocalPlayerId(playerName.trim())
    sendMessage({ type: 'join_game', payload: { id, name: playerName.trim() } })
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
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
                  value={newGameName}
                  onChange={(e) => { setNewGameName(e.target.value); }}
                  className="px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  placeholder="Game Name"
                  disabled={!isConnected}
                />
                <select
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
                  onClick={handleCreateGame}
                  disabled={!isConnected || !playerName.trim() || !newGameName.trim()}
                  className="px-6 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-colors"
                >
                  Create
                </button>
                <button
                  onClick={() => { sendMessage({ type: 'list_games' }); }}
                  disabled={!isConnected}
                  className="px-6 py-2 bg-white/10 hover:bg-white/20 border border-white/20 text-white rounded-lg font-medium transition-colors"
                >
                  Refresh
                </button>
              </div>
            </div>

            {/* Test WebSocket Section */}
            <div className="border-t border-white/20 pt-6">
              <h2 className="text-xl font-semibold text-white mb-4">Test WebSocket Connection</h2>

              <div className="flex gap-2 mb-4">
                <input
                  type="text"
                  value={testMessage}
                  onChange={(e) => { setTestMessage(e.target.value); }}
                  onKeyPress={(e) => e.key === 'Enter' && handleSendTest()}
                  className="flex-1 px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  placeholder="Type a message..."
                  disabled={!isConnected}
                />
                <button
                  onClick={handleSendTest}
                  disabled={!isConnected || !testMessage.trim()}
                  className="px-6 py-2 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-colors"
                >
                  Send
                </button>
              </div>

              {/* Messages Display */}
              <div className="bg-black/30 rounded-lg p-4 h-48 overflow-y-auto">
                <div className="space-y-2">
                  {messages.length === 0 ? (
                    <p className="text-gray-400 text-sm">No messages yet. Send a test message!</p>
                  ) : (
                    messages.map((msg, idx) => (
                      <div key={idx} className="text-sm text-gray-200 font-mono">
                        {msg}
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            {/* Games List */}
            <div className="border-t border-white/20 pt-6">
              <div className="flex items-center justify-between mb-3">
                <h2 className="text-xl font-semibold text-white">Open Games</h2>
                <button
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
                    const isFull = (g.players?.length ?? 0) >= g.maxPlayers
                    return (
                      <div key={g.id} className="flex items-center justify-between bg-white/10 border border-white/20 rounded-md p-3">
                        <div>
                          <div className="text-white font-medium">{g.name} <span className="text-xs text-gray-300">({g.id})</span></div>
                          <div className="text-xs text-gray-300">Players: {g.players?.length ?? 0}/{g.maxPlayers}</div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => { handleJoinGame(g.id); }}
                            disabled={!isConnected || !playerName.trim() || isFull}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium"
                          >Join</button>
                          <button
                            onClick={() => { sendMessage({ type: 'start_game', payload: { gameID: g.id } }); }}
                            disabled={!isConnected || !isFull}
                            className={`px-4 py-2 ${isFull
                              ? 'bg-green-600 hover:bg-green-700'
                              : 'bg-gray-600 cursor-not-allowed'
                              } disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-medium`}
                          >
                            {isFull ? 'Start' : `Waiting (${g.players?.length ?? 0}/${g.maxPlayers})`}
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
          <p className="mt-1">Server running on ws://localhost:8080/ws</p>
        </div>
      </div>
    </div>
  )
}
