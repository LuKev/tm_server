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
import { Responsive, WidthProvider } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'
import { ReplayControls } from './ReplayControls'
import { ReplayLog, type LogLocation } from './ReplayLog'
import { MissingInfoModal, type MissingInfo } from './MissingInfoModal'
import { EndGameScoring } from './EndGameScoring'
import { BONUS_CARD_MAPPING } from '../data/constants'
import { useGameLayout } from '../hooks/useGameLayout'

const ResponsiveGridLayout = WidthProvider(Responsive)

export const Replay = (): React.ReactElement => {
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
        const available = Object.keys(gameState.bonusCards.available).map(Number);
        const taken = Object.values(gameState.bonusCards.playerCards).map(Number);
        // Combine and deduplicate
        const allIds = Array.from(new Set([...available, ...taken]));

        // Sort by ID (which corresponds to the original BON number order 0-9)
        allIds.sort((a, b) => a - b);

        // Map to strings
        return allIds.map(id => BONUS_CARD_MAPPING[id]).filter(s => s);
    }, [gameState]);

    const numCards = (gameState?.bonusCards && availableBonusCardIds.length > 0) ? availableBonusCardIds.length : 9

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

    const {
        layouts,
        rowHeight,
        handleWidthChange,
        handleLayoutChange,
        isLayoutLocked,
        setIsLayoutLocked,
        resetLayout
    } = useGameLayout(gameState, numCards, 'replay');

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
            setLogStrings(data.logStrings)
            setLogLocations(data.logLocations)
            setPlayers(data.players)

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
                        const selections: Record<string, boolean> = {};
                        newMissingInfo.BonusCardSelections = { 0: selections };
                        missingPlayers.forEach((p: string) => {
                            selections[p] = true;
                        });
                    } else if (isPass) {
                        // Use round 1 as placeholder for "Current Round"
                        const selections: Record<string, boolean> = {};
                        newMissingInfo.BonusCardSelections = { 1: selections };
                        missingPlayers.forEach((p: string) => {
                            selections[p] = true;
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

    const jumpTo = useCallback(async (index: number) => {
        if (!gameId) return
        try {
            const res = await fetch('/api/replay/jump', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ gameId, index })
            })
            if (!res.ok) {
                console.error("Failed to jump")
                return
            }
            const stateData = await res.json() as GameState
            setGameState(stateData)
            setCurrentIndex(index)
            setIsAutoPlaying(false) // Stop auto-play on jump
        } catch (err) {
            console.error("Failed to jump:", err)
        }
    }, [gameId, setGameState])

    const prevMove = useCallback(async () => {
        if (!gameId || currentIndex <= 0) return
        await jumpTo(currentIndex - 1)
    }, [gameId, currentIndex, jumpTo])

    // Initial load
    useEffect(() => {
        void startReplay()
    }, [startReplay])

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
                        onPrev={prevMove}
                        onToggleAutoPlay={() => { setIsAutoPlaying(!isAutoPlaying); }}
                        isAutoPlaying={isAutoPlaying}
                        currentIndex={currentIndex}
                        totalActions={totalActions}
                        gameId={gameId ?? ''}
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
                            onClick={resetLayout}
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
                            currentRound={gameState?.round.round ?? 0}
                            onLogClick={jumpTo}
                        />
                    </div>

                    {/* Scoring Tiles */}
                    <div key="scoring" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto">
                            <ScoringTiles
                                tiles={
                                    Array.isArray(gameState?.scoringTiles)
                                        ? (gameState.scoringTiles as unknown as number[])
                                        : (gameState?.scoringTiles?.tiles.map(t => t.type) ?? [])
                                }
                                currentRound={gameState?.round.round ?? 1}
                            />
                        </div>
                    </div>

                    {/* Main game board */}
                    <div key="board" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto p-4 flex items-center justify-center bg-gray-50">
                            <GameBoard onHexClick={undefined} isReplayMode={true} />
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
                                players={gameState?.players}
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
                            <PlayerBoards isReplayMode={true} />
                        </div>
                    </div>

                    {/* Passing Tiles (Bonus Cards) */}
                    <div key="passing" className="bg-white rounded-lg shadow-md overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
                        <div className="drag-handle">
                            <div className="drag-handle-pill" />
                        </div>
                        <div className="flex-1 overflow-auto" style={{ flex: 1 }}>
                            <PassingTiles
                                availableCards={
                                    gameState?.bonusCards
                                        ? Array.from(new Set([
                                            ...Object.keys(gameState.bonusCards.available).map(Number),
                                            ...Object.values(gameState.bonusCards.playerCards).map(Number)
                                        ])).sort((a, b) => a - b)
                                        : []
                                }
                                bonusCardCoins={gameState?.bonusCards?.available}
                                bonusCardOwners={
                                    gameState?.bonusCards?.playerCards
                                        ? Object.entries(gameState.bonusCards.playerCards).reduce<Record<string, string>>((acc, [pid, card]) => {
                                            acc[String(card)] = pid;
                                            return acc;
                                        }, {})
                                        : {}
                                }
                                players={gameState?.players}
                                passedPlayers={new Set(gameState?.passOrder ?? [])}
                            />
                        </div>
                    </div>
                </ResponsiveGridLayout>
            </div>
        </div>
    )
}
