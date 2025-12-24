import React, { useEffect, useState, useMemo } from 'react';

interface ReplayLogProps {
    logStrings: string[];
    currentRound: number;
}

export const ReplayLog: React.FC<ReplayLogProps> = ({ logStrings, currentRound }) => {
    const [viewRound, setViewRound] = useState(currentRound);

    // Sync viewRound with currentRound when it changes (auto-follow)
    useEffect(() => {
        setViewRound(currentRound);
    }, [currentRound]);

    // Parse log strings into rounds
    const rounds = useMemo(() => {
        const roundMap = new Map<number, string[]>();
        let parsingRound = 0; // 0 = Setup

        // Initialize Round 0
        roundMap.set(0, []);

        logStrings.forEach(line => {
            if (line.startsWith('Round ')) {
                const match = /Round (\d+)/.exec(line);
                if (match) {
                    parsingRound = parseInt(match[1]);
                    roundMap.set(parsingRound, []);
                }
            }

            const currentLines = roundMap.get(parsingRound);
            if (currentLines) {
                currentLines.push(line);
            }
        });
        return roundMap;
    }, [logStrings]);

    const currentLines = rounds.get(viewRound) || [];

    return (
        <div className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col h-full">
            <div className="p-2 bg-gray-100 border-b flex justify-between items-center">
                <span className="font-bold text-gray-700">Log (R{viewRound})</span>
                <div className="flex gap-1">
                    <button
                        onClick={() => { setViewRound(Math.max(0, viewRound - 1)); }}
                        disabled={viewRound <= 0}
                        className="px-2 py-0.5 bg-gray-200 hover:bg-gray-300 rounded text-xs disabled:opacity-50"
                    >
                        Prev
                    </button>
                    <button
                        onClick={() => { setViewRound(Math.min(6, viewRound + 1)); }}
                        disabled={viewRound >= 6}
                        className="px-2 py-0.5 bg-gray-200 hover:bg-gray-300 rounded text-xs disabled:opacity-50"
                    >
                        Next
                    </button>
                </div>
            </div>
            <div className="flex-1 overflow-auto p-2 font-mono text-xs">
                <table className="w-full border-collapse table-fixed">
                    <colgroup>
                        <col className="w-24" />
                        <col className="w-24" />
                        <col className="w-24" />
                        <col className="w-24" />
                        <col className="w-24" />
                    </colgroup>
                    <tbody>
                        {currentLines.map((line, index) => {
                            // Check if line is a separator or header
                            if (line.startsWith('Round') || line.startsWith('TurnOrder') || line.startsWith('---')) {
                                return (
                                    <tr key={index} className="bg-gray-50">
                                        <td colSpan={5} className="px-2 py-1 font-bold text-gray-600 border-b overflow-hidden text-ellipsis whitespace-nowrap">
                                            {line}
                                        </td>
                                    </tr>
                                );
                            }

                            // Split by pipe
                            const parts = line.split('|');
                            if (parts.length > 1) {
                                return (
                                    <tr key={index} className="hover:bg-gray-50">
                                        {parts.map((part, i) => (
                                            <td key={i} className="px-2 py-1 border-r border-gray-200 last:border-r-0 whitespace-pre overflow-hidden text-ellipsis">
                                                {part}
                                            </td>
                                        ))}
                                        {/* Fill remaining cells if any */}
                                        {Array.from({ length: 5 - parts.length }).map((_, i) => (
                                            <td key={`empty-${String(i)}`} className="border-r border-gray-200 last:border-r-0"></td>
                                        ))}
                                    </tr>
                                );
                            }

                            // Fallback for other lines (e.g. settings)
                            return (
                                <tr key={index}>
                                    <td colSpan={5} className="px-2 py-1 whitespace-pre overflow-hidden text-ellipsis">
                                        {line}
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    );
};
