// Canvas-based hex grid renderer - based on terra-mystica/stc/game.js
import React, { useEffect, useRef, useCallback } from 'react';
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
  onHexHover?: (q: number, r: number) => void;
}

export const HexGridCanvas: React.FC<HexGridCanvasProps> = ({
  hexes,
  buildings = new Map(),
  bridges = [],
  highlightedHexes = new Set(),
  onHexClick,
  onHexHover,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // Calculate canvas dimensions
  const getDimensions = () => {
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

  // Draw a hex path (from terra-mystica/stc/game.js makeHexPath)
  const makeHexPath = (ctx: CanvasRenderingContext2D, x: number, y: number, size: number) => {
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

    // Debug: show coordinates in dev mode
    if (process.env.NODE_ENV === 'development') {
      ctx.save();
      ctx.fillStyle = getContrastColor(fillColor);
      ctx.font = '10px sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(`${hex.coord.q},${hex.coord.r}`, center.x, center.y);
      ctx.restore();
    }
  }, []);

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
      const key = `${hex.coord.q},${hex.coord.r}`;
      if (highlightedHexes.has(key)) {
        drawHighlight(ctx, hex);
      }
    });

    ctx.restore();
  }, [hexes, buildings, bridges, highlightedHexes, dims.offsetX, dims.offsetY, drawHex, drawBridge, drawBuilding, drawHighlight]);

  // Handle mouse events
  const handleMouseEvent = (e: React.MouseEvent<HTMLCanvasElement>, isClick: boolean) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();

    // Calculate scaling factor between CSS size (rect) and internal canvas size (dims)
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;

    // Apply scaling to mouse coordinates
    const x = (e.clientX - rect.left) * scaleX - dims.offsetX;
    const y = (e.clientY - rect.top) * scaleY - dims.offsetY;

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

    if (closestHex) {
      if (isClick) {
        onHexClick?.(closestHex.coord.q, closestHex.coord.r);
      } else {
        onHexHover?.(closestHex.coord.q, closestHex.coord.r);
      }
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
      onClick={(e) => handleMouseEvent(e, true)}
      onMouseMove={(e) => handleMouseEvent(e, false)}
    />
  );
};
