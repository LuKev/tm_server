import { CultType, FactionType, type GameState } from '../types/game.types';
import { FACTIONS } from '../data/factions';

export interface CultPosition {
    faction: FactionType;
    position: number;
    hasKey: boolean;
}

/**
 * Resolves the faction type from a player's faction data.
 * Handles various formats: number, object with Type/type, or string name.
 */
export const resolveFaction = (factionRaw: unknown): FactionType => {
    if (factionRaw === undefined || factionRaw === null) {
        return FactionType.Unknown;
    }

    // Handle number (enum value)
    if (typeof factionRaw === 'number') {
        return factionRaw as FactionType;
    }

    // Handle object with Type (capitalized) - common from Go JSON
    if (typeof factionRaw === 'object' && 'Type' in factionRaw) {
        return (factionRaw as { Type: number }).Type as FactionType;
    }

    // Handle object with type (lowercase)
    if (typeof factionRaw === 'object' && 'type' in factionRaw) {
        return (factionRaw as { type: number }).type as FactionType;
    }

    // Handle string (e.g. "Nomads")
    if (typeof factionRaw === 'string') {
        // Try to find in FACTIONS data
        const found = FACTIONS.find(f => f.name.toLowerCase() === factionRaw.toLowerCase());
        if (found) {
            return found.id;
        }
        // Or try to match enum keys directly
        const enumKey = Object.keys(FactionType).find(key => key.toLowerCase() === factionRaw.toLowerCase());
        if (enumKey) {
            return FactionType[enumKey as keyof typeof FactionType];
        }
    }

    return FactionType.Unknown;
};

/**
 * Calculates the positions of all players on the cult tracks.
 */
export const getCultPositions = (gameState: GameState | null): Map<CultType, CultPosition[]> => {
    const positions = new Map<CultType, CultPosition[]>();
    positions.set(CultType.Fire, []);
    positions.set(CultType.Water, []);
    positions.set(CultType.Earth, []);
    positions.set(CultType.Air, []);

    const playerIds = (gameState?.turnOrder && gameState.turnOrder.length > 0)
        ? gameState.turnOrder
        : (gameState?.players ? Object.keys(gameState.players) : []);

    if (playerIds.length === 0) {
        return positions;
    }

    playerIds.forEach((playerId: string) => {
        if (!gameState) return;
        const player = gameState.players[playerId];
        Object.entries(player.cults).forEach(([cultKey, position]) => {
            const cult = Number(cultKey) as CultType;
            // Resolve faction ID
            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
            const factionRaw = player.faction ?? player.Faction;
            const factionId = resolveFaction(factionRaw);

            if (factionId !== FactionType.Unknown) {
                positions.get(cult)?.push({
                    faction: factionId,
                    position: position,
                    hasKey: false, // TODO: Track power keys from game state
                });
            }
        });
    });

    return positions;
};
