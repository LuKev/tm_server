import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { WebSocketProvider } from './services/WebSocketContext';
import { Lobby } from './components/Lobby';
import { Game } from './components/Game';
import { MapTest } from './components/MapTest';
import { CultTracksTest } from './components/CultTracks/CultTracksTest';
import { Replay } from './components/Replay';
import { ImportGame } from './components/ImportGame';

function App(): React.ReactElement {
  // Get base path from environment, default to '/' for local dev
  const basePath = import.meta.env.VITE_BASE_PATH || '/';

  return (
    <WebSocketProvider>
      <Router basename={basePath}>
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
          <Routes>
            <Route path="/" element={<Lobby />} />
            <Route path="/import" element={<ImportGame />} />
            <Route path="/game/:gameId" element={<Game />} />
            <Route path="/replay/:gameId" element={<Replay />} />
            <Route path="/maptest" element={<MapTest />} />
            <Route path="/culttrackstest" element={<CultTracksTest />} />
          </Routes>
        </div>
      </Router>
    </WebSocketProvider>
  )
}

export default App
