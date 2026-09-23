import { PowerActionType, TerrainType } from '../types/game.types'
import type { usePowerSpadeDraft } from '../hooks/usePowerSpadeDraft'
import { ActionButton } from './shared/GamePrimitives'

export function PowerSpadeControls({ controller, cost, formatHex, terrainChoices, canSkip, onCancel }: {
  cost: number
  controller: ReturnType<typeof usePowerSpadeDraft>
  formatHex: (hex: { q: number; r: number }) => string
  terrainChoices: Array<{ id: TerrainType; name: string }>
  canSkip: boolean
  onCancel: () => void
}): React.ReactElement | null {
  const { draft, edit } = controller
  if (!draft) return null
  const busy = draft.status === 'submitting'
  return <fieldset className="power-spade-draft" disabled={busy} aria-busy={busy}>
    <legend>Power spades</legend>
    <p>{cost} {draft.useCoins ? 'coins' : 'power'} to claim, plus any additional digging or dwelling costs. Choose all destinations before submitting.</p>
    <div className="flex flex-wrap items-end gap-3">
      {(['first', 'second'] as const).map((slot) => {
        if (slot === 'second' && draft.actionType !== PowerActionType.DoubleSpade) return null
        const target = draft[slot]
        return <div key={slot} className="space-y-2">
          <button type="button" data-testid={`power-spade-${slot}-target`} aria-pressed={draft.choosing === slot}
            onClick={() => { edit({ choosing: slot }); }}>
            {slot === 'first' ? 'First' : 'Second'} target: {target ? formatHex(target.hex) : 'choose on board'}
          </button>
          {target && <label className="flex items-center gap-2">
            Terrain
            <select data-testid={slot === 'first' ? 'hex-action-target-terrain' : 'power-spade-second-terrain'}
              value={target.terrain} onChange={event => { edit({ [slot]: { ...target, terrain: Number(event.target.value) as TerrainType } }); }}>
              {terrainChoices.map(terrain => <option key={terrain.id} value={terrain.id}>{terrain.name}</option>)}
            </select>
          </label>}
          {slot === 'second' && target && <button type="button" onClick={() => { edit({ second: null, choosing: 'first' }); }}>Remove second target</button>}
        </div>
      })}
      {draft.first && <label className="flex items-center gap-2">Action
        <select data-testid="hex-action-mode" value={draft.buildDwelling ? 'transform_build' : 'transform_only'}
          onChange={event => { edit({ buildDwelling: event.target.value === 'transform_build' }); }}>
          <option value="transform_build">Transform + build dwelling on first target</option>
          <option value="transform_only">Transform only</option>
        </select>
      </label>}
      {canSkip && <label><input type="checkbox" checked={draft.useSkip} onChange={event => { edit({ useSkip: event.target.checked }); }} /> Use flight/tunneling</label>}
    </div>
    <p className="text-sm">{draft.first ? `${formatHex(draft.first.hex)} → ${TerrainType[draft.first.terrain]}${draft.buildDwelling ? ' + dwelling' : ''}${draft.second ? `; ${formatHex(draft.second.hex)} → ${TerrainType[draft.second.terrain]}` : ''}. Any unused spades are forfeited.` : 'Choose the first hex, or explicitly claim the action and forfeit its spades.'}</p>
    {draft.error && <p role="alert">{draft.error}</p>}
    <div className="flex gap-2">
      <button type="button" data-testid="hex-action-cancel" onClick={onCancel}>Cancel</button>
      <ActionButton data-testid="hex-action-submit" onClick={controller.submit}>{busy ? 'Submitting…' : draft.first ? 'Submit' : 'Claim and forfeit spades'}</ActionButton>
    </div>
  </fieldset>
}
