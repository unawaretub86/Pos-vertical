// US-062: logic behind the floating assistant chat. Pure functions plus a
// module-level store (compatible with React's useSyncExternalStore). The
// component in components/AssistantWidget.tsx is only a render shell.
//
// The history lives in this module on purpose: every page wraps itself in its
// own <AppLayout>, so component state would die on each navigation. A module
// variable survives SPA navigation and dies on F5. It is never persisted.

import { api } from './api';
import type { AuthUser } from './auth';

export interface AskResponse {
  intent: string;
  period?: { from?: string; to?: string; label?: string };
  text: string;
  data?: unknown;
  suggestions: string[];
  tier?: string;
  confidence?: number;
}

export interface Message {
  role: 'user' | 'assistant';
  text: string;
  intent?: string;
  suggestions?: string[];
}

export interface AssistantState {
  loading: boolean;
  messages: Message[];
}

export interface ParsedAnswer {
  intro: string;
  items: string[];
  footer: string[];
}

export const GREETING_TEXT =
  'Hola, soy tu asistente. Preguntame por tus ventas, tu cartera o tu inventario, o tocá una de estas:';

export const GREETING_CHIPS: string[] = ['cuánto vendí hoy', 'quién me debe', 'qué se está acabando'];

export const ERROR_TEXT = 'No pude consultar los datos. Intentá de nuevo.';

// ─── Network ─────────────────────────────────────────────────────────────────

export function askAssistant(query: string): Promise<AskResponse> {
  return api.post<AskResponse>('/ai/ask', { query: query.trim() });
}

// ─── Visibility rules ────────────────────────────────────────────────────────

export function canSeeAssistant(user: AuthUser | null): boolean {
  return user !== null && user.role === 'admin';
}

// The assistant is hidden (not unmounted) only on the exact POS screen.
export function isHiddenPath(pathname: string): boolean {
  return pathname === '/pos';
}

// ─── Answer rendering helpers ────────────────────────────────────────────────

const LIST_LINE = /^(?:\d+\.|-)\s+/;

// Splits the server text into the header line(s), the list lines (prefix
// removed) and whatever follows the list ("y N más", contrast line, "Ojo: ...").
// A text without list lines is returned whole as `intro`.
export function parseAnswerText(text: string): ParsedAnswer {
  const lines = text.split('\n');
  const first = lines.findIndex((l) => LIST_LINE.test(l));
  if (first === -1) return { intro: text, items: [], footer: [] };

  let end = first;
  while (end < lines.length && LIST_LINE.test(lines[end])) end++;

  return {
    intro: lines.slice(0, first).join('\n').trim(),
    items: lines.slice(first, end).map((l) => l.replace(LIST_LINE, '').trim()),
    footer: lines
      .slice(end)
      .map((l) => l.trim())
      .filter((l) => l !== ''),
  };
}

// Only the chips of the latest assistant answer are tappable (PO decision P-3).
export function activeChips(messages: Message[]): string[] {
  const last = messages[messages.length - 1];
  if (!last || last.role !== 'assistant') return [];
  const chips = last.suggestions ?? [];
  if (chips.length === 0 && last.intent === 'unknown') return [...GREETING_CHIPS];
  return chips;
}

// ─── Store ───────────────────────────────────────────────────────────────────

let state: AssistantState = { loading: false, messages: [] };
let boundUserId: number | null = null;
// Bumped whenever the history is reset, so an answer that was in flight for the
// previous user is discarded instead of landing in the new user's history.
let epoch = 0;
const listeners = new Set<() => void>();

function setState(next: AssistantState): void {
  state = next;
  listeners.forEach((l) => l());
}

export function getState(): AssistantState {
  return state;
}

export function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

// Clears the history when the logged-in user differs from the history's owner.
export function bindUser(userId: number): void {
  if (boundUserId === userId) return;
  const hadOwner = boundUserId !== null;
  boundUserId = userId;
  if (!hadOwner) return; // first binding after load: nothing to clear
  epoch++;
  setState({ loading: false, messages: [] });
}

export async function send(text: string): Promise<void> {
  const query = text.trim();
  if (query === '' || state.loading) return;

  const myEpoch = epoch;
  setState({
    loading: true,
    messages: [...state.messages, { role: 'user', text: query }],
  });

  let reply: Message;
  try {
    const r = await askAssistant(query);
    reply = {
      role: 'assistant',
      text: r.text,
      intent: r.intent,
      suggestions: r.suggestions ?? [],
    };
  } catch {
    // Never expose the raw error ("forbidden", "HTTP 500", "Failed to fetch").
    reply = { role: 'assistant', text: ERROR_TEXT };
  }

  if (myEpoch !== epoch) return;
  setState({ loading: false, messages: [...state.messages, reply] });
}
