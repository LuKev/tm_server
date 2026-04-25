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
  [TerrainType.River]: '#b3d9ff',       // Light Blue
  [TerrainType.Ice]: '#edf8ff',         // Ice
  [TerrainType.Volcano]: '#ff8a2a',     // Volcano
};

// Faction colors (from terra-mystica codebase)
export const FACTION_COLORS: Record<FactionType, string> = {
  [FactionType.Unknown]: '#000000', // Black for unknown
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
  [FactionType.Architects]: '#e74c3c',
  [FactionType.Archivists]: '#f4d03f',
  [FactionType.Atlanteans]: '#5dade2',
  [FactionType.ChashDallah]: '#52b788',
  [FactionType.ChildrenOfTheWyrm]: '#2c2c2c',
  [FactionType.Conspirators]: '#95a5a6',
  [FactionType.Djinni]: '#f4d03f',
  [FactionType.DynionGeifr]: '#95a5a6',
  [FactionType.Goblins]: '#2c2c2c',
  [FactionType.Prospectors]: '#c8956b',
  [FactionType.TheEnlightened]: '#52b788',
  [FactionType.TimeTravelers]: '#c8956b',
  [FactionType.Treasurers]: '#e74c3c',
  [FactionType.Wisps]: '#5dade2',
  [FactionType.IceMaidens]: '#edf8ff',
  [FactionType.Yetis]: '#edf8ff',
  [FactionType.Dragonlords]: '#ff8a2a',
  [FactionType.Acolytes]: '#ff8a2a',
  [FactionType.Shapeshifters]: '#ddd4c7',
  [FactionType.Riverwalkers]: '#ddd4c7',
  [FactionType.Firewalkers]: '#ff8a2a',
  [FactionType.Selkies]: '#edf8ff',
  [FactionType.SnowShamans]: '#edf8ff',
};

// Contrast colors for text/borders
const CONTRAST_COLORS: Record<string, string> = {
  '#f4d03f': '#000',  // Yellow -> Black
  '#c8956b': '#000',  // Brown -> Black
  '#2c2c2c': '#fff',  // Black -> White
  '#5dade2': '#000',  // Blue -> Black
  '#52b788': '#000',  // Green -> Black
  '#95a5a6': '#000',  // Gray -> Black
  '#e74c3c': '#fff',  // Red -> White
  '#b3d9ff': '#000',  // Light Blue -> Black
  '#edf8ff': '#000',  // Ice -> Black
  '#ff8a2a': '#000',  // Volcano -> Black
  '#ddd4c7': '#000',  // Colorless -> Black
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
