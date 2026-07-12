import { GamePhase, type GameState, type PlayerState } from '../types/game.types'

export type PendingDecisionRecord = Record<string, unknown>

export type PendingDecisionView = {
  decision: PendingDecisionRecord | null
  type: string | null
  playerId: string | null
  playerIds: string[]
  auctionFactions: string[]
  hasForLocalPlayer: boolean
  isFastAuctionBidForLocalPlayer: boolean
  isOtherPlayerTurnConfirmationWindow: boolean
  isBlockingLocalInteraction: boolean
}

type AuctionLike = {
  currentBidder?: string
  fastSubmitted?: Record<string, boolean>
} | null | undefined

export const formatActorList = (
  playerIds: string[],
  players: Record<string, PlayerState> | undefined,
  localPlayerId: string | null,
): string => {
  const uniqueIds = playerIds.filter((playerId, index) => playerId && playerIds.indexOf(playerId) === index)
  const labels = uniqueIds.map((playerId) => {
    if (playerId === localPlayerId) return 'You'
    const player = players?.[playerId]
    const displayName = typeof player?.name === 'string' ? player.name.trim() : ''
    return displayName || playerId
  })
  if (labels.length === 0) return 'Unknown player'
  if (labels.length === 1) return labels[0]
  if (labels.length === 2) return `${labels[0]} and ${labels[1]}`
  return `${labels.slice(0, -1).join(', ')}, and ${labels[labels.length - 1]}`
}

const stringArray = (value: unknown): string[] => {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => (typeof item === 'string' ? item : ''))
    .filter((item) => item.length > 0)
}

export const buildPendingDecisionView = (
  rawDecision: GameState['pendingDecision'] | undefined,
  localPlayerId: string | null,
  auctionState: AuctionLike,
): PendingDecisionView => {
  const decision = (rawDecision ?? null) as PendingDecisionRecord | null
  const type = (decision?.type as string | undefined) ?? null
  const playerId = (decision?.playerId as string | undefined) ?? null
  const playerIds = stringArray(decision?.playerIds)
  const auctionFactions = stringArray(decision?.nominatedFactions)
  const hasForLocalPlayer = !!localPlayerId && (playerId === localPlayerId || playerIds.includes(localPlayerId))
  const isFastAuctionBidForLocalPlayer = type === 'fast_auction_bid_matrix'
    && !!localPlayerId
    && !(auctionState?.fastSubmitted?.[localPlayerId] ?? false)
  const isOtherPlayerTurnConfirmationWindow = !!playerId
    && playerId !== localPlayerId
    && (type === 'post_action_free_actions' || type === 'turn_confirmation')

  return {
    decision,
    type,
    playerId,
    playerIds,
    auctionFactions,
    hasForLocalPlayer,
    isFastAuctionBidForLocalPlayer,
    isOtherPlayerTurnConfirmationWindow,
    isBlockingLocalInteraction: hasForLocalPlayer || isOtherPlayerTurnConfirmationWindow,
  }
}

type DecisionStripStatusInput = {
  pendingDecision: PendingDecisionView
  orderedPendingLeechResponders: string[]
  players: Record<string, PlayerState> | undefined
  localPlayerId: string | null
  pendingHalflingsSpades: GameState['pendingHalflingsSpades']
  phase: GamePhase | undefined
  setupMode: 'snellman' | 'auction' | 'fast_auction'
  currentPlayerId: string | undefined
  setupDwellingPlayerId: string | null
  auctionCurrentBidder: string | undefined
}

export const getDecisionStripStatus = ({
  pendingDecision,
  orderedPendingLeechResponders,
  players,
  localPlayerId,
  pendingHalflingsSpades,
  phase,
  setupMode,
  currentPlayerId,
  setupDwellingPlayerId,
  auctionCurrentBidder,
}: DecisionStripStatusInput): string => {
  const actorText = (playerIds: string[], singularAction: string, pluralAction = singularAction): string => {
    const uniqueIds = playerIds.filter((playerId, index) => playerId && playerIds.indexOf(playerId) === index)
    if (uniqueIds.length === 0) return ''
    return `${formatActorList(uniqueIds, players, localPlayerId)} ${uniqueIds.length > 1 ? pluralAction : singularAction}`
  }

  switch (pendingDecision.type) {
    case 'cult_reward_spade':
      return actorText([pendingDecision.playerId ?? ''], 'must use cult spades.')
    case 'spade_followup':
      return actorText([pendingDecision.playerId ?? ''], 'must resolve pending spades.')
    case 'post_action_free_actions':
      return actorText([pendingDecision.playerId ?? ''], 'must finish free actions or confirm the turn.')
    case 'turn_confirmation':
      return actorText([pendingDecision.playerId ?? ''], 'must confirm or undo the turn.')
    case 'favor_tile_selection':
      return actorText([pendingDecision.playerId ?? ''], 'must select a favor tile.')
    case 'town_tile_selection':
      return actorText([pendingDecision.playerId ?? ''], 'must select a town tile.')
    case 'town_cult_top_choice':
      return actorText([pendingDecision.playerId ?? ''], 'must choose cult tracks to top.')
    case 'setup_bonus_card':
      return actorText([pendingDecision.playerId ?? ''], 'must choose a setup bonus card.')
    case 'darklings_ordination':
      return actorText([pendingDecision.playerId ?? ''], 'must choose a Darklings ordination.')
    case 'cultists_cult_choice':
      return actorText([pendingDecision.playerId ?? ''], 'must choose a cult track.')
    case 'djinni_start_cult_choice':
      return actorText([pendingDecision.playerId ?? ''], 'must choose a starting cult track.')
    case 'riverwalkers_priest_choice':
      return actorText([pendingDecision.playerId ?? ''], 'must choose how to use a Riverwalkers priest.')
    case 'treasurers_deposit':
      return actorText([pendingDecision.playerId ?? ''], 'must choose which resources to bank in the Treasury.')
    case 'goblins_cult_steps': {
      const remaining = Number(pendingDecision.decision?.stepsRemaining ?? 0)
      return actorText(
        [pendingDecision.playerId ?? ''],
        remaining === 1 ? 'must choose 1 Goblins cult step.' : `must choose ${String(remaining)} Goblins cult steps.`,
      )
    }
    case 'halflings_spades': {
      const remaining = Number((pendingHalflingsSpades as Record<string, unknown> | undefined)?.spadesRemaining ?? 0)
      return actorText([pendingDecision.playerId ?? ''], remaining === 0 ? 'must decide whether to build a dwelling.' : 'must use Halflings spades.')
    }
    case 'wisps_stronghold_dwelling':
      return actorText([pendingDecision.playerId ?? ''], 'must place the free Wisps dwelling.')
    case 'archivists_bonus_card':
      return actorText([pendingDecision.playerId ?? ''], 'must choose an Archivists bonus card.')
    case 'auction_nomination':
      return actorText([pendingDecision.playerId ?? auctionCurrentBidder ?? ''], 'must nominate a faction.')
    case 'auction_bid':
      return actorText([pendingDecision.playerId ?? auctionCurrentBidder ?? ''], 'must place a bid.')
    case 'fast_auction_bid_matrix':
      return actorText(pendingDecision.playerIds, 'must submit bids.', 'must submit bids.')
    default:
      break
  }

  if (orderedPendingLeechResponders.length > 0) {
    return actorText(orderedPendingLeechResponders, 'must make a leech decision.', 'must make leech decisions.')
  }

  if (phase === GamePhase.FactionSelection) {
    if (setupMode === 'snellman' && currentPlayerId) {
      return actorText([currentPlayerId], 'must choose a faction.')
    }
    if (setupMode === 'auction' && (pendingDecision.playerId ?? auctionCurrentBidder)) {
      return actorText([pendingDecision.playerId ?? auctionCurrentBidder ?? ''], 'must continue the auction.')
    }
    if (setupMode === 'fast_auction' && pendingDecision.playerIds.length > 0) {
      return actorText(pendingDecision.playerIds, 'must submit bids.', 'must submit bids.')
    }
  }

  if (phase === GamePhase.Setup && setupDwellingPlayerId) {
    return actorText([setupDwellingPlayerId], 'must place a dwelling.')
  }

  if (phase === GamePhase.Action && currentPlayerId) {
    return actorText([currentPlayerId], 'must take an action.')
  }

  return 'Waiting for the next required action.'
}
