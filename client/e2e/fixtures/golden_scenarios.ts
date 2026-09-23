import path from 'node:path'
import { fileURLToPath } from 'node:url'

type GoldenScenarioMode = 'smoke' | 'nightly'

type GoldenScenario = {
  id: string
  scriptPath: string
  mode: GoldenScenarioMode
  fixtureLabel: string
  scoreTolerance?: number
  wsOnlyReplay?: boolean
}

const fixturesDir = path.dirname(fileURLToPath(import.meta.url))

export const GOLDEN_SCENARIOS: GoldenScenario[] = [
  {
    id: 's69_g2',
    scriptPath: path.resolve(fixturesDir, 's69_g2_actions.json'),
    mode: 'smoke',
    fixtureLabel: '4pLeague_S69_D1L1_G2',
    wsOnlyReplay: true,
  },
  {
    id: 's60_g4',
    scriptPath: path.resolve(fixturesDir, 's60_g4_actions.json'),
    mode: 'nightly',
    fixtureLabel: '4pLeague_S60_D1L1_G4',
  },
  {
    id: 's61_g3',
    scriptPath: path.resolve(fixturesDir, 's61_g3_actions.json'),
    mode: 'nightly',
    fixtureLabel: '4pLeague_S61_D1L1_G3',
    wsOnlyReplay: true,
  },
]
