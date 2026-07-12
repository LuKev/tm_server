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

## Direct Iteration 5 vs Iteration 1 Check

A separate 168-game `matrix:base_ordered` arena compared the final promoted iteration 5 directly with iteration 1 at 8 simulations:

- Iteration 5: 86 wins.
- Iteration 1: 81 wins.
- Draws: 1.
- Iteration 5 score: 51.49%, 95% CI `[43.93%, 59.05%]`.
- Average final VP: 82.84 for iteration 5 versus 82.58 for iteration 1, a `+0.26 VP` difference.

This direct comparison does not establish a meaningful improvement over iteration 1. The successive one-step promotion results did not compound into a clear end-to-end gain, indicating that the current 168-game gate is vulnerable to noise, non-transitive policies, or repeated fine-tuning drift.

The old arena schedule was subsequently found to be unpaired: candidate ownership alternated by game index while ordered faction matchups were generated in faction-pool order. Candidate faction exposure varied from 6 to 18 games per faction, making the gate unnecessarily sensitive to faction and seat interactions.

A corrected 168-game `matrix:base_paired` arena uses 84 legal faction pairs twice each, with candidate and incumbent swapping ownership while faction seats stay fixed. Under this corrected gate, iteration 5 scored **90-74-4** against iteration 1: **54.76%**, 95% CI `[47.24%, 62.29%]`, and `+1.79 VP`. This supports a modest improvement, but not a marked one, and it remains just below the 55% point threshold.
