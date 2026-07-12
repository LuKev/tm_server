# AI Round 1 Activity and Build Rates

## Purpose

Track whether each model does anything productive before passing in round 1, then how often it builds a Temple/Sanctuary or Stronghold. These are early-game proxies, not promotion gates by themselves. A stronger model should rarely pass without acting, should usually build something, and should learn high round-1 Temple/Stronghold rates for factions that benefit heavily from early priests, favors, or stronghold abilities.

The metrics are captured at the first game state after round 1 ends. For each faction sample:

- `Any build` means at least one map building was added or upgraded after the initial setup state.
- `Passed before action` means the player's first round-1 action was pass.
- `Actions before pass` counts all non-pass AlphaZero actions selected before that player's round-1 pass.
- `Temple/Sanctuary` means the player owns at least one Temple or Sanctuary.
- `Stronghold` means the player owns a Stronghold.
- `Either` means the player owns a Temple, Sanctuary, or Stronghold.

## Expectations

- Giants and Swarmlings should likely approach `95%+` `Either` rate as the engine improves.
- `Any build` should be the first sanity metric reviewed. `Passed before action` should approach `0%` unless a genuine game state makes passing correct.
- Low rates are useful debugging leads, but should be checked against round scoring, faction pairing, seat order, and bonus-card context before treating them as model regressions.
- Compare rates across the same scenario suite where possible. `matrix:base_ordered` is the cleanest cross-model comparison; `training_mix` is better for robustness.

## Model Summary

The five successive July 12 cycles, including per-faction R1 rates and final-VP changes, are recorded in [`terra-mystica-ai-five-cycle-results-20260712.md`](terra-mystica-ai-five-cycle-results-20260712.md).

| Model | Source | Scenario | Games | Sims | Date | Notes |
| --- | --- | --- | ---: | ---: | --- | --- |
| `heuristic_mcts_iter0_selfplay` | `/tmp/tm_az_local_fastlane_20260629/iter0/selfplay_metrics.json` | `matrix:base_ordered` | 168 | 8 | 2026-06-29 | Heuristic MCTS rebuild because retained `/tmp` checkpoints were missing. |
| `h512_iter1_selfplay` | `/tmp/tm_az_local_fastlane_20260629/iter1/selfplay_metrics.json` | `matrix:base_ordered` | 84 | 8 | 2026-06-29 | Neural MCTS self-play from iter0; 4.3x faster records/sec than heuristic rebuild. |
| `h512_iter1_candidate_vs_iter0_arena_84` | `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_84.json` | `matrix:base_ordered` | 84 | 8 | 2026-06-29 | Candidate won 46-38 but did not promote; CI lower bound 44.1% below 45.0% threshold. |
| `h512_iter1_candidate_vs_iter0_arena_168` | `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_168.json` | `matrix:base_ordered` | 168 | 8 | 2026-06-29 | Candidate won 84-80-4 but did not promote; CI lower bound 43.6% below 45.0% threshold. |
| `promoted-h512-selfplay-iter1-20260711_selfplay` | `artifacts/az/models/promoted-h512-selfplay-iter1-20260711/selfplay_metrics.json` | `matrix:base_ordered` | 168 | 8 | 2026-07-11 | Neural-only self-play from recovered 5k incumbent; 336/336 faction-seats built in R1 and 0/336 passed immediately. |
| `promoted-h512-selfplay-iter1-20260711_arena` | `artifacts/az/models/promoted-h512-selfplay-iter1-20260711/arena_168.json` | `matrix:base_ordered` | 168 | 8 | 2026-07-11 | Promoted 90-72-6; score 55.36%, 95% CI 47.84%-62.87%. |

## Per-Faction Rates

### `promoted-h512-selfplay-iter1-20260711_selfplay`

| Faction | Samples | Any build | Passed before action | Avg actions before pass | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Alchemists | 24 | 100% | 0% | 5.17 | 12.5% | 0% | 12.5% |
| Auren | 24 | 100% | 0% | 6.29 | 12.5% | 0% | 12.5% |
| ChaosMagicians | 24 | 100% | 0% | 6.21 | 12.5% | 4.17% | 16.67% |
| Cultists | 24 | 100% | 0% | 6.5 | 20.83% | 4.17% | 25% |
| Darklings | 24 | 100% | 0% | 6.42 | 25% | 0% | 25% |
| Dwarves | 24 | 100% | 0% | 9.83 | 0% | 0% | 0% |
| Engineers | 24 | 100% | 0% | 6.88 | 12.5% | 4.17% | 16.67% |
| Fakirs | 24 | 100% | 0% | 10.46 | 20.83% | 0% | 20.83% |
| Giants | 24 | 100% | 0% | 5.46 | 8.33% | 0% | 8.33% |
| Halflings | 24 | 100% | 0% | 5.71 | 16.67% | 0% | 16.67% |
| Mermaids | 24 | 100% | 0% | 5.54 | 4.17% | 0% | 4.17% |
| Nomads | 24 | 100% | 0% | 5.58 | 20.83% | 0% | 20.83% |
| Swarmlings | 24 | 100% | 0% | 5.79 | 8.33% | 0% | 8.33% |
| Witches | 24 | 100% | 0% | 6.21 | 8.33% | 0% | 8.33% |

### `promoted-h512-selfplay-iter1-20260711_arena` Candidate Seats

| Faction | Samples | Any build | Passed before action | Avg actions before pass | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Alchemists | 6 | 100% | 0% | 6 | 16.67% | 0% | 16.67% |
| Auren | 18 | 100% | 0% | 5.44 | 0% | 0% | 0% |
| ChaosMagicians | 6 | 100% | 0% | 7.67 | 16.67% | 0% | 16.67% |
| Cultists | 18 | 100% | 0% | 6.17 | 16.67% | 0% | 16.67% |
| Darklings | 18 | 100% | 0% | 5.61 | 22.22% | 0% | 22.22% |
| Dwarves | 18 | 100% | 0% | 10.17 | 0% | 0% | 0% |
| Engineers | 6 | 100% | 0% | 6.83 | 16.67% | 0% | 16.67% |
| Fakirs | 18 | 100% | 0% | 9.56 | 5.56% | 0% | 5.56% |
| Giants | 18 | 100% | 0% | 5.22 | 16.67% | 0% | 16.67% |
| Halflings | 6 | 100% | 0% | 5.5 | 0% | 0% | 0% |
| Mermaids | 18 | 100% | 0% | 4.78 | 0% | 0% | 0% |
| Nomads | 6 | 100% | 0% | 5.33 | 0% | 0% | 0% |
| Swarmlings | 6 | 100% | 0% | 6.67 | 33.33% | 0% | 33.33% |
| Witches | 6 | 100% | 0% | 7.33 | 33.33% | 0% | 33.33% |

### `promoted-h512-selfplay-iter1-20260711_arena` Baseline Seats

| Faction | Samples | Any build | Passed before action | Avg actions before pass | Temple/Sanctuary | Stronghold | Either |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Alchemists | 18 | 100% | 0% | 5.83 | 22.22% | 0% | 22.22% |
| Auren | 6 | 100% | 0% | 5.33 | 16.67% | 0% | 16.67% |
| ChaosMagicians | 18 | 100% | 0% | 5.67 | 11.11% | 0% | 11.11% |
| Cultists | 6 | 100% | 0% | 6.67 | 33.33% | 0% | 33.33% |
| Darklings | 6 | 100% | 0% | 6.5 | 16.67% | 0% | 16.67% |
| Dwarves | 6 | 100% | 0% | 10.33 | 16.67% | 0% | 16.67% |
| Engineers | 18 | 100% | 0% | 6.06 | 11.11% | 0% | 11.11% |
| Fakirs | 6 | 100% | 0% | 9.67 | 16.67% | 0% | 16.67% |
| Giants | 6 | 100% | 0% | 5 | 0% | 0% | 0% |
| Halflings | 18 | 100% | 0% | 6.28 | 22.22% | 0% | 22.22% |
| Mermaids | 6 | 100% | 0% | 5.83 | 16.67% | 0% | 16.67% |
| Nomads | 18 | 100% | 0% | 5.61 | 27.78% | 0% | 27.78% |
| Swarmlings | 18 | 100% | 0% | 6.11 | 11.11% | 0% | 11.11% |
| Witches | 18 | 100% | 0% | 5.06 | 11.11% | 0% | 11.11% |

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
