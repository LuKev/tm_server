import { expect, test, type Browser, type Page } from '@playwright/test'
import fs from 'node:fs'
import WebSocket from 'ws'
import { clickByTestId, clickHex } from './support/uiInteractions'
import { GOLDEN_SCENARIOS } from './fixtures/golden_scenarios'

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

type Segment = {
  start: number
  endExclusive: number
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
        if (gameID !== '') {
          this.statesByGame.set(gameID, state)
        }
      }
      if (msgType !== '') {
        const queue = this.queueByType.get(msgType) ?? []
        queue.push(parsed)
        if (queue.length > 8_000) {
          queue.splice(0, queue.length - 8_000)
        }
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

  async waitForType(type: string, timeoutMs = 10_000): Promise<JsonObject> {
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

  async waitForAnyType(types: string[], timeoutMs = 10_000): Promise<JsonObject> {
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

  async waitForRevision(gameID: string, minRevision: number, timeoutMs = 20_000): Promise<JsonObject> {
    const current = this.getState(gameID)
    const currentRevision = Number(current?.revision ?? -1)
    if (current && currentRevision >= minRevision) {
      return current
    }

    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const msg = await this.waitForType('game_state_update', Math.min(1_500, deadline - Date.now()))
      const state = (msg.payload ?? {}) as JsonObject
      if (String(state.id ?? '') !== gameID) {
        continue
      }
      const revision = Number(state.revision ?? -1)
      if (revision >= minRevision) {
        return state
      }
    }
    throw new Error(`timeout waiting for revision >= ${String(minRevision)} for game ${gameID}`)
  }

  getState(gameID: string): JsonObject | null {
    return this.statesByGame.get(gameID) ?? null
  }
}

const asRecord = (v: unknown): JsonObject => (typeof v === 'object' && v !== null ? (v as JsonObject) : {})

const readInt = (params: JsonObject, ...keys: string[]): number => {
  for (const key of keys) {
    const raw = params[key]
    if (typeof raw === 'number' && Number.isFinite(raw)) return Math.trunc(raw)
    if (typeof raw === 'string' && raw.trim() !== '' && Number.isFinite(Number(raw))) return Math.trunc(Number(raw))
  }
  throw new Error(`missing numeric param from keys: ${keys.join(', ')}`)
}

const readBool = (params: JsonObject, key: string, fallback = false): boolean => {
  const raw = params[key]
  if (typeof raw === 'boolean') return raw
  if (typeof raw === 'string') return raw.toLowerCase() === 'true'
  return fallback
}

const readHex = (params: JsonObject, ...keys: string[]): { q: number; r: number } => {
  for (const key of keys) {
    const raw = params[key]
    const obj = asRecord(raw)
    const qRaw = obj.q ?? obj.Q
    const rRaw = obj.r ?? obj.R
    if (typeof qRaw === 'number' && typeof rRaw === 'number') {
      return { q: Math.trunc(qRaw), r: Math.trunc(rRaw) }
    }
  }
  throw new Error(`missing hex param from keys: ${keys.join(', ')}`)
}

async function maybeConfirmAction(page: Page): Promise<void> {
  const confirm = page.getByTestId('confirm-action-confirm').first()
  const appeared = await confirm.waitFor({ state: 'visible', timeout: 1_500 }).then(() => true).catch(() => false)
  if (appeared) {
    await confirm.click()
  }
}

async function hasVisibleTestId(page: Page, testId: string, timeoutMs = 1_500): Promise<boolean> {
  const locator = page.getByTestId(testId).first()
  const attached = await locator.waitFor({ state: 'attached', timeout: timeoutMs }).then(() => true).catch(() => false)
  if (!attached) return false
  return locator.isVisible().catch(() => false)
}

async function pickCultSpot(page: Page, track: number, spaces: number): Promise<void> {
  const preferred = spaces === 3 ? [0] : spaces === 1 ? [4] : [1, 2, 3]
  const fallback = [0, 1, 2, 3, 4].filter((idx) => !preferred.includes(idx))
  const candidates = [...preferred, ...fallback]
  for (const idx of candidates) {
    const spot = page.getByTestId(`cult-spot-${String(track)}-${String(idx)}`).first()
    const visible = await spot.isVisible().catch(() => false)
    const enabled = await spot.isEnabled().catch(() => false)
    if (visible && enabled) {
      await spot.click()
      return
    }
  }
  throw new Error(`no clickable cult spot found for track=${String(track)} spaces=${String(spaces)}`)
}

async function submitHexModal(page: Page, buildDwelling: boolean, targetTerrain?: number): Promise<void> {
  const modal = page.getByTestId('hex-action-modal')
  await expect(modal).toBeVisible({ timeout: 10_000 })

  const modeSelect = page.getByTestId('hex-action-mode')
  const modeVisible = await modeSelect.first().isVisible().catch(() => false)
  if (modeVisible) {
    if (!buildDwelling) {
      await modeSelect.selectOption('transform_only')
    } else if (targetTerrain !== undefined) {
      await modeSelect.selectOption('transform_build')
    }
  }

  if (targetTerrain !== undefined) {
    const terrainSelect = page.getByTestId('hex-action-target-terrain')
    const terrainVisible = await terrainSelect.first().isVisible().catch(() => false)
    if (terrainVisible) {
      await terrainSelect.selectOption(String(targetTerrain))
    }
  }

  await clickByTestId(page, 'hex-action-submit')
}

async function chooseCultTrackWithFallback(
  page: Page,
  cultTrack: number,
  retryOpen?: () => Promise<void>,
): Promise<void> {
  const modal = page.getByTestId('cult-choice-modal').first()
  const initialVisible = await modal.isVisible().catch(() => false)
  if (!initialVisible && retryOpen) {
    await retryOpen()
  }
  await expect(modal).toBeVisible({ timeout: 10_000 })
  await clickByTestId(page, `cult-choice-${String(cultTrack)}`)
}

async function chooseCultistsTrack(page: Page, cultTrack: number): Promise<void> {
  const modal = page.getByTestId('cultists-cult-choice-modal').first()
  const visible = await modal.isVisible().catch(() => false)
  if (visible) {
    await clickByTestId(page, `cultists-cult-choice-${String(cultTrack)}`)
    return
  }
  await chooseCultTrackWithFallback(page, cultTrack)
}

async function openPlayerViewer(browser: Browser, gameID: string, playerId: string): Promise<Page> {
  const context = await browser.newContext()
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
  return page
}

async function createConfiguredGame(wsURL: string, script: GoldenScript, nameSuffix: string): Promise<{ creator: WsBot; gameID: string; revision: number }> {
  const creator = await WsBot.connect(wsURL)
  try {
    const creatorId = script.playerIds[0]
    creator.send('create_game', {
      name: `golden-${nameSuffix}`,
      maxPlayers: script.playerIds.length,
      creator: creatorId,
    })
    const created = await creator.waitForType('game_created', 15_000)
    const createdPayload = (created.payload ?? {}) as JsonObject
    const gameID = String(createdPayload.gameId ?? '')
    if (gameID === '') throw new Error('missing gameId from game_created payload')

    for (const playerId of script.playerIds.slice(1)) {
      const joiner = await WsBot.connect(wsURL)
      try {
        joiner.send('join_game', { id: gameID, name: playerId })
        await joiner.waitForType('game_joined', 15_000)
      } finally {
        joiner.close()
      }
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
      scoringTiles: script.scoringTiles,
      bonusCards: script.bonusCards,
      turnOrderPolicy: script.turnOrderPolicy ?? 'pass_order',
    })
    const applyCommandResponse = await creator.waitForAnyType(
      ['test_command_applied', 'action_rejected', 'error'],
      15_000,
    )
    const responseType = String(applyCommandResponse.type ?? '')
    if (responseType !== 'test_command_applied') {
      const errorMessage = applyCommandResponse.payload
        ? String((applyCommandResponse.payload as JsonObject).message ?? (applyCommandResponse.payload as JsonObject).error ?? '')
        : 'no payload'
      throw new Error(`fixture settings apply failed for ${nameSuffix}: type=${responseType} payload=${errorMessage}`)
    }
    const configuredState = await creator.waitForRevision(gameID, revision + 1, 20_000)
    revision = Number(configuredState.revision ?? revision + 1)

    return { creator, gameID, revision }
  } catch (err) {
    creator.close()
    throw err
  }
}

function buildSegments(actionsLength: number): Segment[] {
  const rawSize = Number(process.env.TM_CLICK_SEGMENT_SIZE ?? 24)
  const segmentSize = Number.isFinite(rawSize) ? Math.max(1, Math.trunc(rawSize)) : 24

  const all: Segment[] = []
  for (let start = 0; start < actionsLength; start += segmentSize) {
    all.push({
      start,
      endExclusive: Math.min(actionsLength, start + segmentSize),
    })
  }

  const startSegment = Math.max(0, Math.trunc(Number(process.env.TM_CLICK_SEGMENT_START ?? 0)))
  const endSegment = Math.min(all.length, Math.trunc(Number(process.env.TM_CLICK_SEGMENT_END ?? all.length)))
  return all.slice(startSegment, Math.max(startSegment, endSegment))
}

async function clickAction(page: Page, creator: WsBot, gameID: string, action: GoldenAction, index: number): Promise<boolean> {
  const params = action.params ?? {}

  switch (action.type) {
    case 'select_faction': {
      const faction = String(params.faction ?? '')
      await clickByTestId(page, `faction-option-${faction}`)
      await maybeConfirmAction(page)
      return true
    }

    case 'setup_dwelling': {
      const hex = readHex(params, 'hex', 'targetHex')
      await clickHex(page, hex.q, hex.r)
      await maybeConfirmAction(page)
      return true
    }

    case 'setup_bonus_card': {
      const bonusCard = readInt(params, 'bonusCard')
      await clickByTestId(page, `setup-bonus-card-${String(bonusCard)}`)
      return true
    }

    case 'accept_leech':
    case 'decline_leech': {
      const offerIndex = readInt(params, 'offerIndex')
      const decision = action.type === 'accept_leech' ? 'accept' : 'decline'
      const testId = `leech-offer-${String(offerIndex)}-${decision}`
      const isVisible = await hasVisibleTestId(page, testId)
      if (!isVisible) {
        const snapshot = creator.getState(gameID)
        const pendingDecision = asRecord(snapshot?.pendingDecision)
        const pendingType = String(pendingDecision.type ?? '')
        const pendingPlayerID = String(pendingDecision.playerId ?? '')
        if (pendingType === 'leech_offer' && pendingPlayerID === action.playerId) {
          throw new Error(`expected leech decision control ${testId} for ${action.playerId}, but it was not visible`)
        }
        console.log(`[click-golden] step=${String(index).padStart(4, '0')} auto-resolved leech for ${action.playerId}`)
        return false
      }
      await clickByTestId(page, testId)
      return true
    }

    case 'send_priest': {
      const track = readInt(params, 'track', 'cultTrack')
      const spaces = readInt(params, 'spaces', 'spacesToClimb')
      await pickCultSpot(page, track, spaces)
      await maybeConfirmAction(page)
      return true
    }

    case 'transform_build': {
      const hex = readHex(params, 'targetHex', 'hex')
      const buildDwelling = readBool(params, 'buildDwelling', true)
      const hasTargetTerrain = params.targetTerrain !== undefined
      const targetTerrain = hasTargetTerrain ? readInt(params, 'targetTerrain') : undefined
      await clickHex(page, hex.q, hex.r)
      await submitHexModal(page, buildDwelling, targetTerrain)
      return true
    }

    case 'upgrade_building': {
      const hex = readHex(params, 'targetHex', 'upgradeHex')
      const newBuildingType = readInt(params, 'newBuildingType')
      await clickHex(page, hex.q, hex.r)
      await clickByTestId(page, `upgrade-option-${String(newBuildingType)}`)
      return true
    }

    case 'conversion': {
      const conversionType = String(params.conversionType ?? '')
      const amount = Math.max(1, readInt(params, 'amount'))
      for (let i = 0; i < amount; i++) {
        await clickByTestId(page, `player-${action.playerId}-conversion-${conversionType}`)
        await maybeConfirmAction(page)
      }
      return true
    }

    case 'replay_conversion': {
      await page.bringToFront().catch(() => null)
      await creator.send('test_apply_conversion', {
        gameID,
        playerID: action.playerId,
        conversionType: String(params.conversionType ?? ''),
        amount: Math.max(1, readInt(params, 'amount')),
      })

      const response = await creator.waitForAnyType(['test_command_applied', 'action_rejected'], 10_000)
      if (String(response.type ?? '') === 'action_rejected') {
        const payload = JSON.stringify(response.payload ?? {})
        throw new Error(`replay conversion was rejected: ${payload}`)
      }
      await maybeConfirmAction(page)
      return true
    }

    case 'burn_power': {
      const amount = Math.max(1, readInt(params, 'amount'))
      for (let i = 0; i < amount; i++) {
        await clickByTestId(page, `player-${action.playerId}-burn-power-1`)
        await maybeConfirmAction(page)
      }
      return true
    }

    case 'advance_shipping': {
      await clickByTestId(page, `player-${action.playerId}-upgrade-shipping`)
      await maybeConfirmAction(page)
      return true
    }

    case 'advance_digging': {
      await clickByTestId(page, `player-${action.playerId}-upgrade-digging`)
      await maybeConfirmAction(page)
      return true
    }

    case 'power_action_claim': {
      const actionType = readInt(params, 'actionType')
      await clickByTestId(page, `power-action-${String(actionType)}`)

      if (actionType === 0) {
        const from = readHex(params, 'bridgeHex1')
        const to = readHex(params, 'bridgeHex2')
        await clickHex(page, from.q, from.r)
        await clickHex(page, to.q, to.r)
        await maybeConfirmAction(page)
        return true
      }

      if (actionType === 4 || actionType === 5) {
        const hex = readHex(params, 'targetHex', 'hex')
        const buildDwelling = readBool(params, 'buildDwelling', true)
        const hasTargetTerrain = params.targetTerrain !== undefined
        const targetTerrain = hasTargetTerrain ? readInt(params, 'targetTerrain') : undefined
        await clickHex(page, hex.q, hex.r)
        await submitHexModal(page, buildDwelling, targetTerrain)
        return true
      }

      await maybeConfirmAction(page)
      return true
    }

    case 'engineers_bridge': {
      const from = readHex(params, 'bridgeHex1', 'fromHex')
      const to = readHex(params, 'bridgeHex2', 'toHex')
      await clickByTestId(page, `player-${action.playerId}-engineers-bridge`)
      await clickHex(page, from.q, from.r)
      await clickHex(page, to.q, to.r)
      await maybeConfirmAction(page)
      return true
    }

    case 'special_action_use': {
      const actionType = readInt(params, 'actionType', 'specialActionType')
      const cultTrack = params.cultTrack !== undefined ? readInt(params, 'cultTrack') : undefined
      const hasTargetHex = params.targetHex !== undefined || params.hex !== undefined
      const targetHex = hasTargetHex ? readHex(params, 'targetHex', 'hex') : undefined
      const buildDwelling = readBool(params, 'buildDwelling', true)
      const hasTargetTerrain = params.targetTerrain !== undefined
      const targetTerrain = hasTargetTerrain ? readInt(params, 'targetTerrain') : undefined

      if (actionType === 9) {
        await clickByTestId(page, 'passing-card-7')
        if (cultTrack === undefined) throw new Error('missing cultTrack for bonus cult action')
        await chooseCultTrackWithFallback(page, cultTrack, async () => {
          await page.getByTestId('passing-card-7').first().evaluate((el) => {
            (el as HTMLElement).click()
          })
        })
        return true
      }

      if (actionType === 7) {
        await clickByTestId(page, `player-${action.playerId}-water2-action`)
        if (cultTrack === undefined) throw new Error('missing cultTrack for water2 action')
        await chooseCultTrackWithFallback(page, cultTrack, async () => {
          await page.getByTestId(`player-${action.playerId}-water2-action`).first().evaluate((el) => {
            (el as HTMLElement).click()
          })
        })
        return true
      }

      if (actionType === 8) {
        await clickByTestId(page, 'passing-card-4')
        if (!targetHex) throw new Error('missing target hex for bonus spade action')
        await clickHex(page, targetHex.q, targetHex.r)
        await submitHexModal(page, buildDwelling, targetTerrain)
        return true
      }

      if (actionType === 1) {
        await clickByTestId(page, `player-${action.playerId}-stronghold-action`)
        if (!targetHex) throw new Error('missing target hex for witches ride')
        await clickHex(page, targetHex.q, targetHex.r)
        await maybeConfirmAction(page)
        return true
      }

      if (actionType === 5) {
        await clickByTestId(page, `player-${action.playerId}-stronghold-action`)
        if (!targetHex) throw new Error('missing target hex for giants special action')
        await clickHex(page, targetHex.q, targetHex.r)
        await submitHexModal(page, buildDwelling, targetTerrain)
        return true
      }

      throw new Error(`unsupported special_action_use actionType=${String(actionType)}`)
    }

    case 'select_favor_tile': {
      const tileType = readInt(params, 'tileType')
      await clickByTestId(page, `favor-tile-${String(tileType)}`)
      await maybeConfirmAction(page)
      return true
    }

    case 'select_town_tile': {
      const tileType = readInt(params, 'tileType')
      await clickByTestId(page, `town-tile-${String(tileType)}`)
      await maybeConfirmAction(page)
      return true
    }

    case 'select_cultists_track': {
      const track = readInt(params, 'track', 'cultTrack')
      await chooseCultistsTrack(page, track)
      return true
    }

    case 'pass': {
      const bonusCard = params.bonusCard !== undefined ? readInt(params, 'bonusCard') : null
      if (bonusCard !== null) {
        await clickByTestId(page, `passing-card-${String(bonusCard)}`)
      } else {
        const passNoCard = page.getByTestId('pass-without-card').first()
        const noCardVisible = await passNoCard.isVisible().catch(() => false)
        if (noCardVisible) {
          await passNoCard.click()
        } else {
          const cardButtons = page.locator('[data-testid^="passing-card-"]')
          const count = await cardButtons.count()
          let clicked = false
          for (let i = 0; i < count; i++) {
            const button = cardButtons.nth(i)
            const enabled = await button.isEnabled().catch(() => false)
            const visible = await button.isVisible().catch(() => false)
            if (enabled && visible) {
              await button.click()
              clicked = true
              break
            }
          }
          if (!clicked) {
            throw new Error('pass action requires card selection but no enabled passing card was found')
          }
        }
      }
      await maybeConfirmAction(page)
      return true
    }

    case 'use_cult_spade': {
      const hex = readHex(params, 'targetHex', 'hex')
      await clickHex(page, hex.q, hex.r)
      await submitHexModal(page, false)
      return true
    }

    default:
      throw new Error(`unsupported action type in click-driven runner: ${action.type}`)
  }
}

test.describe('Golden Full-Game Click-Driven Completion (Segmented)', () => {
  test.setTimeout(2_400_000)

  for (const scenario of GOLDEN_SCENARIOS) {
    const scriptExists = fs.existsSync(scenario.scriptPath)
    const tags = scenario.mode === 'smoke' ? '@smoke @nightly' : '@nightly'
    test(`${tags} replays ${scenario.fixtureLabel} through UI clicks in segments and reaches expected scores`, async ({ browser }) => {
      test.skip(!scriptExists, `missing golden script fixture at ${scenario.scriptPath}`)

      const goldenScript = JSON.parse(fs.readFileSync(scenario.scriptPath, 'utf8')) as GoldenScript
      goldenScript.expectedFinalScores = scenario.expectedScores

    const wsURL = 'ws://127.0.0.1:8080/api/ws'
    const segments = buildSegments(goldenScript.actions.length)
    expect(segments.length).toBeGreaterThan(0)

    for (let segmentIndex = 0; segmentIndex < segments.length; segmentIndex++) {
      const segment = segments[segmentIndex]
      console.log(`[click-golden-segment] ${String(segmentIndex + 1)}/${String(segments.length)} actions=${segment.start}-${String(segment.endExclusive - 1)}`)

      const { creator, gameID, revision: initialRevision } = await createConfiguredGame(
        wsURL,
        goldenScript,
        `${scenario.id}-${String(segmentIndex).padStart(2, '0')}-${segment.start}-${segment.endExclusive}`,
      )

      const viewerPages: Page[] = []
      const pageByPlayer = new Map<string, Page>()

      try {
        let revision = initialRevision

        if (segment.start > 0) {
          creator.send('test_replay_actions_to_index', {
            gameID,
            endExclusive: segment.start,
            actions: goldenScript.actions,
          })
          const replayAck = await creator.waitForAnyType(['test_command_applied', 'action_rejected'], 30_000)
          if (String(replayAck.type ?? '') === 'action_rejected') {
            const payload = JSON.stringify(replayAck.payload ?? {})
            throw new Error(`test_replay_actions_to_index rejected for segment=${segmentIndex}: ${payload}`)
          }
          const replayState = await creator.waitForType('game_state_update', 30_000)
          const statePayload = (replayState.payload ?? {}) as JsonObject
          if (String(statePayload.id ?? '') !== gameID) {
            const synced = await creator.waitForRevision(gameID, revision + 1, 30_000)
            revision = Number(synced.revision ?? revision + 1)
          } else {
            revision = Number(statePayload.revision ?? revision)
          }
        }

        const playerSet = new Set<string>()
        for (let i = segment.start; i < segment.endExclusive; i++) {
          playerSet.add(goldenScript.actions[i].playerId)
        }

        for (const playerId of playerSet.values()) {
          const page = await openPlayerViewer(browser, gameID, playerId)
          viewerPages.push(page)
          pageByPlayer.set(playerId, page)
        }

        for (let index = segment.start; index < segment.endExclusive; index++) {
          const action = goldenScript.actions[index]
          const page = pageByPlayer.get(action.playerId)
          if (!page) {
            throw new Error(`missing viewer for player ${action.playerId}`)
          }

          console.log(`[click-golden-segment] step=${String(index).padStart(4, '0')} player=${action.playerId} type=${action.type}`)
          const advanced = await clickAction(page, creator, gameID, action, index)
          if (!advanced) {
            continue
          }

          const state = await creator.waitForRevision(gameID, revision + 1, 25_000)
          revision = Number(state.revision ?? revision + 1)
        }

        if (segment.endExclusive === goldenScript.actions.length) {
          const finalState = await creator.waitForRevision(gameID, revision, 30_000)
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

          for (const page of viewerPages) {
            for (const expected of Object.values(goldenScript.expectedFinalScores)) {
              await expect(page.getByTestId('player-summary-bar')).toContainText(`${String(expected)} VP`)
            }
          }
        }
      } finally {
        for (const page of viewerPages) {
          await page.context().close()
        }
        creator.close()
      }
    }
    })
  }
})
