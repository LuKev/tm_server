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

  test('rejoins the cached model game after websocket reconnection', async ({ page }) => {
    const getGameStateFrames: string[] = []
    const receivedFrames: string[] = []
    page.on('websocket', (websocket) => {
      websocket.on('framesent', ({ payload }) => {
        if (typeof payload === 'string' && payload.includes('"type":"get_game_state"')) {
          getGameStateFrames.push(payload)
        }
      })
      websocket.on('framereceived', ({ payload }) => {
        if (typeof payload === 'string') receivedFrames.push(payload)
      })
    })

    await page.goto('/ai')
    await expect(page.getByText('connected', { exact: true })).toBeVisible({ timeout: 15_000 })
    await page.getByTestId('ai-start-game').click()
    await expect(page).toHaveURL(/\/game\/\d+$/, { timeout: 20_000 })
    await expect(page.getByTestId('player-summary-bar')).toBeVisible({ timeout: 20_000 })
    await expect.poll(() => getGameStateFrames.length).toBe(1)

    await page.evaluate(() => {
      const closeWebSocket = (window as Window & { __TM_TEST_CLOSE_WEBSOCKET__?: () => void }).__TM_TEST_CLOSE_WEBSOCKET__
      if (!closeWebSocket) throw new Error('WebSocket close test hook is unavailable')
      closeWebSocket()
    })

    await expect.poll(() => getGameStateFrames.length, { timeout: 10_000 }).toBe(2)
    const firstRequest = JSON.parse(getGameStateFrames[0]) as { payload: { gameID: string; playerID: string } }
    const reconnectRequest = JSON.parse(getGameStateFrames[1]) as { payload: { gameID: string; playerID: string } }
    expect(firstRequest.payload.gameID).not.toBe('')
    expect(firstRequest.payload.playerID).not.toBe('')
    expect(reconnectRequest.payload).toEqual(firstRequest.payload)

    const acceptedBefore = receivedFrames.filter((frame) => frame.includes('"type":"action_accepted"')).length
    await page.getByTestId('option-auto-leech-mode').selectOption('accept_1')
    await expect.poll(
      () => receivedFrames.filter((frame) => frame.includes('"type":"action_accepted"')).length,
      { timeout: 10_000 },
    ).toBeGreaterThan(acceptedBefore)
  })
})
