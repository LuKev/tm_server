import { expect, test, type Page } from '@playwright/test'
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
const thisDir = path.dirname(fileURLToPath(import.meta.url))
const scriptPath = path.resolve(thisDir, 'fixtures', 's69_g2_actions.json')
const goldenScript = JSON.parse(fs.readFileSync(scriptPath, 'utf8')) as GoldenScript

async function prepareGameObserver(page: Page, gameID: string, localPlayerId: string): Promise<void> {
  await page.addInitScript(
    ({ playerId }) => {
      localStorage.setItem(
        'tm-game-storage',
        JSON.stringify({
          state: { localPlayerId: playerId },
          version: 0,
        }),
      )
    },
    { playerId: localPlayerId },
  )

  await page.goto(`/game/${gameID}`)
  await expect(page.getByTestId('game-screen')).toBeVisible()
  await expect(page.getByTestId('player-summary-bar')).toBeVisible()
}

test.describe('Golden Full-Game Completion (Real Server + UI Observer)', () => {
  test.setTimeout(300_000)

  test('replays S69_G2 to completion and matches final scores in UI', async ({ page }) => {
    const wsURL = 'ws://127.0.0.1:8080/api/ws'

    const bots = new Map<string, WsBot>()
    try {
      for (const playerId of goldenScript.playerIds) {
        bots.set(playerId, await WsBot.connect(wsURL))
      }

      const creatorId = goldenScript.playerIds[0]
      const creator = bots.get(creatorId)
      if (!creator) throw new Error(`missing creator bot for ${creatorId}`)

      creator.send('create_game', {
        name: 'golden-s69-g2-ui',
        maxPlayers: goldenScript.playerIds.length,
        creator: creatorId,
      })
      const created = await creator.waitForType('game_created', 15_000)
      const createdPayload = (created.payload ?? {}) as JsonObject
      const gameID = String(createdPayload.gameId ?? '')
      if (gameID === '') {
        throw new Error('missing gameId from game_created payload')
      }

      for (const playerId of goldenScript.playerIds.slice(1)) {
        const bot = bots.get(playerId)
        if (!bot) throw new Error(`missing bot for ${playerId}`)
        bot.send('join_game', { id: gameID, name: playerId })
        await bot.waitForType('game_joined', 15_000)
      }

      creator.send('start_game', {
        gameID: gameID,
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

      await prepareGameObserver(page, gameID, creatorId)

      for (let index = 0; index < goldenScript.actions.length; index++) {
        const action = goldenScript.actions[index]
        const actor = bots.get(action.playerId)
        if (!actor) {
          throw new Error(`missing actor bot: ${action.playerId}`)
        }

        const actionId = `ui-golden-${String(index).padStart(4, '0')}`
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
        if (!entry) {
          throw new Error(`missing final scoring entry for ${playerId}`)
        }
        const got = Number(entry.totalVp ?? -1)
        expect(got).toBe(expected)
      }

      await expect.poll(async () => {
        const summaryText = (await page.getByTestId('player-summary-bar').innerText()).replace(/\\s+/g, ' ')
        return summaryText
      }).toContain('166 VP')

      await expect(page.getByTestId('player-summary-bar')).toContainText('166 VP')
      await expect(page.getByTestId('player-summary-bar')).toContainText('137 VP')
      await expect(page.getByTestId('player-summary-bar')).toContainText('130 VP')
      await expect(page.getByTestId('player-summary-bar')).toContainText('124 VP')
    } finally {
      for (const bot of bots.values()) {
        bot.close()
      }
    }
  })
})
