// Color constants - Based on terra-mystica/stc/game.js and common.js

import { TerrainType, FactionType, CultType } from '../types/game.types';

// Terrain colors
export const TERRAIN_COLORS: Record<TerrainType, string> = {
  [TerrainType.Desert]: '#f4d03f',      // Yellow
  [TerrainType.Plains]: '#c8956b',      // Brown
  [TerrainType.Swamp]: '#444',          // Dark grey (buildings are darker black)
  [TerrainType.Lake]: '#5dade2',        // Blue
  [TerrainType.Forest]: '#52b788',      // Green
  [TerrainType.Mountain]: '#95a5a6',    // Gray
  [TerrainType.Wasteland]: '#e74c3c',   // Red
};

// Faction colors (from terra-mystica codebase)
export const FACTION_COLORS: Record<FactionType, string> = {
  [FactionType.Nomads]: '#f4d03f',      // Yellow
  [FactionType.Fakirs]: '#f4d03f',      // Yellow
  [FactionType.ChaosMagicians]: '#e74c3c', // Red
  [FactionType.Giants]: '#e74c3c',      // Red
  [FactionType.Swarmlings]: '#5dade2',  // Blue
  [FactionType.Mermaids]: '#5dade2',    // Blue
  [FactionType.Witches]: '#52b788',     // Green
  [FactionType.Auren]: '#52b788',       // Green
  [FactionType.Halflings]: '#c8956b',   // Brown
  [FactionType.Cultists]: '#c8956b',    // Brown
  [FactionType.Alchemists]: '#2c2c2c',  // Black
  [FactionType.Darklings]: '#2c2c2c',   // Black
  [FactionType.Engineers]: '#95a5a6',   // Gray
  [FactionType.Dwarves]: '#95a5a6',     // Gray
};

// Contrast colors for text/borders
export const CONTRAST_COLORS: Record<string, string> = {
  '#f4d03f': '#000',  // Yellow -> Black
  '#c8956b': '#000',  // Brown -> Black
  '#2c2c2c': '#fff',  // Black -> White
  '#5dade2': '#000',  // Blue -> Black
  '#52b788': '#000',  // Green -> Black
  '#95a5a6': '#000',  // Gray -> Black
  '#e74c3c': '#fff',  // Red -> White
};

// Cult track colors (from terra-mystica/stc/game.js)
export const CULT_COLORS: Record<CultType, string> = {
  [CultType.Fire]: '#f88',
  [CultType.Water]: '#ccf',
  [CultType.Earth]: '#b84',
  [CultType.Air]: '#f0f0f0',
};

/**
 * Get contrast color for a given color
 */
export function getContrastColor(color: string): string {
  return CONTRAST_COLORS[color] || '#000';
}
