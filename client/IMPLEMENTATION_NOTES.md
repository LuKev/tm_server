# Frontend Implementation Notes

## Overview

This frontend is a TypeScript port of the `terra-mystica/stc` JavaScript implementation, using Canvas rendering for simplicity and performance.

## Implementation Strategy

### What We Reused from Reference
✅ **Drawing functions** - Direct ports from `terra-mystica/stc/game.js`:
- `makeHexPath()` - Hex outline drawing
- `drawDwelling()` - Pentagon house shape
- `drawTradingPost()` - House with chimney
- `drawTemple()` - Circle
- `drawStronghold()` - Rounded square  
- `drawSanctuary()` - Double circle (peanut)

✅ **Rendering approach**:
- Canvas-based (not SVG)
- Single canvas element with context transforms
- Redraw on state changes

✅ **Visual style**:
- Same hex size (35px)
- Same colors for terrains and factions
- Same building shapes

### Key Differences from Reference

#### 1. Coordinate System ⚠️ IMPORTANT

**Reference implementation (`terra-mystica/stc/game.js`):**
```javascript
function hexCenter(row, col) {
    var x_offset = row % 2 ? hex_width / 2 : 0;
    var x = 5 + hex_size + col * hex_width + x_offset;
    var y = 5 + hex_size + row * hex_height;
    return [x, y];
}
```

**Our implementation (`client/src/utils/hexUtils.ts`):**
```typescript
export function hexCenter(row: number, col: number): PixelCoord {
  const oddRowOffset = row % 2 ? HEX_WIDTH / 2 : 0;
  const progressiveOffset = Math.floor(row / 2) * HEX_WIDTH;  // ← NEW
  
  return {
    x: 5 + HEX_SIZE + col * HEX_WIDTH + oddRowOffset + progressiveOffset,
    y: 5 + HEX_SIZE + row * HEX_HEIGHT,
  };
}
```

**Why the difference?**
- Backend uses **offset coordinates** where `q` (column) ranges from `-4` to `12`
- Negative q values require progressive shifting of row pairs
- Without this, hexes don't align properly (e.g., (0,1) wouldn't be adjacent to (-1,2))

**Visual result:**
- Rows 0-1: +0 offset
- Rows 2-3: +1 hex width offset  
- Rows 4-5: +2 hex width offset
- Rows 6-7: +3 hex width offset
- Row 8: +4 hex width offset

This creates the proper parallelogram/diamond shape for the Terra Mystica map.

#### 2. TypeScript Types

Added strict typing for all game entities:
```typescript
interface AxialCoord { q: number; r: number }
interface PixelCoord { x: number; y: number }
interface MapHexData {
  coord: AxialCoord;
  terrain: TerrainType;
  isRiver: boolean;
}
```

All types mirror the Go backend structs in `server/internal/models/`.

#### 3. React Component Structure

**Reference**: Monolithic JavaScript with global state
**Ours**: React components with Zustand state management
- `HexGridCanvas.tsx` - Canvas renderer
- `GameBoard.tsx` - Container with state
- `gameStore.ts` - Zustand store for game state

#### 4. Canvas Sizing

**Reference**: Fixed canvas size or manual calculation
**Ours**: Dynamic sizing based on actual hex positions
```typescript
// Calculate actual bounds from hex data
hexes.forEach(hex => {
  const center = hexCenter(hex.coord.r, hex.coord.q);
  minX = Math.min(minX, center.x);
  maxX = Math.max(maxX, center.x);
  // ...
});
const width = maxX - minX + padding * 2;
```

This ensures tight fit regardless of which hexes are present.

#### 5. Bridge Rendering

**Reference**: `drawBridge()` connects adjacent hex centers
**Ours**: ✅ **COMPLETED**: Bridges drawn along hex edges between distance-2 hexes
- Spans across river hexes along edges
- Shortened to 60% length centered on midpoint
- Proper Z-order: river hexes → bridges → land hexes → buildings → highlights

#### 6. Cult Tracks Integration

**Reference**: Separate cult display component
**Ours**: ✅ **COMPLETED**: Integrated sidebar layout
- Adjacent to game board with proper flex layout
- Single-row display (number OR faction markers, not both)
- Centered alignment for numbers and markers
- Faction-colored priest markers on bonus tiles
- Bold capital "P" for priests

## What to Reuse Going Forward

### ✅ Can Port Directly
These can be copied from `terra-mystica/stc/*.js` with minimal changes:

1. **Player UI rendering** (`player.js`)
   - Resource displays
   - Building counts
   - Faction info

2. **Cult track display** (`drawCults()` in `game.js`)
   - ✅ **COMPLETED**: Cult track positions with faction markers
   - Power bowl visualization (not yet implemented)

3. **Color schemes** (`common.js`)
   - Already ported to `colors.ts`
   - Can add color-blind mode symbols

### ⚠️ Needs Adaptation

1. **Action handling**
   - Reference uses form submissions
   - We'll use click-based action inference

2. **State management**  
   - Reference uses jQuery/global state
   - We use Zustand + WebSocket

3. **Map data loading**
   - Reference fetches from server
   - We have static `BASE_GAME_MAP` + dynamic buildings

## Testing

**Test page**: http://localhost:5173/maptest

Verify:
- ✅ All 113 hexes render
- ✅ Correct terrain colors
- ✅ Proper hex alignment (check (0,1) is adjacent to (-1,2) and (0,2))
- ✅ Buildings display correctly with faction colors
- ✅ Click detection works
- ✅ Hover highlighting works

## Future Work

### Phase 10: Player Interface
Port from `terra-mystica/stc/player.js`:
- Resource display
- Building inventory
- Available actions

### Phase 11: State Management
- WebSocket integration (already have `WebSocketContext.tsx`)
- Optimistic updates
- Game state sync

### Phase 12: Polish
- Tooltips on hover
- Action validation highlights
- Error messages
- Loading states

## Files Changed from Original Plan

**Removed** (no longer needed with Canvas):
- `Hex.tsx` (SVG component)
- `HexGrid.tsx` (SVG grid)
- `Building.tsx` (SVG buildings)
- All individual building components

**Added**:
- `HexGridCanvas.tsx` (Canvas renderer with all drawing functions)
- `baseGameMap.ts` (Static map data)
- `hexUtils.ts` (Coordinate system with progressive offset)

**Changed approach**:
- ~~Konva.js or PixiJS~~ → Plain Canvas API (simpler)
- ~~SVG components~~ → Canvas drawing functions
- ~~Separate building components~~ → Drawing functions in main canvas

## Summary

**What stayed the same:**
- Visual appearance matches reference
- Drawing logic matches reference
- Canvas-based rendering

**What changed:**
- TypeScript instead of JavaScript
- React components instead of jQuery
- Progressive offset for backend's coordinate system
- Zustand instead of global state

**Result:** A type-safe, modern frontend that looks and behaves like the reference implementation while working with our Go backend's coordinate system.
