import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import './ImportGame.css';

/**
 * Extract game ID from BGA URL or return as-is if already a game ID
 */
function extractGameId(input: string): string {
    // Remove whitespace
    const trimmed = input.trim();

    // Check if it's a BGA URL with table parameter
    const tableRegex = /table[=?](\d+)/;
    const tableMatch = tableRegex.exec(trimmed);
    if (tableMatch) return tableMatch[1];

    // Check if it's just a numeric ID
    const numericRegex = /^\d+$/;
    if (numericRegex.test(trimmed)) return trimmed;

    // Return as-is (could be "local" or other special values)
    return trimmed;
}

interface ImportStatus {
    status: 'idle' | 'loading' | 'error' | 'success';
    message: string;
}

export const ImportGame: React.FC = () => {
    const [urlInput, setUrlInput] = useState('');
    const [importStatus, setImportStatus] = useState<ImportStatus>({ status: 'idle', message: '' });
    const navigate = useNavigate();

    const handleImport = async (): Promise<void> => {
        const gameId = extractGameId(urlInput);

        if (!gameId) {
            setImportStatus({ status: 'error', message: 'Please enter a valid BGA URL or game ID' });
            return;
        }

        setImportStatus({ status: 'loading', message: 'Fetching game log from BGA... (this may take a minute, and you may need to log in)' });

        try {
            // Start the replay - this will trigger the backend to fetch the log
            const res = await fetch('/api/replay/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ gameId, restart: true })
            });

            if (!res.ok) {
                const errorText = await res.text();
                throw new Error(errorText || 'Failed to fetch game');
            }

            setImportStatus({ status: 'success', message: 'Game loaded! Redirecting...' });

            // Navigate to replay page
            setTimeout(() => {
                void navigate(`/replay/${gameId}`);
            }, 500);
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Unknown error';
            setImportStatus({ status: 'error', message: `Failed to import game: ${message}` });
        }
    };

    const handleLocalGame = (): void => {
        void navigate('/replay/local');
    };

    const handleKeyDown = (e: React.KeyboardEvent): void => {
        if (e.key === 'Enter' && importStatus.status !== 'loading') {
            void handleImport();
        }
    };

    return (
        <div className="import-game-container">
            <div className="import-game-card">
                <h1 className="import-game-title">Import Game</h1>

                <div className="import-game-section">
                    <label className="import-game-label">
                        Paste a BGA game URL or table ID:
                    </label>
                    <input
                        type="text"
                        className="import-game-input"
                        placeholder="https://boardgamearena.com/table?table=754319350"
                        value={urlInput}
                        onChange={(e) => { setUrlInput(e.target.value); }}
                        onKeyDown={handleKeyDown}
                        disabled={importStatus.status === 'loading'}
                    />
                    <button
                        className="import-game-button primary"
                        onClick={() => { void handleImport(); }}
                        disabled={importStatus.status === 'loading' || !urlInput.trim()}
                    >
                        {importStatus.status === 'loading' ? 'Importing...' : 'Import Game'}
                    </button>
                </div>

                {importStatus.status !== 'idle' && (
                    <div className={`import-game-status ${importStatus.status}`}>
                        {importStatus.status === 'loading' && (
                            <div className="import-game-spinner" />
                        )}
                        <span>{importStatus.message}</span>
                    </div>
                )}

                <div className="import-game-divider">
                    <span>or</span>
                </div>

                <div className="import-game-section">
                    <button
                        className="import-game-button secondary"
                        onClick={handleLocalGame}
                        disabled={importStatus.status === 'loading'}
                    >
                        Load Local Test Game
                    </button>
                </div>

                <div className="import-game-info">
                    <h3>Supported Sources</h3>
                    <ul>
                        <li><strong>Board Game Arena</strong> - Paste the game URL or table ID</li>
                        <li><strong>Local</strong> - Uses the test file in the server</li>
                    </ul>
                    <p className="import-game-note">
                        Note: The first time you import a BGA game, a browser window may open for you to log in.
                        Your session will be saved for future imports.
                    </p>
                </div>
            </div>
        </div>
    );
};
