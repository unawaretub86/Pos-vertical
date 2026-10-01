# Plan — Agentic orchestration with Spec-Driven Development (SDD)

> **Type:** proposal · **Current:** pending PO approval · **Rules over:** how a US is developed with agents
> **Date:** 2026-09-23 · Verified against the repo on that date. Translated to English 2026-10-01 (`../CLAUDE.md` → Rule 0).
> **Revision 2 (2026-09-23):** from 6 to 4 agents, `changes/` outside git, worktree created at the start of the US.

**PO request:** an orchestration of agents to develop the tasks. A business agent (plan, review and update documentation), one that reads requirements, one that executes code, one that reviews it and one for unit tests and smoke tests, all supported by SDD or another current methodology.

This document says what is proposed, what was found in the repo and **why** each decision is recommended.

---

## 1. What SDD is and where this project stands

**Spec-Driven Development:** the spec is the source of truth. Code is derived from the spec and verified against it, never the other way around.

| Methodology | Core idea | Artifacts |
|---|---|---|
| **GitHub Spec Kit** | specify → plan → tasks → implement | `spec.md`, `plan.md`, `tasks.md` |
| **Kiro (AWS)** | Requirements in EARS format ("WHEN X, the system SHALL Y") | `requirements.md`, `design.md`, `tasks.md` |
| **OpenSpec** | Living spec + "changes" that carry a spec *delta* and are archived on close | `specs/` + `changes/<id>/` |

### Finding: the repo already works like OpenSpec

| OpenSpec concept | Already exists in the repo |
|---|---|
| Living spec | `project-docs/spec/domain/` (`fiado.md`, `caja.md`, `rentabilidad.md`, `facturacion-dian.md`, `glosario.md`) + `spec/product.md`, `spec/personas.md` |
| Unit of change | `project-docs/sprints/SPRINT-XX-*.md`, frozen on close (`project-docs/README.md`, table "The four types") |
| Syncing the spec delta | CLAUDE.md Rule 2 and DoD item 8 in `project-docs/workflow.md` |
| Verify before asserting | CLAUDE.md Rule 1 (`file:line`) |

**Recommended decision:** do not install Spec Kit or Kiro. Adopt the OpenSpec model on the current structure and add only a temporary working folder per US (`changes/US-XXX/`, outside git, see §3).

**Why:** installing a new framework would duplicate what already exists (`spec/`, sprints, roadmap) and create a second source of truth. That is exactly what Rules 2 and 3 of CLAUDE.md seek to avoid. What is missing is not methodology, it is **separating roles and putting checkpoints between them**.

---

## 2. The four agents

In Claude Code, a subagent is a file `.claude/agents/<name>.md` that defines role, allowed tools and model. User-level skills `ralph`, `ralph-workflow` and `task-planner` exist (in `~/.claude/skills/`), which can be reused inside the prompts.

| # | Agent | Merges | Role | Tools | Writes to |
|---|---|---|---|---|---|
| 1 | `analyst` | spec-keeper + requirements-analyst | Reads `spec/`, the sprint and the code; detects the spec delta and writes `contract.md`. On closing the US, consolidates the delta into `spec/` and closes the US in the sprint | Read, Grep, Glob, Edit, Write, Bash (for `git diff`) | `changes/US-XXX/contract.md`; at close, `project-docs/spec/` and `project-docs/sprints/` |
| 2 | `test-author` | — | Writes the tests **in red** from the contract, without touching the implementation | Read, Edit, Write, Bash (`go test`, `pnpm test`) | Only `*_test.go` and `*.test.ts` |
| 3 | `implementer` | — | Writes code until the tests are green | All | `internal/`, `frontend/` (does not touch tests) |
| 4 | `qa` | code-reviewer + smoke-runner | Reviews the diff against the contract and `CLAUDE.md`, runs the CI commands and a smoke test with the binary | Read, Grep, Glob, Bash, Write (only `qa.md`) | `changes/US-XXX/qa.md` |

### Why 4 and not 6

Each subagent starts cold and re-reads `CLAUDE.md`, the sprint and the code. With 6 agents that happened 6 times per US.
- **spec-keeper and analyst read exactly the same thing.** Separating them cost a cold start without gaining real independence.
- **reviewer and smoke-runner look at the same diff.** One reads it and the other runs it, and merging them gives QA both pieces of evidence for its verdict.

That is two fewer cold starts per US and one less communication channel that can drift out of alignment.

**Clarification about the Analyst:** being "the same agent" at the start and at the close means **the same definition**. If the session is still alive, the orchestrator **resumes** it with `SendMessage` and keeps its context. That was run 1: 3 resumptions took the Analyst from 136k to 169k tokens, against ~100k for each cold start. If the session was cut, it starts cold and resumes from `contract.md`. That is why the contract must still leave the spec delta written explicitly (section "Spec delta"), not implicit in the conversation.

### Why test-author and implementer stay separate

This is the separation that is not negotiable. If the same agent writes the code and its tests, the tests end up describing what the code *does*, not what the contract *asks for*. Separated:
- the tests come from the contract, with the concrete numbers CLAUDE.md demands (for example: "3 payments of $30k + $20k + $10k → total 60,000, remainder 0");
- the implementer cannot edit the tests to "make them pass", because its write permission excludes them.

### Why QA has Bash and how it is controlled

QA needs Bash to run `go test`, `pnpm build` and the binary. The cost is that it stops being read-only: with `sed -i` or `echo >` it could silently "fix" what it should report, and restricting tools does not prevent it.

**Control:** the orchestrator takes `git status --porcelain` and `git diff --stat` before and after QA. If QA left changes in the tree (except `changes/`), they are discarded with `git checkout` and recorded as a finding. QA reports, never fixes.

### Why the orchestrator is not an agent

Claude Code subagents cannot launch other subagents. Orchestration lives in the main thread (Andrey + the Claude session) and is packaged in a command `/us US-XXX` (`.claude/commands/us.md`).

---

## 3. Flow of a US

```
PO picks US ──► [0] orchestrator     creates worktree + branch us/US-XXX from dev
                        │            (EVERYTHING that follows happens inside the worktree)
                        ▼
                [1] analyst          reads spec/, sprint and code → changes/US-XXX/contract.md:
                        │            - criteria in EARS format
                        │            - verified state with file:line (Rule 1)
                        │            - list of tests with concrete numbers
                        │            - "Spec delta": which rule changes and how
                        ▼
              ◆ GATE 1 — Andrey approves the contract ◆
                        ▼
                [2] test-author      writes the tests → confirms they FAIL (red)
                        ▼
                [3] implementer      writes code → tests green
                        ▼
                    git status (snapshot)
                [4] qa               reviews the diff + runs CI + smoke → changes/US-XXX/qa.md
                    git status (did QA touch anything? → discarded and a finding)
                        │
                 BLOCKER? ──yes──► back to [3] (max 2 rounds, then escalate to Andrey)
                        │ no
                        ▼
                [1] analyst          applies the delta in spec/, moves the verified criteria
                                     to the sprint's US, marks it [x]; what did not make it goes to roadmap.md
                        ▼
              ◆ GATE 2 — Andrey approves → commit on the branch → merge to dev ◆
                        ▼
                [0] orchestrator     deletes the worktree (changes/US-XXX/ disappears with it)
```

### Why each step

| Step | Why |
|---|---|
| **[0] Worktree first** | `changes/` is in `.gitignore`, and a new worktree only brings versioned files. If the contract were written in `dev`, the test-author and the implementer would not see it from their worktree. By creating the worktree before the Analyst, all 4 agents work in the same directory and see the ignored files. Also, an implementer that errs does not dirty `dev`. |
| Analyst before everything | Avoids building on an outdated business rule. While planning SPRINT-14, Rule 1 caught four false claims inherited from old docs (CLAUDE.md, Rule 1). |
| Contract with `file:line` | It is where verification is born. If the contract is wrong, everything after it is wrong. |
| GATE 1 before coding | Fixing a contract is cheap; fixing already-written code is not. Andrey decides scope (CLAUDE.md: "Andrey is Product Owner and decides scope and priority"). |
| Red tests before code | Proves the test detects the absence of the feature. A test that passes without code proves nothing. |
| `git status` snapshot around QA | It is the only real guarantee that QA did not silently fix anything (§2). |
| Max 2 rounds | Avoids infinite loops between implementer and QA. If it does not converge in 2 rounds, the problem is in the contract, not the code. |
| Analyst at close, same commit | It is Rule 2 applied mechanically, not depending on someone remembering. |
| Verified criteria to the sprint | `contract.md` is lost with the worktree. What is worth keeping (the criteria with evidence) stays in the sprint's US, which is committed. |

### Artifacts and persistence

```
<US worktree>/changes/US-059/    ← in .gitignore
  contract.md    ← analyst
  qa.md          ← qa (one section per round)
```

**What reaches the commit:** the source code, the tests, the `spec/` update and the sprint with the US closed. Nothing else.

**Why outside git:** `contract.md` and `qa.md` are the communication channel between agents, not product documentation. Committing them would fill the repo with conversations between agents, which age just like the docs Rule 1 already forces us to distrust. The evidence that does matter survives in three versioned places: the tests, `spec/` and the sprint's US.

---

## 4. Example: `qa`

```markdown
---
name: qa
description: Reviews a US's diff against its contract.md and CLAUDE.md, runs CI and smoke. Reports, never fixes.
tools: Read, Grep, Glob, Bash
model: opus
---
You are QA. You do NOT edit files outside changes/<US>/. If something fails, you report it.
Any change of yours in the tree will be discarded and counted as a finding.

1. Review (each finding with file:line), reading contract.md and `git diff dev...HEAD`:
- Does every contract criterion have a test covering it?
- SQL: any BETWEEN on dates? Concatenation instead of `?`?
- Errors without `%w` or swallowed?
- map iteration where order matters?
- Was an already-applied migration edited in migrate.go?
- Frontend: hardcoded colors instead of var(--...)? formatCOP? does it work at 320px?
- Anything about sales/profit/receivables outside a RequireAdmin group?
- Did a business rule change without a "Spec delta" in the contract?

2. Execution: go build ./... · go test ./... · (frontend/) pnpm build · pnpm test ·
   smoke: start the binary and curl the endpoints the US touches.

Output: changes/<US>/qa.md with BLOCKER / MINOR / OK and the real output of each command.
```

**Why this checklist:** it comes literally from the "Architecture", "Conventions" and "Known traps" sections of CLAUDE.md. QA does not invent criteria: it enforces what the PO already decided.

---

## 5. Costs and levels

- **Tokens:** each agent re-reads the context from scratch. With 4 agents, a complete US costs roughly 2.5 to 4 times more than doing it directly. The cost that weighs most is the implementer ↔ QA rounds, not the number of agents.
- **Speed:** goes down. **Quality:** goes up. The human gates stay because they are where the PO decides.

| Level | When | Pipeline |
|---|---|---|
| **S** | Bug fix, text change | Direct, no agents |
| **M** | Change in a single layer, no business rule | worktree → analyst → implementer → qa |
| **L** | New feature or business-rule change | Full pipeline of 4 |

**Why:** using the pipeline to change a label is a waste. Its value appears when business rules are involved, above all in fiado, which is the product's differentiator. At M the test-author is skipped because the change is usually covered by adjusting existing tests; if the US adds new behavior, it is L.

---

## 6. Prerequisites — resolved on 2026-09-23

The reorganization of `project-docs/` had left broken pointers in `CLAUDE.md`, `README.md`, `workflow.md`, `roadmap.md` and three docs in `spec/domain/`, plus three `file:line` citations that already pointed to the wrong lines. Everything was fixed in the repo. The skills `ralph-workflow`, `ralph`, `pos-prodcheck` and `task-planner` were updated in `~/.claude/skills/`, **outside git**, with a prior backup.

**Why it mattered:** if the documents lie, the agents inherit the lie.

**Pending:** the bug tables and competitive context inside `pos-prodcheck/SKILL.md` are from the MVP era. Their pointers were fixed, but their content was not re-verified.

---

## 7. Execution plan

| Step | What | Why in this order |
|---|---|---|
| 1 | ~~Fix broken pointers~~ ✅ done 2026-09-23 | The agents depend on the pointers being true |
| 2 | ~~Pipeline infrastructure~~ ✅ done 2026-09-23 — see §8 | The minimum pipeline infrastructure |
| 3 | ~~Pilot with **US-059**~~ ✅ done 2026-09-23 (`cd2e590`), see `pipeline-bench.md` run 1 — (SPRINT-14: extract the dashboard metrics into a range-parameterizable repository) | Pure refactor, no visible feature and with clear tests. Calibrates the agents at low risk before something big like the tables module (SPRINT-13) |
| 4 | ~~Pilot retrospective~~ ✅ done 2026-09-23: 7 adjustments, listed in `pipeline-bench.md` run 1 | Prompts are calibrated with real use, not on paper |
| 5 | If the pilot works: add the pipeline to `workflow.md` and make this document current | Rule 2: if how we work changes, the doc that defines it changes |

---

## PO decisions

**Made (2026-09-23):**
- [x] 4 agents: analyst, test-author, implementer, qa.
- [x] `changes/US-XXX/` goes in `.gitignore`. Only the code, the tests, `spec/` and the closed sprint reach the commit.
- [x] Worktree created at the start of the US; the 4 agents work inside it.
- [x] Findings section condensed into prerequisites, keeping the `pos-prodcheck` pending item.

- [x] The two human gates are kept (GATE 1: contract; GATE 2: merge).
- [x] US-059 is the pilot.

**Made (2026-10-01):**
- [x] Every artifact the pipeline writes (contracts, QA reports, sprint/spec/roadmap updates, benchmark entries, commits) is in English (`CLAUDE.md` Rule 0). The agent definitions, `/us` command and `contract.md` template were translated accordingly.

---

## 8. Where each piece lives

| Piece | File | What it does |
|---|---|---|
| Orchestrator | `.claude/commands/us.md` | Command `/us US-XXX [M\|L]`. Creates the worktree at `../Pos-vertical-wt/US-XXX` (branch `us/US-XXX`), launches the agents, verifies with `git` that each only touched its own files, stops at the two gates and does the `--squash` merge to `dev` |
| Agent 1 | `.claude/agents/analyst.md` | Mode `contract` (writes the contract) and mode `close` (spec, sprint, roadmap) |
| Agent 2 | `.claude/agents/test-author.md` | Red tests, only `*_test.go` and `*.test.ts(x)` files |
| Agent 3 | `.claude/agents/implementer.md` | Code until green; does not edit tests |
| Agent 4 | `.claude/agents/qa.md` | Diff review + CI + smoke with the binary; only writes `qa.md` |
| Template | `.claude/templates/contract.md` | 8 sections: goal, verified state, ACs in EARS, tests, files, spec delta, out of scope, questions |
| Scratch | `changes/` in `.gitignore` | Lives inside the worktree and is deleted with it |
| Benchmark | `project-docs/pipeline-bench.md` | One row and one detail per `/us` run (tokens, time, stops, QA rounds, defects). Written by the orchestrator in step 6 |

**Checkpoints on the US branch:** `test(US-XXX)` (red) → `feat(US-XXX)` (green) → `docs(US-XXX)` (close). They let the orchestrator verify with `git diff` that the implementer did not touch the tests. A single commit reaches `dev` through the `--squash`.

**Smoke:** `cmd/pos/main.go:21` uses `data.db` from the current directory and `cmd/pos/main.go:41` fixes `localhost:8080`. That is why QA runs the binary only from the worktree (with a fresh, isolated `data.db`) and skips the smoke test if the port is busy, instead of killing Andrey's POS.

**Requirement to run `/us`:** `dev` with no pending changes in docs, `.claude/` or code. The worktree is cut from the last commit, so the agents do not see what is not committed.
