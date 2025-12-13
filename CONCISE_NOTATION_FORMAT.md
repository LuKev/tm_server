# Concise Game Notation Format

This document describes the concise, grid-based notation used for recording Terra Mystica game logs. The format is designed to be human-readable, compact, and easy to parse.

## Structure

The log consists of a **Header** section defining the game setup, followed by a series of **Rounds**.

### Header

The header contains key-value pairs describing the game configuration.

```text
Game: [MapName]
ScoringTiles: [Tile1], [Tile2], ...
BonusCards: [Card1], [Card2], ...
Options: [Option1], ...
```

### Rounds

Each round starts with a round header and a turn order definition, followed by a grid of actions.

```text
Round [N]
TurnOrder: [Faction1], [Faction2], ...
----------------------------------------------------------------
[Faction1]   | [Faction2]   | [Faction3]   
----------------------------------------------------------------
[Action]     | [Action]     | [Action]     
...
```

*   **Grid Layout**: Columns correspond to the factions in the `TurnOrder`.
*   **Timeline**: Actions are listed chronologically.
*   **Reactions**: Reactions (like Leeching) are recorded in the column of the reacting player, on the same row (or next available row) as the triggering action if possible, or immediately following.

## Action Codes

Actions are represented by short, uppercase codes. Parameters are appended with hyphens.

### Building & Upgrading

*   **Build Dwelling**: `[Coord]`
    *   Example: `C4`, `F5`
    *   *Note: Implicitly means "Build Dwelling" if the hex is empty.*
*   **Upgrade**: `[Building]-[Coord]`
    *   `TP`: Trading House
    *   `TE`: Temple
    *   `SH`: Stronghold
    *   `SA`: Sanctuary
    *   Example: `TP-C4`, `SH-F5`

### Terraforming

*   **Dig & Build**: `[Spades]-[Coord]`
    *   `D`: 1 Spade
    *   `DD`: 2 Spades
    *   `DDD`: 3 Spades
    *   Example: `D-C4` (Dig 1 spade and build), `DD-F5`
*   **Transform Only**: `[Spades]-[Coord]-T`
    *   Example: `D-C4-T` (Dig 1 spade, do not build)
*   **Bonus Spades**: `ACTS-[Coord]`
    *   Used for spades from Power Actions 5/6 or Bonus Cards.

### Power Actions

*   **Standard Actions**: `ACT[N]`
    *   `ACT1`: Bridge
    *   `ACT2`: Priest
    *   `ACT3`: Workers
    *   `ACT4`: Coins
    *   `ACT5`: Spade
    *   `ACT6`: 2 Spades
*   **With Targets**:
    *   Bridge: `ACT1-[From]-[To]` (e.g., `ACT1-C4-C5`)
    *   Spade: `ACT5-[Coord]` (e.g., `ACT5-C4`)

### Special Actions

*   **Faction Actions**: `ACT-[Code]`
    *   **Witches**: `ACT-SH-D-[Coord]` (Free Dwelling)
    *   **Nomads**: `ACT-SH-D-[Coord]` (Sandstorm)
    *   **Giants**: `ACT-SH-S-[Coord]` (Free Spade)
    *   **Swarmlings**: `ACT-SH-TP-[Coord]` (Free Upgrade to TP)
    *   **Chaos Magicians**: `ACT-SH-2X` (Double Turn)
    *   **Engineers**: `ACT-BR-[Coord]-[Coord]` (Bridge for 2 workers)
    *   **Mermaids**: `ACT-TOWN-[Coord]` (Form town skipping river)
*   **Darklings**: `ORD-[N]` (Priest Ordination, e.g., `ORD-3`)

### Cult & Resources

*   **Send Priest**: `->[Track]`
    *   Tracks: `F` (Fire), `W` (Water), `E` (Earth), `A` (Air)
    *   Example: `->F`
*   **Conversions**: `C[In]:[Out]`
    *   Example: `C3PW:1W` (3 Power to 1 Worker), `C1P:1W`
*   **Burn Power**: `B[N]`
    *   Example: `B3` (Burn 3 power)

### Passing

*   **Pass**: `Pass-[BonusTile]`
    *   Example: `Pass-BON1`

### Reactions

*   **Leech**: `L` (Leech) or `DL` (Decline Leech)
*   **Cultist Reaction**: `CULT-[Track]`
    *   Example: `CULT-F` (Advance on Fire track due to opponent leeching)

### Other

*   **Advancement**: `+[Track]`
    *   `+SHIP`: Advance Shipping
    *   `+DIG`: Advance Digging

## Example Log

```text
Game: Base
ScoringTiles: SCORE1, SCORE2
BonusCards: BON1, BON2
Options: OPT1

Round 1
TurnOrder: Cultists, Nomads, Witches
----------------------------------------------------------------
Cultists     | Nomads       | Witches      
----------------------------------------------------------------
C4           | L            | DL           
CULT-F       |              |              
             | D-F5         |              
             | ACT4         |              
             |              | C7           
```

In this example:
1.  **Cultists** build at `C4`.
2.  **Nomads** leech (`L`).
3.  **Witches** decline leech (`DL`).
4.  **Cultists** gain a cult step (`CULT-F`) because Nomads leeched.
5.  **Nomads** dig and build at `F5`.
6.  **Nomads** take Power Action 4 (Coins).
7.  **Witches** build at `C7`.
