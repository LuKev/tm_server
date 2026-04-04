import React, { useMemo, useState } from 'react'
import { TerrainType } from '../types/game.types'
import type { CustomMapDefinition } from '../types/map.types'
import { HexGridCanvas } from './GameBoard/HexGridCanvas'
import { TERRAIN_COLORS } from '../utils/colors'
import {
  TERRAIN_BRUSH_OPTIONS,
  applyTerrainToHex,
  buildCustomMapHexes,
  countLandHexes,
  parseCustomMapDefinition,
  resizeCustomMapDefinition,
  serializeCustomMapDefinition,
} from '../utils/customMapUtils'
import './CustomMapEditor.css'

interface CustomMapEditorProps {
  value: CustomMapDefinition
  onChange: (definition: CustomMapDefinition) => void
  onCreateGame?: () => void
  onStartGame?: () => void
  createGameDisabled?: boolean
  startGameDisabled?: boolean
  disabled?: boolean
}

export function CustomMapEditor({
  value,
  onChange,
  onCreateGame,
  onStartGame,
  createGameDisabled = false,
  startGameDisabled = false,
  disabled = false,
}: CustomMapEditorProps): React.ReactElement {
  const [selectedTerrain, setSelectedTerrain] = useState<TerrainType>(TerrainType.Plains)
  const [importText, setImportText] = useState('')
  const [importError, setImportError] = useState<string | null>(null)
  const [saveStatus, setSaveStatus] = useState<string | null>(null)

  const hexes = useMemo(() => buildCustomMapHexes(value), [value])
  const landHexCount = useMemo(() => countLandHexes(value), [value])
  const serializedMap = useMemo(() => serializeCustomMapDefinition(value), [value])

  const handleImport = (): void => {
    try {
      const imported = parseCustomMapDefinition(importText)
      onChange({
        ...imported,
        name: value.name ?? '',
      })
      setImportError(null)
    } catch (error) {
      setImportError(error instanceof Error ? error.message : 'Failed to import map.')
    }
  }

  const handleCopy = async (): Promise<void> => {
    try {
      await navigator.clipboard.writeText(serializedMap)
      setSaveStatus('Copied map text to clipboard.')
    } catch {
      setSaveStatus('Could not copy automatically. Select the text and copy it manually.')
    }
  }

  const handleDownload = (): void => {
    const blob = new Blob([serializedMap], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    const name = value.name?.trim() || 'custom-map'
    anchor.href = url
    anchor.download = `${name}.txt`
    anchor.click()
    URL.revokeObjectURL(url)
    setSaveStatus('Downloaded map text file.')
  }

  return (
    <div className="custom-map-editor">
      <div className="custom-map-editor-header">
        <div>
          <h3>Custom Map Editor</h3>
          <p>Set the board shape, pick a terrain brush, then click hexes to paint the map.</p>
        </div>
        <div className="custom-map-editor-stats">
          <span>{String(value.rowCount)} rows</span>
          <span>{String(landHexCount)} land hexes</span>
        </div>
      </div>

      <div className="custom-map-editor-controls">
        <label className="custom-map-editor-field">
          <span>Name</span>
          <input
            type="text"
            value={value.name ?? ''}
            onChange={(event) => {
              onChange({ ...value, name: event.target.value })
            }}
            placeholder="Custom"
            disabled={disabled}
            data-testid="custom-map-name"
          />
        </label>

        <label className="custom-map-editor-field">
          <span>Rows</span>
          <input
            type="number"
            min={1}
            step={1}
            value={value.rowCount}
            onChange={(event) => {
              onChange(resizeCustomMapDefinition(value, { rowCount: Number(event.target.value) }))
            }}
            disabled={disabled}
            data-testid="custom-map-row-count"
          />
        </label>

        <label className="custom-map-editor-field">
          <span>First row columns</span>
          <input
            type="number"
            min={1}
            step={1}
            value={value.firstRowColumns}
            onChange={(event) => {
              onChange(resizeCustomMapDefinition(value, { firstRowColumns: Number(event.target.value) }))
            }}
            disabled={disabled}
            data-testid="custom-map-first-row-columns"
          />
        </label>

        <label className="custom-map-editor-field">
          <span>Row pattern</span>
          <select
            value={value.firstRowLonger ? 'first-longer' : 'second-longer'}
            onChange={(event) => {
              onChange(resizeCustomMapDefinition(value, { firstRowLonger: event.target.value === 'first-longer' }))
            }}
            disabled={disabled}
            data-testid="custom-map-row-pattern"
          >
            <option value="first-longer">First row longer</option>
            <option value="second-longer">Second row longer</option>
          </select>
        </label>
      </div>

      <div className="custom-map-editor-palette">
        {TERRAIN_BRUSH_OPTIONS.map((option) => (
          <button
            key={option.terrain}
            type="button"
            className={`custom-map-editor-swatch ${selectedTerrain === option.terrain ? 'is-active' : ''}`}
            style={{ backgroundColor: option.terrain === TerrainType.River ? '#b3d9ff' : TERRAIN_COLORS[option.terrain] }}
            onClick={() => { setSelectedTerrain(option.terrain) }}
            disabled={disabled}
            data-testid={`custom-map-brush-${option.label.toLowerCase()}`}
          >
            <span>{option.label}</span>
            <span>{option.importCode}</span>
          </button>
        ))}
      </div>

      <div className="custom-map-editor-canvas">
        <HexGridCanvas
          testId="custom-map-editor-canvas"
          hexes={hexes}
          onHexClick={(q, r) => {
            if (disabled) return
            onChange(applyTerrainToHex(value, q, r, selectedTerrain))
          }}
        />
      </div>

      <div className="custom-map-editor-save">
        <div className="custom-map-editor-import-header">
          <h4>Save Map Grid</h4>
          <p>Export the current board in the same comma-separated row format used for imports.</p>
        </div>
        <div className="custom-map-editor-import-actions">
          <button
            type="button"
            className="custom-map-editor-import-button"
            onClick={() => { void handleCopy() }}
            disabled={disabled}
            data-testid="custom-map-copy-button"
          >
            Copy map text
          </button>
          <button
            type="button"
            className="custom-map-editor-secondary-button"
            onClick={handleDownload}
            disabled={disabled}
            data-testid="custom-map-download-button"
          >
            Download `.txt`
          </button>
        </div>
        <textarea
          value={serializedMap}
          readOnly
          className="custom-map-editor-textarea"
          data-testid="custom-map-export-textarea"
        />
        {saveStatus && <div className="custom-map-editor-status">{saveStatus}</div>}
      </div>

      <div className="custom-map-editor-launch">
        <div className="custom-map-editor-import-header">
          <h4>Use This Map</h4>
          <p>Create a normal lobby game with this map, or start a 1-player game immediately.</p>
        </div>
        <div className="custom-map-editor-import-actions">
          <button
            type="button"
            className="custom-map-editor-import-button"
            onClick={onCreateGame}
            disabled={createGameDisabled}
            data-testid="custom-map-create-game-button"
          >
            Create game with this map
          </button>
          <button
            type="button"
            className="custom-map-editor-secondary-button"
            onClick={onStartGame}
            disabled={startGameDisabled}
            data-testid="custom-map-start-game-button"
          >
            Start 1-player game now
          </button>
        </div>
      </div>

      <div className="custom-map-editor-import">
        <div className="custom-map-editor-import-header">
          <h4>Import Map Grid</h4>
          <p>Upload or paste rows like `K,B,R,G,...` where `K=swamp`, `I=river`, `B=lake`, `R=wasteland`, `G=forest`, `S=mountain`, `Y=desert`, `U=plains`.</p>
        </div>
        <div className="custom-map-editor-import-actions">
          <label className="custom-map-editor-file">
            <span>Load file</span>
            <input
              type="file"
              accept=".txt,.csv"
              disabled={disabled}
              onChange={(event) => {
                const file = event.target.files?.[0]
                if (!file) return
                void file.text().then((text) => {
                  setImportText(text)
                  setImportError(null)
                })
              }}
              data-testid="custom-map-file-input"
            />
          </label>
          <button
            type="button"
            className="custom-map-editor-import-button"
            onClick={handleImport}
            disabled={disabled || importText.trim() === ''}
            data-testid="custom-map-import-button"
          >
            Import grid
          </button>
        </div>
        <textarea
          value={importText}
          onChange={(event) => {
            setImportText(event.target.value)
            if (importError) setImportError(null)
          }}
          className="custom-map-editor-textarea"
          placeholder="K,B,R,G,B,U,I,R,U,K,I,B,G"
          disabled={disabled}
          data-testid="custom-map-import-textarea"
        />
        {importError && <div className="custom-map-editor-error" role="alert">{importError}</div>}
      </div>
    </div>
  )
}
