import { FactionType } from '../../types/game.types'

export type ModelStrength = 'fast' | 'balanced' | 'strong'

export const MODEL_STRENGTHS: Record<ModelStrength, { label: string; simulations: number }> = {
  fast: { label: 'Fast (16)', simulations: 16 },
  balanced: { label: 'Balanced (64)', simulations: 64 },
  strong: { label: 'Deep (160)', simulations: 160 },
}

export const DEFAULT_HUMAN_FACTION = FactionType.Nomads
export const DEFAULT_MODEL_FACTION = FactionType.Witches

export const modelPlayerIdForGame = (gameId: string): string => `TM-AZ-${gameId}`

export function generatedAIPlayerName(): string {
  return `Guest-${Date.now().toString(36)}`
}
