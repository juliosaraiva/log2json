---
name: encode-correction
description: Turn a review comment, a correction from the user, or a reverted change into a deterministic check so the same mistake cannot recur. Use whenever you are told something was wrong.
---

# Encode a correction

A correction that lives only in prose will be repeated. Find the lowest layer that can enforce it.

1. Write the mistake as a pattern: "code that does X in place Y".
2. Search the codebase for other instances. Fix them in the same pull request, or the new check starts red.
3. Choose the layer:
   - **Design.** Can an API or type change make the mistake impossible to write? Do that.
   - **Structure.** Is it visible in the syntax tree or the import graph? Add a rule to `internal/archtest/arch_test.go`.
   - **Behavior.** Is it visible in output for some input? Add an invariant in `cmd/log2json/invariants_test.go`, or a line to a `testdata/sample_*.log` file and regenerate the golden output.
   - **Judgment.** Only if none of the above can express it, add a line to the relevant skill.
4. Write the failure message for the next agent that trips it: what is wrong, why it matters here, and the fix that addresses the cause. Never suggest a suppression.
5. Prove the check works: reintroduce the mistake, see the check fail with your message, then remove it.
6. After a revert, the pull request that re-lands the change must include the check that would have caught the original.
