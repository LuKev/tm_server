# AlphaZero Five-Cycle Results - 2026-07-12

## Method

Each cycle used only neural self-play from the current promoted checkpoint:

- 168 `matrix:base_ordered` self-play games at 8 MCTS simulations.
- Five-epoch h512 hex-network transfer training with the incumbent vocabulary preserved.
- A separate 168-game candidate-versus-incumbent promotion arena at 8 simulations.
- Promotion required score `>=55%`, at least 168 games, and 95% CI lower bound `>=45%`.

Iterations 2 through 5 promoted. Iteration 6 did not, so iteration 5 remains the incumbent.

## Score Results

`VP change` is candidate average final VP minus incumbent average final VP in the promotion arena.

| Iteration | Promoted | Arena | Score | 95% CI | Candidate avg VP | Incumbent avg VP | VP change | Self-play avg VP |
| ---: | --- | --- | ---: | --- | ---: | ---: | ---: | ---: |
| 2 | Yes | 91-71-6 | 55.95% | 48.45%-63.46% | 84.73 | 82.09 | +2.64 | 82.68 |
| 3 | Yes | 97-70-1 | 58.04% | 50.57%-65.50% | 84.25 | 79.52 | +4.73 | 82.80 |
| 4 | Yes | 93-73-2 | 55.95% | 48.45%-63.46% | 84.30 | 81.42 | +2.88 | 83.45 |
| 5 | Yes | 90-73-5 | 55.06% | 47.54%-62.58% | 82.83 | 82.94 | -0.11 | 82.43 |
| 6 | No | 83-81-4 | 50.60% | 43.03%-58.16% | 82.37 | 81.96 | +0.41 | 82.48 |

## Round 1 Temple / Stronghold Rates

Each cell is `Temple or Sanctuary / Stronghold`; every faction has 24 self-play samples per iteration.

| Faction | Iteration 2 | Iteration 3 | Iteration 4 | Iteration 5 | Iteration 6 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Alchemists | 33.3% / 0% | 20.8% / 0% | 8.3% / 0% | 20.8% / 0% | 16.7% / 0% |
| Auren | 8.3% / 0% | 8.3% / 0% | 12.5% / 0% | 4.2% / 0% | 12.5% / 0% |
| Chaos Magicians | 0% / 0% | 20.8% / 0% | 25.0% / 0% | 12.5% / 0% | 8.3% / 0% |
| Cultists | 8.3% / 0% | 16.7% / 0% | 16.7% / 0% | 29.2% / 0% | 29.2% / 0% |
| Darklings | 0% / 0% | 0% / 0% | 20.8% / 0% | 4.2% / 0% | 16.7% / 0% |
| Dwarves | 4.2% / 0% | 0% / 0% | 0% / 0% | 0% / 0% | 0% / 0% |
| Engineers | 25.0% / 0% | 20.8% / 0% | 20.8% / 0% | 33.3% / 0% | 29.2% / 0% |
| Fakirs | 12.5% / 0% | 8.3% / 0% | 16.7% / 0% | 12.5% / 0% | 4.2% / 0% |
| Giants | 4.2% / 0% | 8.3% / 0% | 8.3% / 0% | 20.8% / 0% | 8.3% / 0% |
| Halflings | 25.0% / 0% | 20.8% / 0% | 8.3% / 0% | 29.2% / 0% | 8.3% / 0% |
| Mermaids | 12.5% / 0% | 0% / 0% | 4.2% / 0% | 4.2% / 0% | 8.3% / 0% |
| Nomads | 16.7% / 0% | 8.3% / 0% | 8.3% / 0% | 16.7% / 0% | 25.0% / 0% |
| Swarmlings | 4.2% / 0% | 16.7% / 0% | 8.3% / 0% | 8.3% / 0% | 0% / 0% |
| Witches | 4.2% / 0% | 8.3% / 0% | 20.8% / 0% | 4.2% / 0% | 4.2% / 0% |

## Interpretation

- Relative playing strength improved enough to promote four consecutive candidates.
- Absolute VP did not improve monotonically. Iteration 5 won the arena gate despite a `-0.11` average VP change.
- Round-1 Temple rates remain low and noisy. Iteration 5 Giants improved to `20.8%`, but Swarmlings remained `8.3%`.
- No faction built a round-1 Stronghold in any of the 1,680 tracked player-seats across these five cycles. The current training scale is not teaching the desired opening priorities.
