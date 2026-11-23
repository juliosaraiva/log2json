# log2json

Streaming log-to-NDJSON converter. Go standard library only.

## How work ships here

1. One pull request per change, branched from `origin/main`. One bug, one fix, one PR.
2. A bug fix starts with a test that fails on `main` and names the behavior. Record the failing output.
3. `make verify` must pass on the commit you push. A hook blocks `git push` and PR creation otherwise.
4. The PR body carries the evidence report from `.evidence/<commit>.md` and the before/after output of the built binary.
5. You do not approve your own work. The `pr-verifier` agent, or a person, gives the verdict.

Use the `ship` skill for the full procedure.

## When you are corrected

Do not add a sentence to this file. Encode the correction where it cannot be ignored, in this order of preference:

1. Make the mistake impossible in the API or types.
2. Add a rule to `internal/archtest/arch_test.go` with a message that says how to fix the root cause.
3. Add an invariant to `cmd/log2json/invariants_test.go` or a case to the golden corpus.
4. Only for judgment calls no check can express: a skill under `.claude/skills/`.

Use the `encode-correction` skill.

## Contracts the checks enforce

- Every input line becomes exactly one JSON object on stdout or one reported error on stderr.
- stdout carries NDJSON only.
- Parsers are deterministic and do not import other pipeline stages.
- Output for `testdata/sample_*.log` matches `testdata/golden/`. An intended change updates the golden files in the same PR.
