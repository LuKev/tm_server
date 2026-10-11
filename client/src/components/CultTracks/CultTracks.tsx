// Flat elemental tracks with the original canvas priest-space geometry.
import React, { useRef, useEffect, useState, useMemo, useCallback } from 'react';
import { CultType, FactionType } from '../../types/game.types';
import { FACTION_COLORS, getContrastColor } from '../../utils/colors';
import { drawPriestPawn } from '../shared/iconGeometry';
import { gameThemeColor } from '../../utils/gameTheme';
import { resolveFaction } from '../../utils/gameUtils';

export interface CultPosition {
  faction: FactionType;
  position: number;
  hasKey: boolean; // Gold rim for a faction with a key or at position 10
}

export interface PriestSpot {
  priests?: number;
  power?: number;
  faction?: FactionType; // For colored priest markers
}

const CULT_TRACK_WIDTH = 250 / 4; // 62.5px per cult
const CULT_TRACK_HEIGHT = 560;
const CULT_PRIEST_SPOT_WIDTH = 25;
const CULT_PRIEST_SPOT_HEIGHT = 20;
const CULT_PRIEST_SPOT_SPACING = 5;
const CULT_PRIEST_SPOT_BASE_Y = 462;

export const getPriestSpotRect = (cultIndex: number, tileIndex: number): { x: number; y: number; width: number; height: number } => {
  const gridWidth = 2 * CULT_PRIEST_SPOT_WIDTH + CULT_PRIEST_SPOT_SPACING;
  const startX = (CULT_TRACK_WIDTH - gridWidth) / 2;
  const baseX = CULT_TRACK_WIDTH * cultIndex + startX;

  if (tileIndex < 4) {
    const row = Math.floor(tileIndex / 2);
    const col = tileIndex % 2;
    return {
      x: baseX + col * (CULT_PRIEST_SPOT_WIDTH + CULT_PRIEST_SPOT_SPACING),
      y: CULT_PRIEST_SPOT_BASE_Y + row * (CULT_PRIEST_SPOT_HEIGHT + CULT_PRIEST_SPOT_SPACING),
      width: CULT_PRIEST_SPOT_WIDTH,
      height: CULT_PRIEST_SPOT_HEIGHT,
    };
  }

  return {
    x: baseX + CULT_PRIEST_SPOT_WIDTH / 2 + CULT_PRIEST_SPOT_SPACING / 2,
    y: CULT_PRIEST_SPOT_BASE_Y + 2 * (CULT_PRIEST_SPOT_HEIGHT + CULT_PRIEST_SPOT_SPACING),
    width: CULT_PRIEST_SPOT_WIDTH,
    height: CULT_PRIEST_SPOT_HEIGHT,
  };
}

interface CultTracksProps {
  cultPositions: Map<CultType, CultPosition[]>; // For each cult, list of faction positions
  bonusTiles?: Map<CultType, PriestSpot[]>; // Priest spots at bottom (power or priest rewards)
  onBonusTileClick?: (cult: CultType, tileIndex: number) => void; // Click handler for priest spots
  priestsOnTrack?: Record<CultType, Record<number, string[]>>; // Track -> SpotValue -> PlayerID[]
  players?: Record<string, { faction: FactionType }>; // PlayerID -> Faction info (needed for color)
}

const TRACK_STYLES: Record<CultType, { ink: string }> = {
  [CultType.Fire]: { ink: '#b44336' },
  [CultType.Water]: { ink: '#286da0' },
  [CultType.Earth]: { ink: '#85633f' },
  [CultType.Air]: { ink: '#5d7880' },
};
const POWER_GATES = [{ step: 3, power: 1 }, { step: 5, power: 2 }, { step: 7, power: 2 }, { step: 10, power: 3 }];
const stepY = (step: number): number => 64 + (10 - step) * 36;

const drawEmblem = (ctx: CanvasRenderingContext2D, cult: CultType, x: number, y: number): void => {
  ctx.save();
  ctx.translate(x, y);
  ctx.strokeStyle = '#fffaf0';
  ctx.fillStyle = '#fffaf0';
  ctx.lineWidth = 1.8;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.beginPath();
  switch (cult) {
    case CultType.Fire:
      ctx.moveTo(0, -10); ctx.bezierCurveTo(2, -3, 10, 0, 7, 6);
      ctx.bezierCurveTo(3, 12, -9, 9, -7, 1); ctx.lineTo(-3, -5);
      ctx.quadraticCurveTo(-4, 2, 0, 3); ctx.quadraticCurveTo(3, -1, 0, -10);
      ctx.fill(); break;
    case CultType.Water:
      ctx.moveTo(0, -10); ctx.bezierCurveTo(3, -5, 8, 0, 7, 4);
      ctx.bezierCurveTo(6, 12, -7, 12, -7, 4); ctx.quadraticCurveTo(-8, 0, 0, -10);
      ctx.stroke(); break;
    case CultType.Earth:
      ctx.moveTo(0, -9); ctx.lineTo(9, 0); ctx.lineTo(0, 9); ctx.lineTo(-9, 0); ctx.closePath();
      ctx.moveTo(-9, 0); ctx.lineTo(9, 0); ctx.moveTo(0, -9); ctx.lineTo(0, 9);
      ctx.stroke(); break;
    case CultType.Air:
      ctx.moveTo(-9, -4); ctx.lineTo(3, -4); ctx.bezierCurveTo(11, -4, 8, -12, 3, -8);
      ctx.moveTo(-8, 1); ctx.lineTo(7, 1); ctx.moveTo(-5, 6); ctx.lineTo(2, 6);
      ctx.bezierCurveTo(9, 6, 5, 12, 1, 9); ctx.stroke(); break;
  }
  ctx.restore();
};

export const CultTracks: React.FC<CultTracksProps> = ({ cultPositions, bonusTiles, onBonusTileClick, priestsOnTrack, players }): React.ReactElement => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [hoveredTile, setHoveredTile] = useState<{ cult: CultType, index: number } | null>(null);
  const cults = useMemo(() => [CultType.Fire, CultType.Water, CultType.Earth, CultType.Air], []);

  // Share occupancy between the artwork and buttons, including replay views
  // which supply priestsOnTrack without precomputed bonusTiles.
  const priestSpots = useMemo(() => new Map(cults.map(cult => [cult,
    Array.from({ length: 5 }, (_, index): PriestSpot => {
      const value = index === 0 ? 3 : index === 4 ? 1 : 2;
      const playerId = priestsOnTrack?.[cult]?.[value]?.[value === 2 ? index - 1 : 0];
      const player = playerId ? players?.[playerId] : undefined;
      if (player) {
        return { priests: 1, faction: resolveFaction(player.faction) };
      }
      return bonusTiles?.get(cult)?.[index] ?? {};
    }),
  ])), [cults, bonusTiles, priestsOnTrack, players]);

  // Get first letter of faction (all uppercase)
  const getFactionLetter = useCallback((faction: FactionType): string => {
    const names: Record<FactionType, string> = {
      [FactionType.Unknown]: '?',
      [FactionType.Nomads]: 'N',
      [FactionType.Fakirs]: 'F',
      [FactionType.ChaosMagicians]: 'C',
      [FactionType.Giants]: 'G',
      [FactionType.Swarmlings]: 'S',
      [FactionType.Mermaids]: 'M',
      [FactionType.Witches]: 'W',
      [FactionType.Auren]: 'A',
      [FactionType.Halflings]: 'H',
      [FactionType.Cultists]: 'C',
      [FactionType.Alchemists]: 'A',
      [FactionType.Darklings]: 'D',
      [FactionType.Engineers]: 'E',
      [FactionType.Dwarves]: 'D',
      [FactionType.Architects]: 'R',
      [FactionType.Archivists]: 'V',
      [FactionType.Atlanteans]: 'T',
      [FactionType.ChashDallah]: 'L',
      [FactionType.ChildrenOfTheWyrm]: 'Y',
      [FactionType.Conspirators]: 'O',
      [FactionType.Djinni]: 'J',
      [FactionType.DynionGeifr]: 'Y',
      [FactionType.Goblins]: 'B',
      [FactionType.Prospectors]: 'P',
      [FactionType.TheEnlightened]: 'I',
      [FactionType.TimeTravelers]: 'T',
      [FactionType.Treasurers]: 'R',
      [FactionType.Wisps]: 'W',
      [FactionType.IceMaidens]: 'I',
      [FactionType.Yetis]: 'Y',
      [FactionType.Dragonlords]: 'D',
      [FactionType.Acolytes]: 'A',
      [FactionType.Shapeshifters]: 'S',
      [FactionType.Riverwalkers]: 'R',
      [FactionType.Firewalkers]: 'F',
      [FactionType.Selkies]: 'S',
      [FactionType.SnowShamans]: 'S',
    };
    return names[faction];
  }, []);

  const drawCultMarker = useCallback((ctx: CanvasRenderingContext2D, faction: FactionType, hasKey: boolean): void => {
    const color = FACTION_COLORS[faction];
    ctx.save();
    ctx.beginPath();
    ctx.arc(0, 0, 6, 0, Math.PI * 2);
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = hasKey ? '#a77927' : '#334155';
    ctx.lineWidth = hasKey ? 1.7 : 1;
    ctx.stroke();
    ctx.fillStyle = getContrastColor(color);
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.font = '700 7px sans-serif';
    ctx.fillText(getFactionLetter(faction), 0, 0);
    ctx.restore();
  }, [getFactionLetter]);

  useEffect(() => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext('2d');
    if (!canvas || !ctx) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.save();
    ctx.scale(2, 2);
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';

    cults.forEach((cult, cultIndex) => {
      const { ink } = TRACK_STYLES[cult];
      const x = cultIndex * CULT_TRACK_WIDTH + 3;
      const width = CULT_TRACK_WIDTH - 6;
      const middle = x + width / 2;
      ctx.fillStyle = '#fff';
      ctx.fillRect(x, 6, width, 433);
      ctx.strokeStyle = ink;
      ctx.lineWidth = 1;
      ctx.strokeRect(x, 6, width, 433);
      ctx.fillStyle = ink;
      ctx.fillRect(x, 6, width, 38);
      drawEmblem(ctx, cult, middle, 24);

      const positions = cultPositions.get(cult) ?? [];
      for (let step = 10; step >= 0; step--) {
        const y = stepY(step);
        ctx.beginPath(); ctx.moveTo(x, y + 18); ctx.lineTo(x + width, y + 18);
        ctx.strokeStyle = '#d5d9dc'; ctx.lineWidth = 0.7; ctx.stroke();
        ctx.fillStyle = ink; ctx.font = '700 9px sans-serif'; ctx.fillText(String(step), middle, y);

        const occupants = positions.filter(position => position.position === step);
        occupants.forEach((position, index) => {
          const row = Math.floor(index / 3);
          const count = Math.min(3, occupants.length - row * 3);
          ctx.save();
          ctx.translate(middle + (index % 3 - (count - 1) / 2) * 12, y + (occupants.length > 3 ? -6 + row * 13 : 0));
          drawCultMarker(ctx, position.faction, step === 10 || position.hasKey);
          ctx.restore();
        });
      }

      // Power is earned crossing these gates, not by occupying a row.
      POWER_GATES.forEach(({ step, power }) => {
        const y = stepY(step) + 18;
        ctx.beginPath(); ctx.moveTo(x + 5, y); ctx.lineTo(x + width - 5, y);
        ctx.strokeStyle = '#9d83b5'; ctx.lineWidth = 1.3; ctx.stroke();
        ctx.beginPath(); ctx.arc(middle, y, 6, 0, Math.PI * 2);
        ctx.fillStyle = '#77539a'; ctx.fill();
        ctx.strokeStyle = '#f8f3ed'; ctx.lineWidth = 1; ctx.stroke();
        ctx.fillStyle = '#fff'; ctx.font = '700 7px sans-serif'; ctx.fillText(String(power), middle, y);
      });

      (priestSpots.get(cult) ?? []).forEach((tile, index) => {
        const rect = getPriestSpotRect(cultIndex, index);
        const occupied = !!(tile.priests && tile.faction !== undefined);
        const hovered = onBonusTileClick !== undefined && !occupied && hoveredTile?.cult === cult && hoveredTile.index === index;
        ctx.beginPath(); ctx.rect(rect.x, rect.y, rect.width, rect.height);
        ctx.fillStyle = '#fff'; ctx.fill();
        ctx.strokeStyle = hovered ? gameThemeColor(canvas, 'focus') : ink + '80';
        ctx.lineWidth = hovered ? 2 : 1; ctx.stroke();
        if (occupied && tile.faction !== undefined) {
          const cx = rect.x + rect.width / 2;
          const cy = rect.y + rect.height / 2;
          drawPriestPawn(ctx, cx, cy, 19, FACTION_COLORS[tile.faction]);
        } else {
          ctx.fillStyle = ink; ctx.font = '700 10px sans-serif';
          ctx.fillText(String(tile.power ?? (index === 0 ? 3 : index === 4 ? 1 : 2)), rect.x + rect.width / 2, rect.y + rect.height / 2);
        }
      });
    });
    ctx.restore();
  }, [cultPositions, priestSpots, hoveredTile, cults, drawCultMarker, onBonusTileClick]);

  const description = cults.map(cult => `${CultType[cult]}: ${(cultPositions.get(cult) ?? []).map(position => `${FactionType[position.faction]} ${String(position.position)}`).join(', ')}`).join('; ');
  return (
    <div className="cult-tracks w-full" data-testid="cult-tracks" style={{ position: 'relative' }}>
      <canvas
        ref={canvasRef}
        data-testid="cult-tracks-canvas"
        role="img"
        aria-label={`Cult tracks. ${description}. Power rewards: 1 at step 3, 2 at steps 5 and 7, 3 at step 10.`}
        width={500}
        height={CULT_TRACK_HEIGHT * 2}
        style={{ width: '100%', height: 'auto', display: 'block' }}
      />
      <div aria-hidden={onBonusTileClick === undefined} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
        {cults.flatMap((cult, cultIndex) => (priestSpots.get(cult) ?? []).map((tile, tileIndex) => {
          const rect = getPriestSpotRect(cultIndex, tileIndex);
          const occupied = !!(tile.priests && tile.faction !== undefined);
          const enabled = !occupied && onBonusTileClick !== undefined;
          return (
            <button
              key={`${String(cult)}-${String(tileIndex)}`}
              type="button"
              aria-label={`${CultType[cult]} cult, ${String(tileIndex === 0 ? 3 : tileIndex === 4 ? 1 : 2)} steps${occupied ? ', occupied' : ''}`}
              data-testid={`cult-spot-${String(cult)}-${String(tileIndex)}`}
              disabled={!enabled}
              onClick={() => { onBonusTileClick?.(cult, tileIndex); }}
              onMouseEnter={() => { if (enabled) setHoveredTile({ cult, index: tileIndex }); }}
              onMouseLeave={() => { setHoveredTile(null); }}
              onFocus={() => { setHoveredTile({ cult, index: tileIndex }); }}
              onBlur={() => { setHoveredTile(null); }}
              style={{
                position: 'absolute',
                left: `${String((rect.x / 250) * 100)}%`,
                top: `${String((rect.y / CULT_TRACK_HEIGHT) * 100)}%`,
                width: `${String((rect.width / 250) * 100)}%`,
                height: `${String((rect.height / CULT_TRACK_HEIGHT) * 100)}%`,
                pointerEvents: 'auto', borderRadius: 0,
                border: 'none', background: 'transparent', padding: 0, margin: 0,
                cursor: enabled ? 'pointer' : 'default',
              }}
            />
          );
        }))}
      </div>
    </div>
  );
};
