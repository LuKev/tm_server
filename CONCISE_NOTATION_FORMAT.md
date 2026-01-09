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
StartingVPs: [Faction1]:[VP], [Faction2]:[VP], ...
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

### Setup

*   **Setup Dwelling**: `S-[Coord]`
    *   Example: `S-F3`, `S-C4`
    *   *Used during the initial dwelling placement phase.*
*   **Select Bonus Card**: `BON-[Code]`
    *   Example: `BON-SPD`, `BON-BB`
    *   *Used at the end of the setup phase.*

### Building & Upgrading

*   **Build Dwelling**: `[Coord]`
    *   Example: `C4`, `F5`
    *   *Note: Implicitly means "Transform & Build Dwelling" (usually with 0 spades if already home terrain).*
*   **Upgrade**: `UP-[Building]-[Coord]`
    *   `D`: Dwelling (rare, usually initial)
    *   `TH`: Trading House
    *   `TE`: Temple
    *   `SH`: Stronghold
    *   `SA`: Sanctuary
    *   Example: `UP-TH-C4`, `UP-SH-F5`

### Terraforming

*   **Transform Only**: `T-[Coord]` or `T-[Coord]-[Color]`
    *   **Colors**: `Br` (Brown/Plains), `Bk` (Black/Swamp), `Bl` (Blue/Lake), `G` (Green/Forest), `Gy` (Gray/Mountain), `R` (Red/Wasteland), `Y` (Yellow/Desert)
    *   **Logic**: Omit color if transforming to faction's home terrain.
    *   Example: `T-C4` (Transform to home terrain)
    *   Example: `T-C4-Y` (Transform C4 to Yellow/Desert)
    *   Example: `T-D5-Bk` (Transform D5 to Black/Swamp)
*   **Dig & Build**: `[Coord]` (same as Build Dwelling, context implies digging if needed)
    *   *Note: The notation simplifies Transform & Build into just the target coordinate if a dwelling is built.*
*   **Bonus Spades**: `ACTS-[Coord]`
    *   Used for spades from Power Actions 5/6 or Bonus Cards.

### Power Actions

### Power Actions

*   **Standard Actions**: `ACT[N]`
    *   `ACT1`: Bridge
        *   Format: `ACT1` or `ACT1-<Hex1>-<Hex2>`
        *   Example: `ACT1-C2-D4`
    *   `ACT2`: Priest
    *   `ACT3`: Workers
    *   `ACT4`: Coins
    *   `ACT5`: Spade
    *   `ACT6`: 2 Spades
    *   Example: `ACT4`, `ACT6`

### Special Actions

*   **Faction Actions**: `ACT-[Code]`
    *   **Witches**: `ACT-SH-D-[Coord]` (Free Dwelling)
    *   **Auren**: `ACT-SH-[Track]` (Advance 2 steps on track)
        *   Example: `ACT-SH-W` (Advance 2 on Water)
    *   **Nomads**: `ACT-SH-T-[Coord]` (Sandstorm Transform) or `ACT-SH-T-[Coord].[coord]` (Sandstorm + Build Dwelling)
    *   **Giants**: `ACT-SH-S-[Coord]` (Free Spade)
    *   **Swarmlings**: `ACT-SH-TP-[Coord]` (Free Upgrade to TP)
    *   **Chaos Magicians**: `ACT-SH-2X` (Double Turn)
    *   **Engineers**: `ACT-BR-[Coord]-[Coord]` (Bridge for 2 workers)
    *   **Mermaids**: `ACT-TOWN-[Coord]` (Form town skipping river)
*   **Favor Tile Action**: `ACT-FAV-[Track]`
    *   Example: `ACT-FAV-E` (Advance 1 on Earth from FAV11)
*   **Bonus Card Spade**: `ACTS-[Coord]`
    *   Example: `ACTS-C2`
*   **Bonus Card Cult**: `ACT-BON-[Track]`
    *   Example: `ACT-BON-F` (Advance 1 on Fire from Bonus Card)
*   **Darklings**: `ORD-[N]` (Priest Ordination, e.g., `ORD-3`)

### Town Formation
*   **Town**: `TW[VP]VP`
    *   Format: `TW` followed by the VP value of the town.
    *   `TW5VP`: 5 VP, 6 Coins
    *   `TW6VP`: 6 VP, 8 Power
    *   `TW7VP`: 7 VP, 2 Workers
    *   `TW4VP`: 4 VP, Shipping
    *   `TW8VP`: 8 VP, 1 Cult step
    *   `TW9VP`: 9 VP, 1 Priest
    *   `TW11VP`: 11 VP
    *   `TW2VP`: 2 VP, 2 Cult steps
    *   Example: `UP-TH-G7.TW8VP` (Upgrade to Trading House and form 8VP town)

### Conversions
Conversions are auxiliary actions chained to other actions.
Format: `C[Cost]:[Reward]`
*   **Resources**: `P` (Priest), `W` (Worker), `PW` (Power), `VP` (Victory Point), `C` (Coin).
*   **Order**: Resources are always listed in this order: `P`, `W`, `PW`, `VP`, `C`.
*   **Examples**:
    *   `C3PW:1W` (Convert 3 Power to 1 Worker)
    *   `C1P:1W` (Convert 1 Priest to 1 Worker)
    *   `C1VP:1C` (Alchemists VP to Coin)
    *   `C1PW:1C.ACT-SH-F.C1PW:1C` (Conversions before and after special action)

### Cult & Resources

*   **Send Priest**: `->[Track]` or `->[Track][Spot]`
    *   Tracks: `F` (Fire), `W` (Water), `E` (Earth), `A` (Air)
    *   Example: `->F` (Implicit spot)
    *   Example: `->E3` (Send priest to Earth track spot 3)
*   **Favor Tiles**: `FAV-[Track][Amount]`
    *   `FAV-F1`: Fire 1 (+3 Coins)
    *   `FAV-W2`: Water 2 (Cult Action)
    *   `FAV-E3`: Earth 3
    *   Example: `FAV-F1`, `FAV-A2`

### Conversions/Burning Power

*   **Conversions**: `C[In]:[Out]`
    *   Example: `C3PW:1W` (3 Power to 1 Worker), `C1P:1W`
*   **Burn Power**: `BURN[N]`
    *   Example: `BURN3` (Burn 3 power)
*   Conversions and burns can not be their own action, they are always combined with other actions.

### Passing

*   **Pass**: `PASS` or `PASS-[BonusCard]`
    *   Example: `PASS-BON-SPD`
    *   *Note: Bonus card selection is required for Rounds 1-5.*

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
BonusCards: BON-SPD, BON-4C
StartingVPs: Cultists:20, Nomads:20, Witches:20
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
