import React from 'react';
import { BonusCardType, FactionType, type GameState, type PlayerState } from '../../types/game.types';
import { FACTIONS } from '../../data/factions';
import { FACTION_COLORS } from '../../utils/colors';
import { CoinIcon, WorkerIcon, PriestIcon } from '../shared/Icons';
import { ShippingDiggingDisplay } from '../shared/ShippingDiggingDisplay';

const resolveFactionType = (player: PlayerState): FactionType | null => {
  const raw: unknown = (player as unknown as { faction?: unknown; Faction?: unknown }).faction
    ?? (player as unknown as { Faction?: unknown }).Faction;

  if (raw === undefined || raw === null) return null;

  if (typeof raw === 'number') return raw as FactionType;

  if (typeof raw === 'string') {
    const found = FACTIONS.find(f => f.type === raw || f.name === raw);
    return found ? found.id : null;
  }

  if (typeof raw === 'object') {
    if ('Type' in raw) return (raw as { Type: number }).Type as FactionType;
    if ('type' in raw) return (raw as { type: number }).type as FactionType;
  }

  return null;
};

export const PlayerSummaryBar: React.FC<{ gameState: GameState }> = ({ gameState }) => {
  if (!gameState.players) return null;

  const playerIds =
    gameState.turnOrder && gameState.turnOrder.length > 0
      ? gameState.turnOrder
      : Object.keys(gameState.players).sort();

  const currentPlayerId = gameState.turnOrder?.[gameState.currentTurn];
  const playerCount = playerIds.length;

  // Height is controlled by the surrounding grid item (we render full height).
  return (
    <div
      className="w-full h-full gap-2"
      // Tailwind utilities aren't reliably present in production right now, so keep the
      // layout-critical bits as inline styles.
      style={{
        display: 'grid',
        gap: '0.5rem',
        gridTemplateColumns: `repeat(${String(Math.max(1, playerCount))}, minmax(0, 1fr))`,
        width: '100%',
        height: '100%',
      }}
    >
      {playerIds.map((pid, idx) => {
        const player = gameState.players[pid];
        if (!player) return null;

        const turnOrderIndex = gameState.turnOrder ? gameState.turnOrder.indexOf(pid) : -1;
        const turnOrderNumber = turnOrderIndex !== -1 ? turnOrderIndex + 1 : idx + 1;

        const factionType = resolveFactionType(player) ?? FactionType.Unknown;
        const factionColor = FACTION_COLORS[factionType] ?? '#9ca3af';
        const isCurrent = pid === currentPlayerId;

        const hasTempShippingBonus = gameState.bonusCards?.playerCards?.[pid] === BonusCardType.Shipping;
        const shippingLevel = (player as unknown as { shipping?: number }).shipping ?? 0;
        const diggingLevel = (player as unknown as { digging?: number }).digging ?? 0;
        const vp = player.victoryPoints ?? player.VictoryPoints ?? 0;

        return (
          <div
            key={pid}
            className="min-w-0"
            style={{
              height: '100%',
              paddingLeft: '0.5rem',
              paddingRight: '0.5rem',
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              border: '2px solid #000',
              borderRadius: '0.375rem',
              backgroundColor: isCurrent ? '#FEFCE8' : '#FFFFFF', // yellow-50
              boxSizing: 'border-box',
              outline: isCurrent ? '2px solid #FACC15' : undefined, // yellow-400
              outlineOffset: isCurrent ? '0px' : undefined,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '0.8rem', lineHeight: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', minWidth: 0 }}>
                <div
                  style={{
                    width: '0.5rem',
                    height: '0.5rem',
                    borderRadius: '9999px',
                    border: '1px solid #d1d5db',
                    backgroundColor: factionColor,
                    flexShrink: 0,
                  }}
                  title={FactionType[factionType]}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', color: '#374151', fontWeight: 600 }}>
                  <span>#{turnOrderNumber}</span>
                  <span style={{ color: '#9ca3af' }}>|</span>
                  <span style={{ fontWeight: 700 }}>{vp} VP</span>
                </div>
              </div>
              {isCurrent && (
                <div style={{ fontSize: '0.75rem', fontWeight: 700, color: '#1f2937', lineHeight: 1 }}>
                  TURN
                </div>
              )}
            </div>

            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginTop: '0.25rem', minWidth: 0, fontSize: '0.75rem', lineHeight: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                <CoinIcon style={{ width: '1.15em', height: '1.15em', fontSize: '0.9em' }} />
                <span style={{ fontWeight: 600, color: '#374151' }}>{player.resources.coins}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                <WorkerIcon style={{ width: '1.15em', height: '1.15em', fontSize: '0.9em' }} />
                <span style={{ fontWeight: 600, color: '#374151' }}>{player.resources.workers}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                <PriestIcon style={{ width: '1.15em', height: '1.15em' }} />
                <span style={{ fontWeight: 600, color: '#374151' }}>{player.resources.priests}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', color: '#374151', fontWeight: 600 }}>
                <span style={{ color: '#6b7280' }}>PW</span>
                <span style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {player.resources.power.powerI}/{player.resources.power.powerII}/{player.resources.power.powerIII}
                </span>
              </div>
              <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', flexShrink: 0 }}>
                <ShippingDiggingDisplay
                  factionType={factionType}
                  shipping={shippingLevel}
                  diggingLevel={diggingLevel}
                  hasTempShippingBonus={hasTempShippingBonus}
                  compact={true}
                />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};
