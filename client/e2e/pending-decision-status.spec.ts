import { expect, test } from '@playwright/test'

import { buildPendingDecisionView, getDecisionStripStatus } from '../src/utils/pendingDecision'
import { GamePhase } from '../src/types/game.types'

test('turn confirmation takes priority over an opponent leech offer', () => {
  const localPlayerId = 'human'
  const pendingDecision = buildPendingDecisionView({
    type: 'turn_confirmation',
    playerId: localPlayerId,
  }, localPlayerId, null)

  const status = getDecisionStripStatus({
    pendingDecision,
    orderedPendingLeechResponders: ['TM-AZ-2'],
    players: undefined,
    localPlayerId,
    pendingHalflingsSpades: undefined,
    phase: GamePhase.Action,
    setupMode: 'snellman',
    currentPlayerId: localPlayerId,
    setupDwellingPlayerId: null,
    auctionCurrentBidder: undefined,
  })

  expect(status).toBe('You must confirm or undo the turn.')
})
