import { expect, type Page } from '@playwright/test'

type RealServerPageOptions = {
  disableExpectedRevision?: boolean
  waitForSummaryBar?: boolean
}

type TestSocketWindow = Window & {
  __TM_TEST_GET_REVISION__?: () => number | null
  __TM_DISABLE_EXPECTED_REVISION__?: boolean
  __TM_TEST_IS_CONNECTED__?: () => boolean
  __TM_TEST_SEND_MESSAGE__?: (message: unknown) => void
  __TM_TEST_GET_LOCAL_PLAYER_ID__?: () => string | null
  __TM_TEST_SET_LOCAL_PLAYER_ID__?: (playerId: string) => void
}

const realServerDebug = process.env.TM_E2E_DEBUG === '1'

const debugLog = (...args: unknown[]): void => {
  if (!realServerDebug) return
  // eslint-disable-next-line no-console
  console.log('[real-server-page]', ...args)
}

export async function primeRealServerPage(
  page: Page,
  playerId: string,
  options: RealServerPageOptions = {},
): Promise<void> {
  debugLog('prime', { playerId, disableExpectedRevision: options.disableExpectedRevision ?? false })
  await page.addInitScript(
    ({ localPlayerId, disableExpectedRevision }) => {
      // Observe the real transport from the test page. Production bundles never
      // need privileged application hooks to participate in this harness.
      let socket: WebSocket | null = null
      let revision: number | null = null
      const NativeWebSocket = window.WebSocket
      window.WebSocket = class extends NativeWebSocket {
        constructor(url: string | URL, protocols?: string | string[]) {
          super(url, protocols)
          socket = this
          this.addEventListener('message', event => {
            const message = JSON.parse(String(event.data)) as { type?: string; payload?: { revision?: number } }
            if (message.type === 'game_state_update') revision = message.payload?.revision ?? null
          })
        }
      }
      const testWindow = window as TestSocketWindow
      testWindow.__TM_TEST_IS_CONNECTED__ = () => socket?.readyState === NativeWebSocket.OPEN
      testWindow.__TM_TEST_GET_REVISION__ = () => revision
      testWindow.__TM_TEST_GET_LOCAL_PLAYER_ID__ = () => {
        const stored = JSON.parse(localStorage.getItem('tm-game-storage') ?? '{}') as { state?: { localPlayerId?: string } }
        return stored.state?.localPlayerId ?? null
      }
      testWindow.__TM_TEST_SEND_MESSAGE__ = message => {
        if (!socket || socket.readyState !== NativeWebSocket.OPEN) throw new Error('Real test socket is not connected')
        socket.send(typeof message === 'string' ? message : JSON.stringify(message))
      }
      const nextPlayer = sessionStorage.getItem('tm-test-next-player')
      if (nextPlayer || !localStorage.getItem('tm-game-storage')) {
        localStorage.setItem('tm-game-storage', JSON.stringify({ state: { localPlayerId: nextPlayer ?? localPlayerId }, version: 0 }))
        sessionStorage.removeItem('tm-test-next-player')
      }
      ;(window as TestSocketWindow).__TM_DISABLE_EXPECTED_REVISION__ = disableExpectedRevision
    },
    {
      localPlayerId: playerId,
      disableExpectedRevision: options.disableExpectedRevision ?? false,
    },
  )
}

export async function sendPageSocketMessage(page: Page, message: unknown): Promise<void> {
  debugLog('socket-send', message)
  await page.evaluate((payload) => {
    const testWindow = window as TestSocketWindow
    if (typeof testWindow.__TM_TEST_SEND_MESSAGE__ !== 'function') {
      throw new Error('test websocket send hook is unavailable')
    }
    testWindow.__TM_TEST_SEND_MESSAGE__(payload)
  }, message)
}

export async function setRealServerPageLocalPlayer(page: Page, playerId: string): Promise<void> {
  debugLog('set-local-player', { playerId })
  // A server connection is bound to its seat. Switching only the client store
  // leaves actions authenticated as the previous player; reconnect on every switch.
  await page.evaluate((nextPlayerId) => {
    sessionStorage.setItem('tm-test-next-player', nextPlayerId)
  }, playerId)
  await page.reload()
  await page.waitForFunction(() => (window as TestSocketWindow).__TM_TEST_IS_CONNECTED__?.() === true)
}

export async function loadRealServerGamePage(
  page: Page,
  gameID: string,
  playerId: string,
  options: RealServerPageOptions = {},
): Promise<void> {
  page.setDefaultTimeout(15_000)
  page.setDefaultNavigationTimeout(30_000)
  debugLog('load-start', { gameID, playerId })
  await page.goto(`/game/${gameID}`)
  await expect(page.getByTestId('game-screen')).toBeVisible()
  debugLog('game-screen-visible', { gameID, playerId })

  const deadline = Date.now() + 15_000
  let connected = false
  let lastStatus: Record<string, unknown> | null = null
  let nextDebugAt = 0
  while (Date.now() < deadline) {
    lastStatus = await page.evaluate(() => {
      const testWindow = window as TestSocketWindow
      return {
        connectedHookType: typeof testWindow.__TM_TEST_IS_CONNECTED__,
        sendHookType: typeof testWindow.__TM_TEST_SEND_MESSAGE__,
        isConnected:
          typeof testWindow.__TM_TEST_IS_CONNECTED__ === 'function'
            ? testWindow.__TM_TEST_IS_CONNECTED__()
            : null,
      }
    })
    if (lastStatus.isConnected === true) {
      connected = true
      break
    }
    if (realServerDebug && Date.now() >= nextDebugAt) {
      debugLog('socket-wait', { gameID, playerId, status: lastStatus })
      nextDebugAt = Date.now() + 1_000
    }
    await page.waitForTimeout(100)
  }
  if (!connected) {
    throw new Error(`test websocket did not connect for ${playerId} in game ${gameID}: ${JSON.stringify(lastStatus)}`)
  }
  debugLog('socket-connected', { gameID, playerId })

  await sendPageSocketMessage(page, {
    type: 'join_game',
    payload: { id: gameID, name: playerId },
  })

  await sendPageSocketMessage(page, {
    type: 'get_game_state',
    payload: { gameID, playerID: playerId },
  })

  if (options.waitForSummaryBar ?? true) {
    await expect(page.getByTestId('player-summary-bar')).toBeVisible({ timeout: 15_000 })
    debugLog('summary-visible', { gameID, playerId })
  }
}
