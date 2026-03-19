import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { WebSocketProvider } from './services/WebSocketContext';
import { Lobby } from './components/Lobby';
import { Game } from './components/Game';
import { MapTest } from './components/MapTest';
import { CultTracksTest } from './components/CultTracks/CultTracksTest';
import { Replay } from './components/Replay';
import { ImportGame } from './components/ImportGame';
import './App.css'

function App(): React.ReactElement {
  const showTestRoutes = import.meta.env.DEV

  // Get base path from environment, default to '/' for local dev
  const basePath = import.meta.env.VITE_BASE_PATH || '/';

  return (
    <WebSocketProvider>
      <Router basename={basePath}>
        <div style={{
          minHeight: '100vh',
          background: 'linear-gradient(135deg, #08111f 0%, #13263d 50%, #2d1711 100%)',
        }}
        >
          <Routes>
            <Route path="/" element={<Lobby />} />
            <Route path="/import" element={<ImportGame />} />
            <Route path="/game/:gameId" element={<Game />} />
            <Route path="/replay/:gameId" element={<Replay />} />
            {showTestRoutes && <Route path="/maptest" element={<MapTest />} />}
            {showTestRoutes && <Route path="/culttrackstest" element={<CultTracksTest />} />}
          </Routes>
        </div>
      </Router>
    </WebSocketProvider>
  )
}

export default App
