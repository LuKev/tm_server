// Action submission service
import { useWebSocket } from './WebSocketContext'

export interface SelectFactionActionPayload {
    type: 'select_faction'
    playerID: string
    faction: string
    gameID: string
}

export interface SetupDwellingActionPayload {
    type: 'setup_dwelling'
    playerID: string
    hex: {
        q: number
        r: number
    }
    gameID: string
}

export type ActionPayload = SelectFactionActionPayload | SetupDwellingActionPayload

export interface ActionMessage {
    type: 'perform_action'
    payload: ActionPayload
}

export function useActionService(): { submitSetupDwelling: (playerID: string, q: number, r: number, gameID?: string) => void; submitSelectFaction: (playerID: string, faction: string, gameID: string) => void } {
    const { sendMessage } = useWebSocket()

    const submitSetupDwelling = (playerID: string, q: number, r: number, gameID = "2"): void => {
        const action: ActionMessage = {
            type: 'perform_action',
            payload: {
                type: 'setup_dwelling',
                playerID,
                hex: { q, r },
                gameID
            }
        }

        console.log('Submitting setup dwelling action:', action)
        sendMessage(action)
    }

    const submitSelectFaction = (playerID: string, faction: string, gameID: string): void => {
        const action: ActionMessage = {
            type: 'perform_action',
            payload: {
                type: 'select_faction',
                playerID,
                faction,
                gameID
            }
        }

        console.log('Submitting select faction action:', action)
        sendMessage(action)
    }

    return {
        submitSetupDwelling,
        submitSelectFaction
    }
}
