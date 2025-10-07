import { useParams } from 'react-router-dom'

function Game() {
  const { gameId } = useParams()

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-white mb-4">Game Room</h1>
        <p className="text-gray-300">Game ID: {gameId}</p>
        <p className="text-gray-400 mt-4">Game board will be implemented in Phase 9</p>
      </div>
    </div>
  )
}

export default Game
