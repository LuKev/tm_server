# Faction Implementation Guide

This guide explains how to implement each of the 14 factions in Terra Mystica.

## Faction Structure

Each faction should:
1. Embed `BaseFaction` for default behavior
2. Override specific methods for unique abilities
3. Implement special actions and stronghold abilities
4. Define starting resources and costs

## Common Patterns

### Starting Resources
Most factions start with:
- 15 coins
- 3 workers  
- 0 priests
- 5 power in bowl 1
- 7 power in bowl 2
- 0 power in bowl 3

Some factions have variations (e.g., Chaos Magicians start with 4 workers).

### Building Costs
Standard costs are defined in `faction.go`:
- **Dwelling**: 1 worker
- **Trading House**: 6 coins, 2 workers
- **Temple**: 5 coins, 2 workers
- **Sanctuary**: 8 coins, 4 workers
- **Stronghold**: 6 coins, 4 workers

Individual factions may have different building costs.

### Terraform Costs
Base: 3 workers per spade, reduced by digging level.
- Digging 0: 3 workers/spade
- Digging 1: 2 workers/spade
- Digging 2: 1 worker/spade

Darklings use priests instead of workers for digging.

## Faction List

### Yellow (Desert)
1. **Nomads** - Sandstorm ability (place dwelling on any desert hex)
2. **Fakirs** - Carpet flying (place dwelling on any desert hex, ignoring adjacency)

### Red (Wasteland)
3. **Chaos Magicians** - Start with a single dwelling. Temples and Sanctuaries give two favor tiles instead of one.
4. **Giants** - Reduced terraform costs (2 workers per spade base)

### Blue (Lake)
5. **Swarmlings** - More expensive buildings, but stronghold gives free TH upgrade. Towns give 3 workers on formation.
6. **Mermaids** - Increased shipping. Towns can cross water hexes.

### Green (Forest)
7. **Witches** - Flying (can place buildings ignoring adjacency). Towns provide 5 additional VP.
8. **Auren** - Stronghold provides cult steps.

### Brown (Plains)
9. **Halflings** - Spade efficiency and +1 VP per spade.
10. **Cultists** - Go up on the cults when opponents accept leech.

### Black (Swamp)
11. **Alchemists** - Can arbitrarily trade VP for coins.
12. **Darklings** - Dig with priests instead of workers.

### Gray (Mountain)
13. **Engineers** - Cheaper building costs
14. **Dwarves** - Tunneling ability (can treat certain hexes as adjacent)

## Implementation Template

```go
package factions

import "github.com/lukev/tm_server/internal/models"

type FactionName struct {
	BaseFaction
}

func NewFactionName() *FactionName {
	return &FactionName{
		BaseFaction: BaseFaction{
			Type:        models.FactionFactionName,
			HomeTerrain: models.TerrainType,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  5,
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
	}
}

// Override methods as needed for special abilities

func (f *FactionName) HasSpecialAbility(ability SpecialAbility) bool {
	// Return true for abilities this faction has
	return false
}

func (f *FactionName) GetStrongholdAbility() string {
	return "Description of stronghold ability"
}

func (f *FactionName) ExecuteStrongholdAbility(gameState interface{}) error {
	// Implement stronghold ability logic
	return nil
}
```

## Testing

Each faction should have tests covering:
1. Starting resources
2. Building costs (if modified)
3. Special abilities
4. Stronghold ability
5. Any unique mechanics

## Next Steps

We'll implement each faction one at a time, starting with the simpler ones and working up to more complex abilities.
