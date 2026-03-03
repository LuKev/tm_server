import path from 'node:path'
import { fileURLToPath } from 'node:url'

export type GoldenScenarioMode = 'smoke' | 'nightly'

export type GoldenScenario = {
  id: string
  scriptPath: string
  mode: GoldenScenarioMode
  expectedScores: Record<string, number>
  fixtureLabel: string
}

const fixturesDir = path.dirname(fileURLToPath(import.meta.url))

export const GOLDEN_SCENARIOS: GoldenScenario[] = [
  {
    id: 's69_g2',
    scriptPath: path.resolve(fixturesDir, 's69_g2_actions.json'),
    mode: 'smoke',
    fixtureLabel: '4pLeague_S69_D1L1_G2',
    expectedScores: {
      Nomads: 166,
      Darklings: 137,
      Mermaids: 130,
      Witches: 124,
    },
  },
  {
    id: 's60_g4',
    scriptPath: path.resolve(fixturesDir, 's60_g4_actions.json'),
    mode: 'nightly',
    fixtureLabel: '4pLeague_S60_D1L1_G4',
    expectedScores: {
      Dwarves: 167,
      Darklings: 151,
      Cultists: 149,
      Giants: 115,
    },
  },
  {
    id: 's61_g3',
    scriptPath: path.resolve(fixturesDir, 's61_g3_actions.json'),
    mode: 'nightly',
    fixtureLabel: '4pLeague_S61_D1L1_G3',
    expectedScores: {
      Cultists: 160,
      Darklings: 161,
      Witches: 96,
      Engineers: 113,
    },
  },
]
