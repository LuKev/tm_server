// Test page for Cult Tracks
import React from 'react';
import { CultTracks } from './CultTracks';
import { CultType, FactionType } from '../../types/game.types';

export const CultTracksTest: React.FC = () => {
  // Create test data with various faction positions
  const testCultPositions = new Map();

  // Fire cult
  testCultPositions.set(CultType.Fire, [
    { faction: FactionType.ChaosMagicians, position: 10, hasKey: true },
    { faction: FactionType.Giants, position: 7, hasKey: false },
    { faction: FactionType.Nomads, position: 5, hasKey: false },
    { faction: FactionType.Witches, position: 3, hasKey: false },
  ]);

  // Water cult
  testCultPositions.set(CultType.Water, [
    { faction: FactionType.Mermaids, position: 10, hasKey: false },
    { faction: FactionType.Swarmlings, position: 8, hasKey: false },
    { faction: FactionType.Fakirs, position: 6, hasKey: false },
    { faction: FactionType.Darklings, position: 4, hasKey: false },
    { faction: FactionType.Halflings, position: 2, hasKey: false },
  ]);

  // Earth cult
  testCultPositions.set(CultType.Earth, [
    { faction: FactionType.Dwarves, position: 9, hasKey: false },
    { faction: FactionType.Engineers, position: 6, hasKey: false },
    { faction: FactionType.Halflings, position: 5, hasKey: false },
    { faction: FactionType.Giants, position: 3, hasKey: false },
  ]);

  // Air cult
  testCultPositions.set(CultType.Air, [
    { faction: FactionType.Witches, position: 10, hasKey: true },
    { faction: FactionType.Cultists, position: 7, hasKey: false },
    { faction: FactionType.Auren, position: 5, hasKey: false },
    { faction: FactionType.Alchemists, position: 1, hasKey: false },
  ]);

  // Test bonus tiles (optional) - priests are colored by faction
  const testBonusTiles = new Map();
  testBonusTiles.set(CultType.Fire, [
    { power: 3 },
    { power: 2 },
    { priests: 1, faction: FactionType.Giants }, // Red priest
    { power: 2 },
  ]);
  testBonusTiles.set(CultType.Water, [
    { priests: 1, faction: FactionType.Mermaids }, // Blue priest
    { power: 2 },
    { power: 2 },
    { power: 2 },
  ]);
  testBonusTiles.set(CultType.Earth, [
    { power: 3 },
    { priests: 1, faction: FactionType.Halflings }, // Brown priest
    { power: 2 },
    { power: 2 },
  ]);
  testBonusTiles.set(CultType.Air, [
    { power: 3 },
    { power: 2 },
    { power: 2 },
    { priests: 1, faction: FactionType.Witches }, // Green priest
  ]);

  return (
    <div className="min-h-screen p-8 bg-gray-100">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-4xl font-bold text-gray-800 mb-4">
          Terra Mystica - Cult Tracks Test
        </h1>

        <div className="bg-white rounded-lg shadow-md p-6">
          <CultTracks
            cultPositions={testCultPositions}
            bonusTiles={testBonusTiles}
          />
        </div>

        <div className="mt-6 bg-white rounded-lg shadow-md p-6">
          <h2 className="text-2xl font-semibold mb-4">Test Data</h2>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <h3 className="font-bold mb-2">Fire Cult</h3>
              <ul className="list-disc list-inside">
                <li>Chaos Magicians: 10 (with key)</li>
                <li>Giants: 7</li>
                <li>Nomads: 5</li>
                <li>Witches: 3</li>
              </ul>
            </div>
            <div>
              <h3 className="font-bold mb-2">Water Cult</h3>
              <ul className="list-disc list-inside">
                <li>Mermaids: 10</li>
                <li>Swarmlings: 8</li>
                <li>Fakirs: 6</li>
                <li>Darklings: 4</li>
                <li>Halflings: 2</li>
              </ul>
            </div>
            <div>
              <h3 className="font-bold mb-2">Earth Cult</h3>
              <ul className="list-disc list-inside">
                <li>Dwarves: 9</li>
                <li>Engineers: 6</li>
                <li>Halflings: 5</li>
                <li>Giants: 3</li>
              </ul>
            </div>
            <div>
              <h3 className="font-bold mb-2">Air Cult</h3>
              <ul className="list-disc list-inside">
                <li>Witches: 10 (with key)</li>
                <li>Cultists: 7</li>
                <li>Auren: 5</li>
                <li>Alchemists: 1</li>
              </ul>
            </div>
          </div>

          <div className="mt-4">
            <h3 className="font-bold mb-2">Legend</h3>
            <ul className="list-disc list-inside text-sm">
              <li>Circles: Regular cult positions</li>
              <li>Hexagons: Position 10 or factions with power keys</li>
              <li>Letters: First letter of faction name (lowercase for some factions)</li>
              <li>Bottom numbers/p: Bonus tiles (power points or priests)</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
};
