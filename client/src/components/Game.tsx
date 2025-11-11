import { useParams } from 'react-router-dom'
import { GameBoard } from './GameBoard/GameBoard'
import { CultTracks } from './CultTracks/CultTracks'
import { useGameStore } from '../stores/gameStore'
import { CultType } from '../types/game.types'

export function Game() {
  const { gameId } = useParams()
  const gameState = useGameStore((state) => state.gameState)
  
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
    gameState.order.forEach(playerId => {
      const player = gameState.players[playerId]
      if (!player) return
      
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
    })
    
    return positions
  }

  return (
    <div className="min-h-screen p-4 bg-gray-100">
      <div className="max-w-[1800px] mx-auto">
        <h1 className="text-3xl font-bold text-gray-800 mb-4">Terra Mystica - Game {gameId}</h1>
        
        <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'flex-start' }}>
          {/* Main game board */}
          <div style={{ marginRight: '1rem' }}>
            <GameBoard />
          </div>
          
          {/* Cult Tracks sidebar */}
          <div style={{ width: '280px', flexShrink: 0 }}>
            <div className="bg-white rounded-lg shadow-md p-4" style={{ position: 'sticky', top: '1rem' }}>
              <CultTracks 
                cultPositions={getCultPositions()}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
