// Cult Tracks component - based on terra-mystica/stc/game.js drawCults()
import React, { useRef, useEffect, useState, useMemo, useCallback } from 'react';
import { CultType, FactionType } from '../../types/game.types';
import { FACTION_COLORS, CULT_COLORS, getContrastColor } from '../../utils/colors';

export interface CultPosition {
  faction: FactionType;
  position: number;
  hasKey: boolean; // Power key (shows hex instead of circle at position 10)
}

export interface PriestSpot {
  priests?: number;
  power?: number;
  faction?: FactionType; // For colored priest markers
}

interface CultTracksProps {
  cultPositions: Map<CultType, CultPosition[]>; // For each cult, list of faction positions
  bonusTiles?: Map<CultType, PriestSpot[]>; // Priest spots at bottom (power or priest rewards)
  onBonusTileClick?: (cult: CultType, tileIndex: number) => void; // Click handler for priest spots
  priestsOnTrack?: Record<CultType, Record<number, string[]>>; // Track -> SpotValue -> PlayerID[]
  players?: Record<string, { faction: FactionType }>; // PlayerID -> Faction info (needed for color)
}

export const CultTracks: React.FC<CultTracksProps> = ({ cultPositions, bonusTiles, onBonusTileClick, priestsOnTrack, players }): React.ReactElement => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [hoveredTile, setHoveredTile] = useState<{ cult: CultType, index: number } | null>(null);

  const cults = useMemo(() => [CultType.Fire, CultType.Water, CultType.Earth, CultType.Air], []);
  const cultWidth = 250 / 4; // 62.5px per cult
  const height = 560; // Match game board height

  // Draw hex path (simplified from makeHexPath)
  const drawHexPath = useCallback((ctx: CanvasRenderingContext2D, x: number, y: number, size: number): void => {
    let angle = 0;
    ctx.moveTo(x, y);
    for (let i = 0; i < 6; i++) {
      ctx.lineTo(x, y);
      angle += Math.PI / 3;
      x += Math.sin(angle) * size;
      y += Math.cos(angle) * size;
    }
    ctx.closePath();
  }, []);

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
    };
    return names[faction];
  }, []);

  // Draw a single cult marker (from terra-mystica/stc/game.js)
  const drawCultMarker = useCallback((
    ctx: CanvasRenderingContext2D,
    faction: FactionType,
    isHex: boolean
  ): void => {
    const color = FACTION_COLORS[faction];
    const contrastColor = getContrastColor(color);

    ctx.save();
    ctx.beginPath();

    if (isHex) {
      // Draw hex shape for position 10 or with key
      drawHexPath(ctx, -8, 14, 8.5);
    } else {
      // Draw circle
      ctx.arc(0, 10, 8, 0, Math.PI * 2);
    }

    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 1;
    ctx.stroke();
    ctx.restore();

    // Draw faction letter
    ctx.save();
    ctx.fillStyle = contrastColor;
    ctx.strokeStyle = contrastColor;
    ctx.textAlign = 'center';
    ctx.font = 'bold 10px Verdana';
    ctx.lineWidth = 0.1;

    const letter = getFactionLetter(faction);
    ctx.fillText(letter, -1, 14);
    ctx.strokeText(letter, -1, 14);
    ctx.restore();
  }, [drawHexPath, getFactionLetter]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    ctx.save();
    ctx.scale(2, 2); // Scale for better rendering

    // Draw each cult track
    for (let j = 0; j < 4; j++) {
      const cult = cults[j];

      ctx.save();
      ctx.translate(cultWidth * j, 0);

      // Background
      ctx.beginPath();
      ctx.rect(0, 0, cultWidth, height);
      ctx.fillStyle = CULT_COLORS[cult];
      ctx.fill();

      ctx.translate(0, 10); // Small padding at top

      // Track positions (10 down to 0)
      let seen10 = false;
      const positions = cultPositions.get(cult) || [];

      for (let i = 10; i >= 0; i--) {
        ctx.save();
        ctx.translate(0, (10 - i) * 40 + 10);

        // Draw markers for factions at this position (centered)
        const factionsAtPosition = positions.filter(p => p.position === i);
        if (factionsAtPosition.length > 0) {
          ctx.save();

          // If multiple factions, space them out horizontally
          const markerWidth = 12; // Width taken by each marker
          const totalWidth = factionsAtPosition.length * markerWidth;

          // Start position to center the group
          ctx.translate(cultWidth / 2 - totalWidth / 2 + markerWidth / 2, 0);

          factionsAtPosition.forEach(pos => {
            drawCultMarker(
              ctx,
              pos.faction,
              !seen10 && (i === 10 || pos.hasKey)
            );
            if (i === 10) {
              seen10 = true;
            }
            // Move to the right for next marker
            ctx.translate(markerWidth, 0);
          });

          ctx.restore();
        } else {
          // Show position number only if no factions at this position (centered vertically)
          ctx.fillStyle = '#000';
          ctx.strokeStyle = '#000';
          ctx.font = '15px Verdana';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle'; // Center text vertically
          ctx.lineWidth = 0.1;
          ctx.fillText(String(i), cultWidth / 2, 10); // Center at same Y as circles
          ctx.strokeText(String(i), cultWidth / 2, 10);
        }

        ctx.restore();
      }

      // Priest spots at bottom (2x2 grid)
      ctx.save();
      ctx.translate(0, 460);
      ctx.lineWidth = 0.2;

      const tiles = bonusTiles?.get(cult) || [];
      const tileWidth = 25;
      const tileHeight = 20;
      const tileSpacing = 5;

      // Calculate total grid width and center it within the cult column
      const gridWidth = 2 * tileWidth + tileSpacing;
      const startX = (cultWidth - gridWidth) / 2;

      // Draw 2x2 grid of priest spots + return spot
      for (let tileIndex = 0; tileIndex < 5; tileIndex++) {
        const tile = tiles[tileIndex];
        let x, y;

        // Position calculation: first 4 are 2x2 grid, 5th is centered below
        if (tileIndex < 4) {
          const row = Math.floor(tileIndex / 2);
          const col = tileIndex % 2;
          x = startX + col * (tileWidth + tileSpacing);
          y = row * (tileHeight + tileSpacing);
        } else {
          // 5th spot: centered below the grid
          x = startX + tileWidth / 2 + tileSpacing / 2;
          y = 2 * (tileHeight + tileSpacing);
        }

        // Draw tile background box
        ctx.beginPath();
        ctx.rect(x, y, tileWidth, tileHeight);
        ctx.fillStyle = '#f8f8f8'; // Light gray background
        ctx.fill();

        // Check if this spot has a priest (not clickable if it does)
        const hasPriest = tile?.priests && tile.priests > 0 && tile.faction !== undefined;

        // Draw border (green if hovered and no priest, gray otherwise)
        if (hoveredTile && hoveredTile.cult === cult && hoveredTile.index === tileIndex && !hasPriest) {
          ctx.strokeStyle = '#00ff00'; // Green outline on hover
          ctx.lineWidth = 2;
        } else {
          ctx.strokeStyle = '#999'; // Gray border
          ctx.lineWidth = 1;
        }
        ctx.stroke();

        // Draw tile content
        let text = tileIndex === 0 ? '3' : tileIndex === 4 ? '1' : '2'; // 3, 2, 2, 2, 1
        let color = '#000';
        let font = '14px Verdana';

        // Check if this specific spot (3, 2, 1) is occupied by a player
        // The backend tracks this in priestsOnTrack: Track -> SpotValue (3, 2, 1) -> PlayerID
        // We need to map tileIndex to SpotValue:
        // tileIndex 0 -> Spot 3
        // tileIndex 1, 2, 3 -> Spot 2 (shared? or distinct? The backend uses int keys, so likely distinct if we want to track all 3 spots)
        // Actually, the backend `PriestsOnTrack` uses int keys 3, 2, 1.
        // But there are multiple "2" spots. The user said "For later things ->2E, we should also cover a "2" spot on one of the tracks. It should always be the next available empty spot."
        // For now, let's assume the backend tracks specific spots if we send unique values, OR we just check if *any* player is on "3".
        // The current backend implementation uses `spaces` as the key. `spaces` is 3 for the 3-spot, 2 for the 2-spots.
        // If multiple players take "2" spots, they would overwrite each other in `PriestsOnTrack[track][2]` if we only use `2` as key.
        // However, the user request implies we are starting with the "3" spot.
        // "Whenever a player sends a priest to a particular cult track, let's say the "3" spot in fire, the "3" in the bottom of the cult track should be replaced with a "P"."

        // Let's check if we have a player ID for this spot value


        // We need to pass `priestsOnTrack` to this component to check this.
        // For now, let's rely on the `bonusTiles` prop which seems to be derived from `PriestsOnActionSpaces`?
        // No, `bonusTiles` is `Map<CultType, PriestSpot[]>`. `PriestSpot` has `faction`.
        // If `bonusTiles` is correctly populated with faction info from the backend, we can use it directly.

        if (tile) {
          if (hasPriest && tile.faction !== undefined) {
            text = 'P'; // Capital P for priest
            color = FACTION_COLORS[tile.faction];
            font = 'bold 14px Verdana'; // Bold font for priest
          } else if (tile.power) {
            text = String(tile.power);
          }
        }

        // Check if this specific spot (3, 2, 1) is occupied by a player (from backend state)
        const spotValue = tileIndex === 0 ? 3 : tileIndex === 4 ? 1 : 2;
        if (priestsOnTrack && priestsOnTrack[cult] && priestsOnTrack[cult][spotValue]) {
          const playerIds = priestsOnTrack[cult][spotValue];
          let playerId: string | undefined;

          // Determine which player in the list corresponds to this tile
          if (spotValue === 3 || spotValue === 1) {
            // For 3 and 1 spots, there's only one spot, so take the first player
            if (playerIds.length > 0) playerId = playerIds[0];
          } else {
            // For "2" spots, we have multiple tiles (indices 1, 2, 3)
            // Map tileIndex to array index: 1->0, 2->1, 3->2
            const arrayIndex = tileIndex - 1;
            if (arrayIndex >= 0 && arrayIndex < playerIds.length) {
              playerId = playerIds[arrayIndex];
            }
          }

          if (playerId) {
            // Resolve faction color using players map
            if (players && players[playerId]) {
              const faction = players[playerId].faction;
              // Handle case where faction is an object (from backend) or just enum
              let factionType: FactionType | undefined;
              if (typeof faction === 'number') {
                factionType = faction;
              } else if (typeof faction === 'object' && faction !== null) {
                // @ts-ignore - handle backend struct
                factionType = (faction as any).Type || (faction as any).type;
              }

              if (factionType !== undefined) {
                text = 'P';
                color = FACTION_COLORS[factionType];
                font = 'bold 14px Verdana';
              }
            }
          }
        }





        // Override with specific player tracking if available (for the "3" spot specifically requested)
        // We need to update the parent component to populate `bonusTiles` with this info or pass `priestsOnTrack`.
        // Let's assume `bonusTiles` will be updated in the parent to reflect `priestsOnTrack`.

        ctx.font = font;
        ctx.fillStyle = color;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(text, x + tileWidth / 2, y + tileHeight / 2);
      }

      ctx.restore();
      ctx.restore();
    }

    // Draw horizontal divider lines
    ctx.beginPath();
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 1;
    ctx.translate(0, 50.5);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);
    ctx.moveTo(0, 6); ctx.lineTo(250, 6);

    ctx.translate(0, 120);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);

    ctx.translate(0, 80);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);

    ctx.translate(0, 80);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);

    ctx.stroke();
    ctx.restore();
  }, [cultPositions, bonusTiles, hoveredTile, cultWidth, cults, drawCultMarker]);

  // Handle mouse move for hover effects
  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>): void => {
    const canvas = canvasRef.current;
    if (!canvas || !bonusTiles) return;

    const rect = canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * 2; // Account for canvas scaling
    const y = (e.clientY - rect.top) * 2;

    // Only check priest spots area (bottom of canvas)
    const priestAreaY = 960; // Start of priest spots area (460 * 2 + empirical adjustment)
    if (y < priestAreaY) {
      setHoveredTile(null);
      return;
    }

    // Check if mouse is over any priest spot
    let foundTile = false;
    for (let cultIndex = 0; cultIndex < 4; cultIndex++) {
      const cult = cults[cultIndex];

      const tileWidth = 25;
      const tileHeight = 20;
      const tileSpacing = 5;

      // Calculate centered position (matching rendering logic)
      const gridWidth = 2 * tileWidth + tileSpacing;
      const startX = (cultWidth - gridWidth) / 2;
      const cultX = (cultWidth * cultIndex + startX) * 2; // Scale by 2 for mouse coords
      const cultY = 960; // 460 * 2 + empirical adjustment

      const tiles = bonusTiles?.get(cult) || [];

      // Check all 5 spots (4 in 2x2 grid + 1 return spot)
      for (let tileIndex = 0; tileIndex < 5; tileIndex++) {
        let tileX, tileY;

        // Position calculation: first 4 are 2x2 grid, 5th is centered below
        if (tileIndex < 4) {
          const row = Math.floor(tileIndex / 2);
          const col = tileIndex % 2;
          tileX = cultX + col * (tileWidth + tileSpacing) * 2;
          tileY = cultY + row * (tileHeight + tileSpacing) * 2;
        } else {
          // 5th spot: centered below the grid
          tileX = cultX + (tileWidth / 2 + tileSpacing / 2) * 2;
          tileY = cultY + 2 * (tileHeight + tileSpacing) * 2;
        }

        if (x >= tileX && x <= tileX + tileWidth * 2 &&
          y >= tileY && y <= tileY + tileHeight * 2) {
          // Only hover if spot doesn't have a priest
          const tile = tiles[tileIndex];
          const hasPriest = tile?.priests && tile.priests > 0 && tile.faction !== undefined;

          if (!hasPriest) {
            setHoveredTile({ cult, index: tileIndex });
            foundTile = true;
          }
          break;
        }
      }
      if (foundTile) break;
    }

    if (!foundTile) {
      setHoveredTile(null);
    }
  };

  // Handle mouse click for priest spots
  const handleClick = (e: React.MouseEvent<HTMLCanvasElement>): void => {
    const canvas = canvasRef.current;
    if (!canvas || !bonusTiles) return;

    const rect = canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * 2; // Account for canvas scaling
    const y = (e.clientY - rect.top) * 2;

    // Only handle clicks in priest spots area (bottom of canvas)
    const priestAreaY = 960; // Start of priest spots area (460 * 2 + empirical adjustment)
    if (y < priestAreaY) {
      return; // Ignore clicks outside priest spots area
    }

    if (!onBonusTileClick) return;

    // Check if click is on any priest spot
    for (let cultIndex = 0; cultIndex < 4; cultIndex++) {
      const cult = cults[cultIndex];

      const tileWidth = 25;
      const tileHeight = 20;
      const tileSpacing = 5;

      // Calculate centered position (matching hover detection)
      const gridWidth = 2 * tileWidth + tileSpacing;
      const startX = (cultWidth - gridWidth) / 2;
      const cultX = (cultWidth * cultIndex + startX) * 2; // Scale by 2 for mouse coords
      const cultY = 960; // 460 * 2 + empirical adjustment

      const tiles = bonusTiles?.get(cult) || [];

      // Check all 5 spots (4 in 2x2 grid + 1 return spot)
      for (let tileIndex = 0; tileIndex < 5; tileIndex++) {
        let tileX, tileY;

        // Position calculation: first 4 are 2x2 grid, 5th is centered below
        if (tileIndex < 4) {
          const row = Math.floor(tileIndex / 2);
          const col = tileIndex % 2;
          tileX = cultX + col * (tileWidth + tileSpacing) * 2;
          tileY = cultY + row * (tileHeight + tileSpacing) * 2;
        } else {
          // 5th spot: centered below the grid
          tileX = cultX + (tileWidth / 2 + tileSpacing / 2) * 2;
          tileY = cultY + 2 * (tileHeight + tileSpacing) * 2;
        }

        if (x >= tileX && x <= tileX + tileWidth * 2 &&
          y >= tileY && y <= tileY + tileHeight * 2) {
          // Only click if spot doesn't have a priest
          const tile = tiles[tileIndex];
          const hasPriest = tile?.priests && tile.priests > 0 && tile.faction !== undefined;

          if (!hasPriest) {
            onBonusTileClick(cult, tileIndex);
          }
          return;
        }
      }
    }
  };

  return (
    <div className="cult-tracks w-full">
      <canvas
        ref={canvasRef}
        width={500} // 250 * 2 for scale
        height={1120} // 560 * 2 for scale
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'contain',
          border: '1px solid #333',
          display: 'block',
          cursor: hoveredTile ? 'pointer' : 'default'
        }}
        onMouseMove={handleMouseMove}
        onMouseLeave={() => { setHoveredTile(null); }}
        onClick={handleClick}
      />
    </div>
  );
};
