import { expect, type Locator, type Page } from '@playwright/test'
import { BASE_GAME_MAP } from '../../src/data/baseGameMap'
import { getPriestSpotRect } from '../../src/components/CultTracks/CultTracks'

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

  const dims = boardDimensions()
  const center = hexCenter(r, q)
  const internalX = center.x + dims.offsetX
  const internalY = center.y + dims.offsetY

  await canvas.evaluate((node, target) => {
    const el = node as HTMLCanvasElement
    let parent = el.parentElement
    while (parent) {
      const style = window.getComputedStyle(parent)
      const scrollableX = (style.overflowX === 'auto' || style.overflowX === 'scroll' || style.overflow === 'auto' || style.overflow === 'scroll')
        && parent.scrollWidth > parent.clientWidth
      const scrollableY = (style.overflowY === 'auto' || style.overflowY === 'scroll' || style.overflow === 'auto' || style.overflow === 'scroll')
        && parent.scrollHeight > parent.clientHeight
      if (scrollableX || scrollableY) {
        const rect = el.getBoundingClientRect()
        const logicalWidth = Number(el.dataset.logicalWidth || el.width)
        const logicalHeight = Number(el.dataset.logicalHeight || el.height)
        const scaleX = rect.width / logicalWidth
        const scaleY = rect.height / logicalHeight
        if (scrollableX) {
          const targetX = target.internalX * scaleX
          parent.scrollLeft = Math.max(0, targetX - parent.clientWidth / 2)
        }
        if (scrollableY) {
          const targetY = target.internalY * scaleY
          parent.scrollTop = Math.max(0, targetY - parent.clientHeight / 2)
        }
      }
      parent = parent.parentElement
    }
  }, { internalX, internalY })

  await page.waitForTimeout(50)

  const geometry = await canvas.evaluate((node) => {
    const el = node as HTMLCanvasElement
    const rect = el.getBoundingClientRect()
    return {
      left: rect.left,
      top: rect.top,
      width: rect.width,
      height: rect.height,
    }
  })

  const clickX = geometry.left + (internalX / dims.width) * geometry.width
  const clickY = geometry.top + (internalY / dims.height) * geometry.height

  await page.mouse.click(clickX, clickY, {
    button: 'left',
  })
}

export async function clickCultSpot(page: Page, cultIndex: number, tileIndex: number): Promise<void> {
  const overlayButton = page.getByTestId(`cult-spot-${String(cultIndex)}-${String(tileIndex)}`).first()
  const overlayVisible = await overlayButton.isVisible().catch(() => false)
  if (overlayVisible) {
    await overlayButton.click({ force: true })
    return
  }

  const canvas = page.getByTestId('cult-tracks-canvas')
  await expect(canvas).toBeVisible()

  const rect = getPriestSpotRect(cultIndex, tileIndex)

  // CultTracks converts mouse CSS coordinates with `* 2` internally, so click in the
  // unscaled logical space rather than scaling by rendered bounding box.
  const clickX = rect.x + rect.width / 2
  const clickY = rect.y + rect.height / 2

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

export async function clickByTestId(page: Page, testId: string, options?: { allowForce?: boolean }): Promise<void> {
  const locator = page.getByTestId(testId).first()
  if (options?.allowForce) {
    await locator.waitFor({ state: 'attached', timeout: 10_000 })
  } else {
    await locator.waitFor({ state: 'visible', timeout: 10_000 })
    await expect(locator).toBeEnabled()
  }
  await locator.scrollIntoViewIfNeeded().catch(() => undefined)
  await locator.evaluate((node) => {
    (node as HTMLElement).scrollIntoView({
      block: 'center',
      inline: 'center',
      behavior: 'instant',
    })
  }).catch(() => undefined)

  const clicked = await locator.click({ timeout: 3_000 }).then(() => true).catch(() => false)
  if (clicked) return

  if (!options?.allowForce) {
    const geometry = await locator.evaluate((node) => {
      const rect = (node as HTMLElement).getBoundingClientRect()
      return {
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height,
      }
    }).catch(() => null)
    if (geometry && geometry.width > 0 && geometry.height > 0) {
      await page.mouse.click(
        geometry.left + geometry.width / 2,
        geometry.top + geometry.height / 2,
      ).then(() => undefined).catch(() => undefined)
      const stillVisible = await locator.isVisible().catch(() => false)
      if (stillVisible === false) return
    }
  }

  if (!options?.allowForce) {
    throw new Error(`unable to click data-testid=${testId} without force`)
  }

  await locator.click({ timeout: 3_000, force: true })
}

export async function confirmAction(page: Page): Promise<void> {
  const confirm = page.getByTestId('confirm-action-confirm').first()
  const visible = await confirm.isVisible().catch(() => false)
  if (visible) {
    await confirm.click()
  }
}
