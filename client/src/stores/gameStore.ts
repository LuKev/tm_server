// Game state store using Zustand
import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import type { GameState, PlayerState } from '../types/game.types';

interface GameStore {
  // State
  gameState: GameState | null;
  localPlayerId: string | null;

  // Computed getters
  getCurrentPlayer: () => PlayerState | null;
  isMyTurn: () => boolean;

  // Actions
  setGameState: (state: GameState) => void;
  setLocalPlayerId: (id: string) => void;
  updateGameState: (updater: (draft: GameState) => void) => void;
  reset: () => void;
}

import { persist } from 'zustand/middleware';

export const useGameStore = create<GameStore>()(
  persist(
    immer((set, get) => ({
      // Initial state
      gameState: null,
      localPlayerId: null,

      // Computed getters
      getCurrentPlayer: () => {
        const { gameState, localPlayerId } = get();
        if (!gameState || !localPlayerId) return null;
        return gameState.players[localPlayerId] || null;
      },

      isMyTurn: () => {
        const { gameState, localPlayerId } = get();
        if (!gameState || !localPlayerId) return false;
        const currentPlayerId = gameState.order[gameState.currentTurn];
        return currentPlayerId === localPlayerId;
      },

      // Actions
      setGameState: (gameState) => {
        set({ gameState });
      },

      setLocalPlayerId: (localPlayerId) => {
        set({ localPlayerId });
      },

      updateGameState: (updater) => {
        set((state) => {
          if (state.gameState) {
            updater(state.gameState);
          }
        });
      },

      reset: () => {
        set({ gameState: null, localPlayerId: null });
      },
    })),
    {
      name: 'tm-game-storage', // unique name
      partialize: (state) => ({ localPlayerId: state.localPlayerId }), // only persist localPlayerId
    }
  )
);
