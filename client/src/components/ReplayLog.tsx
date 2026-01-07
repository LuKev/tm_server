import React, { useEffect, useState, useMemo, useRef } from 'react';

export interface LogLocation {
    lineIndex: number;
    columnIndex: number;
}

interface ReplayLogProps {
    logStrings: string[];
    logLocations: LogLocation[];
    currentIndex: number;
    currentRound: number;
    onLogClick: (index: number) => void;
}

export const ReplayLog: React.FC<ReplayLogProps> = ({ logStrings, logLocations, currentIndex, currentRound, onLogClick }) => {
    const [viewRound, setViewRound] = useState(currentRound);
    const scrollRef = useRef<HTMLDivElement>(null);

    // Sync viewRound with currentRound when it changes (auto-follow)
    useEffect(() => {
        setViewRound(currentRound);
    }, [currentRound]);

    // Parse log strings into rounds, tracking global indices
    const rounds = useMemo(() => {
        const roundMap = new Map<number, { line: string, globalIndex: number }[]>();
        let parsingRound = 0; // 0 = Setup

        // Initialize Round 0
        roundMap.set(0, []);

        logStrings.forEach((line, globalIndex) => {
            if (line.startsWith('Round ')) {
                const match = /Round (\d+)/.exec(line);
                if (match) {
                    parsingRound = parseInt(match[1]);
                    roundMap.set(parsingRound, []);
                }
            }

            const currentLines = roundMap.get(parsingRound);
            if (currentLines) {
                currentLines.push({ line, globalIndex });
            }
        });
        return roundMap;
    }, [logStrings]);

    const currentLines = rounds.get(viewRound) ?? [];

    // Determine highlight location
    // currentIndex is the index of the *next* action.
    // We want to highlight the *last executed* action, which is currentIndex - 1.
    const highlightIndex = currentIndex - 1;
    let highlightLoc: LogLocation | null = null;
    if (highlightIndex >= 0 && highlightIndex < logLocations.length) {
        highlightLoc = logLocations[highlightIndex];
    }

    // Auto-scroll to highlighted element (optional, simple implementation)
    useEffect(() => {
        if (highlightLoc && viewRound === currentRound && scrollRef.current) {
            const highlightedElement = scrollRef.current.querySelector('[data-highlighted="true"]');
            if (highlightedElement) {
                highlightedElement.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }
    }, [highlightLoc, viewRound, currentRound]);

    const styles = {

        container: {
            backgroundColor: 'white',
            borderRadius: '0.5rem',
            boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column' as const,
            height: '100%',
        },
        header: {
            padding: '0.5rem',
            backgroundColor: '#f3f4f6', // gray-100
            borderBottom: '1px solid #e5e7eb', // gray-200
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
        },
        headerTitle: {
            fontWeight: 'bold',
            color: '#374151', // gray-700
        },
        buttonGroup: {
            display: 'flex',
            gap: '0.25rem',
        },
        button: {
            padding: '0.125rem 0.5rem',
            backgroundColor: '#e5e7eb', // gray-200
            borderRadius: '0.25rem',
            fontSize: '0.75rem',
            cursor: 'pointer',
            border: 'none',
        },
        buttonDisabled: {
            opacity: 0.5,
            cursor: 'not-allowed',
        },
        logContainer: {
            flex: 1,
            overflow: 'auto',
            padding: '0.5rem',
            fontFamily: 'monospace',
            fontSize: '0.75rem',
        },
        table: {
            width: '100%',
            borderCollapse: 'collapse' as const,
            tableLayout: 'fixed' as const,
        },
        col: {
            width: '6rem', // w-24 approx
        },
        rowHeader: {
            backgroundColor: '#f9fafb', // gray-50
        },
        cellHeader: {
            padding: '0.25rem 0.5rem',
            fontWeight: 'bold',
            color: '#4b5563', // gray-600
            borderBottom: '1px solid #e5e7eb',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap' as const,
        },
        rowHover: {
            cursor: 'pointer',
        },
        cell: {
            padding: '0.25rem 0.5rem',
            borderRight: '1px solid #e5e7eb',
            whiteSpace: 'pre' as const,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
        },
        cellLast: {
            borderRight: 'none',
        },
        highlight: {
            backgroundColor: '#fef08a', // yellow-200
            fontWeight: 'bold',
            color: 'black',
            border: '2px solid #ef4444', // red-500
            boxShadow: '0 0 0 1px #ef4444',
        },
    };

    return (
        <div style={styles.container}>
            <div style={styles.header}>
                <span style={styles.headerTitle}>Log (R{viewRound})</span>
                <div style={styles.buttonGroup}>
                    <button
                        onClick={() => { setViewRound(Math.max(0, viewRound - 1)); }}
                        disabled={viewRound <= 0}
                        style={{
                            ...styles.button,
                            ...(viewRound <= 0 ? styles.buttonDisabled : {})
                        }}
                    >
                        Prev
                    </button>
                    <button
                        onClick={() => { setViewRound(Math.min(6, viewRound + 1)); }}
                        disabled={viewRound >= 6}
                        style={{
                            ...styles.button,
                            ...(viewRound >= 6 ? styles.buttonDisabled : {})
                        }}
                    >
                        Next
                    </button>
                </div>
            </div>
            <div style={styles.logContainer} ref={scrollRef}>
                <table style={styles.table}>
                    <colgroup>
                        <col style={styles.col} />
                        <col style={styles.col} />
                        <col style={styles.col} />
                        <col style={styles.col} />
                        <col style={styles.col} />
                    </colgroup>
                    <tbody>
                        {currentLines.map(({ line, globalIndex }, index) => {
                            // Check if line is a separator or header
                            if (line.startsWith('Round') || line.startsWith('TurnOrder') || line.startsWith('---')) {
                                return (
                                    <tr key={index} style={styles.rowHeader}>
                                        <td colSpan={5} style={styles.cellHeader}>
                                            {line}
                                        </td>
                                    </tr>
                                );
                            }

                            // Check if this line contains the highlight
                            const isHighlightLine = highlightLoc?.lineIndex === globalIndex;

                            // Split by pipe
                            const parts = line.split('|');
                            if (parts.length > 1) {
                                return (
                                    <tr
                                        key={index}
                                        style={styles.rowHover}
                                        onClick={() => {
                                            // Fallback: if clicking the row (e.g. empty space), jump to the last action of the row
                                            let maxActionIndex = -1;
                                            for (let k = 0; k < logLocations.length; k++) {
                                                if (logLocations[k].lineIndex === globalIndex) {
                                                    maxActionIndex = k;
                                                } else if (logLocations[k].lineIndex > globalIndex) {
                                                    break;
                                                }
                                            }
                                            if (maxActionIndex !== -1) {
                                                onLogClick(maxActionIndex + 1);
                                            }
                                        }}
                                    >
                                        {parts.map((part, i) => {
                                            const isHighlighted = isHighlightLine && highlightLoc?.columnIndex === i;
                                            const isLast = i === parts.length - 1 && parts.length === 5; // Approximation for last cell border

                                            return (
                                                <td
                                                    key={i}
                                                    data-highlighted={isHighlighted ? "true" : undefined}
                                                    style={{
                                                        ...styles.cell,
                                                        ...(isLast ? styles.cellLast : {}),
                                                        ...(isHighlighted ? styles.highlight : {})
                                                    }}
                                                    onClick={(e) => {
                                                        e.stopPropagation(); // Prevent row click
                                                        // Find the last action that maps to this specific cell (line + col)
                                                        let maxActionIndex = -1;
                                                        for (let k = 0; k < logLocations.length; k++) {
                                                            const loc = logLocations[k];
                                                            if (loc.lineIndex === globalIndex && loc.columnIndex === i) {
                                                                maxActionIndex = k;
                                                            } else if (loc.lineIndex > globalIndex) {
                                                                break;
                                                            }
                                                        }

                                                        if (maxActionIndex !== -1) {
                                                            onLogClick(maxActionIndex + 1);
                                                        }
                                                    }}
                                                >
                                                    {part}
                                                </td>
                                            );
                                        })}
                                        {/* Fill remaining cells if any */}
                                        {Array.from({ length: 5 - parts.length }).map((_, i) => (
                                            <td
                                                key={`empty-${String(i)}`}
                                                style={{
                                                    ...styles.cell,
                                                    ...(i === (5 - parts.length - 1) ? styles.cellLast : {})
                                                }}
                                            ></td>
                                        ))}
                                    </tr>
                                );
                            }

                            // Fallback for other lines (e.g. settings)
                            return (
                                <tr key={index}>
                                    <td colSpan={5} style={{ ...styles.cell, ...styles.cellLast }}>
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
