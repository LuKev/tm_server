import { useParams } from 'react-router-dom'
import { useMemo, useEffect, useState, useCallback } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { ScoringTiles } from './GameBoard/ScoringTiles'
import { TownTiles } from './GameBoard/TownTiles'
import { FavorTiles } from './GameBoard/FavorTiles'
import { PassingTiles } from './GameBoard/PassingTiles'
import { PlayerBoards } from './GameBoard/PlayerBoards'
import { CultTracks } from './CultTracks/CultTracks'
import { useGameStore } from '../stores/gameStore'
import { CultType, GamePhase, type GameState } from '../types/game.types'
import { getCultPositions } from '../utils/gameUtils'
import { Responsive, WidthProvider, type Layouts } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'
import { ReplayControls } from './ReplayControls'
import { ReplayLog, type LogLocation } from './ReplayLog'
import { MissingInfoModal, type MissingInfo } from './MissingInfoModal'
import { EndGameScoring } from './EndGameScoring'

const ResponsiveGridLayout = WidthProvider(Responsive)

// Bonus Card Mapping
const BONUS_CARD_MAPPING: Record<number, string> = {
    0: "BON-P (Priest)",
    1: "BON-SHIP (Shipping)",
    2: "BON-DW (Dwelling VP)",
    3: "BON-WP (Worker Power)",
    4: "BON-SPD (Spade)",
    5: "BON-TP (Trading House VP)",
    6: "BON-6C (6 Coins)",
    7: "BON-4C (Cult Advance)",
    8: "BON-BB (Stronghold/Sanctuary VP)",
    9: "BON-SHIP-VP (Shipping VP)"
};

export const Replay = () => {
    const { gameId } = useParams()
    const setGameState = useGameStore((state) => state.setGameState)
    const gameState = useGameStore((state) => state.gameState)

    // Replay specific state
    const [currentIndex, setCurrentIndex] = useState(0)
    const [totalActions, setTotalActions] = useState(0)
    const [logStrings, setLogStrings] = useState<string[]>([])
    const [logLocations, setLogLocations] = useState<LogLocation[]>([])
    const [isAutoPlaying, setIsAutoPlaying] = useState(false)
    const [autoPlaySpeed] = useState(200) // ms
    const [missingInfo, setMissingInfo] = useState<MissingInfo | null>(null)
    const [showMissingInfoModal, setShowMissingInfoModal] = useState(false)
    const [players, setPlayers] = useState<string[]>([])


    const availableBonusCardIds = useMemo(() => {
        if (!gameState?.bonusCards) return [];
        // Extract available and taken cards
        const available = Object.keys(gameState.bonusCards.available || {}).map(Number);
        const taken = Object.values(gameState.bonusCards.playerCards || {}).map(Number);
        // Combine and deduplicate
        const allIds = Array.from(new Set([...available, ...taken]));

        // Sort by ID (which corresponds to the original BON number order 0-9)
        allIds.sort((a, b) => a - b);

        // Map to strings
        return allIds.map(id => BONUS_CARD_MAPPING[id]).filter(s => s);
    }, [gameState]);

    const numCards = gameState?.bonusCards ? availableBonusCardIds.length : 9

    const availableTownTiles = useMemo(() => {
        if (!gameState?.townTiles?.available) return [];
        const tiles: number[] = [];
        Object.entries(gameState.townTiles.available).forEach(([id, count]) => {
            for (let i = 0; i < count; i++) {
                tiles.push(Number(id));
            }
        });
        return tiles.sort((a, b) => a - b);
    }, [gameState?.townTiles]);

    // Default layout configuration (same as Game.tsx but with ReplayLog)
    const defaultLayouts = useMemo(() => ({
        lg: [
            { i: 'controls', x: 0, y: 0, w: 24, h: 2, static: true },
            { i: 'log', x: 0, y: 2, w: 6, h: 10, minW: 3, minH: 6 },
            { i: 'scoring', x: 6, y: 2, w: 4, h: 8, minW: 4, minH: 6 },
            { i: 'board', x: 10, y: 2, w: 10, h: 12, minW: 10, minH: 8 },
            { i: 'cult', x: 20, y: 2, w: 4, h: 9, minW: 4, minH: 6 },
            { i: 'towns', x: 4, y: 10, w: 4, h: 3, minW: 4, minH: 2 },
            { i: 'favor', x: 20, y: 11, w: 4, h: 4, minW: 4, minH: 2 },
            { i: 'playerBoards', x: 0, y: 16, w: 20, h: 6, minW: 8, minH: 4 },
            { i: 'passing', x: 24 - numCards, y: 24, w: numCards, h: 4, minW: 4, minH: 2 }
        ],
        md: [
            { i: 'controls', x: 0, y: 0, w: 20, h: 2, static: true },
            { i: 'log', x: 0, y: 2, w: 6, h: 14, minW: 3, minH: 6 },
            { i: 'scoring', x: 6, y: 2, w: 4, h: 8, minW: 4, minH: 6 },
            { i: 'board', x: 10, y: 2, w: 10, h: 8, minW: 6, minH: 6 },
            { i: 'cult', x: 0, y: 16, w: 4, h: 9, minW: 4, minH: 6 }, // Moved down
            { i: 'towns', x: 4, y: 10, w: 4, h: 3, minW: 4, minH: 2 },
            { i: 'favor', x: 16, y: 11, w: 4, h: 4, minW: 4, minH: 2 },
            { i: 'playerBoards', x: 0, y: 14, w: 16, h: 6, minW: 8, minH: 4 },
            { i: 'passing', x: 20 - numCards, y: 20, w: numCards, h: 4, minW: 4, minH: 2 }
        ]
    }), [numCards])

    const [layouts, setLayouts] = useState<Layouts>(defaultLayouts)
    const [rowHeight, setRowHeight] = useState(60)

    // Update layout when player count changes to ensure correct default size for player boards
    useEffect(() => {
        const playerCount = Object.keys(gameState?.players || {}).length
        if (playerCount === 0) return

        setLayouts((currentLayouts) => {
            const newLayouts = { ...currentLayouts }
            let hasChanges = false

            Object.keys(newLayouts).forEach((key) => {
                newLayouts[key] = newLayouts[key].map((item) => {
                    if (item.i === 'playerBoards') {
                        // 6 units per player seems to be the baseline for 1 player
                        const newH = playerCount * 6
                        // Ensure minimum height
                        const finalH = Math.max(newH, item.minH || 4)

                        if (item.h !== finalH) {
                            hasChanges = true
                            return { ...item, h: finalH }
                        }
                    }
                    return item
                })
            })

            return hasChanges ? newLayouts : currentLayouts
        })
    }, [gameState?.players])

    // API Calls
    const startReplay = useCallback(async (restart = false) => {
        if (!gameId) return
        try {
            const res = await fetch('/api/replay/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ gameId, restart })
            })
            interface ReplayStartResponse {
                currentIndex: number;
                totalActions: number;
                logStrings: string[];
                logLocations: LogLocation[];
                players: string[];
                missingInfo?: MissingInfo;
            }

            const data = await res.json() as ReplayStartResponse

            if (data.missingInfo) {
                setMissingInfo(data.missingInfo)
                setShowMissingInfoModal(true)
                return
            }

            setCurrentIndex(data.currentIndex)
            setTotalActions(data.totalActions)
            setLogStrings(data.logStrings || [])
            setLogLocations(data.logLocations || [])
            setPlayers(data.players || [])

            // Fetch initial state
            const stateRes = await fetch(`/api/replay/state?gameId=${gameId}`)
            const stateData = await stateRes.json() as GameState
            setGameState(stateData)
        } catch (err) {
            console.error("Failed to start replay:", err)
        }
    }, [gameId, setGameState])

    const handleProvideInfo = useCallback(async (info: unknown) => {
        if (!gameId) return
        try {
            const res = await fetch('/api/replay/provide_info', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ gameId, info })
            })
            if (!res.ok) {
                console.error("Failed to provide info")
                return
            }
            setShowMissingInfoModal(false)
            setMissingInfo(null)
            // Restart replay with new info
            void startReplay()
        } catch (err) {
            console.error("Failed to provide info:", err)
        }
    }, [gameId, startReplay])



    const nextMove = useCallback(async () => {
        if (!gameId) return
        try {
            const res = await fetch('/api/replay/next', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ gameId })
            })
            if (!res.ok) {
                const errText = await res.text()
                // Check if it's a missing info error
                // The backend returns a plain string error usually, but we might want to parse it if it's JSON
                // Or if it's a 500 with "missing info: ..."
                if (errText.includes("missing info:")) {
                    console.error("Missing info detected:", errText);

                    const isInitial = errText.includes("initial_bonus_card");
                    const isPass = errText.includes("pass_bonus_card");

                    // Extract players
                    const playersMatch = /\[(.*?)\]/.exec(errText);
                    // Split by comma if present, otherwise space (backward compatibility)
                    const playersStr = playersMatch ? playersMatch[1] : "";
                    const missingPlayers = playersStr.includes(",")
                        ? playersStr.split(",").map(s => s.trim())
                        : playersStr.split(" ").map(s => s.trim()).filter(s => s);

                    const newMissingInfo: MissingInfo = {
                        GlobalBonusCards: false,
                        GlobalScoringTiles: false,
                        BonusCardSelections: {},
                        PlayerFactions: {}
                    };

                    if (isInitial) {
                        newMissingInfo.BonusCardSelections = { 0: {} };
                        missingPlayers.forEach((p: string) => {
                            if (newMissingInfo.BonusCardSelections) {
                                newMissingInfo.BonusCardSelections[0][p] = true;
                            }
                        });
                    } else if (isPass) {
                        // Use round 1 as placeholder for "Current Round"
                        newMissingInfo.BonusCardSelections = { 1: {} };
                        missingPlayers.forEach((p: string) => {
                            if (newMissingInfo.BonusCardSelections) {
                                newMissingInfo.BonusCardSelections[1][p] = true;
                            }
                        });
                    }

                    setMissingInfo(newMissingInfo);
                    setShowMissingInfoModal(true);
                    setIsAutoPlaying(false);
                    return;
                }

                console.error("Next move failed:", errText)
                setIsAutoPlaying(false)
                return
            }
            const stateData = await res.json() as GameState
            setGameState(stateData)
            setCurrentIndex(prev => prev + 1)
        } catch (err) {
            console.error("Failed to fetch next move:", err)
            setIsAutoPlaying(false)
        }
    }, [gameId, setGameState])

    // Auto-play effect
    useEffect(() => {
        let interval: NodeJS.Timeout
        if (isAutoPlaying && currentIndex < totalActions) {
            interval = setInterval(nextMove, autoPlaySpeed)
        } else if (currentIndex >= totalActions) {
            setIsAutoPlaying(false)
        }
        return () => { clearInterval(interval); }
    }, [isAutoPlaying, currentIndex, totalActions, nextMove, autoPlaySpeed])

    // Initial load
    useEffect(() => {
        void startReplay()
    }, [startReplay])

    // Helper to get cult positions (reused from Game.tsx)
    // Removed local implementation in favor of shared utility

    const handleWidthChange = (containerWidth: number, margin: [number, number], cols: number, containerPadding: [number, number]) => {
        const safeMargin = margin || [10, 10]
        const safePadding = containerPadding || [10, 10]
        const totalMargin = safeMargin[0] * (cols - 1)
        const totalPadding = safePadding[0] * 2
        const colWidth = (containerWidth - totalMargin - totalPadding) / cols
        setRowHeight(colWidth)
    }

    const [isLayoutLocked, setIsLayoutLocked] = useState(false)

    const handleLayoutChange = (_currentLayout: ReactGridLayout.Layout[], allLayouts: Layouts) => {
        // Enforce aspect ratios based on width (similar to Game.tsx)
        const updatedLayouts = { ...allLayouts }
        let hasChanges = false

        Object.keys(updatedLayouts).forEach(key => {
            const layout = updatedLayouts[key]
            const newLayout = layout.map(item => {
                let newH = item.h
                if (item.i === 'scoring') {
                    newH = item.w * 2
                } else if (item.i === 'cult') {
                    newH = Math.ceil(item.w * 2.25)
                } else if (item.i === 'board') {
                    newH = Math.ceil(item.w * 0.83)
                } else if (item.i === 'towns') {
                    newH = Math.ceil(item.w * 2 / 3)
                } else if (item.i === 'favor') {
                    newH = Math.ceil(item.w * 0.625)
                } else if (item.i === 'passing') {
                    newH = Math.ceil(item.w * (4 / numCards))
                } else if (item.i === 'playerBoards') {
                    const playerCount = Object.keys(gameState?.players || {}).length || 1
                    // Use the same logic as the useEffect for consistency, or keep it responsive
                    // In useEffect we used: playerCount * 6
                    // Here we can enforce it or let the user resize. 
                    // If we enforce it here, resizing might fight back.
                    // Let's only enforce if it's drastically off or just rely on initial set.
                    // Actually, Game.tsx enforces it. Let's stick to the useEffect logic for initial sizing
                    // and allow resizing, but maybe enforce min height?
                    // For now, let's skip strict enforcement on resize for playerBoards to allow user freedom,
                    // or enforce the same ratio.
                    // Let's enforce the ratio used in Game.tsx: newH = Math.ceil(playerCount * item.w * 0.3)
                    // But wait, the useEffect used a fixed height based on player count (playerCount * 6).
                    // Game.tsx uses: Math.ceil(playerCount * item.w * 0.3)
                    // Let's use the Game.tsx logic for consistency if we want it to scale with width.
                    newH = Math.ceil(playerCount * item.w * 0.3)
                }

                if (newH !== item.h) {
                    hasChanges = true
                    return { ...item, h: newH }
                }
                return item
            })
            updatedLayouts[key] = newLayout
        })

        if (hasChanges) {
            setLayouts(updatedLayouts)
        } else {
            setLayouts(allLayouts)
        }
    }

    return (
        <div className="min-h-screen p-4 bg-gray-100">
            {gameState?.phase === GamePhase.End && (
                <EndGameScoring gameState={gameState} />
            )}
            <div className="max-w-[1800px] mx-auto">

                <div className="flex justify-between items-center mb-4">
                    <ReplayControls
                        onStart={startReplay}
                        onNext={nextMove}
                        onToggleAutoPlay={() => { setIsAutoPlaying(!isAutoPlaying); }}
                        isAutoPlaying={isAutoPlaying}
                        currentIndex={currentIndex}
                        totalActions={totalActions}
                        gameId={gameId || ''}
                    />
                    <div className="flex gap-2">
                        <button
                            onClick={() => { setIsLayoutLocked(!isLayoutLocked) }}
                            className={`px-4 py-2 rounded text-sm font-medium transition-colors ${isLayoutLocked
                                ? 'bg-blue-100 text-blue-700 hover:bg-blue-200'
                                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                                }`}
                        >
                            {isLayoutLocked ? 'Unlock Layout' : 'Lock Layout'}
                        </button>
                        <button
                            onClick={() => { setLayouts(defaultLayouts); }}
                            className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-sm font-medium text-gray-700 transition-colors"
                        >
                            Reset Layout
                        </button>
                    </div>
                </div>

                <MissingInfoModal
                    isOpen={showMissingInfoModal}
                    missingInfo={missingInfo}
                    players={players}
                    availableBonusCards={availableBonusCardIds}
                    onSubmit={handleProvideInfo}
                    onClose={() => { setShowMissingInfoModal(false); }}
                />

                <ResponsiveGridLayout
                    className={`layout ${isLayoutLocked ? 'layout-locked' : ''}`}
                    layouts={layouts}
                    breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
                    cols={{ lg: 24, md: 20, sm: 12, xs: 8, xxs: 4 }}
                    rowHeight={rowHeight}
                    onWidthChange={handleWidthChange}
                    onLayoutChange={handleLayoutChange}
                    isDraggable={!isLayoutLocked}
                    isResizable={!isLayoutLocked}
                    resizeHandles={['e']}
                    draggableHandle=".drag-handle"
                >
                    {/* Log Viewer */}
                    <div key="log" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <ReplayLog
                            logStrings={logStrings}
                            logLocations={logLocations}
                            currentIndex={currentIndex}
                            currentRound={gameState?.round?.round || 0}
                        />
                    </div>

                    {/* Scoring Tiles */}
                    <div key="scoring" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto">
                            <ScoringTiles
                                tiles={gameState?.scoringTiles?.tiles?.map(t => t.type) || []}
                                currentRound={gameState?.round?.round || 1}
                            />
                        </div>
                    </div>

                    {/* Main game board */}
                    <div key="board" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto p-4 flex items-center justify-center bg-gray-50">
                            <GameBoard onHexClick={undefined} />
                        </div>
                    </div>

                    {/* Cult Tracks sidebar */}
                    <div key="cult" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto p-2">
                            <CultTracks
                                cultPositions={
                                    gameState?.phase === GamePhase.FactionSelection
                                        ? new Map([
                                            [CultType.Fire, []],
                                            [CultType.Water, []],
                                            [CultType.Earth, []],
                                            [CultType.Air, []],
                                        ])
                                        : getCultPositions(gameState)
                                }
                                priestsOnTrack={gameState?.cultTracks?.priestsOnTrack}
                                players={gameState?.players as any}
                            />
                        </div>
                    </div>

                    {/* Town Tiles */}
                    <div key="towns" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto">
                            <TownTiles availableTiles={availableTownTiles} />
                        </div>
                    </div>

                    {/* Favor Tiles */}
                    <div key="favor" className="bg-white rounded-lg shadow-md overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto" style={{ flex: 1 }}>
                            <FavorTiles />
                        </div>
                    </div>

                    {/* Player Boards */}
                    <div key="playerBoards" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-hidden">
                            <PlayerBoards />
                        </div>
                    </div>

                    {/* Passing Tiles (Bonus Cards) */}
                    <div key="passing" className="bg-white rounded-lg shadow-md overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto" style={{ flex: 1 }}>
                            <PassingTiles />
                        </div>
                    </div>
                </ResponsiveGridLayout>
            </div>
        </div>
    )
}
