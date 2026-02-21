# Multiplayer Deferred Roadmap

This file tracks agreed out-of-scope items for multiplayer v1 and concrete follow-up tasks for later milestones.

## P1 (Next After V1)
1. Spectator mode
- Add websocket read-only subscriptions for active games.
- Add spectator-specific game route and role in state payload.
- Hide all action controls for spectator clients.

2. Turn timers
- Add per-turn countdown state on server.
- Add timeout policy configuration (auto-pass, grace period, reconnect grace).
- Add visible timer UI and timeout warnings.

3. Mid-game resign/drop handling
- Add explicit `resign` action and confirmation flow.
- Define game continuation policy (end immediately vs continue with placeholders/AI).
- Make final scoring robust for resigned/dropped players.

4. Login/auth system
- Add account login and authenticated session handling for multiplayer seats.
- Replace display-name-only seat binding with authenticated identity binding.
- Add auth-aware reconnect/session-restore rules and protected game actions.

5. Golden coverage expansion
- Add more "golden" full-game fixtures for multiplayer end-to-end validation.
- Include games that cover currently underrepresented branches (Engineers bridge, Halflings 3-spade flow, Cultists cult-choice flow, Mermaids connect edge cases).
- Keep expected final score assertions per fixture as hard pass/fail gates.

## P2 (Post-Stability)
1. Spectator permissions
- Private/public table visibility.
- Invite-only spectator links.

2. Timer controls
- Host-configurable timer presets.
- Pause/resume for admin/host.

3. Recovery and observability
- Rejoin diagnostics panel (seat restore attempts, auth failures).
- Action timeline export for dispute/debug.

## P3 (Future Expansion)
1. Alternate setup modes
- Auction/bidding mode parity with BGA-like starts.
- Implement both regular auction and fast auction setup variants.

2. Expanded faction/map support
- Fire & Ice factions.
- Landscape expansion.

3. Competitive features
- Ranked matchmaking and rating updates.
