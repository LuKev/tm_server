import { FactionType, PowerActionType } from '../types/game.types'

// Display/payment-choice estimate only. The server validates and charges actions.
export function powerActionCost(action: PowerActionType, faction?: FactionType): number {
  const base = action === PowerActionType.Bridge || action === PowerActionType.Priest ? 3
    : action === PowerActionType.DoubleSpade ? 6 : 4
  return base - (faction === FactionType.Yetis ? 1 : 0)
}
