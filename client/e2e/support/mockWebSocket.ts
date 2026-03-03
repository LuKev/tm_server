import { expect, type Page } from '@playwright/test'

export type WsMessage = Record<string, unknown>

declare global {
  interface Window {
    __tmE2E?: {
      sent: unknown[]
      sockets: Array<{ emitMessage: (message: unknown) => void; readyState: number }>
      emit: (message: unknown) => void
      clearSent: () => void
      performActions: () => Array<{ type: string; params: Record<string, unknown> }>
    }
  }
}

export async function installMockWebSocket(page: Page, localPlayerId = 'p1'): Promise<void> {
  await page.addInitScript((playerId: string) => {
    const sent: unknown[] = []
    const sockets: Array<{ emitMessage: (message: unknown) => void; readyState: number }> = []

    class MockWebSocket {
      static CONNECTING = 0
      static OPEN = 1
      static CLOSING = 2
      static CLOSED = 3

      readyState = MockWebSocket.CONNECTING
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null

      constructor(url: string) {
        this.url = url
        const socketRef = {
          readyState: this.readyState,
          emitMessage: (message: unknown) => {
            if (this.readyState !== MockWebSocket.OPEN) return
            this.onmessage?.({ data: JSON.stringify(message) } as MessageEvent)
          },
        }
        sockets.push(socketRef)

        setTimeout(() => {
          this.readyState = MockWebSocket.OPEN
          socketRef.readyState = this.readyState
          this.onopen?.(new Event('open'))
        }, 0)
      }

      send(data: string): void {
        try {
          sent.push(JSON.parse(data) as unknown)
        } catch {
          sent.push(data)
        }
      }

      close(): void {
        this.readyState = MockWebSocket.CLOSED
        this.onclose?.(new CloseEvent('close'))
      }

      addEventListener(): void {
        // no-op for test harness
      }

      removeEventListener(): void {
        // no-op for test harness
      }
    }

    const performActions = (): Array<{ type: string; params: Record<string, unknown> }> => {
      return sent
        .filter((raw) => typeof raw === 'object' && raw !== null)
        .map((raw) => raw as Record<string, unknown>)
        .filter((msg) => msg.type === 'perform_action')
        .map((msg) => {
          const payload = (msg.payload ?? {}) as Record<string, unknown>
          return {
            type: String(payload.type ?? ''),
            params: (payload.params ?? {}) as Record<string, unknown>,
          }
        })
    }

    window.__tmE2E = {
      sent,
      sockets,
      emit: (message: unknown) => {
        sockets.forEach((socket) => {
          socket.emitMessage(message)
        })
      },
      clearSent: () => {
        sent.length = 0
      },
      performActions,
    }

    Object.defineProperty(window, 'WebSocket', {
      writable: true,
      configurable: true,
      value: MockWebSocket,
    })

    localStorage.setItem(
      'tm-game-storage',
      JSON.stringify({
        state: { localPlayerId: playerId },
        version: 0,
      }),
    )
  }, localPlayerId)
}

export async function emitWs(page: Page, message: WsMessage): Promise<void> {
  await page.evaluate((msg) => {
    window.__tmE2E?.emit(msg)
  }, message)
}

export async function waitForSocketReady(page: Page): Promise<void> {
  await expect
    .poll(async () => {
      return page.evaluate(() => {
        const sockets = window.__tmE2E?.sockets ?? []
        return sockets.some((socket) => socket.readyState === 1)
      })
    })
    .toBe(true)
}

export async function clearSentMessages(page: Page): Promise<void> {
  await page.evaluate(() => {
    window.__tmE2E?.clearSent()
  })
}

export async function waitForPerformAction(
  page: Page,
  expectedType: string,
  expectedParams?: Record<string, unknown>,
): Promise<void> {
  await expect
    .poll(async () => {
      return page.evaluate(() => {
        const actions = window.__tmE2E?.performActions() ?? []
        return actions.length > 0 ? actions[actions.length - 1] : null
      })
    })
    .not.toBeNull()

  const last = await page.evaluate(() => {
    const actions = window.__tmE2E?.performActions() ?? []
    return actions[actions.length - 1] ?? null
  })

  expect(last?.type).toBe(expectedType)
  if (expectedParams !== undefined) {
    expect(last?.params).toEqual(expectedParams)
  }
}

export async function waitForMessageType(page: Page, messageType: string): Promise<void> {
  await expect
    .poll(async () => {
      return page.evaluate((type) => {
        const msgs = window.__tmE2E?.sent ?? []
        return msgs.some((msg) => {
          if (typeof msg !== 'object' || msg === null) return false
          return (msg as Record<string, unknown>).type === type
        })
      }, messageType)
    })
    .toBe(true)
}
