#!/usr/bin/env bash
# Second-opinion review by Gemini (Antigravity CLI `agy`) for the /us pipeline.
#
#   scripts/gemini-review.sh diff     <US-ID> <worktree> [round] [red-sha]
#   scripts/gemini-review.sh contract <US-ID> <worktree>
#
# Writes <worktree>/changes/<US-ID>/gemini-<mode>[-round-N].md. Never fails the pipeline:
# the outcome (OK | EMPTY | TIMEOUT | NO_AGY) is on the first line of the file and on stdout.
# Gemini is advisory only (experiments/model-bench: it missed a seeded bug and 3 of 9 calls
# returned nothing). Every finding must be verified by the orchestrator before it is acted on.
set -u

MODE=${1:-}; ID=${2:-}; WT=${3:-}; ROUND=${4:-1}; RED=${5:-}
if [[ ! $MODE =~ ^(diff|contract)$ || -z $ID || ! -d $WT ]]; then
  echo "usage: $0 diff|contract <US-ID> <worktree> [round] [red-sha]" >&2; exit 64
fi

OUTDIR="$WT/changes/$ID"; mkdir -p "$OUTDIR"
if [[ $MODE == diff ]]; then OUT="$OUTDIR/gemini-diff-round-$ROUND.md"; else OUT="$OUTDIR/gemini-contract.md"; fi
PACKET="$OUTDIR/gemini-$MODE-input.md"
MAX_INLINE=24000   # Windows argv limit is ~32K; above this the packet is read from a file
TIMEOUT=300

finish() { # status, body
  { echo "STATUS: $1"; echo "MODE: $MODE  ROUND: $ROUND  DATE: $(date -u +%FT%TZ)"
    echo "ADVISORY: unverified. Check every finding against the code (file:line) before acting."
    echo; echo "$2"; } > "$OUT"
  echo "gemini-review: $1 -> $OUT"; exit 0
}

AGY=${AGY:-$(command -v agy 2>/dev/null || true)}
[[ -z $AGY && -x "$HOME/AppData/Local/agy/bin/agy.exe" ]] && AGY="$HOME/AppData/Local/agy/bin/agy.exe"
[[ -z $AGY ]] && finish NO_AGY "agy not found. Set AGY or add ~/AppData/Local/agy/bin to PATH."

RULES='Project rules (CLAUDE.md): everything written in the repo is English (only end-user UI strings and Colombian domain terms are Spanish). Date ranges are half-open [from, to), never BETWEEN. SQL uses ? parameters, never concatenation. Never edit an already-applied migration; add the next one. Historically referenced entities are deleted logically (active = 0). Errors are wrapped with %w, never swallowed. Frontend uses CSS variables, never hardcoded colors; currency via formatCOP; HTTP via api. No CGO dependencies. A behavior change updates project-docs/spec/ in the same change. Never skip or weaken an existing test.'

{
  if [[ $MODE == diff ]]; then
    BASE=$(git -C "$WT" merge-base dev HEAD 2>/dev/null || echo dev)
    echo "# Review packet - $ID, round $ROUND"
    echo; echo "## Rules"; echo "$RULES"
    echo; echo "## Contract (acceptance criteria)"
    head -c 12000 "$OUTDIR/contract.md" 2>/dev/null || echo "(no contract.md)"
    if [[ -n $RED ]]; then
      echo; echo "## Test files changed after the red-tests commit (must be empty)"
      git -C "$WT" diff --stat "$RED"..HEAD -- '*_test.go' '*.test.ts' '*.test.tsx'
    fi
    echo; echo "## Diff against dev (-U8)"
    git -C "$WT" diff -U8 "$BASE"..HEAD -- . ':(exclude)changes' ':(exclude)pnpm-lock.yaml' ':(exclude)*.lock'
  else
    echo "# Contract review packet - $ID"
    echo; echo "## Rules"; echo "$RULES"
    echo; echo "## Contract"
    head -c 20000 "$OUTDIR/contract.md" 2>/dev/null || echo "(no contract.md)"
  fi
} > "$PACKET"

if [[ $MODE == diff ]]; then
  TASK='You are an independent reviewer. Review the DIFF in the packet against the CONTRACT and RULES. List only real defects, one per line, as: SEVERITY | file:line | defect | requirement or AC violated (SEVERITY is BLOCKER or MINOR). State explicitly whether any existing test was weakened, skipped or edited, and whether a behavior change lacks a spec update. Quote the offending line. Do not praise, do not restate the diff, do not invent files. If nothing is wrong, answer exactly: CLEAN. Max 15 lines. Use only the packet: do not open or search any other file.'
else
  TASK='You are an independent requirements reviewer. Read the CONTRACT in the packet. List only real problems, one per line, as: KIND | section | problem. KIND is AMBIGUOUS, UNTESTABLE, MISSING_CASE or RULE_CONFLICT (against RULES). Typical misses: empty/zero/negative values, concurrency, inactive or deleted entities, permissions (admin vs cashier), half-open date ranges, spec not updated. Do not praise or summarize. If nothing is wrong, answer exactly: CLEAN. Max 15 lines. Use only the packet: do not open or search any other file.'
fi

# Small packets go inline; large ones are read from the file. Headless mode auto-denies reads outside the
# workspace, so the packet folder is added with --add-dir (an allow rule in settings.json does not work).
if (( $(wc -c < "$PACKET") <= MAX_INLINE )); then
  PROMPT="$TASK"$'\n\n'"$(cat "$PACKET")"
else
  WINPATH=$(cygpath -m "$PACKET" 2>/dev/null || echo "$PACKET")
  PROMPT="$TASK"$'\n\n'"The packet is in the file $WINPATH. Read it completely first, then answer."
fi

ADDDIR=$(cygpath -m "$OUTDIR" 2>/dev/null || echo "$OUTDIR")
ANSWER=""
for attempt in 1 2; do
  ANSWER=$(timeout "$TIMEOUT" "$AGY" --mode plan --add-dir "$ADDDIR" --print-timeout "${TIMEOUT}s" --print "$PROMPT" 2>&1); rc=$?
  [[ $rc -eq 124 ]] && finish TIMEOUT "agy exceeded ${TIMEOUT}s (attempt $attempt)."
  [[ -n ${ANSWER//[[:space:]]/} && $ANSWER != *"no output produced"* ]] && break
  ANSWER=""
done
[[ -z $ANSWER ]] && finish EMPTY "agy returned nothing twice (usually a denied read_file or a silent failure)."
finish OK "$ANSWER"
