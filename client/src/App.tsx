import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Lobby from './components/Lobby'
import Game from './components/Game'
import { WebSocketProvider } from './services/WebSocketContext'

function App() {
  return (
    <WebSocketProvider>
      <Router>
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
          <Routes>
            <Route path="/" element={<Lobby />} />
            <Route path="/game/:gameId" element={<Game />} />
          </Routes>
        </div>
      </Router>
    </WebSocketProvider>
  )
}

export default App
