---
name: fuzz-loop
description: Hunt for bugs with fuzzing and exploratory inputs, then open one fix pull request per distinct bug. Use after a release, on a schedule, or when asked to find bugs.
---

# Fuzz loop

1. Start from fresh `origin/main`. Build the binary.
2. Run `go test ./cmd/log2json -run '^$' -fuzz FuzzPipelineInvariants -fuzztime 5m`.
3. Explore by hand what the fuzzer cannot judge: is the output *right*, not merely well formed? Feed real-world shapes of each supported format and compare the fields with what a person would expect. Look for values that change meaning between input and output.
4. For each finding, reduce it to the smallest input and write one line: input, actual output, expected output.
5. Remove duplicates: two findings with the same root cause are one bug. Check open pull requests and issues before starting a fix.
6. Classify each as new since the last release or pre-existing, and as high, medium, or low severity. Silent data loss or corruption is high.
7. For each bug, run the `ship` skill once. Never batch two bugs into one pull request.
8. If a bug shows the invariants were too weak to catch it, strengthen the invariant or the generator in the same pull request as the fix.
9. Report a table: bug, severity, new or pre-existing, pull request, verdict.

Stop when a five-minute fuzz run and one exploratory pass find nothing new. A duration is not a finish condition.
