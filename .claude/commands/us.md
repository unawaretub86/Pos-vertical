---
description: Runs a US through the 4-agent SDD pipeline (analyst → test-author → implementer → qa → analyst) with two PO gates
argument-hint: US-XXX [M|L]
---

Orchestrate US **$ARGUMENTS** per `project-docs/tosdd-plan.md`. You are the orchestrator: you launch the subagents, verify what they do and stop at the gates. **You do not write code, tests or docs yourself.**

**Language:** everything written during this pipeline (contracts, QA reports, sprint updates, spec deltas, roadmap entries, gate summaries, commit messages, benchmark entries) is in **English**, never Spanish (CLAUDE.md Rule 0). Tell every subagent this in its prompt. Only end-user UI strings and verbatim user phrases used as test data may be Spanish.

If no level is given, use **L**. At level **M**, skip step 2 (test-author).

Conventions of this command:
- `WT` = absolute path of the US worktree.
- `ID` = the US identifier (`US-XXX`).
- Every git command runs with `git -C "$WT"`.
- To each subagent you pass in the prompt: the `ID`, the absolute path of `WT`, the sprint and the mode (if applicable).
- **Resume instead of relaunching (mandatory check, not optional):** before each `Agent` call in this pipeline, first run `ListAgents` and look for a live agent of the same role (`analyst`, `test-author`, `implementer`, `qa`) for this same `$ID`. If one shows up, use `SendMessage` to its name/ID instead of `Agent`. Only launch `Agent` (cold start) if `ListAgents` shows none alive for that role+US. This check was botched two runs in a row (3 and 4, see `project-docs/pipeline-bench.md`) by skipping it on reaching step 5 (`close`) — do not skip it there either.
- **Benchmark:** from each agent notification, record `subagent_tokens`, `tool_uses` and `duration_ms`, and keep count of unplanned stops (outside GATE 1 and 2). They are logged in step 6.
- **Progress log (run 17):** after each agent notification, each checkpoint commit, each Gemini result and each gate decision, append one line to `$WT/changes/$ID/progress.md`: `<ISO time> | <step> | <agent/event> | tokens <n> | tools <n> | <duration> | <outcome or PO decision, one clause>`. For Gemini, include raised / verified / false positives. Subagent transcripts do not cross sessions, so this file is what lets an interrupted run (rate limit, session end) be resumed and benchmarked. It is ignored by git and disappears with the worktree after step 6 has used it.
- **No stdin, bounded commands (run 25):** tell every subagent in its prompt to never run a command that reads stdin (`python -` without a heredoc, interactive tools, prompts) and to bound long commands with a timeout, running servers in the background. In run 25 such a command stalled an agent for 600 s and left a stray process.
- gopls errors inside the worktree ("not included in your workspace", `undefined` symbols) are IDE noise; what counts is `go build` and `go test`.

## 0. Prepare the worktree

1. In the main checkout, run `git status --porcelain`. If there are uncommitted changes in `CLAUDE.md`, `project-docs/`, `.claude/` or the code, **stop and tell Andrey**: the worktree is cut from the last commit of `dev` and the agents would not see those changes.
2. Locate the US with Grep in `project-docs/sprints/`. If it does not exist but `$ID` matches an `R-NN` item of the roadmap (`project-docs/roadmap.md`), it is a US coming from the roadmap: do not register it here; the `analyst` registers it in step 1 (mode `contract`) and the orchestrator commits it separately before `RED` (see step 1). If it exists, confirm it is still `[ ]`.
3. Create the worktree: `git worktree add "../Pos-vertical-wt/$ID" -b "us/$ID" dev`. If the branch already exists, read `$WT/changes/$ID/progress.md` and `git -C "$WT" status` to see where it stopped, then ask Andrey whether to resume or discard it. On resume, agents from an earlier session cannot be resumed with `SendMessage`; launch them cold and tell them what the worktree already holds.
4. `mkdir -p "$WT/changes/$ID"`.

## 1. Contract — agent `analyst`, mode `contract`

Launch `analyst`. If `$ID` comes from the roadmap (step 0.2), also tell it to register the US in the matching sprint during this same `contract` mode. Then verify that `$WT/changes/$ID/contract.md` exists and has the 8 sections of the template.

If the analyst touched the sprint (US registration from the roadmap), make that checkpoint separately, **before** `RED`: `git -C "$WT" commit -am "docs($ID): register US"`.

### Gemini second opinion on the contract (advisory)

Right after verifying `contract.md`, run `bash scripts/gemini-review.sh contract "$ID" "$WT"` (Bash tool, `run_in_background`; takes about 2-5 minutes, up to ~14 with a retry, never fails the pipeline). It writes `$WT/changes/$ID/gemini-contract.md` with `STATUS: OK | EMPTY | TIMEOUT | NO_AGY`. Do not wait for it more than a few minutes: if it is not ready when you present GATE 1, present without it and say so.

If `STATUS: OK` and not `CLEAN`, **verify each line yourself** against the contract and the code (`file:line`). Show Andrey only the ones that hold up, under a "Gemini (verified)" heading, and list the discarded ones with the reason. Gemini never changes the contract: only the `analyst` does, after Andrey decides.

### ◆ GATE 1 — Andrey approves the contract

Show the analyst's summary, the DISCREPANCIES and the questions for the PO, with the path to the contract. **Stop and wait.** If Andrey asks for changes, relaunch `analyst` with his corrections.

If Andrey's corrections only touch §7 (Out of scope) and §8 (Questions for the PO) of the contract — that is, they do not change acceptance criteria or test cases — you may launch the step 2 `test-author` **in parallel** with the analyst's adjustment, instead of waiting for the analyst to finish. Do not advance to step 3 without an explicit "yes" from Andrey on the contract.

## 2. Red tests — agent `test-author` (level L only)

Launch `test-author`. Then verify:
- `git -C "$WT" status --porcelain` shows only `*_test.go`, `*.test.ts` or `*.test.tsx` (and `changes/`, which is ignored). If there is any other file, revert with `git -C "$WT" checkout -- <file>` (or delete it if new) and relaunch the agent stating the violation.
- Its table has no `PASSES⚠️` tests. If there are any, show them to Andrey before continuing.

Checkpoint: `git -C "$WT" add -A && git -C "$WT" commit -m "test($ID): red tests from contract"`. Save the hash as `RED`.

### 2b. If the contract changes after red

If a stop (a `PASSES⚠️`, something the contract does not resolve) leads the PO to decide and the analyst changes the contract **before** the `RED` checkpoint, resume the `test-author` so it adjusts or adds only the affected tests. Repeat the step 2 verifications and only then make the checkpoint. If the change arrives **after** `RED`, make a second commit `test($ID): contract update` and use that hash as `RED`.

## 3. Implementation — agent `implementer`

Launch `implementer`. Then verify:
- `git -C "$WT" diff RED -- '*_test.go' '*.test.ts' '*.test.tsx'` is empty (at level M, compare against the branch base commit). If it is not, revert those files and treat it as a BLOCKER of the round.

Checkpoint: `git -C "$WT" commit -am "feat($ID): implementation"` (add new files with `add -A` first).

## 4. QA — agent `qa`

1. Snapshot: `git -C "$WT" status --porcelain` and `git -C "$WT" rev-parse HEAD`.
2. Launch `qa` and give it the round number and the `RED` hash. **In the same turn**, start Gemini in parallel: `bash scripts/gemini-review.sh diff "$ID" "$WT" <round> "$RED"` (Bash tool, `run_in_background`). It writes `$WT/changes/$ID/gemini-diff-round-<round>.md`. Gemini only reads the diff and writes under `changes/`; it cannot edit code.
3. Snapshot again. If QA changed anything outside `changes/`, revert with `git -C "$WT" checkout -- .` (and `git -C "$WT" clean -fd -e changes/` for new files) and note the violation in `qa.md`.
4. Wait for QA. Give Gemini until about 10 minutes after launch; if it has not finished, proceed without it and record `no answer` (its `STATUS` may be `EMPTY`/`TIMEOUT`; that is not a failure, record it). **Verify every Gemini finding** against the code with `file:line`. A verified finding Claude's QA missed is added to the round's BLOCKER/MINOR list (note `found by Gemini` in `qa.md`); one that does not hold up is discarded and counted as a false positive. A Gemini `CLEAN` never overrides a QA BLOCKER, and never turns a BLOCKER into OK.
5. Read the verdict (QA's, plus any verified Gemini BLOCKER):
   - **BLOCKER** and it is round 1 or 2 → go back to step 3, passing the implementer only that round's BLOCKERs.
   - **BLOCKER** in round 3 → **stop and escalate to Andrey**. Two rounds without converging indicate the problem is in the contract.
   - **OK** → step 5.

## 5. Close — agent `analyst`, mode `close`

Before launching anything: `ListAgents` and look for the step 1 `analyst` of this `$ID`. If still alive, `SendMessage` with mode `close`; if not, only then a cold `Agent`. Then verify with `git -C "$WT" status` that only `project-docs/` changed and make the checkpoint: `git -C "$WT" commit -am "docs($ID): spec delta and sprint closure"`.

### ◆ GATE 2 — Andrey approves

Show:
- `git -C "$WT" diff dev...HEAD --stat`;
- the final QA verdict with pending MINORs;
- the spec, sprint and roadmap changes.

**Stop and wait.** With an explicit "yes":

```
git checkout dev
git merge --squash "us/$ID"
git commit        # message: feat($ID): <US name>, plus the Co-Authored-By trailer
git worktree remove --force "../Pos-vertical-wt/$ID"
rm -rf "../Pos-vertical-wt/$ID"   # on Windows ignored files remain (node_modules, .next)
git worktree prune
git branch -D "us/$ID"
```

Before `merge --squash`, if the main checkout has staged files unrelated to the US, remove them from the index (`git restore --staged`) and re-add them after the commit, so they are not mixed in.

`changes/$ID/` disappears with the worktree; that is expected. **Do not push** unless Andrey asks.

## 6. Benchmark

Before the merge commit, add to `project-docs/pipeline-bench.md`:
- a row in the "Runs" table;
- a "Run N — $ID" section with: per-agent table (invocations, last-value tokens, sum of tools and duration), what it found, unplanned stops and why, operational friction and what to measure next.

For Gemini, record per round: STATUS, duration, findings raised, verified true, false positives, and anything it caught that QA missed (this is the data that decides how far to trust it).

Include it in the same merge commit. If you spot an adjustment to the pipeline, **propose it to Andrey**; do not apply it without his "yes".

## Orchestrator rules

- Never skip a gate. Never merge without the GATE 2 "yes".
- Gemini is advisory. Never act on a Gemini finding you have not verified, and never let it be the only reviewer.
- Do not fix an agent's work with your own hands: relaunch it with the reason.
- If an agent reports something the contract does not resolve, ask Andrey; do not decide scope.
- Report faithfully: if a step was skipped or failed, say so.
