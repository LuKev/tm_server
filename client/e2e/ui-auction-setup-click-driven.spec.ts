import { expect, test, type Browser, type Page } from '@playwright/test'
import { BASE_GAME_MAP } from '../src/data/baseGameMap'
import { FactionType, TerrainType } from '../src/types/game.types'
import { realServerWsURL } from './support/realServerConfig'
import { loadRealServerGamePage, primeRealServerPage } from './support/realServerPage'
import { clickByTestId, clickHex } from './support/uiInteractions'
import { WsBot, type JsonObject } from './support/wsBot'

async function fetchGameState(
  observer: WsBot,
  gameID: string,
  playerID: string,
  timeoutMs = 10_000,
  forceRefresh = false,
): Promise<JsonObject> {
  const existing = observer.getState(gameID)
  if (!forceRefresh && existing) return existing
  observer.send('get_game_state', { gameID, playerID })
  const msg = await observer.waitForType('game_state_update', timeoutMs)
  const state = observer.getState(gameID)
  if (state) return state
  return (msg.payload ?? {}) as JsonObject
}

async function waitForRevisionViaFetch(
  observer: WsBot,
  gameID: string,
  playerID: string,
  minRevision: number,
  timeoutMs = 20_000,
): Promise<JsonObject> {
  const current = observer.getState(gameID)
  if (current && Number(current.revision ?? -1) >= minRevision) {
    return current
  }
  if (!current) {
    observer.send('get_game_state', { gameID, playerID })
  }
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    await observer.waitForType('game_state_update', Math.min(1_000, deadline - Date.now())).catch(() => null)
    const latest = observer.getState(gameID)
    const revision = Number((latest ?? {}).revision ?? -1)
    if (latest && revision >= minRevision) {
      return latest
    }
  }
  observer.send('get_game_state', { gameID, playerID })
  await observer.waitForType('game_state_update', 2_000).catch(() => null)
  const refreshed = observer.getState(gameID)
  const refreshedRevision = Number((refreshed ?? {}).revision ?? -1)
  if (refreshed && refreshedRevision >= minRevision) {
    return refreshed
  }
  throw new Error(`timeout waiting for revision >= ${String(minRevision)} for game=${gameID}`)
}

async function openPlayerGamePage(browser: Browser, gameID: string, playerId: string): Promise<Page> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await primeRealServerPage(page, playerId, { disableExpectedRevision: true })
  await loadRealServerGamePage(page, gameID, playerId)
  return page
}

function auctionSetupTitle(setupMode: 'auction' | 'fast_auction'): RegExp {
  return setupMode === 'auction' ? /Auction Setup/ : /Fast Auction Setup/
}

async function waitForAuctionSetupPanel(
  page: Page,
  setupMode: 'auction' | 'fast_auction',
  timeoutMs = 8_000,
): Promise<void> {
  const byTestId = page.getByTestId('auction-setup-panel')
  const byHeading = page.getByRole('heading', { name: auctionSetupTitle(setupMode) })
  try {
    await byTestId.waitFor({ state: 'visible', timeout: timeoutMs })
    return
  } catch {
    // no-op, fall back to heading
  }
  await byHeading.waitFor({ state: 'visible', timeout: timeoutMs })
}

async function clickSetupBonusCardUntilRevisionAdvances(
  page: Page,
  observer: WsBot,
  gameID: string,
  observerPlayerID: string,
  revision: number,
): Promise<number> {
  const getAvailableCards = (state: JsonObject): number[] => {
    const rawAvailable = ((state.bonusCards ?? {}) as JsonObject).available
    if (!rawAvailable || typeof rawAvailable !== 'object') {
      return []
    }
    return Object.keys(rawAvailable)
      .map((id) => Number(id))
      .filter((id) => Number.isFinite(id) && Number.isInteger(id))
      .sort((a, b) => a - b)
  }

  let loops = 0
  let lastError: string | null = null

  while (loops < 2) {
    loops++
    const state = observer.getState(gameID) ?? (await fetchGameState(observer, gameID, observerPlayerID, 5_000))
    const candidates = getAvailableCards(state)
    if (candidates.length === 0) {
      throw new Error(`failed to select setup bonus card: no available bonus cards in state for revision ${revision}`)
    }

    let clicked = false
    for (const cardId of candidates) {
      const locator = page.getByTestId(`setup-bonus-card-${String(cardId)}`)
      const visible = await locator.isVisible().catch(() => false)
      const enabled = await locator.isEnabled().catch(() => false)
      if (!visible || !enabled) continue
      await locator.click()
      clicked = true
      const next = await waitForRevisionViaFetch(observer, gameID, observerPlayerID, revision + 1, 4_000).catch(() => null)
      if (next) {
        return Number(next.revision ?? revision + 1)
      }
      const err = await page.getByTestId('action-error-message').textContent().catch(() => null)
      if (err && err.trim() !== '') {
        lastError = err.trim()
        continue
      }
      // If no action-error appears, keep a more specific trace for diagnostics.
      lastError = `no revision progression after clicking bonus card ${cardId} while available cards are [${candidates.join(', ')}]`
    }
    if (!clicked) {
      // Fall back to scanning rendered buttons in case the available map is
      // out of sync with client rendering.
      const fallbackButtons = page.locator('[data-testid^="setup-bonus-card-"]')
      const count = await fallbackButtons.count()
      for (let i = 0; i < count; i++) {
        const b = fallbackButtons.nth(i)
        const visible = await b.isVisible().catch(() => false)
        const enabled = await b.isEnabled().catch(() => false)
        if (!visible || !enabled) continue
        const testId = await b.getAttribute('data-testid').catch(() => null)
        await b.click()
        const next = await waitForRevisionViaFetch(observer, gameID, observerPlayerID, revision + 1, 4_000).catch(() => null)
        if (next) {
          return Number(next.revision ?? revision + 1)
        }
        const fallbackErr = await page.getByTestId('action-error-message').textContent().catch(() => null)
        if (fallbackErr && fallbackErr.trim() !== '') {
          lastError = fallbackErr.trim()
          continue
        }
        lastError = `no revision progression after clicking ${String(testId ?? 'unknown setup bonus button')}`
      }
    }
  }
  throw new Error(lastError ? `failed to select setup bonus card: ${lastError}` : 'failed to select setup bonus card')
}

async function resolveLobbyGameIDFromPage(page: Page, gameName: string): Promise<string> {
  const gameID = await page.waitForFunction((targetGameName) => {
    const buttons = Array.from(document.querySelectorAll<HTMLElement>('[data-testid^="lobby-start-"]'))
    for (const button of buttons) {
      const rowText = button.parentElement?.parentElement?.textContent ?? ''
      if (!rowText.includes(targetGameName)) continue
      const testId = button.dataset.testid ?? ''
      if (!testId.startsWith('lobby-start-')) continue
      return testId.replace('lobby-start-', '')
    }
    return null
  }, gameName, { timeout: 15_000 })
    .then((handle) => handle.jsonValue())

  const resolvedGameID = typeof gameID === 'string' ? gameID : ''
  if (resolvedGameID === '') {
    throw new Error(`could not resolve lobby game id for ${gameName}`)
  }
  return resolvedGameID
}

function terrainForFaction(faction: string): TerrainType | null {
  const normalized = String(faction)
  const factionType = Number.isInteger(Number(normalized)) ? Number(normalized) : normalized

  switch (factionType) {
    case 'Halflings':
    case FactionType.Halflings:
    case 'Cultists':
    case FactionType.Cultists:
      return TerrainType.Plains
    case 'Alchemists':
    case FactionType.Alchemists:
    case 'Darklings':
    case FactionType.Darklings:
      return TerrainType.Swamp
    case 'Mermaids':
    case FactionType.Mermaids:
    case 'Swarmlings':
    case FactionType.Swarmlings:
      return TerrainType.Lake
    case 'Witches':
    case FactionType.Witches:
    case 'Auren':
    case FactionType.Auren:
      return TerrainType.Forest
    case 'Engineers':
    case FactionType.Engineers:
    case 'Dwarves':
    case FactionType.Dwarves:
      return TerrainType.Mountain
    case 'Giants':
    case FactionType.Giants:
    case 'ChaosMagicians':
    case FactionType.ChaosMagicians:
      return TerrainType.Wasteland
    case 'Nomads':
    case FactionType.Nomads:
    case 'Fakirs':
    case FactionType.Fakirs:
      return TerrainType.Desert
    default:
      return null
  }
}

function directNeighborsOf(q: number, r: number): Set<string> {
  const deltas = [
    { dq: 1, dr: 0 },
    { dq: 1, dr: -1 },
    { dq: 0, dr: -1 },
    { dq: -1, dr: 0 },
    { dq: -1, dr: 1 },
    { dq: 0, dr: 1 },
  ]
  const out = new Set<string>()
  for (const d of deltas) {
    out.add(`${String(q + d.dq)},${String(r + d.dr)}`)
  }
  return out
}

function setupDwellingCandidateOrder(state: JsonObject, playerID: string): Array<{ q: number; r: number }> {
  const players = (state.players ?? {}) as Record<string, JsonObject>
  const player = (players[playerID] ?? {}) as JsonObject
  const faction = String(player.faction ?? '')
  const terrain = terrainForFaction(faction)
  if (terrain === null) {
    return BASE_GAME_MAP.map((h) => ({ q: h.coord.q, r: h.coord.r }))
  }

  const hexes = ((state.map ?? {}) as JsonObject).hexes as Record<string, JsonObject> | undefined
  if (!hexes) {
    return BASE_GAME_MAP.map((h) => ({ q: h.coord.q, r: h.coord.r }))
  }

  const emptyHome: Array<{ q: number; r: number }> = []
  const playerBuildings: Array<{ q: number; r: number }> = []
  for (const [key, hex] of Object.entries(hexes)) {
    const hexTerrain = Number(hex.terrain ?? -1)
    const building = (hex.building ?? null) as JsonObject | null
    if (building && String(building.ownerPlayerId ?? '') === playerID) {
      const [qS, rS] = key.split(',')
      playerBuildings.push({ q: Number(qS), r: Number(rS) })
    }
    if (hexTerrain === terrain && !building) {
      const [qS, rS] = key.split(',')
      emptyHome.push({ q: Number(qS), r: Number(rS) })
    }
  }

  if (playerBuildings.length === 0) {
    return emptyHome
  }

  const preferred = new Set<string>()
  for (const b of playerBuildings) {
    for (const n of directNeighborsOf(b.q, b.r)) {
      preferred.add(n)
    }
  }

  const prioritized = emptyHome.filter((h) => preferred.has(`${String(h.q)},${String(h.r)}`))
  const fallback = emptyHome.filter((h) => !preferred.has(`${String(h.q)},${String(h.r)}`))
  return [...prioritized, ...fallback]
}

async function clickFirstLegalHexUntilRevisionIncrements(
  page: Page,
  observer: WsBot,
  gameID: string,
  observerPlayerID: string,
  revision: number,
  state: JsonObject,
  actorID: string,
): Promise<number> {
  const candidates = setupDwellingCandidateOrder(state, actorID)
  const fallback = BASE_GAME_MAP.map((hex) => ({ q: hex.coord.q, r: hex.coord.r }))
  const seen = new Set<string>()
  const maxAttempts = candidates.length + fallback.length + 8

  let attempts = 0
  for (const hex of [...candidates, ...fallback]) {
    if (attempts >= maxAttempts) break
    attempts += 1
    const key = `${String(hex.q)},${String(hex.r)}`
    if (seen.has(key)) continue
    seen.add(key)
    await clickHex(page, hex.q, hex.r)
    const next = await waitForRevisionViaFetch(observer, gameID, observerPlayerID, revision + 1, 450).catch(() => null)
    const nextRevision = Number((next ?? {}).revision ?? revision)
    if (nextRevision > revision) {
      return nextRevision
    }
  }
  const error = `failed to find a legal setup dwelling hex for ${actorID} (revision=${String(revision)}, attempts=${String(attempts)}, candidateCount=${String(candidates.length)}, fallbackCount=${String(fallback.length)})`
  throw new Error(error)
}

async function runAuctionToActionPhase(
  browser: Browser,
  hostPage: Page,
  setupMode: 'auction' | 'fast_auction',
): Promise<void> {
  const wsURL = realServerWsURL()
  const runId = String(Math.floor(Math.random() * 10_000))
  const prefix = setupMode === 'fast_auction' ? 'fa' : 'au'
  const hostName = `${prefix}h${runId}`
  const joiners = [`${prefix}p2${runId}`, `${prefix}p3${runId}`, `${prefix}p4${runId}`]
  const gameName = `e2e-${setupMode}-${runId}`
  await hostPage.addInitScript(() => {
    ;(window as Window & { __TM_DISABLE_EXPECTED_REVISION__?: boolean }).__TM_DISABLE_EXPECTED_REVISION__ = true
  })
  await hostPage.goto('/')
  await expect(hostPage.getByTestId('lobby-screen')).toBeVisible()
  await hostPage.getByTestId('lobby-player-name').fill(hostName)
  await hostPage.getByTestId('lobby-game-name').fill(gameName)
  await hostPage.getByTestId('lobby-max-players').selectOption('4')
  await hostPage.getByTestId('lobby-randomize-turn-order').uncheck()
  await hostPage.getByTestId('lobby-setup-mode').selectOption(setupMode)
  await hostPage.getByTestId('lobby-create-game').click()

  const gameID = await resolveLobbyGameIDFromPage(hostPage, gameName)
  const startBtn = hostPage.getByTestId(`lobby-start-${gameID}`)

  const joinBots = new Map<string, WsBot>()
  const observer = await WsBot.connect(wsURL)
  try {
    for (const playerId of joiners) {
      const bot = await WsBot.connect(wsURL)
      joinBots.set(playerId, bot)
      bot.send('join_game', { id: gameID, name: playerId })
      const joinResult = await bot.waitForAnyType(['game_joined', 'error'], 15_000)
      const joinType = String(joinResult.type ?? '')
      if (joinType !== 'game_joined') {
        throw new Error(`join_game failed for ${playerId}: ${JSON.stringify(joinResult.payload ?? {})}`)
      }
    }

    await expect(startBtn).toBeEnabled({ timeout: 15_000 })
    await startBtn.click()
    await expect(hostPage).toHaveURL(new RegExp(`/game/${gameID}$`))
    await waitForAuctionSetupPanel(hostPage, setupMode)

    const initial = await fetchGameState(observer, gameID, hostName, 15_000)
    let revision = Number(initial.revision ?? 0)
    const withActorPage = async <T>(playerID: string, action: (page: Page) => Promise<T>): Promise<T> => {
      const actorPage = await openPlayerGamePage(browser, gameID, playerID)
      try {
        return await action(actorPage)
      } finally {
        await actorPage.context().close()
      }
    }

    let guard = 0
    while (guard < 300) {
      guard++
      const state = await fetchGameState(observer, gameID, hostName, 10_000, true)
      revision = Number(state.revision ?? revision)

      const phase = Number(state.phase ?? -1)
      if (phase === 3) break

      const setupSubphase = String(state.setupSubphase ?? '')
      const setupDwellingOrder = (state.setupDwellingOrder ?? []) as string[]
      const setupBonusOrder = (state.setupBonusOrder ?? []) as string[]
      const setupDwellingIndex = Number(state.setupDwellingIndex ?? 0)
      const setupBonusIndex = Number(state.setupBonusIndex ?? 0)
      const pendingDecision = (state.pendingDecision ?? {}) as JsonObject
      const pendingType = String(pendingDecision.type ?? '')
      const pendingPlayerId = String(
        pendingDecision.playerId ?? ((state.auctionState ?? {}) as JsonObject).currentBidder ?? '',
      )
      if (phase === 1) {
        const actorId = pendingPlayerId
        if (!actorId) throw new Error('missing pending auction player id')

        if (pendingType === 'auction_nomination') {
          let advanced = false
          await withActorPage(actorId, async (actorPage) => {
            await waitForAuctionSetupPanel(actorPage, setupMode, 8_000)
            const options = actorPage.locator('[data-testid^="auction-nominate-"]')
            await expect
              .poll(async () => options.count(), { timeout: 8_000 })
              .toBeGreaterThan(0)
            const count = await options.count()
            for (let i = 0; i < count; i++) {
              const option = options.nth(i)
              const visible = await option.isVisible().catch(() => false)
              const enabled = await option.isEnabled().catch(() => false)
              if (!visible || !enabled) continue
              await option.click()
              const next = await waitForRevisionViaFetch(observer, gameID, hostName, revision + 1, 1_500).catch(() => null)
              if (next) {
                revision = Number(next.revision ?? revision + 1)
                advanced = true
                break
              }
            }
          })
          if (!advanced) {
            throw new Error(`failed to make progress on auction nomination for ${actorId}`)
          }
          continue
        } else if (pendingType === 'auction_bid') {
          const auctionState = (state.auctionState ?? {}) as JsonObject
          const currentBids = (auctionState.currentBids ?? {}) as Record<string, number>
          const factionHolders = (auctionState.factionHolders ?? {}) as Record<string, string>

          const desiredFactions = Object.keys(currentBids).sort((a, b) => {
            const holderA = String(factionHolders[a] ?? '')
            const holderB = String(factionHolders[b] ?? '')
            const aFree = holderA === '' ? 0 : 1
            const bFree = holderB === '' ? 0 : 1
            if (aFree !== bFree) return aFree - bFree
            return a.localeCompare(b)
          })
          let advanced = false
          await withActorPage(actorId, async (actorPage) => {
            await waitForAuctionSetupPanel(actorPage, setupMode, 8_000)
            const bids = actorPage.locator('[data-testid^="auction-bid-input-"]')
            await expect
              .poll(async () => bids.count(), { timeout: 8_000 })
              .toBeGreaterThan(0)
            const count = await bids.count()
            for (const factionName of desiredFactions) {
              const input = actorPage.getByTestId(`auction-bid-input-${factionName}`).first()
              const visible = await input.isVisible().catch(() => false)
              const enabled = await input.isEnabled().catch(() => false)
              if (!visible || !enabled) continue
              const currentBid = Number(currentBids[factionName] ?? 0)
              const holder = String(factionHolders[factionName] ?? '')
              if (holder !== '' && holder !== actorId && currentBid >= 40) continue
              const bidValue = holder !== '' && holder !== actorId ? Math.min(40, currentBid + 1) : currentBid
              await input.fill(String(bidValue))
              const submit = actorPage.getByTestId(`auction-bid-submit-${factionName}`).first()
              const submitVisible = await submit.isVisible().catch(() => false)
              const submitEnabled = await submit.isEnabled().catch(() => false)
              if (!submitVisible || !submitEnabled) continue
              await submit.click()
              const next = await waitForRevisionViaFetch(observer, gameID, hostName, revision + 1, 1_500).catch(() => null)
              if (next) {
                revision = Number(next.revision ?? revision + 1)
                advanced = true
                break
              }
              const err = await actorPage.getByTestId('action-error-message').textContent().catch(() => null)
              if (err && err.trim() !== '') {
              }
            }
            if (!advanced && count > 0) {
              for (let i = 0; i < count; i++) {
                const input = bids.nth(i)
                const visible = await input.isVisible().catch(() => false)
                const enabled = await input.isEnabled().catch(() => false)
                if (!visible || !enabled) continue
                const inputId = await input.getAttribute('data-testid')
                if (!inputId) continue
                const suffix = inputId.replace('auction-bid-input-', '')
                const submit = actorPage.getByTestId(`auction-bid-submit-${suffix}`).first()
                const submitVisible = await submit.isVisible().catch(() => false)
                const submitEnabled = await submit.isEnabled().catch(() => false)
                if (!submitVisible || !submitEnabled) continue
                await submit.click()
                const next = await waitForRevisionViaFetch(observer, gameID, hostName, revision + 1, 1_500).catch(() => null)
                if (next) {
                  revision = Number(next.revision ?? revision + 1)
                  advanced = true
                  break
                }
              }
            }
          })
          if (!advanced) {
            const auctionState = (state.auctionState ?? {}) as JsonObject
            throw new Error(`failed to make progress on auction bid for ${actorId}`)
          }
          continue
        } else if (pendingType === 'fast_auction_bid_matrix') {
          await withActorPage(actorId, async (actorPage) => {
            await waitForAuctionSetupPanel(actorPage, setupMode, 8_000)
            const inputs = actorPage.locator('[data-testid^="fast-auction-bid-input-"]')
            await expect
              .poll(async () => inputs.count(), { timeout: 8_000 })
              .toBeGreaterThan(0)
            const count = await inputs.count()
            for (let i = 0; i < count; i++) {
              await inputs.nth(i).fill(String((i + 1) * 2))
            }
            await clickByTestId(actorPage, 'fast-auction-submit')
            const next = await waitForRevisionViaFetch(observer, gameID, hostName, revision + 1, 15_000)
            revision = Number(next.revision ?? revision + 1)
          })
          continue
        } else {
          throw new Error(`unexpected pending auction decision: ${pendingType}`)
        }
      }

      if (phase === 0 && setupSubphase === 'dwellings') {
        const order = (state.setupDwellingOrder ?? []) as string[]
        const idx = Number(state.setupDwellingIndex ?? 0)
        const actorId = order[idx] ?? ''
        if (!actorId) throw new Error('missing setup dwelling actor id')
        await withActorPage(actorId, async (actorPage) => {
          revision = await clickFirstLegalHexUntilRevisionIncrements(actorPage, observer, gameID, hostName, revision, state, actorId)
        })
        continue
      }

      if (phase === 0 && setupSubphase === 'bonus_cards') {
        const actorId = pendingPlayerId
        if (!actorId) throw new Error('missing setup bonus actor id')
        await withActorPage(actorId, async (actorPage) => {
          await actorPage.locator('[data-testid^="setup-bonus-card-"]').first().waitFor({ state: 'visible', timeout: 8_000 })
          revision = await clickSetupBonusCardUntilRevisionAdvances(actorPage, observer, gameID, hostName, revision)
        })
        continue
      }

      const next = await waitForRevisionViaFetch(observer, gameID, hostName, revision + 1, 15_000)
      revision = Number(next.revision ?? revision + 1)
    }

    if (guard >= 300) throw new Error(`${setupMode} setup flow exceeded guard`)
    const finalState = observer.getState(gameID)
    expect(Number((finalState ?? {}).phase ?? -1)).toBe(3)
  } finally {
    observer.close()
    for (const bot of joinBots.values()) bot.close()
  }
}

test.describe('Auction Setup UI Click-Driven (Real Server)', () => {
  test.setTimeout(360_000)

  test('@smoke auction setup goes from lobby to action phase through UI decisions', async ({ browser, page }) => {
    await runAuctionToActionPhase(browser, page, 'auction')
  })

  test('@smoke fast auction setup goes from lobby to action phase through UI decisions', async ({ browser, page }) => {
    await runAuctionToActionPhase(browser, page, 'fast_auction')
  })
})
