# Agentic pipeline benchmark

> **Type:** current · **Rules over:** how the `/us` pipeline performs run by run · **Updated:** 2026-11-05

Each `/us` run adds a row to the runs table and a detail section. The orchestrator does it in step 6 of `.claude/commands/us.md`, in the same merge commit. It exists to decide with data, not impressions, whether the pipeline is worth what it costs and what to adjust. The design is in [tosdd-plan.md](tosdd-plan.md).

> **Language rule (`../CLAUDE.md` → Rule 0):** everything in this file is English. Runs 1–9 were originally written in Spanish and translated on 2026-10-01; number formats were converted (`1.547` → `1,547`, `3,9 min` → `3.9 min`). Quoted Spanish strings are UI text or user phrases.

## Current state — where to resume

> Updated at the end of each work session with the pipeline.

**Last run:** 12 (US-080, ESC/POS printing, level L; two orchestrator sessions overlapped at close). Before: 11 (US-079), 10 (US-077), 9 (US-056). No `us/*` worktree is open. `dev` is at the US-080 merge (verify with `git log`).

**SPRINT-18 (Production Stabilization):** US-075..US-080 are done in code and tests and merged to `dev` (US-080 still needs real-printer checks P-1..P-4 in `gaps.md`); US-081 (LAN multi-register concurrency) is pending. US-078 was merged outside `/us` (experiment follow-up, `bde4d9f`).

**Next US:** US-082 (SPRINT-19, fiado collection, planned 2026-11-05 ahead of US-081; Q-1/Q-2 already decided by the PO), then US-083, US-043, US-084, US-085, then US-081. Run `/us US-082 [M|L]`. Before starting, make sure no other Claude session is orchestrating the same US (see run 12).

**Pending from the PO (manual, parked 2026-10-20):** the list is in [gaps.md](gaps.md) → "Pending PO validations" (US-077, US-076, US-078, US-062, US-054, US-074, US-034).

**Pipeline changes since run 9 (applied, with the PO's approval):**
- Hooks in `.claude/settings.json` run `.claude/hooks/pos-guard.js`: pnpm only, no commit on `master`/`main`, applied migrations, `t.Skip`, CGO (blocks); `BETWEEN`, SQL concatenation, hard-coded colors, Spanish docs, missing spec (warn/ask). Documented in `CLAUDE.md`.
- `scripts/gemini-review.sh` is called by `/us`: `contract` mode before GATE 1 and `diff` mode in every QA round, in parallel with `qa`. Advisory: the orchestrator verifies every finding. Timeout 420 s with one retry; the pipeline waits at most about 10 minutes.
- Experiment M-1..M-3 (model comparison) did not change any agent's model: the recommendations (cheaper implementer when a red test defines done, QA on the strong model, Gemini never the only reviewer) wait for more data. See `experiments/model-bench/README.md` on branch `experiment/model-bench`.

**Open, no adjustment applied:** (1) the pipeline cannot verify UI in a browser (no browser tool in `qa`; run 7 used headless Edge by hand). (2) The auto-mode classifier can stop Bash and PowerShell calls (seen when force-deleting branches and in run 7). (3) Finished agents vanish from `ListAgents`, so resume-instead-of-relaunch rarely applies (every agent in run 10 was cold).

**How to resume:** in a new session, run `/us US-082` (or the next pending US) and read the PO list in `gaps.md` first.

## How it is measured

| Metric | Source | Note |
|---|---|---|
| Tokens per agent | `subagent_tokens` of each agent's notification | The **last value** of each agent is recorded. When resumed with `SendMessage`, the value grows with the accumulated context, so resumptions are not summed |
| Tool uses / duration | `tool_uses` and `duration_ms` of each notification | These **are** summed across resumptions. Duration is agent time; it does not include the PO's wait at the gates |
| Unplanned stops | The orchestrator | Times it stopped outside GATE 1 and 2 over something the contract did not resolve |
| QA rounds | `qa.md` | 1 = approved on the first try |
| Defects found | Contract (DISCREPANCIES) + findings from test-author and QA | Only real defects in the code or the docs, not scope questions |
| Roadmap items | `close` mode | `R-NN` IDs created |
| Violations | Orchestrator `git` checks | An agent touched files outside its own |

## Runs

| # | Date | US | Level | Tokens (last value per agent) | Tools | Agent min | Unplanned stops | QA rounds | Defects | Roadmap | Violations | Diff |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2026-09-23 | US-059 | L | ~440k (A 169k · T 102k · I 71k · Q 101k) | 125 | ~27.5 | 2 | 1 | 5 | 5 | 0 | +1,547 / −224, 13 files |
| 2 | 2026-09-23 | US-063 | L | ~310k (A 138k · T 71k · I 35k · Q 67k) | 111 | ~18.5 | 0 | 1 | 4 | 2 (+1 closed) | 0 | +466 / −43, 7 files |
| 3 | 2026-09-23 | US-064 | M | ~308k (A 173k = 117k+57k · I 71k · Q 64k) | 145 | ~18.0 | 0 | 1 | 1 | 0 (+1 closed) | 0 | +353 / −12, 7 files |
| 4 | 2026-09-23 | US-065 | M | ~282k (A 176k = 117k+59k · I 59k · Q 47k) | 113 | ~13.3 | 0 | 1 | 0 | 1 closed (R-45) | 1 (resume) | +217 / −20, 7 files |
| 5 | 2026-09-23 | US-060 | L | ~522k (A 154k · T 109k · I 175k · Q 85k) | 160 | ~83.8 | 1 (implementer died from output limit) | 1 | 5 | 1 (R-46) | 0 | +1,333 / −2, 9 files |
| 6 | 2026-09-24 | US-061 | M | ~708k (A 372k=149k+109k+115k · I 186k, 2 earlier attempts without telemetry · Q 150k) | 281 (incomplete, 2 implementer attempts with no `tool_uses` reported) | ~46.7 agent (+~310 min wall-clock lost in 2 hangs, see below) | 2 (implementer hung twice) | 1 | 6 | 2 new (R-47, R-48) + 1 partially closed (R-06) | 0 | +1,867 / −20, 16 files |
| 7 | 2026-09-29 | US-062 | L | ~538k (A 340k=135k+102k+75k+28k · T 65k · I 64k · Q 69k) | 168 | ~27.4 | 0 (plus 1 stop from the auto-mode classifier, unrelated to the contract) | 1 | 7 contract discrepancies (D-1..D-7) + 1 pre-existing out-of-scope finding | 7 new (R-49 to R-55) | 0 | +1,198 / −8, 9 files |
| 8 | 2026-09-30 | US-054 | L | ~191k (A 79k=51k+55k+79k · T 32k · I 45k · Q 36k) | 51 | ~9.9 | 0 (one PASSES⚠️ shown to the PO by rule, not a stop) | 1 | 6 contract discrepancies (outdated sprint, dynamic routes, `go build` without `web/out`…) | 1 new (R-62) + 1 closed (R-55) | 0 | +452 / −55, 22 files |
| 9 | 2026-09-30 | US-056 | L | ~514k (A 213k=82k+49k+83k · T 95k=82k+13k · I 114k · Q 92k=63k+29k) | 205 | ~22 | 1 (main checkout with a change in `.claude/`; plus one PASSES⚠️ shown to the PO by rule) | 1 (+ smoke 1b) | 8 contract discrepancies + 1 test defect (seed clashes with migration) | 5 new (R-63..R-67) + 1 closed (R-59) | 0 | +2,155 / −26, 26 files |
| 10 | 2026-10-20 | US-077 | L | ~218k (A 128k=63k+26k+38k · T 29k · I 30k · Q 31k) | 70 | ~6.8 | 0 (one PASSES⚠️ shown to the PO by rule, not a stop) | 1 | 4 contract discrepancies (D-1..D-4) + 3 wording fixes from Gemini | 1 new (R-73) + 1 closed (R-49) | 0 | +249 / −11, 4 files |
| 11 | about 2026-10-28 (steps 1-5; GATE 2 re-verified 2026-11-04) | US-079 | L | not recorded (lost with the earlier session) | not recorded | not recorded | not recorded | 1 | not recorded | 2 new (R-74, R-75) + 1 closed (R-35) | not recorded | +200 / −145, 32 files |
| 12 | 2026-11-05 | US-080 | L | ~535k (A 276k=76k+45k+73k+82k · T 84k · I 124k · Q 51k) | 167 | ~23.9 | 1 (two orchestrator sessions ran the close in parallel; PO closed one before merge) | 1 | 7 contract discrepancies (D-1..D-7) + 5 at close + 2 Gemini-verified MINOR | 7 new (R-76..R-82) | 1 (duplicate close by a parallel session) | +1,988 / −30, 32 files (excluding this benchmark entry) |

### Trend

| | Run 1 | Run 2 | Reading |
|---|---|---|---|
| Tokens | ~440k | ~310k (−30%) | Drops due to the size of the US and because agents were resumed instead of relaunched |
| Agent min | ~27.5 | ~18.5 | — |
| Unplanned stops | 2 | **0** | The adjustment "the analyst verifies by executing" worked |
| QA rounds | 1 | 1 | — |
| Violations | 0 | 0 | The separation between tests and code holds |

---

## Run 1 — US-059 (pilot)

**US:** extract the dashboard metrics into a range-parameterizable repository (SPRINT-14). Pure refactor. **Commit:** `cd2e590`.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract, GATE 1 adjustment, T-29 / NULL adjustment, close) | 169k | 24 + 7 + 5 + 10 = 46 | 287 + 109 + 59 + 119 s = 574 s |
| test-author | 2 (29 tests, then T-30) | 102k | 16 + 3 = 19 | 328 + 31 s = 359 s |
| implementer | 1 | 71k | 22 | 264 s |
| qa | 1 | 101k | 38 | 454 s |
| **Total** | 8 | ~440k | 125 | 1,651 s ≈ 27.5 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Defect 🔴 | D-4: aging receivables multiply the payments by the number of the customer's credit sales ($30,000 instead of $40,000) → R-04 | analyst |
| Defect 🔴 (to confirm) | D-5: UTC dates compared as text in local time → R-05 | analyst |
| Defect | D-6: inclusive `to` in `buildDateWhere` and in `ExpenseRepository.List` → R-06 | analyst |
| Defect | `Scan` error silently ignored in aging, top by margin and low stock | test-author |
| Doc defect | The sprint said 8 queries; there were 10. The sprint interface did not cover top seller or aging | analyst |
| Useless test avoided | T-15, as it stood in the contract, passed against an empty stub | test-author |
| Contract error | T-29 was supposed to be red, but the code already answered 500 | test-author |
| False doc | CLAUDE.md suggested CI runs `pnpm lint`; it does not, and there are 41 prior problems | qa |

### Unplanned stops (2)

1. **T-29 already passed:** the contract assumed a behavior the code did not have. It was reclassified as a non-regression test.
2. **`days_overdue` NULL:** the contract did not say how to treat those rows, and propagating the error broke the equivalence. The PO chose option A: exclude them explicitly in the SQL.

Both came from the Analyst **deducing** behavior instead of **executing** it.

### Operational friction

- `git worktree remove` failed on Windows because of ignored files (`node_modules`, `.next`, 565 MB). The folder had to be deleted by hand.
- gopls flags false errors inside the worktree, because it is not in the Go workspace. It is noise.
- The contract changed after red (T-30) and `/us` had no step for that.

### Adjustments applied after this run

| Adjustment | Where |
|---|---|
| Forced worktree removal + `rm -rf` of whatever remains | `.claude/commands/us.md` step 5 |
| Resume agents with `SendMessage` if the session is still alive (the analyst went from 136k to 169k tokens in 3 resumptions, against ~100k per cold start) | `us.md` conventions, `tosdd-plan.md` §2 |
| Step 2b: new test-author round if the contract changes after red | `us.md` |
| The analyst verifies **by executing** (curl or a quick test) claims about HTTP responses and errors, not only by reading | `.claude/agents/analyst.md` |
| Note about false gopls errors | `us.md` |
| Record the benchmark on every run | `us.md` step 6 |
| CLAUDE.md: CI does not run lint | `CLAUDE.md:52` |

### What to measure in run 2

- Do unplanned stops drop with the Analyst's executed verification?
- Cost of an **M** US versus an **L**.
- R-04 is a small fix: it shows whether the pipeline is overkill for small changes.

---

## Run 2 — US-063 (fix of bug D-4, from roadmap R-04)

**US:** fix the dashboard aging-receivables balance (SPRINT-14). A bug fix that changes a number visible to the owner. **Branch commits:** `180e201` → `fbccc6b` (RED) → `d999fad` → `0c5b276`, merged into a single commit on `dev`.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract + US registration, GATE 1 decisions, close); resumed with `SendMessage` | 138k | 33 + 8 + 18 = 59 | 333 + 33 + 122 s = 488 s |
| test-author | 1 | 71k | 15 | 222 s |
| implementer | 1 | 35k | 7 | 113 s |
| qa | 1 | 67k | 30 | 290 s |
| **Total** | 6 | ~310k | 111 | 1,112 s ≈ 18.5 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Defect 🔴 (real scope larger) | DISC-1: the bug not only gave wrong amounts, it also **hid debtors**. Eli owed $15,000 and did not appear because the calculation gave −$15,000 | analyst (executed) |
| Defect 🔴 | DISC-3: `TotalPortfolioBalance` subtracts credit balances ($40,000 vs $55,000) → R-44 | analyst |
| Defect (UI effect) | DISC-4: the payment modal showed the wrong debt and the row disappeared after paying. The backend fix resolves it | analyst (read) |
| Minor defect | `ORDER BY days_overdue DESC` without a tie-break with `LIMIT` → R-45 | qa |
| Debt | `CustomerRepository.List` duplicates the balance formula → R-45 | implementer + qa |

A grep search confirmed there are no other JOINs that multiply amounts.

### Unplanned stops: 0

It only went through the two gates. The Analyst executed the current behavior with a temporary test before writing the contract, and no test fell into `PASSES⚠️`.

### Operational friction

- The US came from the roadmap and did not exist in the sprint. `/us` assumes it exists (step 0.2), so the Analyst registered it in `contract` mode. That left the sprint modified in the worktree, and the orchestrator committed it separately (`docs(US-063): register US`) so the `RED` commit had only tests.
- GATE 1 left decisions that only touched §7 and §8 of the contract. The Analyst recorded them **in parallel** with the test-author, without conflict.
- The roadmap had no format for closed items; the Analyst created the "Closed" section.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| Step 0.2: if the US comes from the roadmap (R-NN), the Analyst registers it in the sprint during `contract` mode and the orchestrator commits it separately before `RED` | `.claude/commands/us.md` |
| If the GATE 1 decisions only touch §7 and §8, the test-author may run in parallel | `.claude/commands/us.md` |
| "Closed" roadmap format documented in its header | `project-docs/roadmap.md` |

### What to measure in run 3

- An **M** US (no test-author), to compare cost against L.
- Whether R-44 (another small balance fix) confirms ~300k tokens as the floor of an L US, or whether it drops further.

---

## Run 3 — US-064 (R-44, level M)

**US:** fix `TotalPortfolioBalance` so it does not subtract credit balances (bug declared at US-063's GATE 1). Small bug fix, no separate test-author (level M). **Branch:** `us/US-064` → `78fbdac` (registration) → `71a30a2` (implementation) → `a78a959` (close), merged into `dev`.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 2 (contract + US registration; close) — **launched cold, not resumed with SendMessage** | 117k + 57k = 173k | 39 + 22 = 61 | 375 + 161 s = 536 s |
| test-author | 0 (level M skips it) | — | — | — |
| implementer | 1 | 71k | 35 | 234 s |
| qa | 1 | 64k | 49 | 310 s |
| **Total** | 4 | ~308k | 145 | 1,080 s ≈ 18.0 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Doc defect | The text of R-44 in the roadmap cited figures from the seed of `TestMetricsRepo_T35` ($40,000/$55,000) that do not match what that seed produces when executed ($60,000/$75,000) — the text forgot one customer from the seed (Beto, $20,000) | analyst (executed) |

No blocking scope discrepancies. The bug itself was already characterized since US-063's GATE 1; this run only confirmed it by executing with its own fixture (25,000 with the bug, 45,000 correct) and fixed it.

### Unplanned stops: 0

Only the two gates. The §8 note (fix the R-44 figure) was informational, not blocking, and Andrey resolved it at the same GATE 1.

### Operational friction

- **Violation of the "resume instead of relaunch" convention:** the orchestrator launched a new `analyst` for `close` mode (`a7354c5de9d3b5136`) instead of resuming with `SendMessage` the contract's `analyst` (`ac2bf2bf927d46ac4`), which was still alive in the session. It cost an extra cold start (~57k tokens) instead of a cheaper resumption. It is an orchestrator error, not an agent's — fix in the next run.
- Before step 0, pre-existing dirty gitlinks were found in `.claude/worktrees/*` (leftovers of Claude Code worktrees, not the POS's) and an unrelated staged `rewrite_links.py`. The step 0.1 rule requires stopping; Andrey was consulted and authorized going on. Neither was touched.
- Level **M** worked without friction: the implementer wrote tests and implementation in a single commit, following the contract's test table just as detailed as in level L; QA found no tests weakened for lack of the test-author.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| Before launching any agent, the orchestrator explicitly checks whether there is a live `agentId` in the session for that role and that US before deciding `Agent` vs `SendMessage` (not just "remembering it", but an explicit check in the relevant step) | `.claude/commands/us.md` steps 1 and 5 |

### What to measure in run 4

- If the resumption friction is fixed, does the analyst cost drop in a similar close (2 invocations → 1 resumed)?
- Level M still gives ~300k total tokens despite skipping the test-author: the saving of skipping an agent (~65-100k) is offset by more analyst cost due to the resumption error. Repeat level M without that error to isolate the true cost floor of a small US.

---

## Run 4 — US-065 (R-45, level M)

**US:** hygiene of the balance formula detected in US-063's QA — `CustomerRepository.List` stops textually duplicating the balance `FROM` (reuses `customerBalanceExpr`/`customerBalanceFrom`) and `AgingReceivables` gains a stable tie-break (`balance DESC, name ASC`) when `days_overdue` ties. Small bug fix + hygiene refactor, level M. **Branch:** `us/R-45` → `ee4f334` (implementation) → `0c3cf4a` (close), merged into `dev`. The analyst registered the US in the sprint as **US-065** during `contract` mode (R-45 came from the roadmap, with no entry of its own in the sprint).

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 2 (contract + US registration; close) — **again launched cold, not resumed with SendMessage** | 117k + 59k = 176k | 41 + 19 = 60 | 360 + 123 s = 483 s |
| test-author | 0 (level M skips it) | — | — | — |
| implementer | 1 | 59k | 27 | 165 s |
| qa | 1 | 47k | 26 | 148 s |
| **Total** | 4 | ~282k | 113 | 796 s ≈ 13.3 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Minor defect | Redundant wording in `spec/domain/fiado.md:38` — the new delta sentence repeated "orders by `days_overdue` descending" twice in the same sentence | qa |

No scope discrepancies: the text of `roadmap.md:48` described both points accurately (confirmed with grep and, in the tie-break case, by executing a temporary fixture that showed the real bug — with a `LIMIT` smaller than the tied rows, the customer with the largest debt could be left out of the cut by insertion order).

### Unplanned stops: 0

Only the two gates. The only friction was not a contract stop but an orchestrator error (see below).

### Operational friction

- **The "resume instead of relaunch" violation of run 3 was repeated, despite being documented as a pending adjustment:** the orchestrator launched a new `analyst` (`ab24d286fefcf0abb`) for `close` mode instead of resuming with `SendMessage` the contract's `analyst` (`ae7f21565663133bd`), which was still alive in the session. The adjustment proposed in run 3 ("explicit live-`agentId` check before `Agent` vs `SendMessage`") was never applied to `us.md` — it stayed as a proposal pending the PO's "yes" and nobody raised it again before run 4. Same extra cold-start cost (~59k tokens) as in run 3.
- Same pre-existing dirty state as in run 3 (`rewrite_links.py` staged in the main checkout, modified gitlinks in `.claude/worktrees/*`), already authorized by Andrey then; it was left intact again, without asking again because it had not changed since the previous run.
- Level **M** again worked without contract/test friction: the implementer wrote the 3 new tests (T-39, T-40, T-41) plus the code changes in a single commit, without QA finding weak tests for lack of the test-author.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| The run 3 one is still not applied and is still needed: before launching any agent, the orchestrator must explicitly list (not just "remember") the live `agentId`s of the session for that US before deciding `Agent` vs `SendMessage`. Two runs in a row with the same error suggest a mechanical step is needed in `us.md`, not a convention note | `.claude/commands/us.md` steps 1 and 5 |

### What to measure in run 5

- If the adjustment above is applied before running: does total cost drop from ~280k to something closer to ~220k (analyst in a single resumed invocation)?
- The floor of an M US without resumption errors is still not measured cleanly — 2 runs in a row with the same process defect inflating analyst cost by ~50-60k tokens.

---

## Run 5 — US-060 (T1/T2 intent engine, level L)

**US:** `internal/assistant/` package from scratch — normalization, 11 regex intents (T1) and fuzzy matching with Levenshtein (T2), relative period extraction (today/yesterday/week/month/year + weekdays with typos). Pure Go, no DB or HTTP; first piece of a chain of 3 (US-060 → US-061 → US-062). Level **L**, first run with the mechanical resume adjustment already applied. **Branch:** `us/US-060` → `4f0ba6b` (RED, 31 tests) → `b36f108` (implementation) → `4a20d52` (close), merged into `dev` by squash.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (initial contract; adjustment for 2 GATE 1 decisions; add T-30/T-31 to the contract; close) — **all 4 resumed with `SendMessage`, zero cold starts after the first** | 73k → 110k → 115k → 154k (last: 154k) | 23+34+5+20 = 82 | 303+130+29+155 = 617 s |
| test-author | 2 (29 initial tests; add T-30/T-31) — resumed with `SendMessage` | 98k → 109k (last: 109k) | 18+6 = 24 | 634+77 = 711 s |
| implementer | 2 (start; **resumed after a crash**) | failed with no usage reported → 175k (last: 175k) | 30 (only the invocation that finished) | 3,417 s (includes the cut turn, resumed in the same conversation) |
| qa | 1 | 85k | 24 | 285 s |
| **Total** | 9 | ~522k (154+109+175+85) | 160 | 5,029 s ≈ 83.8 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Sprint defect 🔴 (verified by executing Levenshtein outside the repo) | The `levenshtein<=1` threshold that the sprint itself uses as an example does not reach its own cases: `"aller"→"ayer"` and `"kien"→"quien"` have a real distance of 2, not 1 | analyst |
| Contract ambiguity | "Time-candidate word that does not resolve" (→ `Unresolved`) had no defined candidacy criterion; "jueves" (Thursday) was not in any vocabulary of the system | analyst |
| Coverage gap | AC-13 ("mes pasado") and AC-14 ("este año") were in §3 of the contract but without their own row in the §4 test table — nobody was going to test them | test-author |
| Minor | `ambiguousSeed` deliberately shared between the T1 banks of SALES and PROFIT strains the letter of AC-25 (non-overlap); it does not block, it is tension between two ACs of the contract itself | qa |
| Minor | The weekday vocabulary lacks the minimum-length guard that does apply to the intents fuzzy match — no false positive today, risk if the vocabulary grows → **R-46** | qa |

### Unplanned stops (1)

**The implementer was cut mid-turn:** `API Error: Claude's response exceeded the 64000 output token maximum`, probably generating a whole file (with phrase bank and synonym map) in a single block of text. Zero production code had been written (`git status` clean, tests intact) — nothing had to be reverted. It was resumed with `SendMessage` explicitly asking it to split the work into smaller files/edits, and it finished without being cut again. The agent's total duration (3,417 s) includes the lost turn, so it is not comparable 1:1 with implementers of earlier runs without this incident.

### Operational friction

- The mechanical resume adjustment (applied before this run, see `us.md`) worked: the 3 agents with more than one invocation (analyst ×4, test-author ×2, implementer ×2) were always resumed with `SendMessage`, never with an unnecessary cold `Agent` — first run without that violation since run 2.
- When applying the fix to `us.md`, the orchestrator accidentally committed `rewrite_links.py` (an unrelated file pre-staged from before the session) in the same commit by using `git commit -m` without excluding it from the index. It was corrected with a separate commit (`git rm --cached`) before touching the US worktree. Lesson: check `git status` after a targeted `git add <file>`, do not assume the index only has what was just added.
- Level **L** with test-author worked as expected: it found a real coverage gap (AC-13/AC-14 without a test) that at level M would probably have gone unnoticed until QA or later.
- The implementer's crash from the output limit is new in the run history — it had not happened in the previous 4, possibly because of the size of this US's phrase/synonym bank compared with earlier small fixes.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| When the implementer is about to write a large file with extensive data tables/banks (regex, synonyms, seed phrases), tell it upfront to split the writing into several `Write`/`Edit` calls instead of a whole file in a single block, to avoid the 64k output-token limit per turn | `.claude/agents/implementer.md` or `us.md` step 3 |
| Before any `git commit` in the main checkout outside the US worktree, explicitly check `git status` so as not to drag along unrelated pre-staged files | `us.md`, general note for the orchestrator |

### What to measure in run 6

- Does the warning to the implementer about splitting large files avoid the crash in the next US with a similar data volume (US-061 or US-062)?
- US-061 depends on US-059 (metrics) and US-060 (engine), both already closed — a good case to measure whether an "integration" US (service + endpoint + log, with DB and HTTP) costs more or less than this run's pure engine.

## Run 6 — US-061 (service, `/ai/ask` endpoint and log, level M)

**US:** glues US-059 (metrics) to US-060 (intent engine) — new service (`internal/service/assistant.go`), log repo (`internal/repository/assistant_queries.go`), handler+endpoint (`internal/handler/assistant.go`), migration 15. Level **M**, no test-author. **Branch:** `us/US-061` → `1454fd6` (implementation+tests, no separate RED) → `f793074` (close), merged into `dev` by squash.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (initial contract; correction with the 6 GATE 1 decisions — the orchestrator investigated D-2 on its own before relaunching; close) — **all 3 cold, `ListAgents` never showed one alive** | 149k → 109k → 115k (last: 115k) | 41+51+33 = 125 | 374+522+275 = 1,171 s |
| implementer | 3 (start; resumed cold after an output-limit crash at 600 s; resumed cold again after a second 5 h hang killed by hand) | failed with no usage reported → failed with no usage reported → 186k (last: 186k, the only one with telemetry) | 79 (only the invocation that finished) | 846 s reported (agent) + ~310 min of wall-clock with no telemetry in the 2 failed attempts |
| qa | 1 | 150k | 77 | 785 s |
| **Total** | 7 | ~708k (372+186+150, underestimated — 2 implementer attempts without `subagent_tokens`) | 281 (underestimated, same reason) | 2,802 s of telemetry ≈ 46.7 min + ~310 min of unmeasured wall-clock |

### What it found

| Type | Finding | Who |
|---|---|---|
| Real defect 🔴 (blocking, roadmap R-06) | `buildDateWhere` (report.go:192) inclusive on `to` — the PO demanded fixing it globally instead of accepting the risk; the fix also uncovered the same latent bug in `dashboard.go:142-159` | analyst, GATE 1 decision |
| Real gap | `ReportRepository.TopProducts` orders by revenue, but the TOP_PRODUCTS spec requires the "#1 by units" contrast — investigated by the orchestrator (not the analyst) after the PO's instruction "find out, look it up"; resolved with a new method (`TopSellers`) so as not to break the existing reports endpoint, which does depend on the revenue ordering | orchestrator + analyst |
| Real risk, caught before production | The engine never returns `Tier="none"`; the migration requires `CHECK tier IN (...,'none')`. Without mapping `""→"none"` before the INSERT, 100% of `UNKNOWN` queries —exactly the ones the log exists to capture— would have broken the INSERT | analyst, in the initial contract (AC-15) |
| Pre-existing defect in `dev`, not this US's | Grouped `SalesSummary` collapses into a single bucket when `created_at` is a hand-inserted literal `time.Time` (the project's fixture pattern); it does not trigger today because `sale.go` always uses `DEFAULT CURRENT_TIMESTAMP`. QA reproduced it against `dev` without this US's diff → **R-48** | qa |
| Minor, accepted | 4 new training phrases in `engine.go` not listed in the original contract — QA verified that reverting them breaks 4/7 `Period.Label` cases and that they break nothing in US-060 | qa |
| Deferred to roadmap | R-06 only covers `report.go`; `ExpenseRepository.List` (expense.go:73) still has an inclusive `to`, no US triggers it yet → partial close of R-06 | analyst, close mode |
| Deferred to roadmap | `frontend/app/reports/page.tsx:98` builds `to` as 23:59:59 instead of start of the next day; after the `buildDateWhere` fix a 1-second window is left uncovered → **R-47** | analyst, contract |

### Unplanned stops (2)

**Implementer attempt 1:** cut by the same output limit that had already occurred in run 5 (large file in a single block), this time detected by the automatic watchdog at 600 s without progress. The orchestrator verified `git status`/`go build` before relaunching: nothing broken, it was resumed cold with an explicit instruction to work in small blocks.

**Implementer attempt 2, the new finding of this run:** the same "split the work into small blocks" warning made the agent progress VERY slowly instead of being cut — it ran **5 real hours** (the PO asked "how the hell has it been 5 hours?") without the 600 s watchdog detecting it as hung, because technically it kept making tool calls, only at an extremely slow pace (final diff of that round: +15 lines over the previous attempt). The orchestrator compared real progress (`git diff --stat` between the post-crash-1 state and the 5 h state) to confirm it was not perception, killed the process with `TaskStop`, verified that the half-edited file (`engine.go`) had not been left broken (`go build` passed anyway), and relaunched a third time with deliberately narrowed scope (only the 2 missing test files, with the instruction to run `go test` of those packages after each block instead of at the end). That third round did finish, in 846 s.

### Operational friction

- The run 5 adjustment ("warn it to split large files") **is not enough on its own** against implementer hangs — it may avoid the output crash (it did) while at the same time producing a much worse silent hang (extreme slowness undetected by the no-progress time watchdog).
- The implementer's 2 failed attempts did not generate a notification with `subagent_tokens`/`tool_uses` — this run's benchmark underestimates the real token and tool cost of those 2 attempts, it only has that of the round that finished.
- The orchestrator has no own mechanism for active monitoring outside notifications — it tried `ScheduleWakeup` to check in 20 min and canceled it immediately on noticing that tool is specific to `/loop` and does not apply to this session. The attempt-2 check (at 5 h) was reactive, triggered by the PO's complaint, not by the orchestrator's own monitoring.
- Investigating D-2 (orchestrator instead of analyst) worked well: the PO's instruction was "look it up yourself", and verifying real code (`grep` of `TopProducts`/`TopByMargin`, who else calls `TopProducts`) before relaunching the analyst avoided an extra round of questions.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| Give the implementer an explicit wall-clock limit (e.g. 20-25 min) to complete a declared block of work, not only a progress watchdog; if exceeded, the orchestrator checks `git diff --stat` against the last known checkpoint and decides to kill/continue with data, not blindly | `.claude/agents/implementer.md`, `us.md` step 3 |
| When the implementer starts a "complete what is missing" round (not a full contract from scratch), ask it for an explicit list of the pending tests/files BEFORE writing code, so the orchestrator can measure real progress against a checklist instead of only `git diff --stat` | `.claude/agents/implementer.md` |
| Do not use `ScheduleWakeup` outside a `/loop` session — confirmed in this run that it is specific to that skill and must be canceled if called by mistake | operational note for the orchestrator, no file change required |

### What to measure in run 7

- Does the wall-clock limit (if the PO approves it) reduce the implementer's silent hangs, or is the problem specific to "complete someone else's work" rounds like this run's?
- US-062 (chat widget, frontend) is the first US of this block that touches React/Next.js — a good case to measure whether the hang pattern is a problem of the `implementer` agent in general or something specific to Go/backend.

## Run 7 — US-062 (assistant chat widget, level L)

**US:** floating chat widget in the authenticated app that consumes `POST /ai/ask` (US-061): `frontend/lib/assistant.ts` (pure logic + module store), `AssistantWidget.tsx` (render only), mounting in `AppLayout.tsx`, plus a backend change by PO decision at GATE 1 (suggestion chips as full phrases, `internal/service/assistant.go`). First frontend US of the pipeline. **Branch:** `us/US-062` → `4ef5a34` (RED) → `378c6d0` (implementation) → `1b32e38` (close) → `740d067` (R-55), merged into `dev` by squash.

### Per agent

| Agent | Invocations | Tokens (last) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; adjustment with the GATE 1 decisions; close; R-55 registration) — **all 4 cold, `ListAgents` never showed one alive** | 135k → 102k → 75k → 28k | 48+26+26+12 = 112 | 652+193+175+50 = 1,070 s |
| test-author | 1 | 65k | 16 | 135 s |
| implementer | 1 | 64k | 17 | 236 s |
| qa | 1 | 69k | 23 | 206 s |
| **Total** | 7 | ~538k (340+65+64+69) | 168 | 1,646 s ≈ 27.4 min |

### What it found

| Type | Finding | Who |
|---|---|---|
| Real defect 🔴 | The chips the backend emitted did not work when resent (`hoy`, `ayer`, `cuánto vendiste`… returned `unknown`). The analyst proved it by executing the engine; the PO chose option A (change the suggestions in `assistant.go`, code of an already-closed US-061) → round-trip T-12 and T-13 | analyst (D-2) |
| False sprint claim | "`AppLayout` stays mounted on navigation": each of the 13 pages mounts its own `AppLayout`; a `useState` would lose the history → module store (D-1) | analyst |
| Other contract discrepancies | D-3..D-7: `formatCOP` barely applies, the lists must be parsed from the `text`, `isMobile` is not reusable, `interactiveWidget` absent (keyboard not testable), `#000` over `--accent` | analyst |
| Pre-existing finding, out of scope 🔴 | `web/static.go:23` opens the path with a leading slash, `io/fs` rejects it, and the binary serves `index.html` for every URL (including the `.js` files): the embedded UI is left blank. Also `frontend/lib/api.ts:3` fixes `http://localhost:8080/api` and CORS only accepts `:3000`. **Automated QA did not see it; manual verification with a browser found it** → **R-55** | orchestrator, with headless Edge (Playwright) |
| QA minors, accepted | `boxShadow` with fixed `rgba`; `catch {}` without log (→ R-54); `useIsMobile` starts at `false` | qa |
| Deferred to roadmap | R-49 (401 → `/login`), R-50 (shared layout), R-51 (component tests), R-52 (incomplete periods per intent), R-53 (ambiguous chips lose the period), R-54 (`catch {}` without log), R-55 (embedded frontend) | analyst |

### Unplanned stops: 0 from the contract

No stop over something the contract did not resolve, no implementer hang. There was an interruption external to the pipeline: the auto-mode classifier returned errors on Bash and PowerShell during close (~10 attempts) and the pipeline stalled at the `close` checkpoint until it came back on its own. With read-only tools (Grep/Read) the content of the docs could be verified, but not `git status`.

### Manual verification with a browser

The `qa` agent has no browser tool and marked M-1..M-9 as pending (correct). The orchestrator did not try until the PO asked why it had not been done. It was then done with `playwright-core` installed outside the repo (in the job's temp directory) and the installed Edge: 23 checks, 21 OK and 2 false failures from the script's own errors (case-sensitive regex, incomplete data seed). It covered M-1..M-6, M-8, M-9, P-1 option A, P-3, greeting, load and focus, history on SPA navigation and F5, no storage, light/dark theme, 320 px and cashier role. Left unverified: the keyboard on a real phone (AC-20) and the overlap with the Toast.

Real cost of setting up the environment: the first pass was empty without knowing it (pages came out blank because of R-55 and "0 buttons" passed trivially); an assertion that "the page rendered text" had to be added for the M-3 check to be worth anything. Compiling with `-overlay` to change the port (8080 was taken by another PO process), rebuilding the frontend with `NEXT_PUBLIC_API_URL=/api` and a Node server with its own proxy, because of R-55 and the CORS to `:3000`.

### Operational friction

- A single `implementer.md` commit (`37e5c10`) plus a prompt with a warning to chunk files and report by blocks coincided with the first run without implementer hangs in three. The improvement cannot be attributed to either one.
- The analyst was launched cold all 4 times: `ListAgents` showed no live instance each time it was queried (finished agents disappear from the list shortly after delivering). The mechanical check was complied with, but in practice there is now never a possible resumption unless the orchestrator writes to the agent before it finishes.
- Git Bash rewrites values like `NEXT_PUBLIC_API_URL=/api` into `C:/Program Files/Git/api` and mangles accents in inline `curl -d`; use `MSYS_NO_PATHCONV=1` and `--data-binary @file`.
- Unrelated uncommitted work appeared in the main checkout during the session (an in-progress fix of `web/static.go`, a draft `SPRINT-15-Asistente-Continuacion.md`, `web/out/.gitkeep` deleted by a build). The squash merge did not touch them because they did not overlap with the diff and the commit was made without `-a`.

### Proposed adjustments (pending the PO's "yes")

| Adjustment | Where |
|---|---|
| For USs with UI, add to step 4 a browser verification (Playwright + Edge, or the built-in browser tool if available) done by the orchestrator or an agent with that tool, and require that "X does not appear" checks include an assertion that the page actually rendered | `.claude/commands/us.md`, contract §4 (M-n) |
| Document in `project-docs/` (or as a project skill via `/run-skill-generator`) how to start the binary with a test port/DB and the frontend with a relative API, so as not to rebuild it each time; no longer needed once R-55 is closed | `project-docs/` or `.claude/skills/` |
| Clarify in `us.md` that, after delivering, an agent disappears from `ListAgents`, and that resumption is only possible if the orchestrator sends `SendMessage` before that; or change the check to "use the agentId of the previous invocation" | `.claude/commands/us.md` |

### What to measure in run 8

- Does the implementer finish again without hangs in a US of similar size (frontend + backend)? With one more sample the effect of the `implementer.md` adjustment can be tried to be separated from the effect of the warning in the prompt.
- If the PO approves browser verification, how much time and tokens it adds per US with UI.
- A US that starts with `ListAgents` showing something alive, to test that the resumption path still works.

## Run 8 — US-054 (LAN access + serving the frontend from the binary, level L)

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract, GATE 1 adjustment, close; the last two via `SendMessage`) | 79k | 23 | ~4.2 min |
| test-author | 1 | 32k | 6 | ~1.1 min |
| implementer | 1 | 45k | 12 | ~2.8 min |
| qa | 1 | 36k | 10 | ~1.8 min |

**What it found.** The contract reproduced with an `fstest.MapFS` the real behavior of `FileServer` and detected what the sprint did not say: dynamic routes (`customers/[id]`, `catalogo/[token]`) exported as a single `_.html`, `go build` broken on a clean checkout because of `//go:embed all:out` without a tracked `.gitkeep`, and `lan_access_enabled` already existing. All decisions (P-1..P-8) were resolved at a single GATE 1. QA: 0 BLOCKER, 3 MINOR; none changed code.

**Orchestrator verification.** Before the merge the binary was built with `web/out` populated and checked by `curl`: `/login`, `/customers/5` and `/catalogo/abc` → HTML; JS chunk → `text/javascript`; `/_next/nope.js` → 404; `/api/setup/status` → 200. This closes what QA left untested (its smoke ran with `web/out` empty). Still unverified in a real browser: T-12 and AC-16 / T-M.

**Unplanned stops:** 0. A `PASSES⚠️` case in T-11 (regression guard) was shown to the PO by rule.

**Operational friction.**
- `ListAgents` did show the analyst alive after delivering (unlike run 7), and resumption by `SendMessage` worked twice. The agent shows in the list for the duration of the session; it disappeared in run 7 for another, unidentified cause.
- The implementer wrote `servidor.md` (spec) on its own; the analyst verified it in `close`. It is an edge of the role split, no harm.
- Notably cheaper run than the previous ones (~191k vs 282k–708k): narrow contract, no QA rounds and no hangs. A smaller US than US-062/US-061; do not conclude yet.

### What to measure in run 9
- If browser verification (pending adjustment from run 7) is approved, how much it adds per US with UI.
- Whether the implementer again finishes without hangs.

## Run 9 — US-056 (order per table, level L)

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract, GATE 1 adjustment, close; all three cold) | 83k (82k + 49k + 83k across the three) | 112 | ~7.0 min |
| test-author | 2 (red; seed correction) | 13k (82k + 13k) | 16 | ~4.0 min |
| implementer | 1 | 114k | 44 | ~6.6 min |
| qa | 2 (review; smoke 1b) | 29k (63k + 29k) | 33 | ~4.4 min |

**What it found.** The contract discovered by executing: the sprint took for granted the existence of hiding carts with `restaurant_mode` (it does not exist), PATCH did not pass CORS (`middleware.go:97`), `/mesas/[orderId]` needed `generateStaticParams` and `dynamicRoutes`, and the FKs are not enforced, so an atomic INSERT was needed (R-59 closed without a migration). Seven PO questions were resolved at a single GATE 1 plus one re-ask (Q-6, about what polling was) and Q-8. QA: 0 BLOCKER, 4 MINOR in the review and 1 MINOR in the smoke; none changed code.

**Test defect.** The test-author's seed `INSERT INTO users (id=1…)` clashed with the migration (`migrate.go:252`, `INSERT OR IGNORE`), so T-3, T-14 and T-15 could not pass. The implementer detected it, not the test-author. It cost an extra test-author invocation and a new `RED` commit (`1bcdd65`).

**Unplanned stops:** 1. The main checkout had a modified gitlink under `.claude/worktrees/`; the PO said to go on. Also, a `PASSES⚠️` (T-14) was shown to the PO by rule.

**Operational friction.**
- `ListAgents` showed no finished agent of the same role at any step; all invocations were cold.
- The smoke was skipped on the first pass because port 8080 is fixed (`cmd/pos/main.go:74-81`) and was taken by the PO; it was repeated as round 1b when they freed it.
- `go run scripts/copy-frontend.go` deletes `web/out/.gitkeep` in the worktree; QA restored it with `git checkout`.
- `TestDashboardToday_T20_Golden` fails by date in `dev` too (R-67).

### What to measure in run 10
- If the test-author runs its tests against the real migrations, how many test rounds are avoided.
- Cost of the separate smoke when the port is taken.

## Run 10 — US-077 (redirect to /login on 401, level L; first run with Gemini as second reviewer)

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract, GATE 1 wording adjustment, close; all cold, `ListAgents` showed none alive) | 128k (63k + 26k + 38k) | 48 | ~2.4 min |
| test-author | 1 | 29k | 6 | ~0.7 min |
| implementer | 1 | 30k | 9 | ~1.6 min |
| qa | 1 | 31k | 7 | ~2.2 min |

**What it found.** The sprint omitted the wrong-password 401 on `POST /auth/login` (a redirect there would reload the page and swallow the error); there is no `logout` function, the primitive is `clearAuth()` (`frontend/lib/auth.ts:36`); `api.ts` called `res.json()` before checking `res.ok`, so a non-JSON 401 body would have thrown a `SyntaxError`. QA: 0 BLOCKER, 3 MINOR accepted without items.

**Unplanned stops:** 0. Seven tests passed before the implementation (negative and regression guards, T-2, T-4..T-7, T-10); shown to the PO by rule, accepted as regression guards.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~5 min | 4 | 3 (AC-1 vs AC-4 path set, AC-2 body read ambiguity, T-8 conditional) | 1 (asked for a spec update; the contract had grepped `spec/` and no rule changed) | n/a (no QA on contracts); 3 wording fixes applied by the analyst |
| diff, round 1 | TIMEOUT at 300 s | 5 min | 0 | 0 | 0 | none (never answered) |

The diff packet was 22 KB, smaller than the 57 KB packet that answered in 3m23s during script testing, so latency varies and size is not the cause. The script does not retry on TIMEOUT (it retries only on empty output).

**Operational friction.**
- `pnpm install` was needed in the new worktree before any vitest run; the contract was written without it, so nothing was executed during contract mode.
- The pos-guard hook is active in this session (first run with it); no block fired.
- `TestDashboardToday_T20_Golden` still fails on `dev` (R-67); `pnpm lint` is 48 problems on `dev` and the worktree.

### What to measure in run 11
- Gemini diff review with a longer timeout and one retry on TIMEOUT: does it finish, and does it find anything QA missed?
- Whether the contract-mode review keeps paying off (3 of 4 findings valid) or only catches wording.

## Run 11 — US-079 (lint 48 -> 0, R-35, level L)

Steps 1-5 (contract through close) ran in an earlier session (about 2026-10-28). The orchestrator of this session did not launch the agents: it resumed at GATE 2 and re-verified the worktree.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | not recorded | not recorded | not recorded | not recorded |
| test-author | not recorded | not recorded | not recorded | not recorded |
| implementer | not recorded | not recorded | not recorded | not recorded |
| qa | 1 round | not recorded | not recorded | not recorded |

Per-agent telemetry was lost with the earlier session; the transcripts in `~/.claude/projects/<project>/*.jsonl` may still hold it.

**What it found.** QA round 1: OK, 0 BLOCKER, 3 MINOR (latent drawer key; `data-theme` effect re-run; throwing `localStorage` -> R-75). Contract defects: not recorded.

**Result.** Diff 32 files, +200 / −145. `pnpm lint` 48 -> 0. New tests `theme.test.ts` and `breakpoint.test.ts` (frontend tests 63 -> 71). CI now runs `pnpm lint`. New roadmap items R-74 and R-75; R-35 closed. Manual PO checks M-1..M-3 parked in `gaps.md`.

**Re-verification at GATE 2 (this session).** `go build` ok; `pnpm lint` 0 problems; `pnpm test` 71 passed; `go test` fails only `TestDashboardToday_T20_Golden`, a stale test (R-67 was fixed on `dev` after the worktree was cut).

**Unplanned stops:** not recorded. QA rounds: 1. Violations: not recorded.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|
| contract | OK (`changes/US-079/gemini-contract.md`) | 10 (AMBIGUOUS/MISSING_CASE/UNTESTABLE, e.g. AC-5 fallback for a throwing `localStorage`, AC-6 SSR behavior of `useIsMobile`, tests T-6/T-7 conditional on extracted pure functions, Q-2 CI lint vs scope) | not recorded | not recorded | not recorded |
| diff, round 1 | clean per the sprint doc | 0 | n/a | n/a | not verified by the orchestrator beyond QA |

**Operational friction.**
- The worktree was cut from a `dev` older than the R-67 fix, so `go test` shows a stale failure unrelated to the US.
- The session was interrupted between close and GATE 2, so there was no live agent to resume.

## Run 12 — US-080 (ESC/POS direct printing, level L)

Steps 0-4 ran in session A (the original orchestrator). Session B was opened with "follow pipeline" while A was still alive, found QA round 1 OK and no Gemini diff output, and resumed at step 5. Both sessions then ran Gemini diff round 1 and the analyst close in parallel on the same worktree. A committed a follow-up (`2685c0a`, spec anchors) on top of B's close commit (`5c29491`); nothing conflicted. B noticed the overlap while collecting telemetry from A's transcript and stopped before merging; the PO closed A and B merged.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; contract update after GATE 1; close in A; close in B, cold, `ListAgents` showed none alive) | 276k (75.9k + 44.9k + 72.6k + 82.3k) | 90 | ~9.8 min |
| test-author | 1 | 84.0k | 24 | ~5.9 min |
| implementer | 1 | 124.1k | 41 | ~5.7 min |
| qa | 1 round | 51.0k | 12 | ~2.5 min |

The duplicate close (72.6k or 82.3k, ~2-4 min) is pure waste caused by the overlap.

**What it found.** Contract: 7 discrepancies sprint vs code (D-1 no reprint flow exists; D-2 chi route shape; others in the contract). QA round 1: OK, 0 BLOCKER, 4 MINOR (stale `print_method` on other LAN devices -> R-80; `WindowsTransport.Send` ignores `ctx` -> R-81; 409 before 404 in browser mode -> R-82; spec delta pending, done at close). Close: 5 discrepancies documented in the spec (a `PUT /setup` without printer fields resets to browser; printed ticket has more lines than AC-14; auto-print skipped if loading the ticket fails; roadmap table column mismatch on R-74/R-75; Q-10 still open). QA's smoke test with a Node TCP listener confirmed the ESC/POS byte stream (init, CP858, accents, feed, partial cut). The winspool path was only cross-compiled, never executed.

**Unplanned stops:** 1 — overlap of two orchestrator sessions on the same US. QA rounds: 1. Violations: 1 (duplicate close). QA could not stop its own smoke server (permission denied); the port was free by the time B checked.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~2 min | 12 (1 RULE_CONFLICT, 6 MISSING_CASE, 4 AMBIGUOUS, 1 UNTESTABLE) | verified in session A; count not recorded | the RULE_CONFLICT (Spanish file name `impresion.md`) is false: every `spec/domain/` file has a Spanish name, content is English | n/a |
| diff, round 1 (A's run, 01:33Z) | OK | not recorded | 2 (MINOR test-print button hidden in browser mode vs AC-1; "no spec update") | 1 (button hidden, `settings/page.tsx:543` vs `contract.md:57`; accepted by the PO) | 1 (spec: input predated the close commit) | yes, the AC-1 deviation |
| diff, round 1 (B's run, 01:38Z) | OK | ~6 min | 2 (BLOCKER "no spec update"; MINOR mixed-payment lines before items, `printer.go:262`) | 1 (order deviates from AC-14's literal wording but mirrors `TicketPreview.tsx:167`, which AC-14 also requires; not a defect) | 1 (spec BLOCKER: input predated the close commit) | yes, the AC-14 wording conflict |

Gemini's "missing spec update" is a structural false positive: the diff review runs in step 4, before the analyst writes the spec in step 5. It appeared in both diff runs.

**Operational friction.**
- Two live sessions orchestrated the same US. Nothing in the pipeline detects this: `ListAgents` showed the peer session but not its subagents, and the worktree looked idle.
- The second Gemini diff run overwrote the first one's output file (`gemini-diff-round-1.md`).
- `data.db` from QA's smoke test stays in the worktree root (gitignored).

### Proposed adjustments (not applied; need the PO's yes)
1. At step 0, if `us/$ID` already exists, also check `ListAgents` for peer sessions and ask the PO whether another session owns the US, before resuming.
2. Tell `gemini-review.sh diff` (or its prompt) that the spec delta is written at close, to stop the recurring "no spec update" false positive.

### What to measure in run 13
- Whether Gemini diff keeps catching AC-wording deviations QA marks OK (2 of 2 runs here).
- Gemini contract findings: record verified/false counts at GATE 1.

## Experiment M-1 — model comparison on US-078 (real FK enforcement)

Not a `/us` run. Same task, same base commit (`bench/us078/base`, 6f3ff47), same prompt, one git worktree per arm (`bench/us078/<arm>`), rollback tag `pipeline-v1`. Agent: `general-purpose` (not `implementer`, which forbids editing tests). Grader: 6 hidden tests plus per-test failure listing, no-skip and no-removed-assertion checks.

| Arm | Hidden tests | Own failures | Skips / removed asserts | Diff | Tokens | Tools | Duration |
|---|---|---|---|---|---|---|---|
| opus-1 | 6/6 | 0 | 0 / 0 | +13 −1 | 65,455 | 18 | 120 s |
| sonnet-1 | 6/6 | 0 | 0 / 0 | +16 −1 | 65,426 | 13 | 109 s |
| haiku-1 | 6/6 | 0 | 0 / 0 | +15 −1 | 90,806 | 27 | 161 s |
| haiku-2 | 6/6 | 0 | 0 / 0 | +17 −1 | 93,812 | 32 | 200 s |
| haiku-3 | 6/6 | 0 | 0 / 0 | +17 −1 | 95,896 | 28 | 250 s |
| haiku-4 | 6/6 | 0 | 0 / 0 | +18 −1 | 82,583 | 19 | 125 s |
| haiku-5 | 6/6 | 0 | 0 / 0 | +25 −1 | 88,595 | 29 | 194 s |

**Result.** All 7 arms made the same change (DSN `_foreign_keys=on` → `_pragma=foreign_keys(1)`, plus the same 4 fixture seeds). The task did not discriminate between models. Haiku matched Opus/Sonnet on quality but used more tokens and tool calls. Best-of-N never mattered because every arm passed.

**Cost.** Only a combined token total per agent is available (no input/output split), so cost is a bound at list prices: Opus $0.26–$1.31, Sonnet $0.13–$0.65, one Haiku $0.01–$0.05.

**Gemini (Antigravity `agy`, read-only plan mode, full context inline).** Review of the haiku-5 diff: CLEAN, agreeing with the grader. Earlier test: 2 of 5 findings were false positives because the diff was truncated. Treated as a suspect generator, never as the judge.

**Side findings (verified).** The DSN options `_journal_mode=WAL` and `_busy_timeout=5000` are mattn syntax, ignored by `modernc.org/sqlite`: a fresh DB reports `journal_mode=delete busy_timeout=0` (probable cause of the pre-existing flaky `TestOrder_T3_ConcurrentOpenSingleWinner`). Physical `DELETE` remains in `internal/repository/customer.go:104`, `supplier.go:89`, `category.go:118`.

**Caveats.** n=1 for Opus and Sonnet; `effort` not equalized (defaults differ per model); aliases `haiku`/`sonnet`/`opus` assumed to be the 5.5 family, unverified; agent changes are uncommitted in the worktrees.

## Experiment M-2 — model comparison on US-078b (WAL + busy_timeout) and Gemini tests

Not a `/us` run. Base `bench/us078/r2-base` (a108316 = M-1 base + opus-1's FK fix + visible test `pragmas_test.go`), one worktree per arm (`bench/r2/<arm>`). Prompt identical for all arms. Agent `general-purpose`, model set per call. Grader: 8 hidden pragma tests + backup-contains-data test + the 6 M-1 FK tests, full suite, `TestOrder_T3_ConcurrentOpenSingleWinner` x30, no-skip / no-removed-test-line checks. On the base, T3 failed 1 of 30 runs and 5 of the 8 pragma tests were red.

| Arm | Hidden + visible (17) | T3 x30 fails | Diff | Tokens | Tools | Duration |
|---|---|---|---|---|---|---|
| opus-1 | 17/17 | 0 | +4 −1 | 56,649 | 11 | 102 s |
| opus-2 | 17/17 | 0 | +3 −1 | 55,573 | 11 | 94 s |
| sonnet-1 | 17/17 | 0 | +2 −1 | 52,555 | 6 | 63 s |
| sonnet-2 | 17/17 | 0 | +1 −1 | 51,337 | 5 | 63 s |
| haiku-1 | 17/17 | 0 | +4 −1 | 65,718 | 18 | 140 s |
| haiku-2 | 17/17 | 0 | +3 −1 | 63,140 | 13 | 104 s |
| haiku-3 | 17/17 | 0 | +3 −1 | 61,833 | 13 | 105 s |
| haiku-4 | 17/17 | 0 | +3 −1 | 63,878 | 13 | 105 s |
| haiku-5 | 17/17 | 0 | +3 −1 | 62,103 | 13 | 109 s |

**Result.** All 9 arms changed only `internal/database/database.go`, to the same DSN (`_pragma=busy_timeout(5000)`, `journal_mode(WAL)`, `foreign_keys(1)`, `_time_format=sqlite`; haiku-2 orders journal_mode first, harmless because the driver sorts busy_timeout first, `sqlite.go:141-150`). The only failing test in the suite is the known `TestDashboardToday_T20_Golden`. The task again did not discriminate between models. Each model read the driver source (`applyQueryParams`, `sqlite.go:136`) instead of guessing. Haiku used 18-25% more tokens and 2-3x more tool calls than Sonnet.

**Cost bounds** (combined tokens only; list prices): Opus $0.22-$1.13, Sonnet $0.10-$0.53, Haiku $0.01-$0.03 per arm.

**Gemini (Antigravity `agy`, plan mode, read-only; allowed reads: repo, scratchpad and, added for M-2, `go/pkg/mod/modernc.org`).** Four tests on the same task:
| Test | Gemini | Verified against |
|---|---|---|
| Code: rewrite `New` | Correct, identical to the Claude arms; stated "Confidence: 100"; output ended with garbled trailing text | grader (same DSN as opus-1) |
| Research: DSN keys of modernc v1.46.1 | Essentials correct (`_pragma` only, mattn keys ignored, per-connection scope). Did not read the source although allowed; marked the function name UNSURE (real: `applyQueryParams`, `sqlite.go:136`); doubted that v1.46.1 exists (it is in `go.mod:9`); omitted that busy_timeout runs first | driver source |
| Review of a diff with a seeded bug (pragmas set by one `db.Exec`, so only one pooled connection gets `busy_timeout`/`foreign_keys`; hidden tests `BusyTimeoutOnEveryConnection` and `ConcurrentWritersNoBusy` catch it) | **CLEAN: missed the bug.** Its reasoning assumed the ignored `_journal_mode`/`_busy_timeout` keys work, contradicting its own research answer | hidden tests |
| Review of the correct diff (control) | CLEAN (correct) | grader |

Same seeded-bug prompt to Claude, no tools: Haiku found the bug (flagged `_busy_timeout`/`_journal_mode` as unverified); Sonnet found it, named the root cause (mattn keys ignored) and the exact fix, and added the `db.Close()` leak on error. n=1 per reviewer.

**Side findings (agent reports; the reset-password one verified by `grep`).** `New` does not close `db` when `Ping` or `migrate` fails (`database.go`). `cmd/reset-password/main.go:24` opens SQLite with no pragmas (reported by haiku-1; verified). `New(":memory:")` gives each pooled connection its own database (pre-existing). No test covers `backup.go`; the grader's backup test passes on all arms.

**Caveats.** n=2 for Opus and Sonnet, n=5 for Haiku; `effort` not equalized; aliases assumed to be the 5.5 family; Gemini's code test had no write or exec rights by design (patch applied by the grader); the agent changes are uncommitted in the worktrees.

## Experiment M-3 — model comparison on customer logical delete (design task)

Not a `/us` run, and not numbered US-079: that number is already taken by "Fix 42 Lint Errors (R-35)" (`SPRINT-18-Estabilizacion-Produccion.md:28`, `:168`), which an agent noticed. Base `bench/r3/base` (34ea6f6 = M-2 base + sonnet-2's WAL/FK DSN fix + visible test `customer_delete_test.go`, red with `FOREIGN KEY constraint failed`), one worktree per arm (`bench/r3/<arm>`), tag `bench-r3-base`. Prompt identical for all arms and deliberately silent on several points (double delete, customer without history, selling to an inactive customer, spec updates), which each agent had to decide and report. Grader: 9 hidden handler tests (6 red on the base), visible test, full suite, test-line diff, spec diff.

| Arm | Hidden (9) | Visible | Suite besides T20 | Diff | Existing-test edit | Spec | Tokens | Tools | Duration |
|---|---|---|---|---|---|---|---|---|---|
| opus-1 | 9/9 | pass | green | +24 −5 | `== 18` → `== 19` | fiado.md | 78,579 | 21 | 170 s |
| opus-2 | 9/9 | pass | green | +118 −9 | `MAX == 18` → "row for version 18 exists once" | fiado.md | 86,476 | 25 | 253 s |
| sonnet-1 | 9/9 | pass | green | +14 −5 | `== 19` | fiado.md | 64,591 | 10 | 69 s |
| sonnet-2 | 9/9 | pass | green | +13 −5 | `== 19` | none | 62,676 | 8 | 65 s |
| haiku-1 | 9/9 | pass | green | +32 −13 | `== 19` | fiado.md | 88,205 | 22 | 152 s |
| haiku-2 | 9/9 | pass | green | +18 −6 | `== 19` | fiado.md | 97,127 | 25 | 182 s |
| haiku-3 | 9/9 | pass | **`TestMigration18_T1` fails** | +21 −2 | none (left failing, asked for permission) | fiado.md | 88,120 | 16 | 145 s |
| haiku-4 | 9/9 | pass | green | +24 −7 | `== 19` | fiado.md | 84,855 | 21 | 191 s |
| haiku-5 | 9/9 | pass | green | +24 −6 | `!= 18` → `< 18` (weakened: accepts anything ≥ 18) | fiado.md | 83,558 | 25 | 155 s |

**Result.** All 9 arms chose the same core design: migration 19 `ALTER TABLE customers ADD COLUMN active INTEGER NOT NULL DEFAULT 1`, `Delete` = `UPDATE customers SET active = 0 WHERE id = ? AND active = 1`, `List` and its search filtered by `c.active = 1`, `GetByID`/history/reports unfiltered, second delete = 404. The hidden tests did not separate them. The differences were in judgement:
- **Test edit.** The existing `TestMigration18_T1` pins the latest migration to 18 and breaks with any new migration. 6 arms moved it to 19; opus-2 re-expressed it (still checks that migration 18 ran, no longer pins the head); haiku-5 weakened it to `< 18` (it would no longer detect a missing new migration); haiku-3 left the suite red and asked for permission instead of using the exception the prompt gave.
- **Scope.** opus-2 also changed `handler/customer.go` and `service/customer.go` (typed `ErrCustomerHasBalance`; any other error now returns a generic 500 instead of leaking raw SQL text as 409) and added a handler test: +118 lines, the only arm that fixed the raw-error leak. haiku-1 changed `dashboard.go` so the dashboard portfolio sums `Debtors` instead of `List`, removing a divergence the Sonnets flagged (verified: base `dashboard.go:172` sums `customers.List`).
- **Spec (Rule 2).** 8 arms added a deletion rule to `fiado.md`; sonnet-2 added nothing, saying the spec did not cover it.
- **Verification.** opus-1, opus-2 and haiku-2 wrote throwaway HTTP tests (409 with balance, 404, totals stable); opus-1 got the search param wrong (`search` vs `q`), haiku-2 noticed the same symptom.

**Gap nobody closed (verified).** `internal/service/sale.go:131-144` only requires a customer id for credit; an inactive customer can still receive credit sales or payments, and the customer-list-based dashboard figure then diverges from `total_portfolio`. All arms flagged it; none changed it (product decision).

**Stale spec (verified).** `spec/domain/restaurante.md:85` says "The declared foreign keys are not enforced at runtime (roadmap R-56)"; with the DSN fix they are.

**Caveats.** `effort` not equalized; one design task; the hidden tests encode my own reading of the open points (double delete accepts 200 or 404); agent changes are uncommitted in the worktrees.

**Gemini in M-3 (Antigravity `agy`, plan mode, read-only, everything inline).**
| Use | Result |
|---|---|
| Implementer (`gemini-1`: files inline, asked for a unified diff, applied with `git apply`, graded like the other arms) | Applied cleanly. Hidden 9/9, visible pass, suite green, +9 −5, same design as the 9 Claude arms (migration 19, `UPDATE … AND active = 1`, filtered `List`, test moved to `== 19`). Smallest diff of all; no spec update (like sonnet-2). Decisions on open points reported correctly. |
| Reviewer of the haiku-5 diff (test weakened to `v < 18`) | **Found it**, quoting the violated requirement ("never weaken any other test"). Only real defect in that diff. |
| Reviewer of the haiku-1 diff | No result: the call asked to read a file and headless mode auto-denied it (reproduced on retry). |
| Reviewer of the opus-2 diff | No result: no output within 280 s (also empty in the first run). |

Together with M-2 (code correct; research mostly correct but did not read the source; seeded bug missed, control diff CLEAN), Gemini is useful as an implementer for a well-specified change and caught a weakened test here, but it has produced one clear review miss, two unusable runs, and wrong reasoning in M-2. n is small.
