import { useWebSocket } from './WebSocketContext'
import { useGameStore } from '../stores/gameStore'

export interface PerformActionPayload {
  type: string
  gameID: string
  actionId: string
  expectedRevision: number
  params?: Record<string, unknown>
}

export interface ActionMessage {
  type: 'perform_action'
  payload: PerformActionPayload
}

const makeActionID = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `action-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function useActionService(): {
  submitAction: (gameID: string, type: string, params?: Record<string, unknown>) => void
  submitSetupDwelling: (playerID: string, q: number, r: number, gameID?: string) => void
  submitSelectFaction: (playerID: string, faction: string, gameID: string) => void
} {
  const { sendMessage } = useWebSocket()

  const submitAction = (gameID: string, type: string, params: Record<string, unknown> = {}): void => {
    const expectedRevision = useGameStore.getState().gameState?.revision ?? 0

    const message: ActionMessage = {
      type: 'perform_action',
      payload: {
        type,
        gameID,
        actionId: makeActionID(),
        expectedRevision,
        params,
      },
    }

    sendMessage(message)
  }

  const submitSetupDwelling = (_playerID: string, q: number, r: number, gameID = '2'): void => {
    submitAction(gameID, 'setup_dwelling', { hex: { q, r } })
  }

  const submitSelectFaction = (_playerID: string, faction: string, gameID: string): void => {
    submitAction(gameID, 'select_faction', { faction })
  }

  return {
    submitAction,
    submitSetupDwelling,
    submitSelectFaction,
  }
}
