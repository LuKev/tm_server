import { expect, test, type Page } from '@playwright/test'
import {
  BonusCardType,
  BuildingType,
  CultType,
  FactionType,
  FavorTileType,
  GamePhase,
  PowerActionType,
  SpecialActionType,
  TerrainType,
  TownTileId,
  type GameState,
  type PlayerState,
} from '../src/types/game.types'
import { makeBaseGameState, withBuildings } from './support/gameStateFactory'
import { formatDisplayCoordinate, getDisplayCoordinate } from '../src/utils/hexUtils'
import {
  clearSentMessages,
  emitWs,
  installMockWebSocket,
  waitForMessageType,
  waitForSocketReady,
  waitForPerformAction,
} from './support/mockWebSocket'
import { clickByTestId, clickCultSpot, clickHex, confirmAction } from './support/uiInteractions'

const unknownFactionPlayers = (): Record<string, PlayerState> => ({
  p1: {
    id: 'p1',
    name: 'p1',
    faction: FactionType.Unknown,
    resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
    shipping: 0,
    digging: 0,
    cults: {},
    buildings: {},
    victoryPoints: 20,
  },
  p2: {
    id: 'p2',
    name: 'p2',
    faction: FactionType.Unknown,
    resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
    shipping: 0,
    digging: 0,
    cults: {},
    buildings: {},
    victoryPoints: 20,
  },
  p3: {
    id: 'p3',
    name: 'p3',
    faction: FactionType.Unknown,
    resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
    shipping: 0,
    digging: 0,
    cults: {},
    buildings: {},
    victoryPoints: 20,
  },
  p4: {
    id: 'p4',
    name: 'p4',
    faction: FactionType.Unknown,
    resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
    shipping: 0,
    digging: 0,
    cults: {},
    buildings: {},
    victoryPoints: 20,
  },
})

const openGameWithState = async (page: Page, state: GameState, localPlayerId = 'p1'): Promise<void> => {
  await installMockWebSocket(page, localPlayerId)
  await page.goto('/game/test-game')
  await waitForSocketReady(page)
  await emitWs(page, { type: 'game_state_update', payload: state })
  await expect(page.getByTestId('game-screen')).toBeVisible()
  await expect(page.getByTestId('hex-grid-canvas')).toBeVisible()
  await clearSentMessages(page)
}

test.describe('UI Action Contract (Playwright + mocked websocket)', () => {
  test('lobby create/leave/start reflects single-seat behavior', async ({ page }) => {
    await installMockWebSocket(page, 'host')
    await page.goto('/')

    await expect(page.getByTestId('lobby-screen')).toBeVisible()
    await page.getByTestId('lobby-player-name').fill('host')
    await page.getByTestId('lobby-game-name').fill('UI Contract Game')
    await page.getByTestId('lobby-max-players').selectOption('5')

    await page.getByTestId('lobby-create-game').click()

    await expect.poll(async () => {
      return page.evaluate(() => {
        const msgs = window.__tmE2E?.sent ?? []
        const last = msgs[msgs.length - 1] as Record<string, unknown> | undefined
        if (!last) return null
        return { type: last.type, payload: last.payload }
      })
    }).toMatchObject({
      type: 'create_game',
      payload: {
        name: 'UI Contract Game',
        maxPlayers: 5,
        creator: 'host',
      },
    })

    await emitWs(page, {
      type: 'lobby_state',
      payload: [
        { id: 'g-ui', name: 'UI Contract Game', host: 'host', players: ['host', 'p2', 'p3', 'p4'], maxPlayers: 5 },
        { id: 'g-other', name: 'Other Game', host: 'other', players: ['other'], maxPlayers: 5 },
      ],
    })

    await expect(page.getByTestId('lobby-create-game')).toBeDisabled()
    await expect(page.getByTestId('lobby-join-g-other')).toBeDisabled()

    await page.getByTestId('lobby-leave-g-ui').click()
    await expect.poll(async () => {
      return page.evaluate(() => {
        const msgs = window.__tmE2E?.sent ?? []
        const leave = [...msgs]
          .reverse()
          .find((msg) => typeof msg === 'object' && msg !== null && (msg as Record<string, unknown>).type === 'leave_game')
        if (!leave) return null
        const parsed = leave as Record<string, unknown>
        return { type: parsed.type, payload: parsed.payload }
      })
    }).toMatchObject({
      type: 'leave_game',
      payload: {
        id: 'g-ui',
        name: 'host',
      },
    })

    await page.getByTestId('lobby-randomize-turn-order').uncheck()
    await page.getByTestId('lobby-setup-mode').selectOption('fast_auction')
    await emitWs(page, {
      type: 'lobby_state',
      payload: [{ id: 'g-ui', name: 'UI Contract Game', host: 'host', players: ['host', 'p2', 'p3', 'p4', 'p5'], maxPlayers: 5 }],
    })
    await page.getByTestId('lobby-start-g-ui').click()

    await expect.poll(async () => {
      return page.evaluate(() => {
        const msgs = window.__tmE2E?.sent ?? []
        const start = [...msgs]
          .reverse()
          .find((msg) => typeof msg === 'object' && msg !== null && (msg as Record<string, unknown>).type === 'start_game')
        if (!start) return null
        const parsed = start as Record<string, unknown>
        return { type: parsed.type, payload: parsed.payload }
      })
    }).toMatchObject({
      type: 'start_game',
      payload: {
        gameID: 'g-ui',
        randomizeTurnOrder: false,
        setupMode: 'fast_auction',
      },
    })
  })

  test('lobby join emits join_game when player is not already seated', async ({ page }) => {
    await installMockWebSocket(page, 'guest')
    await page.goto('/')

    await page.getByTestId('lobby-player-name').fill('guest')
    await emitWs(page, {
      type: 'lobby_state',
      payload: [{ id: 'g-ui', name: 'Joinable', host: 'host', players: ['host'], maxPlayers: 5 }],
    })

    await page.getByTestId('lobby-join-g-ui').click()
    await expect.poll(async () => {
      return page.evaluate(() => {
        const msgs = window.__tmE2E?.sent ?? []
        const join = [...msgs]
          .reverse()
          .find((msg) => typeof msg === 'object' && msg !== null && (msg as Record<string, unknown>).type === 'join_game')
        if (!join) return null
        const parsed = join as Record<string, unknown>
        return { type: parsed.type, payload: parsed.payload }
      })
    }).toMatchObject({
      type: 'join_game',
      payload: {
        id: 'g-ui',
        name: 'guest',
      },
    })
  })

  test('lobby navigates to the started game even if lobby_state arrives immediately after', async ({ page }) => {
    await installMockWebSocket(page, 'host')
    await page.goto('/')

    await page.getByTestId('lobby-player-name').fill('host')
    const startedState = makeBaseGameState({
      id: 'g-start',
      started: true,
      players: {
        host: {
          id: 'host',
          name: 'host',
          faction: FactionType.Unknown,
          resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
          shipping: 0,
          digging: 0,
          cults: {},
          buildings: {},
          victoryPoints: 20,
        },
        p2: {
          id: 'p2',
          name: 'p2',
          faction: FactionType.Unknown,
          resources: { coins: 15, workers: 7, priests: 1, power: { powerI: 5, powerII: 7, powerIII: 0 } },
          shipping: 0,
          digging: 0,
          cults: {},
          buildings: {},
          victoryPoints: 20,
        },
      },
      turnOrder: ['host', 'p2'],
    })
    await page.evaluate((state) => {
      const emit = window.__tmE2E?.emit
      if (!emit) throw new Error('missing mock websocket emitter')

      emit({
        type: 'game_state_update',
        payload: state,
      })
      emit({
        type: 'lobby_state',
        payload: [],
      })
    }, startedState)

    await expect(page).toHaveURL(/\/game\/g-start$/)
  })

  test('faction selection emits select_faction', async ({ page }) => {
    const state = makeBaseGameState({
      phase: GamePhase.FactionSelection,
      setupMode: 'snellman',
      players: unknownFactionPlayers(),
      turnOrder: ['p1', 'p2', 'p3', 'p4'],
      currentTurn: 0,
    })

    await openGameWithState(page, state)
    await clickByTestId(page, 'faction-option-Auren')
    await confirmAction(page)

    await waitForPerformAction(page, 'select_faction', { faction: 'Auren' })
  })

  test('auction and fast auction pending decisions emit expected actions', async ({ page }) => {
    const auctionState = makeBaseGameState({
      phase: GamePhase.FactionSelection,
      setupMode: 'auction',
      pendingDecision: {
        type: 'auction_nomination',
        playerId: 'p1',
      },
      auctionState: {
        active: true,
        mode: 'auction',
        nominationPhase: true,
        currentBidder: 'p1',
        currentBidderIndex: 0,
        nominationsComplete: 0,
        nominationOrder: [],
        currentBids: {},
        factionHolders: {},
        seatOrder: ['p1', 'p2', 'p3', 'p4'],
        playerHasFaction: {},
        fastSubmitted: {},
        fastBids: {},
      },
    })

    await openGameWithState(page, auctionState)
    await clickByTestId(page, 'auction-nominate-Nomads')
    await confirmAction(page)
    await waitForPerformAction(page, 'auction_nominate', { faction: 'Nomads' })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...auctionState,
        pendingDecision: {
          type: 'auction_bid',
          playerId: 'p1',
          nominatedFactions: ['Nomads'],
        },
        auctionState: {
          ...(auctionState.auctionState ?? {}),
          nominationOrder: ['Nomads'],
          currentBids: { Nomads: 0 },
        },
      },
    })
    await clearSentMessages(page)

    await page.getByTestId('auction-bid-input-Nomads').fill('7')
    await clickByTestId(page, 'auction-bid-submit-Nomads')
    await confirmAction(page)
    await waitForPerformAction(page, 'auction_bid', { faction: 'Nomads', vpReduction: 7 })

    const fastAuctionState = {
      ...auctionState,
      setupMode: 'fast_auction' as const,
      pendingDecision: {
        type: 'fast_auction_bid_matrix',
        playerId: 'p1',
        nominatedFactions: ['Nomads', 'Darklings'],
      },
      auctionState: {
        ...(auctionState.auctionState ?? {}),
        mode: 'fast_auction' as const,
        nominationOrder: ['Nomads', 'Darklings'],
      },
    }

    await emitWs(page, { type: 'game_state_update', payload: fastAuctionState })
    await clearSentMessages(page)

    await page.getByTestId('fast-auction-bid-input-Nomads').fill('5')
    await page.getByTestId('fast-auction-bid-input-Darklings').fill('9')
    await clickByTestId(page, 'fast-auction-submit')
    await confirmAction(page)
    await waitForPerformAction(page, 'fast_auction_submit_bids', {
      bids: {
        Nomads: 5,
        Darklings: 9,
      },
    })
  })

  test('setup dwelling and setup bonus card flow emits expected actions', async ({ page }) => {
    const setupState = makeBaseGameState({
      phase: GamePhase.Setup,
      setupSubphase: 'dwellings',
      setupDwellingOrder: ['p1', 'p2', 'p3', 'p4'],
      setupDwellingIndex: 0,
    })

    await openGameWithState(page, setupState)
    await clickHex(page, 0, 0)
    await confirmAction(page)
    await waitForPerformAction(page, 'setup_dwelling', { hex: { q: 0, r: 0 } })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...setupState,
        pendingDecision: {
          type: 'setup_bonus_card',
          playerId: 'p1',
        },
        bonusCards: {
          ...(setupState.bonusCards ?? {}),
          available: {
            [BonusCardType.Spade]: 0,
            [BonusCardType.CultAdvance]: 0,
            [BonusCardType.Coins6]: 0,
          },
        },
      },
    })
    await clearSentMessages(page)

    await clickByTestId(page, 'setup-bonus-card-4')
    await waitForPerformAction(page, 'setup_bonus_card', { bonusCard: 4 })
  })

  test('cult priest send and leech decisions emit expected actions', async ({ page }) => {
    const state = makeBaseGameState({
      phase: GamePhase.Action,
      currentTurn: 0,
      turnOrder: ['p1', 'p2', 'p3', 'p4'],
    })

    await openGameWithState(page, state)
    await clickCultSpot(page, 0, 0)
    await confirmAction(page)
    await waitForPerformAction(page, 'send_priest', { cultTrack: CultType.Fire, spacesToClimb: 3 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingDecision: {
          type: 'leech_offer',
          playerId: 'p1',
          offers: [{ Amount: 3 }],
        },
      },
    })
    await clearSentMessages(page)

    await clickByTestId(page, 'leech-offer-0-accept')
    await waitForPerformAction(page, 'accept_leech', { offerIndex: 0 })

    await clearSentMessages(page)
    await clickByTestId(page, 'leech-offer-0-decline')
    await waitForPerformAction(page, 'decline_leech', { offerIndex: 0 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        currentTurn: 2,
        pendingDecision: {
          type: 'post_action_free_actions',
          playerId: 'p2',
        },
        pendingLeechOffers: {
          p1: [{ Amount: 2 }],
        },
      },
    })
    await clearSentMessages(page)

    await expect(page.getByTestId('leech-offer-0-accept')).toBeVisible()
    await clickByTestId(page, 'leech-offer-0-accept')
    await waitForPerformAction(page, 'accept_leech', { offerIndex: 0 })
  })

  test('post-action free window renders inline turn confirm and undo actions', async ({ page }) => {
    const state = makeBaseGameState({
      currentTurn: 1,
      pendingDecision: {
        type: 'post_action_free_actions',
        playerId: 'p1',
      },
    })

    await openGameWithState(page, state)
    await expect(page.getByTestId('game-decision-strip')).toContainText('Confirm Turn')
    await expect(page.getByTestId('game-decision-strip')).not.toContainText('Undo restores the last undo checkpoint')
    await expect(page.getByTestId('turn-end-confirm')).toBeVisible()
    await expect(page.getByTestId('turn-end-undo')).toBeVisible()
    await expect(page.getByTestId('game-decision-strip')).toContainText('Confirm Turn')
    await expect(page.getByTestId('game-decision-strip')).not.toContainText('You can still make conversions or burn power before confirming')

    await clickByTestId(page, 'turn-end-confirm')
    await waitForPerformAction(page, 'confirm_turn')

    await clearSentMessages(page)
    await clickByTestId(page, 'turn-end-undo')
    await waitForPerformAction(page, 'undo_turn')
  })

  test('turn confirmation stays inline and local conversions still work during post-action free actions', async ({ page }) => {
    const state = makeBaseGameState({
      currentTurn: 1,
      pendingDecision: {
        type: 'post_action_free_actions',
        playerId: 'p1',
      },
    })

    await openGameWithState(page, state)
    await expect(page.getByTestId('turn-end-confirm')).toBeVisible()
    await clickByTestId(page, 'player-p1-conversion-worker_to_coin')
    await waitForPerformAction(page, 'conversion', { conversionType: 'worker_to_coin', amount: 1 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingDecision: {
          type: 'turn_confirmation',
          playerId: 'p1',
        },
      },
    })
    await clearSentMessages(page)

    await expect(page.getByTestId('turn-end-confirm')).toBeVisible()
    await expect(page.getByTestId('turn-end-undo')).toBeVisible()
    await clickByTestId(page, 'turn-end-confirm')
    await waitForPerformAction(page, 'confirm_turn')
  })

  test('display coordinates use server-style board labels', () => {
    expect(formatDisplayCoordinate({ q: 0, r: 0 })).toBe('A1')
    expect(formatDisplayCoordinate({ q: 3, r: 4 })).toBe('E6')
    expect(formatDisplayCoordinate({ q: -2, r: 5 })).toBe('F1')
    expect(formatDisplayCoordinate({ q: 8, r: 8 })).toBe('I12')
    expect(getDisplayCoordinate({ q: 0, r: 2 })).toBeNull()
  })

  test('hex action strip uses display coordinates instead of axial coordinates', async ({ page }) => {
    const state = makeBaseGameState()

    await openGameWithState(page, state)
    await clickHex(page, 0, 0)
    await expect(page.getByTestId('game-decision-strip')).toContainText('Selected hex: A1')
  })

  test('next player cannot act until the previous player confirms', async ({ page }) => {
    const state = makeBaseGameState({
      currentTurn: 1,
      pendingDecision: {
        type: 'turn_confirmation',
        playerId: 'p1',
      },
    })

    await openGameWithState(page, state, 'p2')
    await expect(page.getByTestId('player-p2-conversion-worker_to_coin')).toBeDisabled()
    await expect(page.getByTestId('player-p2-upgrade-shipping')).toBeDisabled()
    await expect(page.getByTestId(`power-action-${String(PowerActionType.Priest)}`)).toHaveCSS('cursor', 'not-allowed')
    await page.getByTestId(`power-action-${String(PowerActionType.Priest)}`).click({ force: true })
    await expect.poll(async () => page.evaluate(() => (window.__tmE2E?.sent ?? []).length)).toBe(0)
  })

  test('faction-specific ship and dig upgrades are hidden when the faction has no track', async ({ page }) => {
    const basePlayers = makeBaseGameState().players
    const state = makeBaseGameState({
      players: {
        ...basePlayers,
        p1: { ...basePlayers.p1, faction: FactionType.Darklings },
      },
    })

    await openGameWithState(page, state)
    await expect(page.getByTestId('player-p1-upgrade-shipping')).toBeVisible()
    await expect(page.getByTestId('player-p1-upgrade-digging')).toHaveCount(0)

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        players: {
          ...state.players,
          p1: { ...state.players.p1, faction: FactionType.Dwarves },
        },
      },
    })
    await expect(page.getByTestId('player-p1-upgrade-shipping')).toHaveCount(0)
    await expect(page.getByTestId('player-p1-upgrade-digging')).toBeVisible()

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        players: {
          ...state.players,
          p1: { ...state.players.p1, faction: FactionType.Fakirs },
        },
      },
    })
    await expect(page.getByTestId('player-p1-upgrade-shipping')).toHaveCount(0)
    await expect(page.getByTestId('player-p1-upgrade-digging')).toBeVisible()
  })

  test('player options reflect state and emit set_player_options actions', async ({ page }) => {
    const state = makeBaseGameState({
      players: {
        ...makeBaseGameState().players,
        p1: {
          ...makeBaseGameState().players.p1,
          options: {
            autoLeechMode: 'accept_2',
            autoConvertOnPass: true,
            confirmActions: true,
            showIncomePreview: true,
          },
        },
      },
    })

    await openGameWithState(page, state)

    await expect(page.getByTestId('option-auto-leech-mode')).toHaveValue('accept_2')
    await expect(page.getByTestId('option-auto-convert-pass')).toBeChecked()
    await expect(page.getByTestId('option-show-income-preview')).toBeChecked()

    await clearSentMessages(page)
    await page.getByTestId('option-auto-leech-mode').selectOption('decline_vp')
    await waitForPerformAction(page, 'set_player_options', { autoLeechMode: 'decline_vp' })
    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        players: {
          ...state.players,
          p1: {
            ...state.players.p1,
            options: {
              autoLeechMode: 'decline_vp',
              autoConvertOnPass: true,
              confirmActions: true,
              showIncomePreview: true,
            },
          },
        },
      },
    })
    await expect(page.getByTestId('option-auto-leech-mode')).toHaveValue('decline_vp')

    await clearSentMessages(page)
    await page.getByTestId('option-auto-convert-pass').click()
    await waitForPerformAction(page, 'set_player_options', { autoConvertOnPass: false })

    await clearSentMessages(page)
    await page.getByTestId('option-show-income-preview').click()
    await waitForPerformAction(page, 'set_player_options', { showIncomePreview: false })
  })

  test('decision strip shows waiting/prompt states and the local board shows conversions and towns', async ({ page }) => {
    const state = makeBaseGameState({
      currentTurn: 1,
      players: {
        ...makeBaseGameState().players,
        p1: { ...makeBaseGameState().players.p1, name: 'Alice', townTiles: [TownTileId.Vp4Ship1] },
        p2: { ...makeBaseGameState().players.p2, name: 'Bob', townTiles: [TownTileId.Vp7Workers2] },
        p3: { ...makeBaseGameState().players.p3, name: 'Carol' },
      },
    })

    await openGameWithState(page, state)
    await expect(page.getByTestId('game-decision-strip')).toContainText('Bob must take an action.')

    await expect(page.getByTestId('player-summary-bar')).toContainText('Alice')
    await expect(page.getByTestId('player-summary-bar')).toContainText('Bob')
    await expect(page.getByTestId('player-summary-bar')).toContainText('YOU')

    const localBoard = page.getByTestId('player-board-p1')
    await expect(page.getByTestId('player-p1-conversion-worker_to_coin')).toBeVisible()
    await expect(localBoard.getByText('Conversions')).toBeVisible()
    await expect(localBoard.getByText('Towns')).toBeVisible()
    await expect(localBoard.locator('.pb-town-slot')).toHaveCount(1)
    await expect(localBoard).not.toContainText('Ship/Carpet')
    await expect(page.getByTestId('player-p2-conversion-worker_to_coin')).toHaveCount(0)

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingDecision: { type: 'favor_tile_selection', playerId: 'p1' },
      },
    })
    await expect(page.getByTestId('game-decision-strip')).toContainText('You must select a favor tile.')
    await expect(page.getByTestId('game-decision-strip')).toContainText('Please select a favor tile')
    await expect(page.getByTestId(`favor-tile-${String(FavorTileType.Water2)}`)).toHaveClass(/favor-tile-selectable/)

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingDecision: { type: 'town_tile_selection', playerId: 'p1' },
      },
    })
    await expect(page.getByTestId('game-decision-strip')).toContainText('You must select a town tile.')
    await expect(page.getByTestId('game-decision-strip')).toContainText('Please select a town tile')
    await expect(page.getByTestId(`town-tile-${String(TownTileId.Vp7Workers2)}`)).toHaveClass(/town-tile-slot-selectable/)

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        currentTurn: 0,
        pendingDecision: {
          type: 'leech_offer',
          playerId: 'p2',
          offers: [{ Amount: 2 }],
        },
        pendingLeechOffers: {
          p2: [{ Amount: 2 }],
          p3: [{ Amount: 1 }],
        },
      },
    })
    await expect(page.getByTestId('game-decision-strip')).toContainText('Bob and Carol must make leech decisions.')
    await expect(page.getByTestId('game-decision-strip')).toContainText('Waiting On Leech Responses')
    await expect(page.getByTestId('game-decision-strip')).toContainText('Waiting for Bob and Carol to accept or decline leech.')

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingDecision: {
          type: 'cult_reward_spade',
          playerId: 'p2',
          spadesRemaining: 1,
        },
        pendingLeechOffers: {},
      },
    })
    await expect(page.getByTestId('game-decision-strip')).toContainText('Bob must use cult spades.')
  })

  test('selectable town tiles highlight in green on hover', async ({ page }) => {
    const state = makeBaseGameState({
      pendingDecision: {
        type: 'town_tile_selection',
        playerId: 'p1',
      },
    })

    await openGameWithState(page, state)

    const tile = page.getByTestId(`town-tile-${String(TownTileId.Vp7Workers2)}`).first()
    await tile.hover()
    const highlightShadow = await tile.evaluate((node) => {
      const tileNode = node.querySelector('.town-tile-stack-0') as HTMLElement | null
      return tileNode ? getComputedStyle(tileNode).boxShadow : ''
    })
    expect(highlightShadow).toMatch(/34,\s*197,\s*94/)
  })

  test('ship town tile uses iconography instead of the Ship/Carpet text label', async ({ page }) => {
    const state = makeBaseGameState({
      pendingDecision: {
        type: 'town_tile_selection',
        playerId: 'p1',
      },
    })

    await openGameWithState(page, state)
    await expect(page.getByTestId(`town-tile-${String(TownTileId.Vp4Ship1)}`)).not.toContainText('Ship/Carpet')
  })

  test('transform/build, building upgrade, conversion, ship/dig and burn are wired to perform_action', async ({ page }) => {
    let state = makeBaseGameState({
      phase: GamePhase.Action,
      currentTurn: 0,
      turnOrder: ['p1', 'p2', 'p3', 'p4'],
    })

    state = withBuildings(state, [
      { q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Nomads, type: BuildingType.Dwelling, terrain: TerrainType.Desert },
    ])

    await openGameWithState(page, state)

    await clickHex(page, 1, 0)
    await expect(page.getByTestId('game-decision-strip')).toContainText('Selected hex: A2')
    await page.getByTestId('hex-action-mode').selectOption('transform_build')
    await page.getByTestId('hex-action-target-terrain').selectOption(String(TerrainType.Desert))
    await clickByTestId(page, 'hex-action-submit')
    await waitForPerformAction(page, 'transform_build', {
      hex: { q: 1, r: 0 },
      buildDwelling: true,
      targetTerrain: TerrainType.Desert,
    })

    await clearSentMessages(page)
    await clickHex(page, 0, 0)
    await expect(page.getByTestId('game-decision-strip')).toContainText('Select an upgrade for A1.')
    await clickByTestId(page, 'upgrade-option-1')
    await waitForPerformAction(page, 'upgrade_building', {
      targetHex: { q: 0, r: 0 },
      newBuildingType: BuildingType.TradingHouse,
    })

    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-conversion-worker_to_coin')
    await confirmAction(page)
    await waitForPerformAction(page, 'conversion', { conversionType: 'worker_to_coin', amount: 1 })

    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-upgrade-shipping')
    await confirmAction(page)
    await waitForPerformAction(page, 'advance_shipping', {})

    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-upgrade-digging')
    await confirmAction(page)
    await waitForPerformAction(page, 'advance_digging', {})

    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-burn-power-1')
    await confirmAction(page)
    await waitForPerformAction(page, 'burn_power', { amount: 1 })
  })

  test('favor/town selection and town cult top choice decisions emit expected actions', async ({ page }) => {
    const base = makeBaseGameState({ phase: GamePhase.Action })
    await openGameWithState(page, base)

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: { type: 'favor_tile_selection', playerId: 'p1' },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, `favor-tile-${String(FavorTileType.Water2)}`)
    await confirmAction(page)
    await waitForPerformAction(page, 'select_favor_tile', { tileType: FavorTileType.Water2 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: { type: 'town_tile_selection', playerId: 'p1' },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, `town-tile-${String(TownTileId.Vp7Workers2)}`, { allowForce: true })
    await confirmAction(page)
    await waitForPerformAction(page, 'select_town_tile', { tileType: TownTileId.Vp7Workers2 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: {
          type: 'town_cult_top_choice',
          playerId: 'p1',
          candidateTracks: [CultType.Fire, CultType.Earth],
          maxSelections: 1,
        },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, 'town-cult-top-choice-0')
    await clickByTestId(page, 'town-cult-top-choice-confirm')
    await waitForPerformAction(page, 'select_town_cult_top', { tracks: [CultType.Fire] })
  })

  test('power actions including target hex and pending-spade discard are wired', async ({ page }) => {
    let state = makeBaseGameState({ phase: GamePhase.Action })
    state = withBuildings(state, [
      { q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Nomads, type: BuildingType.Dwelling, terrain: TerrainType.Desert },
    ])

    await openGameWithState(page, state)

    await clickByTestId(page, `power-action-${String(PowerActionType.Priest)}`)
    await confirmAction(page)
    await waitForPerformAction(page, 'power_action_claim', { actionType: PowerActionType.Priest })

    await clearSentMessages(page)
    await clickByTestId(page, `power-action-${String(PowerActionType.Spade)}`)
    await clickHex(page, 1, 0)
    await clickByTestId(page, 'hex-action-submit')
    await waitForPerformAction(page, 'power_action_claim', {
      actionType: PowerActionType.Spade,
      targetHex: { q: 1, r: 0 },
      buildDwelling: true,
      targetTerrain: TerrainType.Desert,
    })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingSpades: { p1: 1 },
        pendingSpadeBuildAllowed: { p1: false },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, 'discard-pending-spade')
    await waitForPerformAction(page, 'discard_pending_spade', { count: 1 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        pendingCultRewardSpades: { p1: 1 },
      },
    })
    await clearSentMessages(page)
    await clickHex(page, 1, 0)
    await page.getByTestId('hex-action-target-terrain').selectOption(String(TerrainType.Wasteland))
    await clickByTestId(page, 'hex-action-submit')
    await waitForPerformAction(page, 'use_cult_spade', {
      hex: { q: 1, r: 0 },
      targetTerrain: TerrainType.Wasteland,
    })
  })

  test('pass action and bonus-card special actions are wired', async ({ page }) => {
    const state = makeBaseGameState({
      phase: GamePhase.Action,
      bonusCards: {
        available: {
          [BonusCardType.Priest]: 0,
          [BonusCardType.Shipping]: 0,
          [BonusCardType.DwellingVP]: 0,
          [BonusCardType.WorkerPower]: 0,
          [BonusCardType.Spade]: 0,
          [BonusCardType.TradingHouseVP]: 0,
          [BonusCardType.Coins6]: 0,
          [BonusCardType.CultAdvance]: 0,
          [BonusCardType.StrongholdSanctuaryVP]: 0,
          [BonusCardType.ShippingVP]: 0,
        },
        playerCards: { p1: BonusCardType.Spade },
        playerHasCard: { p1: true },
      },
    })

    await openGameWithState(page, state)
    await clickByTestId(page, 'passing-card-6')
    await confirmAction(page)
    await waitForPerformAction(page, 'pass', { bonusCard: BonusCardType.Coins6 })

    await clearSentMessages(page)
    await clickByTestId(page, 'passing-card-4')
    await clickHex(page, 2, 0)
    await clickByTestId(page, 'hex-action-submit')
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.BonusCardSpade,
      targetHex: { q: 2, r: 0 },
      buildDwelling: true,
      targetTerrain: TerrainType.Desert,
    })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...state,
        bonusCards: {
          ...(state.bonusCards ?? {}),
          playerCards: { p1: BonusCardType.CultAdvance },
        },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, 'passing-card-7')
    await clickByTestId(page, `cult-choice-${String(CultType.Earth)}`)
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.BonusCardCultAdvance,
      cultTrack: CultType.Earth,
    })
  })

  test('special stronghold/square actions emit correct payloads', async ({ page }) => {
    const submitHexModalIfPresent = async (): Promise<void> => {
      const submit = page.getByTestId('hex-action-submit').first()
      const visible = await submit.isVisible().catch(() => false)
      if (visible) {
        await submit.click()
      }
    }

    let giantsState = makeBaseGameState({
      players: {
        ...makeBaseGameState().players,
        p1: {
          ...makeBaseGameState().players.p1,
          faction: FactionType.Giants,
          hasStrongholdAbility: true,
          specialActionsUsed: {},
        },
      },
    })

    giantsState = withBuildings(giantsState, [
      { q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Giants, type: BuildingType.Stronghold, terrain: TerrainType.Wasteland },
    ])

    await openGameWithState(page, giantsState)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await expect(page.getByTestId('player-p1-stronghold-action')).toHaveClass(/pb-special-action-active/)
    await clickHex(page, 1, 0)
    await clickByTestId(page, 'hex-action-submit')
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.GiantsTransform,
      targetHex: { q: 1, r: 0 },
      buildDwelling: true,
      targetTerrain: TerrainType.Wasteland,
    })

    const engineersState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Engineers,
            hasStrongholdAbility: false,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Engineers, type: BuildingType.Dwelling }],
    )

    await emitWs(page, { type: 'game_state_update', payload: engineersState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-engineers-bridge')
    await clickHex(page, 0, 0)
    await clickHex(page, 1, 0)
    await confirmAction(page)
    await waitForPerformAction(page, 'engineers_bridge', {
      bridgeHex1: { q: 0, r: 0 },
      bridgeHex2: { q: 1, r: 0 },
    })

    const mermaidsState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Mermaids,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Mermaids, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: mermaidsState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-mermaids-connect')
    await clickHex(page, 1, 1)
    await confirmAction(page)
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.MermaidsRiverTown,
      targetHex: { q: 1, r: 1 },
    })

    const aurenState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Auren,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Auren, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: aurenState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await clickByTestId(page, `cult-choice-${String(CultType.Earth)}`)
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.AurenCultAdvance,
      cultTrack: CultType.Earth,
    })

    const witchesState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Witches,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Witches, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: witchesState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await clickHex(page, 1, 0)
    await submitHexModalIfPresent()
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.WitchesRide,
      targetHex: { q: 1, r: 0 },
    })

    const swarmlingsState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Swarmlings,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Swarmlings, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: swarmlingsState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await clickHex(page, 1, 0)
    await submitHexModalIfPresent()
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.SwarmlingsUpgrade,
      upgradeHex: { q: 1, r: 0 },
    })

    const nomadsState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.Nomads,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Nomads, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: nomadsState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await clickHex(page, 1, 0)
    await submitHexModalIfPresent()
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.NomadsSandstorm,
      targetHex: { q: 1, r: 0 },
      buildDwelling: true,
      targetTerrain: TerrainType.Desert,
    })

    const chaosState = withBuildings(
      makeBaseGameState({
        players: {
          ...makeBaseGameState().players,
          p1: {
            ...makeBaseGameState().players.p1,
            faction: FactionType.ChaosMagicians,
            hasStrongholdAbility: true,
          },
        },
      }),
      [{ q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.ChaosMagicians, type: BuildingType.Stronghold }],
    )

    await emitWs(page, { type: 'game_state_update', payload: chaosState })
    await clearSentMessages(page)
    await clickByTestId(page, 'player-p1-stronghold-action')
    await clickByTestId(page, 'chaos-double-turn-submit')
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.ChaosMagiciansDoubleTurn,
      firstAction: {
        type: 'transform_build',
        params: {},
      },
      secondAction: {
        type: 'transform_build',
        params: {},
      },
    })
  })

  test('water2, darklings ordination, cultists choice, and halflings decisions are wired', async ({ page }) => {
    const base = makeBaseGameState({
      players: {
        ...makeBaseGameState().players,
        p1: {
          ...makeBaseGameState().players.p1,
          faction: FactionType.Darklings,
        },
      },
      favorTiles: {
        ...makeBaseGameState().favorTiles,
        playerTiles: {
          p1: [FavorTileType.Water2],
          p2: [],
          p3: [],
          p4: [],
        },
      },
    })

    await openGameWithState(page, base)

    await page.getByTestId('player-p1-water2-action').evaluate((node) => {
      (node as HTMLButtonElement).click()
    })
    await expect(page.getByTestId(`cult-choice-${String(CultType.Water)}`)).toBeVisible()
    await clickByTestId(page, `cult-choice-${String(CultType.Water)}`)
    await waitForPerformAction(page, 'special_action_use', {
      specialActionType: SpecialActionType.Water2CultAdvance,
      cultTrack: CultType.Water,
    })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: {
          type: 'darklings_ordination',
          playerId: 'p1',
        },
      },
    })
    await expect(page.getByTestId('darklings-ordination-2')).toBeVisible()
    await expect(page.getByText('Darklings can only convert workers to priests when a player just upgraded to a stronghold.')).toBeVisible()
    await clearSentMessages(page)
    await clickByTestId(page, 'darklings-ordination-2')
    await waitForPerformAction(page, 'darklings_ordination', { workersToConvert: 2 })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: {
          type: 'cultists_cult_choice',
          playerId: 'p1',
        },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, `cultists-cult-choice-${String(CultType.Air)}`)
    await waitForPerformAction(page, 'select_cultists_track', { cultTrack: CultType.Air })

    await emitWs(page, {
      type: 'game_state_update',
      payload: {
        ...base,
        pendingDecision: {
          type: 'halflings_spades',
          playerId: 'p1',
        },
        pendingHalflingsSpades: {
          spadesRemaining: 0,
          transformedHexes: [{ q: 1, r: 0 }, { q: 2, r: 0 }],
        },
      },
    })
    await clearSentMessages(page)
    await clickByTestId(page, 'halflings-build-1-0')
    await waitForPerformAction(page, 'halflings_build_dwelling', { targetHex: { q: 1, r: 0 } })

    await clearSentMessages(page)
    await clickByTestId(page, 'halflings-skip-dwelling')
    await waitForPerformAction(page, 'halflings_skip_dwelling', {})
  })

  test('game page sends initial get_game_state request', async ({ page }) => {
    await installMockWebSocket(page, 'p1')
    await page.goto('/game/test-game')
    await waitForSocketReady(page)
    await waitForMessageType(page, 'get_game_state')
  })
})
