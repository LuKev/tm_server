import { expect, test, type Browser, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { realServerWsURL } from './support/realServerConfig'
import { loadRealServerGamePage, primeRealServerPage, sendPageSocketMessage } from './support/realServerPage'
import { WsBot, type JsonObject } from './support/wsBot'

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

const debugEnabled = process.env.TM_E2E_DEBUG === '1'

const debugLog = (...args: unknown[]): void => {
  if (!debugEnabled) return
  // eslint-disable-next-line no-console
  console.log('[illegal-e2e]', ...args)
}


async function openPlayerPage(browser: Browser, gameID: string, playerId: string): Promise<Page> {
  debugLog('open-player-page:start', { gameID, playerId })
  const context = await browser.newContext()
  const page = await context.newPage()
  await primeRealServerPage(page, playerId, { disableExpectedRevision: true })
  await loadRealServerGamePage(page, gameID, playerId)
  debugLog('open-player-page:ready', { gameID, playerId })
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
    debugLog('game-created', { gameID, nameSuffix })

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
    debugLog('game-started', { gameID, revision })

    creator.send('test_apply_fixture_settings', {
      gameID,
      scoringTiles: script.scoringTiles,
      bonusCards: script.bonusCards,
      turnOrderPolicy: script.turnOrderPolicy ?? 'pass_order',
    })
    await creator.waitForType('test_command_applied', 15_000)
    const configured = await creator.waitForRevision(gameID, revision + 1, 20_000)
    revision = Number(configured.revision ?? revision + 1)
    debugLog('fixture-configured', { gameID, revision })
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
    const wsURL = realServerWsURL()
    const { creator, gameID, revision } = await createConfiguredGame(wsURL, s69Script, 'wrong-turn')
    const state = creator.getState(gameID)
    const turnOrder = ((state ?? {}).turnOrder ?? []) as string[]
    const currentTurn = Number((state ?? {}).currentTurn ?? 0)
    const currentPlayerId = turnOrder[currentTurn] ?? s69Script.playerIds[0]
    const nonCurrentPlayerId = s69Script.playerIds.find((playerId) => playerId !== currentPlayerId) ?? s69Script.playerIds[1]
    debugLog('wrong-turn-target', { gameID, currentPlayerId, nonCurrentPlayerId, revision })
    const page = await openPlayerPage(browser, gameID, nonCurrentPlayerId)
    try {
      const workerToCoin = page.getByTestId(`player-${nonCurrentPlayerId}-conversion-worker_to_coin`)
      const powerToPriest = page.getByTestId(`player-${nonCurrentPlayerId}-conversion-power_to_priest`)
      await expect(workerToCoin).toBeVisible()
      await expect(powerToPriest).toBeVisible()
      await expect(page.getByTestId('player-summary-bar')).toContainText('TURN')
      await expect(page.getByTestId('player-summary-bar')).not.toContainText(`${nonCurrentPlayerId} TURN`)
      await workerToCoin.click()
      await expect.poll(async () => {
        const state = creator.getState(gameID)
        return Number((state ?? {}).revision ?? -1)
      }).toBe(revision)
    } finally {
      await page.context().close()
      creator.close()
    }
  })

  test('@smoke server rejection is surfaced as visible action error banner', async ({ browser }) => {
    const wsURL = realServerWsURL()
    const { creator, gameID, revision: initialRevision } = await createConfiguredGame(wsURL, s69Script, 'rejection')
    const state = creator.getState(gameID)
    const turnOrder = ((state ?? {}).turnOrder ?? []) as string[]
    const currentTurn = Number((state ?? {}).currentTurn ?? 0)
    const playerId = turnOrder[currentTurn] ?? s69Script.playerIds[0]
    debugLog('rejection-target', { gameID, playerId, initialRevision })
    const playerPage = await openPlayerPage(browser, gameID, playerId)
    try {
      await sendPageSocketMessage(playerPage, {
        type: 'perform_action',
        payload: {
          type: 'conversion',
          gameID,
          actionId: 'illegal-conversion-e2e',
          params: {
            conversionType: 'power_to_priest',
            amount: 99,
          },
        },
      })
      debugLog('illegal-action-sent', { gameID, playerId })

      const banner = playerPage.getByTestId('action-error-message')
      await expect(banner).toBeVisible({ timeout: 10_000 })
      const bannerText = (await banner.textContent()) ?? ''
      expect(bannerText.trim().length).toBeGreaterThan(0)

      const currentRevision = Number((creator.getState(gameID) ?? {}).revision ?? initialRevision)
      await expect.poll(async () => {
        return Number((creator.getState(gameID) ?? {}).revision ?? -1)
      }).toBe(currentRevision)
    } finally {
      await playerPage.context().close()
      creator.close()
    }
  })
})
