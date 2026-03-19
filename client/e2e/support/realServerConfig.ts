const realServerPort = (): string => process.env.TM_PLAYWRIGHT_SERVER_PORT ?? '8080'

export const realServerWsURL = (): string => `ws://127.0.0.1:${realServerPort()}/api/ws`
