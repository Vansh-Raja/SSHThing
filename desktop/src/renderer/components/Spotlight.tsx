/**
 * Spotlight — TUI-style arming-chord overlay.
 *
 * Press `/` anywhere (outside inputs) to open. Then press a chord key
 * to arm an action:
 *
 *   S  → Search hosts (opens command palette)
 *   N  → New host
 *   C  → Connect to selected host
 *   M  → Mount selected host
 *   E  → Execute on selected host
 *   H  → Health-check selected host
 *   T  → Tokens page
 *   ?  → Help overlay
 *   Esc → Close spotlight
 */
import { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import {
  SearchIcon,
  PlusIcon,
  ConnectIcon,
  MountIcon,
  TerminalIcon,
  CheckIcon,
  TokenIcon,
  GearIcon,
} from './icons';

interface Chord {
  key: string;
  label: string;
  icon: React.ReactNode;
  action: () => void;
}

interface SpotlightProps {
  open: boolean;
  onClose: () => void;
  onOpenPalette: () => void;
  onOpenHelp: () => void;
}

export default function Spotlight({ open, onClose, onOpenPalette, onOpenHelp }: SpotlightProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
        return;
      }
      const k = e.key.toLowerCase();
      switch (k) {
        case 's':
          e.preventDefault();
          onClose();
          onOpenPalette();
          break;
        case 'n':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-new-host'));
          break;
        case 'c':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-connect-selected'));
          break;
        case 'm':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-mount-selected'));
          break;
        case 'e':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-exec-selected'));
          break;
        case 'h':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-health-selected'));
          break;
        case 't':
          e.preventDefault();
          onClose();
          window.dispatchEvent(new CustomEvent('sshthing:cmd-tokens'));
          break;
        case '?':
          e.preventDefault();
          onClose();
          onOpenHelp();
          break;
        default:
          break;
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose, onOpenPalette, onOpenHelp]);

  useEffect(() => {
    if (open && panelRef.current) {
      panelRef.current.focus();
    }
  }, [open]);

  if (!open) return null;

  const chords: Chord[] = [
    { key: 'S', label: 'Search hosts', icon: <SearchIcon width={16} height={16} />, action: () => { onClose(); onOpenPalette(); } },
    { key: 'N', label: 'New host', icon: <PlusIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-new-host')); } },
    { key: 'C', label: 'Connect', icon: <ConnectIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-connect-selected')); } },
    { key: 'M', label: 'Mount', icon: <MountIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-mount-selected')); } },
    { key: 'E', label: 'Execute', icon: <TerminalIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-exec-selected')); } },
    { key: 'H', label: 'Health check', icon: <CheckIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-health-selected')); } },
    { key: 'T', label: 'Tokens', icon: <TokenIcon width={16} height={16} />, action: () => { onClose(); window.dispatchEvent(new CustomEvent('sshthing:cmd-tokens')); } },
    { key: '?', label: 'Help', icon: <GearIcon width={16} height={16} />, action: () => { onClose(); onOpenHelp(); } },
  ];

  return createPortal(
    <div
      className="palette-overlay"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        ref={panelRef}
        className="palette-panel"
        role="dialog"
        aria-label="Spotlight"
        tabIndex={-1}
        style={{ maxWidth: 480, padding: 20 }}
      >
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--muted)', marginBottom: 16, textAlign: 'center' }}>
          Press a key to arm an action
        </div>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, 1fr)',
            gap: 12,
          }}
        >
          {chords.map((c) => (
            <button
              key={c.key}
              type="button"
              className="spotlight-chord"
              onClick={c.action}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 8,
                padding: '16px 8px',
                borderRadius: 8,
                background: 'var(--paper-3)',
                border: '1px solid var(--line)',
                cursor: 'pointer',
                color: 'var(--text)',
                transition: 'background 0.15s, border-color 0.15s',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = 'var(--paper-2)';
                (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--accent)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = 'var(--paper-3)';
                (e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--line)';
              }}
            >
              <kbd
                style={{
                  fontSize: 18,
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--accent)',
                  minWidth: 28,
                  textAlign: 'center',
                }}
              >
                {c.key}
              </kbd>
              <span style={{ fontSize: 11, color: 'var(--muted)' }}>{c.label}</span>
              <span style={{ color: 'var(--muted-2)' }}>{c.icon}</span>
            </button>
          ))}
        </div>
        <div style={{ fontSize: 11, color: 'var(--muted-2)', marginTop: 16, textAlign: 'center' }}>
          Esc to close · Click or press the key
        </div>
      </div>
    </div>,
    document.body,
  );
}
