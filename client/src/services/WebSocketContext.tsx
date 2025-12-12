import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react'
import { useGameStore } from '../stores/gameStore'

interface WebSocketMessage {
  type: string
  payload?: unknown
}

interface WebSocketContextType {
  isConnected: boolean
  sendMessage: (message: unknown) => void
  lastMessage: unknown
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error'
}

const WebSocketContext = createContext<WebSocketContextType | null>(null)

// eslint-disable-next-line react-refresh/only-export-components
export const useWebSocket = (): WebSocketContextType => {
  const context = useContext(WebSocketContext)
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider')
  }
  return context
}

interface WebSocketProviderProps {
  children: React.ReactNode
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<unknown>(null)
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | undefined>(undefined)

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setConnectionStatus('connecting')
    const ws = new WebSocket('ws://localhost:8080/ws')

    ws.onopen = () => {
      // console.log('WebSocket connected')
      setIsConnected(true)
      setConnectionStatus('connected')
    }

    ws.onmessage = (event) => {
      try {
        const payload = typeof event.data === 'string' ? event.data : ''
        const data = JSON.parse(payload) as WebSocketMessage

        // Handle game_state_update messages
        if (data.type === 'game_state_update' && data.payload) {
          // console.log('WebSocketContext: Received game_state_update:', data.payload)
          useGameStore.getState().setGameState(data.payload as import('../types/game.types').GameState)
          // console.log('WebSocketContext: Game state updated in store')
        }

        setLastMessage(data)
      } catch {
        // If not JSON, treat as plain text
        setLastMessage(event.data)
      }
    }

    ws.onerror = (_error) => {
      // console.error('WebSocket error:', error)
      setConnectionStatus('error')
    }

    ws.onclose = () => {
      // console.log('WebSocket disconnected')
      setIsConnected(false)
      setConnectionStatus('disconnected')

      // Attempt to reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('Attempting to reconnect...')
        connect()
      }, 3000)
    }

    wsRef.current = ws
  }, [])

  const sendMessage = useCallback((message: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const payload = typeof message === 'string' ? message : JSON.stringify(message)
      wsRef.current.send(payload)
    } else {
      console.warn('WebSocket is not connected')
    }
  }, [])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  return (
    <WebSocketContext.Provider value={{ isConnected, sendMessage, lastMessage, connectionStatus }}>
      {children}
    </WebSocketContext.Provider>
  )
}
