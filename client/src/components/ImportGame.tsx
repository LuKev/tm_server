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



                <div className="import-game-info">
                    <h3>Bookmarklet Import (Recommended)</h3>
                    <p>Copy the code below and save it as a bookmark:</p>
                    <textarea
                        className="bookmarklet-code"
                        readOnly
                        value={"javascript:(function(){var logs=document.getElementById('gamelogs')||document.getElementById('logs');if(!logs){alert('No game logs found!\\n\\nMake sure you are on the Game Review page:\\nhttps://boardgamearena.com/gamereview?table=YOUR_TABLE_ID\\n\\n(Not the /table page)');return;}var html=logs.innerHTML;var m=window.location.href.match(/table[=/](\\d+)/);if(!m){alert('Could not find game ID in URL');return;}var gameId=m[1];var form=document.createElement('form');form.method='POST';form.action='https://kezilu.com/api/replay/import_form';var i1=document.createElement('input');i1.type='hidden';i1.name='gameId';i1.value=gameId;form.appendChild(i1);var i2=document.createElement('input');i2.type='hidden';i2.name='html';i2.value=html;form.appendChild(i2);document.body.appendChild(form);form.submit();})();"}
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
                            <li>Name it "Import to Kezilu".</li>
                            <li>Paste the code into the URL/Location field.</li>
                            <li>Save the bookmark.</li>
                        </ol>
                        <strong>To use:</strong> Go to the <strong>Game Review</strong> page on Board Game Arena
                        (URL should look like: <code>boardgamearena.com/gamereview?table=XXX</code>) and click the bookmark.
                        <br /><br />
                        <em>Note: This works on the Game Review page, not the live game page (/table).</em>
                    </p>
                </div>

                <div className="import-game-info">
                    <h3>Supported Sources</h3>
                    <ul>
                        <li><strong>Bookmarklet</strong> - Best for importing directly from BGA (recommended)</li>
                        <li><strong>Paste URL</strong> - Works if you are running the server locally (requires login)</li>
                    </ul>
                </div>
            </div>
        </div>
    );
};
