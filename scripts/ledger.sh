#!/usr/bin/env bash
# Counts pull requests and reverts for a repository so the revert rate is a
# measured number, not an impression.
#
#   scripts/ledger.sh [owner/repo]
#
# A revert is a merged pull request whose title starts with "Revert" or that
# carries the label "revert". Rate = reverts / merged pull requests that are
# not themselves reverts.
set -euo pipefail

repo="${1:-$(git remote get-url origin | sed -E 's#(\.git)?$##; s#.*[:/]([^/]+/[^/]+)$#\1#')}"

rows="$(gh api --paginate "repos/$repo/pulls?state=all&per_page=100" \
  --jq '.[] | [.state, (if .merged_at then "merged" else "unmerged" end), (if ((.title | test("^Revert")) or ([.labels[].name] | index("revert") != null)) then "revert" else "change" end)] | join(" ")')"

opened=0; merged=0; open=0; reverts=0
while read -r state merge_state kind; do
  [ -z "${state:-}" ] && continue
  opened=$((opened + 1))
  [ "$state" = "open" ] && open=$((open + 1))
  if [ "$merge_state" = "merged" ]; then
    if [ "$kind" = "revert" ]; then
      reverts=$((reverts + 1))
    else
      merged=$((merged + 1))
    fi
  fi
done <<< "$rows"

echo "| Metric | Value |"
echo "| --- | --- |"
echo "| Pull requests opened | $opened |"
echo "| Still open | $open |"
echo "| Merged (excluding reverts) | $merged |"
echo "| Reverts merged | $reverts |"
if [ "$merged" -gt 0 ]; then
  rate="$(awk -v r="$reverts" -v m="$merged" 'BEGIN { printf "%.2f%%", 100 * r / m }')"
  upper="$(awk -v r="$reverts" -v m="$merged" 'BEGIN { u = 100 * 3 / m; if (u > 100) u = 100; printf "%.2f%%", u }')"
  echo "| Revert rate | $rate |"
  [ "$reverts" -eq 0 ] && echo "| 95% upper bound with zero reverts (rule of three) | $upper |"
else
  echo "| Revert rate | not measurable: nothing merged yet |"
fi
