import { expect, test } from '@playwright/test'

test.describe('Model opponent game flow (Real Server)', () => {
  test('@smoke starts a playable model game from the lobby', async ({ page }) => {
    await page.goto('/')

    await expect(page.getByTestId('lobby-screen')).toBeVisible()
    await expect(page.locator('.lobby-status-label')).toHaveText('connected', { timeout: 15_000 })

    await page.getByTestId('lobby-player-name').fill('human')
    await page.getByTestId('lobby-opponent-type').selectOption('model')
    await expect(page.getByTestId('lobby-human-faction')).toBeVisible()
    await expect(page.getByTestId('lobby-model-faction')).toBeVisible()
    await expect(page.getByTestId('lobby-create-game')).toHaveText('Start AI Game')

    await page.getByTestId('lobby-create-game').click()

    await expect(page).toHaveURL(/\/game\/\d+$/, { timeout: 20_000 })
    await expect(page.getByTestId('game-screen')).toBeVisible()
    await expect(page.getByTestId('player-summary-bar')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByText('TM-AZ-', { exact: false }).first()).toBeVisible()
  })
})
