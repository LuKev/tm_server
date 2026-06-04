import { useState, type ReactElement } from 'react'
import { Bot, Loader2 } from 'lucide-react'
import './AIWorkbench.css'

interface RankedAction {
  id: string
  type: string
  label: string
  playerId: string
  visits: number
  prior: number
  q: number
  prob: number
}

interface SuggestResponse {
  rootPlayerId: string
  turnPlayerId: string
  round: number
  phase: number
  result: {
    selected: RankedAction
    actions: RankedAction[]
    simulations: number
  }
}

export function AIWorkbench(): ReactElement {
  const [gameId, setGameId] = useState('')
  const [snapshot, setSnapshot] = useState('')
  const [rootPlayerId, setRootPlayerId] = useState('')
  const [simulations, setSimulations] = useState(1)
  const [result, setResult] = useState<SuggestResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const requestSuggestion = async (): Promise<void> => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch('/api/ai/suggest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gameId: gameId.trim(),
          snapshot,
          rootPlayerId: rootPlayerId.trim(),
          topN: 20,
          search: {
            simulations,
            cpuct: 1.5,
            temperature: 1,
            maxDepth: 120,
          },
        }),
      })
      if (!response.ok) {
        throw new Error(await response.text())
      }
      setResult(await response.json() as SuggestResponse)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AI request failed.')
      setResult(null)
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="ai-page">
      <section className="ai-shell">
        <header className="ai-header">
          <p className="ai-kicker">Search</p>
          <h1>1v1 Engine</h1>
        </header>

        <section className="ai-controls">
          <label>
            <span>Game ID</span>
            <input value={gameId} onChange={(event) => { setGameId(event.target.value) }} placeholder="Optional" />
          </label>
          <label>
            <span>Root Player</span>
            <input value={rootPlayerId} onChange={(event) => { setRootPlayerId(event.target.value) }} placeholder="Optional" />
          </label>
          <label>
            <span>Simulations</span>
            <input
              type="number"
              min={1}
              max={512}
              value={simulations}
              onChange={(event) => { setSimulations(Number(event.target.value)) }}
            />
          </label>
          <button className="ai-run-button" onClick={() => { void requestSuggestion() }} disabled={loading}>
            {loading ? <Loader2 size={18} className="ai-spin" /> : <Bot size={18} />}
            <span>{loading ? 'Running' : 'Suggest'}</span>
          </button>
        </section>

        <label className="ai-snapshot">
          <span>Snapshot</span>
          <textarea value={snapshot} onChange={(event) => { setSnapshot(event.target.value) }} />
        </label>

        {error !== null && <div className="ai-error">{error}</div>}

        {result !== null && (
          <section className="ai-results">
            <div className="ai-summary">
              <span>Round {result.round}</span>
              <span>Turn {result.turnPlayerId || 'unknown'}</span>
              <span>{result.result.simulations} sims</span>
            </div>
            <table>
              <thead>
                <tr>
                  <th>Move</th>
                  <th>Player</th>
                  <th>Visits</th>
                  <th>Policy</th>
                  <th>Value</th>
                </tr>
              </thead>
              <tbody>
                {result.result.actions.map((action) => (
                  <tr key={action.id}>
                    <td>
                      <div className="ai-move-label">{action.label}</div>
                      <div className="ai-move-id">{action.id}</div>
                    </td>
                    <td>{action.playerId}</td>
                    <td>{action.visits}</td>
                    <td>{formatPercent(action.prob)}</td>
                    <td>{action.q.toFixed(3)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )}
      </section>
    </main>
  )
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(1)}%`
}
