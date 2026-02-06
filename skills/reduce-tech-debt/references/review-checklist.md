# Tech Debt Reduction Review Checklist

Use this checklist before finalizing duplicate/dead-code cleanup.

1. Behavior preserved
- Confirm tests pass for all touched flows.
- Confirm no API contract changes unless requested.

2. Duplication refactor quality
- Extracted helpers have clear names and inputs.
- Call sites are simpler, not more abstract/confusing.
- No hidden global state introduced.

3. Dead-code removals are safe
- No dynamic references were missed.
- No framework convention file was removed accidentally.
- Public exports removed only when all consumers were checked.

4. Scope discipline
- Changes are grouped into small, reviewable commits.
- Each patch has a clear rationale in commit message/summary.
