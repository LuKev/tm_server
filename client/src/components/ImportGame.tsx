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

    // Return as-is for non-numeric replay IDs
    return trimmed;
}

interface ImportStatus {
    status: 'idle' | 'loading' | 'error' | 'success';
    message: string;
}

export const ImportGame: React.FC = () => {
    const [urlInput, setUrlInput] = useState('');
    const [manualGameIdInput, setManualGameIdInput] = useState('');
    const [manualLogInput, setManualLogInput] = useState('');
    const [manualFormat, setManualFormat] = useState<'auto' | 'concise' | 'snellman' | 'bga'>('auto');
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

    const handleImportText = async (): Promise<void> => {
        if (!manualLogInput.trim()) {
            setImportStatus({ status: 'error', message: 'Please paste a log before importing.' });
            return;
        }

        const gameId = manualGameIdInput.trim() ? extractGameId(manualGameIdInput) : `import-${Date.now()}`;
        setImportStatus({ status: 'loading', message: 'Parsing log and building replay...' });

        try {
            const res = await fetch('/api/replay/import_text', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    gameId,
                    logText: manualLogInput,
                    format: manualFormat
                })
            });

            if (!res.ok) {
                const errorText = await res.text();
                throw new Error(errorText || 'Failed to import log text');
            }

            setImportStatus({ status: 'success', message: 'Log imported! Redirecting...' });
            setTimeout(() => {
                void navigate(`/replay/${gameId}`);
            }, 500);
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Unknown error';
            setImportStatus({ status: 'error', message: `Failed to import log text: ${message}` });
        }
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
                    <span>or paste a log</span>
                </div>

                <div className="import-game-section">
                    <label className="import-game-label">
                        Replay ID (optional):
                    </label>
                    <input
                        type="text"
                        className="import-game-input"
                        placeholder="my-snellman-game-2026"
                        value={manualGameIdInput}
                        onChange={(e) => { setManualGameIdInput(e.target.value); }}
                        disabled={importStatus.status === 'loading'}
                    />
                    <label className="import-game-label">
                        Log format:
                    </label>
                    <select
                        className="import-game-input import-game-select"
                        value={manualFormat}
                        onChange={(e) => { setManualFormat(e.target.value as 'auto' | 'concise' | 'snellman' | 'bga'); }}
                        disabled={importStatus.status === 'loading'}
                    >
                        <option value="auto">Auto-detect</option>
                        <option value="concise">Concise notation</option>
                        <option value="snellman">Snellman text log</option>
                        <option value="bga">BGA text log</option>
                    </select>
                    <label className="import-game-label">
                        Paste log text:
                    </label>
                    <textarea
                        className="import-game-input import-game-textarea"
                        placeholder="Game: Base&#10;ScoringTiles: ...&#10;..."
                        value={manualLogInput}
                        onChange={(e) => { setManualLogInput(e.target.value); }}
                        disabled={importStatus.status === 'loading'}
                    />
                    <button
                        className="import-game-button primary"
                        onClick={() => { void handleImportText(); }}
                        disabled={importStatus.status === 'loading' || !manualLogInput.trim()}
                    >
                        {importStatus.status === 'loading' ? 'Importing...' : 'Import Pasted Log'}
                    </button>
                </div>


                <div className="import-game-info">
                    <h3>Bookmarklet Import (Recommended)</h3>
                    <p>Copy the code below and save it as a bookmark:</p>
                    <textarea
                        className="bookmarklet-code"
                        readOnly
                        value={"javascript:(function(){var url=window.location.href;var html,gameId;if(url.includes('terra.snellman.net')){var ledger=document.getElementById('ledger');if(!ledger){alert('No ledger found!\\n\\nMake sure to click \"Load full log\" first.');return;}if((ledger.innerText||'').indexOf('Load full log')!==-1){alert('Full log not loaded!\\n\\nClick \"Load full log\", wait for it to finish, then click the bookmark again.');return;}html=document.documentElement.outerHTML;var m=url.match(/\\/(faction|game)\\/([^/]+)/);if(!m){alert('Could not find game ID in URL');return;}gameId=m[2];}else{var logs=document.getElementById('gamelogs')||document.getElementById('logs');if(!logs){alert('No game logs found!\\n\\nMake sure you are on the Game Review page:\\nhttps://boardgamearena.com/gamereview?table=YOUR_TABLE_ID\\n\\n(Not the /table page)');return;}html=logs.innerHTML;var m=url.match(/table[=/](\\d+)/);if(!m){alert('Could not find game ID in URL');return;}gameId=m[1];}var form=document.createElement('form');form.method='POST';form.action='https://kezilu.com/api/replay/import_form';var i1=document.createElement('input');i1.type='hidden';i1.name='gameId';i1.value=gameId;form.appendChild(i1);var i2=document.createElement('input');i2.type='hidden';i2.name='html';i2.value=html;form.appendChild(i2);document.body.appendChild(form);form.submit();})();"}
                        onClick={(e) => {
                            const target = e.target as HTMLTextAreaElement;
                            target.select();
                            void navigator.clipboard.writeText(target.value);
                        }}
                    />
                    <p className="bookmarklet-instructions">
                        <strong>How to install:</strong>
                        <ol>
                            <li>Click the code above to copy it.</li>
                            <li>Create a new bookmark in your browser (Ctrl/Cmd+D or right-click bookmarks bar).</li>
                            <li>Name it &quot;Import to Kezilu&quot;.</li>
                            <li>Paste the code into the URL/Location field.</li>
                            <li>Save the bookmark.</li>
                        </ol>
                        <strong>To use:</strong>
                        <ul>
                            <li><strong>BGA:</strong> Go to the Game Review page (<code>boardgamearena.com/gamereview?table=XXX</code>) and click the bookmark.</li>
                            <li><strong>Snellman:</strong> Go to the faction page (<code>terra.snellman.net/faction/GAME_ID</code>), click &quot;Load full log&quot;, then click the bookmark.</li>
                        </ul>
                    </p>
                </div>

                <div className="import-game-info">
                    <h3>Supported Sources</h3>
                    <ul>
                        <li><strong>Board Game Arena</strong> - Game Review pages</li>
                        <li><strong>Snellman</strong> - Faction/game pages (remember to load full log first)</li>
                    </ul>
                </div>
            </div>
        </div>
    );
};
