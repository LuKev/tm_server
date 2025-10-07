import { useState, useEffect } from 'react'
import { useWebSocket } from '../services/WebSocketContext'

function Lobby() {
  const { isConnected, sendMessage, lastMessage, connectionStatus } = useWebSocket()
  const [playerName, setPlayerName] = useState('')
  const [testMessage, setTestMessage] = useState('')
  const [messages, setMessages] = useState<string[]>([])

  useEffect(() => {
    if (lastMessage) {
      setMessages(prev => [...prev, lastMessage])
    }
  }, [lastMessage])

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
            {/* Player Name Input */}
            <div>
              <label className="block text-sm font-medium text-gray-200 mb-2">
                Player Name
              </label>
              <input
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                className="w-full px-4 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                placeholder="Enter your name"
              />
            </div>

            {/* Test WebSocket Section */}
            <div className="border-t border-white/20 pt-6">
              <h2 className="text-xl font-semibold text-white mb-4">Test WebSocket Connection</h2>
              
              <div className="flex gap-2 mb-4">
                <input
                  type="text"
                  value={testMessage}
                  onChange={(e) => setTestMessage(e.target.value)}
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
                        {typeof msg === 'string' ? msg : JSON.stringify(msg)}
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex gap-4 pt-4">
              <button
                disabled={!isConnected || !playerName.trim()}
                className="flex-1 px-6 py-3 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-semibold transition-colors"
              >
                Create Game
              </button>
              <button
                disabled={!isConnected || !playerName.trim()}
                className="flex-1 px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg font-semibold transition-colors"
              >
                Join Game
              </button>
            </div>
          </div>
        </div>

        {/* Info Footer */}
        <div className="mt-6 text-center text-sm text-gray-400">
          <p>Phase 1: Basic WebSocket connection established</p>
          <p className="mt-1">Server running on ws://localhost:8080/ws</p>
        </div>
      </div>
    </div>
  )
}

export default Lobby
