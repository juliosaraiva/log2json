---
name: ship
description: Take one change from idea to an open pull request with verification evidence. Use for any bug fix, feature, or refactor in this repository.
---

# Ship one change

The goal is a pull request a reviewer can accept from its evidence without reading every line.

1. **State the change in one sentence.** If it needs "and", split it into two pull requests.
2. **Branch from fresh trunk.** `git fetch origin main && git checkout -b <type>/<slug> origin/main`.
3. **Bug fix: reproduce first.** Write the test that names the correct behavior. Run it against unchanged code and keep the failing output. A fix with no failing-first test is not accepted.
4. **Make the smallest change that passes.** Prefer deleting code. Touch one package when you can. Do not reformat or rename nearby code.
5. **Show the real artifact.** Build the binary and run the input from step 3 through it, before and after. Keep both outputs.
6. **Run `make verify` on the committed change.** Fix what fails at its root. Never weaken a gate, delete a golden file, or add a `nolint` to get green.
7. **Intended output change?** Regenerate with `go test ./cmd/log2json -run TestGoldenCorpus -update-golden` and read the diff line by line. Every changed line must be explained by the one-sentence change.
8. **Push and open the pull request.** Body sections: Why, What changed, Before and after, the contents of `.evidence/<commit>.md`, Risk, and Verdict.
9. **Verdict.** Hand the PR number to the `pr-verifier` agent. If no independent verifier can run, write `BLOCKED: independent review` in the Verdict section and stop. Your own review, passing tests, and CI do not count as the verdict.
10. **After CI.** Confirm hosted checks on the pushed head. If the head changes, steps 6 to 9 repeat for the new commit.

## Risk classes

| Class | Examples | Who may merge |
| --- | --- | --- |
| Inert | docs, comments, test-only additions | verifier PASS and green CI |
| Ordinary | bug fix with failing-first test and no golden diff | verifier PASS and green CI |
| Contract | any golden diff, new flag, changed field name or type, exit code | a person |
| Infrastructure | workflows, release config, hooks, this stack | a person |
