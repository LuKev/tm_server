import { useState, useEffect, useMemo, useCallback } from 'react';
import { type Layouts, type Layout } from 'react-grid-layout';
import { type GameState } from '../types/game.types';

type LayoutMode = 'game' | 'replay';

export const useGameLayout = (
    gameState: GameState | null,
    numCards: number,
    mode: LayoutMode
): {
    layouts: Layouts;
    rowHeight: number;
    handleWidthChange: (containerWidth: number, margin: [number, number], cols: number, containerPadding: [number, number]) => void;
    handleLayoutChange: (currentLayout: Layout[], allLayouts: Layouts) => void;
    isLayoutLocked: boolean;
    setIsLayoutLocked: React.Dispatch<React.SetStateAction<boolean>>;
    resetLayout: () => void;
} => {
    const defaultLayouts = useMemo(() => {
        if (mode === 'replay') {
            return {
                lg: [
                    { i: 'controls', x: 0, y: 0, w: 24, h: 2, static: true },
                    { i: 'log', x: 0, y: 2, w: 6, h: 10, minW: 3, minH: 6 },
                    { i: 'scoring', x: 6, y: 2, w: 4, h: 8, minW: 4, minH: 6 },
                    { i: 'board', x: 10, y: 2, w: 10, h: 12, minW: 10, minH: 8 },
                    { i: 'cult', x: 20, y: 2, w: 4, h: 9, minW: 4, minH: 6 },
                    { i: 'towns', x: 20, y: 19, w: 4, h: 3, minW: 4, minH: 2 },
                    { i: 'favor', x: 20, y: 15, w: 4, h: 4, minW: 4, minH: 2 },
                    { i: 'playerBoards', x: 0, y: 16, w: 20, h: 6, minW: 8, minH: 4 },
                    { i: 'passing', x: 24 - numCards, y: 11, w: numCards, h: 4, minW: 4, minH: 2 }
                ],
                md: [
                    { i: 'controls', x: 0, y: 0, w: 20, h: 2, static: true },
                    { i: 'log', x: 0, y: 2, w: 6, h: 14, minW: 3, minH: 6 },
                    { i: 'scoring', x: 6, y: 2, w: 4, h: 8, minW: 4, minH: 6 },
                    { i: 'board', x: 10, y: 2, w: 10, h: 8, minW: 6, minH: 6 },
                    { i: 'cult', x: 0, y: 16, w: 4, h: 9, minW: 4, minH: 6 },
                    { i: 'towns', x: 16, y: 19, w: 4, h: 3, minW: 4, minH: 2 },
                    { i: 'favor', x: 16, y: 15, w: 4, h: 4, minW: 4, minH: 2 },
                    { i: 'playerBoards', x: 0, y: 14, w: 16, h: 6, minW: 8, minH: 4 },
                    { i: 'passing', x: 20 - numCards, y: 11, w: numCards, h: 4, minW: 4, minH: 2 }
                ]
            };
        }

        // Game mode
        return {
            lg: [
                { i: 'scoring', x: 0, y: 0, w: 4, h: 8, minW: 4, minH: 6 },
                { i: 'board', x: 4, y: 0, w: 16, h: 16, minW: 12, minH: 10 },
                { i: 'cult', x: 20, y: 0, w: 4, h: 9, minW: 4, minH: 6 },
                { i: 'towns', x: 0, y: 8, w: 4, h: 3, minW: 4, minH: 2 },
                { i: 'favor', x: 20, y: 9, w: 4, h: 4, minW: 4, minH: 2 },
                { i: 'playerBoards', x: 0, y: 16, w: 20, h: 6, minW: 8, minH: 4 },
                { i: 'passing', x: 24 - numCards, y: 13, w: numCards, h: 4, minW: 4, minH: 2 }
            ],
            md: [
                { i: 'scoring', x: 0, y: 0, w: 4, h: 8, minW: 4, minH: 6 },
                { i: 'board', x: 4, y: 0, w: 12, h: 12, minW: 8, minH: 8 },
                { i: 'cult', x: 16, y: 0, w: 4, h: 9, minW: 4, minH: 6 },
                { i: 'towns', x: 0, y: 8, w: 4, h: 3, minW: 4, minH: 2 },
                { i: 'favor', x: 16, y: 9, w: 4, h: 4, minW: 4, minH: 2 },
                { i: 'playerBoards', x: 0, y: 12, w: 16, h: 6, minW: 8, minH: 4 },
                { i: 'passing', x: 20 - numCards, y: 13, w: numCards, h: 4, minW: 4, minH: 2 }
            ]
        };
    }, [mode, numCards]);

    const [layouts, setLayouts] = useState<Layouts>(defaultLayouts);
    const [isLayoutLocked, setIsLayoutLocked] = useState(false);
    const [rowHeight, setRowHeight] = useState(60);

    // Update layout when numCards changes
    useEffect(() => {
        setLayouts((currentLayouts) => {
            const newLayouts = { ...currentLayouts };
            let hasChanges = false;

            for (const key of Object.keys(newLayouts)) {
                newLayouts[key] = newLayouts[key].map((item) => {
                    if (item.i === 'passing') {
                        if (item.w !== numCards || item.h !== 4) {

                            hasChanges = true;
                            return { ...item, w: numCards, h: 4 };
                        }
                    }
                    return item;
                });
            }

            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
            return hasChanges ? newLayouts : currentLayouts;
        });
    }, [numCards]);

    // Update layout when player count changes
    useEffect(() => {
        const playerCount = Object.keys(gameState?.players ?? {}).length;
        if (playerCount === 0) return;

        setLayouts((currentLayouts) => {
            const newLayouts = { ...currentLayouts };
            let hasChanges = false;

            for (const key of Object.keys(newLayouts)) {
                newLayouts[key] = newLayouts[key].map((item) => {
                    if (item.i === 'playerBoards') {
                        // Game.tsx uses: Math.ceil(playerCount * item.w * 0.5)
                        // Replay.tsx uses: playerCount * 6 (but commented out logic for 0.3 ratio)
                        // Let's standardize on the Game.tsx logic which seems more responsive
                        const newH = Math.ceil(playerCount * item.w * 0.5);
                        const finalH = Math.max(newH, item.minH ?? 4);

                        if (item.h !== finalH) {

                            hasChanges = true;
                            return { ...item, h: finalH };
                        }
                    }
                    return item;
                });
            }

            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
            return hasChanges ? newLayouts : currentLayouts;
        });
    }, [gameState?.players]);

    const handleWidthChange = useCallback((containerWidth: number, margin: [number, number] | null | undefined, cols: number, containerPadding: [number, number] | null | undefined) => {
        const safeMargin = margin ?? [10, 10];
        const safePadding = containerPadding ?? [10, 10];
        const totalMargin = safeMargin[0] * (cols - 1);
        const totalPadding = safePadding[0] * 2;
        const colWidth = (containerWidth - totalMargin - totalPadding) / cols;
        setRowHeight(colWidth);
    }, []);

    const handleLayoutChange = useCallback((_currentLayout: Layout[], allLayouts: Layouts) => {
        const updatedLayouts = { ...allLayouts };
        let hasChanges = false;

        for (const key of Object.keys(updatedLayouts)) {
            const layout = updatedLayouts[key];
            const newLayout = layout.map(item => {
                let newH = item.h;
                if (item.i === 'scoring') {
                    newH = item.w * 2;
                } else if (item.i === 'cult') {
                    newH = Math.ceil(item.w * 2.25);
                } else if (item.i === 'board') {
                    newH = Math.ceil(item.w * 0.83);
                } else if (item.i === 'towns') {
                    newH = Math.ceil(item.w * 2 / 3);
                } else if (item.i === 'favor') {
                    newH = Math.ceil(item.w * 0.625);
                } else if (item.i === 'passing') {
                    newH = Math.ceil(item.w * (4 / numCards));
                } else if (item.i === 'playerBoards') {
                    const playerCount = Object.keys(gameState?.players ?? {}).length || 1;
                    newH = Math.ceil(playerCount * item.w * 0.37);
                }

                if (newH !== item.h) {

                    hasChanges = true;
                    return { ...item, h: newH };
                }
                return item;
            });
            updatedLayouts[key] = newLayout;
        }

        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (hasChanges) {
            setLayouts(updatedLayouts);
        } else {
            setLayouts(allLayouts);
        }
    }, [gameState?.players, numCards]);

    const resetLayout = useCallback(() => {
        setLayouts(defaultLayouts);
    }, [defaultLayouts]);

    return {
        layouts,
        rowHeight,
        handleWidthChange,
        handleLayoutChange,
        isLayoutLocked,
        setIsLayoutLocked,
        resetLayout
    };
};
