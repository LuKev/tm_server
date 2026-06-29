import { expect, test } from '@playwright/test'

test.describe('Model opponent game flow (Real Server)', () => {
  test('@smoke starts a playable model game from the AI page', async ({ page }) => {
    await page.goto('/')

    await expect(page.getByTestId('lobby-screen')).toBeVisible()
    await expect(page.locator('.lobby-status-label')).toHaveText('connected', { timeout: 15_000 })
    await expect(page.getByRole('button', { name: /^AI$/ })).toHaveCount(0)

    await page.getByTestId('lobby-play-ai').click()
    await expect(page).toHaveURL(/\/ai$/)
    await expect(page.getByTestId('play-ai-screen')).toBeVisible()
    await expect(page.getByText('Snapshot')).toHaveCount(0)
    await expect(page.getByTestId('ai-human-faction')).toBeVisible()
    await expect(page.getByTestId('ai-model-faction')).toBeVisible()
    await expect(page.getByTestId('ai-model-strength')).toHaveValue('balanced')

    await page.getByTestId('ai-start-game').click()

    await expect(page).toHaveURL(/\/game\/\d+$/, { timeout: 20_000 })
    await expect(page.getByTestId('game-screen')).toBeVisible()
    await expect(page.getByTestId('player-summary-bar')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByText('TM-AZ-', { exact: false }).first()).toBeVisible()
  })
})
