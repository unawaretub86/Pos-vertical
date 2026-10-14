# GEMINI.md

The rules for this repository are in `CLAUDE.md` at the repository root. Before answering or changing anything, read `CLAUDE.md` in full and follow it: it overrides your defaults. If `CLAUDE.md` conflicts with a request, say so instead of ignoring it.

Hard rules you must not miss (full text in `CLAUDE.md`):
- Everything written in the repo (docs, specs, comments, commit messages, identifiers) is English. Only end-user product text is Spanish.
- Verify every claim about the code with a `file:line` or command output; do not repeat documentation without checking it.
- Historically referenced entities are deleted logically (`active = 0`), never with a physical `DELETE`.
- Date ranges are half-open `[from, to)`; never `BETWEEN`.
- Never edit an already-applied migration; add the next one.
- Do not commit unless asked; do not touch unrelated files.
