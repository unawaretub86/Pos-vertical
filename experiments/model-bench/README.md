# Model-comparison experiment (October 2026)

Question: can the `/us` pipeline use cheaper models (Sonnet, Haiku) or Gemini for some roles without losing quality, and is it worth changing the pipeline? Claude Code subagents were run on **Opus 5.5, Sonnet 5.5 and Haiku 5.5**, plus Gemini through the Antigravity CLI (`agy`), on three real tasks of this repository. Run on 2026-10-12/13 by the Tech Lead with the PO.

This folder is the durable record. The full result tables are in `project-docs/pipeline-bench.md` (sections M-1, M-2, M-3, same branch).

## Method (what was controlled)
- Same task, same base commit, same prompt for every arm; one git worktree and one branch per arm.
- Hidden tests the agents never saw (`graders/hidden_*_test.go.txt`; renamed `.txt` so `go build ./...` ignores them) plus a visible failing test per task.
- Traps checked mechanically: `t.Skip` added, assertion lines removed, existing tests edited.
- Cost: only a combined token total per subagent is available (no input/output split), so cost is a bound at list prices, not a measurement.
- To compare at equal cost, Haiku arms were run in several copies (n=5), Opus and Sonnet in n=1 (M-1) or n=2 (M-2, M-3).
- Rollback tag `pipeline-v1` marks the pipeline before the experiment.

## Rounds
| ID | Task | Arms | Result |
|---|---|---|---|
| M-1 | US-078: really enforce SQLite foreign keys | opus 1, sonnet 1, haiku 5 | All 7 identical (DSN `_pragma=foreign_keys(1)` + 4 fixtures), 6/6 hidden tests. Task too easy to discriminate. |
| M-2 | US-078b: make WAL and `busy_timeout` effective on every pooled connection | opus 2, sonnet 2, haiku 5 | All 9 identical, 17/17 tests, flaky `TestOrder_T3` 0/30. Again no discrimination. |
| M-3 | Customer logical delete (design task with open points) | opus 2, sonnet 2, haiku 5, gemini 1 | All 10 pass 9/9 hidden tests with the same core design. They differed in judgement, see below. |

## Findings
1. On test-guided mechanical work every model reached the same correct change; Haiku costs 5 to 30 times less at list price but used 1.1-1.4x the tokens and 2-3x the tool calls.
2. **The tests decided, not the model.** A failing visible test and clear limits did the work.
3. Differences appeared around the code (M-3): haiku-3 left the suite red and asked permission instead of using the allowed exception; haiku-5 weakened a test (`!= 18` to `< 18`); opus-2 alone fixed the raw-SQL-error leak (+118 lines); haiku-1 changed the dashboard portfolio unprompted; sonnet-2 skipped the spec.
4. Real product defects surfaced as side effects: WAL and `busy_timeout` were never active (the DSN keys were mattn syntax), foreign keys were off, customers with history could not be deleted, a v11 database with sales would not upgrade once keys were enforced (found by the independent QA, reproduced, fixed in `fix/close-open-items`).
5. **Gemini** (read-only `agy`, everything inline): wrote a correct patch for a specified change (M-3, smallest diff); researched the DSN question mostly right but did not read the source it was allowed to read and doubted a version that exists; **missed a seeded bug** that Haiku and Sonnet found (M-2) and said CLEAN on a diff with contradictory reasoning; caught a weakened test (M-3). 3 of 9 calls returned nothing (auto-denied file read, timeouts). Verdict so far: usable as a generator and as an extra reviewer whose every finding is verified, not as the only reviewer.

## Recommendations (hypotheses to adopt only after PO approval)
- Implementer on Haiku or Sonnet when a red test defines "done"; escalate to Opus on a failed test or an open design.
- Keep QA and the review of touched tests on the strong model.
- Use Gemini as a second opinion; never as the verdict.
- Do not change `.claude/agents/*.md` yet: evidence is three tasks, n small, `effort` not equalized.

## Caveats
`effort` not equalized (defaults differ per model); model aliases assumed to be the 5.5 family; `general-purpose` agent used instead of `implementer` (which forbids editing tests); the hidden tests encode the Tech Lead's reading of open points; agent hand-backs were taken as claims and checked with the grader.

## Contents
- `graders/` : `grade.sh` (M-1), `grade2.sh` (M-2), `grade3.sh` (M-3) and the hidden tests as `.go.txt`. The scripts assume the scratch layout `arms/`, `arms2/`, `arms3/` next to `us078-base`; adapt the paths, then copy the hidden test into the arm, run, and remove it (the scripts do this).
- `patches/` : the final diff of every arm against its round base, plus the seeded-bug diff and Gemini's patch.
- `gemini/` : every prompt sent to Gemini and its answer (`no output` where the call failed).

## Branches and tags
| Ref | What |
|---|---|
| tag `pipeline-v1` (763d709) | Pipeline before the experiment; rollback point |
| `bench/us078/base` | M-1 base: dev + visible FK test (red) |
| `bench/us078/{opus-1,sonnet-1,haiku-1..5}` | M-1 arms, one commit each |
| `bench/us078/r2-base`, tag `bench-r2-base` | M-2 base: M-1 base + opus-1's FK fix + visible pragma test |
| `bench/r2/{opus-1,opus-2,sonnet-1,sonnet-2,haiku-1..5}` | M-2 arms |
| `bench/r2/seeded` | M-2 seeded-bug diff used to test reviewers (not an arm) |
| `bench/r3/base`, tag `bench-r3-base` | M-3 base: M-2 base + WAL/FK fix + visible customer-delete test |
| `bench/r3/{opus-1,opus-2,sonnet-1,sonnet-2,haiku-1..5,gemini-1}` | M-3 arms |
| `fix/close-open-items` | The production change that closes what the experiment found (US-078, R-56); not an experiment arm. Needs PO validation before merging. |
| `experiment/model-bench` | This folder and the `pipeline-bench.md` sections |

The agents' conversation transcripts are not stored here; they live in the Claude Code session log of 2026-10-12/13.
