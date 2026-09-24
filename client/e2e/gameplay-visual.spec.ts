import { expect, test } from '@playwright/test'
import { writeFile } from 'node:fs/promises'
import { makeBaseGameState, withBuildings } from './support/gameStateFactory'
import { BuildingType, FactionType, GamePhase, TerrainType } from '../src/types/game.types'
import { emitWs, installMockWebSocket, waitForSocketReady } from './support/mockWebSocket'

for (const width of [1280, 1920]) {
  for (const playerCount of [2, 4]) {
    test(`gameplay baseline ${width}px ${playerCount} players`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: width === 1280 ? 720 : 1080 })
      const errors: string[] = []
      page.on('pageerror', error => errors.push(error.message))
      page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
      await installMockWebSocket(page)
      await page.goto('/game/test-game')
      await waitForSocketReady(page)
      const state = withBuildings(makeBaseGameState(), [
        { q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Nomads, type: BuildingType.Dwelling, terrain: TerrainType.Desert },
        { q: 3, r: 0, ownerPlayerId: 'p2', faction: FactionType.Darklings, type: BuildingType.TradingHouse, terrain: TerrainType.Swamp },
      ])
      state.scoringTiles = { priestsSent: {}, tiles: [0, 2, 4, 5, 7, 8].map(type => ({
        type, actionType: 0, actionVP: 2, cultTrack: 0, cultThreshold: 4, cultRewardType: 0, cultRewardAmount: 1,
      })) }
      state.turnOrder = state.turnOrder.slice(0, playerCount)
      state.players = Object.fromEntries(state.turnOrder.map(id => [id, state.players[id]]))
      await emitWs(page, { type: 'game_state_update', payload: state })
      await expect(page.getByTestId('hex-grid-canvas')).toBeVisible()
      await page.addStyleTag({ content: '*, *::before, *::after { animation: none !important; transition: none !important; }' })
      await page.evaluate(() => document.fonts.ready)
      await expect(page.locator('.react-grid-item')).toHaveCount(8)
      await expect(page.getByTestId('game-decision-strip')).toHaveCSS('position', 'static')
      await expect(page.getByTestId('game-board')).toHaveCSS('display', 'flex')
      // Let the grid and canvas ResizeObservers settle before capturing geometry.
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(resolve)))))
      const bounds = async (name: string) => {
        const box = await page.locator(`[data-panel="${name}"]`).boundingBox()
        expect(box).not.toBeNull()
        return box!
      }
      const board = await bounds('board')
      const scoring = await bounds('scoring')
      const cult = await bounds('cult')
      expect(scoring.x + scoring.width).toBeLessThan(board.x)
      expect(board.x + board.width).toBeLessThan(cult.x)
      expect((await bounds('summary')).y).toBeLessThan(board.y)
      expect((await bounds('playerBoards')).y).toBeGreaterThan(board.y + board.height)
      expect((await bounds('towns')).y).toBeGreaterThan(scoring.y)
      expect((await bounds('favor')).y).toBeGreaterThan(cult.y)
      // Visible wrappers alone miss clipped cubes and overflowing town stacks.
      const assertSidebarFits = async () => {
        const heights = await page.locator('.scoring-tile').evaluateAll(tiles => tiles.map(tile => tile.getBoundingClientRect().height))
        expect(Math.max(...heights) - Math.min(...heights)).toBeLessThan(1)
        for (const panel of ['scoring', 'towns']) {
          const overflow = await page.locator(`[data-panel="${panel}"]`).evaluate(root => {
            const viewport = root.querySelector(':scope > .drag-handle + div')!.getBoundingClientRect()
            const selector = '.scoring-tile, .content-row > *, .cult-dot, .town-tile'
            return Array.from(root.querySelectorAll(selector)).flatMap(element => {
              const box = element.getBoundingClientRect()
              return box.left < viewport.left - 1 || box.right > viewport.right + 1
                || box.top < viewport.top - 1 || box.bottom > viewport.bottom + 1
                ? [element.className] : []
            })
          })
          expect(overflow, `${panel} artwork stays inside its viewport`).toEqual([])
        }
      }
      await assertSidebarFits()
      const playerPanel = await bounds('playerBoards')
      const playerContent = (await page.locator('.player-boards-container').boundingBox())!
      expect(playerPanel.height - playerContent.height).toBeLessThan(100)
      const panelBounds = await page.locator('.react-grid-item').evaluateAll(items => items.map(item => {
        const { x, y, width, height } = item.getBoundingClientRect()
        return { x, y, width, height }
      }))
      await page.screenshot({ path: testInfo.outputPath('gameplay.png'), fullPage: true, animations: 'disabled' })
      await expect(page).toHaveScreenshot(`gameplay-${width}-${playerCount}.png`, { fullPage: true, animations: 'disabled', maxDiffPixels: 150 })
      await writeFile(testInfo.outputPath('panel-bounds.json'), JSON.stringify(panelBounds, null, 2))
      await page.getByTestId('layout-lock-toggle').click()
      await expect(page.locator('.layout')).toHaveClass(/layout-locked/)
      await assertSidebarFits()
      await page.getByTestId('layout-lock-toggle').click()
      await expect(page.locator('.layout')).not.toHaveClass(/layout-locked/)
      // Exercise the real grid handles, then verify reset restores this panel.
      const scoringPanel = page.locator('[data-panel="scoring"]')
      const originalWidth = (await scoringPanel.boundingBox())!.width
      const resize = scoringPanel.locator('.react-resizable-handle-e')
      await resize.scrollIntoViewIfNeeded()
      const resizeBox = (await resize.boundingBox())!
      await page.mouse.move(resizeBox.x + resizeBox.width / 2, resizeBox.y + resizeBox.height / 2)
      await page.mouse.down()
      await page.mouse.move(resizeBox.x + 110, resizeBox.y + resizeBox.height / 2, { steps: 10 })
      await page.mouse.up()
      await expect.poll(async () => (await scoringPanel.boundingBox())!.width).toBeGreaterThan(originalWidth)
      const handle = scoringPanel.locator('.drag-handle')
      await handle.scrollIntoViewIfNeeded()
      const dragBox = (await handle.boundingBox())!
      const oldTransform = await scoringPanel.evaluate(el => el.style.transform)
      await page.mouse.move(dragBox.x + 20, dragBox.y + dragBox.height / 2)
      await page.mouse.down()
      await page.mouse.move(dragBox.x + 320, dragBox.y + dragBox.height / 2, { steps: 10 })
      await page.mouse.up()
      await expect.poll(() => scoringPanel.evaluate(el => el.style.transform)).not.toBe(oldTransform)
      await page.screenshot({ path: testInfo.outputPath('resized-and-dragged.png'), fullPage: true })
      await page.getByTestId('layout-lock-toggle').click()
      await expect(scoringPanel).not.toHaveClass(/react-draggable/)
      await page.getByTestId('layout-lock-toggle').click()
      await page.getByTestId('layout-reset').click()
      await expect.poll(async () => Math.round((await scoringPanel.boundingBox())!.width)).toBe(Math.round(originalWidth))
      await expect.poll(async () => Math.round((await scoringPanel.boundingBox())!.x)).toBe(Math.round(scoring.x))
      const playersPanel = page.locator('[data-panel="playerBoards"]')
      const playersWidth = (await playersPanel.boundingBox())!.width
      const playersResize = playersPanel.locator('.react-resizable-handle-e')
      await playersResize.scrollIntoViewIfNeeded()
      const playersResizeBox = (await playersResize.boundingBox())!
      await page.mouse.move(playersResizeBox.x + playersResizeBox.width / 2, playersResizeBox.y + playersResizeBox.height / 2)
      await page.mouse.down()
      await page.mouse.move(playersResizeBox.x - playersWidth * 0.3, playersResizeBox.y + playersResizeBox.height / 2, { steps: 10 })
      await page.mouse.up()
      await expect.poll(async () => (await playersPanel.boundingBox())!.width).toBeLessThan(playersWidth)
      await expect(playersPanel.getByText('p1 (Nomads)', { exact: true })).toBeVisible()
      const conversion = page.getByTestId('player-p1-conversion-worker_to_coin')
      await conversion.scrollIntoViewIfNeeded()
      await expect(conversion).toBeVisible()
      await page.screenshot({ path: testInfo.outputPath('player-boards-resized.png'), fullPage: true, animations: 'disabled' })
      await page.getByTestId('layout-reset').click()
      for (const type of ['setup_dwelling', 'leech_offer', 'favor_tile_selection', 'town_tile_selection']) {
        await emitWs(page, { type: 'game_state_update', payload: {
          ...state,
          phase: type === 'setup_dwelling' ? GamePhase.Setup : GamePhase.Action,
          setupSubphase: type === 'setup_dwelling' ? 'dwellings' : undefined,
          setupDwellingOrder: state.turnOrder, setupDwellingIndex: 0,
          pendingDecision: { type, playerId: 'p1' },
          pendingLeechOffers: type === 'leech_offer' ? { p1: [{ Amount: 2 }] } : {},
          pendingTownFormations: type === 'town_tile_selection' ? { p1: [{ hexes: [{ q: 0, r: 0 }] }] } : {},
        } })
        await expect(page.getByTestId('game-decision-strip')).toBeVisible()
        await page.screenshot({ path: testInfo.outputPath(`${type}.png`), fullPage: true, animations: 'disabled' })
        if (type === 'town_tile_selection') {
          await page.getByTestId('town-tile-0').click()
          await expect(page.getByTestId('town-anchor-0-0')).toBeVisible()
          await page.screenshot({ path: testInfo.outputPath('town-anchor.png'), fullPage: true, animations: 'disabled' })
        }
      }
      expect(errors).toEqual([])
    })
  }
}
