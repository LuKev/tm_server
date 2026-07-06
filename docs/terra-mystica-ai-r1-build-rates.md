# AI R1 Temple / Stronghold Rates

## Purpose

Track how often each model builds a Temple/Sanctuary or Stronghold by the end of round 1. This is an early-game proxy, not a promotion gate by itself. A stronger model should usually learn high round-1 Temple/Stronghold rates for factions that benefit heavily from early priests, favors, or stronghold abilities.

The metric is captured at the first game state after round 1 ends. For each faction sample:

- `Temple/Sanctuary` means the player owns at least one Temple or Sanctuary.
- `Stronghold` means the player owns a Stronghold.
- `Either` means the player owns a Temple, Sanctuary, or Stronghold.

## Expectations

- Giants and Swarmlings should likely approach `95%+` `Either` rate as the engine improves.
- Low rates are useful debugging leads, but should be checked against round scoring, faction pairing, seat order, and bonus-card context before treating them as model regressions.
- Compare rates across the same scenario suite where possible. `matrix:base_ordered` is the cleanest cross-model comparison; `training_mix` is better for robustness.

## Model Summary

| Model | Source | Scenario | Games | Sims | Date | Notes |
| --- | --- | --- | ---: | ---: | --- | --- |
| `heuristic_mcts_iter0_selfplay` | `/tmp/tm_az_local_fastlane_20260629/iter0/selfplay_metrics.json` | `matrix:base_ordered` | 168 | 8 | 2026-06-29 | Heuristic MCTS rebuild because retained `/tmp` checkpoints were missing. |
| `h512_iter1_selfplay` | `/tmp/tm_az_local_fastlane_20260629/iter1/selfplay_metrics.json` | `matrix:base_ordered` | 84 | 8 | 2026-06-29 | Neural MCTS self-play from iter0; 4.3x faster records/sec than heuristic rebuild. |
| `h512_iter1_candidate_vs_iter0_arena_84` | `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_84.json` | `matrix:base_ordered` | 84 | 8 | 2026-06-29 | Candidate won 46-38 but did not promote; CI lower bound 44.1% below 45.0% threshold. |
| `h512_iter1_candidate_vs_iter0_arena_168` | `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_168.json` | `matrix:base_ordered` | 168 | 8 | 2026-06-29 | Candidate won 84-80-4 but did not promote; CI lower bound 43.6% below 45.0% threshold. |

## Per-Faction Rates

### `heuristic_mcts_iter0_selfplay`

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 24 | 16.7% | 4.2% | 20.8% |
| Auren | 24 | 16.7% | 0.0% | 16.7% |
| Chaos Magicians | 24 | 4.2% | 0.0% | 4.2% |
| Cultists | 24 | 16.7% | 4.2% | 20.8% |
| Darklings | 24 | 4.2% | 8.3% | 12.5% |
| Dwarves | 24 | 0.0% | 4.2% | 4.2% |
| Engineers | 24 | 12.5% | 4.2% | 16.7% |
| Fakirs | 24 | 4.2% | 4.2% | 8.3% |
| Giants | 24 | 4.2% | 4.2% | 8.3% |
| Halflings | 24 | 8.3% | 4.2% | 12.5% |
| Mermaids | 24 | 8.3% | 4.2% | 12.5% |
| Nomads | 24 | 4.2% | 4.2% | 8.3% |
| Swarmlings | 24 | 4.2% | 4.2% | 8.3% |
| Witches | 24 | 4.2% | 0.0% | 4.2% |

### `h512_iter1_selfplay`

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 7 | 14.3% | 0.0% | 14.3% |
| Auren | 6 | 33.3% | 0.0% | 33.3% |
| Chaos Magicians | 17 | 11.8% | 0.0% | 11.8% |
| Cultists | 7 | 0.0% | 0.0% | 0.0% |
| Darklings | 7 | 42.9% | 0.0% | 42.9% |
| Dwarves | 7 | 0.0% | 0.0% | 0.0% |
| Engineers | 7 | 42.9% | 0.0% | 42.9% |
| Fakirs | 17 | 5.9% | 0.0% | 5.9% |
| Giants | 17 | 0.0% | 0.0% | 0.0% |
| Halflings | 7 | 0.0% | 0.0% | 0.0% |
| Mermaids | 17 | 0.0% | 0.0% | 0.0% |
| Nomads | 17 | 11.8% | 17.6% | 29.4% |
| Swarmlings | 17 | 5.9% | 5.9% | 11.8% |
| Witches | 18 | 0.0% | 0.0% | 0.0% |

### `h512_iter1_candidate_vs_iter0_arena_84` Candidate Seats

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 0 | n/a | n/a | n/a |
| Auren | 6 | 33.3% | 0.0% | 33.3% |
| Chaos Magicians | 6 | 16.7% | 0.0% | 16.7% |
| Cultists | 7 | 0.0% | 14.3% | 14.3% |
| Darklings | 7 | 0.0% | 14.3% | 14.3% |
| Dwarves | 7 | 0.0% | 0.0% | 0.0% |
| Engineers | 0 | n/a | n/a | n/a |
| Fakirs | 11 | 0.0% | 27.3% | 27.3% |
| Giants | 11 | 27.3% | 0.0% | 27.3% |
| Halflings | 0 | n/a | n/a | n/a |
| Mermaids | 11 | 0.0% | 0.0% | 0.0% |
| Nomads | 6 | 16.7% | 0.0% | 16.7% |
| Swarmlings | 6 | 50.0% | 16.7% | 66.7% |
| Witches | 6 | 50.0% | 0.0% | 50.0% |

### `h512_iter1_candidate_vs_iter0_arena_84` Baseline Seats

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 7 | 14.3% | 0.0% | 14.3% |
| Auren | 0 | n/a | n/a | n/a |
| Chaos Magicians | 11 | 0.0% | 0.0% | 0.0% |
| Cultists | 0 | n/a | n/a | n/a |
| Darklings | 0 | n/a | n/a | n/a |
| Dwarves | 0 | n/a | n/a | n/a |
| Engineers | 7 | 28.6% | 0.0% | 28.6% |
| Fakirs | 6 | 33.3% | 16.7% | 50.0% |
| Giants | 6 | 0.0% | 0.0% | 0.0% |
| Halflings | 7 | 0.0% | 0.0% | 0.0% |
| Mermaids | 6 | 16.7% | 0.0% | 16.7% |
| Nomads | 11 | 18.2% | 27.3% | 45.5% |
| Swarmlings | 11 | 9.1% | 9.1% | 18.2% |
| Witches | 12 | 8.3% | 8.3% | 16.7% |

### `h512_iter1_candidate_vs_iter0_arena_168` Candidate Seats

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 6 | 0.0% | 0.0% | 0.0% |
| Auren | 18 | 11.1% | 5.6% | 16.7% |
| Chaos Magicians | 6 | 33.3% | 0.0% | 33.3% |
| Cultists | 18 | 16.7% | 16.7% | 33.3% |
| Darklings | 18 | 16.7% | 22.2% | 38.9% |
| Dwarves | 18 | 16.7% | 5.6% | 22.2% |
| Engineers | 6 | 16.7% | 16.7% | 33.3% |
| Fakirs | 18 | 22.2% | 5.6% | 27.8% |
| Giants | 18 | 16.7% | 11.1% | 27.8% |
| Halflings | 6 | 16.7% | 0.0% | 16.7% |
| Mermaids | 18 | 0.0% | 0.0% | 0.0% |
| Nomads | 6 | 16.7% | 16.7% | 33.3% |
| Swarmlings | 6 | 0.0% | 0.0% | 0.0% |
| Witches | 6 | 0.0% | 33.3% | 33.3% |

### `h512_iter1_candidate_vs_iter0_arena_168` Baseline Seats

| Faction | Samples | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: |
| Alchemists | 18 | 5.6% | 16.7% | 16.7% |
| Auren | 6 | 16.7% | 33.3% | 50.0% |
| Chaos Magicians | 18 | 27.8% | 5.6% | 27.8% |
| Cultists | 6 | 16.7% | 16.7% | 33.3% |
| Darklings | 6 | 66.7% | 16.7% | 83.3% |
| Dwarves | 6 | 0.0% | 0.0% | 0.0% |
| Engineers | 18 | 16.7% | 33.3% | 44.4% |
| Fakirs | 6 | 0.0% | 16.7% | 16.7% |
| Giants | 6 | 16.7% | 16.7% | 33.3% |
| Halflings | 18 | 16.7% | 0.0% | 16.7% |
| Mermaids | 6 | 0.0% | 0.0% | 0.0% |
| Nomads | 18 | 11.1% | 11.1% | 22.2% |
| Swarmlings | 18 | 16.7% | 16.7% | 33.3% |
| Witches | 18 | 22.2% | 16.7% | 38.9% |
