import React from 'react';
import { type GameState, type FactionType } from '../types/game.types';
import { FACTION_COLORS } from '../utils/colors';
import { FACTIONS } from '../data/factions';

interface EndGameScoringProps {
    gameState: GameState;
}

export const EndGameScoring: React.FC<EndGameScoringProps> = ({ gameState }) => {
    if (!gameState.finalScoring) return null;

    const scores = Object.values(gameState.finalScoring).sort((a, b) => {
        if (b.totalVp !== a.totalVp) return b.totalVp - a.totalVp;
        return b.totalResourceValue - a.totalResourceValue;
    });

    const getFactionLabel = (playerId: string): string => {
        const player = gameState.players[playerId];
        if (!player) return "Unknown";

        // Try to get faction name using all possible formats
        const factionRaw = player.faction ?? player.Faction;
        if (!factionRaw) return "Unknown";

        // Handle direct number (faction type enum)
        if (typeof factionRaw === 'number') {
            const f = FACTIONS.find(f => f.id === factionRaw);
            return f ? f.name : "Unknown";
        }

        // Handle object with Type (from Go serialization)
        if (typeof factionRaw === 'object' && 'Type' in factionRaw) {
            const type = (factionRaw as { Type: number }).Type;
            const f = FACTIONS.find(f => f.id === (type as FactionType));
            return f ? f.name : "Unknown";
        }

        // Handle string faction name
        if (typeof factionRaw === 'string') {
            return factionRaw;
        }

        return "Unknown";
    };

    const getFactionColor = (playerId: string): string => {
        const player = gameState.players[playerId];
        if (!player) return '#ccc';

        // Try to resolve faction type for color
        let factionType = 0;
        const factionRaw = player.faction ?? player.Faction;

        if (factionRaw) {
            if (typeof factionRaw === 'number') factionType = factionRaw;
            else if (typeof factionRaw === 'object') {
                if ('Type' in factionRaw) factionType = (factionRaw as { Type: number }).Type;
                else if ('type' in factionRaw) factionType = (factionRaw as { type: number }).type;
            } else if (typeof factionRaw === 'string') {
                const f = FACTIONS.find(f => f.name === factionRaw || f.type === factionRaw);
                if (f) factionType = f.id;
            }
        }

        return FACTION_COLORS[factionType as FactionType] || '#ccc';
    };

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg shadow-xl p-6 max-w-4xl w-full max-h-[90vh] overflow-auto">
                <h2 className="text-3xl font-bold mb-6 text-center">Final Scoring</h2>

                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="bg-gray-100 border-b-2 border-gray-300">
                                <th className="p-3">Rank</th>
                                <th className="p-3">Player</th>
                                <th className="p-3 text-right">Base VP</th>
                                <th className="p-3 text-right">Area VP (Size)</th>
                                <th className="p-3 text-right">Cult VP</th>
                                <th className="p-3 text-right">Resource VP</th>
                                <th className="p-3 text-right font-bold">Total VP</th>
                            </tr>
                        </thead>
                        <tbody>
                            {scores.map((score, index) => (
                                <tr
                                    key={score.playerId}
                                    className={`border-b border-gray-200 hover:bg-gray-50 ${index === 0 ? 'bg-yellow-50' : ''}`}
                                >
                                    <td className="p-3 font-bold text-gray-500">#{index + 1}</td>
                                    <td className="p-3 font-medium flex items-center gap-2">
                                        <div
                                            className="w-4 h-4 rounded-full border border-gray-300"
                                            style={{ backgroundColor: getFactionColor(score.playerId) }}
                                        />
                                        {score.playerName} ({getFactionLabel(score.playerId)})
                                        {index === 0 && <span className="text-yellow-500 ml-2">👑</span>}
                                    </td>
                                    <td className="p-3 text-right">{score.baseVp}</td>
                                    <td className="p-3 text-right">
                                        {score.areaVp} <span className="text-gray-400 text-sm">({score.largestAreaSize})</span>
                                    </td>
                                    <td className="p-3 text-right">{score.cultVp}</td>
                                    <td className="p-3 text-right">
                                        {score.resourceVp} <span className="text-gray-400 text-sm">(Val: {score.totalResourceValue})</span>
                                    </td>
                                    <td className="p-3 text-right font-bold text-lg">{score.totalVp}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>

                <div className="mt-6 text-center">
                    <button
                        className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-gray-800 font-medium transition-colors"
                        onClick={() => { window.location.reload(); }}
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>
    );
};
