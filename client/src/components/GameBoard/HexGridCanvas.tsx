// Canvas-based hex grid renderer - based on terra-mystica/stc/game.js
import React, { useEffect, useRef, useCallback, useMemo } from 'react';
import type { MapHexData } from '../../data/baseGameMap';
import { hexCenter, HEX_SIZE } from '../../utils/hexUtils';
import { TERRAIN_COLORS, FACTION_COLORS, getContrastColor } from '../../utils/colors';
import type { Building, Bridge } from '../../types/game.types';
import { BuildingType } from '../../types/game.types';

interface HexGridCanvasProps {
  hexes: MapHexData[];
  buildings?: Map<string, Building>;
  bridges?: Bridge[];
  highlightedHexes?: Set<string>;
  onHexClick?: (q: number, r: number) => void;
  onBridgeEdgeClick?: (from: { q: number; r: number }, to: { q: number; r: number }) => void;
  bridgeEdgeSelectionEnabled?: boolean;
  onHexHover?: (q: number, r: number) => void;
  showCoords?: boolean;
  disableHover?: boolean;
}

export const HexGridCanvas: React.FC<HexGridCanvasProps> = ({
  hexes,
  buildings = new Map<string, Building>(),
  bridges = [],
  highlightedHexes = new Set(),
  onHexClick,
  onBridgeEdgeClick,
  bridgeEdgeSelectionEnabled = false,
  onHexHover,
  showCoords = true,
  disableHover: _disableHover = false,
}): React.ReactElement => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // Calculate canvas dimensions
  const getDimensions = (): { width: number; height: number; offsetX: number; offsetY: number } => {
    let minX = Infinity, maxX = -Infinity;
    let minY = Infinity, maxY = -Infinity;

    hexes.forEach(hex => {
      const center = hexCenter(hex.coord.r, hex.coord.q);
      minX = Math.min(minX, center.x);
      maxX = Math.max(maxX, center.x);
      minY = Math.min(minY, center.y);
      maxY = Math.max(maxY, center.y);
    });

    const paddingX = HEX_SIZE;
    const paddingY = HEX_SIZE * 2;

    return {
      width: maxX - minX + paddingX * 2,
      height: maxY - minY + paddingY * 2,
      offsetX: -minX + paddingX,
      offsetY: -minY + paddingY,
    };
  };

  const dims = getDimensions();

  const rotate60 = (coord: { q: number; r: number }, turns: number): { q: number; r: number } => {
    let x = coord.q;
    let z = coord.r;
    let y = -x - z;
    for (let i = 0; i < turns % 6; i++) {
      [x, y, z] = [-z, -x, -y];
    }
    return { q: x, r: z };
  };

  const bridgeCandidates = useMemo(() => {
    const byKey = new Map<string, MapHexData>();
    hexes.forEach((hex) => {
      byKey.set(`${String(hex.coord.q)},${String(hex.coord.r)}`, hex);
    });

    const candidates: Array<{
      from: { q: number; r: number };
      to: { q: number; r: number };
      start: { x: number; y: number };
      end: { x: number; y: number };
    }> = [];
    const seen = new Set<string>();
    const baseTarget = { q: 1, r: -2 };
    const baseMidA = { q: 0, r: -1 };
    const baseMidB = { q: 1, r: -1 };

    hexes.forEach((source) => {
      if (source.isRiver) return;

      for (let rot = 0; rot < 6; rot++) {
        const targetDelta = rotate60(baseTarget, rot);
        const midADelta = rotate60(baseMidA, rot);
        const midBDelta = rotate60(baseMidB, rot);

        const target = { q: source.coord.q + targetDelta.q, r: source.coord.r + targetDelta.r };
        const midA = { q: source.coord.q + midADelta.q, r: source.coord.r + midADelta.r };
        const midB = { q: source.coord.q + midBDelta.q, r: source.coord.r + midBDelta.r };

        const targetHex = byKey.get(`${String(target.q)},${String(target.r)}`);
        const midAHex = byKey.get(`${String(midA.q)},${String(midA.r)}`);
        const midBHex = byKey.get(`${String(midB.q)},${String(midB.r)}`);
        if (!targetHex || targetHex.isRiver) continue;
        if (!midAHex?.isRiver || !midBHex?.isRiver) continue;

        const keyParts = [
          `${String(source.coord.q)},${String(source.coord.r)}`,
          `${String(target.q)},${String(target.r)}`,
        ].sort();
        const key = keyParts.join('|');
        if (seen.has(key)) continue;
        seen.add(key);

        const fromCenter = hexCenter(source.coord.r, source.coord.q);
        const toCenter = hexCenter(target.r, target.q);
        const midX = (fromCenter.x + toCenter.x) / 2;
        const midY = (fromCenter.y + toCenter.y) / 2;
        const dx = toCenter.x - fromCenter.x;
        const dy = toCenter.y - fromCenter.y;
        const scale = 0.3;

        candidates.push({
          from: { q: source.coord.q, r: source.coord.r },
          to: target,
          start: { x: midX - dx * scale, y: midY - dy * scale },
          end: { x: midX + dx * scale, y: midY + dy * scale },
        });
      }
    });

    return candidates;
  }, [hexes]);

  // Draw a hex path (from terra-mystica/stc/game.js makeHexPath)
  const makeHexPath = (ctx: CanvasRenderingContext2D, x: number, y: number, size: number): void => {
    let angle = 0;
    ctx.beginPath();
    ctx.moveTo(x, y);
    for (let i = 0; i < 6; i++) {
      ctx.lineTo(x, y);
      angle += Math.PI / 3;
      x += Math.sin(angle) * size;
      y += Math.cos(angle) * size;
    }
    ctx.closePath();
  };

  // Draw hex for a given coordinate
  const drawHex = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    const center = hexCenter(hex.coord.r, hex.coord.q);
    const x = center.x - Math.cos(Math.PI / 6) * HEX_SIZE;
    const y = center.y + Math.sin(Math.PI / 6) * HEX_SIZE;

    makeHexPath(ctx, x, y, HEX_SIZE);

    // Fill with terrain color
    const fillColor = hex.isRiver ? '#b3d9ff' : TERRAIN_COLORS[hex.terrain];
    ctx.fillStyle = fillColor;
    ctx.fill();

    // Stroke (normal border only, highlight drawn separately later)
    ctx.strokeStyle = '#333';
    ctx.lineWidth = 1;
    ctx.stroke();

    // Show coordinates when showCoords is true
    if (showCoords) {
      ctx.save();
      ctx.fillStyle = getContrastColor(fillColor);
      ctx.font = '10px sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(`${String(hex.coord.q)},${String(hex.coord.r)}`, center.x, center.y);
      ctx.restore();
    }
  }, [showCoords]);

  // Draw highlight border on top of everything
  const drawHighlight = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    const center = hexCenter(hex.coord.r, hex.coord.q);
    const x = center.x - Math.cos(Math.PI / 6) * HEX_SIZE;
    const y = center.y + Math.sin(Math.PI / 6) * HEX_SIZE;

    makeHexPath(ctx, x, y, HEX_SIZE);

    ctx.strokeStyle = '#FFD700'; // Gold color for highlight
    ctx.lineWidth = 3;
    ctx.stroke();
  }, []);

  // Draw dwelling (from terra-mystica/stc/game.js)
  const drawDwelling = useCallback((ctx: CanvasRenderingContext2D, r: number, q: number, color: string) => {
    const loc = hexCenter(r, q);
    const contrastColor = getContrastColor(color);

    ctx.save();
    ctx.beginPath();
    ctx.moveTo(loc.x, loc.y - 10);
    ctx.lineTo(loc.x + 10, loc.y);
    ctx.lineTo(loc.x + 10, loc.y + 10);
    ctx.lineTo(loc.x - 10, loc.y + 10);
    ctx.lineTo(loc.x - 10, loc.y);
    ctx.closePath();

    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = contrastColor;
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.restore();
  }, []);

  // Draw trading house (from terra-mystica/stc/game.js)
  const drawTradingPost = useCallback((ctx: CanvasRenderingContext2D, r: number, q: number, color: string) => {
    const loc = hexCenter(r, q);
    const contrastColor = getContrastColor(color);

    ctx.save();
    ctx.beginPath();
    ctx.moveTo(loc.x, loc.y - 20);
    ctx.lineTo(loc.x + 10, loc.y - 10);
    ctx.lineTo(loc.x + 10, loc.y - 3);
    ctx.lineTo(loc.x + 20, loc.y - 3);
    ctx.lineTo(loc.x + 20, loc.y + 10);
    ctx.lineTo(loc.x - 10, loc.y + 10);
    ctx.lineTo(loc.x - 10, loc.y);
    ctx.lineTo(loc.x - 10, loc.y - 10);
    ctx.closePath();

    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = contrastColor;
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.restore();
  }, []);

  // Draw temple (from terra-mystica/stc/game.js)
  const drawTemple = useCallback((ctx: CanvasRenderingContext2D, r: number, q: number, color: string) => {
    const loc = hexCenter(r, q);
    const contrastColor = getContrastColor(color);

    ctx.save();
    ctx.beginPath();
    ctx.arc(loc.x, loc.y - 5, 14, 0, Math.PI * 2);
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = contrastColor;
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.restore();
  }, []);

  // Draw stronghold (from terra-mystica/stc/game.js)
  const drawStronghold = useCallback((ctx: CanvasRenderingContext2D, r: number, q: number, color: string) => {
    const loc = hexCenter(r, q);
    const contrastColor = getContrastColor(color);
    const yOffset = loc.y - 5;
    const size = 15;
    const bend = 10;

    ctx.save();

    ctx.beginPath();
    ctx.moveTo(loc.x - size, yOffset - size);
    ctx.quadraticCurveTo(loc.x - bend, yOffset,
      loc.x - size, yOffset + size);
    ctx.quadraticCurveTo(loc.x, yOffset + bend,
      loc.x + size, yOffset + size);
    ctx.quadraticCurveTo(loc.x + bend, yOffset,
      loc.x + size, yOffset - size);
    ctx.quadraticCurveTo(loc.x, yOffset - bend,
      loc.x - size, yOffset - size);

    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = contrastColor;
    ctx.lineWidth = 2;
    ctx.stroke();

    ctx.restore();
  }, []);

  // Draw sanctuary (from terra-mystica/stc/game.js)
  const drawSanctuary = useCallback((ctx: CanvasRenderingContext2D, r: number, q: number, color: string) => {
    const loc = hexCenter(r, q);
    const contrastColor = getContrastColor(color);
    const yOffset = loc.y - 5;
    const size = 7;

    ctx.save();

    ctx.beginPath();
    ctx.arc(loc.x - size, yOffset, 12, Math.PI / 2, -Math.PI / 2, false);
    ctx.arc(loc.x + size, yOffset, 12, -Math.PI / 2, Math.PI / 2, false);
    ctx.closePath();

    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = contrastColor;
    ctx.lineWidth = 2;
    ctx.stroke();

    ctx.restore();
  }, []);

  // Draw a building
  const drawBuilding = useCallback((ctx: CanvasRenderingContext2D, building: Building, r: number, q: number) => {
    const color = FACTION_COLORS[building.faction];

    switch (building.type) {
      case BuildingType.Dwelling:
        drawDwelling(ctx, r, q, color);
        break;
      case BuildingType.TradingHouse:
        drawTradingPost(ctx, r, q, color);
        break;
      case BuildingType.Temple:
        drawTemple(ctx, r, q, color);
        break;
      case BuildingType.Stronghold:
        drawStronghold(ctx, r, q, color);
        break;
      case BuildingType.Sanctuary:
        drawSanctuary(ctx, r, q, color);
        break;
    }
  }, [drawDwelling, drawTradingPost, drawTemple, drawStronghold, drawSanctuary]);

  // Draw bridge along hex edge (from terra-mystica/stc/game.js)
  // Bridges connect hexes at distance 2 (not adjacent) across river edges
  const drawBridge = useCallback((ctx: CanvasRenderingContext2D, bridge: Bridge) => {
    const from = hexCenter(bridge.fromCoord.r, bridge.fromCoord.q);
    const to = hexCenter(bridge.toCoord.r, bridge.toCoord.q);
    const color = FACTION_COLORS[bridge.faction];

    // Calculate midpoint and shorten the bridge to look like it's on the edge
    // Bridge should be about 60% of the full distance, centered on the midpoint
    const midX = (from.x + to.x) / 2;
    const midY = (from.y + to.y) / 2;
    const dx = to.x - from.x;
    const dy = to.y - from.y;
    const scale = 0.3; // How far from midpoint to extend (30% each way = 60% total)

    const startX = midX - dx * scale;
    const startY = midY - dy * scale;
    const endX = midX + dx * scale;
    const endY = midY + dy * scale;

    ctx.save();

    ctx.beginPath();
    ctx.moveTo(startX, startY);
    ctx.lineTo(endX, endY);

    // Dark outline
    ctx.strokeStyle = '#222';
    ctx.lineWidth = 10;
    ctx.stroke();

    // Faction color
    ctx.strokeStyle = color;
    ctx.lineWidth = 8;
    ctx.stroke();

    ctx.restore();
  }, []);

  // Render the canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Apply transform
    ctx.save();
    ctx.translate(dims.offsetX, dims.offsetY);

    // Z-order: River hexes → Bridges → Land hexes → Buildings → Highlights

    // 1. Draw river hexes first
    hexes.forEach(hex => {
      if (hex.isRiver) {
        drawHex(ctx, hex);
      }
    });

    // 2. Draw bridges (on top of river hexes, below land hexes)
    bridges.forEach(bridge => {
      drawBridge(ctx, bridge);
    });

    // 3. Draw land hexes (non-river)
    hexes.forEach(hex => {
      if (!hex.isRiver) {
        drawHex(ctx, hex);
      }
    });

    // 4. Draw buildings
    buildings.forEach((building, key) => {
      const [q, r] = key.split(',').map(Number);
      drawBuilding(ctx, building, r, q);
    });

    // 5. Draw highlights on top of everything
    hexes.forEach(hex => {
      const key = `${String(hex.coord.q)},${String(hex.coord.r)}`;
      if (highlightedHexes.has(key)) {
        drawHighlight(ctx, hex);
      }
    });

    ctx.restore();
  }, [hexes, buildings, bridges, highlightedHexes, dims.offsetX, dims.offsetY, drawHex, drawBridge, drawBuilding, drawHighlight]);

  const pointToSegmentDistance = (
    px: number,
    py: number,
    ax: number,
    ay: number,
    bx: number,
    by: number,
  ): number => {
    const abx = bx - ax;
    const aby = by - ay;
    const apx = px - ax;
    const apy = py - ay;
    const abLenSq = abx * abx + aby * aby;
    if (abLenSq === 0) {
      const dx = px - ax;
      const dy = py - ay;
      return Math.sqrt(dx * dx + dy * dy);
    }
    const t = Math.max(0, Math.min(1, (apx * abx + apy * aby) / abLenSq));
    const closestX = ax + t * abx;
    const closestY = ay + t * aby;
    const dx = px - closestX;
    const dy = py - closestY;
    return Math.sqrt(dx * dx + dy * dy);
  };

  // Handle mouse events
  const handleMouseEvent = (e: React.MouseEvent<HTMLCanvasElement>, isClick: boolean): void => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();

    // Calculate scaling factor between CSS size (rect) and internal canvas size (dims)
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;

    // Apply scaling to mouse coordinates
    const x = (e.clientX - rect.left) * scaleX - dims.offsetX;
    const y = (e.clientY - rect.top) * scaleY - dims.offsetY;

    if (isClick && bridgeEdgeSelectionEnabled && onBridgeEdgeClick) {
      let bestFrom: { q: number; r: number } | null = null;
      let bestTo: { q: number; r: number } | null = null;
      let bestDist = Infinity;
      bridgeCandidates.forEach((candidate) => {
        const dist = pointToSegmentDistance(
          x,
          y,
          candidate.start.x,
          candidate.start.y,
          candidate.end.x,
          candidate.end.y,
        );
        if (dist < bestDist) {
          bestDist = dist;
          bestFrom = candidate.from;
          bestTo = candidate.to;
        }
      });
      if (bestFrom !== null && bestTo !== null && bestDist <= 12) {
        onBridgeEdgeClick(bestFrom, bestTo);
        return;
      }
    }

    // Find which hex was clicked (simple distance check)
    let closestHex: MapHexData | null = null;
    let closestDist = Infinity;

    hexes.forEach(hex => {
      const center = hexCenter(hex.coord.r, hex.coord.q);
      const dx = center.x - x;
      const dy = center.y - y;
      const dist = Math.sqrt(dx * dx + dy * dy);

      if (dist < HEX_SIZE && dist < closestDist) {
        closestDist = dist;
        closestHex = hex;
      }
    });

    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (!closestHex) {
      return;
    }

    // Type assertion: we know closestHex is not null here
    const hex = closestHex as MapHexData;
    if (isClick) {
      onHexClick?.(hex.coord.q, hex.coord.r);
    } else {
      onHexHover?.(hex.coord.q, hex.coord.r);
    }
  };

  return (
    <canvas
      ref={canvasRef}
      width={dims.width}
      height={dims.height}
      style={{
        width: '100%',
        height: 'auto', // Maintain aspect ratio
        maxWidth: '100%',
        maxHeight: '100%',
        border: '1px solid #ccc',
        backgroundColor: '#f0f0f0',
        cursor: 'pointer',
        display: 'block' // Remove inline spacing
      }}
      onClick={(e) => { handleMouseEvent(e, true); }}
      onMouseMove={(e) => { handleMouseEvent(e, false); }}
    />
  );
};
