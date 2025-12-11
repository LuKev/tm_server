import { useParams } from 'react-router-dom'
import { useMemo, useEffect, useState } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { ScoringTiles } from './GameBoard/ScoringTiles'
import { TownTiles } from './GameBoard/TownTiles'
import { CultTracks } from './CultTracks/CultTracks'
import { FactionSelector } from './FactionSelector'
import { FACTIONS } from '../data/factions'
import { useGameStore } from '../stores/gameStore'
import { useActionService } from '../services/actionService'
import { CultType, GamePhase } from '../types/game.types'
import { useWebSocket } from '../services/WebSocketContext'
import { Responsive, WidthProvider, Layouts } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import './Game.css'

const ResponsiveGridLayout = WidthProvider(Responsive)

export function Game() {
  const { gameId } = useParams()
  const { sendMessage, isConnected } = useWebSocket()
  const gameState = useGameStore((state) => state.gameState)
  const localPlayerId = useGameStore((state) => state.localPlayerId)

  console.log('Game.tsx: Render', { gameId, gameState, localPlayerId })
  const { submitSetupDwelling, submitSelectFaction } = useActionService()

  // Default layout configuration
  const defaultLayouts = {
    lg: [
      { i: 'scoring', x: 0, y: 0, w: 2, h: 10, minW: 2, minH: 6 },
      { i: 'board', x: 2, y: 0, w: 8, h: 11, minW: 6, minH: 10 },
      { i: 'cult', x: 10, y: 0, w: 2, h: 10, minW: 2, minH: 6 },
      { i: 'towns', x: 0, y: 10, w: 4, h: 4, minW: 2, minH: 3 }
    ]
  }

  const [layouts, setLayouts] = useState<Layouts>(defaultLayouts)

  useEffect(() => {
    if (isConnected && gameId && !gameState) {
      console.log('Game.tsx: Requesting game state for', gameId)
      sendMessage({ type: 'get_game_state', payload: { gameID: gameId } })
    }
  }, [isConnected, gameId, gameState, sendMessage])

  // Handle hex clicks
  const handleHexClick = (q: number, r: number) => {
    if (!localPlayerId) {
      console.warn('No local player ID set')
      return
    }

    console.log(`Hex clicked: q=${q}, r=${r}`)

    // For now, always submit setup dwelling action
    // TODO: Add logic to determine action type based on game phase and hex state
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

    if (!gameState || !gameState.players || !gameState.order) return map

    gameState.order.forEach((playerId: string, index: number) => {
      const player = gameState.players[playerId] as any
      // Check both lowercase and PascalCase for faction and VP
      const factionRaw = player.faction || player.Faction

      if (player && factionRaw !== undefined) {
        let factionId: number | undefined

        if (typeof factionRaw === 'object' && factionRaw !== null) {
          // Handle Faction object (Go struct marshalled to JSON)
          // Go fields are usually capitalized (Type), but check both
          factionId = factionRaw.Type || factionRaw.type
        } else if (typeof factionRaw === 'number') {
          factionId = factionRaw
        }

        if (factionId !== undefined) {
          const factionType = FACTIONS.find(f => f.id === factionId)?.type
          if (factionType) {
            map.set(factionType, {
              playerNumber: index + 1,
              vp: player.VictoryPoints || player.victoryPoints || 20
            })
          }
        }
      }
    })

    return map
  }, [gameState])

  // Get current player's position in turn order
  const currentPlayerPosition = useMemo(() => {
    if (!gameState || !gameState.order || !localPlayerId) return 1
    const index = gameState.order.indexOf(localPlayerId)
    return index >= 0 ? index + 1 : 1
  }, [gameState, localPlayerId])

  // Convert player cult positions to CultTracks format
  const getCultPositions = () => {
    const positions = new Map()

    if (!gameState) {
      // Return empty positions if no game state
      positions.set(CultType.Fire, [])
      positions.set(CultType.Water, [])
      positions.set(CultType.Earth, [])
      positions.set(CultType.Air, [])
      return positions
    }

    // Initialize each cult with empty array
    positions.set(CultType.Fire, [])
    positions.set(CultType.Water, [])
    positions.set(CultType.Earth, [])
    positions.set(CultType.Air, [])

    // Collect all player positions on each cult track
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
                position: position as number,
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

  return (
    <div className="min-h-screen p-4 bg-gray-100">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-3xl font-bold text-gray-800">Terra Mystica - Game {gameId}</h1>
          <button
            onClick={() => setLayouts(defaultLayouts)}
            className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded text-sm font-medium text-gray-700 transition-colors"
          >
            Reset Layout
          </button>
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
          className="layout"
          layouts={layouts}
          breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
          cols={{ lg: 12, md: 10, sm: 6, xs: 4, xxs: 2 }}
          rowHeight={60}
          onLayoutChange={(layout, allLayouts) => setLayouts(allLayouts)}
          isDraggable={true}
          isResizable={true}
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
        </ResponsiveGridLayout>
      </div>
    </div>
  )
}
