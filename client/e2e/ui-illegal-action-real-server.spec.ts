import { expect, test, type Browser, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import WebSocket from 'ws'
import { clickByTestId } from './support/uiInteractions'

type JsonObject = Record<string, unknown>

type GoldenAction = {
  playerId: string
  type: string
  params: JsonObject
}

type GoldenScript = {
  fixture: string
  playerIds: string[]
  scoringTiles: string[]
  bonusCards: string[]
  turnOrderPolicy?: string
  actions: GoldenAction[]
  expectedFinalScores: Record<string, number>
}

class WsBot {
  private readonly ws: WebSocket
  private readonly queueByType: Map<string, JsonObject[]> = new Map()
  private readonly statesByGame: Map<string, JsonObject> = new Map()

  private constructor(ws: WebSocket) {
    this.ws = ws
    this.ws.on('message', (raw) => {
      const payload = typeof raw === 'string' ? raw : raw.toString('utf8')
      let parsed: JsonObject
      try {
        parsed = JSON.parse(payload) as JsonObject
      } catch {
        return
      }
      const msgType = String(parsed.type ?? '')
      if (msgType === 'game_state_update') {
        const state = (parsed.payload ?? {}) as JsonObject
        const gameID = String(state.id ?? '')
        if (gameID !== '') this.statesByGame.set(gameID, state)
      }
      if (msgType !== '') {
        const queue = this.queueByType.get(msgType) ?? []
        queue.push(parsed)
        if (queue.length > 2_000) queue.splice(0, queue.length - 2_000)
        this.queueByType.set(msgType, queue)
      }
    })
  }

  static async connect(url: string): Promise<WsBot> {
    const ws = new WebSocket(url)
    await new Promise<void>((resolve, reject) => {
      ws.once('open', () => resolve())
      ws.once('error', (err) => reject(err))
    })
    return new WsBot(ws)
  }

  close(): void {
    this.ws.close()
  }

  send(type: string, payload?: JsonObject): void {
    this.ws.send(JSON.stringify({ type, payload }))
  }

  async waitForType(type: string, timeoutMs = 12_000): Promise<JsonObject> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const queue = this.queueByType.get(type)
      if (queue && queue.length > 0) {
        const msg = queue.shift()
        if (msg) return msg
      }
      await new Promise((resolve) => setTimeout(resolve, 25))
    }
    throw new Error(`timeout waiting for websocket message type=${type}`)
  }

  async waitForRevision(gameID: string, minRevision: number, timeoutMs = 20_000): Promise<JsonObject> {
    const current = this.statesByGame.get(gameID)
    const currentRevision = Number((current ?? {}).revision ?? -1)
    if (current && currentRevision >= minRevision) return current
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      await this.waitForType('game_state_update', Math.min(1_000, deadline - Date.now()))
      const latest = this.statesByGame.get(gameID)
      const rev = Number((latest ?? {}).revision ?? -1)
      if (latest && rev >= minRevision) return latest
    }
    throw new Error(`timeout waiting for revision >= ${String(minRevision)}`)
  }

  getState(gameID: string): JsonObject | null {
    return this.statesByGame.get(gameID) ?? null
  }
}

async function openPlayerPage(browser: Browser, gameID: string, playerId: string): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await page.addInitScript(
    ({ localPlayerId }) => {
      localStorage.setItem('tm-game-storage', JSON.stringify({ state: { localPlayerId }, version: 0 }))
    },
    { localPlayerId: playerId },
  )
  await page.goto(`/game/${gameID}`)
  await expect(page.getByTestId('game-screen')).toBeVisible()
  return page
}

async function createConfiguredGame(wsURL: string, script: GoldenScript, nameSuffix: string): Promise<{ creator: WsBot; gameID: string; revision: number }> {
  const creator = await WsBot.connect(wsURL)
  try {
    const creatorId = script.playerIds[0]
    creator.send('create_game', {
      name: `illegal-e2e-${nameSuffix}`,
      maxPlayers: script.playerIds.length,
      creator: creatorId,
    })
    const created = await creator.waitForType('game_created', 15_000)
    const payload = (created.payload ?? {}) as JsonObject
    const gameID = String(payload.gameId ?? '')
    if (gameID === '') throw new Error('missing game id from create_game')

    for (const playerId of script.playerIds.slice(1)) {
      const joiner = await WsBot.connect(wsURL)
      try {
        joiner.send('join_game', { id: gameID, name: playerId })
        await joiner.waitForType('game_joined', 15_000)
      } finally {
        joiner.close()
      }
    }

    creator.send('start_game', { gameID, randomizeTurnOrder: false, setupMode: 'snellman' })
    const initial = await creator.waitForRevision(gameID, 0, 20_000)
    let revision = Number(initial.revision ?? 0)

    creator.send('test_apply_fixture_settings', {
      gameID,
      scoringTiles: script.scoringTiles,
      bonusCards: script.bonusCards,
      turnOrderPolicy: script.turnOrderPolicy ?? 'pass_order',
    })
    await creator.waitForType('test_command_applied', 15_000)
    const configured = await creator.waitForRevision(gameID, revision + 1, 20_000)
    revision = Number(configured.revision ?? revision + 1)
    return { creator, gameID, revision }
  } catch (err) {
    creator.close()
    throw err
  }
}

const thisDir = path.dirname(fileURLToPath(import.meta.url))
const s69Script = JSON.parse(
  fs.readFileSync(path.resolve(thisDir, 'fixtures', 's69_g2_actions.json'), 'utf8'),
) as GoldenScript

test.describe('Illegal Action UI (Real Server)', () => {
  test.setTimeout(180_000)

  test('@smoke wrong-turn controls are blocked in the UI', async ({ browser }) => {
    const wsURL = 'ws://127.0.0.1:8080/api/ws'
    const { creator, gameID, revision } = await createConfiguredGame(wsURL, s69Script, 'wrong-turn')
    const p2 = await openPlayerPage(browser, gameID, s69Script.playerIds[1])
    try {
      const factionButton = p2.getByTestId('faction-option-Auren')
      await expect(factionButton).toBeVisible()
      await expect(factionButton).toBeDisabled()
      await factionButton.click({ force: true })
      await expect.poll(async () => {
        const state = creator.getState(gameID)
        return Number((state ?? {}).revision ?? -1)
      }).toBe(revision)
    } finally {
      await p2.context().close()
      creator.close()
    }
  })

  test('@smoke server rejection is surfaced as visible action error banner', async ({ browser }) => {
    const wsURL = 'ws://127.0.0.1:8080/api/ws'
    const { creator, gameID, revision: initialRevision } = await createConfiguredGame(wsURL, s69Script, 'rejection')
    const playerId = s69Script.playerIds[0]
    const playerPage = await openPlayerPage(browser, gameID, playerId)
    try {
      creator.send('test_replay_actions_to_index', {
        gameID,
        endExclusive: 17,
        actions: s69Script.actions,
      })
      const ack = await creator.waitForType('test_command_applied', 20_000)
      const ackPayload = (ack.payload ?? {}) as JsonObject
      const replayRevision = Number(ackPayload.newRevision ?? initialRevision)
      await creator.waitForRevision(gameID, replayRevision, 20_000)

      await clickByTestId(playerPage, `player-${playerId}-conversion-power_to_priest`)

      const banner = playerPage.getByTestId('action-error-message')
      await expect(banner).toBeVisible({ timeout: 10_000 })
      const bannerText = (await banner.textContent()) ?? ''
      expect(bannerText.trim().length).toBeGreaterThan(0)
      expect(bannerText).toMatch(/need|not enough|insufficient|only have/i)

      const currentRevision = Number((creator.getState(gameID) ?? {}).revision ?? replayRevision)
      await expect.poll(async () => {
        return Number((creator.getState(gameID) ?? {}).revision ?? -1)
      }).toBe(currentRevision)
    } finally {
      await playerPage.context().close()
      creator.close()
    }
  })
})
