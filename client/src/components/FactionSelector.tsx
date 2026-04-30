import type { FactionData } from '../data/factions'
import { FACTIONS } from '../data/factions'

interface FactionSelectorProps {
    selectedFactions: Map<string, { playerNumber: number, vp: number }> // faction type -> player info
    onSelect: (factionType: string) => void
    isMyTurn: boolean
    currentPlayerPosition: number
    enableFanFactions: boolean
    enableFireIceFactions: boolean
}

export function FactionSelector({
    selectedFactions,
    onSelect,
    isMyTurn,
    currentPlayerPosition,
    enableFanFactions,
    enableFireIceFactions,
}: FactionSelectorProps): React.ReactElement {
    const allFactions = FACTIONS
        .filter((faction) => enableFanFactions || !faction.isFanFaction)
        .filter((faction) => enableFireIceFactions || !faction.isFireIceFaction)
        .map((faction) => faction.type)

    // Get colors of selected factions to disable same-color factions
    const selectedColors = new Set(
        Array.from(selectedFactions.keys()).map(factionType => {
            const faction = FACTIONS.find(f => f.type === factionType)
            return faction?.color
        }).filter(Boolean)
    )

    // Helper to get faction data
    const getFactionData = (type: string): FactionData | undefined => {
        return FACTIONS.find(f => f.type === type)
    }

    // Helper to check if faction is available
    const isFactionAvailable = (type: string): boolean => {
        // Check if already selected
        if (selectedFactions.has(type)) return false

        // Check if same color as selected faction
        const faction = getFactionData(type)
        if (!faction) return false
        if (selectedColors.has(faction.color)) return false

        return true
    }

    // Color mapping for inline styles
    const colorHexMap: Record<string, { bg: string, border: string }> = {
        'yellow': { bg: '#FACC15', border: '#CA8A04' }, // yellow-400, yellow-600
        'green': { bg: '#22C55E', border: '#15803D' }, // green-500, green-700
        'amber': { bg: '#D97706', border: '#92400E' }, // amber-600, amber-800
        'blue': { bg: '#60A5FA', border: '#2563EB' }, // blue-400, blue-600
        'red': { bg: '#EF4444', border: '#B91C1C' }, // red-500, red-700
        'gray': { bg: '#9CA3AF', border: '#4B5563' }, // gray-400, gray-600
        'slate': { bg: '#334155', border: '#0F172A' }, // slate-700, slate-900
    }

    return (
        <div className="w-full mb-6" data-testid="faction-selector">
            <div className="bg-[#f5e6d3] rounded-lg shadow-2xl p-6 w-full">
                {/* Header */}
                <div className="bg-white border-2 border-gray-800 rounded-md py-2 px-4 mb-4 text-center">
                    <h1 className="text-lg font-bold text-gray-900">
                        You must select a Faction to play in position #{currentPlayerPosition}
                    </h1>
                </div>

                <div className="grid gap-4 w-full" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', width: '100%' }}>
                    {allFactions.map((factionType) => {
                        const faction = getFactionData(factionType)
                        const isSelected = selectedFactions.has(factionType)
                        const isAvailable = isFactionAvailable(factionType)
                        const selectionInfo = selectedFactions.get(factionType)

                        if (!faction) return null

                        const colors = colorHexMap[faction.color] || { bg: '#6B7280', border: '#374151' }

                        return (
                            <button
                                key={factionType}
                                data-testid={`faction-option-${factionType}`}
                                onClick={() => { if (isMyTurn && isAvailable) onSelect(factionType); }}
                                disabled={!isMyTurn || !isAvailable}
                                style={{
                                    backgroundColor: colors.bg,
                                    borderColor: colors.border,
                                }}
                                className={`
                                    relative flex flex-col items-center justify-center p-3 rounded-lg transition-all duration-200 border-2
                                    ${isAvailable && isMyTurn
                                        ? 'hover:scale-105 cursor-pointer opacity-100 hover:shadow-lg'
                                        : 'opacity-40 cursor-not-allowed grayscale'}
                                `}
                            >
                                {/* Show player info if selected */}
                                {isSelected && selectionInfo && (
                                    <div className="absolute -top-2 -right-2 bg-yellow-400 border-2 border-yellow-700 rounded-full px-2 py-0.5 text-xs font-bold text-gray-900 shadow-md">
                                        #{selectionInfo.playerNumber} - {selectionInfo.vp} VP
                                    </div>
                                )}

                                {/* Faction name */}
                                <div className="text-center">
                                    <p className="text-sm font-bold text-white drop-shadow-md">
                                        {faction.name}
                                    </p>
                                </div>
                            </button>
                        )
                    })}
                </div>
            </div>
        </div>
    )
}
