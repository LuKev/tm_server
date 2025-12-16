import { useParams } from 'react-router-dom'
import { useMemo, useEffect, useState } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { ScoringTiles } from './GameBoard/ScoringTiles'
import { TownTiles } from './GameBoard/TownTiles'
import { FavorTiles } from './GameBoard/FavorTiles'
import { PassingTiles } from './GameBoard/PassingTiles'
import { CultTracks } from './CultTracks/CultTracks'
import type { CultPosition } from './CultTracks/CultTracks'
import { FactionSelector } from './FactionSelector'
import { FACTIONS } from '../data/factions'
import { useGameStore } from '../stores/gameStore'
import { useActionService } from '../services/actionService'
import { CultType, GamePhase, type FactionType } from '../types/game.types'
import { useWebSocket } from '../services/WebSocketContext'
import { Responsive, WidthProvider, type Layouts } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'

const ResponsiveGridLayout = WidthProvider(Responsive)

export const Game = () => {
  const { gameId } = useParams()
  const { isConnected, sendMessage } = useWebSocket()
  const gameState = useGameStore((state) => state.gameState)
  const localPlayerId = useGameStore((state) => state.localPlayerId)

  const { submitSetupDwelling, submitSelectFaction } = useActionService()

  const numCards = gameState?.bonusCards?.length ?? 9

  // Default layout configuration
  // Adjusted for square grid cells (rowHeight = colWidth)
  // Granularity doubled (24 cols for lg, 20 for md)
  const defaultLayouts = useMemo(() => ({
    lg: [
      { i: 'scoring', x: 0, y: 0, w: 4, h: 8, minW: 4, minH: 6 },
      { i: 'board', x: 4, y: 0, w: 16, h: 16, minW: 12, minH: 10 },
      { i: 'cult', x: 20, y: 0, w: 4, h: 9, minW: 4, minH: 6 },
      { i: 'towns', x: 0, y: 8, w: 4, h: 3, minW: 4, minH: 2 },
      { i: 'favor', x: 20, y: 9, w: 4, h: 4, minW: 4, minH: 2 },
      { i: 'passing', x: 24 - numCards, y: 16, w: numCards, h: 4, minW: 4, minH: 2 }
    ],
    md: [
      { i: 'scoring', x: 0, y: 0, w: 4, h: 8, minW: 4, minH: 6 },
      { i: 'board', x: 4, y: 0, w: 12, h: 12, minW: 8, minH: 8 },
      { i: 'cult', x: 16, y: 0, w: 4, h: 9, minW: 4, minH: 6 },
      { i: 'towns', x: 0, y: 8, w: 4, h: 3, minW: 4, minH: 2 },
      { i: 'favor', x: 16, y: 9, w: 4, h: 4, minW: 4, minH: 2 },
      { i: 'passing', x: 20 - numCards, y: 16, w: numCards, h: 4, minW: 4, minH: 2 }
    ]
  }), [numCards])

  const [layouts, setLayouts] = useState<Layouts>(defaultLayouts)
  const [isLayoutLocked, setIsLayoutLocked] = useState(false)
  const [rowHeight, setRowHeight] = useState(60)

  useEffect(() => {
    if (isConnected && gameId && !gameState) {
      sendMessage({ type: 'get_game_state', payload: { gameID: gameId } })
    }
  }, [isConnected, gameId, gameState, sendMessage])

  // Update layout when numCards changes to ensure correct default size
  useEffect(() => {
    setLayouts((currentLayouts) => {
      const newLayouts = { ...currentLayouts }
      let hasChanges = false

      Object.keys(newLayouts).forEach((key) => {
        newLayouts[key] = newLayouts[key].map((item) => {
          if (item.i === 'passing') {
            if (item.w !== numCards || item.h !== 4) {
              hasChanges = true
              return { ...item, w: numCards, h: 4 }
            }
          }
          return item
        })
      })

      return hasChanges ? newLayouts : currentLayouts
    })
  }, [numCards])

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

    if (!gameState?.players || !gameState.order) return map

    gameState.order.forEach((playerId: string, index: number) => {
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
              // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
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
    if (!gameState?.order || !localPlayerId) return 1
    const index = gameState.order.indexOf(localPlayerId)
    return index !== -1 ? index + 1 : 1
  }, [gameState, localPlayerId])

  // Helper to get cult positions
  const getCultPositions = (): Map<CultType, CultPosition[]> => {
    const positions = new Map<CultType, CultPosition[]>()

    if (!gameState) {
      positions.set(CultType.Fire, [])
      positions.set(CultType.Water, [])
      positions.set(CultType.Earth, [])
      positions.set(CultType.Air, [])
      return positions
    }

    positions.set(CultType.Fire, [])
    positions.set(CultType.Water, [])
    positions.set(CultType.Earth, [])
    positions.set(CultType.Air, [])

    if (gameState.order && gameState.players) {
      gameState.order.forEach((playerId: string) => {
        const player = gameState.players[playerId]
        if (!player) return

        if (player.cults) {
          Object.entries(player.cults).forEach(([cultKey, position]) => {
            const cult = Number(cultKey) as CultType
            if (position !== undefined) {
              positions.get(cult)?.push({
                faction: player.faction,
                position: position,
                hasKey: false, // TODO: Track power keys from game state
              })
            }
          })
        }
      })
    }

    return positions
  }

  const isMyTurn = gameState?.order[gameState.currentTurn] === localPlayerId

  const handleWidthChange = (containerWidth: number, margin: [number, number], cols: number, containerPadding: [number, number]) => {
    const safeMargin = margin || [10, 10]
    const safePadding = containerPadding || [10, 10]
    const totalMargin = safeMargin[0] * (cols - 1)
    const totalPadding = safePadding[0] * 2
    const colWidth = (containerWidth - totalMargin - totalPadding) / cols
    setRowHeight(colWidth)
  }

  const handleLayoutChange = (_currentLayout: ReactGridLayout.Layout[], allLayouts: Layouts) => {
    // Enforce aspect ratios based on width
    // Scoring/Cult: h = w * 2
    // Board: h = ceil(w * 0.9)
    // Towns: h = ceil(w * 2/3)
    // Favor: h = ceil(w * 0.625) (8:5 ratio)

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
          newH = Math.ceil(item.w * 0.9)
        } else if (item.i === 'towns') {
          newH = Math.ceil(item.w * 2 / 3)
        } else if (item.i === 'favor') {
          newH = Math.ceil(item.w * 0.625)
        } else if (item.i === 'passing') {
          newH = Math.ceil(item.w * (4 / numCards))
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
              onClick={() => { setLayouts(defaultLayouts); }}
              className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-sm font-medium text-gray-700 transition-colors"
            >
              Reset Layout
            </button>
          </div>
        </div>

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
                tiles={gameState?.scoringTiles || []}
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
                cultPositions={
                  gameState?.phase === GamePhase.FactionSelection
                    ? new Map([
                      [CultType.Fire, []],
                      [CultType.Water, []],
                      [CultType.Earth, []],
                      [CultType.Air, []],
                    ])
                    : getCultPositions()
                }
              />
            </div>
          </div>

          {/* Town Tiles */}
          <div key="towns" className="bg-white rounded-lg shadow-md overflow-hidden flex flex-col">
            <div className="drag-handle">
              <div className="drag-handle-pill" />
            </div>
            <div className="flex-1 overflow-auto p-2">
              <TownTiles availableTiles={gameState?.townTiles} />
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
