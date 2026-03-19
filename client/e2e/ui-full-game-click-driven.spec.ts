import { expect, test, type Page } from '@playwright/test'
import fs from 'node:fs'
import { clickByTestId, clickHex } from './support/uiInteractions'
import { GOLDEN_SCENARIOS } from './fixtures/golden_scenarios'
import { realServerWsURL } from './support/realServerConfig'
import { loadRealServerGamePage, sendPageSocketMessage, setRealServerPageLocalPlayer } from './support/realServerPage'
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

async function waitForPageToMatchSnapshot(
  page: Page,
  snapshot: JsonObject | undefined,
  gameID: string,
  localPlayerId: string,
): Promise<void> {
  const snapshotRevision = Number(snapshot?.revision ?? -1)
  if (!Number.isFinite(snapshotRevision) || snapshotRevision < 0) return

  const readPageRevision = async (): Promise<number> => page.evaluate(() => {
    const testWindow = window as Window & {
      __TM_TEST_GET_REVISION__?: () => number | null
    }
    if (typeof testWindow.__TM_TEST_GET_REVISION__ !== 'function') return -1
    const revision = testWindow.__TM_TEST_GET_REVISION__()
    return typeof revision === 'number' ? revision : -1
  }).catch(() => -1)

  const pageRevision = await readPageRevision()
  if (pageRevision >= snapshotRevision) return

  await sendPageSocketMessage(page, {
    type: 'get_game_state',
    payload: { gameID, playerID: localPlayerId },
  })
  await page.waitForFunction((minRevision) => {
    const testWindow = window as Window & {
      __TM_TEST_GET_REVISION__?: () => number | null
    }
    if (typeof testWindow.__TM_TEST_GET_REVISION__ !== 'function') return false
    const revision = testWindow.__TM_TEST_GET_REVISION__()
    return typeof revision === 'number' && revision >= minRevision
  }, snapshotRevision, { timeout: 2_000 })
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
  await expect(page.getByTestId('hex-action-submit').first()).toBeVisible({ timeout: 10_000 })

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
  const initialVisible = await hasVisibleTestId(page, `cult-choice-${String(cultTrack)}`, 500)
  if (!initialVisible && retryOpen) {
    await retryOpen()
  }
  await expect(page.getByTestId(`cult-choice-${String(cultTrack)}`).first()).toBeVisible({ timeout: 10_000 })
  await clickByTestId(page, `cult-choice-${String(cultTrack)}`)
}

async function chooseCultistsTrack(page: Page, cultTrack: number): Promise<void> {
  const visible = await hasVisibleTestId(page, `cultists-cult-choice-${String(cultTrack)}`, 500)
  if (visible) {
    await clickByTestId(page, `cultists-cult-choice-${String(cultTrack)}`)
    return
  }
  await chooseCultTrackWithFallback(page, cultTrack)
}

async function switchActorPage(page: Page, gameID: string, playerId: string, currentPlayerId: string): Promise<string> {
  if (currentPlayerId === '') {
    await page.goto('/')
    await page.evaluate(({ localPlayerId }) => {
      localStorage.setItem('tm-game-storage', JSON.stringify({ state: { localPlayerId }, version: 0 }))
    }, { localPlayerId: playerId })
    await loadRealServerGamePage(page, gameID, playerId)
    return playerId
  }

  if (currentPlayerId !== playerId) {
    await setRealServerPageLocalPlayer(page, playerId)
    await sendPageSocketMessage(page, {
      type: 'join_game',
      payload: { id: gameID, name: playerId },
    })
    await sendPageSocketMessage(page, {
      type: 'get_game_state',
      payload: { gameID, playerID: playerId },
    })
    await page.waitForFunction((expectedPlayerId) => {
      const testWindow = window as Window & {
        __TM_TEST_GET_LOCAL_PLAYER_ID__?: () => string | null
      }
      return typeof testWindow.__TM_TEST_GET_LOCAL_PLAYER_ID__ === 'function'
        && testWindow.__TM_TEST_GET_LOCAL_PLAYER_ID__() === expectedPlayerId
    }, playerId, { timeout: 2_000 })
  }

  return playerId
}

async function confirmPendingTurnIfNeeded(
  page: Page,
  creator: WsBot,
  gameID: string,
  actorPagePlayerId: string,
  nextActionPlayerId: string,
  nextActionType: string,
  revisionRef: { current: number },
): Promise<string> {
  const snapshot = creator.getState(gameID)
  const pending = getPendingDecisionInfo(snapshot)
  if (pending.playerId === '') {
    return actorPagePlayerId
  }
  if (pending.type !== 'post_action_free_actions' && pending.type !== 'turn_confirmation') {
    return actorPagePlayerId
  }
  if (
    pending.type === 'post_action_free_actions'
    && pending.playerId === nextActionPlayerId
    && (nextActionType === 'conversion' || nextActionType === 'burn_power')
  ) {
    return actorPagePlayerId
  }

  const pendingPlayerId = pending.playerId
  const pagePlayerId = await switchActorPage(page, gameID, pendingPlayerId, actorPagePlayerId)
  await waitForPageToMatchSnapshot(page, snapshot, gameID, pendingPlayerId)
  await clickByTestId(page, 'turn-end-confirm')

  const state = await creator.waitForRevision(gameID, revisionRef.current + 1, 5_000)
  revisionRef.current = Number(state.revision ?? revisionRef.current + 1)
  return pagePlayerId
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

function describeActionSnapshot(action: GoldenAction, snapshot: JsonObject | undefined): string {
  const parts: string[] = []
  const activeTurnPlayer = getCurrentTurnPlayerId(snapshot)
  if (activeTurnPlayer !== '') {
    parts.push(`currentTurn=${activeTurnPlayer}`)
  }

  if (action.type === 'upgrade_building') {
    const hex = readHex(action.params ?? {}, 'targetHex', 'upgradeHex', 'hex')
    const map = asRecord(snapshot?.map)
    const hexes = asRecord(map.hexes)
    const targetHex = asRecord(hexes[`${String(hex.q)},${String(hex.r)}`])
    const building = asRecord(targetHex.building)
    const player = asRecord(asRecord(snapshot?.players)[action.playerId])
    const resources = asRecord(player.resources)
    parts.push(
      `targetHex=${String(hex.q)},${String(hex.r)}`,
      `buildingOwner=${String(building.ownerPlayerId ?? '')}`,
      `buildingType=${String(building.type ?? '')}`,
      `workers=${String(resources.workers ?? '')}`,
      `coins=${String(resources.coins ?? '')}`,
      `priests=${String(resources.priests ?? '')}`,
    )
  }

  return parts.join(' ')
}

function assertActionApplied(
  action: GoldenAction,
  snapshotBefore: JsonObject | undefined,
  snapshotAfter: JsonObject | undefined,
): void {
  if (action.type === 'upgrade_building') {
    const hex = readHex(action.params ?? {}, 'targetHex', 'upgradeHex', 'hex')
    const expectedBuildingType = readInt(action.params ?? {}, 'newBuildingType')
    const map = asRecord(snapshotAfter?.map)
    const hexes = asRecord(map.hexes)
    const targetHex = asRecord(hexes[`${String(hex.q)},${String(hex.r)}`])
    const building = asRecord(targetHex.building)
    const actualOwner = String(building.ownerPlayerId ?? '')
    const actualBuildingType = Number(building.type ?? Number.NaN)
    const beforeHexes = asRecord(asRecord(snapshotBefore?.map).hexes)
    const changedHexes: string[] = []
    for (const [key, rawAfterHex] of Object.entries(hexes)) {
      const afterBuilding = asRecord(asRecord(rawAfterHex).building)
      const beforeBuilding = asRecord(asRecord(beforeHexes[key]).building)
      const afterOwner = String(afterBuilding.ownerPlayerId ?? '')
      const beforeOwner = String(beforeBuilding.ownerPlayerId ?? '')
      const afterType = String(afterBuilding.type ?? '')
      const beforeType = String(beforeBuilding.type ?? '')
      if (afterOwner !== action.playerId && beforeOwner !== action.playerId) continue
      if (afterOwner === beforeOwner && afterType === beforeType) continue
      changedHexes.push(`${key}:${beforeOwner}/${beforeType}->${afterOwner}/${afterType}`)
    }
    if (actualOwner !== action.playerId || actualBuildingType !== expectedBuildingType) {
      throw new Error(
        `post-action state mismatch for upgrade_building targetHex=${String(hex.q)},${String(hex.r)} expectedOwner=${action.playerId} expectedType=${String(expectedBuildingType)} actualOwner=${actualOwner} actualType=${String(building.type ?? '')} changedHexes=${changedHexes.join(',')}`,
      )
    }
  }
}

const isLeechAction = (actionType: string): boolean => actionType === 'accept_leech' || actionType === 'decline_leech'

const getPendingSpadePlayers = (snapshot: JsonObject | undefined): string[] => {
  const players: string[] = []
  const pushPendingPlayers = (raw: unknown): void => {
    const pending = asRecord(raw)
    for (const [playerId, countRaw] of Object.entries(pending)) {
      const count = typeof countRaw === 'number' ? countRaw : Number(countRaw)
      if (!Number.isFinite(count) || count <= 0 || players.includes(playerId)) continue
      players.push(playerId)
    }
  }

  pushPendingPlayers(snapshot?.pendingCultRewardSpades)
  pushPendingPlayers(snapshot?.pendingSpades)
  return players
}

const hasPendingTownSelection = (snapshot: JsonObject | undefined, playerId: string): boolean => {
  const pendingTownFormations = asRecord(snapshot?.pendingTownFormations)
  const towns = pendingTownFormations[playerId]
  return Array.isArray(towns) && towns.length > 0
}

const actionResolvesPendingSpade = (action: GoldenAction, snapshot: JsonObject | undefined): boolean => {
  const pendingCultRewardSpades = asRecord(snapshot?.pendingCultRewardSpades)
  const pendingSpades = asRecord(snapshot?.pendingSpades)
  const cultRewardCount = Number(pendingCultRewardSpades[action.playerId] ?? 0)
  if (Number.isFinite(cultRewardCount) && cultRewardCount > 0) {
    return action.type === 'use_cult_spade' || action.type === 'discard_pending_spade'
  }

  const spadeCount = Number(pendingSpades[action.playerId] ?? 0)
  if (Number.isFinite(spadeCount) && spadeCount > 0) {
    return action.type === 'transform_build' || action.type === 'discard_pending_spade'
  }

  return false
}

const actionResolvesPendingTownSelection = (action: GoldenAction, snapshot: JsonObject | undefined): boolean => (
  action.type === 'select_town_tile' && hasPendingTownSelection(snapshot, action.playerId)
)

const actionHasAvailableLeechOffer = (action: GoldenAction, snapshot: JsonObject | undefined): boolean => {
  if (!isLeechAction(action.type)) return false
  const offerIndex = Number(action.params?.offerIndex ?? -1)
  if (!Number.isFinite(offerIndex) || offerIndex < 0) return false
  const pendingLeechOffers = asRecord(snapshot?.pendingLeechOffers)
  const offers = pendingLeechOffers[action.playerId]
  return Array.isArray(offers) && offerIndex < offers.length
}

const actionResolvesPendingDecision = (
  action: GoldenAction,
  pendingType: string,
  pendingPlayerId: string,
): boolean => {
  if (pendingType === '' || pendingPlayerId === '') return false
  if (action.playerId !== pendingPlayerId) return false

  switch (pendingType) {
    case 'setup_bonus_card':
      return action.type === 'setup_bonus_card'
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

const actionRequiresTurnOwnership = (action: GoldenAction, snapshot: JsonObject | undefined): boolean => {
  switch (action.type) {
    case 'accept_leech':
    case 'decline_leech':
    case 'select_favor_tile':
    case 'select_town_tile':
    case 'select_town_cult_top':
    case 'darklings_ordination':
    case 'select_cultists_track':
    case 'use_cult_spade':
    case 'discard_pending_spade':
      return false
    default:
      return true
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
      const attached = await control.waitFor({ state: 'attached', timeout: 1_500 }).then(() => true).catch(() => false)
      const visible = attached ? await control.isVisible().catch(() => false) : false
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
      throw new Error(`strict click replay does not allow replay-only action type=${action.type}`)
    }

    case 'burn_power': {
      const amount = Math.max(1, readInt(params, 'amount'))
      const control = page.getByTestId(`player-${action.playerId}-burn-power-1`).first()
      const attached = await control.waitFor({ state: 'attached', timeout: 1_500 }).then(() => true).catch(() => false)
      const visible = attached ? await control.isVisible().catch(() => false) : false
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
      const cultistsVisible = await hasVisibleTestId(page, `cultists-cult-choice-${String(track)}`, 500)
      const genericVisible = await hasVisibleTestId(page, `cult-choice-${String(track)}`, 500)
      if (!cultistsVisible && !genericVisible) {
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

test.describe('Golden Full-Game Click-Driven Completion', () => {
  test.setTimeout(2_400_000)

  for (const scenario of GOLDEN_SCENARIOS) {
    const scriptExists = fs.existsSync(scenario.scriptPath)
    const tags = scenario.mode === 'smoke' ? '@smoke @nightly' : '@nightly'
    test(`${tags} replays ${scenario.fixtureLabel} through UI clicks and reaches expected scores`, async ({ browser }) => {
      test.skip(!scriptExists, `missing golden script fixture at ${scenario.scriptPath}`)

      const goldenScript = JSON.parse(fs.readFileSync(scenario.scriptPath, 'utf8')) as GoldenScript
      goldenScript.expectedFinalScores = scenario.expectedScores

      const wsURL = realServerWsURL()
      const { creator, gameID, revision: initialRevision } = await createConfiguredGame(
        wsURL,
        goldenScript,
        `${scenario.id}-full`,
      )

      const actorContext = await browser.newContext()
      const actorPage = await actorContext.newPage()

      try {
        let revision = initialRevision
        let actorPagePlayerId = ''
        const revisionRef = { current: revision }
        debugLog(`[click-replay] scenario=${scenario.id} full-run actions=${goldenScript.actions.length}`)

        const playerSet = new Set<string>()
        for (const action of goldenScript.actions) {
          playerSet.add(action.playerId)
        }

        const syncRevision = async (): Promise<void> => {
          const state = await creator.waitForRevision(gameID, revision + 1, 3_000).catch(() => null)
          if (state) {
            revision = Number(state.revision ?? revision + 1)
            revisionRef.current = revision
            return
          }
          const snapshot = creator.getState(gameID)
          const latestRevision = Number(snapshot?.revision ?? revision)
          if (latestRevision > revision) {
            revision = latestRevision
            revisionRef.current = revision
          }
        }

        const executeScriptedActionAt = async (index: number): Promise<{ advanced: boolean; rejectionMessage: string }> => {
          const action = goldenScript.actions[index]
          actorPagePlayerId = await confirmPendingTurnIfNeeded(
            actorPage,
            creator,
            gameID,
            actorPagePlayerId,
            action.playerId,
            action.type,
            revisionRef,
          )
          revision = revisionRef.current
          const snapshotBeforeAction = creator.getState(gameID)
          debugLog(`[click-replay] run index=${index} actor=${action.playerId} type=${action.type} revision=${revision}`)

          actorPagePlayerId = await switchActorPage(actorPage, gameID, action.playerId, actorPagePlayerId)

          let advanced = false
          let rejectionMessage = ''
          const revisionBeforeAction = Number(snapshotBeforeAction?.revision ?? revision)
          try {
            await waitForPageToMatchSnapshot(actorPage, snapshotBeforeAction, gameID, action.playerId)
            advanced = await clickAction(actorPage, creator, gameID, action)
          } catch (err) {
            rejectionMessage = err instanceof Error ? err.message : String(err)
          }

          debugLog(
            `[click-replay] result index=${index} actor=${action.playerId} type=${action.type} advanced=${String(advanced)} revision=${revision} rejection=${rejectionMessage}`,
          )

          if (!advanced) return { advanced: false, rejectionMessage }
          await syncRevision()
          if (revision <= revisionBeforeAction) {
            const bannerText = await actorPage.getByTestId('action-error-message').textContent().catch(() => null)
            const detail = bannerText && bannerText.trim() !== '' ? bannerText.trim() : 'no revision change after UI action'
            return { advanced: false, rejectionMessage: detail }
          }
          try {
            assertActionApplied(action, snapshotBeforeAction, creator.getState(gameID))
          } catch (err) {
            rejectionMessage = err instanceof Error ? err.message : String(err)
            return { advanced: false, rejectionMessage }
          }
          debugLog(`[click-replay] synced index=${index} revision=${revision}`)
          return { advanced: true, rejectionMessage: '' }
        }

        const remainingActionIndexes = new Set<number>()
        for (let index = 0; index < goldenScript.actions.length; index++) {
          remainingActionIndexes.add(index)
        }

        const sortedRemaining = (): number[] => Array.from(remainingActionIndexes.values()).sort((a, b) => a - b)

        let safetyCounter = 0
        const maxSteps = goldenScript.actions.length * 12 + 200
        while (remainingActionIndexes.size > 0) {
          safetyCounter++
          if (safetyCounter > maxSteps) {
            const snapshot = creator.getState(gameID)
            const pendingDecision = getPendingDecisionInfo(snapshot)
            const pendingPreview = sortedRemaining().slice(0, 10).map((idx) => {
              const action = goldenScript.actions[idx]
              return `${idx}:${action.playerId}:${action.type}`
            })
            throw new Error(
              `replay safety limit reached for scenario=${scenario.id} revision=${revision} phase=${String(snapshot?.phase ?? '')} pendingDecision=${pendingDecision.type}:${pendingDecision.playerId} remaining=${pendingPreview.join(' | ')}`,
            )
          }

          const snapshotBefore = creator.getState(gameID)
          const pendingBefore = getPendingDecisionInfo(snapshotBefore)
          const pendingSpadePlayers = getPendingSpadePlayers(snapshotBefore)
          const activeTurnPlayer = getCurrentTurnPlayerId(snapshotBefore)
          const phase = Number(snapshotBefore?.phase ?? -1)
          if (phase === 5) {
            const pendingPreview = sortedRemaining().slice(0, 10).map((idx) => {
              const pendingAction = goldenScript.actions[idx]
              return `${idx}:${pendingAction.playerId}:${pendingAction.type}`
            })
            throw new Error(
              `game reached final phase with remaining scripted actions scenario=${scenario.id} revision=${revision} remaining=${pendingPreview.join(' | ')}`,
            )
          }

          const nextIndex = (() => {
            const remaining = sortedRemaining()
            if (pendingBefore.type !== '' && pendingBefore.playerId !== '') {
              if (pendingBefore.type !== 'post_action_free_actions' && pendingBefore.type !== 'turn_confirmation') {
                const resolverIndex = remaining.find((idx) => {
                  const candidate = goldenScript.actions[idx]
                  return actionResolvesPendingDecision(candidate, pendingBefore.type, pendingBefore.playerId)
                })
                if (resolverIndex === undefined) {
                  throw new Error(
                    `no scripted resolver for pending decision ${pendingBefore.type}:${pendingBefore.playerId} scenario=${scenario.id} revision=${revision}`,
                  )
                }
                return resolverIndex
              }
            }

            if (pendingSpadePlayers.length > 0) {
              const resolverIndex = remaining.find((idx) => actionResolvesPendingSpade(goldenScript.actions[idx], snapshotBefore))
              if (resolverIndex === undefined) {
                throw new Error(
                  `no scripted resolver for pending spades players=${pendingSpadePlayers.join(',')} scenario=${scenario.id} revision=${revision}`,
                )
              }
              return resolverIndex
            }

            const pendingTownIndex = remaining.find((idx) => actionResolvesPendingTownSelection(goldenScript.actions[idx], snapshotBefore))
            if (pendingTownIndex !== undefined) {
              return pendingTownIndex
            }

            const availableLeechIndex = remaining.find((idx) => actionHasAvailableLeechOffer(goldenScript.actions[idx], snapshotBefore))

            if (activeTurnPlayer !== '') {
              const turnOwnerIndex = remaining.find((idx) => {
                const candidate = goldenScript.actions[idx]
                return candidate.playerId === activeTurnPlayer && actionRequiresTurnOwnership(candidate, snapshotBefore)
              })
              if (availableLeechIndex !== undefined && availableLeechIndex < (turnOwnerIndex ?? Number.POSITIVE_INFINITY)) {
                return availableLeechIndex
              }
              if (turnOwnerIndex !== undefined) {
                return turnOwnerIndex
              }
            }

            if (availableLeechIndex !== undefined) {
              return availableLeechIndex
            }

            return remaining[0]
          })()

          const action = goldenScript.actions[nextIndex]
          debugLog(
            `[click-replay] select index=${nextIndex} actor=${action.playerId} type=${action.type} revision=${revision} pendingDecision=${pendingBefore.type}:${pendingBefore.playerId} pendingSpades=${pendingSpadePlayers.join(',')} activeTurn=${activeTurnPlayer}`,
          )
          const execution = await executeScriptedActionAt(nextIndex)
          if (execution.advanced) {
            remainingActionIndexes.delete(nextIndex)
            continue
          }

          const snapshotAfter = creator.getState(gameID)
          const pendingAfter = getPendingDecisionInfo(snapshotAfter)
          const pendingPreview = sortedRemaining().slice(0, 10).map((idx) => {
            const pendingAction = goldenScript.actions[idx]
            return `${idx}:${pendingAction.playerId}:${pendingAction.type}`
          })
          const snapshotDetails = describeActionSnapshot(action, snapshotBefore)
          throw new Error(
            `failed scripted action for scenario=${scenario.id} revision=${revision} phase=${String(phase)} index=${nextIndex}:${action.playerId}:${action.type} pendingDecision=${pendingAfter.type}:${pendingAfter.playerId} rejection=${execution.rejectionMessage}${snapshotDetails === '' ? '' : ` snapshot=${snapshotDetails}`} remaining=${pendingPreview.join(' | ')}`,
          )
        }

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

        for (const playerId of playerSet.values()) {
          actorPagePlayerId = await switchActorPage(actorPage, gameID, playerId, actorPagePlayerId)
          for (const vp of finalTotals.values()) {
            await expect(actorPage.getByTestId('player-summary-bar')).toContainText(`${String(vp)} VP`)
          }
        }
      } finally {
        await actorContext.close().catch(() => undefined)
        creator.close()
      }
    })
  }
})
