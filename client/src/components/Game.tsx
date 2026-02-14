import { useParams } from 'react-router-dom'
import { useMemo, useEffect } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { ScoringTiles } from './GameBoard/ScoringTiles'
import { TownTiles } from './GameBoard/TownTiles'
import { FavorTiles } from './GameBoard/FavorTiles'
import { PassingTiles } from './GameBoard/PassingTiles'
import { PlayerBoards } from './GameBoard/PlayerBoards'
import { PlayerSummaryBar } from './GameBoard/PlayerSummaryBar'
import { CultTracks } from './CultTracks/CultTracks'
import { FactionSelector } from './FactionSelector'
import { FACTIONS } from '../data/factions'
import { useGameStore } from '../stores/gameStore'
import { useActionService } from '../services/actionService'
import { CultType, GamePhase, type FactionType } from '../types/game.types'
import { useWebSocket } from '../services/WebSocketContext'
import { Responsive, WidthProvider } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'
import { getCultPositions } from '../utils/gameUtils'
import { useGameLayout } from '../hooks/useGameLayout'

const ResponsiveGridLayout = WidthProvider(Responsive)

export const Game = () => {
  const { gameId } = useParams()
  const { isConnected, sendMessage } = useWebSocket()
  const gameState = useGameStore((state) => state.gameState)
  const localPlayerId = useGameStore((state) => state.localPlayerId)

  const { submitSetupDwelling, submitSelectFaction } = useActionService()

  const numCards = useMemo(() => {
    if (!gameState?.bonusCards) return 9;
    const available = Object.keys(gameState.bonusCards.available ?? {}).length;
    const taken = Object.keys(gameState.bonusCards.playerCards ?? {}).length;
    return available + taken;
  }, [gameState?.bonusCards]);

  const {
    layouts,
    rowHeight,
    handleWidthChange,
    handleLayoutChange,
    isLayoutLocked,
    setIsLayoutLocked,
    resetLayout
  } = useGameLayout(gameState, numCards, 'game');

  useEffect(() => {
    if (isConnected && gameId && !gameState) {
      sendMessage({ type: 'get_game_state', payload: { gameID: gameId } })
    }
  }, [isConnected, gameId, gameState, sendMessage])

  // Handle hex clicks
  const handleHexClick = (q: number, r: number): void => {
    if (!localPlayerId) {
      console.warn('No local player ID set')
      return
    }
    submitSetupDwelling(localPlayerId, q, r, gameId)
  }

  const handleFactionSelect = (factionType: string) => {
    if (localPlayerId && gameId) {
      submitSelectFaction(localPlayerId, factionType, gameId)
    }
  }

  // Build map of selected factions with player info
  const selectedFactionsMap = useMemo(() => {
    const map = new Map<string, { playerNumber: number, vp: number }>()

    if (!gameState?.players || !gameState.turnOrder) return map

    gameState.turnOrder.forEach((playerId: string, index: number) => {
      const player = gameState.players[playerId]
      if (!player) return

      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      const factionRaw = player.faction ?? player.Faction

      if (factionRaw !== undefined) {
        let factionId: number | undefined

        if (typeof factionRaw === 'object' && factionRaw !== null && 'Type' in factionRaw) {
          factionId = (factionRaw as { Type: number }).Type
        } else if (typeof factionRaw === 'object' && factionRaw !== null && 'type' in factionRaw) {
          factionId = (factionRaw as { type: number }).type
        } else if (typeof factionRaw === 'number') {
          factionId = factionRaw
        }

        if (factionId !== undefined) {
          const factionType = FACTIONS.find(f => f.id === (factionId as FactionType))?.type
          if (factionType) {
            map.set(factionType, {
              playerNumber: index + 1,

              vp: player.victoryPoints ?? player.VictoryPoints ?? 20
            })
          }
        }
      }
    })

    return map
  }, [gameState])

  // Get current player's position (1-based index)
  const currentPlayerPosition = useMemo(() => {
    if (!gameState?.turnOrder || !localPlayerId) return 1
    const index = gameState.turnOrder.indexOf(localPlayerId)
    return index !== -1 ? index + 1 : 1
  }, [gameState, localPlayerId])

  const cultPositions = useMemo(() => {
    if (gameState?.phase === GamePhase.FactionSelection) {
      return new Map([
        [CultType.Fire, []],
        [CultType.Water, []],
        [CultType.Earth, []],
        [CultType.Air, []],
      ])
    }
    return getCultPositions(gameState)
  }, [gameState])

  const isMyTurn = gameState?.turnOrder[gameState.currentTurn] === localPlayerId

  return (
    <div className="min-h-screen p-4 bg-gray-100">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-3xl font-bold text-gray-800">Terra Mystica - Game {gameId}</h1>
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

        {gameState?.players && (
          <div className="mb-3">
            <PlayerSummaryBar gameState={gameState} />
          </div>
        )}

        {/* Faction Selector - shown above game board during faction selection phase */}
        {gameState?.phase === GamePhase.FactionSelection && (
          <FactionSelector
            selectedFactions={selectedFactionsMap}
            onSelect={handleFactionSelect}
            isMyTurn={isMyTurn}
            currentPlayerPosition={currentPlayerPosition}
          />
        )}

        {/* Draggable Grid Layout */}
        <ResponsiveGridLayout
          className={`layout ${isLayoutLocked ? 'layout-locked' : ''}`}
          layouts={layouts}
          breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
          cols={{ lg: 24, md: 20, sm: 12, xs: 8, xxs: 4 }}
          rowHeight={rowHeight}
          onLayoutChange={handleLayoutChange}
          onWidthChange={handleWidthChange}
          isDraggable={!isLayoutLocked}
          isResizable={!isLayoutLocked}
          resizeHandles={['e']}
          draggableHandle=".drag-handle"
        >
          {/* Scoring Tiles */}
          <div key="scoring" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto">
              <ScoringTiles
                tiles={gameState?.scoringTiles?.tiles.map(t => t.type) || []}
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
              <GameBoard onHexClick={handleHexClick} />
            </div>
          </div>

          {/* Cult Tracks sidebar */}
          <div key="cult" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto p-2">
              <CultTracks
                cultPositions={cultPositions}
              />
            </div>
          </div>

          {/* Town Tiles */}
          <div key="towns" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto">
              <TownTiles availableTiles={
                gameState?.townTiles?.available
                  ? Object.entries(gameState.townTiles.available).flatMap(([id, count]) => Array.from({ length: count }, () => Number(id)))
                  : []
              } />
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

        <details className="mt-8 p-4 bg-gray-200 rounded">
          <summary className="font-bold cursor-pointer">Debug: Game State Players</summary>
          <pre className="mt-2 text-xs overflow-auto max-h-96">
            {JSON.stringify(gameState?.players, null, 2)}
          </pre>
        </details>
      </div>
    </div>
  )
}
