// Test page to verify hex map rendering
import React, { useState } from 'react';
import { HexGridCanvas } from './GameBoard/HexGridCanvas';
import { CultTracks } from './CultTracks/CultTracks';
import type { CultPosition, PriestSpot } from './CultTracks/CultTracks';
import { BASE_GAME_MAP } from '../data/baseGameMap';
import { BuildingType, FactionType, CultType } from '../types/game.types';
import type { Building, Bridge } from '../types/game.types';

export const MapTest: React.FC = (): React.ReactElement => {
  const [showBuildings, setShowBuildings] = useState(false);
  const [showBridges, setShowBridges] = useState(false);
  const [hoveredHex, setHoveredHex] = useState<string | null>(null);

  // Create some test buildings
  const testBuildings = new Map<string, Building>();
  if (showBuildings) {
    // Add a dwelling on (0, 0)
    testBuildings.set('0,0', {
      ownerPlayerId: 'player1',
      faction: FactionType.Nomads,
      type: BuildingType.Dwelling,
    });

    // Add a trading house on (1, 0)
    testBuildings.set('1,0', {
      ownerPlayerId: 'player2',
      faction: FactionType.Witches,
      type: BuildingType.TradingHouse,
    });

    // Add a temple on (2, 0)
    testBuildings.set('2,0', {
      ownerPlayerId: 'player3',
      faction: FactionType.Mermaids,
      type: BuildingType.Temple,
    });

    // Add a stronghold on (3, 0)
    testBuildings.set('3,0', {
      ownerPlayerId: 'player4',
      faction: FactionType.Engineers,
      type: BuildingType.Stronghold,
    });

    // Add a sanctuary on (4, 0)
    testBuildings.set('4,0', {
      ownerPlayerId: 'player5',
      faction: FactionType.Alchemists,
      type: BuildingType.Sanctuary,
    });
  }

  // Create some test bridges
  // Bridges connect hexes at distance 2 (not adjacent) across river edges
  const testBridges: Bridge[] = [];
  if (showBridges) {
    // Yellow bridge: (0,1) to (1,2) across river edge
    testBridges.push({
      ownerPlayerId: 'player1',
      faction: FactionType.Nomads,
      fromCoord: { q: 0, r: 1 },
      toCoord: { q: 1, r: 2 },
    });

    // Blue bridge: (4,1) to (5,2) across river hexes
    testBridges.push({
      ownerPlayerId: 'player3',
      faction: FactionType.Mermaids,
      fromCoord: { q: 4, r: 1 },
      toCoord: { q: 5, r: 2 },
    });

    // Green bridge: (3,5) to (4,6) across river edge
    testBridges.push({
      ownerPlayerId: 'player2',
      faction: FactionType.Witches,
      fromCoord: { q: 3, r: 5 },
      toCoord: { q: 4, r: 6 },
    });
  }

  const handleHexClick = (q: number, r: number): void => {
    // console.log(`Clicked hex (${q}, ${r})`);
    alert(`Clicked hex (${String(q)}, ${String(r)})`);
  };

  const handleHexHover = (q: number, r: number): void => {
    setHoveredHex(`${String(q)},${String(r)}`);
  };

  const handleBonusTileClick = (cult: CultType, tileIndex: number): void => {
    const cultNames = ['Fire', 'Water', 'Earth', 'Air'];
    const cultName = cultNames[cult];
    const tiles = testBonusTiles.get(cult);
    const tile = tiles?.[tileIndex];

    if (tile?.priests && tile.faction !== undefined) {
      const factionNames: Partial<Record<FactionType, string>> = {
        [FactionType.Giants]: 'Giants',
        [FactionType.Swarmlings]: 'Swarmlings',
        [FactionType.Halflings]: 'Halflings',
        [FactionType.Dwarves]: 'Dwarves',
      };
      const factionName = factionNames[tile.faction] ?? 'Unknown';
      alert(`Clicked: ${cultName} cult - Priest tile (${factionName})`);
    } else if (tile?.power) {
      const spotName = tileIndex === 4 ? 'Return spot (1 power)' : `Power ${String(tile.power)} spot`;
      alert(`Clicked: ${cultName} cult - ${spotName}`);
    } else {
      alert(`Clicked: ${cultName} cult - Tile ${String(tileIndex + 1)}`);
    }
  };

  const highlightedHexes = new Set<string>();
  if (hoveredHex) {
    highlightedHexes.add(hoveredHex);
  }

  // Create test cult positions - 4 factions per cult with ties and position 10
  const testCultPositions = new Map<CultType, CultPosition[]>();

  // Fire: Giants at position 10 (hex), Swarmlings and Halflings tied at 6
  testCultPositions.set(CultType.Fire, [
    { faction: FactionType.Giants, position: 10, hasKey: false },
    { faction: FactionType.Swarmlings, position: 6, hasKey: false },
    { faction: FactionType.Halflings, position: 6, hasKey: false },
    { faction: FactionType.Dwarves, position: 2, hasKey: false },
  ]);

  // Water: Halflings at position 10 (hex), Giants and Dwarves tied at 4
  testCultPositions.set(CultType.Water, [
    { faction: FactionType.Halflings, position: 10, hasKey: false },
    { faction: FactionType.Swarmlings, position: 7, hasKey: false },
    { faction: FactionType.Giants, position: 4, hasKey: false },
    { faction: FactionType.Dwarves, position: 4, hasKey: false },
  ]);

  // Earth: No position 10, but Swarmlings and Dwarves tied at 5
  testCultPositions.set(CultType.Earth, [
    { faction: FactionType.Giants, position: 8, hasKey: false },
    { faction: FactionType.Swarmlings, position: 5, hasKey: false },
    { faction: FactionType.Dwarves, position: 5, hasKey: false },
    { faction: FactionType.Halflings, position: 2, hasKey: false },
  ]);

  // Air: All different positions, no ties, no position 10
  testCultPositions.set(CultType.Air, [
    { faction: FactionType.Giants, position: 9, hasKey: false },
    { faction: FactionType.Swarmlings, position: 7, hasKey: false },
    { faction: FactionType.Halflings, position: 5, hasKey: false },
    { faction: FactionType.Dwarves, position: 3, hasKey: false },
  ]);

  // Test bonus tiles with some priests (5 spots: 3, 2, 2, 2, 1 return)
  const testBonusTiles = new Map<CultType, PriestSpot[]>();
  testBonusTiles.set(CultType.Fire, [
    { priests: 1, faction: FactionType.Giants }, // Red priest on 3 spot
    { priests: 1, faction: FactionType.Swarmlings }, // Blue priest on 2 spot
    { power: 2 },
    { power: 2 },
    { power: 1 }, // Return spot
  ]);
  testBonusTiles.set(CultType.Water, [
    { power: 3 },
    { power: 2 },
    { priests: 1, faction: FactionType.Halflings }, // Brown priest
    { power: 2 },
    { power: 1 }, // Return spot
  ]);
  testBonusTiles.set(CultType.Earth, [
    { power: 3 },
    { power: 2 },
    { power: 2 },
    { priests: 1, faction: FactionType.Dwarves }, // Gray priest
    { power: 1 }, // Return spot
  ]);
  testBonusTiles.set(CultType.Air, [
    { power: 3 },
    { power: 2 },
    { power: 2 },
    { power: 2 },
    { power: 1 }, // Return spot
  ]);

  return (
    <div className="min-h-screen p-8 bg-gray-100">
      <div className="w-full max-w-[1800px] mx-auto">
        <h1 className="text-4xl font-bold text-gray-800 mb-4">
          Terra Mystica - Map Test
        </h1>

        <div className="bg-white rounded-lg shadow-md p-4 mb-4">
          <h2 className="text-xl font-semibold mb-3">Test Controls</h2>

          <div className="flex gap-4 items-center">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={showBuildings}
                onChange={(e) => { setShowBuildings(e.target.checked); }}
                className="w-4 h-4"
              />
              <span>Show test buildings</span>
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={showBridges}
                onChange={(e) => { setShowBridges(e.target.checked); }}
                className="w-4 h-4"
              />
              <span>Show test bridges</span>
            </label>
          </div>

          {hoveredHex && (
            <div className="mt-2 text-sm text-gray-600">
              Hovering: <code className="bg-gray-100 px-2 py-1 rounded">{hoveredHex}</code>
            </div>
          )}
        </div>

        <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'flex-start' }}>
          {/* Main map area */}
          <div className="bg-white rounded-lg shadow-md p-4" style={{ marginRight: '1rem' }}>
            <h2 className="text-xl font-semibold mb-3">Base Game Map (113 hexes)</h2>

            <div className="overflow-auto">
              <HexGridCanvas
                hexes={BASE_GAME_MAP}
                buildings={testBuildings}
                bridges={testBridges}
                highlightedHexes={highlightedHexes}
                onHexClick={handleHexClick}
                onHexHover={handleHexHover}
              />
            </div>
          </div>

          {/* Cult Tracks sidebar */}
          <div style={{ width: '280px', flexShrink: 0 }}>
            <div className="bg-white rounded-lg shadow-md p-4" style={{ position: 'sticky', top: '2rem' }}>
              <h2 className="text-xl font-semibold mb-3">Cult Tracks</h2>
              <CultTracks
                cultPositions={testCultPositions}
                bonusTiles={testBonusTiles}
                onBonusTileClick={handleBonusTileClick}
              />
            </div>
          </div>
        </div>

        <div className="mt-4 bg-white rounded-lg shadow-md p-4">
          <h2 className="text-xl font-semibold mb-2">Map Legend</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#f4d03f' }}></div>
              <span>Desert (Yellow)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#c8956b' }}></div>
              <span>Plains (Brown)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#2c2c2c' }}></div>
              <span>Swamp (Black)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#5dade2' }}></div>
              <span>Lake (Blue)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#52b788' }}></div>
              <span>Forest (Green)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#95a5a6' }}></div>
              <span>Mountain (Gray)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#e74c3c' }}></div>
              <span>Wasteland (Red)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded" style={{ backgroundColor: '#b3d9ff' }}></div>
              <span>River (Light Blue)</span>
            </div>
          </div>
        </div>

        <div className="mt-4 text-sm text-gray-600">
          <p><strong>Instructions:</strong></p>
          <ul className="list-disc list-inside mt-2">
            <li>Hover over hexes to highlight them</li>
            <li>Click hexes to see their coordinates</li>
            <li>Toggle "Show test buildings" to see all 5 building types</li>
            <li>In dev mode, coordinates are shown on each hex</li>
          </ul>
        </div>
      </div>
    </div>
  );
};
