# Reviewing log2json

For a pilot PR, give a fresh reviewer the intent, diff, affected callers and
synthetic input. Run two separate passes. Identify the revision and record
when a pass is unavailable.

- **Correctness.** Follow a line through detection, parsing and emission.
  Check record count, JSON field types, stderr and any affected flag. If the
  parser is shared, exercise each input path that uses it.
- **Defensive security.** Look for changed trust boundaries, unbounded work
  on input, or exposure of raw log data. For workflow or release changes,
  have the responsible human inspect permissions and publishing behavior.

For each finding give the triggering input or condition, impact and file/line.
Fix it and link the check, or record why it was dismissed. Recheck new changes,
affected callers and open findings. Suppress duplicates. After three rounds,
hand the decision to a maintainer. Existing CI, approval and ownership rules
apply throughout.
