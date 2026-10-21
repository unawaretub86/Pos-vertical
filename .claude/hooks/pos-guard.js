#!/usr/bin/env node
// Pos-vertical rule guard for Claude Code. Enforces mechanical rules of CLAUDE.md.
//   pos-guard.js bash-pre   PreToolUse(Bash):       pnpm only, no commit on master/main, spec with service/handler changes
//   pos-guard.js edit-post  PostToolUse(Edit|Write): applied migrations, t.Skip, CGO, BETWEEN, SQL concat, hard-coded colors, Spanish docs
// exit 2 = block (stderr goes to the model). Heuristic findings use additionalContext (warn) or "ask".
// Hooks must never crash a session: any internal error exits 0.
const { execFileSync } = require('node:child_process');
const path = require('node:path');
const fs = require('node:fs');

const mode = process.argv[2];
let raw = '';
process.stdin.on('data', (c) => (raw += c));
process.stdin.on('end', () => {
  let input;
  try { input = JSON.parse(raw); } catch { process.exit(0); }
  try {
    if (mode === 'bash-pre') bashPre(input);
    else if (mode === 'edit-post') editPost(input);
  } catch (e) { process.stderr.write(`pos-guard: internal error ignored: ${e.message}\n`); }
  process.exit(0);
});

const git = (cwd, args) => execFileSync('git', args, { cwd, encoding: 'utf8', maxBuffer: 32 * 1024 * 1024, stdio: ['ignore', 'pipe', 'ignore'] });
const block = (msg) => { process.stderr.write(`pos-guard: ${msg}\n`); process.exit(2); };
const warn = (event, msg) => {
  process.stdout.write(JSON.stringify({ hookSpecificOutput: { hookEventName: event, additionalContext: `pos-guard warning: ${msg}` } }));
  process.exit(0);
};
const ask = (msg) => {
  process.stdout.write(JSON.stringify({ hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'ask', permissionDecisionReason: `pos-guard: ${msg}` } }));
  process.exit(0);
};

function bashPre(input) {
  const cmd = input?.tool_input?.command ?? '';
  if (/(?:^|[;&|(]\s*|\bthen\s+|\bdo\s+)(?:npm|yarn)\s+(?:install|i|ci|add|uninstall|remove)\b/.test(cmd.trim())) {
    block('this project uses pnpm, never npm/yarn (CLAUDE.md > Stack). Use `pnpm add|install|remove` inside frontend/.');
  }
  if (!/\bgit\b[^|;&]*\bcommit\b/.test(cmd)) return;

  const m = cmd.match(/git\s+-C\s+("[^"]+"|'[^']+'|\S+)/);
  const cwd = m ? m[1].replace(/^["']|["']$/g, '') : (input.cwd || process.cwd());
  let branch = '';
  try { branch = git(cwd, ['rev-parse', '--abbrev-ref', 'HEAD']).trim(); } catch { return; }
  if (branch === 'master' || branch === 'main') {
    block(`refusing to commit on ${branch}. Work on dev or a us/<ID> branch (AGENTS.md > Git).`);
  }
  let staged = [];
  try { staged = git(cwd, ['diff', '--cached', '--name-only']).split('\n').filter(Boolean); } catch { return; }
  const code = staged.filter((f) => /^internal\/(service|repository|handler)\/.+\.go$/.test(f) && !f.endsWith('_test.go'));
  const spec = staged.some((f) => f.startsWith('project-docs/spec/'));
  if (code.length && !spec) {
    ask(`staged backend code (${code.slice(0, 3).join(', ')}${code.length > 3 ? ', ...' : ''}) without any project-docs/spec/ change. If a business rule changed, the spec goes in the same commit (CLAUDE.md Rule 2). Continue only if no rule changed.`);
  }
}

function editPost(input) {
  const fp = input?.tool_input?.file_path;
  if (!fp) return;
  const abs = path.resolve(input.cwd || process.cwd(), fp);
  let root;
  try { root = git(path.dirname(abs), ['rev-parse', '--show-toplevel']).trim(); } catch { return; }
  const rel = path.relative(root, abs).split(path.sep).join('/');
  if (rel.startsWith('..')) return;

  let diff = '';
  try { diff = git(root, ['diff', 'HEAD', '-U0', '--no-color', '--', rel]); } catch { /* untracked or no HEAD */ }
  if (!diff && fs.existsSync(abs)) {
    // new file: treat every line as added
    diff = '+++ b/' + rel + '\n' + fs.readFileSync(abs, 'utf8').split('\n').map((l) => '+' + l).join('\n');
  }
  const added = [], removed = [];
  let ctx = '';
  const removedInOldMigration = [];
  for (const line of diff.split('\n')) {
    if (line.startsWith('@@')) { ctx = line.replace(/^@@[^@]*@@/, ''); continue; }
    if (line.startsWith('+++') || line.startsWith('---')) continue;
    if (line.startsWith('+')) added.push(line.slice(1));
    else if (line.startsWith('-')) { removed.push(line.slice(1)); if (/func\s+migration\d+\b/.test(ctx) && line.slice(1).trim()) removedInOldMigration.push(line.slice(1).trim()); }
  }
  const warnings = [];

  if (rel === 'internal/database/migrate.go' && removedInOldMigration.length) {
    block(`${rel}: lines were removed from an existing migration function (${removedInOldMigration[0].slice(0, 60)}...). Never edit an applied migration; add the next numbered one (CLAUDE.md > Architecture). Revert with git checkout and add a new migration.`);
  }
  if (rel.endsWith('_test.go')) {
    const skip = added.find((l) => /\bt\.Skip(Now|f)?\s*\(/.test(l));
    if (skip) block(`${rel}: adds ${skip.trim()}. Never skip a test; fix it or ask Andrey.`);
    const weakened = removed.filter((l) => /\b(t\.(Error|Errorf|Fatal|Fatalf)|assert\.|require\.)/.test(l));
    if (weakened.length) warnings.push(`${rel}: removed assertion line(s) from an existing test (e.g. "${weakened[0].trim().slice(0, 70)}"). Do not weaken tests; if the old behavior changed on purpose, say so.`);
  }
  if (rel === 'go.mod' && added.some((l) => /mattn\/go-sqlite3|go-sqlite3|\bcgo\b/i.test(l))) {
    block('go.mod: CGO dependency (mattn/go-sqlite3 or similar). The product ships as a pure-Go single exe; CGO needs an explicit PO decision (CLAUDE.md > Stack).');
  }
  if (/^internal\/.+\.go$/.test(rel) && !rel.endsWith('_test.go')) {
    if (added.some((l) => /\bBETWEEN\b/i.test(l))) warnings.push(`${rel}: BETWEEN found. Date ranges are half-open [from, to) (CLAUDE.md > Conventions).`);
    if (added.some((l) => /fmt\.Sprintf\(\s*[`"]\s*(SELECT|INSERT|UPDATE|DELETE)\b/i.test(l))) warnings.push(`${rel}: SQL built with fmt.Sprintf. Use ? parameters; only constant fragments may be concatenated.`);
  }
  if (/^frontend\/.+\.(tsx|ts)$/.test(rel) && !/\.test\.tsx?$/.test(rel)) {
    const hard = added.filter((l) => /#[0-9a-fA-F]{3,8}\b(?![\w-])|\brgba?\(\s*\d/.test(l) && !/^\s*(\/\/|\*)/.test(l));
    if (hard.length) warnings.push(`${rel}: hard-coded color (${hard[0].trim().slice(0, 60)}). Use CSS variables such as var(--surface-2) or the dark theme breaks.`);
  }
  if (/\.md$/.test(rel) && /^(project-docs\/|\.claude\/|CLAUDE\.md$|GEMINI\.md$)/.test(rel) && !/^project-docs\/(archive|research)\//.test(rel)) {
    const es = spanishLines(added);
    if (es.length) warnings.push(`${rel}: ${es.length} added line(s) look Spanish (e.g. "${es[0].trim().slice(0, 70)}"). Docs are English only (CLAUDE.md Rule 0); UI strings and domain terms in quotes/backticks are fine.`);
  }
  if (warnings.length) warn('PostToolUse', warnings.join(' | '));
}

// Heuristic: strip code, quoted strings and links, then count Spanish accents/stopwords.
function spanishLines(lines) {
  const STOP = new Set(['el', 'la', 'los', 'las', 'del', 'para', 'que', 'con', 'una', 'por', 'como', 'pero', 'sin', 'esta', 'este', 'cuando', 'sobre', 'tambien', 'también', 'debe', 'ser', 'más', 'si', 'se']);
  const out = [];
  for (const l of lines) {
    const s = l.replace(/`[^`]*`/g, ' ').replace(/"[^"]*"|“[^”]*”|'[^']*'/g, ' ').replace(/\]\([^)]*\)/g, ' ').replace(/[#>*|_\-]/g, ' ');
    const words = s.toLowerCase().match(/[a-záéíóúñü]+/g) || [];
    if (words.length < 4) continue;
    const hits = words.filter((w) => STOP.has(w)).length + (/[ñáéíóú¿¡]/i.test(s) ? 2 : 0);
    if (hits >= 3 && hits / words.length > 0.2) out.push(l);
  }
  return out;
}
