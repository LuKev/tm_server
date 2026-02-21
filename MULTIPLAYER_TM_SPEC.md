# Terra Mystica Multiplayer Spec (Locked V2)

## 1. Purpose
Build a real-time multiplayer mode (2-5 players) on top of the existing replay viewer + rules engine so players can play full base-game Terra Mystica with authoritative server rules and interactive UI.

This spec is implementation-oriented and scoped to current repository structure:
- Backend engine: `server/internal/game/*`
- Multiplayer transport: `server/internal/websocket/*`
- Frontend board/UI: `client/src/components/*`

## 2. Product Scope
### 2.1 In Scope
1. Lobby -> start game -> full playthrough with up to 5 players.
2. Snellman-style faction pick (no auction), all at 20 VP.
3. Full setup flow (faction, dwellings, bonus cards).
4. Full round/action/reaction gameplay including leech, towns, favor, cult, power/special actions.
5. Pre-commit action confirmation for irreversible actions; no post-confirm undo in v1.
6. Reconnect-safe multiplayer state sync via websocket.

### 2.2 Out of Scope for First Multiplayer Iteration
1. BGA auction/bidding setup mode.
2. Fire & Ice factions.
3. Landscapes expansion.
4. AI players.
5. Matchmaking/ranked ladders.
6. Spectator mode.
7. Turn timers.
8. Mid-game resign/drop handling.
9. Deferred items are tracked in `MULTIPLAYER_TODO_LATER.md`.

### 2.3 Defaults and Config
1. Setup turn order default is randomized, host-configurable.
2. Map locked to Base Game for v1.
3. Mini-expansion options `shipping-bonus` and `temple-scoring-tile` are enabled by default.

## 3. Current State (Repository Reality)
### 3.1 Reusable Existing Assets
1. Core rules/actions already implemented for most base-game behavior in `server/internal/game/*`.
2. Replay parsers cover broad action vocabulary from Snellman + BGA logs in `server/internal/notation/*`.
3. Frontend already renders board, buildings, bridges, cult tracks, favor/town/bonus tiles, player boards.

### 3.2 Critical Gaps
1. Websocket `perform_action` only handles `select_faction` and `setup_dwelling`.
2. Client action service only submits those two actions.
3. No per-game websocket room filtering; broadcasts go to all clients.
4. Server state serialization lacks pending choice state needed for modals (leech, favor, towns, spades, etc.).
5. Setup dwelling logic does not enforce true TM setup order/special faction setup constraints.
6. Turn/seat authority is weak (player identity comes from client payload, not authenticated seat binding).
7. No pre-commit confirmation model for irreversible actions.

## 4. Rule Coverage Model
Use existing engine as base and add missing/strict behavior so live play == replay correctness.

### 4.1 Mandatory Action Families
1. Faction selection.
2. Setup dwelling placement.
3. Setup bonus tile selection.
4. Transform only / transform+build / build on home terrain.
5. Building upgrades (D->TP, TP->TE/SH, TE->SA).
6. Shipping and digging upgrades.
7. Priest send to cult (3/2/1 spots, including reclaim/1-step).
8. Power actions (bridge, priest, workers, coins, spade, double spade with split/throwaway second spade).
9. Special actions (SH, favor, bonus-card, faction-specific).
10. Conversions and burning power as free actions.
11. Leech accept/decline (with Cultists follow-up selection).
12. Favor tile selection.
13. Town tile selection (including cult-top disambiguation choice when keys are limiting).
14. Cult reward spades (transform only, no build).
15. Pass with bonus tile claim and pass-order effects.
16. Action confirmation with no post-confirm undo in v1.

### 4.2 Additional Action Types Seen in Existing Logs and Required for Full Multiplayer Parity
1. `BURNn` power burns.
2. Compound conversions in any legal pre/post order during a turn.
3. Reclaim-priest style cult send (`->Track1`).
4. Chaos Magicians double-turn orchestration.
5. Explicit “decline optional dwelling after spades” flow (Halflings / bonus spade contexts).
6. Engineers SH bridge action.
7. Mermaids river-town connect action.

## 5. Authoritative State Machine
## 5.1 Top-Level Phase + Subphase
Keep existing `GamePhase` enum but add explicit `subphase` and `pendingDecision` payload.

- `phase=faction_selection`
- `phase=setup`
- `phase=income`
- `phase=action`
- `phase=cleanup`
- `phase=end`

Add:
- `subphase`: finer-grained step (e.g., `setup_dwelling_forward`, `setup_dwelling_reverse`, `setup_bonus_pick`).
- `pendingDecision`: union object for modal-required choices.

### 5.2 Pending Decision Union
`pendingDecision.type` values:
1. `leech_offer`
2. `favor_tile_selection`
3. `town_tile_selection`
4. `town_cult_top_choice`
5. `spade_followup`
6. `halflings_spades`
7. `halflings_optional_dwelling`
8. `darklings_ordination`
9. `cultists_cult_choice`
10. `bridge_placement`
11. `pass_confirm_unspent_specials`
12. `upgrade_branch_choice` (TP->TE vs SH)

Each pending decision includes:
- `playerId`
- `sourceActionId`
- decision-specific legal options
- blocking semantics (`blocksTurnAdvance=true/false`)

## 6. Multiplayer Transport & Protocol
## 6.1 Websocket Rooms
Replace global broadcast-only flow with game rooms.

Server requirements:
1. Connection tracks joined game ids.
2. `game_state_update` only broadcast to clients subscribed to that game.
3. Lobby updates remain global.

## 6.2 Action Envelope (Single Canonical Contract)
`perform_action.payload`:
- `gameId: string`
- `actionId: string` (client-generated UUID for idempotency)
- `expectedRevision: number` (optimistic concurrency)
- `seatId: string` (resolved from authenticated session; server validates ownership)
- `type: string`
- `params: object`

Server responses:
1. `action_accepted` (`actionId`, `newRevision`).
2. `action_rejected` (`actionId`, error code, user message, optional legal alternatives).
3. `game_state_update` (full or delta; full first pass).
4. `decision_required` (if pending decision is created for current actor).

## 6.3 Security/Authority Rules
1. Ignore client-supplied `playerID` for trust; use connection seat mapping.
2. Reject action if caller is not the required actor for the current pending decision/turn.
3. Reject stale `expectedRevision` with resync message.

## 7. Setup Flow Spec (Strict)
### 7.1 Faction Selection
1. Turn order is randomized by default, with host override support.
2. No duplicate faction.
3. Start VP = 20.
4. On completion -> setup dwellings starts.

### 7.2 Starting Dwellings
Implement true base-game setup ordering with faction exceptions.

First placement pass:
1. P1 -> P2 -> ... -> Pn.
2. Chaos Magicians place only one dwelling total during setup.

Second placement pass:
1. Pn -> ... -> P2 -> P1.
2. Chaos Magicians skip second placement.
3. Nomads place third dwelling at designated setup step.

Validation:
1. Hex empty.
2. Hex on faction home terrain.
3. Not river.

### 7.3 Setup Bonus Cards
1. Reverse order from final setup placement order.
2. One unique card per player.
3. After all choose -> leftover cards each gain 1 coin before round 1 income.

## 8. Main Action & Reaction Ordering
### 8.1 Main Action Lock
At any time, exactly one active actor for main action.

### 8.2 Reaction Queue
If main action triggers leech:
1. Create deterministic leech queue in turn order among eligible neighbors.
2. Resolve all accepts/declines before next main action.
3. Cultists bonus resolution occurs after full queue resolution.

### 8.3 Pending Choice Precedence
If multiple decisions spawned by one action (example: upgrade -> favor -> town):
1. Favor decision resolves first where required by rules.
2. Then town tile decisions in deterministic order.
3. Then follow-up optional decisions (e.g., spade leftover discard).

## 9. UI Interaction Contract by User Requirement
## 9.1 Hex Click Behavior
On hex click, server returns legal intents for that hex for current player/context:
- `build_dwelling`
- `transform_only`
- `transform_and_build`
- `upgrade` (when own building)
- `invalid` with reason

For transform-only modal, show all legal target terrains and exact resource cost.

## 9.2 Cult Priest Spots
1. Cult track UI has clickable 3/2/1 spots.
2. Hover highlight restored and tested.
3. Disabled state for filled spots or invalid send.

## 9.3 Leech Modal
For each responder, modal text:
- “Accept X power for (X-1) VP?”
Buttons:
- Accept
- Decline

## 9.4 Upgrade Modal
When TP clicked/upgraded:
- Modal choices `Temple` or `Stronghold` (if both legal).
- If one illegal (resources/building limits), disable with reason.

## 9.5 Power Actions
1. Click octagon -> claim action.
2. For spade actions, flow enters spade follow-up chooser.
3. Double spade must support:
- two different transform targets
- at most one dwelling built across both transforms
- discard second spade option
 - sequential resolution only (step-by-step, no batch preselection)
4. Bridge action uses edge click (not center-hex click).

## 9.6 Special Actions
1. Orange octagons on SH / favor / bonus cards remain distinct clickable controls.
2. FAV water2 / cult bonus / Auren SH show cult-choice modal.
3. Darklings SH shows 0/1/2/3 worker conversion modal.
4. Halflings SH uses 3-spade sequence UI (and optional one dwelling).
5. Engineers action is rendered as an orange square in the same board location where other factions have SH action markers; action effect is bridge placement and counts as a main action.
6. Mermaids connect action is rendered as the same square-style special action control; action requires river click and is only legal if it forms a town, and does not consume main action (per user requirement).
7. Engineers bridge special action and Mermaids connect special action are reusable whenever legal (not once-per-round-limited octagons).

## 9.7 Pass
1. Click available bonus card.
2. Card darkens for the round and shows owner marker.
3. If unused special actions/free pending actions remain, show confirmation modal before pass.

## 9.8 Conversions
1. Player-board conversion buttons send free-action commands.
2. Allowed during player’s turn before/after main action until pass.
3. Server validates affordability and legal conversion ratios.

## 9.9 Error Surface
Any rejection must produce visible inline toast/banner + context near clicked control.

## 10. Action Confirmation (No Undo in V1)
1. Any action that commits irreversible state (build/upgrade/pass/leech response/tile selection) must go through explicit confirm in UI.
2. Once confirmed and accepted by server, the action is final in v1 (no post-confirm undo command).
3. Client may allow cancellation/adjustment before confirmation while action is still local-only preview state.
4. Server does not expose `undo_last_main_action` in v1 protocol.

## 11. Backend Changes
## 11.1 Websocket Layer
Files: `server/internal/websocket/*`
1. Replace limited `switch` with registry mapping all action types -> constructor/validator.
2. Add room subscriptions and room-scoped broadcast.
3. Add structured error codes.

## 11.2 Game Manager/API
Files: `server/internal/game/manager.go`
1. Add revision counter per game.
2. Add `ExecuteActionWithMeta` (action id, seat id, expected revision).
3. Add irreversible-action confirmation handling (`preview`/`confirm`) and optional action audit trail.

## 11.3 State Serialization
Files: `server/internal/game/manager.go` and related serializers.
Must include all pending and interaction-critical fields:
1. `pendingLeechOffers`
2. `pendingFavorTileSelection`
3. `pendingTownFormations`
4. `pendingSpades`
5. `pendingCultRewardSpades`
6. `pendingHalflingsSpades`
7. `pendingDarklingsPriestOrdination`
8. `pendingCultistsCultSelection`
9. `currentPendingDecision` (new normalized view)
10. `revision`

## 11.4 Rules Engine Tightening
Files: `server/internal/game/*`
1. Enforce turn ownership in all main actions.
2. Enforce phase/subphase for setup and action-only moves.
3. Enforce setup ordering and faction-specific setup exceptions.
4. Block turn progression while unresolved reaction queue/pending decisions exist.
5. Add missing main/free action types as first-class actions (not parser-only):
- burn
- conversion
- reclaim priest 1-step send explicit path
- bridge placement by edge geometry command

## 12. Frontend Changes
## 12.1 Action Service Expansion
File: `client/src/services/actionService.ts`
1. Replace narrow union with full action union.
2. Include `actionId` + `expectedRevision`.
3. Add helpers by action family (build/transform/upgrade/leech/etc).

## 12.2 Game State Store
File: `client/src/stores/gameStore.ts` (and types)
1. Track `revision`.
2. Track `pendingDecision`.
3. Track transient UI modal state keyed to pending decision.

## 12.3 Board Interaction Wiring
Files:
- `client/src/components/GameBoard/GameBoard.tsx`
- `client/src/components/GameBoard/HexGridCanvas.tsx`
- `client/src/components/CultTracks/CultTracks.tsx`
- `client/src/components/GameBoard/PowerActions.tsx`
- `client/src/components/GameBoard/PlayerBoards.tsx`
- `client/src/components/GameBoard/PassingTiles.tsx`

Work:
1. Wire all clickable elements to action submissions.
2. Add edge hit-testing in hex canvas for bridge placement.
3. Restore/verify cult priest spot hover + click behavior.
4. Add modal stack manager for pending decisions and confirmations.
5. Add irreversible-action confirm UX and cancellation before commit.

## 13. Action Catalog (Wire Types)
Canonical action `type` list for websocket payload:
1. `select_faction`
2. `setup_dwelling`
3. `setup_bonus_card`
4. `transform_build`
5. `upgrade_building`
6. `advance_shipping`
7. `advance_digging`
8. `send_priest`
9. `power_action_claim`
10. `power_spade_followup`
11. `power_bridge_place`
12. `engineers_bridge`
13. `special_action_use`
14. `special_followup`
15. `conversion`
16. `burn_power`
17. `discard_pending_spade`
18. `pass`
19. `accept_leech`
20. `decline_leech`
21. `select_favor_tile`
22. `select_town_tile`
23. `select_town_cult_top`
24. `use_cult_spade`
25. `select_cultists_track`
26. `halflings_apply_spade`
27. `halflings_build_dwelling`
28. `halflings_skip_dwelling`
29. `darklings_ordination`

## 14. Test Plan (Bazel)
## 14.1 Backend Unit + Integration
1. Setup order tests including Chaos/Nomads exceptions.
2. Action legality tests per catalog entry.
3. Leech queue + Cultists resolution tests.
4. Town + key-limited cult-top selection tests.
5. Confirmation/commit tests:
- preview cancellation leaves state unchanged
- confirmed actions are final and cannot be reversed in v1

## 14.2 Websocket Contract Tests
1. Room isolation.
2. Revision mismatch rejection.
3. Seat-authority rejection.
4. Idempotent `actionId` handling.

## 14.3 Frontend Interaction Tests
1. Hex click opens correct modal options.
2. Bridge edge click accuracy.
3. Priest spot click/hover behavior.
4. Pending modal flows (leech, favor, town, darklings, halflings).
5. Confirm-before-commit affordance and irreversible-action messaging.

## 15. Implementation Milestones
1. Milestone A: protocol + rooming + revision + serialization of pending state. **Status: Completed**
2. Milestone B: server action registry and strict turn/phase validation. **Status: Completed**
3. Milestone C: setup correctness + pass/bonus flow correctness. **Status: Completed**
4. Milestone D: wire core board actions + modals. **Status: Completed**
5. Milestone E: advanced actions (double spade split, bridges, special actions). **Status: Completed**
6. Milestone F: confirm-before-commit UX for irreversible actions. **Status: Completed**
7. Milestone G: full regression suite + multiplayer soak tests. **Status: Completed**

## 16. Locked Decisions (2026-02-21)
1. Multiplayer v1 uses no auction; all factions start at 20 VP.
2. Setup/faction order defaults to random and must be host-configurable.
3. Setup rules are strict base-rule ordering, including faction exceptions.
4. Setup bonus-card picks use strict reverse setup order.
5. Map is Base Game only for v1.
6. `shipping-bonus` and `temple-scoring-tile` are enabled by default.
7. Drop/resign handling is deferred.
8. Engineers bridge special action counts as main action and is shown as a square in the same board location where SH action markers appear for other factions.
9. Mermaids connect special action does not consume main action and is reusable whenever legal; same reusability rule applies to Engineers bridge special action.
10. Free conversions are allowed only during the acting player’s own turn.
11. Spade followups are forced sequential (no batch target preselection).
12. No post-confirm undo in v1.
13. If undo is reintroduced later, upgrade branch choice is part of action fingerprint and must remain fixed for forced replay.
14. Reconnect should restore seat automatically via token/session identity.
15. Spectators are out of scope for v1 and tracked in deferred roadmap.
16. Turn timers are out of scope for v1 and tracked in deferred roadmap.

## 17. Implementation Completion Status
Completed in current implementation:
1. Websocket room isolation + revisioned action protocol + seat-bound authority.
2. Expanded action registry in `perform_action` routing beyond faction/setup dwelling.
3. State serialization of pending decision structures needed by modal-driven UI.
4. Strict setup sequencing implementation (including Chaos/Nomads behavior and reverse bonus-pick order).
5. Frontend modal/interaction wiring for setup bonus pick, leech, favor, town, halflings, darklings, cultists, conversion/burn, and ship/dig upgrades.
6. Engineers reusable bridge special action implemented as worker-costed main action (`engineers_bridge`) and wired to square control.
7. Mermaids reusable connect action wired to square control and `special_action_use` (`MermaidsRiverTown`) targeting flow.
8. Pass flow now warns when optional specials/spades remain before confirmation.
9. Bridge edge hit-testing added in `HexGridCanvas` and wired through `GameBoard` bridge mode.
10. Chaos Magicians double-turn modal now authors nested `firstAction`/`secondAction` payloads for `special_action_use`.
11. `town_cult_top_choice` is fully wired end-to-end (engine action, websocket route, pending-decision serialization, and frontend modal with track selection constraints).
12. ACT6 split follow-up is implemented: leftover free spade becomes a blocking sequential follow-up, optional dwelling on the follow-up transform is rule-constrained, and explicit `discard_pending_spade` supports legal throwaway.
13. Pending spade and cult-reward-spade follow-ups are enforced server-side; unrelated actions are rejected until follow-ups resolve.
14. Replay compatibility remains intact with strict multiplayer setup updates (setup-dwelling validation now supports strict multiplayer mode and replay legacy mode as needed).
15. Websocket contract coverage now includes `spade_followup` + `discard_pending_spade` transitions.
16. A 5-seat reconnect churn multiplayer soak test is implemented in websocket E2E coverage.
17. UX polish pass completed for advanced modal flows (Chaos parameter templates/validation and pending-spade/cult-spade clarity banners).

Validation completed for this spec revision:
1. `bazel test //internal/websocket:websocket_test //internal/game:game_test //internal/replay:replay_test` (from `server/`) passed.
2. `bazel test //...` (from `server/`) passed.
3. `npm run type-check && npm run build` (from `client/`) passed.

Remaining out-of-scope items are tracked in `MULTIPLAYER_TODO_LATER.md`.

## 18. Golden End-to-End Validation Plan (Snellman)
### 18.1 Primary Golden Game
1. Source URL: `https://terra.snellman.net/faction/4pLeague_S69_D1L1_G2`
2. Local fixture: `server/internal/replay/testdata/snellman_batch/4pLeague_S69_D1L1_G2.txt`
3. Expected final totals (must match exactly):
- Nomads (RA): 166
- Darklings (tom8918): 137
- Mermaids (Gewalf): 130
- Witches (Fujiwara): 124

### 18.2 Test Architecture
Run this as a two-layer golden test so rules, protocol, and UI are all validated.

Layer A: Protocol/Rules golden replay (websocket E2E)
1. Start in-memory websocket server (`httptest`) with game + lobby managers.
2. Create game, join 4 seats, start with deterministic host options matching the fixture assumptions.
3. Parse/drive the full action stream from the fixture-derived concise actions.
4. Submit each move as `perform_action` over websocket using `actionId` + `expectedRevision`.
5. Require `action_accepted` and monotonic revision progression at every step.
6. Assert final phase is end and final scoring totals match exactly.

Layer B: UI golden replay (browser E2E)
1. Launch 4 browser clients (one per seat) and connect to one live game.
2. Drive the same action plan through actual UI interactions (hex clicks, modals, card/tile picks, cult selections, pass flow).
3. Wait for `game_state_update`/revision advancement after each committed action.
4. Fail if any unexpected `action_rejected` is surfaced.
5. At completion, assert final score panel matches expected totals exactly.

### 18.3 Oracle/Comparison Strategy
1. Use replay parser/simulator output as oracle stream for move order and expected terminal totals.
2. Normalize player identity keys (username/display/faction alias differences) before score comparison.
3. Capture per-step audit artifacts:
- action index
- actor
- websocket payload
- revision before/after
- pending decision type (if any)

### 18.4 Success Criteria
1. Game runs from lobby creation through final scoring without manual intervention.
2. All actions are accepted in legal order with no deadlock in pending-decision flows.
3. Final totals exactly equal:
- Nomads 166
- Darklings 137
- Mermaids 130
- Witches 124
4. Run is repeatable (stable across multiple executions).

### 18.5 Coverage Notes
1. This golden game exercises a broad multiplayer path: setup, leech accept/decline, conversions, upgrades, power actions, cult spades, favor/town picks, passing, and endgame scoring.
2. It does not cover every faction-specific edge branch (for example Engineers bridge, Halflings 3-spade branch, Cultists pending cult-choice branch), so focused scenario tests remain required alongside the golden run.
