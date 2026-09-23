import { useEffect, useRef, useState } from 'react'
import { PowerActionType, TerrainType } from '../types/game.types'
import { useActionService } from '../services/actionService'
import { useWebSocket } from '../services/WebSocketContext'

type Hex = { q: number; r: number }
export type SpadeTarget = { hex: Hex; terrain: TerrainType }
export type PowerSpadeDraft = {
  actionType: PowerActionType.Spade | PowerActionType.DoubleSpade
  useCoins: boolean
  terrain: TerrainType
  first: SpadeTarget | null
  second: SpadeTarget | null
  choosing: 'first' | 'second'
  buildDwelling: boolean
  useSkip: boolean
  status: 'editing' | 'submitting' | 'accepted'
  error: string | null
}

export function powerSpadeParams(draft: PowerSpadeDraft): Record<string, unknown> {
  return {
    spadeActionVersion: 2,
    actionType: draft.actionType,
    useCoins: draft.useCoins,
    ...(draft.first ? {
      targetHex: draft.first.hex, targetTerrain: draft.first.terrain,
      buildDwelling: draft.buildDwelling,
      ...(draft.useSkip ? { useSkip: true } : {}),
      ...(draft.second ? { secondHex: draft.second.hex, secondTerrain: draft.second.terrain } : {}),
    } : { declineReward: true }),
  }
}

export function usePowerSpadeDraft(gameId: string | undefined) {
  const [draft, setDraft] = useState<PowerSpadeDraft | null>(null)
  const pendingId = useRef<string | null>(null)
  const { submitAction } = useActionService()
  const { lastMessage, isConnected } = useWebSocket()

  useEffect(() => {
    if (!lastMessage || typeof lastMessage !== 'object') return
    const message = lastMessage as { type?: string; payload?: { actionId?: string; message?: string } }
    if (!pendingId.current || message.payload?.actionId !== pendingId.current) return
    if (message.type !== 'action_accepted' && message.type !== 'action_rejected') return
    pendingId.current = null
    setDraft(current => current && ({ ...current,
      status: message.type === 'action_accepted' ? 'accepted' : 'editing',
      error: message.type === 'action_rejected' ? message.payload?.message ?? 'Action rejected. Adjust your choices and try again.' : null,
    }))
  }, [lastMessage])

  useEffect(() => {
    if (isConnected || !pendingId.current) return
    pendingId.current = null
    setDraft(current => current && ({ ...current, status: 'editing', error: 'Connection lost. Check the refreshed board before submitting again.' }))
  }, [isConnected])

  const edit = (patch: Partial<PowerSpadeDraft>): void => {
    if (pendingId.current) return
    setDraft(current => current && ({ ...current, error: null, ...patch }))
  }
  return {
    draft,
    edit,
    begin(actionType: PowerSpadeDraft['actionType'], terrain: TerrainType, useCoins: boolean) {
      if (pendingId.current) return
      setDraft({ actionType, terrain, useCoins, first: null, second: null, choosing: 'first', buildDwelling: true, useSkip: false, status: 'editing', error: null })
    },
    selectHex(hex: Hex) {
      if (pendingId.current) return
      setDraft(current => current && ({ ...current, [current.choosing]: { hex, terrain: current.terrain }, error: null }))
    },
    cancel() { if (!pendingId.current) setDraft(null) },
    submit() {
      if (!gameId || !draft || pendingId.current || draft.status === 'accepted') return
      if (!isConnected) { edit({ error: 'Reconnect before submitting.' }); return }
      pendingId.current = submitAction(gameId, 'power_action_claim', powerSpadeParams(draft))
      setDraft({ ...draft, status: 'submitting', error: null })
    },
  }
}
