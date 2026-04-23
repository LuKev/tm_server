import { BrowserRouter as Router, Route, Routes, useLocation } from 'react-router-dom';
import { WebSocketProvider } from './services/WebSocketContext';
import { Lobby } from './components/Lobby';
import { Game } from './components/Game';
import { Replay } from './components/Replay';
import { ImportGame } from './components/ImportGame';
import './App.css'

function AppShell(): React.ReactElement {
  const location = useLocation();
  const isPlainGameShell = location.pathname.startsWith('/game/') || location.pathname.startsWith('/replay/');

  return (
    <div
      style={{
        minHeight: '100vh',
        background: isPlainGameShell
          ? '#ffffff'
          : 'linear-gradient(135deg, #08111f 0%, #13263d 50%, #2d1711 100%)',
        color: isPlainGameShell ? '#111827' : undefined,
      }}
    >
      <Routes>
        <Route path="/" element={<Lobby />} />
        <Route path="/import" element={<ImportGame />} />
        <Route path="/game/:gameId" element={<Game />} />
        <Route path="/replay/:gameId" element={<Replay />} />
      </Routes>
    </div>
  );
}

function App(): React.ReactElement {
  // Get base path from environment, default to '/' for local dev
  const basePath = import.meta.env.VITE_BASE_PATH || '/';

  return (
    <WebSocketProvider>
      <Router basename={basePath}>
        <AppShell />
      </Router>
    </WebSocketProvider>
  );
}

export default App
