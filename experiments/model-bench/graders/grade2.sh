#!/usr/bin/env bash
# usage: grade2.sh <arm-worktree>
A="$1"; G="$(dirname "$0")"; BASE=bench/us078/r2-base
cd "$A" || exit 1
echo "== $(basename "$A")"
echo "diff vs base:"; git diff --shortstat $BASE | sed 's/^/  /'; echo "  files: $(git diff --name-only $BASE | tr '\n' ' ') untracked: $(git status --short | grep -c '^??')"
echo "t.Skip added: $(git diff $BASE -- '*_test.go' | grep -c '^+.*t\.Skip')  test lines removed: $(git diff $BASE -- '*_test.go' | grep -c '^-[^-]')"
echo "DSN:"; grep -n 'sql.Open' internal/database/database.go | sed 's/^/  /'
echo "build: $(go build ./... 2>&1 | head -3 | tr '\n' ' ')ok?"
for f in pragma backup fk; do cp $G/hidden_${f}_test.go.txt internal/database/hidden_${f}_test.go; done
echo "hidden+visible:"; go test ./internal/database/ -count=1 -run 'TestHidden|TestPragmas|TestForeignKeys' -v 2>&1 | grep -E '^(--- |ok|FAIL)' | sed 's/^/  /'
rm -f internal/database/hidden_*_test.go
echo "full suite (excl. T20):"; go test ./... -count=1 2>&1 | grep -E '^(ok|FAIL|--- FAIL)' | grep -v 'T20_Golden' | grep -v '^ok' | sed 's/^/  /'; echo "  (only non-ok lines shown)"
echo "T3 x30 failures:"; go test ./internal/repository/ -run TestOrder_T3_ConcurrentOpenSingleWinner -count=30 2>&1 | grep -c '^--- FAIL' | sed 's/^/  /'
