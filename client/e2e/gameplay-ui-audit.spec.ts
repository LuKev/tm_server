import { expect, test, type Page } from '@playwright/test'
import { makeBaseGameState, withBuildings } from './support/gameStateFactory'
import { BuildingType, FactionType, FavorTileType, PowerActionType, SpecialActionType, TownTileId } from '../src/types/game.types'
import { clearSentMessages, emitWs, installMockWebSocket, waitForPerformAction, waitForSocketReady } from './support/mockWebSocket'
import { FACTION_BOARDS } from '../src/data/factionBoards'
import { BASE_GAME_MAP } from '../src/data/baseGameMap'
import { hexCenter, HEX_SIZE } from '../src/utils/hexUtils'

function auditState() {
  const state = withBuildings(makeBaseGameState(), [
    { q: 0, r: 0, ownerPlayerId: 'p1', faction: FactionType.Nomads, type: BuildingType.Dwelling },
    { q: 3, r: 0, ownerPlayerId: 'p2', faction: FactionType.Darklings, type: BuildingType.Sanctuary },
  ])
  state.turnOrder.forEach((id, index) => {
    state.favorTiles!.playerTiles[id] = [index * 3, index * 3 + 1, index * 3 + 2] as FavorTileType[]
    state.players[id].townTiles = [index * 2, index * 2 + 1]
  })
  state.players.p1.options!.confirmActions = false
  Object.assign(state.map.hexes['0,0'], { hasTownTile: true, townTileType: TownTileId.Vp5Coins6, townTileOwnerPlayerId: 'p1' })
  state.scoringTiles!.tiles = [0, 1, 4, 5, 7, 8].map(type => ({ type, actionType: 0, actionVP: 2, cultTrack: 0, cultThreshold: 4, cultRewardType: 0, cultRewardAmount: 1 }))
  return state
}

async function openGame(page: Page) {
  await installMockWebSocket(page)
  await page.goto('/game/test-game')
  await waitForSocketReady(page)
  await emitWs(page, { type: 'game_state_update', payload: auditState() })
  await expect(page.locator('.pb-favor-tile .favor-tile-face')).toHaveCount(12)
  await page.evaluate(() => document.fonts.ready)
}

for (const width of [1280, 1920]) {
  test(`tile artwork and resource numerals fit at ${width}px and after panel resize`, async ({ page }, testInfo) => {
    await page.setViewportSize({ width, height: 1080 })
    const errors: string[] = []
    page.on('pageerror', error => errors.push(error.message))
    await openGame(page)
    const checkGeometry = async () => {
      const overflow = await page.locator('.favor-tile-face').evaluateAll(faces => faces.flatMap(face => {
        const outer = face.getBoundingClientRect()
        const problems = Array.from(face.querySelectorAll('.cult-step-circle, .favor-reward-artwork, .favor-reward-artwork svg, .favor-reward-artwork span')).flatMap(child => {
          const box = child.getBoundingClientRect()
          const corners = [[box.left, box.top], [box.right, box.top], [box.left, box.bottom], [box.right, box.bottom]]
          const outsideEllipse = corners.some(([x, y]) => ((x - (outer.left + outer.width / 2)) / (outer.width * 0.485)) ** 2 + ((y - (outer.top + outer.height / 2)) / (outer.height * 0.47)) ** 2 > 1.02)
          return outsideEllipse || box.left < outer.left + 1 || box.right > outer.right - 1 || box.top < outer.top + 1 || box.bottom > outer.bottom - 1
            ? [(face as HTMLElement).dataset.favor] : []
        })
        return problems
      }))
      expect(overflow, 'All supply and owned rewards stay inside their tile').toEqual([])
      const clippedBonus = await page.locator('.passing-tile').evaluateAll(tiles => tiles.flatMap(tile => {
        const outer = tile.getBoundingClientRect()
        return Array.from(tile.querySelectorAll('svg, .vp-number, .passing-tile-arrow')).flatMap(child => {
          const box = child.getBoundingClientRect()
          return box.left < outer.left - 1 || box.right > outer.right + 1 || box.top < outer.top - 1 || box.bottom > outer.bottom + 1 ? [tile.parentElement?.getAttribute('data-testid')] : []
        })
      }))
      expect(clippedBonus, 'Bonus artwork stays inside its card').toEqual([])
      const actionOverflow = await page.getByTestId('power-actions-section').evaluate(section => {
        const viewport = section.closest('[data-panel="board"]')!.getBoundingClientRect()
        return Array.from(section.querySelectorAll('.power-action-tile, .octagon-wrapper, .octagon-content > *')).flatMap(el => {
          const box = el.getBoundingClientRect()
          return box.bottom > viewport.bottom - 4 || box.top < viewport.top ? [el.className] : []
        })
      })
      expect(actionOverflow, 'All power action borders and symbols remain visible').toEqual([])
      const numeralOverflow = await page.locator('svg[data-icon="coin"], svg[data-icon="worker"], svg[data-icon="power"], svg[data-icon="priest"]').evaluateAll(icons => icons.flatMap(icon => {
        const text = icon.querySelector('text') as SVGGraphicsElement | null
        if (!text?.textContent) return []
        const box = text.getBBox()
        const bounds = (icon as SVGSVGElement).viewBox.baseVal
        return box.x < 0 || box.y < 0 || box.x + box.width > bounds.width || box.y + box.height > bounds.height ? [text.textContent] : []
      }))
      expect(numeralOverflow, 'Resource numerals scale with their SVG').toEqual([])
    }
    await checkGeometry()
    await expect(page.getByTestId(`power-action-${PowerActionType.Coins}`).locator('svg[data-icon="coin"] text')).toHaveText('7')
    const darklings = page.getByTestId('player-board-p2')
    await expect(darklings.locator('.cost-indicator svg[data-icon="coin"] text', { hasText: '10' })).toHaveCount(1)
    await page.screenshot({ path: testInfo.outputPath('all-tile-variants.png'), fullPage: true })
    const panel = page.locator('[data-panel="playerBoards"]')
    const original = (await panel.boundingBox())!
    await panel.scrollIntoViewIfNeeded()
    const dragWidth = async (delta: number) => {
      const handle = (await panel.locator('.react-resizable-handle-e').boundingBox())!
      await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2)
      await page.mouse.down()
      await page.mouse.move(handle.x + handle.width / 2 + delta, handle.y + handle.height / 2, { steps: 10 })
      await page.mouse.up()
    }
    await dragWidth(-original.width * 0.3)
    await expect.poll(async () => (await panel.boundingBox())!.width).toBeLessThan(original.width)
    await checkGeometry()
    await page.screenshot({ path: testInfo.outputPath('player-panel-narrow.png'), fullPage: true })
    expect(errors).toEqual([])
  })
}

test('town hover and keyboard preview show the reward without sending an action', async ({ page }) => {
  await openGame(page)
  await clearSentMessages(page)
  const canvas = page.getByTestId('hex-grid-canvas')
  await canvas.scrollIntoViewIfNeeded()
  const centers = BASE_GAME_MAP.map(hex => hexCenter(hex.coord.r, hex.coord.q))
  const offsetX = -Math.min(...centers.map(center => center.x)) + HEX_SIZE
  const offsetY = -Math.min(...centers.map(center => center.y)) + HEX_SIZE * 2
  const center = hexCenter(0, 0)
  const box = (await canvas.boundingBox())!
  await canvas.hover({ position: {
    x: (center.x + offsetX) / Number(await canvas.getAttribute('data-logical-width')) * box.width,
    y: (center.y + offsetY) / Number(await canvas.getAttribute('data-logical-height')) * box.height,
  } })
  await expect(page.getByRole('tooltip')).toContainText('5 VP · 6 coins')
  await expect(page.getByRole('tooltip').getByTestId('town-face-0')).toBeVisible()
  await page.getByTestId('hex-markings-toggle').hover()
  await expect(page.getByRole('tooltip')).toHaveCount(0)
  await canvas.focus()
  await page.keyboard.press('Home')
  await expect(page.getByRole('tooltip')).toContainText('5 VP · 6 coins')
  expect(await page.evaluate(() => window.__tmE2E?.performActions())).toEqual([])
})

test('depleted town supply stays empty and the shared Water favor retains its action', async ({ page }) => {
  await openGame(page)
  const state = auditState()
  state.townTiles!.available = {}
  state.favorTiles!.playerTiles.p1 = [FavorTileType.Water2]
  await emitWs(page, { type: 'game_state_update', payload: state })
  await expect(page.locator('[data-panel="towns"] .town-tile-slot')).toHaveCount(0)
  await page.getByTestId('player-p1-water2-action').click()
  await page.getByTestId('cult-choice-0').click()
  await waitForPerformAction(page, 'special_action_use', { specialActionType: SpecialActionType.Water2CultAdvance, cultTrack: 0 })
})


test('every faction board keeps building income icons inside their borders', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1280, height: 1080 })
  await openGame(page)
  const factions = Object.keys(FACTION_BOARDS).map(Number).filter(faction => faction !== FactionType.Unknown)
  for (let start = 0; start < factions.length; start += 4) {
    const state = auditState()
    state.turnOrder.forEach((id, index) => { state.players[id].faction = factions[start + index] ?? FactionType.Nomads })
    await emitWs(page, { type: 'game_state_update', payload: state })
    await expect(page.getByTestId('player-board-p1')).toContainText(FactionType[factions[start]])
    const clipped = await page.locator('.building-slot').evaluateAll(slots => slots.flatMap(slot => {
      const outer = slot.getBoundingClientRect()
      return Array.from(slot.querySelectorAll('.income-reveal > svg')).flatMap(icon => {
        const box = icon.getBoundingClientRect()
        return box.left < outer.left || box.right > outer.right || box.top < outer.top || box.bottom > outer.bottom
          ? [slot.closest('.player-board-section')?.getAttribute('data-testid')] : []
      })
    }))
    expect(clipped, `Income in factions ${factions.slice(start, start + 4).map(faction => FactionType[faction]).join(', ')}`).toEqual([])
    await page.locator('[data-panel="playerBoards"]').screenshot({ path: testInfo.outputPath(`factions-${start}.png`) })
  }
})

test('all scoring variants and Fire and Ice labels fit their divided sections', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1280, height: 1080 })
  await openGame(page)
  for (const [index, tile] of (['distance', 'stronghold_sanctuary', 'edge', 'cluster'] as const).entries()) {
    const state = auditState()
    state.fireIceFinalScoringSetting = 'on'
    state.fireIceFinalScoringTile = tile
    state.scoringTiles!.tiles = (index === 0 ? [0, 1, 2, 3, 4, 5] : [3, 4, 5, 6, 7, 8]).map(type => ({ type, actionType: 0, actionVP: 2, cultTrack: 0, cultThreshold: 4, cultRewardType: 0, cultRewardAmount: 1 }))
    await emitWs(page, { type: 'game_state_update', payload: state })
    await expect(page.locator('.fire-ice-scoring-tile')).toBeVisible()
    const overflow = await page.locator('.scoring-tile .tile-section').evaluateAll(sections => sections.flatMap(section => {
      const outer = section.getBoundingClientRect()
      return Array.from(section.querySelectorAll('.content-row > *, .fire-ice-scoring-content')).flatMap(child => {
        const box = child.getBoundingClientRect()
        return box.left < outer.left || box.right > outer.right || box.top < outer.top || box.bottom > outer.bottom ? [child.className] : []
      })
    }))
    expect(overflow, `${tile} scoring content fits between the borders`).toEqual([])
    await page.locator('[data-panel="scoring"]').screenshot({ path: testInfo.outputPath(`scoring-${tile}.png`) })
  }
})
