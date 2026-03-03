import { expect, test, type Browser, type BrowserContext, type Page, type TestInfo } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
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

  await page.addInitScript(
    ({ localPlayerId }) => {
      localStorage.setItem(
        'tm-game-storage',
        JSON.stringify({
          state: { localPlayerId },
          version: 0,
        }),
      )
    },
    { localPlayerId: playerId },
  )

  await page.goto(`/game/${gameID}`)
  await expect(page.getByTestId('game-screen')).toBeVisible()
  await expect(page.getByTestId('player-summary-bar')).toBeVisible()
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
    const wsURL = 'ws://127.0.0.1:8080/api/ws'
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
        if (!actor) throw new Error(`missing actor bot: ${action.playerId}`)

        const actionId = `ui-golden-multi-pov-${String(index).padStart(4, '0')}`
        actor.send('perform_action', {
          type: action.type,
          gameID,
          actionId,
          expectedRevision: revision,
          params: action.params ?? {},
        })

        const accepted = await actor.waitForType('action_accepted', 20_000)
        const acceptedPayload = (accepted.payload ?? {}) as JsonObject
        const acceptedActionId = String(acceptedPayload.actionId ?? '')
        if (acceptedActionId !== actionId) {
          throw new Error(`unexpected action_accepted id: got=${acceptedActionId} expected=${actionId}`)
        }

        const state = await actor.waitForRevision(gameID, revision + 1, 20_000)
        revision = Number(state.revision ?? revision + 1)
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
