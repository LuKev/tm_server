import { useParams } from 'react-router-dom'
import { useMemo, useEffect } from 'react'
import { GameBoard } from './GameBoard/GameBoard'
import { CultTracks } from './CultTracks/CultTracks'
import { FactionSelector } from './FactionSelector'
import { FACTIONS } from '../data/factions'
import { useGameStore } from '../stores/gameStore'
import { useActionService } from '../services/actionService'
import { CultType, GamePhase } from '../types/game.types'
import { useWebSocket } from '../services/WebSocketContext'

export function Game() {
  const { gameId } = useParams()
  const { sendMessage, isConnected } = useWebSocket()
  const gameState = useGameStore((state) => state.gameState)
  const localPlayerId = useGameStore((state) => state.localPlayerId)

  console.log('Game.tsx: Render', { gameId, gameState, localPlayerId })
  const { submitSetupDwelling, submitSelectFaction } = useActionService()

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
        <h1 className="text-3xl font-bold text-gray-800 mb-4">Terra Mystica - Game {gameId}</h1>

        {/* Faction Selector - shown above game board during faction selection phase */}
        {gameState?.phase === GamePhase.FactionSelection && (
          <FactionSelector
            selectedFactions={selectedFactionsMap}
            onSelect={handleFactionSelect}
            isMyTurn={isMyTurn}
            currentPlayerPosition={currentPlayerPosition}
          />
        )}

        {/* Game Board and Cult Tracks - side by side */}
        <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'flex-start' }}>
          {/* Main game board */}
          <div style={{ marginRight: '1rem' }}>
            <GameBoard onHexClick={handleHexClick} />
          </div>

          {/* Cult Tracks sidebar */}
          <div style={{ width: '280px', flexShrink: 0 }}>
            <div className="bg-white rounded-lg shadow-md p-4" style={{ position: 'sticky', top: '1rem' }}>
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
        </div>
      </div>
    </div>
  )
}
