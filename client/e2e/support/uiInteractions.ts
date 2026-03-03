import { expect, type Locator, type Page } from '@playwright/test'
import { BASE_GAME_MAP } from '../../src/data/baseGameMap'

const HEX_SIZE = 35
const HEX_WIDTH = Math.cos(Math.PI / 6) * HEX_SIZE * 2
const HEX_HEIGHT = Math.sin(Math.PI / 6) * HEX_SIZE + HEX_SIZE

const hexCenter = (r: number, q: number): { x: number; y: number } => {
  const oddRowOffset = r % 2 ? HEX_WIDTH / 2 : 0
  const progressiveOffset = Math.floor(r / 2) * HEX_WIDTH
  return {
    x: 5 + HEX_SIZE + q * HEX_WIDTH + oddRowOffset + progressiveOffset,
    y: 5 + HEX_SIZE + r * HEX_HEIGHT,
  }
}

const boardDimensions = (): { width: number; height: number; offsetX: number; offsetY: number } => {
  let minX = Number.POSITIVE_INFINITY
  let maxX = Number.NEGATIVE_INFINITY
  let minY = Number.POSITIVE_INFINITY
  let maxY = Number.NEGATIVE_INFINITY

  BASE_GAME_MAP.forEach((hex) => {
    const center = hexCenter(hex.coord.r, hex.coord.q)
    minX = Math.min(minX, center.x)
    maxX = Math.max(maxX, center.x)
    minY = Math.min(minY, center.y)
    maxY = Math.max(maxY, center.y)
  })

  const paddingX = HEX_SIZE
  const paddingY = HEX_SIZE * 2

  return {
    width: maxX - minX + paddingX * 2,
    height: maxY - minY + paddingY * 2,
    offsetX: -minX + paddingX,
    offsetY: -minY + paddingY,
  }
}

export async function clickHex(page: Page, q: number, r: number): Promise<void> {
  const canvas = page.getByTestId('hex-grid-canvas')
  await canvas.scrollIntoViewIfNeeded()
  await expect(canvas).toBeVisible()
  const box = await canvas.boundingBox()
  if (!box) {
    throw new Error('Hex canvas bounding box unavailable')
  }

  const dims = boardDimensions()
  const center = hexCenter(r, q)
  const internalX = center.x + dims.offsetX
  const internalY = center.y + dims.offsetY

  const clickX = (internalX / dims.width) * box.width
  const clickY = (internalY / dims.height) * box.height

  await canvas.click({
    force: true,
    timeout: 10_000,
    position: {
      x: clickX,
      y: clickY,
    },
  })
}

export async function clickCultSpot(page: Page, cultIndex: number, tileIndex: number): Promise<void> {
  const canvas = page.getByTestId('cult-tracks-canvas')
  await expect(canvas).toBeVisible()

  const cultWidth = 250 / 4
  const tileWidth = 25
  const tileHeight = 20
  const tileSpacing = 5
  const gridWidth = tileWidth * 2 + tileSpacing
  const startX = (cultWidth - gridWidth) / 2

  const tileOriginX = (() => {
    if (tileIndex < 4) {
      const col = tileIndex % 2
      return cultIndex * cultWidth + startX + col * (tileWidth + tileSpacing)
    }
    return cultIndex * cultWidth + startX + tileWidth / 2 + tileSpacing / 2
  })()

  const tileOriginY = (() => {
    if (tileIndex < 4) {
      const row = Math.floor(tileIndex / 2)
      return 460 + row * (tileHeight + tileSpacing)
    }
    return 460 + 2 * (tileHeight + tileSpacing)
  })()

  // CultTracks converts mouse CSS coordinates with `* 2` internally, so click in the
  // unscaled logical space rather than scaling by rendered bounding box.
  const clickX = tileOriginX + tileWidth / 2
  const clickY = tileOriginY + tileHeight / 2

  await canvas.click({
    force: true,
    timeout: 10_000,
    position: {
      x: clickX,
      y: clickY,
    },
  })
}

export async function clickWhenVisible(locator: Locator): Promise<void> {
  await expect(locator).toBeVisible()
  await locator.click()
}

export async function clickByTestId(page: Page, testId: string): Promise<void> {
  const locator = page.getByTestId(testId).first()
  await locator.waitFor({ state: 'attached', timeout: 10_000 })
  await locator.scrollIntoViewIfNeeded().catch(() => undefined)
  const clicked = await locator.click({ timeout: 3_000, force: true }).then(() => true).catch(() => false)
  if (clicked) return

  await Promise.race([
    locator.evaluate((el) => {
      (el as HTMLElement).click()
    }),
    new Promise<never>((_, reject) => {
      setTimeout(() => reject(new Error(`timeout dispatching click for data-testid=${testId}`)), 3_000)
    }),
  ])
}

export async function confirmAction(page: Page): Promise<void> {
  const confirm = page.getByTestId('confirm-action-confirm').first()
  const visible = await confirm.isVisible().catch(() => false)
  if (visible) {
    await confirm.click()
  }
}
