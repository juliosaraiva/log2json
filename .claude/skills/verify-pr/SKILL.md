---
name: verify-pr
description: Independently verify a pull request you did not write and give a PASS, HOLD, or FAIL verdict tied to its exact head commit. Use when asked to review, verify, or approve a PR.
---

# Verify a pull request

You are checking someone else's claim. Do not trust the description. Reproduce everything.

1. Refuse if you wrote any commit in the pull request. Say so and stop.
2. Fetch the head commit and record its SHA. Your verdict applies to that SHA only.
3. Read the diff before the description. Write down what you believe it changes.
4. Compare with the description. A difference between the two is a finding.
5. For a bug fix, check out the base, apply only the new tests, and confirm they fail for the stated reason. Then confirm they pass on the head.
6. Run `make verify` on the head yourself. Do not accept a pasted report.
7. Build the binary and try to break the change: empty input, blank lines, very long lines, non-ASCII text, mixed formats, each `--format`, `--adaptive`, and `--fields`.
8. Read the golden diff if there is one. Each changed line must follow from the stated change.
9. Look for gates made weaker: removed tests, new `nolint`, loosened invariants, generator changes that avoid the failing input.

## Verdict

Post one comment:

```
Verdict: PASS | HOLD | FAIL
Head: <sha>
Risk class: inert | ordinary | contract | infrastructure
Reproduced: <commands you ran and what you saw>
Findings: <numbered, or "none">
```

PASS means you would merge it yourself. HOLD means it is correct as far as you can tell but its risk class needs a person. FAIL lists what must change. A new push voids the verdict.
