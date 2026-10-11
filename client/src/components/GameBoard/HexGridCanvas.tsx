// Canvas-based hex grid renderer - based on terra-mystica/stc/game.js
import React, { useEffect, useRef, useCallback, useMemo, useLayoutEffect, useState } from 'react';
import type { MapHexData } from '../../types/map.types';
import { buildDisplayCoordinateMap, formatDisplayCoordinate, getDisplayCoordinate, hexCenter, HEX_SIZE } from '../../utils/hexUtils';
import { TERRAIN_COLORS, FACTION_COLORS, getContrastColor } from '../../utils/colors';
import type { Building, Bridge } from '../../types/game.types';
import { BuildingType, FactionType, TerrainType } from '../../types/game.types';
import { TOWN_TILES } from '../../data/townTiles';
import { TownTileFace } from '../shared/TownTileFace';
import { drawTownTileFace } from '../shared/townTileGeometry';
import { gameThemeColor } from '../../utils/gameTheme';

interface HexGridCanvasProps {
  hexes: MapHexData[];
  buildings?: Map<string, Building>;
  playerFactions?: Map<string, FactionType>;
  bridges?: Bridge[];
  highlightedHexes?: Set<string>;
  onHexClick?: (q: number, r: number) => void;
  onBridgeEdgeClick?: (from: { q: number; r: number }, to: { q: number; r: number }) => void;
  bridgeEdgeSelectionEnabled?: boolean;
  onHexHover?: (q: number, r: number) => void;
  showCoords?: boolean;
  disableHover?: boolean;
  testId?: string;
}

export const HexGridCanvas: React.FC<HexGridCanvasProps> = ({
  hexes,
  buildings = new Map<string, Building>(),
  playerFactions = new Map<string, FactionType>(),
  bridges = [],
  highlightedHexes = new Set(),
  onHexClick,
  onBridgeEdgeClick,
  bridgeEdgeSelectionEnabled = false,
  onHexHover,
  showCoords = true,
  disableHover: _disableHover = false,
  testId,
}): React.ReactElement => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [hoveredTown, setHoveredTown] = useState<string | null>(null);
  const [keyboardPreview, setKeyboardPreview] = useState(false);
  const [keyboardHex, setKeyboardHex] = useState(0);
  const [keyboardFocused, setKeyboardFocused] = useState(false);
  const [canvasMetrics, setCanvasMetrics] = useState(() => ({
    cssWidth: 1,
    cssHeight: 1,
    backingWidth: 1,
    backingHeight: 1,
    backingScale: 1,
  }));
  const displayCoordinates = useMemo(() => buildDisplayCoordinateMap(hexes), [hexes])

  // Calculate canvas dimensions
  const dims = useMemo((): { width: number; height: number; offsetX: number; offsetY: number } => {
    if (hexes.length === 0) {
      return {
        width: 1,
        height: 1,
        offsetX: 0,
        offsetY: 0,
      };
    }

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
  }, [hexes]);

  useLayoutEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const host = canvas.parentElement ?? canvas;

    const syncCanvasMetrics = (): void => {
      const nextCssWidth = Math.max(host.clientWidth || dims.width, 1);
      const nextCssHeight = Math.max((nextCssWidth / dims.width) * dims.height, 1);
      const pixelRatio = window.devicePixelRatio || 1;
      const nextBackingWidth = Math.max(1, Math.round(nextCssWidth * pixelRatio));
      const nextBackingHeight = Math.max(1, Math.round(nextCssHeight * pixelRatio));
      const nextBackingScale = nextBackingWidth / dims.width;

      setCanvasMetrics((current) => {
        if (
          current.cssWidth === nextCssWidth
          && current.cssHeight === nextCssHeight
          && current.backingWidth === nextBackingWidth
          && current.backingHeight === nextBackingHeight
          && current.backingScale === nextBackingScale
        ) {
          return current;
        }

        return {
          cssWidth: nextCssWidth,
          cssHeight: nextCssHeight,
          backingWidth: nextBackingWidth,
          backingHeight: nextBackingHeight,
          backingScale: nextBackingScale,
        };
      });
    };

    syncCanvasMetrics();

    const resizeObserver = new ResizeObserver(() => {
      syncCanvasMetrics();
    });
    resizeObserver.observe(host);
    window.addEventListener('resize', syncCanvasMetrics);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener('resize', syncCanvasMetrics);
    };
  }, [dims.height, dims.width]);

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

  }, []);

  // Labels stay readable above pieces and town tiles, including when markings
  // are restored after a player temporarily hides them.
  const drawCoordinate = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    if (!showCoords) return;
    const displayCoord = getDisplayCoordinate(hex.coord, displayCoordinates);
    if (displayCoord === null) return;
    const center = hexCenter(hex.coord.r, hex.coord.q);
    const occupied = buildings.has(`${String(hex.coord.q)},${String(hex.coord.r)}`) || hex.hasTownTile;
    const fillColor = hex.hasTownTile ? '#f8eed1' : TERRAIN_COLORS[hex.terrain];
    ctx.save();
    ctx.fillStyle = getContrastColor(fillColor);
    ctx.font = occupied ? '600 7px sans-serif' : '10px sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(displayCoord, center.x + (hex.hasTownTile ? -21 : 0), center.y + (occupied && !hex.hasTownTile ? -24 : 0));
    ctx.restore();
  }, [buildings, displayCoordinates, showCoords]);

  // Draw highlight border on top of everything
  const drawHighlight = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    const center = hexCenter(hex.coord.r, hex.coord.q);
    const x = center.x - Math.cos(Math.PI / 6) * HEX_SIZE;
    const y = center.y + Math.sin(Math.PI / 6) * HEX_SIZE;

    makeHexPath(ctx, x, y, HEX_SIZE);

    ctx.strokeStyle = gameThemeColor(ctx.canvas, 'focus');
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

  const drawTownTileMarker = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    if (!hex.hasTownTile || hex.townTileType == null) return;

    const center = hexCenter(hex.coord.r, hex.coord.q);
    const tile = TOWN_TILES[hex.townTileType];
    const ownerFaction = hex.townTileOwnerPlayerId ? playerFactions.get(hex.townTileOwnerPlayerId) : undefined;
    const borderColor = ownerFaction != null ? FACTION_COLORS[ownerFaction] : '#9b773b';
    const size = HEX_SIZE - 1;

    ctx.save();
    // Use the supply tile's face beneath the piece, with a slim faction rim
    // following the anchor hex. Hover exposes artwork covered by buildings.
    drawTownTileFace(ctx, tile, center.x, center.y, (HEX_SIZE - 3) * 2);
    makeHexPath(ctx, center.x - Math.cos(Math.PI / 6) * size, center.y + size / 2, size);
    ctx.strokeStyle = borderColor;
    ctx.lineWidth = 1.5;
    ctx.stroke();
    ctx.restore();
  }, [playerFactions]);

  const drawChildrenRiverToken = useCallback((ctx: CanvasRenderingContext2D, hex: MapHexData) => {
    if (!hex.powerTokenOwnerPlayerId) return;

    const center = hexCenter(hex.coord.r, hex.coord.q);
    const y = center.y + (hex.hasTownTile ? -8 : -2);

    ctx.save();

    ctx.beginPath();
    ctx.arc(center.x, y, 8, 0, Math.PI * 2);
    ctx.fillStyle = '#2c2c2c';
    ctx.fill();
    ctx.strokeStyle = '#0f172a';
    ctx.lineWidth = 1.5;
    ctx.stroke();

    ctx.beginPath();
    ctx.arc(center.x, y, 5.5, 0, Math.PI * 2);
    ctx.fillStyle = '#7C3AED';
    ctx.fill();

    ctx.fillStyle = '#ffffff';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.font = '700 5.5px sans-serif';
    ctx.fillText('PW', center.x, y + 0.2);

    ctx.restore();
  }, []);

  // Render the canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Draw in logical board units, then scale to the rendered CSS size and DPR.
    ctx.save();
    ctx.setTransform(
      canvasMetrics.backingScale,
      0,
      0,
      canvasMetrics.backingScale,
      dims.offsetX * canvasMetrics.backingScale,
      dims.offsetY * canvasMetrics.backingScale,
    );

    // Z-order: River hexes → Bridges → Land hexes → Town tiles → Children tokens → Buildings → Highlights

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

    // 4. Draw town tile markers
    hexes.forEach(hex => {
      if (hex.hasTownTile) {
        drawTownTileMarker(ctx, hex);
      }
    });

    // 5. Draw Children of the Wyrm river power tokens
    hexes.forEach(hex => {
      if (hex.powerTokenOwnerPlayerId) {
        drawChildrenRiverToken(ctx, hex);
      }
    });

    // 6. Draw buildings
    buildings.forEach((building, key) => {
      const [q, r] = key.split(',').map(Number);
      drawBuilding(ctx, building, r, q);
    });

    hexes.forEach(hex => { drawCoordinate(ctx, hex); });

    // 7. Draw highlights on top of everything
    hexes.forEach(hex => {
      const key = `${String(hex.coord.q)},${String(hex.coord.r)}`;
      if (highlightedHexes.has(key) || (keyboardFocused && hex === hexes[keyboardHex])) {
        drawHighlight(ctx, hex);
      }
    });

    ctx.restore();
  }, [
    hexes,
    buildings,
    bridges,
    highlightedHexes,
    keyboardFocused,
    keyboardHex,
    dims.offsetX,
    dims.offsetY,
    canvasMetrics.backingScale,
    drawHex,
    drawCoordinate,
    drawBridge,
    drawTownTileMarker,
    drawChildrenRiverToken,
    drawBuilding,
    drawHighlight,
  ]);

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
    if (rect.width === 0 || rect.height === 0) return;

    // Convert browser event coordinates back into the logical board space.
    const x = ((e.clientX - rect.left) / rect.width) * dims.width - dims.offsetX;
    const y = ((e.clientY - rect.top) / rect.height) * dims.height - dims.offsetY;

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
      if (!isClick) setHoveredTown(null);
      return;
    }

    // Type assertion: we know closestHex is not null here
    const hex = closestHex as MapHexData;
    setKeyboardPreview(false);
    if (isClick) {
      setKeyboardHex(hexes.indexOf(hex));
      onHexClick?.(hex.coord.q, hex.coord.r);
    } else {
      setHoveredTown(hex.hasTownTile ? `${String(hex.coord.q)},${String(hex.coord.r)}` : null);
      onHexHover?.(hex.coord.q, hex.coord.r);
    }
  };

  const focusedHex = hexes[keyboardHex] ?? hexes[0];
  const focusedBuilding = focusedHex ? buildings.get(`${String(focusedHex.coord.q)},${String(focusedHex.coord.r)}`) : undefined;
  const focusedDescription = focusedHex
    ? `${formatDisplayCoordinate(focusedHex.coord, displayCoordinates)}, ${TerrainType[focusedHex.terrain]}${focusedHex.hasTownTile && focusedHex.townTileType !== undefined ? `, ${String(TOWN_TILES[focusedHex.townTileType].vp)} VP town, ${TOWN_TILES[focusedHex.townTileType].description}` : ''}${focusedBuilding ? `, ${FactionType[focusedBuilding.faction]} ${BuildingType[focusedBuilding.type]}, owner ${focusedBuilding.ownerPlayerId}` : ', empty'}`
    : 'empty map';

  const hoveredTownHex = hexes.find(hex => `${String(hex.coord.q)},${String(hex.coord.r)}` === hoveredTown);
  const previewHex = hoveredTownHex ?? (keyboardFocused && keyboardPreview && focusedHex?.hasTownTile ? focusedHex : undefined);
  const previewTile = previewHex?.townTileType !== undefined ? TOWN_TILES[previewHex.townTileType] : undefined;
  const previewCenter = previewHex ? hexCenter(previewHex.coord.r, previewHex.coord.q) : { x: 0, y: 0 };
  const cssScale = canvasMetrics.cssWidth / dims.width;
  const previewOwner = previewHex?.townTileOwnerPlayerId ? playerFactions.get(previewHex.townTileOwnerPlayerId) : undefined;

  return (
    <div className="hex-board-interaction" style={{ position: 'relative', width: '100%' }}>
    <canvas
      ref={canvasRef}
      data-testid={testId}
      data-show-coords={showCoords}
      data-logical-width={dims.width}
      data-logical-height={dims.height}
      tabIndex={onHexClick ? 0 : undefined}
      role={onHexClick ? 'button' : 'img'}
      aria-label={`Board: ${focusedDescription}.${onHexClick ? ' Arrow keys move, Enter or Space selects.' : ''}`}
      onFocus={() => { setKeyboardFocused(true); setKeyboardPreview(true); }}
      onBlur={() => { setKeyboardFocused(false); }}
      onKeyDown={(event) => {
        const current = hexes[keyboardHex] ?? hexes[0];
        if (!current) return;
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onHexClick?.(current.coord.q, current.coord.r);
          return;
        }
        if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
        event.preventDefault();
        setKeyboardPreview(true);
        const origin = hexCenter(current.coord.r, current.coord.q);
        const direction = event.key;
        const candidates = hexes.map((hex, index) => {
          const point = hexCenter(hex.coord.r, hex.coord.q);
          const dx = point.x - origin.x;
          const dy = point.y - origin.y;
          const along = direction === 'ArrowRight' ? dx : direction === 'ArrowLeft' ? -dx : direction === 'ArrowDown' ? dy : -dy;
          const across = direction === 'ArrowRight' || direction === 'ArrowLeft' ? Math.abs(dy) : Math.abs(dx);
          return { index, along, score: along + across * 2 };
        }).filter(candidate => candidate.along > 1).sort((a, b) => a.score - b.score);
        const next = direction === 'Home' ? 0 : direction === 'End' ? hexes.length - 1 : candidates[0]?.index ?? (hexes[keyboardHex] ? keyboardHex : 0);
        setKeyboardHex(next);
        onHexHover?.(hexes[next].coord.q, hexes[next].coord.r);
      }}
      width={canvasMetrics.backingWidth}
      height={canvasMetrics.backingHeight}
      style={{
        width: `${String(canvasMetrics.cssWidth)}px`,
        height: `${String(canvasMetrics.cssHeight)}px`,
        border: '1px solid var(--game-border, #ccc)',
        backgroundColor: 'var(--game-surface-muted, #f0f0f0)',
        cursor: 'pointer',
        display: 'block',
        boxSizing: 'border-box',
      }}
      onClick={(e) => { handleMouseEvent(e, true); }}
      onPointerMove={(e) => { handleMouseEvent(e, false); }}
      onPointerLeave={() => { setHoveredTown(null); }}
    />
    {previewHex?.townTileType !== undefined && previewTile && <div role="tooltip" data-testid="town-tile-preview" style={{
      position: 'absolute', pointerEvents: 'none', zIndex: 20, width: 120, padding: 8,
      left: Math.max(0, Math.min(canvasMetrics.cssWidth - 120, (previewCenter.x + dims.offsetX) * cssScale + 20)),
      top: Math.max(0, (previewCenter.y + dims.offsetY) * cssScale - 152),
      background: '#fff', border: '1px solid var(--game-border)', color: '#252b32', boxSizing: 'border-box',
    }}>
      <div style={{ width: 80, height: 92, margin: '0 auto 4px' }}><TownTileFace tileId={previewHex.townTileType} borderColor={previewOwner !== undefined ? FACTION_COLORS[previewOwner] : undefined} /></div>
      <div style={{ fontSize: 12, textAlign: 'center', lineHeight: 1.25 }}>{previewTile.vp} VP · {previewTile.description}</div>
    </div>}
    {bridgeEdgeSelectionEnabled && onBridgeEdgeClick && (
      <label className="board-keyboard-bridges">
        Bridge endpoints
        <select aria-label="Choose bridge endpoints" value="" onChange={(event) => {
          const candidate = bridgeCandidates[Number(event.target.value)];
          if (candidate) onBridgeEdgeClick(candidate.from, candidate.to);
        }}>
          <option value="" disabled>Select an edge</option>
          {bridgeCandidates.map((candidate, index) => <option key={index} value={index}>
            {formatDisplayCoordinate(candidate.from, displayCoordinates)} to {formatDisplayCoordinate(candidate.to, displayCoordinates)}
          </option>)}
        </select>
      </label>
    )}
    </div>
  );
};
