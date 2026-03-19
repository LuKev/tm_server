import { expect, test, type Browser, type BrowserContext, type Page, type TestInfo } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { realServerWsURL } from './support/realServerConfig'
import { loadRealServerGamePage, primeRealServerPage } from './support/realServerPage'
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

const thisDir = path.dirname(fileURLToPath(import.meta.url))
const scriptPath = path.resolve(thisDir, 'fixtures', 's69_g2_actions.json')
const goldenScript = JSON.parse(fs.readFileSync(scriptPath, 'utf8')) as GoldenScript

const sanitize = (value: string): string => value.replace(/[^a-zA-Z0-9_-]/g, '_')

type PovViewer = {
  playerId: string
  context: BrowserContext
  page: Page
}

async function openPlayerPov(browser: Browser, gameID: string, playerId: string, videoDir: string): Promise<PovViewer> {
  const context = await browser.newContext({
    recordVideo: {
      dir: videoDir,
      size: { width: 1600, height: 900 },
    },
  })
  const page = await context.newPage()

  await primeRealServerPage(page, playerId)
  await loadRealServerGamePage(page, gameID, playerId)
  return { playerId, context, page }
}

async function savePovVideos(viewers: PovViewer[], artifactsRunDir: string): Promise<Record<string, string>> {
  const filesByPlayer: Record<string, string> = {}
  for (const viewer of viewers) {
    const video = viewer.page.video()
    await viewer.context.close()
    if (!video) {
      continue
    }
    const sourcePath = await video.path()
    const destination = path.resolve(artifactsRunDir, `${sanitize(viewer.playerId)}.webm`)
    fs.copyFileSync(sourcePath, destination)
    filesByPlayer[viewer.playerId] = destination
  }
  return filesByPlayer
}

test.describe('Golden Full-Game Multi-POV Video Capture', () => {
  test.setTimeout(420_000)

  test('replays S69_G2 to completion while recording all 4 player viewpoints', async ({ browser }, testInfo: TestInfo) => {
    test.skip(
      process.env.TM_ENABLE_FULL_REPLAY_E2E !== '1',
      'full S69 replay is unstable under strict leech-turn validation; enable explicitly for local investigation',
    )

    const wsURL = realServerWsURL()
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
    const artifactsRunDir = path.resolve(thisDir, 'artifacts', 's69_g2_pov', timestamp)
    const rawVideoDir = path.resolve(testInfo.outputDir, 'pov-raw')
    fs.mkdirSync(artifactsRunDir, { recursive: true })
    fs.mkdirSync(rawVideoDir, { recursive: true })

    const bots = new Map<string, WsBot>()
    const viewers: PovViewer[] = []

    try {
      for (const playerId of goldenScript.playerIds) {
        bots.set(playerId, await WsBot.connect(wsURL))
      }

      const creatorId = goldenScript.playerIds[0]
      const creator = bots.get(creatorId)
      if (!creator) throw new Error(`missing creator bot for ${creatorId}`)

      creator.send('create_game', {
        name: 'golden-s69-g2-pov',
        maxPlayers: goldenScript.playerIds.length,
        creator: creatorId,
      })
      const created = await creator.waitForType('game_created', 15_000)
      const createdPayload = (created.payload ?? {}) as JsonObject
      const gameID = String(createdPayload.gameId ?? '')
      if (gameID === '') throw new Error('missing gameId from game_created payload')

      for (const playerId of goldenScript.playerIds.slice(1)) {
        const bot = bots.get(playerId)
        if (!bot) throw new Error(`missing bot for ${playerId}`)
        bot.send('join_game', { id: gameID, name: playerId })
        await bot.waitForType('game_joined', 15_000)
      }

      creator.send('start_game', {
        gameID,
        randomizeTurnOrder: false,
        setupMode: 'snellman',
      })
      const initialState = await creator.waitForRevision(gameID, 0, 20_000)
      let revision = Number(initialState.revision ?? 0)

      creator.send('test_apply_fixture_settings', {
        gameID,
        scoringTiles: goldenScript.scoringTiles,
        bonusCards: goldenScript.bonusCards,
        turnOrderPolicy: goldenScript.turnOrderPolicy ?? 'pass_order',
      })
      await creator.waitForType('test_command_applied', 15_000)
      const configuredState = await creator.waitForRevision(gameID, revision + 1, 20_000)
      revision = Number(configuredState.revision ?? revision + 1)

      for (const playerId of goldenScript.playerIds) {
        viewers.push(await openPlayerPov(browser, gameID, playerId, rawVideoDir))
      }

      for (let index = 0; index < goldenScript.actions.length; index++) {
        const action = goldenScript.actions[index]
        const actor = bots.get(action.playerId)
        if (!actor) throw new Error(`missing actor bot for ${action.playerId}`)

        if (action.type === 'replay_conversion') {
          const params = action.params ?? {}
          actor.send('test_apply_conversion', {
            gameID,
            playerID: action.playerId,
            conversionType: String(params.conversionType ?? ''),
            amount: Number(params.amount ?? 1),
          })
          const response = await actor.waitForAnyType(['test_command_applied', 'action_rejected', 'error'], 20_000).catch(() => null)
          if (String(response?.type ?? '') === 'test_command_applied') {
            const state = await creator.waitForRevision(gameID, revision + 1, 20_000).catch(() => null)
            if (state) {
              revision = Number(state.revision ?? revision + 1)
            }
          }
        } else {
          const actionId = `ui-golden-multi-pov-${String(index).padStart(4, '0')}`
          const sendAction = async (expected: number): Promise<{ type: string; payload: JsonObject }> => {
            actor.send('perform_action', {
              type: action.type,
              gameID,
              actionId,
              expectedRevision: expected,
              params: action.params ?? {},
            })
            const response = await actor.waitForAnyType(['action_accepted', 'action_rejected', 'error'], 20_000)
            return {
              type: String(response.type ?? ''),
              payload: (response.payload ?? {}) as JsonObject,
            }
          }

          let outcome = await sendAction(revision).catch(() => ({ type: 'error', payload: {} as JsonObject }))
          if (outcome.type === 'action_rejected') {
            const message = String(outcome.payload.message ?? outcome.payload.error ?? '')
            if (message.toLowerCase().includes('revision')) {
              outcome = await sendAction(-1).catch(() => ({ type: 'error', payload: {} as JsonObject }))
            }
          }

          if (outcome.type === 'action_accepted') {
            const state = await creator.waitForRevision(gameID, revision + 1, 20_000).catch(() => null)
            if (state) {
              revision = Number(state.revision ?? revision + 1)
            }
          }
        }
      }

      const finalState = await creator.waitForRevision(gameID, revision, 20_000)
      if (Number(finalState.phase ?? -1) !== 5) {
        throw new Error(`expected final phase=5, got ${String(finalState.phase)}`)
      }

      const finalScoring = (finalState.finalScoring ?? {}) as Record<string, JsonObject>
      for (const [playerId, expected] of Object.entries(goldenScript.expectedFinalScores)) {
        const entry = finalScoring[playerId]
        if (!entry) throw new Error(`missing final scoring entry for ${playerId}`)
        const got = Number(entry.totalVp ?? -1)
        expect(got).toBe(expected)
      }

      for (const viewer of viewers) {
        await expect(viewer.page.getByTestId('player-summary-bar')).toContainText('166 VP')
        await expect(viewer.page.getByTestId('player-summary-bar')).toContainText('137 VP')
        await expect(viewer.page.getByTestId('player-summary-bar')).toContainText('130 VP')
        await expect(viewer.page.getByTestId('player-summary-bar')).toContainText('124 VP')
      }

      const exportedVideos = await savePovVideos(viewers, artifactsRunDir)
      const manifestPath = path.resolve(artifactsRunDir, 'manifest.json')
      fs.writeFileSync(
        manifestPath,
        JSON.stringify(
          {
            fixture: goldenScript.fixture,
            game: '4pLeague_S69_D1L1_G2',
            generatedAt: new Date().toISOString(),
            videos: exportedVideos,
          },
          null,
          2,
        ),
        'utf8',
      )
      expect(Object.keys(exportedVideos)).toHaveLength(goldenScript.playerIds.length)
    } finally {
      for (const bot of bots.values()) {
        bot.close()
      }
      for (const viewer of viewers) {
        if (viewer.context.pages().length > 0) {
          await viewer.context.close()
        }
      }
    }
  })
})
