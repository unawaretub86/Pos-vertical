# Agentic pipeline benchmark

> **Type:** current · **Rules over:** how the `/us` pipeline performs run by run · **Updated:** 2026-11-11 (run 31) · 2026-11-10 (run 30)

Each `/us` run adds a row to the runs table and a detail section. The orchestrator does it in step 6 of `.claude/commands/us.md`, in the same merge commit. It exists to decide with data, not impressions, whether the pipeline is worth what it costs and what to adjust. The design is in [tosdd-plan.md](tosdd-plan.md).

> **Language rule (`../CLAUDE.md` → Rule 0):** everything in this file is English. Runs 1–9 were originally written in Spanish and translated on 2026-10-01; number formats were converted (`1.547` → `1,547`, `3,9 min` → `3.9 min`). Quoted Spanish strings are UI text or user phrases.

## Current state — where to resume

> Updated at the end of each work session with the pipeline.

**Last run:** 30 (US-093, reset-data tool, level L). Before: 29 (US-098), 28 (US-097), 27 (US-094), 26 (US-095). No `us/*` worktree is open. `dev` is at the US-095 merge (verify with `git log`).

**SPRINT-18 (Production Stabilization):** US-075..US-080 are done in code and tests and merged to `dev` (US-080 still needs real-printer checks P-1..P-4 in `gaps.md`); US-081 (LAN concurrency) is done in code and tests (real-device checks M-1..M-3 in `gaps.md`). SPRINT-18 is complete in code. US-078 was merged outside `/us` (experiment follow-up, `bde4d9f`).

**SPRINT-19 (Fiado collection):** complete in code and tests; all 8 US (US-082, US-083, US-086, US-087, US-043, US-084, US-088, US-085) merged to `dev` (manual checks US-082 M-1..M-3, US-083 M-1, US-086 M-1, US-087 M-1, US-043 M-1, US-084 M-1 and US-085 M-1 in `gaps.md`).

**SPRINT-20 (Go-live for client #1):** US-090, US-091, US-096, US-092 and US-095 merged to `dev` (manual checks US-090 M-1, US-091 M-1, US-092 M-1 and US-095 M-1 in `gaps.md`). US-094 approved by the PO on 2026-11-08.

**Next US:** US-089 (the last open one in SPRINT-20; US-094 and US-093 are done). Roadmap items R-102..R-109, R-112..R-129, R-131..R-149 await PO prioritization. Run `/us <ID> [M|L]`. Step 0 now writes `changes/<ID>/owner` (one orchestrator per US, run 26); at GATE 2 grep `dev` for new R-numbers before merging (run 23).

**Pending from the PO (manual, parked 2026-10-20):** the list is in [gaps.md](gaps.md) → "Pending PO validations" (US-077, US-076, US-078, US-062, US-054, US-074, US-034).

**Pipeline changes since run 9 (applied, with the PO's approval):**
- After run 25 (PO yes, 2026-11-09): every subagent prompt forbids stdin-reading commands and asks for bounded long commands (`.claude/commands/us.md` conventions); QA smoke builds the binary with the real frontend export when the US touches `frontend/` (`.claude/agents/qa.md` §3).
- Hooks in `.claude/settings.json` run `.claude/hooks/pos-guard.js`: pnpm only, no commit on `master`/`main`, applied migrations, `t.Skip`, CGO (blocks); `BETWEEN`, SQL concatenation, hard-coded colors, Spanish docs, missing spec (warn/ask). Documented in `CLAUDE.md`.
- `scripts/gemini-review.sh` is called by `/us`: `contract` mode before GATE 1 and `diff` mode in every QA round, in parallel with `qa`. Advisory: the orchestrator verifies every finding. Timeout 420 s with one retry; the pipeline waits at most about 10 minutes.
- After run 13 (PO: "use gemini as u want"): the `diff` prompt states that the spec delta is written at close and must not be reported, and asks for weakened/skipped tests as defect lines instead of prose; the script keeps only the finding lines (or `CLEAN`) and falls back to the raw answer if none match.
- Experiment M-1..M-3 (model comparison) did not change any agent's model: the recommendations (cheaper implementer when a red test defines done, QA on the strong model, Gemini never the only reviewer) wait for more data. See `experiments/model-bench/README.md` on branch `experiment/model-bench`.

**Open, no adjustment applied:** (1) the pipeline cannot verify UI in a browser (no browser tool in `qa`; run 7 used headless Edge by hand). (2) The auto-mode classifier can stop Bash and PowerShell calls (seen when force-deleting branches and in run 7). (3) Finished agents vanish from `ListAgents`, so resume-instead-of-relaunch rarely applies (every agent in run 10 was cold).

**How to resume:** in a new session, run `/us <ID>` for the US the PO prioritizes and read the PO list in `gaps.md` first.

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
| 13 | 2026-11-05 | US-082 | L | ~929k (A 409k=110k+127k+172k · T 243k=119k+124k · I 205k=95k+110k · Q 72k) | 232 | ~39.3 | 1 (analyst lost when the earlier session ended; resumed from transcript) | 1 | 6 contract discrepancies (D-1..D-6) + 5 Gemini-verified contract fixes + 1 red-test gap + 1 AC-7 miss + 6 QA MINOR | 7 new (R-83..R-89) | 0 | +1,146 / −93, 29 files (excluding this benchmark entry) |
| 14 | 2026-11-06 | US-083 | L | ~372k (A 220k=56k+60k+105k · T 28k · I 40k · Q 85k) | 131 | ~21.8 | 0 | 1 | 4 contract discrepancies (D-1..D-4) + 1 Gemini-verified contract fix + 3 QA MINOR (pre-existing) | 5 new (R-90..R-94) | 0 | +160 / −35, 9 files (excluding this benchmark entry) |
| 15 | 2026-11-06 | US-086 | L | ~336k (A 158k, last of 110k→112k→117k→158k · T 32k · I 46k · Q 100k) | 144 | ~28.6 | 1 (AC-7 premise false, PO rewording after QA) | 1 | 4 contract discrepancies (D-1..D-4) + 3 Gemini-verified contract additions + 5 QA MINOR | 4 new (R-95..R-98) + 1 closed (R-92) | 0 | +256 / −27, 13 files (excluding this benchmark entry) |
| 16 | 2026-11-06/07 | US-087 | L | ~397k (A 179k, last of 121k→128k→130k→179k · T 46k · I 42k · Q 130k) | 215 | ~35.8 | 1 (vacuous T-4 CustomersPage case, PO dropped it before RED) | 1 | 4 contract discrepancies (D-1..D-4, incl. a third broken page `/barra`) + 4 Gemini-verified contract fixes + 1 vacuous test + 0 QA MINOR (3 INFO) | 3 new (R-99..R-101) + 1 closed (R-93) | 0 | +247 / −22, 11 files (excluding this benchmark entry) |
| 17 | 2026-11-07 | US-043 | L | ≥238k recorded (A 111k contract-resume + 111k close · T not recovered · I 54k resume, first run cut off · Q 73k) | ≥99 | ≥12.9 (session 2 only) | 1 (API rate limit killed the implementer; next session restarted cold) | 1 | 8 contract discrepancies (D-1..D-8) + 4 QA MINOR | 4 new (R-102..R-105) | 0 | +465 / −44, 12 files excluding tests (+985 / −44, 16 files total, excluding this benchmark entry) |
| 18 | 2026-11-07 | US-084 | L | ~416k (A 193k, last of 132k→144k→193k · T 42k · I 75k · Q 107k) | 144 | ~26.1 | 0 | 1 | 8 contract discrepancies (D-1..D-8) + 2 Gemini-verified contract additions (Q-7, Q-8) + 3 QA MINOR + 1 pre-existing money bug (R-110) | 5 new (R-106..R-110) | 0 | +626 / −335, 13 files (excluding this benchmark entry) |
| 19 | 2026-11-07 | US-088 (R-110) | L | ~348k (A 166k, last of 134k→141k→166k · T 76k · I 26k · Q 80k) | 161 | ~24.8 | 0 | 1 | 5 contract discrepancies (D-1..D-5) + 1 Gemini-verified contract addition (AC-7) + 2 QA MINOR | 5 new (R-111..R-115) + 1 closed (R-110) | 0 | +512 / −22, 8 files (excluding this benchmark entry) |
| 20 | 2026-11-07 | US-085 | L | ~295k (A 144k, last of 72k→91k→144k · T 33k · I 47k · Q 70k) | 99 | ~17.9 | 0 | 1 | 6 contract discrepancies (D-1..D-6) + 2 Gemini-verified contract fixes (G-1, G-2) + 2 QA MINOR | 4 new (R-116..R-119) | 0 | +312 / −22, 8 files (excluding this benchmark entry) |
| 21 | 2026-11-07 | US-081 | L | ~416k (A 173k, last of 99k→124k→173k · T 101k · I 64k · Q 78k) | 180 | ~31.4 | 1 (PASSES⚠️ review, pipeline-mandated) | 1 | 2 contract discrepancies (D-1, D-2) + 1 root cause found by probe (deferred tx upgrade) + 3 Gemini-verified contract fixes (G-1..G-3) + 3 QA MINOR | 4 new (R-120..R-123) | 0 | +996 / −54, 17 files (excluding this benchmark entry) |
| 22 | 2026-11-07 | US-090 | M | ~261k (A 124k, last value after contract 91k · I 72k · Q 65k) | ~119 | ~22 | 0 | 1 | 3 contract discrepancies fixed (D-1..D-3) + 4 sent to roadmap (D-4..D-7) + 1 Gemini-verified contract fix (AC-10 wording) + 4 QA MINOR | 6 new (R-124..R-129) | 0 | +402 / −20, 10 files (excluding this benchmark entry) |
| 23 | 2026-11-08 | US-091 | L | ~399k (A 193k, last of 145k→170k→193k · T 88k · I 33k · Q 85k) | 149 | ~31.6 | 1 (PASSES⚠️ review, pipeline-mandated) | 1 | 6 contract discrepancies (D-1..D-6) + 1 Gemini-verified contract fix (AC-6) + 3 QA MINOR | 3 new (R-134..R-136) + 1 moved to a US (R-111 → US-095) | 0 | +982 / −24, 12 files (excluding this benchmark entry) |
| 24 | 2026-11-08 | US-096 | L | ~535k (A 192k · T 202k · I 56k · Q 85k) | ~170 | ~45.0 | 1 (QA r1 AC-8 30s budget exceeded; PO raised to <= 40s) | 2 | 10 contract findings (D-1..D-10) + 3 Gemini-verified (G-8, G-11, G-4) + 4 QA MINOR | 4 new (R-137..R-140) + 1 closed (R-130) | 0 | +1,827 / −11, 14 files (excluding this benchmark entry) |
| 25 | 2026-11-09 | US-092 | L | ~603k (A 201k, last of 136k→141k→201k; pre-rate-limit part not counted · T 140k · I 154k · Q 108k) | ~208 | ~63 (incl. a 10 min implementer stall) | 3 (rate limit on the analyst; implementer stalled 600 s; QA MINOR 4 needed a PO decision) | 1 | 4 contract discrepancies (D-1..D-4) + 5 QA MINOR | 4 new (R-141..R-144) | 0 | +2,086 / −51, 26 files (excluding this benchmark entry) |
| 26 | 2026-11-09 | US-095 | L | ~446k incl. waste (A 133k contract · A' 41k GATE 1 adjust · T 68k · I n/a · Q 51k · duplicate A 153k) | ~110+ | ~15+ (implementer and close not reported) | 4 (implementer stall; rate limit mid-implementation; two sessions orchestrating the same US; takeover at GATE 2) | 1 | 5 contract discrepancies (D-1..D-5) + 3 Gemini-verified contract fixes + 1 test-author finding (R-148) + 3 QA MINOR | 5 new (R-145..R-149) + R-111 closed | 0 | +878 / −27, 11 files (excluding this benchmark entry) |
| 27 | 2026-11-09 | US-094 | L | ~669k (A 170k, last of 140k→170k · T 97k · I 159k · Q 102k) | 162 | ~38.3 | 1 (API rate limit 429 interrupted close analyst; completed in Antigravity) | 1 | 5 contract discrepancies (D-1..D-5) + 3 Gemini-verified contract fixes (AC-10, AC-24, AC-22) + 5 QA MINOR | 8 new (R-150..R-157) | 0 | +3,225 / −13, 18 files (excluding this benchmark entry) |
| 28 | 2026-11-09 | US-097 (R-157) | L (M requested; analyst recommended L at GATE 1) | ~175k (A 74.6k, last of 47.9k→49.7k→74.6k · T 34.7k · I 26.6k · Q 39.2k) | 71 | ~14.3 | 0 | 1 | 1 contract discrepancy (D-1: 6 frontend strings + 1 backend message, not the 4 in the roadmap) + 2 QA MINOR + 1 pre-existing real bug | 1 new (R-158); R-157 closed | 0 | +134 / −13, 9 files (excluding this benchmark entry) |
| 29 | 2026-11-10 | US-098 (R-158, closes R-05) | L | ~448k (A 171.7k · T 167.0k · I 43.2k · Q 66.5k) | 135 | ~24.5 | 1 | 1 | 6 contract discrepancies (D-1..D-6) + 1 contract error caught by the test-author + 3 QA MINOR | 6 new (R-159..R-164); R-158 and R-05 closed | 0 | +929 / −55, 27 files (excluding this benchmark entry) |
| 30 | 2026-11-10 | US-093 (reset-data tool) | L | ~490k (A 219.9k · T 97.2k · I 98.1k · Q 75.2k) | 186 (A 110 · T 21 · I 30 · Q 25) | ~39 excluding the analyst (T ~5.1 · I ~27 · Q ~7); analyst n/a (cumulative agent wall time 3.2 h includes waiting) | 1 | 1 | 6 contract discrepancies (D-1..D-6) + 3 Gemini-verified gaps (Q-13) + 3 test-side errors found by the implementer + 5 QA MINOR | 3 new (R-165..R-167); R-153 cross-referenced | 0 | +2,149 / −0, 14 files, 0 docs (excluding the docs and this benchmark entry) |
| 31 | 2026-11-11 | US-089 (autostart at Windows logon, daily backups to a configurable folder, 30 copies) | L | ~495k (A 146.0k · T 136.9k · I 139.4k · Q 73.0k) | 283 (A 147 · T 49 · I 63 · Q 24) | ~29 excluding the analyst (T ~13.5 · I ~10 · Q ~5.5); analyst n/a (contract 408 s, sync 2,185 s wall, mostly waiting) | 0 | 1 | 8 contract discrepancies (D-1..D-8) + 5 Gemini-verified contract gaps + 6 test-side errors (invented API names, double-appended tests, 3 fixture collisions, 3 pinned schema versions) + 5 QA MINOR | 5 new (R-168..R-172); R-167 closed | 0 | +2,359 / −53, 22 files (excluding docs, `changes/` and `INSTALL.md`) |
| 32 | 2026-11-12 | US-099 (R-168: pre-migration copy gets its own file name) | M | ~176k (A 81.0k, last of 65.7k→81.0k · I 60.2k · Q 34.6k) | 73 (A 33 · I 31 · Q 9) | ~5.2 (I ~3.2 · Q ~1.9); analyst contract 139 s, close 48 s | 0 | 1 | 0 contract discrepancies (1 addition: `.gitignore` did not cover `.pre-reset-*`) + 3 QA MINOR (2 were the close step, 1 weak red on T-5) | 0 new; R-168 and K2 closed | 0 | +319 / −12, 9 files (incl. docs) |
| 33 | 2026-11-12 | US-100 (R-162: time-zone startup warning) | M | ~103k (A 51.4k, last of 40.8k→51.4k · I 26.5k · Q 24.9k) | 35 (A 19 · I 10 · Q 6) | ~4.2 (A ~2.0 · I ~1.6 · Q ~0.6) | 0 | 1 | 0 contract discrepancies + 1 QA MINOR (spec pending until close) + 1 Gemini false positive | 1 new (R-176); R-162 and K3 closed | 0 | +90 / −1, 2 code files (excluding docs) |

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

## Run 13 — US-082 (payment method on *abonos*; cash *abonos* in the register close, level L)

Started in a session that the PO confirmed as owner after `ListAgents` showed two idle peer sessions and a fresh US-082 commit (`1cbf2ea`) from one of them. The first analyst call was still running when that session ended; the next session found `changes/US-082/` empty and resumed the same agent with `SendMessage` (its transcript survived), so no cold relaunch was needed.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 recorded (contract, resumed after the session loss; contract update after GATE 1; close) + 1 lost (first contract attempt, no telemetry) | 409k (110.3k + 126.8k + 172.1k) | 67 | ~10.2 min |
| test-author | 2 (red tests; fix of a missing mock method after RED) | 243k (119.0k + 123.9k) | 64 | ~10.6 min |
| implementer | 2 (round 1; AC-7 follow-up) | 205k (95.1k + 110.0k) | 69 | ~12.8 min |
| qa | 1 round | 72.4k | 32 | ~5.7 min |

**What it found.** Contract: D-1..D-6 (stale sprint/spec line refs; unknown customer gets 400 not 404; duplicate close email; close modal omits `cash_in`; `:memory:` + held `Conn()` opens a second empty DB — a test trap the analyst hit while probing). The analyst ran a probe test that confirmed the bug ($130,000 expected instead of $150,000) and the 403 for cashiers. Red tests: `sale_test.go`'s `mockRegisterRepo` lacked the new interface method, so `internal/service` did not compile with the implementation — found by the implementer, fixed by the test-author in a second test commit (`a6b8763`, new RED). Implementation round 1 mapped a failed open-register lookup to 400 instead of AC-7's 500; sent back before QA. QA round 1: OK, 0 BLOCKER, 6 MINOR (500 on internal failures accepted into the spec; ST1005 capital letter required by AC-6; open-register cards lack the line → R-87; gofmt churn in `model.go`; D-2 → R-83; `Close` ignores other sums' errors → R-88). QA's binary smoke on port 18082 confirmed expected 155,000 with transfer/card *abonos* excluded.

**Unplanned stops:** 0 to the PO outside the gates (the ownership question at step 0 was the run-12 check, asked by design). Internal re-dispatches: 2 (red-test gap, AC-7 miss). 1 agent lost to the session ending. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | not recorded | 13 (2 RULE_CONFLICT, 6 MISSING_CASE, 4 AMBIGUOUS, 1 UNTESTABLE) | 5 (zero-*abono* SUM test; `Summary` error path; `/detail` fields untested; $0 line shown or not; notice placement) — all accepted by the PO at GATE 1 | 8 (cut-off spec delta, undefined Q-A/Q-B: false; "any role"/other roles: only admin and cashier exist; inactive customer: existing, desirable; race: already out of scope; 320 px: manual by design) | n/a (G-2 led to the `getRegister` 404/500 fix) |
| diff, round 1 | OK | not recorded (finished after QA, ~6 min) | 0 defect lines; 2 prose statements ("no spec update"; 8 existing tests edited) | 0 defects (tests were edited only for the new signature) | 1 ("no spec update", structural, same as run 12) | no |

The diff answer was mostly the model's own reasoning text (~150 lines) instead of the requested ≤15 finding lines; the prompt's "State explicitly whether…" sentence invited prose.

**Operational friction.**
- The orchestrating session ended while the analyst ran; the background agent stopped. Resume via `SendMessage` worked from the saved transcript.
- Contract §5/§6 told the implementer to write the spec, while `/us` reserves the spec for the analyst at close; the orchestrator told the implementer not to touch it. The pos-guard Rule 2 hook did not block the implementation commit.
- A test-side mock outside the files the contract named needed the new interface method; the test-author did not compile-check `internal/service` against an interface stub.

### Adjustments applied (PO: "use gemini as u want")
1. `gemini-review.sh diff`: the prompt says the spec delta is written at close by the analyst and must not be reported; edited/weakened tests are reported as defect lines.
2. The script keeps only lines that match the requested format (or `CLEAN`), and falls back to the raw answer when none match.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12 proposal 1 still pending: step 0 checks `ListAgents` for peer sessions (done by hand this run).
2. Contract template: the analyst should not assign the spec delta to the implementer (§5/§6), since `/us` writes it at close.
3. Test-author: after adding a method to an interface, grep every test stub of that interface (`grep -rn "func (m \*mock<X>)"`).

### What to measure in run 14
- Whether the filtered Gemini diff output stays within format and drops the "no spec update" false positive.

## Run 14 — US-083 (business name in the collection WhatsApp message, R-61, level L)

Step 0 checked `ListAgents` by hand (one idle peer session, no `us/US-083` branch), so no ownership question was needed. The step-1 analyst was resumed with `SendMessage` twice (contract update, close) even though finished subagents did not appear in `ListAgents`; the resume by agent id worked both times.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; contract update after GATE 1; close) | 220k (55.6k + 59.5k + 105.0k) | 60 | ~7.4 min |
| test-author | 1 | 27.5k | 5 | ~0.8 min |
| implementer | 1 | 39.9k | 11 | ~1.9 min |
| qa | 1 round | 84.7k | 55 | ~11.7 min |

**What it found.** Contract: D-1 (stale line refs in the sprint and R-61), D-2 (two different current texts, neither with the business name, so "keep the current text" was undefined), D-3 (the analyst reproduced the defect with a copy of the component tree under `react-dom/server` instead of only reading), D-4 (empty business name is reachable: fresh DB, `/setup` loading or failed, whitespace-only name accepted by `PUT /setup`). The PO accepted Q-1..Q-4 as recommended plus one Gemini-verified addition (AC-8 browser step). Red tests and implementation needed no re-dispatch. QA round 1: OK, 0 BLOCKER, 3 MINOR, all pre-existing: the customer detail page in the static export reads `params.id = '_'` (`web/static.go:48-49` serves the placeholder) and requests `/api/customers/NaN/detail` → R-92; the same provider-outside-AppLayout defect in `/pos` and table checkout, so US-080 direct printing never fires → R-93; 320 px clipping → R-94. QA ran T-M1 in headless Edge with playwright-core against a binary on port 18083, including `GET /api/setup` aborted.

**Unplanned stops:** 0. Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | not recorded | 5 (1 RULE_CONFLICT, 3 MISSING_CASE, 1 UNTESTABLE) | 1 (T-M1 had no step for AC-8) | 4 (spec at close: pipeline design; zero balance: callers guard at `CustomerDetailClient.tsx:364` / `page.tsx:606`; empty customer name: rejected at `internal/service/customer.go:56`; no-digit phone: pre-existing, unchanged) | n/a |
| diff, round 1 | OK | not recorded (finished before QA) | 0 (`CLEAN`) | 0 | 0 | no (QA's 3 MINOR are pre-existing code outside the diff) |

The filtered diff output stayed in format and did not repeat the "no spec update" false positive (run-13 measure: yes).

**Operational friction.**
- QA's most valuable finding (R-92) came from running the real binary, not from the diff: the defect is outside the changed lines and invisible to unit tests and to a diff-only reviewer.
- QA took ~12 min, most of it building a scratch binary and driving the browser.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12/13 proposals 1–3 still pending (peer-session check in step 0, contract template not assigning the spec, test-stub grep).
2. When a US touches a page, QA's smoke should open that page in the binary by direct URL (not only through in-app navigation), since dynamic routes behave differently in the static export.

### What to measure in run 15
- Whether a binary smoke by direct URL finds more pre-existing defects per run, and its time cost.

## Run 15 — US-086 (dynamic pages read their id from the URL path in the static export, R-92, level L)

PO prioritized R-92 ahead of US-043 (2026-11-06). The analyst registered US-086 in SPRINT-19 from the roadmap (`08f98fa`). It was resumed with `SendMessage` three times (GATE 1 adjustment, contract update after RED, close). The tokens column records the **last value** (158.0k), per "How it is measured"; run 14 summed the analyst's invocations instead, so the two rows are not directly comparable (run 14 last-value: 105k).

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; GATE 1 adjustment; contract update after RED; close) | 158.0k (110.0k → 112.0k → 116.8k → 158.0k) | 69 | ~12.1 min |
| test-author | 2 (red; T-8 after contract update) | 31.7k (28.2k → 31.7k) | 13 | ~1.0 min |
| implementer | 1 | 46.1k | 10 | ~2.5 min |
| qa | 1 round | 100.0k | 52 | ~13.0 min |

**What it found.** The contract analyst built the binary from `dev` and confirmed R-92 by running it (first confirmation on `dev`), and found the same defect on `/mesas/<orderId>` (actions went to `/api/orders/_/...`) and `/catalogo/<token>` (`/api/public/catalog/undefined`). GATE 1 (Q-1..Q-4, accepted as recommended): both extra pages in scope, client-side helper with URLs unchanged, full-reload navigation left out (D-4 → R-98), verification by QA T-M1 plus a PO manual check. After RED, the PO approved three Gemini-derived additions (null cases for `/mesas` and `/catalogo`, malformed percent-encoding → T-8), committed as `548ece8` (`test: contract update`, step 2b). Implementation needed no re-dispatch. QA round 1: OK, 0 BLOCKER, 5 MINOR. M-1: AC-7 ("SHALL keep working under `pnpm dev`") rested on a false premise, because the three pages already answer 500 under `next dev` with `output: "export"`; the PO reworded it to "unchanged" → R-95. M-2/M-3 duplicated `popstate`-only path hook → R-96. M-4 order id not validated → R-97. M-5 self-contradicting R-92 row, fixed at close. QA drove the compiled binary in headless Edge over CDP with direct-URL loads and in-app clicks (run-14 proposal 2, applied by hand), and captured exceptions and console errors.

**Unplanned stops:** 1 (AC-7 rewording after QA). The post-RED contract update is counted as a late GATE 1 addendum, not a stop: GATE 1 was presented before Gemini finished, as the command allows. Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | not recorded (finished after GATE 1) | 5 (1 RULE_CONFLICT, 3 MISSING_CASE, 1 UNTESTABLE) | 3 (null `orderId`, null catalog token, malformed percent-encoding) | 1 (spec at close: pipeline design) + 1 discarded at the time but later borne out: UNTESTABLE on AC-7 (`pnpm dev`), which QA then showed could not be met | n/a |
| diff, round 1 | OK | not recorded (finished before QA) | 0 (`CLEAN`) | 0 | 0 | no (QA's MINORs are in unchanged code or need a running dev server) |

Contract-mode precision this run: 3–4 of 5. It is the best Gemini contract result so far; the discarded AC-7 finding should have been taken as a hint to verify the premise.

**Operational friction.**
- The orchestrating session ended at GATE 2; the merge and this entry were done from a new session, with metrics read back from the earlier transcript.
- QA's `next dev` background command first failed (exit 127, `pnpm` not on the Git Bash path in background) before it ran.
- A stray `python -` process from QA could not be stopped: `taskkill` was denied by the permission classifier.
- QA took ~13 min, again mostly binary build plus browser driving.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12/13/14 proposals still pending (peer-session check in step 0, contract template not assigning the spec, test-stub grep, direct-URL binary smoke in QA).
2. The analyst verifies every "SHALL keep working under X" criterion by running X during the contract, not only by reading, since AC-7 was the only defect that reached QA.

### What to measure in run 16
- Whether a dev-server or binary premise check in the contract removes post-QA PO stops.

## Run 16 — US-087 (pages read the business config outside its provider, R-93, level L)

Started right after the run-15 merge, in the same session. The analyst registered US-087 from the roadmap (`eb94149`) and was resumed with `SendMessage` three times (GATE 1 fixes, CustomersPage drop, close); the test-author once. Tokens are last values.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; GATE 1 fixes; CustomersPage drop; close) | 178.9k (120.6k → 127.8k → 129.6k → 178.9k) | 109 | ~18.4 min |
| test-author | 2 (red; drop vacuous case) | 45.7k (43.0k → 45.7k) | 18 | ~2.6 min |
| implementer | 1 | 42.4k | 21 | ~2.3 min |
| qa | 1 round | 130.2k | 67 | ~12.5 min |

**What it found.** The analyst confirmed the defect by execution (binary in headless Edge plus a `react-dom/server` probe that records whether each `useBusinessConfig()` call sits inside the provider) and found it wider than R-93: D-1 `/barra` mounts no provider at all, so `loading` stays `true` and the bar screen has been blank in every mode since US-058; D-2 the table-checkout ticket view renders outside `AppLayout`, so its `Imprimir` also ignored ESC/POS; D-3/D-4 two specs described behaviour that did not happen, and `/barra` had no spec rule. GATE 1: Q-1..Q-4 accepted as recommended (`/barra` in scope with the provider only; checkout ticket inside the layout; new `frontend/vitest.config.mts` with only the `@` alias so tests can import pages; R-80 out). The test-author confirmed the right red reason with a throwaway alias config outside the repo, and flagged the T-4 CustomersPage case as vacuous (the hook only runs in `CustomerRow`, which does not render under SSR, so the log is empty and `every(Boolean)` always passes); the PO dropped it before RED. Implementation needed no re-dispatch. QA round 1: OK, 0 BLOCKER, 0 MINOR, 3 INFO; T-M1 (a)–(f) plus (e2) passed in the compiled binary, and QA added a fake TCP printer to see the success toast. QA emptied `PATH` for the binary so its browser auto-open could not reach Andrey's desktop.

**Unplanned stops:** 1 (vacuous test, `PASSES⚠️` shown to the PO by rule; it led to a contract change before RED). Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | not recorded (finished before GATE 1) | 8 (1 RULE_CONFLICT, 2 AMBIGUOUS, 4 MISSING_CASE, 1 UNTESTABLE) | 4 as wording fixes, two of them merged into one (AC-8 coupling of print method and restaurant mode; AC-5 toasts; T-2/T-3 coverage overstated; T-4 cannot see the Sidebar) | 3 (spec at close: pipeline design; two "pending Q-n" ambiguities resolved by the gate) | n/a |
| diff, round 1 | OK | not recorded (finished before QA) | 0 (`CLEAN`) | 0 | 0 | no (QA found no MINOR either) |

Gemini's T-4 finding (Sidebar invisible to the test) sat next to the real gap the test-author then found (CustomersPage invisible too): the coverage claim of T-4 was the weak point, and neither the analyst nor the orchestrator caught the second half before red.

**Operational friction.**
- First run where the contract-stage execution (run-15 proposal 2, applied by hand) widened the scope before any code: `/barra` would otherwise have stayed blank.
- The test-author needed a config file it is not allowed to create; a scratchpad config passed with `--config` solved it without breaking the file-ownership rule.
- The contract and test edits after the PASSES⚠️ ran in parallel (different files), which kept the stop short.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12..15 proposals still pending.
2. test-author: every "every call is X" style assertion must also assert the log is non-empty, and a case that cannot record anything is reported as PASSES⚠️ (this run's test-author did it unprompted).

### What to measure in run 17
- Whether the new `vitest.config.mts` lets later US cover page-level behaviour in unit tests and shortens QA's browser time.

## Run 17 — US-043 (customer debt and history in the POS at checkout, with credit-limit confirmation, level L)

Split across two sessions. Session 1 (2026-11-07, early morning) ran the contract, GATE 1, Gemini contract review and red tests (`26d1dd7`), then launched the implementer, which was killed mid-work by the API 5-hour rate limit (HTTP 429). Session 2 found the worktree with the implementer's uncommitted work, could not resume the agent (`SendMessage`: "No transcript found" — agent transcripts do not cross sessions), and launched cold implementer, QA and close-analyst agents. Session-1 metrics were recovered from the session `.jsonl`; the test-author notification and the first implementer's usage were not found there.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | contract (session 1, last notification is a 2-tool resume) + close (session 2, cold) | 111.1k (contract, last) · 111.4k (close) | ≥54 (2 recorded for contract + 52 close) | ≥5.3 min (0.3 + 4.9) |
| test-author | 1 (session 1) | not recovered | not recovered | not recovered |
| implementer | 2 (session 1 cut off by rate limit; session 2 cold, reviewed and verified) | 53.8k (session 2) | 11 (session 2) | ~2.3 min (session 2) |
| qa | 1 round | 72.7k | 34 | ~5.6 min |

**What it found.** The contract listed 8 discrepancies (D-1..D-8) between spec, sprint and code; the D-7 line-number correction turned out unnecessary at close (`internal/service/customer.go:54,68` already correct, verified by the orchestrator). The session-1 implementer had finished the code before the rate limit hit; the session-2 implementer found every AC (AC-1..AC-22) already met and wrote no code. QA round 1: OK, 0 BLOCKER, 4 MINOR — stale summary on re-selecting the same customer (R-104), `addUsualProducts` writes a plain cart value from `cartRef` (R-105), the one-line `OrderClient.tsx` prop removal outside the contract file list (forced by AC-9), and four UI strings the contract did not name (documented in the spec at close). The smoke test checked `pos-summary` in the binary (200 / empty 200 / 404 / 400 / cashier 200, inactive product excluded). T-M1 (browser) was not run: parked as US-043 M-1 in `gaps.md`. The close also sent R-102 (debt age from oldest unpaid credit) and R-103 (hard-coded aging colours) to the roadmap, as the contract assigned.

**Unplanned stops:** 1 (API rate limit during implementation). Internal re-dispatches: 0. QA rounds: 1. Violations: 0 (no test file changed after RED; QA left no tracked changes).

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | not recorded (session 1) | 11 (2 RULE_CONFLICT, 3 AMBIGUOUS, 1 UNTESTABLE, 5 MISSING_CASE) | not recovered (verified at GATE 1 in session 1; the final contract has AC-21 inactive products excluded, an AC-18 double-press guard and AC-22 error wrapping, matching three of them) | not recovered (the spec-at-close RULE_CONFLICT is pipeline design) | n/a |
| diff, round 1 | OK | ~3 min | 0 (`CLEAN`) | 0 | 0 | no; QA found 4 MINOR that Gemini did not raise |

**Operational friction.**
- A rate-limited subagent leaves a half-done worktree and a dead agent id. The orchestrator recovered state from `git status`, the branch log and the previous session's `.jsonl` (task-notification summaries), which worked but took manual digging.
- Gate decisions and per-finding Gemini verification from session 1 live only in that session's transcript; the run entry cannot fully reconstruct them.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12..16 proposals still pending.
2. **Applied 2026-11-07 (PO yes)** in `.claude/commands/us.md` (conventions + step 0.3): at each gate and after each agent notification, the orchestrator appends a one-line progress note (agent, tokens, tools, duration, gate decision) to `changes/<ID>/progress.md`, so a run interrupted by a rate limit or a session end can be resumed and benchmarked without mining transcripts.

### What to measure in run 18
- Whether QA can run any of T-M1 itself (headless Edge, as in run 7 and run 16) instead of parking it for the PO.

## Run 18 — US-084 (quick *abono* modal from the Customers list, level L)

First run with the progress log (`changes/<ID>/progress.md`, applied after run 17); every metric below comes from it. Same session as run 17. The analyst was resumed with `SendMessage` twice (GATE 1 fixes, close), so resume-instead-of-relaunch worked for the first time since run 16 even though `ListAgents` did not list the finished agent.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; GATE 1 fixes; close) | 192.7k (132.2k → 143.7k → 192.7k) | 60 | ~11.6 min |
| test-author | 1 | 41.6k | 8 | ~1.4 min |
| implementer | 1 | 74.5k | 24 | ~3.9 min |
| qa | 1 round | 107.2k | 52 | ~9.2 min |

**What it found.** The analyst ran a throw-away Go probe through `h.Routes()` and found 8 discrepancies, three of them by running: D-4 a negative balance is reachable (credit, full *abono*, void), D-5 credit-sale balances carry IVA cents that the whole-peso field cannot settle, D-8 `GET /customers` answers a cashier with `total_portfolio`. D-6 (read, then confirmed by run) was a money hole left by US-082: the dashboard `Registrar pago` sent no method, so transfers counted as cash. GATE 1: Q-1..Q-8 accepted as recommended (shared `AbonoModal`, dashboard included). The implementer needed no re-dispatch. QA round 1: OK, 0 BLOCKER, 3 MINOR, and it ran all of T-M1 in headless Edge over the binary (first run where the manual browser plan was executed by QA end to end; only real-device input and the light-theme overlay look were left to the PO). QA also found a pre-existing money bug outside the story; the close analyst ran it and showed it is wider (cash voids too): R-110, every voided sale is subtracted twice from the expected register cash.

**Unplanned stops:** 0. Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~3 min | 11 (2 RULE_CONFLICT, 2 AMBIGUOUS, 2 UNTESTABLE, 5 MISSING_CASE) | 2 (sub-$1 balance unpayable → Q-7; `Total` with cents vs digits-only field → Q-8) | 9 (spec at close: design; 4 pending-question items resolved by the gate; 2 from a truncated input — Q-6 and §6 looked missing; concurrency already out of scope; detail button already hidden at balance ≤ 0, `CustomerDetailClient.tsx:370`, `:545`) | n/a |
| diff, round 1 | OK | ~6 min | 0 (`CLEAN`) | 0 | 0 | no; QA found 3 MINOR and R-110 |

Two of the nine contract false positives came from Gemini reading a truncated contract: the input builder cuts long contracts. Worth checking the size limit in `scripts/gemini-review.sh`.

**Operational friction.**
- None blocking. The progress log made metrics and the gate record trivial to assemble.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12..16 proposals still pending.
2. `scripts/gemini-review.sh contract`: check the truncation limit; a contract of this size (8 sections, 20 ACs) lost its tail and produced 2 false positives.

### What to measure in run 19
- Whether QA's headless-Edge T-M1 holds up for a backend-only fix (R-110), or the plan shrinks to Go tests plus a register-close smoke run.

## Run 19 — US-088 (R-110: a void no longer subtracts the sale twice from the register, level L)

Registered from the roadmap (`122918d`, committed after GATE 1 confirmed the sprint placement, before RED). Same session as runs 17-18; metrics from `changes/US-088/progress.md`. The analyst was resumed with `SendMessage` twice (GATE 1 fix, close).

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract + registration; GATE 1 fix; close) | 166.3k (134.1k → 140.7k → 166.3k) | 84 | ~11.2 min |
| test-author | 1 | 75.8k | 20 | ~3.2 min |
| implementer | 1 | 25.6k | 9 | ~2.9 min |
| qa | 1 round | 80.0k | 48 | ~7.5 min |

**What it found.** The analyst reproduced R-110 by running through `h.Routes()` and widened it: transfer and mixed voids also subtract, voids after the close write into the closed register, the dashboard and assistant cash estimate go negative (D-2), and there is no void action in the UI at all (D-1, so shop data is almost certainly clean and no migration was needed). It also found a test hazard: closing a register over HTTP in a `:memory:` test starts the close-email goroutine, and the next query can land on a fresh empty database (8/10 probe runs failed); the contract steered the test-author around it (now R-115). The fix was a 14-line deletion. GATE 1: Q-1..Q-4 as recommended plus one Gemini-found assertion (AC-7, recomputed `expected` of a closed register after a void), which the analyst valued by running before RED. QA round 1: OK, 0 BLOCKER, 2 MINOR; QA re-ran the new tests on a scratch copy of RED to confirm they fail there, and smoke-tested the whole scenario in the binary.

**Unplanned stops:** 0. Internal re-dispatches: 0. QA rounds: 1. Violations: 0. One self-inflicted slip: the implementer ran `git checkout <RED> -- internal/repository/sale.go` at the end of a command, undoing its own edit, then re-applied it; the orchestrator re-checked the diff and re-ran the repository and handler tests before the checkpoint.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~4 min | 10 (2 RULE_CONFLICT, 6 MISSING_CASE, 2 AMBIGUOUS) | 1 (AC-7 recomputed `expected` of the closed register) | 9 (spec at close: design ×2; 5 about void behaviour the fix does not touch; half-open boundary unrelated; Q-3 "truncated" — the input was truncated again) | n/a |
| diff, round 1 | OK | ~2 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 2 MINOR were a stale roadmap ref and the accepted Q-1 trade-off |

Second run in a row where a contract-mode false positive came from Gemini reading a truncated contract (run 18 proposal 2).

**Operational friction.**
- None blocking. Registering the US from the roadmap and committing it after GATE 1 (rather than right after the contract) let the PO confirm sprint placement first.

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12..16 proposals and run-18 proposal 2 (Gemini contract truncation) still pending.
2. implementer: never run `git checkout`/`git restore` on non-test files; to compare against RED use `git stash` or a scratch worktree. This run's revert was caught only because the orchestrator re-ran the tests.

### What to measure in run 20
- Whether US-085 (aging chips, frontend) keeps QA's headless-Edge T-M1 complete, as in runs 18-19.

## Run 20 — US-085 (aging filter chips with amounts on the Customers tab, level L)

Already registered in SPRINT-19. Same session from contract to merge; metrics from `changes/US-085/progress.md`. The analyst was resumed with `SendMessage` twice (GATE 1 decisions, close) although `ListAgents` no longer listed it.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; GATE 1 decisions; close) | 144.4k (71.8k → 91.1k → 144.4k) | 49 | ~9.1 min |
| test-author | 1 | 33.3k | 5 | ~0.9 min |
| implementer | 1 | 47.4k | 10 | ~2.1 min |
| qa | 1 round | 69.6k | 35 | ~5.7 min |

**What it found.** The analyst checked by running that `GET /customers` has no pagination (chip totals over the loaded list are complete) and that `total_portfolio` ignores the search, and timed `daysOverdue` at the boundaries. Main discrepancy: a same-day *fiado* is 0 days old and would fall in no `1–14 días` chip (D-1); the PO renamed the first chip `Hasta 14 días` (0–14). Others: stale `file:line` refs in the sprint and the spec, a wrong R-05 premise (elapsed 24-hour periods, no time-zone shift, D-3), row color bands (7/14) that do not match the chip buckets (D-4), and a hard-coded amber the spec described as `var(--warning)` (D-5). GATE 1: Q-1..Q-6 as recommended plus G-1..G-3. The implementer shared one `now` per fetch between chips and row badge so they never disagree; QA accepted it as a MINOR (R-118). QA round 1: OK, 0 BLOCKER, 2 MINOR; T-M1 complete in headless Edge over CDP, including a real *abono* through the modal and 320 px emulation.

**Unplanned stops:** 0. Internal re-dispatches: 0. QA rounds: 1. Violations: 0. The test-author fixed six helper-API details the contract left open (return shape of `bucketTotals`, bucket keys, generic input type, label arguments); the orchestrator passed them to the implementer as binding instead of stopping, since they do not change behavior.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~3 min | 13 (3 RULE_CONFLICT, 6 MISSING_CASE, 3 AMBIGUOUS, 1 UNTESTABLE) | 3 (G-1 an active chip emptied by an *abono* could not be deselected; G-2 empty-range text with search + chip; G-3 labels left on open questions) | 10 (half-open rule applied to day buckets; spec at close ×2; negative age and `null` date already covered; roles — `GET /customers` is open to all, `handler.go:279`; search/refetch race pre-existing; 3 duplicates of G-3) | n/a |
| diff, round 1 | OK | ~3 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 2 MINOR were the per-fetch `now` trade-off and a missing group label |

G-1 was a real AC conflict (AC-7 vs AC-8) that neither the analyst nor the orchestrator caught; the best contract-mode yield so far for a UI US.

**Operational friction.**
- New worktree needed `pnpm install` before the test-author could run vitest (known).
- The QA smoke opened the binary's default-browser tab on the PO's desktop (`localhost:8091`).

### Proposed adjustments (not applied; need the PO's yes)
1. Run-12..16, run-18 proposal 2 and run-19 proposal 2 still pending.
2. contract template: §5 should state the helper signatures and return shapes when the tests target new pure helpers, so the test-author does not have to choose them (six choices this run).

### What to measure in run 21
- US-081 (LAN multi-register concurrency) is backend and concurrency-heavy: whether the test-author's tests are deterministic (no flaky goroutine timing) and whether QA can reproduce the race in the binary.

## Run 21 — US-081 (multi-register LAN concurrency and stability, level L)

Already registered in SPRINT-18. Same session from contract to merge; metrics from `changes/US-081/progress.md`. The analyst was resumed with `SendMessage` twice (GATE 1 decisions, close) although `ListAgents` did not list it as live.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; GATE 1 decisions; close) | 173.4k (98.7k → 124.3k → 173.4k) | 112 | ~12.6 min |
| test-author | 1 | 100.6k | 26 | ~6.9 min |
| implementer | 1 | 64.0k | 15 | ~4.6 min |
| qa | 1 round | 78.4k | 27 | ~7.3 min |

**What it found.** The analyst ran a throwaway probe (file-backed WAL DB through `h.Routes()`) instead of reasoning from the sprint text: concurrent credit (fiado) sales lost 50–75 % to `SQLITE_BUSY` because every `BeginTx` was DEFERRED and the credit path reads before it writes; cash sales were unaffected. `_txlock=immediate` fixed it in the probe (50/50), and the fix was one DSN parameter. D-1: the sprint asked the client to retry on 503, but the backend never returned 503 (busy was 409 on orders/tables and a raw 400 on sales). D-2: "multi-register" is really many devices on one open register (R-18 is real multi-register). The contract designed the retry around idempotency: 503 only on rolled-back writes, no retry on network errors. GATE 1: Q-1..Q-5 as recommended plus G-1..G-3. The test-author changed one existing test the contract did not list (T22 409 → 503), which the PO accepted at the PASSES⚠️ stop. QA round 1: OK, 0 BLOCKER, 3 MINOR; smoke in the binary 50/50 concurrent credit sales with exact register totals; concurrency tests repeated 3× with no flakiness. QA MINOR-1 caught a wrong rationale in the contract (deferred `tx.Rollback()` does not undo a failed COMMIT; the driver does, `modernc.org/sqlite tx.go:41-50`), which kept it out of the spec.

**Unplanned stops:** 1 (PASSES⚠️ review of T-2, T-5, T-8, F-4: all expected regression guards, plus the T22 edit). Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~4 min | 13 | 3 (G-1 sale-specific network text on a shared client; G-2 `tables.go` 503 untested; G-3 "503 means nothing committed" not stated for orders/tables) | 10 (open PO questions ×3, nonexistent `ventas.md`, path prefix, `qty = 0` already rejected at `service/sale.go:154`, inactive entities / register close / void permissions out of scope, `EmitFromSale` recover already in §7) | n/a |
| diff, round 1 | OK | ~4 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 3 MINOR were a contract rationale, an untested commit-time BUSY and abort handling |

**Operational friction.**
- New worktree needed `pnpm install` before the test-author (known).
- `internal/handler` tests now take ~43 s because lock tests wait out the 5 s `busy_timeout`.
- Token cost was the highest of the L runs since run 1, driven by the analyst's probe and the test-author's 572-line concurrency file.

### Proposed adjustments (not applied; need the PO's yes)
1. Earlier pending proposals unchanged (runs 12..16, 18-2, 19-2, 20-2).
2. contract template §5: list existing tests whose assertions the AC contradicts (here T22), so the test-author does not edit an unlisted test and stop the pipeline.

### What to measure in run 22
- Whether the CI wall time grows past what is acceptable with the 5 s lock tests (handler package ~43 s).

## Run 33 — US-100 (R-162: startup warning when the PC time zone is not UTC-5, level M)

Created from dev `4d3b22f`. A roadmap item (R-162, go-live K3), registered by the analyst as US-100 in SPRINT-20. Commits: register `1b5eca9`, implementation `b23c538`, docs close after it. Second level M run in a row: the implementer wrote tests T-1..T-5 itself (orchestrator told it so in the prompt, see proposal in run 32).

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 2 (contract 40.8k/10 tools/80 s; close, resumed with SendMessage 51.4k/9/40 s) | 51.4k | 19 | ~2.0 min (120 s) |
| implementer | 1 round | 26.5k | 10 | ~1.6 min (95 s) |
| qa | 1 round | 24.9k | 6 | ~0.6 min (37 s) |

**What it found.**
- The analyst found that Go on Windows reports the zone as `-05` (not `COT`) and ignores `TZ`, so the contract compares the offset in seconds and the helper takes an injected `time.Time`. No discrepancy against R-162 or `tiempo.md`.
- QA round 1: OK, 0 BLOCKER, 1 MINOR (spec line pending at QA time, applied at close). Real-binary smoke logged `time zone: UTC-05:00` with no warning; the warning path is covered by unit tests only.
- GATE 1 decisions: Q-1 log only (R-176 opened for a UI banner), Q-2 fixed -05:00 offset.

**Unplanned stops:** 0 outside GATE 1 and GATE 2. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|
| contract | OK | 1 | 0 | 1 (spec assigned to the analyst at close is the pipeline design, a known pending proposal) | n/a |
| diff, round 1 | OK, `CLEAN` | 0 | 0 | 0 | no |

**Operational friction.**
- The stale `../Pos-vertical-wt/` dirs and the `us/R-170` and `us/US-099` branches are still present.
- IDE diagnostics on the worktree (`undefined: runAutostartFlag`, module not in workspace) are gopls noise; `go build` passes.

### What to measure in run 34
- Whether the level M tests-by-implementer proposal gets applied, and the `us.md` wording at M.

## Run 32 — US-099 (R-168: the pre-migration copy gets its own file name, level M)

Created from dev `306f98b`. A roadmap item (R-168, go-live K2), registered by the analyst as US-099 in SPRINT-20. Commits: register `3f32b14`, implementation `e38c898`, docs close `25aeaa8`. First run at level M since run 22: no test-author, so the implementer wrote tests T-1..T-8 itself.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 2 (contract 65.7k/21 tools/139 s; close, resumed with SendMessage 81.0k/12/48 s) | 81.0k | 33 | ~3.1 min (187 s) |
| implementer | 1 round | 60.2k | 31 | ~3.2 min (190 s) |
| qa | 1 round | 34.6k | 9 | ~1.9 min (111 s) |

**What it found.**
- The analyst reproduced the bug by running it (v22 DB through `New`, then `RunBackup`: the same-day file moved from v22 to v23) and found no discrepancy against the roadmap text. One addition: `.gitignore:11` covered only `data.db.backup-*`; Q-3 added the two missing patterns.
- QA round 1: OK, 0 BLOCKER, 3 MINOR (spec and roadmap pending at QA time, both done at close; T-5 fails against the old code only by compile error because the replace-on-retry behavior is unchanged). Real-binary smoke: `data.db.pre-migration-22-20261112` kept v22 after the same-day startup backup.
- The implementer's red state was a compile error (`backupBeforeMigration` signature), not an assertion failure; QA confirmed by reading that T-1 and T-2 would fail on the old behavior.

**Unplanned stops:** 0 outside GATE 1 and GATE 2. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|
| contract | OK, `CLEAN`, finished after GATE 1 | 0 | 0 | 0 | n/a |
| diff, round 1 | OK, `CLEAN`, ~4 min | 0 | 0 | 0 | no |

**Operational friction.**
- At level M the pipeline text assigns the tests to the test-author, so the orchestrator had to tell the implementer to write them; `.claude/commands/us.md` does not say so (proposal below).
- `ListAgents` showed no agents after they finished; `SendMessage` to the analyst id resumed it for the close.
- A stale `../Pos-vertical-wt/R-170` dir and others from earlier runs are still on disk (Windows leftovers).

**Proposal (not applied, needs PO yes):** in `.claude/commands/us.md`, state that at level M the implementer writes the tests, and that the "tests untouched" check then compares only pre-existing assertions (no removed lines).

### What to measure in run 33
- Whether a level M run with implementer-written tests keeps the same QA findings as level L (here 0 BLOCKER).

## Run 31 — US-089 (autostart at Windows logon, daily backups to a configurable folder with 30 copies, level L)

Created from dev `076607e`. Registered in SPRINT-20 before the run; no roadmap item opened it, and it closed R-167. Commits: RED `8da705f`, contract update `e04c610`, dedup `d811cb9`, fixtures `c6a109f`, implementation `50b9f47`, then the docs close.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract 132.9k/52 tools/408 s; GATE 1 decisions 19.2k/3/9 s; contract sync after RED 59.5k/23/2,185 s; close 146.0k/69/205 s) | 146.0k | 147 | n/a (the 2,185 s sync is mostly waiting) |
| test-author | 3 (RED 108.4k/25/342 s; round 2, resumed for the contract sync 128.3k/18/313 s; round 3, resumed for the fixtures 136.9k/6/157 s; tokens are the last value, tools and seconds per invocation) | 136.9k | 49 | ~13.5 min (812 s, summed) |
| implementer | 1 round | 139.4k | 63 | ~10 min (602 s) |
| qa | 1 round | 73.0k | 24 | ~5.5 min (332 s) |

**What it found.**
- Contract D-1..D-8: `data.db` is a relative path, so a scheduled task without a "Start in" folder would create an empty database in `C:\Windows\System32` (D-1, the blocker for the install step); `INSTALL.md` said no window opens but the exe is a console app (D-2); the glob prune breaks on `[` in the path and could delete manual `data.db.backup-*` copies (D-3); a failed same-day re-run destroyed the good copy and the pre-migration copy shares the daily name (D-4); `PUT /setup` zeroes omitted fields and `GET /setup` is cashier-readable, so the backup settings needed their own endpoints (D-5); `gaps.md` said `pos.db` (D-6); two version pins (D-7); a stale `setupTestDB` line reference (D-8). Backup under concurrent writes was verified by running before the contract was written.
- Gemini contract review raised 11: 5 verified minor gaps (the AC-10 row-count rule against AC-17 under concurrent writes; AC-14 initial values; AC-14 `last_success_at` and `last_file` preserved after a failed attempt; AC-13 `dir` omitted or `null`; AC-6 file-name time zone) and 6 false positives. It finished after GATE 1, so the gaps were closed in a contract sync round (step 2b) instead of before the PO decided.
- Test-side errors: the test-author invented API names that the contract did not fix (recorded in contract §5 at the sync); it appended T-6b and T-9b twice (compile error), fixed after the checkpoint with a dedup commit; the implementer then found three fixtures colliding with the seeded admin user (`internal/database/migrate.go:273-275`) and three tests pinning schema 22 where contract D-7 listed two; a resumed test-author fixed them.
- QA round 1: OK, 0 BLOCKER, 5 MINOR (spec delta pending at QA time, `INSTALL.md` "Ajustes" against the sidebar's "Configuración", both autostart flags together, the pre-migration copy skipped on a brand-new database, the ninth column and two new components). Three became roadmap rows (R-170, R-171, plus R-168 from the contract); the deviations are in the sprint notes.

**Unplanned stops:** 0 outside GATE 1 and GATE 2 (one ownership and US-ID question at the start). QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|
| contract | OK, ~14 min | 11 | 5 (minor) | 6 | n/a |
| diff, round 1 | OK, `CLEAN`, ~12 min | 0 | 0 | 0 | no |

QA found nothing Gemini caught that QA missed; Gemini's diff review found nothing.

**Operational friction.**
- The test-author ran `python -` without a heredoc and left a stray `python.exe` that the orchestrator killed.
- The first analyst contract took 408 s but the sync took 2,185 s of wall time, mostly waiting.
- Writing tests first meant the API names (`BackupSource`, `RunBackup`, `canWrite`, `buildTaskXML`, `installAutostart`) had to be read out of the RED commit into contract §5 after the fact.

**Diff.** `git diff dev...HEAD --shortstat` excluding `project-docs/`, `changes/` and `INSTALL.md`: 22 files, +2,359 / −53 (production code: `cmd/pos/autostart.go` 206, `internal/database/backup.go` 286 / −25, `internal/handler/backup.go` 89, `internal/repository/setup.go` 103, migration 23 and `main.go` wiring; frontend `BackupSection` 115, `BackupBanner` 45, `backupSettings.ts` 69; the rest are tests).

### What to measure in run 32
- Whether the contract should require a full grep for schema-version pins and seeded rows (the admin user) before RED, so fixtures and pins are right in the first test-author round.
- Whether the Gemini contract review should start before GATE 1 is presented, to avoid a post-gate sync round.
- The PO manual checks US-089 M-1..M-3 in `gaps.md`: the real Task Scheduler install and reboot have not been run.

## Run 30 — US-093 (reset-data tool, level L)

Created from dev `8e133b5`. One orchestrator session (`pos-vertical-83`); no other session touched the worktree. Commits: first RED `ac1cbc5`, RED update `8875006`, implementation `171f836`. Registered in SPRINT-20 before the run; no roadmap item.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 5 (contract 124.4k/48 tools/528 s; GATE 1 adjust for Q-13 151.1k/25/97 s; sync with the test-author 156.9k/3/20 s; table-count fix 158.2k/3/12 s; close 219.9k/31 tools) | 219.9k | 110 | n/a (cumulative agent wall time 3.2 h includes waiting for the PO and other agents) |
| test-author | 2 (RED 92.96k/16/267 s; fixes after the implementer's report 97.2k/5/36 s) | 97.2k | 21 | ~5.1 min |
| implementer | 1 round (including a 3-test failure report) | 98.1k | 30 | ~27 min (1,640 s) |
| qa | 1 round | 75.2k | 25 | ~7 min (418 s) |

**What it found.**
- Contract D-1..D-6: stock cannot be rebuilt purely from `stock_entries` (the manual product form writes stock with no entry, R-153); the ticket number is the sale id, so the `sqlite_sequence` counters must restart; the SQLite exclusive lock is not kept after COMMIT (verified by running two processes); the daily backup prunes `<db>.backup-*` so the reset backup needs another name; the `electronic_documents` statuses (`draft`, `pending`, `accepted`, `rejected`, `cancelled`); an active admin is needed for the audit row.
- Gemini contract review raised 14: 3 verified (no active admin; fiscal guard keyed on the current `ambiente` while `electronic_documents` has no such column; inactive products) and became Q-13; 11 false positives, 6 of them caused by input truncation.
- Implementer's first run reported 3 test-side errors: the contract said 24 user tables (it is 25), a `dump` helper with a spacing mistake, and the daily-backup glob. The tests had been written without any production code to compile against. The test-author fixed them as a second RED (`8875006`) and contract V-4 was corrected.
- QA round 1: OK, 0 BLOCKER, 5 MINOR (an orphan backup after the three late refusals; a pointless `//nolint:errcheck`; a swallowed shop-name scan error; the undocumented 1 s advance of the backup name, now in `reset.md`; `TableCounts` concatenates names from `sqlite_master`). Three became roadmap rows (R-165 audit label, R-166 hygiene, R-167 backup and US-089).

**Unplanned stops:** 1. The implementer reported 3 test-side errors (above) and the pipeline paused for a second RED. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|
| contract (two concurrent runs) | OK twice | 14 | 3 | 11 (6 caused by input truncation) | n/a |
| diff, round 1 | OK, `CLEAN` | 0 on a ~90 KB diff (input may be truncated) | 0 | 0 | no |

**Operational friction.**
- Gemini was started twice by accident (a nested background `&`); the second run overwrote the first one's output file.
- Writing tests first for a tool whose package does not exist at all means the tests do not compile; the exported names had to be fixed in the contract before the test-author could work (§5 "Exported API" was added during the run).

**Diff.** `git diff dev...HEAD --stat` excluding docs: 14 files, +2,149 / −0 (production code in `internal/resetdata` 582 lines, `cmd/reset-data` 186, `internal/database/database.go` 6; the rest are tests).

### What to measure in run 31
- Whether fixing the exported API names in the contract up front removes the "tests do not compile" first-run failure for a new package.
- The PO manual check US-093 M-1 in `gaps.md`: the destructive path has not been run on a real database.

## Run 29 — US-098 (R-158, UTC-text date bounds so evening sales count in "today", level L)

Created from dev `6f1b1ee`. One orchestrator session (`pos-vertical-83`); no other session touched the worktree. Roadmap item R-158 (which also closes root cause R-05) registered as US-098 in SPRINT-20 during contract mode; the `changes/` folder kept the ID R-158.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 5 (contract 126.6k/35 tools/394 s; GATE 1 adjust 131.2k/3/15 s; contract sync with the test-author findings 140.2k/5/37 s; close 171.7k/15/125 s; plus one short GATE 1 record step counted inside the adjust) | 171.7k | 58 | ~9.5 min |
| test-author | 1 | 167.0k | 56 | ~10.7 min |
| implementer | 1 round | 43.2k | 9 | ~2.9 min |
| qa | 1 round (+1 interrupted attempt) | 66.5k (tokens of the interrupted attempt unknown) | 12 | ~1.4 min |

**What it found.**
- Contract D-1..D-6: R-05's aging claim was wrong (aging was never broken); `LowStock` is a second instance of the same bug; the `.UTC()` boundary-second bug (a `+00:00` suffix mis-buckets a row at the exact boundary); four inclusive `<=` list filters; stale roadmap line references; no spec file defined the zone rule.
- A contract error caught by the test-author: the AC-6 / T-7 "este mes" expectation was unreachable (72.000 / 4, not >= 87.000); fixed before RED.
- The test-author flagged that T-8 (day grouping) depends on the OS zone.
- QA round 1: 3 MINOR (T-8 under `TZ=UTC` on Windows is a test artifact; `dbTimePtr` has no nil check; comment alignment) and 1 NOTE (spec delta pending, as designed) became 1 roadmap item.
- Roadmap: R-158 and R-05 closed; R-159..R-164 opened (6).

**Unplanned stops:** 1. QA's first attempt died with API 403 `oauth_org_not_allowed` (subscription/billing) before writing `qa.md`; the same agent was resumed and the second attempt completed. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~9 min | 11 | 0 | 11 (4 caused by input truncation; the rest pre-existing or out of scope) | n/a |
| diff, round 1 | OK | ~5 min | 0 (`CLEAN`) | 0 | 0 | no |

**Operational friction.**
- The orchestrator's `git commit -am` skipped the new untracked spec file `tiempo.md` in the close commit; caught immediately and committed separately. Lesson: stage untracked files before the close checkpoint.
- The test-author fixed 74 fixture statements through central exec helpers (`metricsExec`, `execSeed`, `assistantTestEnv.exec`) instead of one by one.
- The analyst's close list cited one test (`TestMetricsRepo_T30_AgingExcludesNullDaysOverdue`) it had not run; the orchestrator verified that every cited test name exists.

### What to measure in run 30
- Whether the evening-window fix holds on the first real evening after 19:00 (PO manual check US-098 M-1 in `gaps.md`).
- CI behavior of T-8 on Linux (roadmap R-159).

## Run 28 — US-097 (R-157, singular wording in the catalog import, level L)

Created from dev `90501e3`. Same orchestrator session (`pos-vertical-83`); no other session touched the worktree. Level M was requested; the analyst recommended L at GATE 1 because the implementer cannot write the count-1 tests, and the PO approved it. Roadmap item registered as US-097 in SPRINT-20 during contract mode.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract 47.9k/16 tools/106 s; GATE 1 adjust 49.7k/2/14 s; close 74.6k/13/111 s) | 74.6k | 31 | ~3.9 min |
| test-author | 1 | 34.7k | 9 | ~0.8 min |
| implementer | 1 round | 26.6k | 16 | ~4.4 min |
| qa | 1 round | 39.2k | 15 | ~5.2 min |

**What it found.**
- Contract D-1: the roadmap listed 4 plural strings; the real set was 6 frontend strings plus 1 backend message (`el archivo tiene 1 errores`), all fixed.
- The test-author wrote 1 intentional PASSES regression guard (plural unchanged, AC-7).
- QA round 1: no code issue; 2 MINOR (sprint ACs and a stale roadmap row) closed at close.
- QA flagged the spec delta as BLOCKER, but by pipeline design the analyst applies it at close; the orchestrator treated it as pending-close.
- Pre-existing real product bug found by QA running `go test` after 19:00 America/Bogota: roadmap R-158 (the dashboard/report "today" window uses local midnight while sales `created_at` is UTC text, so evening sales are omitted; same defect as R-05).

**Unplanned stops:** 0. QA rounds: 1. Violations: 0. Roadmap: 1 new item (R-158), R-157 closed.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~4 min (arrived after GATE 1; started after the analyst finished) | 7 | 0 | 7 (all moot or already covered by AC-3 / §6) | n/a |
| diff, round 1 | OK | ~5 min | 0 (`CLEAN`) | 0 | 0 | no |

**Operational friction.**
- The day-end rate limit did not recur.
- QA's BLOCKER on the spec delta not yet applied. Proposal (not applied; needs the PO's yes): `qa.md` treats "spec delta not yet applied" as a NOTE, as it did in run 27.
- The implementer reported the pre-existing handler failures honestly; the orchestrator reproduced them on clean dev before accepting.
- The roadmap's R-157 text said 4 strings; the real set was 7.

### What to measure in run 29
- Whether QA classifies "spec delta pending" as NOTE.
- Whether the pre-existing time-dependent test failure (R-158) bites CI after 19:00 Bogota.

## Run 27 — US-094 (product catalog import from CSV, level L)

Created from dev `53aaf35`. Owner lock respected (`pos-vertical-83`). Staged through the standard 4-agent pipeline; the close analyst hit the Claude 5-hour rate limit while editing sprint documents and was completed in Antigravity.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; GATE 1 adjust; close; close correction after a review found wrong test names in the sprint list) | 170.3k (140.4k → 170.3k → close; later invocations not reported) | 74+ | ~14.6+ min |
| test-author | 1 | 97.4k | 20 | ~6.4 min |
| implementer | 1 round | 159.2k | 44 | ~11.1 min |
| qa | 1 round | 102.4k | 24 | ~5.8 min |

**What it found.**
- CSV import with automatic synonym recognition and manual column mapping (RFC 4180, `;`/`,`/tab, UTF-8/Windows-1252 auto-detect, UTF-16 and Excel `.xlsx` refusal with help messages).
- 2 MB and 5,000-row bounds. Dry run with preview of up to 5 valid rows and 200 formatted errors.
- Atomic execution in one transaction: categories created on the fly (case/accent-insensitive match), products inserted in file order, `stock_entries` row `Importación de catálogo` created for each item with initial stock > 0, and `audit_log` row recorded.
- Front-end modal in `/inventory` gated on `admin` role (`canImportProducts`); downloadable Colombian-format CSV template.
- QA round 1: OK, 0 BLOCKER, 5 MINOR (readHeaderLine untested, missing columns mapping reset, 1 productos plural, error swallowing, duplicate precedence), 1 NOTE (spec delta at close). Smoke test passed on real export.
- The close done by another tool (Antigravity) after the rate-limit stop cited nonexistent test names in the sprint "Verified criteria" list and roadmap numbers (R-155..R-157) that did not exist; the orchestrator review caught both and a fourth analyst invocation fixed them.

**Unplanned stops:** 1 (close analyst hit 5-hour rate limit; the close finished in Antigravity, then needed a correction invocation). QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~5 min | 10 | 3 (AC-10 negative message unreachable, AC-24 button visibility for cashier, AC-22 category UNIQUE race reported as barcode) | 7 (truncation x2, spec delta exists, template in AC-28, categories have no active flag, Q-3 pending gate, AC-7 two distinct cases) | n/a |
| diff, round 1 | OK | ~10 min | 0 (`CLEAN`) | 0 | 0 | no |

**Operational friction.**
- The contract input truncation again generated false positives on a long contract (44 KB).
- Close analyst interrupted mid-edit by API limit; Antigravity recovered the worktree and completed documentation.

### What to measure in run 28
- Verification of US-093 reset tool with master data preserved (including stock from catalog import).

## Run 26 — US-095 (admin void button for a sale in the UI, level L)

Already registered in SPRINT-20. **Two sessions orchestrated this US in the same worktree.**
- **Session A** created the worktree (`dc4a569`) and the contract, presented GATE 1, and then waited for the PO.
- **Session B** found the worktree, took the GATE 1 "yes" in its own conversation at 11:52, and ran test-author, implementer, QA and close. It hit an implementer stall and the 5-hour rate limit (it switched the implementer to Sonnet 5.5 after the limit), and stopped at GATE 2.
- **At 15:30 session A received the PO's GATE 1 "yes"** and resumed its analyst, which re-applied the same decisions to `contract.md`, an ignored file. It changed no code.
- **Session A then noticed the foreign lines in `progress.md`.** The PO chose to have A take over GATE 2 and merge B's work.

Metrics are partial, because B reported no tokens for the implementer or the close.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst (A) | 2 (contract; duplicate GATE 1 adjust) | 132.8k → 153.2k | 57 | ~8.9 min |
| analyst (B) | 2 (GATE 1 adjust; close) | 40.7k (close n/a) | 19+ | ~1.2+ min |
| test-author (B) | 1 | 67.9k | 14 | ~2.3 min |
| implementer (B) | 2+ (stalled; resumed; rate limit; resumed on Sonnet 5.5) | n/a | n/a | >10 min plus waiting |
| qa (B) | 1 round | 51.0k | 20 | ~2.9 min |

**What it found.** No production Go change was needed: the void endpoint already worked, and US-091 had certified its money effects. Things the analyst found by running the backend:
- A void on an already-closed register leaves the stored close unchanged. Re-reading that register shows the recomputed `expected` drifting by the voided cash, and the cash handed back is recorded nowhere (R-112). The PO chose to allow the void and show a warning that tells the admin to record a `Salida`.
- The 409/403 bodies carry codes (`sale_has_accepted_invoice`, `forbidden`) in `error`, so the frontend maps them to Spanish.

The test-author found that `recent_sales` has no tie-break for same-second sales (R-148). QA round 1: OK, 0 BLOCKER, 3 MINOR (R-149). Gemini diff: CLEAN.

**Unplanned stops:** 4 (implementer stall, rate limit, the duplicate orchestration, and the takeover at GATE 2). QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~5 min | 10 | 3 (AC-2 loading state, AC-7 error text source, AC-3 unknown-method label) | 7 (2 from the truncated input; the rest already covered or unreachable) | n/a |
| diff, round 1 | OK | n/a | 0 (`CLEAN`) | 0 | 0 | no |

**Operational friction.**
- **Duplicate orchestration.** Both sessions read `ListAgents`, but neither checked who owned the worktree, and the PO's GATE 1 answers went to different conversations. One analyst invocation (153k tokens) was wasted, and the PO had to understand two parallel runs. Fixed in `.claude/commands/us.md` step 0: `changes/<ID>/owner`, re-checked before every agent launch, gate and checkpoint (`59dc404`, PO yes).
- **The two GATE 1 interpretations differed on one detail.** For an unknown payment method, B's code shows the raw key, while A's contract edit said `Otro método`. The code was kept, because the backend only accepts cash, transfer, card and credit (`internal/service/sale.go:142`).
- **The implementer stalled again** (B ran before the run-25 stdin rule was committed).

### What to measure in run 27
- Whether the `owner` check prevents a repeat, and whether any agent stalls now that the stdin rule is in every prompt.

## Run 25 — US-092 (opening fiado balance per customer, level L)

Already registered in SPRINT-20. Started on 2026-11-08 and finished on 2026-11-09 in the same session, after an account rate limit (5-hour window) interrupted the analyst. Every interrupted agent was resumed with `SendMessage` and kept its transcript. US-096 merged to `dev` from another session while this run was in QA, so `dev` was merged into `us/US-092` before the close (clean; `go test ./internal/...` green) and the analyst was told the next free R-number.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract, cut by rate limit; contract resume; GATE 1 decisions; close) | 200.7k (136.3k → 141.2k → 200.7k; the pre-limit part was not reported) | 59+ | ~18.8+ min |
| test-author | 1 | 140.4k | 42 | ~11.1 min |
| implementer | 2 (stalled; resumed) | 154.1k | 65 | ~20.0 min (incl. a 600 s stall with no file changes) |
| qa | 1 round | 107.9k | 42 | ~13.2 min |

**What it found.** This US changes a business rule. The contract chose a separate table `customer_opening_balances` (migration 22) over the alternatives:
- A fake credit sale would leak into about 8 sales queries.
- A customer column would be wiped by the full-row customer update (`internal/repository/customer.go:92`).

The analyst ran the queries it relied on and found three facts:
- SQLite's two-argument `min(NULL, x)` returns NULL, so the oldest-debt expression needs a `COALESCE`.
- Credit `created_at` is UTC text, so the debt date is stored in the same layout.
- The UNIQUE violation surfaces as error 2067, which the existing helper detects.

Discrepancies:
- **D-1:** US-096's invariant wording ("credit − abonos") is reconciled at close to opening + credit − abonos.
- **D-2:** stale line refs in the spec.
- **D-3:** customer buttons show for cashiers (R-144).
- **D-4:** customers may already exist, so a detail-page entry point was added.

GATE 1: Q-1..Q-5 as recommended, and Q-6 tuteo.

All 15 tests were RED for the right reasons. QA round 1: OK, 0 BLOCKER, 5 MINOR; 20 of 21 mutations caught (the note-trim mutant survived, R-142); API smoke test on the binary passed. After QA, the PO decided that a correction may leave a negative balance, documented in the spec, with a UI warning on the roadmap (R-141).

**Unplanned stops:** 3: the rate limit, the implementer stall, and the PO decision on QA MINOR 4. Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | TIMEOUT (`agy` print timeout, partial empty answer) | 7 min | 0 | 0 | 0 | n/a. The input was again truncated (20.7k of 30.7k) |
| diff, round 1 | OK | ~3 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 5 MINOR covered a test gap, UX, check order and an undecided rule |

**Operational friction.**
- **The implementer stalled for 600 s with zero file changes** and was killed by the stream watchdog. After a `SendMessage` resume it finished normally. In the same run the test-author left a stray `python -` background process waiting on stdin, which the permission classifier did not let it kill. Commands that wait on stdin are the likely cause of both.
- **QA could not smoke-test the UI**, because a plain `go build` embeds an empty `web/out`. US-091's export step exists, but QA only uses it when told to.
- **A parallel session merged US-096 mid-run.** Merging `dev` into the US branch before the close avoided the run-23 R-number collision.

### Proposed adjustments (not applied; need the PO's yes)
1. Earlier pending proposals unchanged (runs 12..16, 18-2, 19-2, 20-2, 21-2, 22-2, 23-2..23-5).
2. **Applied 2026-11-09.** Every agent prompt says: never run a command that reads stdin (`python -` without a heredoc, interactive tools), and bound long commands with a timeout.
3. **Applied 2026-11-09.** QA smoke builds the binary with the real export (`pnpm build` + `go run scripts/copy-frontend.go` in a scratch copy) whenever the US touches the frontend, so the UI is at least loaded once.
4. Before step 5 (close), the orchestrator merges `dev` into the US branch if `dev` moved, and passes the next free R-number to the analyst. This was done by hand in this run.

### What to measure in run 26
- Whether proposal 2 removes the agent stalls.

## Run 24 — US-096 (money QA layer 1: property-based invariant tests over random sequences of money operations, level L)

Prioritized by the PO on 2026-11-08 (roadmap R-130). Contract to close ran in one session (2026-11-08). Renamed from US-095 at GATE 1 due to ID collision with the UI void button on unmerged `us/US-091`. Subagents were deployed for QA round 2 and analyst close.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 4 (contract; GATE 1 updates; Q-8 add; close) | 192.4k (162.1k → 192.4k) | 75 | ~14.4 min |
| test-author | 1 | 202.2k | 63 | ~45.4 min |
| implementer | 1 | 56.3k | 12 | ~3.7 min |
| qa | 2 rounds | ~85k | 30 | ~10.0 min |

**What it found.** A high-impact property testing and fuzzing layer for money operations using `pgregory.net/rapid` v1.3.0.
100 random sequences of ~30 operations drove `h.Routes()` checking 6 invariants after every step (payments vs total, register expected cash, customer balance consistency, exact void reversal of register/balance/stock, rejection preservation, and positive/whole amounts).

Six server-side validation gaps were uncovered and fixed with 400 responses:
- `opening_amount < 0` (AC-12)
- movement `amount <= 0` (AC-13)
- sale `discount < 0` (AC-14)
- payments vs server-recalculated total discrepancy > 0.5 (AC-15)
- sale `discount > subtotal` (AC-16, found by Gemini)
- recalculated sale `total <= 0` (AC-17, Q-8)

Test-author: T-0..T-12 implemented; 6 validation tests and `FuzzSaleAmounts` went RED against base, green after implementation. T-6 (whole-peso check on full domain) correctly RED under build tag `moneyexact` with shrunk counterexample ($1.001 at 19% tax -> 190.19 tax).
QA: 3/3 mutations caught (m1 register cashAbonos, m2 customer completed filter, m3 stock restore). Seed reproducibility verified (identical counterexample with `-rapid.seed`). Binary smoke test passed all new 400 responses.

**Unplanned stops:** 1 (QA round 1: T-1 + T-2 took 28.4 - 30.8 s, exceeding the 30 s AC-8 budget in 3 of 5 runs; PO approved Option 1 to raise budget to <= 40 s without weakening tests). QA round 2: PASS (17.5 s wall time).

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~3 min | 11 | 3 (G-8 sale discount > subtotal; G-11 close closed/no-open missing in AC-6; G-4 actor role) | 8 | n/a |
| diff, round 1 | OK | ~3 min | 0 (`CLEAN`) | 0 | 0 | no |

**Operational friction.**
- **AC-8 time budget calibration:** A 30 s limit for 100 sequences of ~30 HTTP steps across two properties was too tight for CI spread on Windows (~28-31 s). Raising to <= 40 s avoided stripping assertions or reducing sequence counts.
- **Out of scope finding (R-137):** Recalculated total NaN passes `recalcTotal <= 0` and is rejected by the SQLite driver with 400; sent to roadmap for clean float validation.

### Proposed adjustments (not applied; need the PO's yes)
1. Earlier pending proposals unchanged.
2. For rapid state machine tests in CI, set wall-time budgets with a ~30% buffer over average single-run measurements to absorb Windows scheduling variance without triggering false blocker escalations.

### What to measure in run 25
- CI execution time on GitHub Actions with the new property test suite added to `internal/handler`.

## Run 23 — US-091 (end-to-end certification of register close and fiado math, and F5 on every route of the binary, level L)

Already registered in SPRINT-20. Contract to merge ran in one session (2026-11-08). The analyst was resumed with `SendMessage` twice (for the GATE 1 decisions and for the close) although `ListAgents` showed it as completed.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; GATE 1 decisions; close) | 192.9k (144.7k → 169.9k → 192.9k) | 75 | ~13.6 min |
| test-author | 1 | 87.8k | 26 | ~6.0 min |
| implementer | 1 | 33.3k | 21 | ~3.7 min |
| qa | 1 round | 84.8k | 27 | ~8.3 min |

**What it found.** This was a certification US. The analyst ran the sprint's scripted shift through `h.Routes()`, and the draft totals ($157.000 close, $15.000 balance) were right. The backend already met every money AC, so no production Go code changed.

Discrepancies:
- **D-1:** the "no cash abono without an open register" rule already existed (US-082).
- **D-2:** F5 already worked at the server. The draft's "same page" claim did not hold for the 4 redirecting public pages, and unknown paths return the home page with 200.
- **D-3:** the dry run asks for a void, but there is no void action in the UI. This led to US-095.
- **D-4:** transfer and card abonos show on no close figure (R-134).
- **D-5:** an HTTP close on `:memory:` failed 174 of 200 runs, so the shift tests use a file DB.
- **D-6:** `scripts/copy-frontend.go` deleted the tracked `web/out/.gitkeep`.

Test-author: T-1..T-6 passed on the base and T-7 was RED, as the contract expected. The PO accepted this at the PASSES⚠️ stop. The implementer only touched the copy script (`.gitkeep`, an `Fprintf` argument bug, Rule 0 translation) and added a CI step that runs the real-export test.

QA round 1: OK, 0 BLOCKER, 3 MINOR (R-135, R-136). QA ran 18 mutations on a `git archive` copy and the tests caught all 18. The shift tests passed 20 of 20 repeated runs, and the smoke test on the binary passed.

**Unplanned stops:** 1 (PASSES⚠️ review, pipeline-mandated), plus 1 merge conflict at GATE 2 (see below). Internal re-dispatches: 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~5 min | 10 | 1 (AC-6 did not state the admin abono amount or the resulting balance) | 9. Three came from input truncated at 20.7k of the 27.6k contract (same as run 18); the others were out of scope, already covered, or manual by design | n/a |
| diff, round 1 | OK | ~3 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 3 MINOR were test robustness and a pre-existing swallowed error |

**Operational friction.**
- **Parallel session collision.** While this run was at GATE 2, another session committed R-130..R-133 (money QA layers) to `dev` (`c7de6c2`) and opened `us/US-096`. US-091 had opened R-130..R-132 in its worktree, so `merge --squash` conflicted in `roadmap.md`. The orchestrator resolved it by renumbering the US-091 items to R-134..R-136 (roadmap, `caja.md`, `servidor.md`, SPRINT-20). The other session had already caught the US-095 collision and renamed its US to US-096.
- **Cleanup ran after the failed merge.** The orchestrator chained the cleanup with `;` after `merge --squash`, so the worktree and branch were removed before the conflict was seen. The branch was recreated from its hash (`fac6ab1`) and nothing was lost.
- The Gemini contract input truncation (run 18) again produced false "missing section" findings on a long contract.

### Proposed adjustments (not applied; need the PO's yes)
1. Earlier pending proposals unchanged (runs 12..16, 18-2, 19-2, 20-2, 21-2, 22-2).
2. Raise or remove the Gemini contract input limit (run 18 proposal). This is the second run where truncation produced most of the false positives.
3. For a certification US (backend already passes when the contract runs it), the PASSES⚠️ stop is expected. Let the contract mark those tests as "certifying", so the stop only fires for tests not marked that way.
4. New roadmap R-numbers and US IDs are reserved at merge, not when an analyst writes them in a worktree: at GATE 2, the orchestrator greps `dev` for the IDs the US opened and renumbers before merging. Parallel sessions make this collision likely.
5. In `.claude/commands/us.md` GATE 2, run the cleanup only after the merge commit succeeds (`&&`, not separate lines).

### What to measure in run 24
- Whether the new CI export step (`pnpm build` + copy + `go test -tags export`) stays around 30 s in GitHub Actions.

## Run 22 — US-090 (certify every sale flow with DIAN turned off, level M)

Already registered in SPRINT-20. Contract to close ran in one session (2026-11-07); the merge was done in the next session (2026-11-08) after the PO's GATE 2 "yes". Level M: no test-author, the implementer wrote tests and code. Metrics come from `changes/US-090/progress.md` and the `<usage>` blocks of the earlier session transcript. That session did not log the qa and close token lines, so the qa and analyst-close values below are matched by order and are approximate.

| Agent | Invocations | Tokens (last value) | Tools | Duration |
|---|---|---|---|---|
| analyst | 3 (contract; AC-10 fix; close) | ~124k (91.4k after contract) | ~53 | ~7+ min (contract 5.3 min, fix 1.8 min, close not logged) |
| implementer | 1 | 72.5k | 29 | ~6.9 min |
| qa | 1 round | ~64.7k | 37 | ~6.3 min |

**What it found.** The backend already met AC-1..AC-8 when the analyst ran them in the contract, so production Go code was not changed. D-1: the Settings DIAN section called `/fiscal-config`, which does not exist (404); it now uses `/fiscal/config`. D-2: the section always rendered, against the spec; it now renders only when `enabled === true`. D-3: a false "nil if FE disabled" comment was corrected. D-4..D-7 (emission context, buyer-email rule, void deadlock after DIAN is turned off, DIAN cannot be fully enabled) went to the roadmap. QA ran mutations on a `git archive` copy, and every one made a test fail. The log-buffer assertion also catches `slog` output.

**Unplanned stops:** 0. QA rounds: 1. Violations: 0.

**Gemini (`scripts/gemini-review.sh`, advisory).**
| Mode | STATUS | Duration | Raised | Verified true | False positives | Caught that QA missed |
|---|---|---|---|---|---|---|
| contract | OK | ~5 min | 13 | 1 (AC-10 contradicted the preamble wording) | 12 | n/a |
| diff, round 1 | OK | ~2 min | 0 (`CLEAN`) | 0 | 0 | no; QA's 4 MINOR were test strength, a toast message, spec timing and a pre-existing swallowed error |

**Operational friction.**
- The session ended at GATE 2; the next session resumed from `progress.md` with no loss. That is the progress log working as intended.
- `progress.md` left the qa and close metric lines as `n/a`, so they had to be recovered from the transcript.

### Proposed adjustments (not applied; need the PO's yes)
1. Earlier pending proposals unchanged (runs 12..16, 18-2, 19-2, 20-2, 21-2).
2. The orchestrator writes a `metrics` line to `progress.md` for every agent notification, not only some, so a run that crosses sessions keeps its numbers.

### What to measure in run 23
- Whether level M stays under ~250k tokens when the backend already passes (an audit-type US).

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
