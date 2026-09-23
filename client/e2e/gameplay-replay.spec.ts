import { expect, test } from '@playwright/test'
import { makeBaseGameState } from './support/gameStateFactory'
import { installMockWebSocket } from './support/mockWebSocket'

for (const width of [1280, 1920]) {
  test(`replay keeps its existing panels and read-only controls at ${width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize({ width, height: width === 1280 ? 720 : 1080 })
    await installMockWebSocket(page)
    const state = makeBaseGameState()
    const errors: string[] = []
    page.on('pageerror', error => errors.push(error.message))
      page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
    await page.route('**/api/replay/start', route => route.fulfill({ json: {
      currentIndex: 0, totalActions: 2, logStrings: ['p1 builds a dwelling', 'p2 passes'], logLocations: [], players: state.turnOrder,
    } }))
    await page.route('**/api/replay/state?*', route => route.fulfill({ json: state }))
    await page.route('**/api/replay/next', route => route.fulfill({ json: { ...state, currentTurn: 1 } }))
    await page.goto('/replay/test-game')
    await expect(page.getByTestId('hex-grid-canvas')).toBeVisible()
    await expect(page.locator('[data-panel]')).toHaveCount(9)
    await expect(page.getByTestId('power-action-4')).toBeDisabled()
    await page.getByRole('group', { name: 'Replay controls' }).getByRole('button', { name: 'Next', exact: true }).click()
    await expect(page.getByText('Action: 1 / 2')).toBeVisible()
    await page.getByRole('button', { name: 'Lock Layout', exact: true }).click()
    await expect(page.locator('.layout')).toHaveClass(/layout-locked/)
    await page.screenshot({ path: testInfo.outputPath(`replay-${width}.png`), fullPage: true, animations: 'disabled' })
    await page.getByRole('button', { name: 'Unlock Layout', exact: true }).click()
    await page.getByRole('button', { name: 'Reset Layout', exact: true }).click()
    expect(errors).toEqual([])
    expect(await page.evaluate(() => window.__tmE2E?.performActions())).toEqual([])
  })
}
