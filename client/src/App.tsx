import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { WebSocketProvider } from './services/WebSocketContext';
import { Lobby } from './components/Lobby';
import { Game } from './components/Game';
import { MapTest } from './components/MapTest';
import { CultTracksTest } from './components/CultTracks/CultTracksTest';

function App(): React.ReactElement {
  return (
    <WebSocketProvider>
      <Router>
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
          <Routes>
            <Route path="/" element={<Lobby />} />
            <Route path="/game/:gameId" element={<Game />} />
            <Route path="/maptest" element={<MapTest />} />
            <Route path="/culttrackstest" element={<CultTracksTest />} />
          </Routes>
        </div>
      </Router>
    </WebSocketProvider>
  )
}

export default App
