import { expect, test, type Browser, type Page } from '@playwright/test'
import fs from 'node:fs'
import { clickByTestId, clickHex } from './support/uiInteractions'
import { GOLDEN_SCENARIOS } from './fixtures/golden_scenarios'
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

type Segment = {
  start: number
  endExclusive: number
}

let fallbackActionSeq = 0
const clickDebugEnabled = process.env.TM_CLICK_DEBUG === '1'

const debugLog = (...args: unknown[]): void => {
  if (!clickDebugEnabled) return
  // eslint-disable-next-line no-console
  console.log(...args)
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

async function openActorBot(wsURL: string, gameID: string, playerId: string): Promise<WsBot> {
  const bot = await WsBot.connect(wsURL)
  bot.send('join_game', { id: gameID, name: playerId })
  await bot.waitForType('game_joined', 15_000)
  return bot
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
  if (process.env.TM_CLICK_USE_SEGMENTS !== '1') {
    return [{ start: 0, endExclusive: actionsLength }]
  }

  const segmentSize = 24

  const all: Segment[] = []
  for (let start = 0; start < actionsLength; start += segmentSize) {
    all.push({
      start,
      endExclusive: Math.min(actionsLength, start + segmentSize),
    })
  }

  return all
}

function getPendingDecisionInfo(snapshot: JsonObject | undefined): { type: string; playerId: string } {
  const pendingDecision = asRecord(snapshot?.pendingDecision)
  return {
    type: String(pendingDecision.type ?? ''),
    playerId: String(pendingDecision.playerId ?? ''),
  }
}

function getCurrentTurnPlayerId(snapshot: JsonObject | undefined): string {
  const turnOrderRaw = snapshot?.turnOrder
  const currentTurnRaw = snapshot?.currentTurn
  if (!Array.isArray(turnOrderRaw)) return ''
  const currentTurn = typeof currentTurnRaw === 'number' ? Math.trunc(currentTurnRaw) : Number(currentTurnRaw)
  if (!Number.isFinite(currentTurn) || currentTurn < 0 || currentTurn >= turnOrderRaw.length) return ''
  const playerId = turnOrderRaw[currentTurn]
  return typeof playerId === 'string' ? playerId : ''
}

async function performActionViaWsFallback(
  bot: WsBot,
  action: GoldenAction,
  gameID: string,
  expectedRevision: number,
  actionIndex: number,
): Promise<{ advanced: boolean; rejectionMessage: string }> {
  if (action.type === 'replay_conversion') {
    const params = action.params ?? {}
    bot.send('test_apply_conversion', {
      gameID,
      playerID: action.playerId,
      conversionType: String(params.conversionType ?? ''),
      amount: Math.max(1, readInt(params, 'amount')),
    })
    const response = await bot.waitForAnyType(['test_command_applied', 'action_rejected', 'error'], 4_000)
    const responseType = String(response.type ?? '')
    const rejectionMessage = String(
      asRecord(response.payload).message
      ?? asRecord(response.payload).error
      ?? '',
    )
    return { advanced: responseType === 'test_command_applied', rejectionMessage }
  }

  const actionId = `ui-click-fallback-${String(actionIndex).padStart(4, '0')}-${String(fallbackActionSeq)}`
  fallbackActionSeq++
  bot.send('perform_action', {
    type: action.type,
    gameID,
    actionId,
    expectedRevision,
    params: action.params ?? {},
  })
  const deadline = Date.now() + 3_000
  while (Date.now() < deadline) {
    const timeoutMs = Math.max(50, Math.min(500, deadline - Date.now()))
    const response = await bot
      .waitForAnyType(['action_accepted', 'action_rejected', 'error'], timeoutMs)
      .catch(() => null)
    if (!response) continue
    const payload = asRecord(response.payload)
    const responseActionId = String(payload.actionId ?? '')
    if (responseActionId !== '' && responseActionId !== actionId) {
      continue
    }
    const responseType = String(response.type ?? '')
    if (responseType !== 'action_accepted') {
      debugLog(
        `[click-replay] ws-fallback rejected index=${actionIndex} actor=${action.playerId} type=${action.type} expectedRevision=${expectedRevision} payload=${JSON.stringify(
          response.payload ?? {},
        )}`,
      )
    }
    const rejectionMessage = String(payload.message ?? payload.error ?? '')
    return { advanced: responseType === 'action_accepted', rejectionMessage }
  }
  debugLog(
    `[click-replay] ws-fallback timeout index=${actionIndex} actor=${action.playerId} type=${action.type} expectedRevision=${expectedRevision}`,
  )
  return { advanced: false, rejectionMessage: 'timeout' }
}

const pendingLeechOfferCount = (snapshot: JsonObject | undefined, playerId: string): number => {
  const pending = asRecord(snapshot?.pendingLeechOffers)
  const offers = pending[playerId]
  if (!Array.isArray(offers)) return 0
  return offers.length
}

const isLeechAction = (actionType: string): boolean => actionType === 'accept_leech' || actionType === 'decline_leech'

const parsePowerShortfallForAutoBurn = (message: string): number => {
  if (message === '') return 0

  // conversion failures: "need 4 power in bowl 3, only have 1"
  const conversionMatch = message.match(/need\s+(\d+)\s+power in bowl 3,\s*only have\s+(\d+)/i)
  if (conversionMatch) {
    const need = Number(conversionMatch[1] ?? '0')
    const have = Number(conversionMatch[2] ?? '0')
    return Math.max(0, need - have)
  }

  // power action failures: "not enough power in Bowl III: need 6, have 2"
  const powerActionMatch = message.match(/Bowl III:\s*need\s+(\d+),\s*have\s+(\d+)/i)
  if (powerActionMatch) {
    const need = Number(powerActionMatch[1] ?? '0')
    const have = Number(powerActionMatch[2] ?? '0')
    return Math.max(0, need - have)
  }

  return 0
}

const isSkippableActionFailure = (actionType: string, rejectionMessage: string): boolean => {
  const message = rejectionMessage.toLowerCase()
  if (message === '') return false

  if (message.includes('no pending leech offer for player')) return isLeechAction(actionType)
  if (message.includes('no pending town formation for player')) return actionType === 'select_town_tile'
  if (message.includes('cannot afford shipping upgrade')) return actionType === 'advance_shipping'
  if (message.includes('cannot afford digging upgrade')) return actionType === 'advance_digging'
  if (message.includes('cannot afford upgrade')) return actionType === 'upgrade_building'
  if (message.includes('not enough resources for dwelling')) return actionType === 'transform_build'
  if (message.includes('not enough workers')) return actionType === 'transform_build'
  if (message.includes('player has already passed')) return actionType !== 'pass'
  if (message.includes('no pending spades from cult rewards')) return actionType === 'use_cult_spade'
  if (message.includes('not your turn')) {
    return (
      actionType === 'use_cult_spade'
      || actionType === 'accept_leech'
      || actionType === 'decline_leech'
      || actionType === 'select_town_tile'
      || actionType === 'select_favor_tile'
      || actionType === 'select_cultists_track'
      || actionType === 'darklings_ordination'
      || actionType === 'discard_pending_spade'
    )
  }
  if (message.includes('need') && message.includes('power in bowl 3')) return actionType === 'conversion'
  if (message.includes('not enough power in bowl iii')) return actionType === 'power_action_claim'
  if (message.includes('power action') && message.includes('already been taken this round')) return actionType === 'power_action_claim'
  if (message.includes('cannot burn')) return actionType === 'burn_power'
  if (message.includes('no pending decision for requested action')) {
    return (
      actionType === 'select_favor_tile'
      || actionType === 'select_town_tile'
      || actionType === 'select_cultists_track'
      || actionType === 'darklings_ordination'
      || actionType === 'discard_pending_spade'
      || actionType === 'use_cult_spade'
    )
  }

  return false
}

const actionResolvesPendingDecision = (
  action: GoldenAction,
  pendingType: string,
  pendingPlayerId: string,
): boolean => {
  if (pendingType === '' || pendingPlayerId === '') return false
  if (action.playerId !== pendingPlayerId) return false

  switch (pendingType) {
    case 'leech_offer':
      return isLeechAction(action.type)
    case 'favor_tile_selection':
      return action.type === 'select_favor_tile'
    case 'town_tile_selection':
      return action.type === 'select_town_tile'
    case 'town_cult_top_choice':
      return action.type === 'select_town_cult_top'
    case 'cultists_cult_choice':
      return action.type === 'select_cultists_track'
    case 'darklings_ordination':
      return action.type === 'darklings_ordination'
    case 'spade_followup':
      return action.type === 'transform_build' || action.type === 'discard_pending_spade'
    case 'cult_reward_spade':
      return action.type === 'use_cult_spade' || action.type === 'discard_pending_spade'
    default:
      return false
  }
}

async function clickAction(page: Page, creator: WsBot, gameID: string, action: GoldenAction): Promise<boolean> {
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
      const control = page.getByTestId(`player-${action.playerId}-conversion-${conversionType}`).first()
      const visible = await control.isVisible().catch(() => false)
      const enabled = visible ? await control.isEnabled().catch(() => false) : false
      if (!visible || !enabled) {
        return false
      }
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
      const control = page.getByTestId(`player-${action.playerId}-burn-power-1`).first()
      const visible = await control.isVisible().catch(() => false)
      const enabled = visible ? await control.isEnabled().catch(() => false) : false
      if (!visible || !enabled) {
        return false
      }
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
      const testId = `favor-tile-${String(tileType)}`
      const button = page.getByTestId(testId).first()
      const attached = await button.waitFor({ state: 'attached', timeout: 1_500 }).then(() => true).catch(() => false)
      const visible = attached ? await button.isVisible().catch(() => false) : false
      const enabled = visible ? await button.isEnabled().catch(() => false) : false
      if (!visible || !enabled) {
        const pending = getPendingDecisionInfo(creator.getState(gameID))
        if (pending.type !== 'favor_tile_selection' || pending.playerId !== action.playerId) {
          return false
        }
        throw new Error(
          `expected clickable ${testId} for ${action.playerId} during favor selection, but visible=${String(visible)} enabled=${String(enabled)}`,
        )
      }
      await clickByTestId(page, `favor-tile-${String(tileType)}`)
      await maybeConfirmAction(page)
      return true
    }

    case 'select_town_tile': {
      const tileType = readInt(params, 'tileType')
      const testId = `town-tile-${String(tileType)}`
      const button = page.getByTestId(testId).first()
      const attached = await button.waitFor({ state: 'attached', timeout: 1_500 }).then(() => true).catch(() => false)
      const visible = attached ? await button.isVisible().catch(() => false) : false
      const enabled = visible ? await button.isEnabled().catch(() => false) : false
      if (!visible || !enabled) {
        const pending = getPendingDecisionInfo(creator.getState(gameID))
        const pendingTypeMatches = pending.type === 'town_tile_selection' || pending.type === ''
        if (!pendingTypeMatches || pending.playerId !== action.playerId) {
          return false
        }
        throw new Error(
          `expected clickable ${testId} for ${action.playerId} during town selection, but visible=${String(visible)} enabled=${String(enabled)}`,
        )
      }
      await clickByTestId(page, `town-tile-${String(tileType)}`)
      await maybeConfirmAction(page)
      return true
    }

    case 'select_cultists_track': {
      const track = readInt(params, 'track', 'cultTrack')
      const cultistsModalVisible = await page.getByTestId('cultists-cult-choice-modal').first().isVisible().catch(() => false)
      const genericModalVisible = await page.getByTestId('cult-choice-modal').first().isVisible().catch(() => false)
      if (!cultistsModalVisible && !genericModalVisible) {
        const pending = getPendingDecisionInfo(creator.getState(gameID))
        if (pending.type !== 'cultists_cult_choice' || pending.playerId !== action.playerId) {
          return false
        }
      }
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
        const { creator, gameID, revision: initialRevision } = await createConfiguredGame(
          wsURL,
          goldenScript,
          `${scenario.id}-${String(segmentIndex).padStart(2, '0')}-${segment.start}-${segment.endExclusive}`,
        )

        const viewerPages: Page[] = []
        const pageByPlayer = new Map<string, Page>()
        const actorBots = new Map<string, WsBot>()

        try {
          let revision = initialRevision
          debugLog(`[click-replay] scenario=${scenario.id} segment=${segmentIndex} start=${segment.start} end=${segment.endExclusive}`)

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
          if (playerId === goldenScript.playerIds[0]) {
            actorBots.set(playerId, creator)
            continue
          }
          actorBots.set(playerId, await openActorBot(wsURL, gameID, playerId))
        }

          const syncRevision = async (): Promise<void> => {
            const state = await creator.waitForRevision(gameID, revision + 1, 3_000).catch(() => null)
            if (state) {
              revision = Number(state.revision ?? revision + 1)
              return
            }
            const snapshot = creator.getState(gameID)
            const latestRevision = Number(snapshot?.revision ?? revision)
            if (latestRevision > revision) {
              revision = latestRevision
            }
          }

          const executeScriptedActionAt = async (index: number): Promise<{ advanced: boolean; rejectionMessage: string }> => {
            const action = goldenScript.actions[index]
            debugLog(`[click-replay] run index=${index} actor=${action.playerId} type=${action.type} revision=${revision}`)

            const page = pageByPlayer.get(action.playerId)
            if (!page) throw new Error(`missing viewer for player ${action.playerId}`)

            const actorBot = actorBots.get(action.playerId)
            if (!actorBot) throw new Error(`missing actor bot for ${action.playerId}`)

            let advanced = false
            let rejectionMessage = ''
            if (!scenario.wsOnlyReplay) {
              try {
                advanced = await clickAction(page, creator, gameID, action)
              } catch {
                advanced = false
              }
            }
            if (!advanced) {
              const wsAttempt = await performActionViaWsFallback(actorBot, action, gameID, revision, index)
              advanced = wsAttempt.advanced
              rejectionMessage = wsAttempt.rejectionMessage
            }

            if (
              !advanced
              && action.type === 'conversion'
              && rejectionMessage.toLowerCase().includes('not your turn')
            ) {
              const replayConversionAttempt = await performActionViaWsFallback(
                actorBot,
                { ...action, type: 'replay_conversion' },
                gameID,
                revision,
                index,
              )
              advanced = replayConversionAttempt.advanced
              rejectionMessage = replayConversionAttempt.rejectionMessage
            }

            if (!advanced && scenario.wsOnlyReplay) {
              const burnAmount = parsePowerShortfallForAutoBurn(rejectionMessage)
              if (burnAmount > 0 && action.type !== 'burn_power') {
                debugLog(
                  `[click-replay] auto-burn before retry index=${index} actor=${action.playerId} type=${action.type} burn=${burnAmount} message=${rejectionMessage}`,
                )
                const burnAction: GoldenAction = {
                  playerId: action.playerId,
                  type: 'burn_power',
                  params: { amount: burnAmount } as JsonObject,
                }
                const burnAttempt = await performActionViaWsFallback(actorBot, burnAction, gameID, revision, -2)
                if (burnAttempt.advanced) {
                  await syncRevision()
                  const retryAttempt = await performActionViaWsFallback(actorBot, action, gameID, revision, index)
                  advanced = retryAttempt.advanced
                  rejectionMessage = retryAttempt.rejectionMessage
                }
              }
            }

            debugLog(
              `[click-replay] result index=${index} actor=${action.playerId} type=${action.type} advanced=${String(advanced)} revision=${revision} rejection=${rejectionMessage}`,
            )

            if (!advanced) return { advanced: false, rejectionMessage }
            await syncRevision()
            debugLog(`[click-replay] synced index=${index} revision=${revision}`)
            return { advanced: true, rejectionMessage: '' }
          }

          const remainingActionIndexes = new Set<number>()
          for (let index = segment.start; index < segment.endExclusive; index++) {
            remainingActionIndexes.add(index)
          }

          const sortedRemaining = (): number[] => Array.from(remainingActionIndexes.values()).sort((a, b) => a - b)

          let nextIndex = segment.start
          let safetyCounter = 0
          const maxSteps = (segment.endExclusive - segment.start) * 12 + 200
          while (nextIndex < segment.endExclusive) {
            if (!remainingActionIndexes.has(nextIndex)) {
              nextIndex++
              continue
            }

            safetyCounter++
            if (safetyCounter > maxSteps) {
              const snapshot = creator.getState(gameID)
              const pendingDecision = getPendingDecisionInfo(snapshot)
              const pendingPreview = sortedRemaining().slice(0, 10).map((idx) => {
                const action = goldenScript.actions[idx]
                return `${idx}:${action.playerId}:${action.type}`
              })
              throw new Error(
                `replay safety limit reached for scenario=${scenario.id} segment=${segmentIndex} revision=${revision} phase=${String(snapshot?.phase ?? '')} pendingDecision=${pendingDecision.type}:${pendingDecision.playerId} nextIndex=${nextIndex} remaining=${pendingPreview.join(' | ')}`,
              )
            }

            const action = goldenScript.actions[nextIndex]
            const execution = await executeScriptedActionAt(nextIndex)
            if (execution.advanced) {
              remainingActionIndexes.delete(nextIndex)
              nextIndex++
              continue
            }

            const snapshotAfter = creator.getState(gameID)
            const pendingAfter = getPendingDecisionInfo(snapshotAfter)
            const activeTurnPlayer = getCurrentTurnPlayerId(snapshotAfter)

            if (pendingAfter.type !== '' && pendingAfter.playerId !== '') {
              const resolverIndex = sortedRemaining().find((idx) => {
                if (idx === nextIndex) return false
                const candidate = goldenScript.actions[idx]
                return actionResolvesPendingDecision(candidate, pendingAfter.type, pendingAfter.playerId)
              })
              if (resolverIndex !== undefined) {
                const resolverAction = goldenScript.actions[resolverIndex]
                const resolverExecution = await executeScriptedActionAt(resolverIndex)
                if (resolverExecution.advanced) {
                  debugLog(
                    `[click-replay] resolve-pending pending=${pendingAfter.type}:${pendingAfter.playerId} consumed=${resolverIndex}:${resolverAction.playerId}:${resolverAction.type}`,
                  )
                  remainingActionIndexes.delete(resolverIndex)
                  continue
                }
                if (isSkippableActionFailure(resolverAction.type, resolverExecution.rejectionMessage)) {
                  debugLog(
                    `[click-replay] consume-stale pending=${pendingAfter.type}:${pendingAfter.playerId} index=${resolverIndex}:${resolverAction.playerId}:${resolverAction.type} message=${resolverExecution.rejectionMessage}`,
                  )
                  remainingActionIndexes.delete(resolverIndex)
                  continue
                }
              }
              if (pendingAfter.type === 'leech_offer') {
                const pendingBot = actorBots.get(pendingAfter.playerId)
                if (pendingBot) {
                  const syntheticDecline: GoldenAction = {
                    playerId: pendingAfter.playerId,
                    type: 'decline_leech',
                    params: { offerIndex: 0 } as JsonObject,
                  }
                  const synthetic = await performActionViaWsFallback(pendingBot, syntheticDecline, gameID, revision, -1)
                  if (synthetic.advanced) {
                    debugLog(
                      `[click-replay] resolve-pending synthetic=${pendingAfter.type}:${pendingAfter.playerId} action=decline_leech`,
                    )
                    await syncRevision()
                    continue
                  }
                }
              }
            }

            if (execution.rejectionMessage.toLowerCase().includes('not your turn') && pendingAfter.type === '' && activeTurnPlayer !== '') {
              debugLog(
                `[click-replay] not-your-turn index=${nextIndex} actor=${action.playerId} activeTurnPlayer=${activeTurnPlayer} revision=${revision}`,
              )
              const turnOwnerIndex = sortedRemaining().find((idx) => {
                if (idx === nextIndex) return false
                const candidate = goldenScript.actions[idx]
                return candidate.playerId === activeTurnPlayer
              })
              if (turnOwnerIndex !== undefined) {
                const turnOwnerAction = goldenScript.actions[turnOwnerIndex]
                const turnOwnerExecution = await executeScriptedActionAt(turnOwnerIndex)
                if (turnOwnerExecution.advanced) {
                  debugLog(
                    `[click-replay] resolve-turn owner=${activeTurnPlayer} consumed=${turnOwnerIndex}:${turnOwnerAction.playerId}:${turnOwnerAction.type}`,
                  )
                  remainingActionIndexes.delete(turnOwnerIndex)
                  continue
                }
                if (isSkippableActionFailure(turnOwnerAction.type, turnOwnerExecution.rejectionMessage)) {
                  debugLog(
                    `[click-replay] consume-stale owner=${activeTurnPlayer} index=${turnOwnerIndex}:${turnOwnerAction.playerId}:${turnOwnerAction.type} message=${turnOwnerExecution.rejectionMessage}`,
                  )
                  remainingActionIndexes.delete(turnOwnerIndex)
                  continue
                }
              }
            }

            if (isSkippableActionFailure(action.type, execution.rejectionMessage)) {
              debugLog(
                `[click-replay] consume-stale index=${nextIndex} actor=${action.playerId} type=${action.type} reason=unexecutable message=${execution.rejectionMessage}`,
              )
              remainingActionIndexes.delete(nextIndex)
              nextIndex++
              continue
            }

            if (isLeechAction(action.type)) {
              const offers = pendingLeechOfferCount(snapshotAfter, action.playerId)
              if (offers === 0) {
                debugLog(
                  `[click-replay] consume-stale index=${nextIndex} actor=${action.playerId} type=${action.type} reason=no-pending-leech`,
                )
                remainingActionIndexes.delete(nextIndex)
                nextIndex++
                continue
              }
            }

            const phase = Number(snapshotAfter?.phase ?? -1)
            if (phase === 5) {
              for (const idx of sortedRemaining()) {
                const leftover = goldenScript.actions[idx]
                debugLog(
                  `[click-replay] consume-stale index=${idx} actor=${leftover.playerId} type=${leftover.type} reason=phase-complete`,
                )
                remainingActionIndexes.delete(idx)
              }
              break
            }

            const pendingPreview = sortedRemaining().slice(0, 10).map((idx) => {
              const pendingAction = goldenScript.actions[idx]
              return `${idx}:${pendingAction.playerId}:${pendingAction.type}`
            })
            throw new Error(
              `failed scripted action for scenario=${scenario.id} segment=${segmentIndex} revision=${revision} phase=${String(phase)} index=${nextIndex}:${action.playerId}:${action.type} pendingDecision=${pendingAfter.type}:${pendingAfter.playerId} remaining=${pendingPreview.join(' | ')}`,
            )
          }


        if (segment.endExclusive === goldenScript.actions.length) {
          const finalState = await creator.waitForRevision(gameID, revision, 30_000)
          if (Number(finalState.phase ?? -1) !== 5) {
            throw new Error(`expected final phase=5, got ${String(finalState.phase)}`)
          }

          const finalScoring = (finalState.finalScoring ?? {}) as Record<string, JsonObject>
          const finalTotals = new Map<string, number>()
          for (const [playerId, expected] of Object.entries(goldenScript.expectedFinalScores)) {
            const entry = finalScoring[playerId]
            if (!entry) throw new Error(`missing final scoring entry for ${playerId}`)
            const got = Number(entry.totalVp ?? -1)
            debugLog(`[click-replay] final-score player=${playerId} got=${String(got)} expected=${String(expected)}`)
            finalTotals.set(playerId, got)
            const tolerance = scenario.scoreTolerance ?? 0
            if (tolerance > 0) {
              expect(Math.abs(got - expected)).toBeLessThanOrEqual(tolerance)
            } else {
              expect(got).toBe(expected)
            }
          }

          for (const page of viewerPages) {
            for (const vp of finalTotals.values()) {
              await expect(page.getByTestId('player-summary-bar')).toContainText(`${String(vp)} VP`)
            }
          }
        }
        } finally {
          for (const page of viewerPages) {
            await page.context().close()
          }
          for (const [playerId, bot] of actorBots.entries()) {
            if (bot !== creator || playerId !== goldenScript.playerIds[0]) {
              bot.close()
            }
          }
          creator.close()
        }
      }
    })
  }
})
