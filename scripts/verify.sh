#!/usr/bin/env bash
# Runs every deterministic gate against the current commit and records the
# result against the exact patch. scripts/push-gate.sh refuses a push unless
# this passed for the commit being pushed.
#
#   scripts/verify.sh            run all gates
#   FUZZTIME=60s scripts/verify.sh
set -uo pipefail

cd "$(git rev-parse --show-toplevel)"

BASE_REF="${BASE_REF:-origin/main}"
FUZZTIME="${FUZZTIME:-15s}"
EVIDENCE_DIR=".evidence"
mkdir -p "$EVIDENCE_DIR"

if [ -n "$(git status --porcelain)" ]; then
  echo "verify: the working tree has uncommitted changes. Commit first; evidence is recorded against a commit." >&2
  exit 1
fi

HEAD_SHA="$(git rev-parse HEAD)"
BASE_SHA="$(git merge-base "$BASE_REF" HEAD 2>/dev/null || git rev-parse "$BASE_REF")"
PATCH_ID="$(git diff "$BASE_SHA" HEAD | git patch-id --stable | cut -d' ' -f1)"
PATCH_ID="${PATCH_ID:-empty}"
REPORT="$EVIDENCE_DIR/$HEAD_SHA.md"
LOG="$EVIDENCE_DIR/$HEAD_SHA.log"
: > "$LOG"

FAILED=0
ROWS=""

gate() {
  local name="$1"; shift
  local start end status detail
  start=$(date +%s)
  {
    echo
    echo "=== $name: $*"
  } >> "$LOG"
  if "$@" >> "$LOG" 2>&1; then
    status="pass"
  else
    status="FAIL"
    FAILED=1
  fi
  end=$(date +%s)
  detail="$(tail -n 1 "$LOG" | cut -c1-90 | tr '|' '/')"
  ROWS="${ROWS}| ${name} | ${status} | $((end - start))s | \`${detail}\` |
"
  printf '%-22s %s (%ss)\n' "$name" "$status" "$((end - start))"
}

check_gofmt() {
  local out
  out="$(gofmt -l cmd internal)"
  if [ -n "$out" ]; then
    echo "gofmt needed on: $out"
    return 1
  fi
  echo "gofmt clean"
}

check_lint() {
  if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint not installed; hosted CI is the authority for lint"
    return 0
  fi
  local out
  if out="$(golangci-lint run ./... 2>&1)"; then
    echo "golangci-lint clean"
    return 0
  fi
  case "$out" in
    *"unsupported version of the configuration"*)
      echo "local golangci-lint is v2 and cannot read the v1 config; running the equivalent linter set"
      golangci-lint run --no-config --default none \
        -E errcheck,govet,staticcheck,ineffassign,unused,misspell,gocritic,gosec,prealloc ./... \
        && echo "golangci-lint v2 equivalent set clean"
      return $?
      ;;
  esac
  echo "$out"
  return 1
}

check_binary() {
  local bin="$EVIDENCE_DIR/log2json-$HEAD_SHA"
  go build -o "$bin" ./cmd/log2json || return 1
  local total_in=0 total_out=0 f in out
  for f in testdata/sample_*.log; do
    in=$(grep -c '' "$f")
    out=$("$bin" --adaptive --quiet < "$f" | grep -c '^{.*}$')
    total_in=$((total_in + in))
    total_out=$((total_out + out))
    if [ "$in" != "$out" ]; then
      echo "$f: $in lines in, $out JSON objects out"
      rm -f "$bin"
      return 1
    fi
  done
  rm -f "$bin"
  echo "built binary: $total_in lines in, $total_out JSON objects out"
}

gate "gofmt"            check_gofmt
gate "go vet"           go vet ./...
gate "lint"             check_lint
gate "tests (race)"     go test -race -count=1 ./...
gate "architecture"     go test -count=1 ./internal/archtest
gate "golden corpus"    go test -count=1 ./cmd/log2json -run TestGoldenCorpus
gate "fuzz $FUZZTIME"   go test ./cmd/log2json -run '^$' -fuzz FuzzPipelineInvariants -fuzztime "$FUZZTIME"
gate "built binary"     check_binary

RESULT="pass"
[ "$FAILED" = 1 ] && RESULT="fail"

{
  echo "### Verification evidence"
  echo
  echo "- Commit: \`$HEAD_SHA\`"
  echo "- Base: \`$BASE_SHA\` ($BASE_REF)"
  echo "- Patch id: \`$PATCH_ID\`"
  echo "- Result: **$RESULT**"
  echo "- Toolchain: $(go version | cut -d' ' -f3-4)"
  echo
  echo "| Gate | Result | Time | Last line |"
  echo "| --- | --- | --- | --- |"
  printf '%s' "$ROWS"
} > "$REPORT"

printf '{"head":"%s","base":"%s","patch_id":"%s","result":"%s"}\n' \
  "$HEAD_SHA" "$BASE_SHA" "$PATCH_ID" "$RESULT" > "$EVIDENCE_DIR/latest.json"

echo
echo "report: $REPORT"
echo "log:    $LOG"
[ "$RESULT" = "pass" ]
