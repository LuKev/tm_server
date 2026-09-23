import { expect, test } from '@playwright/test'
import { BonusCardType, GamePhase, TerrainType, type GameState, type MapHex } from '../src/types/game.types'
import { WsBot } from './support/wsBot'
import { realServerWsURL } from './support/realServerConfig'
import { primeRealServerPage, loadRealServerGamePage } from './support/realServerPage'
import { clickHex } from './support/uiInteractions'

const neighbors = (a: MapHex, b: MapHex): boolean => {
  const q = a.coord.q - b.coord.q
  const r = a.coord.r - b.coord.r
  return Math.max(Math.abs(q), Math.abs(r), Math.abs(q + r)) === 1
}

for (const actionType of [4, 5]) {
test(`real server accepts atomic ${actionType === 4 ? 'single' : 'double'} spades and preserves a rejected target`, async ({ page }) => {
  const bots = await Promise.all([WsBot.connect(realServerWsURL()), WsBot.connect(realServerWsURL())])
  const [engineers, witches] = bots
  try {
    engineers.send('create_game', { name: 'modernization-spades', maxPlayers: 2, creator: 'engineers' })
    const created = await engineers.waitForType('game_created')
    const gameID = String((created.payload as { gameId: string }).gameId)
    witches.send('join_game', { id: gameID, name: 'witches' })
    await witches.waitForType('game_joined')
    engineers.send('start_game', { gameID, randomizeTurnOrder: false, setupMode: 'snellman' })
    let state = await engineers.waitForRevision(gameID, 0) as unknown as GameState
    engineers.send('test_apply_fixture_settings', {
      gameID, scoringTiles: ['SCORE5', 'SCORE8', 'SCORE7', 'SCORE9', 'SCORE6', 'SCORE2'],
      bonusCards: ['BON-WP', 'BON-P', 'BON-6C', 'BON-TP', 'BON-SPD'], turnOrderPolicy: 'pass_order',
    })
    await engineers.waitForType('test_command_applied')
    state = await engineers.waitForRevision(gameID, Number(state.revision) + 1) as unknown as GameState
    const act = async (player: string, type: string, params: Record<string, unknown>) => {
      const bot = player === 'engineers' ? engineers : witches
      bot.send('perform_action', { gameID, actionId: `${type}-${String(state.revision)}`, expectedRevision: state.revision, type, params })
      const response = await bot.waitForAnyType(['action_accepted', 'action_rejected', 'error'])
      expect(response.type, JSON.stringify(response)).toBe('action_accepted')
      state = await engineers.waitForRevision(gameID, Number(state.revision) + 1) as unknown as GameState
    }
    const hexes = Object.values(state.map.hexes)
    const targetsFor = (home: MapHex) => hexes.filter(hex => neighbors(home, hex) && [TerrainType.Forest, TerrainType.Wasteland].includes(hex.terrain))
    const home = hexes.find(hex => hex.terrain === TerrainType.Mountain && targetsFor(hex).length >= 2)!
    expect(home).toBeDefined()
    const targets = targetsFor(home).slice(0, 2)
    for (let step = 0; state.phase !== GamePhase.Action && step < 20; step++) {
      if (state.phase === GamePhase.FactionSelection) {
        const player = state.turnOrder[state.currentTurn]
        await act(player, 'select_faction', { faction: player === 'engineers' ? 'Engineers' : 'Witches' })
      } else if (state.setupSubphase === 'bonus_cards') {
        const player = state.setupBonusOrder![state.setupBonusIndex!]
        const card = player === 'engineers' ? BonusCardType.WorkerPower : Number(Object.keys(state.bonusCards!.available).find(key => Number(key) !== BonusCardType.WorkerPower))
        await act(player, 'setup_bonus_card', { bonusCard: card })
      } else {
        const player = state.setupDwellingOrder![state.setupDwellingIndex!]
        const liveHexes = Object.values(state.map.hexes)
        const homeNow = liveHexes.find(hex => hex.coord.q === home.coord.q && hex.coord.r === home.coord.r)!
        const hex = player === 'engineers' && !homeNow.building ? homeNow : liveHexes.find(hex =>
          !hex.building && hex.terrain === (player === 'engineers' ? TerrainType.Mountain : TerrainType.Forest)
          && !targets.some(target => target.coord.q === hex.coord.q && target.coord.r === hex.coord.r))!
        await act(player, 'setup_dwelling', { hex: hex.coord })
      }
    }
    expect(state.phase).toBe(GamePhase.Action)
    const cost = actionType === 4 ? 4 : 6
    await act('engineers', 'burn_power', { amount: cost })
    expect(state.players.engineers.resources.power.powerIII).toBeGreaterThanOrEqual(cost)
    await primeRealServerPage(page, 'engineers')
    await loadRealServerGamePage(page, gameID, 'engineers')
    await page.getByTestId(`power-action-${actionType}`).click()
    // An occupied target is rejected by the actual server, leaving the draft editable.
    await clickHex(page, home.coord.q, home.coord.r)
    await page.getByTestId('hex-action-mode').selectOption('transform_only')
    await page.getByTestId('hex-action-submit').click()
    await expect(page.locator('.power-spade-draft [role="alert"]')).toBeVisible()
    await expect(page.getByTestId('hex-action-submit')).toBeEnabled()
    await clickHex(page, targets[0].coord.q, targets[0].coord.r)
    if (actionType === 5) {
      await page.getByTestId('power-spade-second-target').click()
      await clickHex(page, targets[1].coord.q, targets[1].coord.r)
    }
    const revision = Number(state.revision)
    await page.getByTestId('hex-action-submit').click()
    await expect(page.locator('.power-spade-draft')).toHaveCount(0)
    const after = await engineers.waitForRevision(gameID, revision + 1) as unknown as GameState
    for (const target of targets.slice(0, actionType === 4 ? 1 : 2)) {
      const hex = Object.values(after.map.hexes).find(hex => hex.coord.q === target.coord.q && hex.coord.r === target.coord.r)!
      expect(hex.terrain).toBe(TerrainType.Mountain)
      expect(hex.building).toBeFalsy()
    }
    expect(after.players.engineers.resources.power.powerIII).toBe(state.players.engineers.resources.power.powerIII - cost)
  } finally {
    bots.forEach(bot => bot.close())
  }
})
}
