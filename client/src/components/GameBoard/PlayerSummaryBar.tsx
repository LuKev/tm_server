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

  // Height is controlled by the surrounding grid item (we render full height).
  return (
    <div className="w-full h-full flex gap-2">
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
            className={`flex-1 h-full px-2 flex flex-col justify-center rounded-md border border-gray-200 shadow-sm bg-white ${isCurrent ? 'bg-yellow-50 ring-2 ring-yellow-400' : ''}`}
          >
            <div className="flex items-center justify-between" style={{ fontSize: '0.8rem', lineHeight: 1 }}>
              <div className="flex items-center gap-2 min-w-0">
                <div
                  className="w-2 h-2 rounded-full border border-gray-300 flex-shrink-0"
                  style={{ backgroundColor: factionColor }}
                  title={FactionType[factionType]}
                />
                <div className="flex items-center gap-1 text-gray-700 font-semibold">
                  <span>#{turnOrderNumber}</span>
                  <span className="text-gray-400">|</span>
                  <span className="font-bold">{vp} VP</span>
                </div>
              </div>
              {isCurrent && (
                <div className="text-xs font-bold text-gray-800" style={{ lineHeight: 1 }}>
                  TURN
                </div>
              )}
            </div>

            <div className="flex items-center gap-3 mt-1" style={{ fontSize: '0.75rem', lineHeight: 1 }}>
              <div className="flex items-center gap-1">
                <CoinIcon style={{ width: '1.15em', height: '1.15em', fontSize: '0.9em' }} />
                <span className="font-semibold text-gray-700">{player.resources.coins}</span>
              </div>
              <div className="flex items-center gap-1">
                <WorkerIcon style={{ width: '1.15em', height: '1.15em', fontSize: '0.9em' }} />
                <span className="font-semibold text-gray-700">{player.resources.workers}</span>
              </div>
              <div className="flex items-center gap-1">
                <PriestIcon style={{ width: '1.15em', height: '1.15em' }} />
                <span className="font-semibold text-gray-700">{player.resources.priests}</span>
              </div>
              <div className="flex items-center gap-1 text-gray-700 font-semibold">
                <span className="text-gray-500">PW</span>
                <span style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {player.resources.power.powerI}/{player.resources.power.powerII}/{player.resources.power.powerIII}
                </span>
              </div>
              <div className="ml-auto flex items-center">
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
