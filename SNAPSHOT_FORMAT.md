# Terra Mystica Snapshot Format

This document describes the concise snapshot notation used to represent the state of a Terra Mystica game. This format is designed to be human-readable, minimal, and capable of fully reconstructing a game position.

## Overview

The snapshot format uses a structured, YAML-like syntax. It is divided into global metadata, player states, map changes, and global game state.

## Structure

### Global Metadata

*   **Round**: Integer (0-6). The current game round.
*   **Phase**: String. Current game phase (e.g., `Action`, `Income`, `CultIncome`, `Setup`).
*   **MapType**: String. The map layout being used (e.g., `base`, `fire_and_ice`).
*   **Turn**: String. The faction name of the player whose turn it currently is.
*   **TurnOrder**: List of strings. The order of factions for the current round.
*   **PassOrder**: List of strings. The order in which factions have passed in the current round.

### Players Section

Each player is identified by their Faction Name.

*   **VP**: Integer. Current Victory Points.
*   **Res**: Resources string.
    *   Format: `[W]w [P]p [C]c / [Bowl1]/[Bowl2]/[Bowl3]`
    *   Example: `2w 1p 5c / 0/4/2` (2 Workers, 1 Priest, 5 Coins, Power bowls 1/2/3)
*   **Keys**: Integer. Number of unused keys.
*   **Shipping**: Integer. Current shipping level.
*   **Digging**: Integer. Current digging level.
*   **Range**: Integer. (Fakirs only) Current carpet flight range.
*   **Cult**: Cult track positions.
    *   Format: `[Fire]/[Water]/[Earth]/[Air]`
    *   Example: `0/0/1/0`
*   **Map**: List of buildings owned by the player.
    *   Format: `[q],[r]:[BuildingCode]`
    *   Building Codes: `D` (Dwelling), `TP` (Trading House), `TE` (Temple), `SH` (Stronghold), `SA` (Sanctuary).
    *   Example: `4,4:D, 5,-2:TP`
*   **Bridges**: List of bridges.
    *   Format: `[q1],[r1]-[q2],[r2]`
    *   Example: `4,4-5,3`
*   **Towns**: List of owned town tiles.
    *   Example: `"Ship town", "8 point town"`
*   **Bonus**: The currently held bonus card.
    *   Format: `"[BonusName]"` or `"[BonusName]" (Used)`
*   **Favor**: List of owned favor tiles.
    *   Format: `[TileCode]` or `[TileCode] (Used)`
    *   Example: `FIRE1, EARTH2`
*   **StrongholdAction**: Status of the stronghold special action.
    *   Values: `Available`, `Used`, `None`

### Map Section

Lists hexes that have been terraformed but do not currently contain a building (or have special tokens).

*   Format: `[q],[r]: [TerrainColor]`
*   Example: `4,5: Green`

### State Section

Tracks global game components.

*   **ScoringTiles**: Ordered list of scoring tiles for rounds 1-6.
*   **Bonuses**: Map of available bonus cards and the coins accumulated on them.
    *   Format: `"[BonusName]": [Coins]`
*   **Favors**: Map of available favor tiles and their remaining count.
    *   Format: `"[TileName]": [Count]`
*   **Towns**: Map of available town tiles and their remaining count.
    *   Format: `"[TownName]": [Count]`
*   **PowerActions**: Status of the main board power actions.
    *   Format: `"[ActionName]": [Status]`
    *   Status: `Available` or `Used`
*   **CultBoard**: Lists players occupying the priest spots on each cult track.
    *   Format: `[TrackName]: [[Faction], ...]`

## Example Snapshot

```yaml
Round: 2
Phase: Action
MapType: base
Turn: Witches
TurnOrder: [Witches, Nomads]
PassOrder: []

Players:
  Witches:
    VP: 20
    Res: 2w 1p 5c / 0/4/2
    Keys: 1
    Shipping: 1
    Digging: 0
    Cult: 0/0/1/0
    Map: 4,4:D, 5,-2:TP, 6,-3:SH
    Bridges: 4,4-5,3
    Towns: "Ship town"
    Bonus: "spade" (Used)
    Favor: FIRE1, EARTH2
    StrongholdAction: Available
  
  Nomads:
    VP: 15
    Res: 1w 2p 8c / 2/2/8
    Keys: 0
    Shipping: 0
    Digging: 1
    Cult: 1/0/0/0
    Map: 3,5:D, 7,-1:TE
    Towns: []
    Bonus: "6 coins"
    Favor: WATER1
    StrongholdAction: None

Map:
  4,5: Green

State:
  ScoringTiles: "SCORE1", "SCORE2", "SCORE3", "SCORE4", "SCORE5", "SCORE6"
  Bonuses:
    "cult coins": 1
    "temp ship": 0
  Favors:
    "Fire +3": 1
    "Water +1: Trading House VP": 3
  Towns:
    "5 VP, 6 Coins": 2
    "6 VP, 8 Power": 2
  PowerActions:
    "Bridge": Available
    "7 Coins": Used
  CultBoard:
    Fire: [Nomads]
    Water: []
    Earth: [Witches]
    Air: []
```
