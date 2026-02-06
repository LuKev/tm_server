---
name: reduce-tech-debt
description: Reduce maintainability debt by finding and consolidating duplicated code and by identifying and removing dead code. Use when requests mention tech debt cleanup, repeated logic, copy-paste code, unused functions/files/exports, or safe refactoring to improve maintainability without changing behavior.
---

# Reduce Tech Debt

## Overview

Use this skill to reduce duplication and dead code with a safety-first workflow:
1. Measure and locate duplicates/dead code.
2. Refactor in small, behavior-preserving patches.
3. Verify with tests/lint/type checks after each batch.

## Workflow

1. Map the stack quickly.
- Detect primary languages and tooling from files such as `package.json`, `pyproject.toml`, `go.mod`, `Cargo.toml`.
- Prefer ecosystem-native analyzers first, then use bundled scripts as fallback.

2. Create a baseline report.
- Save duplicate and dead-code findings before edits (paths, symbols, severity, confidence).
- Prioritize high-churn and high-risk modules first.

3. Reduce duplicate code.
- Prefer extraction to helper functions/modules when at least 2 call sites share stable logic.
- Avoid over-abstraction; do not merge code paths with materially different business rules.
- Keep naming domain-specific and call sites readable.

4. Remove dead code.
- Remove unused imports, params, locals, private helpers, stale exports, and unreachable branches.
- Remove unused files only after confirming there are no dynamic references (runtime loaders, reflection, route/file-system conventions).

5. Verify safety.
- Run relevant tests and static checks after each focused patch.
- If confidence is low, stage changes in smaller commits and keep removals conservative.

## Duplicate Code Playbook

1. Detect candidates.
- Run `scripts/find_duplicate_blocks.py` for fast clone hints.
- If available, run stronger ecosystem tools (for example, `jscpd`) and intersect results.

2. Triage candidates.
- Prioritize clones with:
  - shared bug-fix history,
  - business-critical paths,
  - repeated branchy logic.
- Skip intentionally duplicated code (tests/fixtures, migrations, generated outputs).

3. Refactor safely.
- Extract pure helper functions first.
- Pass dependencies as parameters; avoid hidden global coupling.
- Preserve public APIs unless explicitly requested.

4. Validate.
- Run tests touching all modified call sites.
- Check complexity/readability did not regress.

## Dead Code Playbook

1. Collect findings from native analyzers first.
- TypeScript/JavaScript: TS compiler + lint rules + optional dead-export scanners.
- Python: linters + `vulture` (if available).
- Go: `staticcheck` (`U1000`) and compiler warnings.

2. Use bundled scanner for TS/JS exports.
- Run `scripts/find_unused_ts_exports.py` as a heuristic pass.
- Treat output as candidates; confirm each item before removal.

3. Remove in safe order.
- Unused imports/locals/params.
- Unused private functions/classes.
- Unused exported symbols.
- Unused files/modules (last, highest caution).

4. Re-run checks and tests after each removal batch.

## Safety Rules

- Never delete code with uncertain runtime references.
- Treat dynamic usage patterns as blockers until proven unused.
- Keep changes narrow and reversible.
- Prefer multiple small patches over one sweeping rewrite.

## Commands

Run bundled scripts from the skill directory or by absolute path.

```bash
python3 scripts/find_duplicate_blocks.py --root .
python3 scripts/find_unused_ts_exports.py --root .
```

If project-native tools exist, run those too and prioritize their findings.

## Resources

- `scripts/find_duplicate_blocks.py`: heuristic duplicate block detector.
- `scripts/find_unused_ts_exports.py`: TypeScript/JavaScript dead-export candidate scanner.
- `references/review-checklist.md`: concise checklist for safe refactor review.
