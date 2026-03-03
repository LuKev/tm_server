import WebSocket from 'ws'

type WsMessage = Record<string, unknown>

export type JsonObject = Record<string, unknown>

type WsBotOptions = {
  queueLimit?: number
}

export class WsBot {
  private readonly ws: WebSocket
  private readonly queueLimit: number
  private readonly queueByType: Map<string, WsMessage[]> = new Map()
  private readonly statesByGame: Map<string, WsMessage> = new Map()

  private constructor(ws: WebSocket, options: WsBotOptions = {}) {
    this.ws = ws
    this.queueLimit = Math.max(1, Math.trunc(options.queueLimit ?? 2_000))

    this.ws.on('message', (raw) => {
      const payload = typeof raw === 'string' ? raw : raw.toString('utf8')
      let parsed: WsMessage
      try {
        parsed = JSON.parse(payload) as WsMessage
      } catch {
        return
      }

      const msgType = String(parsed.type ?? '')
      if (msgType === 'game_state_update') {
        const state = (parsed.payload ?? {}) as WsMessage
        const gameID = String(state.id ?? '')
        if (gameID !== '') {
          this.statesByGame.set(gameID, state)
        }
      }

      if (msgType !== '') {
        const queue = this.queueByType.get(msgType) ?? []
        queue.push(parsed)
        if (queue.length > this.queueLimit) {
          queue.splice(0, queue.length - this.queueLimit)
        }
        this.queueByType.set(msgType, queue)
      }
    })
  }

  static async connect(url: string, options?: WsBotOptions): Promise<WsBot> {
    const ws = new WebSocket(url)
    await new Promise<void>((resolve, reject) => {
      ws.once('open', () => resolve())
      ws.once('error', (err) => reject(err))
    })
    return new WsBot(ws, options)
  }

  close(): void {
    this.ws.close()
  }

  send(type: string, payload?: WsMessage): void {
    this.ws.send(JSON.stringify({ type, payload }))
  }

  async waitForType(type: string, timeoutMs = 10_000): Promise<WsMessage> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const queue = this.queueByType.get(type)
      if (queue && queue.length > 0) {
        const msg = queue.shift()
        if (msg) {
          return msg
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 25))
    }
    throw new Error(`timeout waiting for websocket message type=${type}`)
  }

  async waitForAnyType(types: string[], timeoutMs = 10_000): Promise<WsMessage> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      for (const type of types) {
        const queue = this.queueByType.get(type)
        if (queue && queue.length > 0) {
          const msg = queue.shift()
          if (msg) {
            return msg
          }
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 25))
    }
    throw new Error(`timeout waiting for websocket message types=${types.join(',')}`)
  }

  async waitForRevision(gameID: string, minRevision: number, timeoutMs = 20_000): Promise<WsMessage> {
    const current = this.getState(gameID)
    const currentRevision = Number(current?.revision ?? -1)
    if (current && currentRevision >= minRevision) {
      return current
    }

    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const msg = await this.waitForType('game_state_update', Math.min(1_500, deadline - Date.now()))
      const state = (msg.payload ?? {}) as WsMessage
      const id = String(state.id ?? '')
      if (id !== gameID) {
        continue
      }
      const revision = Number(state.revision ?? -1)
      if (revision >= minRevision) {
        return state
      }
    }
    throw new Error(`timeout waiting for revision >= ${String(minRevision)} for game ${gameID}`)
  }

  getState(gameID: string): WsMessage | null {
    return this.statesByGame.get(gameID) ?? null
  }
}
