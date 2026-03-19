import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react'
import { useGameStore } from '../stores/gameStore'
import type { GameState } from '../types/game.types'

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

type TestWebSocketWindow = Window & {
  __TM_TEST_IS_CONNECTED__?: () => boolean
  __TM_TEST_SEND_MESSAGE__?: (message: unknown) => void
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<unknown>(null)
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | undefined>(undefined)

  const installTestHooks = useCallback((ws: WebSocket | null) => {
    if (!import.meta.env.DEV || typeof window === 'undefined') return
    const testWindow = window as TestWebSocketWindow
    if (!ws) {
      testWindow.__TM_TEST_IS_CONNECTED__ = () => false
      testWindow.__TM_TEST_SEND_MESSAGE__ = undefined
      return
    }
    testWindow.__TM_TEST_IS_CONNECTED__ = () => ws.readyState === WebSocket.OPEN
    testWindow.__TM_TEST_SEND_MESSAGE__ = (message: unknown) => {
      if (ws.readyState !== WebSocket.OPEN) {
        throw new Error('WebSocket is not connected')
      }
      const payload = typeof message === 'string' ? message : JSON.stringify(message)
      ws.send(payload)
    }
  }, [])

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN || wsRef.current?.readyState === WebSocket.CONNECTING) {
      return
    }

    setConnectionStatus('connecting')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host; // This will be kezilu.com in production
    // Connect to /api/ws which Cloudflare routes to the backend
    const wsUrl = `${protocol}//${host}/api/ws`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws
    installTestHooks(ws)

    ws.onopen = () => {
      if (wsRef.current !== ws) return
      setIsConnected(true)
      setConnectionStatus('connected')
    }

    ws.onmessage = (event) => {
      try {
        const payload = typeof event.data === 'string' ? event.data : ''
        const data = JSON.parse(payload) as WebSocketMessage

        // Handle game_state_update messages
        if (data.type === 'game_state_update' && data.payload) {
          useGameStore.getState().setGameState(data.payload as GameState)
        }

        setLastMessage(data)
      } catch {
        // If not JSON, treat as plain text
        setLastMessage(event.data)
      }
    }

    ws.onerror = (_error) => {
      if (wsRef.current !== ws) return
      setConnectionStatus('error')
    }

    ws.onclose = () => {
      if (wsRef.current !== ws) return
      wsRef.current = null
      setIsConnected(false)
      setConnectionStatus('disconnected')
      installTestHooks(null)

      // Attempt to reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }

  }, [installTestHooks])

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
      const ws = wsRef.current
      wsRef.current = null
      installTestHooks(null)
      if (ws) {
        ws.close()
      }
    }
  }, [connect, installTestHooks])

  return (
    <WebSocketContext.Provider value={{ isConnected, sendMessage, lastMessage, connectionStatus }}>
      {children}
    </WebSocketContext.Provider>
  )
}
