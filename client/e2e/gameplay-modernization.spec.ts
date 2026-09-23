import { expect, test, type Page } from '@playwright/test'
import { PowerActionType, TerrainType, GamePhase, FactionType, type GameState } from '../src/types/game.types'
import { makeBaseGameState } from './support/gameStateFactory'
import { acknowledgeLastAction, clearSentMessages, emitWs, installMockWebSocket, waitForPerformAction, waitForSocketReady } from './support/mockWebSocket'
import { clickHex } from './support/uiInteractions'

async function openGame(page: Page, state = makeBaseGameState()): Promise<void> {
  state.players.p1.resources.power.powerIII = 12
  await installMockWebSocket(page)
  await page.goto('/game/test-game')
  await waitForSocketReady(page)
  await emitWs(page, { type: 'game_state_update', payload: state })
  await expect(page.getByTestId('hex-grid-canvas')).toBeVisible()
}

test('two-spade draft is atomic, ignores duplicate submit, and survives rejection', async ({ page }) => {
  await openGame(page)
  await page.getByTestId(`power-action-${String(PowerActionType.DoubleSpade)}`).click()
  await clickHex(page, 1, 0)
  await page.getByTestId('hex-action-mode').selectOption('transform_only')
  await page.getByTestId('power-spade-second-target').click()
  await clickHex(page, 2, 0)
  await page.getByTestId('power-spade-second-terrain').selectOption(String(TerrainType.Plains))
  await clearSentMessages(page)
  await page.getByTestId('hex-action-submit').click()
  const params = {
    actionType: PowerActionType.DoubleSpade, spadeActionVersion: 2, useCoins: false,
    targetHex: { q: 1, r: 0 }, targetTerrain: TerrainType.Desert, buildDwelling: false,
    secondHex: { q: 2, r: 0 }, secondTerrain: TerrainType.Plains,
  }
  await waitForPerformAction(page, 'power_action_claim', params)
  await expect(page.getByTestId('hex-action-submit')).toBeDisabled()
  await page.getByTestId('hex-action-submit').evaluate(element => (element as HTMLButtonElement).click())
  expect(await page.evaluate(() => window.__tmE2E?.performActions().length)).toBe(1)
  await acknowledgeLastAction(page, 'action_rejected', 'Target is not reachable')
  await expect(page.getByRole('alert')).toContainText('Target is not reachable')
  await expect(page.getByTestId('power-spade-second-terrain')).toHaveValue(String(TerrainType.Plains))
  await page.getByTestId('hex-action-submit').click()
  await acknowledgeLastAction(page)
  await expect(page.getByTestId('hex-action-submit')).toHaveCount(0)
})

test('cancelled spade draft sends nothing and keyboard board selection uses the same flow', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await openGame(page)
  await page.getByTestId(`power-action-${String(PowerActionType.Spade)}`).focus()
  await page.keyboard.press('Enter')
  const board = page.getByTestId('hex-grid-canvas')
  await board.focus()
  await page.keyboard.press('Home')
  await page.keyboard.press('ArrowRight')
  await page.keyboard.press('Enter')
  await expect(page.getByTestId('power-spade-first-target')).toContainText('A2')
  await clearSentMessages(page)
  await page.getByTestId('hex-action-cancel').click()
  expect(await page.evaluate(() => window.__tmE2E?.performActions())).toEqual([])
  await expect(page.getByTestId(`power-action-${String(PowerActionType.Spade)}`)).toHaveAttribute('aria-pressed', 'false')

  // Keep keyboard selection safe when replacing a larger map while focused.
  await page.getByTestId(`power-action-${String(PowerActionType.Spade)}`).click()
  await board.focus()
  await page.keyboard.press('End')
  const replacement = makeBaseGameState()
  replacement.players.p1.resources.power.powerIII = 12
  replacement.map.hexes = { '0,0': replacement.map.hexes['0,0'] }
  await emitWs(page, { type: 'game_state_update', payload: replacement })
  await board.focus()
  await page.keyboard.press('ArrowLeft')
  await page.keyboard.press('Enter')
  await expect(page.getByTestId('power-spade-first-target')).toContainText('A1')
  expect(errors).toEqual([])
})

test('end scoring dialog contains focus, closes with Escape and can reopen', async ({ page }, testInfo) => {
  const state = makeBaseGameState({ phase: GamePhase.End, finished: true })
  state.finalScoring = Object.fromEntries(Object.values(state.players).map((player, index) => [player.id, {
    playerId: player.id, playerName: player.name, baseVp: 60, areaVp: 18, cultVp: 16,
    resourceVp: 4, totalVp: 98 - index, largestAreaSize: 10, totalResourceValue: 12,
    fireIceVp: 0, fireIceMetricValue: 0,
  }])) as GameState['finalScoring']
  await openGame(page, state)
  const dialog = page.getByRole('dialog', { name: 'Final Scoring' })
  await expect(dialog).toBeVisible()
  await expect(dialog).toHaveAttribute('open', '')
  await expect(dialog.locator('.modal-container')).toHaveCSS('opacity', '1')
  await expect(dialog.locator('.modal-container')).toHaveCSS('background-color', 'rgb(255, 255, 255)')
  await page.screenshot({ path: testInfo.outputPath('final-scoring.png'), animations: 'disabled' })
  for (let index = 0; index < 5; index++) {
    await page.keyboard.press('Tab')
    expect(await dialog.evaluate(element => element.contains(document.activeElement))).toBe(true)
  }
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  const reopen = page.getByRole('button', { name: 'View final scoring' })
  await reopen.click()
  await expect(dialog).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(reopen).toBeFocused()
})

test('cult spaces and tile rewards have accessible names', async ({ page }) => {
  await openGame(page)
  await expect(page.getByTestId('cult-spot-0-0')).toHaveAccessibleName(/Fire cult/)
  await expect(page.getByTestId('favor-tile-0')).toHaveAccessibleName(/favor/)
  await expect(page.getByTestId('passing-card-0')).toHaveAccessibleName(/bonus card/)
})


test('disconnect releases pending spade submission and keeps choices for reconnect', async ({ page }) => {
  await openGame(page)
  await page.getByTestId('power-action-4').click()
  await clickHex(page, 1, 0)
  await page.getByTestId('hex-action-submit').click()
  await expect(page.getByTestId('hex-action-submit')).toBeDisabled()
  await page.evaluate(() => window.__tmE2E?.sockets.filter(socket => socket.readyState === 1).forEach(socket => socket.close()))
  await expect(page.locator('.power-spade-draft [role="alert"]')).toContainText('Connection lost')
  await expect(page.getByTestId('power-spade-first-target')).toContainText('A2')
  await waitForSocketReady(page)
  await emitWs(page, { type: 'game_state_update', payload: makeBaseGameState() })
  await expect(page.getByTestId('hex-action-submit')).toBeEnabled()
  await page.getByTestId('hex-action-cancel').click()
  await expect(page.locator('.power-spade-draft')).toHaveCount(0)
})

test('spectator can inspect a custom map on a narrow viewport without action controls', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 768, height: 1024 })
  const state = makeBaseGameState()
  state.players.p1.name = 'A very long player name that must remain readable'
  state.map.hexes = {
    '-2,-1': { coord: { q: -2, r: -1 }, terrain: TerrainType.Forest, displayCoord: 'North' },
    '-1,-1': { coord: { q: -1, r: -1 }, terrain: TerrainType.Mountain, displayCoord: 'Center' },
    '-1,0': { coord: { q: -1, r: 0 }, terrain: TerrainType.Desert, displayCoord: 'South' },
  }
  await installMockWebSocket(page, 'spectator')
  await page.goto('/game/test-game')
  await waitForSocketReady(page)
  await emitWs(page, { type: 'game_state_update', payload: state })
  await expect(page.getByTestId('hex-grid-canvas')).toBeVisible()
  await expect(page.getByTestId('power-action-4')).toBeDisabled()
  await expect.poll(async () => {
    const panels = await page.locator('[data-panel]').evaluateAll(elements => elements.map(element => {
      const rect = element.getBoundingClientRect()
      return { name: element.getAttribute('data-panel'), x: rect.x, y: rect.y, right: rect.right, bottom: rect.bottom }
    }))
    const overlaps: string[] = []
    for (let i = 0; i < panels.length; i++) for (let j = i + 1; j < panels.length; j++) {
      const a = panels[i], b = panels[j]
      if (!(a.right <= b.x || b.right <= a.x || a.bottom <= b.y || b.bottom <= a.y)) overlaps.push(`${a.name}/${b.name}`)
    }
    return overlaps
  }).toEqual([])
  await page.screenshot({ path: testInfo.outputPath('narrow-custom-map-spectator.png'), fullPage: true, animations: 'disabled' })
  expect(await page.evaluate(() => window.__tmE2E?.performActions())).toEqual([])
})

test('power action pilot records normal, hover, focus, selected and used states', async ({ page }, testInfo) => {
  await openGame(page)
  const actions = page.getByTestId('power-actions-section')
  const single = page.getByTestId('power-action-4')
  await actions.screenshot({ path: testInfo.outputPath('power-normal.png') })
  await single.hover()
  await actions.screenshot({ path: testInfo.outputPath('power-hover.png') })
  await single.focus()
  await page.keyboard.press('Tab')
  await page.keyboard.press('Shift+Tab')
  await expect(single).toHaveCSS('outline-style', 'solid')
  await actions.screenshot({ path: testInfo.outputPath('power-focus.png') })
  await page.keyboard.press('Enter')
  await expect(single).toHaveAttribute('aria-pressed', 'true')
  await actions.screenshot({ path: testInfo.outputPath('power-selected.png') })
  await page.getByTestId('hex-action-cancel').click()
  const state = makeBaseGameState()
  state.powerActions = { UsedActions: { 4: true } } as GameState['powerActions']
  await emitWs(page, { type: 'game_state_update', payload: state })
  await expect(single).toBeDisabled()
  await actions.screenshot({ path: testInfo.outputPath('power-used.png') })
})


test('Yetis power discount is consistent on the tile and draft', async ({ page }) => {
  const state = makeBaseGameState()
  state.players.p1.faction = FactionType.Yetis
  await openGame(page, state)
  const action = page.getByTestId('power-action-5')
  await expect(action).toHaveAccessibleName(/5 power/)
  await action.click()
  await expect(page.locator('.power-spade-draft')).toContainText('5 power to claim')
})
