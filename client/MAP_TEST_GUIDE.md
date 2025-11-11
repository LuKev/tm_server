# Map Rendering Test Guide

## ✅ What Was Implemented

### Phase 9.1 - Hex Grid System
- ✅ `src/utils/hexUtils.ts` - Hex coordinate utilities
- ✅ `src/data/baseGameMap.ts` - Base game map with all 113 hexes
- ✅ `src/components/GameBoard/Hex.tsx` - Individual hex component
- ✅ `src/components/GameBoard/HexGrid.tsx` - Full grid renderer

### Phase 9.2 - Terrain & Building Visuals
- ✅ `src/utils/colors.ts` - Terrain and faction colors
- ✅ `src/components/GameBoard/Building.tsx` - All 5 building types:
  - Dwelling (pentagon house)
  - Trading House (house with chimney)
  - Temple (circle)
  - Stronghold (rounded square)
  - Sanctuary (double circle)
- ✅ `src/components/GameBoard/GameBoard.tsx` - Main board container
- ✅ `src/components/MapTest.tsx` - Standalone test page

## 🚀 How to Test

### Step 1: Start the Dev Server

```bash
cd client
npm run dev
```

The server should start on `http://localhost:5173`

### Step 2: Open the Map Test Page

Navigate to: **http://localhost:5173/maptest**

You should see:
- Title: "Terra Mystica - Map Test"
- Controls panel with a "Show test buildings" checkbox
- The full base game map with 113 hexes
- Color legend showing all terrain types

### Step 3: Verify the Map

**Visual checks:**
1. ✅ **113 hexes total** - Count should match
2. ✅ **9 rows** (rows 0-8)
3. ✅ **Terrain colors** - Match the legend:
   - Yellow = Desert
   - Brown = Plains
   - Black = Swamp
   - Blue = Lake
   - Green = Forest
   - Gray = Mountain
   - Red = Wasteland
   - Light Blue = River
4. ✅ **River hexes** - Should be light blue with "isRiver" marking
5. ✅ **Hex coordinates** - In dev mode, each hex shows its (q,r) coordinates

**Interactive tests:**
1. ✅ **Hover** - Hovering over a hex should:
   - Highlight it with a green border
   - Display the coordinate at the top (e.g., "Hovering: 0,0")

2. ✅ **Click** - Clicking a hex should:
   - Log the coordinate to console
   - Show an alert with the coordinate

3. ✅ **Buildings toggle** - Check "Show test buildings":
   - Row 0, hex 0: Yellow Dwelling (Nomads)
   - Row 0, hex 1: Green Trading House (Witches)
   - Row 0, hex 2: Blue Temple (Mermaids)
   - Row 0, hex 3: Gray Stronghold (Engineers)
   - Row 0, hex 4: Black Sanctuary (Alchemists)

### Step 4: Verify Building Shapes

With "Show test buildings" enabled, verify shapes:

**Dwelling** (hex 0,0):
- Pentagon shape (house with pointed roof)
- Yellow color

**Trading House** (hex 1,0):
- House with chimney on right side
- Green color

**Temple** (hex 2,0):
- Simple circle
- Blue color

**Stronghold** (hex 3,0):
- Rounded square
- Gray color

**Sanctuary** (hex 4,0):
- Two circles side-by-side (peanut shape)
- Black color

### Step 5: Check the Console

Open browser DevTools (F12) and check:
- No errors in console
- Clicking hexes logs: `Clicked hex (q, r)`
- No TypeScript errors
- No React warnings

## 📊 Expected Results

### Successful Test Checklist

- [ ] Map renders without errors
- [ ] 113 hexes visible
- [ ] All 7 terrain colors display correctly
- [ ] River hexes are light blue
- [ ] Hover highlights hexes with green border
- [ ] Click shows alert with coordinates
- [ ] All 5 building types render correctly
- [ ] Buildings have correct colors per faction
- [ ] No console errors
- [ ] Coordinates visible in dev mode

### Common Issues & Solutions

**Issue: Map doesn't render**
- Check console for errors
- Verify `npm run dev` is running
- Check that all imports are correct

**Issue: Buildings don't show**
- Make sure "Show test buildings" is checked
- Check that hex (0,0) through (4,0) are visible

**Issue: Colors look wrong**
- Check `src/utils/colors.ts` color definitions
- Verify terrain enum matches

**Issue: Hexes overlap or misaligned**
- Check `src/utils/hexUtils.ts` coordinate calculations
- Verify HEX_SIZE constant

## 🎯 Next Steps

After verifying the map renders correctly:

1. **Test in actual game** - Navigate to `/game/test` to see the map in game context
2. **Add click handlers** - Implement action logic when clicking hexes
3. **Add player state** - Show current player's buildings
4. **Add action highlights** - Show valid placement locations
5. **Add tooltips** - Display hex info on hover

## 📝 Notes

- Map uses SVG rendering (simple, no animations)
- Coordinates in dev mode help with debugging
- Base map matches `server/internal/game/terrain_layout.go`
- River hexes are marked for shipping calculations
- Building colors come from faction types

## 🐛 Debugging Tips

**To see raw hex data:**
```typescript
console.log('Total hexes:', BASE_GAME_MAP.length);
console.log('First hex:', BASE_GAME_MAP[0]);
```

**To verify a specific hex:**
```typescript
const hex = BASE_GAME_MAP.find(h => h.coord.q === 0 && h.coord.r === 0);
console.log('Hex (0,0):', hex);
```

**To check building rendering:**
Open DevTools → Elements and inspect the SVG elements.
Each building should be a `<g>` or `<path>` element with appropriate styling.

## ✨ Success Criteria

The map test is successful when:
1. All 113 hexes render in correct positions
2. Terrain colors match the legend
3. Buildings display with correct shapes and colors
4. Interactive features (hover, click) work
5. No console errors or warnings

Once this works, you're ready to integrate with the full game!
