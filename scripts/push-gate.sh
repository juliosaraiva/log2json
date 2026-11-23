#!/usr/bin/env bash
# Claude Code PreToolUse hook for the Bash tool. Blocks `git push` and pull
# request creation unless scripts/verify.sh passed for the exact commit at HEAD.
# Exit code 2 blocks the tool call and shows stderr to the agent.
set -uo pipefail

payload="$(cat)"
command_text="$(printf '%s' "$payload" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input", {}).get("command", ""))' 2>/dev/null)" || command_text="$payload"
[ -z "$command_text" ] && command_text="$payload"

if ! printf '%s' "$command_text" | grep -Eq 'git( +-[cC] +[^ ]+)* +push|gh +pr +create|gh +api +[^|;&]*repos/[^ /]+/[^ /]+/pulls( |$)'; then
  exit 0
fi

root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$root" || exit 0

head_sha="$(git rev-parse HEAD)"
latest=".evidence/latest.json"

fail() {
  echo "push-gate: $1" >&2
  echo "push-gate: run scripts/verify.sh, fix what it reports, and push the commit it verified. Do not bypass this gate." >&2
  exit 2
}

[ -n "$(git status --porcelain)" ] && fail "the working tree has uncommitted changes, so no evidence can match what would be pushed."
[ -f "$latest" ] || fail "no verification evidence exists for this checkout."
grep -q "\"head\":\"$head_sha\"" "$latest" || fail "evidence is for a different commit than HEAD ($head_sha)."
grep -q '"result":"pass"' "$latest" || fail "the last verification of HEAD failed."
exit 0
