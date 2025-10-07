import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react'

interface WebSocketContextType {
  isConnected: boolean
  sendMessage: (message: any) => void
  lastMessage: any
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error'
}

const WebSocketContext = createContext<WebSocketContextType | null>(null)

export const useWebSocket = () => {
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
  const [lastMessage, setLastMessage] = useState<any>(null)
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>()

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setConnectionStatus('connecting')
    const ws = new WebSocket('ws://localhost:8080/ws')

    ws.onopen = () => {
      console.log('WebSocket connected')
      setIsConnected(true)
      setConnectionStatus('connected')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        setLastMessage(data)
      } catch (e) {
        // If not JSON, treat as plain text
        setLastMessage(event.data)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      setConnectionStatus('error')
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
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

  const sendMessage = useCallback((message: any) => {
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
