'use client';

import { useEffect, useRef, useState, useSyncExternalStore, FormEvent } from 'react';
import { usePathname } from 'next/navigation';
import { getUser } from '@/lib/auth';
import { useIsMobile } from '@/hooks/useIsMobile';
import {
  GREETING_CHIPS,
  GREETING_TEXT,
  activeChips,
  bindUser,
  canSeeAssistant,
  getState,
  isHiddenPath,
  parseAnswerText,
  send,
  subscribe,
  type AssistantState,
  type Message,
} from '@/lib/assistant';

const EMPTY_STATE: AssistantState = { loading: false, messages: [] };
const getServerState = () => EMPTY_STATE;

// Visual viewport box as a "top|height" string (a primitive keeps the snapshot
// stable). It shrinks when the on-screen keyboard opens, so the panel can follow
// it and the input is never covered. Empty when the API is unavailable.
function subscribeViewport(cb: () => void): () => void {
  const vv = window.visualViewport;
  if (!vv) return () => {};
  vv.addEventListener('resize', cb);
  vv.addEventListener('scroll', cb);
  return () => {
    vv.removeEventListener('resize', cb);
    vv.removeEventListener('scroll', cb);
  };
}
const getViewportSnapshot = () => {
  const vv = window.visualViewport;
  return vv ? `${vv.offsetTop}|${vv.height}` : '';
};
const getServerViewport = () => '';

function Chips({ chips, disabled }: { chips: string[]; disabled: boolean }) {
  if (chips.length === 0) return null;
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '4px' }}>
      {chips.map((chip) => (
        <button
          key={chip}
          type="button"
          disabled={disabled}
          onClick={() => void send(chip)}
          style={{
            padding: '8px 12px',
            minHeight: '36px',
            borderRadius: '18px',
            border: '1px solid var(--border-emphasis)',
            background: 'var(--surface-2)',
            color: 'var(--text-primary)',
            fontSize: '13px',
            cursor: disabled ? 'not-allowed' : 'pointer',
            opacity: disabled ? 0.5 : 1,
            textAlign: 'left',
          }}
        >
          {chip}
        </button>
      ))}
    </div>
  );
}

function AnswerBody({ text }: { text: string }) {
  const { intro, items, footer } = parseAnswerText(text);
  const paragraph = { margin: 0, whiteSpace: 'pre-wrap' as const, overflowWrap: 'anywhere' as const };
  return (
    <>
      {intro !== '' && <p style={paragraph}>{intro}</p>}
      {items.length > 0 && (
        <ul style={{ margin: '6px 0', paddingLeft: '18px' }}>
          {items.map((item, i) => (
            <li key={i} style={{ overflowWrap: 'anywhere', marginBottom: '2px' }}>
              {item}
            </li>
          ))}
        </ul>
      )}
      {footer.map((line, i) => (
        <p key={i} style={{ ...paragraph, marginTop: i === 0 && items.length === 0 ? 0 : '4px' }}>
          {line}
        </p>
      ))}
    </>
  );
}

function Bubble({ message }: { message: Message }) {
  const mine = message.role === 'user';
  return (
    <div style={{ display: 'flex', justifyContent: mine ? 'flex-end' : 'flex-start' }}>
      <div
        style={{
          maxWidth: '85%',
          padding: '8px 12px',
          borderRadius: '12px',
          fontSize: '14px',
          lineHeight: 1.45,
          background: mine ? 'var(--accent)' : 'var(--surface-2)',
          color: mine ? '#000' : 'var(--text-primary)',
          border: mine ? 'none' : '1px solid var(--border)',
        }}
      >
        {mine ? (
          <p style={{ margin: 0, overflowWrap: 'anywhere' }}>{message.text}</p>
        ) : (
          <AnswerBody text={message.text} />
        )}
      </div>
    </div>
  );
}

export default function AssistantWidget() {
  const pathname = usePathname();
  const isMobile = useIsMobile();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const { loading, messages } = useSyncExternalStore(subscribe, getState, getServerState);
  const viewportRaw = useSyncExternalStore(subscribeViewport, getViewportSnapshot, getServerViewport);

  // AppLayout renders this only once ready (client side), so reading the
  // stored user during render cannot cause a hydration mismatch.
  const user = getUser();
  const userId = user?.id;

  useEffect(() => {
    if (userId !== undefined) bindUser(userId);
  }, [userId]);

  let viewport: { top: number; height: number } | null = null;
  if (open && isMobile && viewportRaw !== '') {
    const [top, height] = viewportRaw.split('|').map(Number);
    viewport = { top, height };
  }

  // Keep the newest message in view.
  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ block: 'end' });
  }, [open, messages, loading, viewport]);

  // Focus the input when the panel opens and when an answer arrives.
  useEffect(() => {
    if (open && !loading) inputRef.current?.focus();
  }, [open, loading]);

  if (!canSeeAssistant(user)) return null;

  const hidden = isHiddenPath(pathname);
  const chips = messages.length === 0 ? GREETING_CHIPS : activeChips(messages);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const text = draft.trim();
    if (text === '' || loading) return;
    setDraft('');
    void send(text);
  };

  const panelStyle: React.CSSProperties = isMobile
    ? {
        position: 'fixed',
        left: 0,
        right: 0,
        top: viewport ? `${viewport.top}px` : 0,
        height: viewport ? `${viewport.height}px` : '100dvh',
        borderRadius: 0,
      }
    : {
        position: 'fixed',
        right: '24px',
        bottom: '24px',
        width: '380px',
        maxWidth: 'calc(100vw - 48px)',
        height: 'min(560px, calc(100dvh - 48px))',
        borderRadius: '12px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.35)',
      };

  return (
    <div style={{ display: hidden ? 'none' : 'block' }}>
      {!open && (
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label="Abrir asistente"
          style={{
            position: 'fixed',
            right: '16px',
            bottom: '16px',
            width: '52px',
            height: '52px',
            borderRadius: '50%',
            border: 'none',
            background: 'var(--accent)',
            color: '#000',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 90,
            boxShadow: '0 4px 16px rgba(0,0,0,0.3)',
          }}
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
        </button>
      )}

      {open && (
        <section
          aria-label="Asistente"
          style={{
            ...panelStyle,
            zIndex: 110,
            display: 'flex',
            flexDirection: 'column',
            background: 'var(--canvas)',
            border: '1px solid var(--border)',
            overflow: 'hidden',
          }}
        >
          <header
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '10px 12px',
              borderBottom: '1px solid var(--border)',
              background: 'var(--surface-1)',
              flexShrink: 0,
            }}
          >
            <span style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>Asistente</span>
            <button
              type="button"
              onClick={() => setOpen(false)}
              aria-label="Cerrar asistente"
              style={{
                width: '36px',
                height: '36px',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                background: 'var(--surface-2)',
                color: 'var(--text-primary)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </header>

          <div
            style={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              overflowX: 'hidden',
              padding: '12px',
              display: 'flex',
              flexDirection: 'column',
              gap: '8px',
            }}
          >
            {messages.length === 0 && (
              <div style={{ color: 'var(--text-primary)', fontSize: '14px', lineHeight: 1.45 }}>
                <p style={{ margin: '0 0 8px' }}>{GREETING_TEXT}</p>
              </div>
            )}
            {messages.map((m, i) => (
              <Bubble key={i} message={m} />
            ))}
            <Chips chips={chips} disabled={loading} />
            {loading && (
              <p role="status" style={{ margin: 0, fontSize: '13px', color: 'var(--text-muted)' }}>
                Pensando…
              </p>
            )}
            <div ref={bottomRef} />
          </div>

          <form
            onSubmit={handleSubmit}
            style={{
              display: 'flex',
              gap: '8px',
              padding: '10px 12px',
              borderTop: '1px solid var(--border)',
              background: 'var(--surface-1)',
              flexShrink: 0,
            }}
          >
            <input
              ref={inputRef}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              disabled={loading}
              placeholder="Escribí tu pregunta"
              aria-label="Pregunta para el asistente"
              style={{
                flex: 1,
                minWidth: 0,
                height: '40px',
                padding: '0 12px',
                borderRadius: '8px',
                border: '1px solid var(--border-emphasis)',
                background: 'var(--surface-2)',
                color: 'var(--text-primary)',
                fontSize: '16px',
              }}
            />
            <button
              type="submit"
              disabled={loading || draft.trim() === ''}
              style={{
                height: '40px',
                padding: '0 14px',
                borderRadius: '8px',
                border: 'none',
                background: 'var(--accent)',
                color: '#000',
                fontSize: '14px',
                fontWeight: 600,
                cursor: loading || draft.trim() === '' ? 'not-allowed' : 'pointer',
                opacity: loading || draft.trim() === '' ? 0.5 : 1,
                flexShrink: 0,
              }}
            >
              Enviar
            </button>
          </form>
        </section>
      )}
    </div>
  );
}
