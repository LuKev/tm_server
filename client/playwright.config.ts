import { defineConfig, devices } from '@playwright/test'

const clientPort = Number(process.env.TM_PLAYWRIGHT_CLIENT_PORT ?? '4173')
const serverPort = Number(process.env.TM_PLAYWRIGHT_SERVER_PORT ?? '8080')
const serverCommand = process.env.TM_PLAYWRIGHT_SERVER_COMMAND ?? 'bazel run //cmd/server:server'
const serverCwd = process.env.TM_PLAYWRIGHT_SERVER_CWD ?? '../server'
const clientCommand =
  process.env.TM_PLAYWRIGHT_CLIENT_COMMAND ?? `npm run dev -- --host 127.0.0.1 --port ${String(clientPort)}`
const clientCwd = process.env.TM_PLAYWRIGHT_CLIENT_CWD
const captureArtifacts = process.env.TM_PLAYWRIGHT_CAPTURE === '1'

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: true,
  forbidOnly: process.env.CI === '1',
  retries: process.env.CI === '1' ? 2 : 0,
  workers: process.env.CI === '1' ? 2 : undefined,
  reporter: process.env.CI === '1' ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: `http://127.0.0.1:${String(clientPort)}`,
    trace: captureArtifacts ? 'on-first-retry' : 'off',
    screenshot: captureArtifacts ? 'only-on-failure' : 'off',
    video: captureArtifacts ? 'retain-on-failure' : 'off',
  },
  webServer: [
    {
      command: serverCommand,
      cwd: serverCwd,
      url: `http://127.0.0.1:${String(serverPort)}/health`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        ...process.env,
        PORT: String(serverPort),
        TM_ENABLE_TEST_COMMANDS: '1',
      },
    },
    {
      command: clientCommand,
      cwd: clientCwd,
      url: `http://127.0.0.1:${String(clientPort)}`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
