## Intent and decision

What will a CLI user observe? What decision needs a maintainer?

## Evidence

Tested commit, command or CI link, expected result, and observed output. For a
bug, show the same test failing on the old code and passing with the fix. Use
synthetic log lines. Note any limits or checks that could not run.

## Risk and review

Proposed tier (low / ordinary / high), who is affected, how to detect harm,
and what reverting restores. List sensitive paths and responsible owners.
Link the fresh-context correctness and defensive security findings. Record a
fix or a reason for dismissing each substantive finding.

Before human review, enter the time, exact commit, and proposed automated-only
verdict (clear / hold / unavailable) with the reason. After human review, record
material corrections caught only by a person on that revision, or mark pending
or none. Append an entry for each new head so earlier verdicts stay visible.

## Rollout and recovery

Accountable human, first users or release, signal to stop, how to roll back,
and how to verify recovery or repair output already produced.
