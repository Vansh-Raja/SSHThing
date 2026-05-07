/**
 * CommandPalette — VS Code-style Cmd+K overlay.
 *
 * Sections (in order when no query or generic query):
 *   1. Hosts    — fuzzy match on label / hostname / username / tags / group
 *   2. Groups   — fuzzy match on group name
 *   3. Commands — always dominant when query starts with `/` or `:`
 *   4. Recent   — recently connected hosts (shown when query is empty)
 *
 * Slash-command mode: query starts with `/` or `:` → only Commands shown.
 * Empty query: Recent + a Commands hint.
 * Anything else: ranked Hosts / Groups / Commands.
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useNavigate } from 'react-router-dom';
import Fuse from 'fuse.js';
import {
  TerminalIcon,
  FolderIcon,
  GearIcon,
  KeyIcon,
  TeamsIcon,
  SearchIcon,
  ConnectIcon,
  UserIcon,
} from './icons';
import type { SVGProps } from 'react';

// ──────────────────────────────────────────────────────────
// Recent-host helpers (exported for App.tsx)
// ──────────────────────────────────────────────────────────

const RECENT_KEY = 'sshthing-recent-hosts';
const MAX_RECENT = 10;

function getRecentHostIds(): string[] {
  try {
    return JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]') as string[];
  } catch {
    return [];
  }
}

export function recordRecentHost(hostId: string): void {
  const recent = getRecentHostIds().filter((id) => id !== hostId);
  localStorage.setItem(RECENT_KEY, JSON.stringify([hostId, ...recent].slice(0, MAX_RECENT)));
}

// ──────────────────────────────────────────────────────────
// Section + item types
// ──────────────────────────────────────────────────────────

type ItemKind = 'host' | 'group' | 'command' | 'recent';

interface PaletteItem {
  id: string;
  kind: ItemKind;
  label: string;
  subtitle: string;
  icon: React.ReactNode;
  kbd?: string;
  /** payload varies by kind */
  host?: HostSummary;
  group?: GroupSummary;
  action?: () => void | Promise<void>;
}

interface Section {
  title: string;
  items: PaletteItem[];
}

const MAX_PER_SECTION = 6;

// ──────────────────────────────────────────────────────────
// Icon helper (16 px variant used consistently in rows)
// ──────────────────────────────────────────────────────────

function RowIcon({ children }: { children: React.ReactNode }) {
  return <span className="palette-row-icon">{children}</span>;
}

function iconProps(): SVGProps<SVGSVGElement> {
  return { width: 15, height: 15 };
}

// ──────────────────────────────────────────────────────────
// Props
// ──────────────────────────────────────────────────────────

type CommandPaletteProps = {
  open: boolean;
  onClose: () => void;
  hosts: HostSummary[];
  onSelectHost: (host: HostSummary) => void;
  /** Called when the /help command is chosen */
  onHelpOpen?: () => void;
  /** Initial query (e.g. forwarded from topbar search). */
  initialQuery?: string;
};

// ──────────────────────────────────────────────────────────
// Component
// ──────────────────────────────────────────────────────────

export default function CommandPalette({
  open,
  onClose,
  hosts,
  onSelectHost,
  onHelpOpen,
  initialQuery = '',
}: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [groups, setGroups] = useState<GroupSummary[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  // Reset + pre-fill on open.
  useEffect(() => {
    if (open) {
      setQuery(initialQuery);
      setSelectedIdx(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open, initialQuery]);

  // Load groups when open.
  useEffect(() => {
    if (!open) return;
    window.sshthing
      .listGroups()
      .then((res) => setGroups(res.groups))
      .catch(() => setGroups([]));
  }, [open]);

  // ── Command definitions ────────────────────────────────

  const allCommands = useMemo((): PaletteItem[] => {
    const navigate_ = navigate; // capture stable ref

    const syncAction = async () => {
      onClose();
      try {
        await window.sshthing.syncNow();
      } catch {
        // ignore — toast happens elsewhere
      }
    };

    const lockAction = async () => {
      onClose();
      try {
        await window.sshthing.lockVault();
      } catch {
        // ignore
      }
      navigate_('/unlock');
    };

    const signOutAction = async () => {
      onClose();
      try {
        await window.sshthing.authSignOut();
      } catch {
        // ignore
      }
    };

    const probeAllAction = () => {
      onClose();
      const visible = hosts.slice();
      visible.forEach((h) => {
        window.sshthing.healthProbe(h.id).catch(() => {
          // ignore
        });
      });
      // No toast here to avoid import coupling; App can handle via notifications.
    };

    return [
      {
        id: 'cmd-sync',
        kind: 'command',
        label: '/sync',
        subtitle: 'Sync now with cloud',
        icon: <RowIcon><SearchIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: syncAction,
      },
      {
        id: 'cmd-lock',
        kind: 'command',
        label: '/lock',
        subtitle: 'Lock vault and sign out',
        icon: <RowIcon><GearIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: lockAction,
      },
      {
        id: 'cmd-sign-in',
        kind: 'command',
        label: '/sign-in',
        subtitle: 'Go to sign-in page',
        icon: <RowIcon><UserIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: () => { onClose(); navigate_('/sign-in'); },
      },
      {
        id: 'cmd-sign-out',
        kind: 'command',
        label: '/sign-out',
        subtitle: 'Sign out of SSHThing Cloud',
        icon: <RowIcon><UserIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: signOutAction,
      },
      {
        id: 'cmd-settings',
        kind: 'command',
        label: '/settings',
        subtitle: 'Open settings',
        icon: <RowIcon><GearIcon {...iconProps()} /></RowIcon>,
        kbd: '⌘,',
        action: () => { onClose(); navigate_('/settings'); },
      },
      {
        id: 'cmd-teams',
        kind: 'command',
        label: '/teams',
        subtitle: 'Open teams',
        icon: <RowIcon><TeamsIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: () => { onClose(); navigate_('/teams'); },
      },
      {
        id: 'cmd-account',
        kind: 'command',
        label: '/account',
        subtitle: 'Open account',
        icon: <RowIcon><UserIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: () => { onClose(); navigate_('/account'); },
      },
      {
        id: 'cmd-keys',
        kind: 'command',
        label: '/keys',
        subtitle: 'Manage SSH keys',
        icon: <RowIcon><KeyIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: () => { onClose(); navigate_('/keys'); },
      },
      {
        id: 'cmd-help',
        kind: 'command',
        label: '/help',
        subtitle: 'Show keyboard shortcuts',
        icon: <RowIcon><GearIcon {...iconProps()} /></RowIcon>,
        kbd: '?',
        action: () => { onClose(); onHelpOpen?.(); },
      },
      {
        id: 'cmd-probe-all',
        kind: 'command',
        label: '/probe-all',
        subtitle: `Probe all ${hosts.length} visible hosts`,
        icon: <RowIcon><TerminalIcon {...iconProps()} /></RowIcon>,
        kbd: '↵',
        action: probeAllAction,
      },
    ];
  }, [navigate, onClose, hosts, onHelpOpen]);

  // ── Fuse instances ────────────────────────────────────

  const hostFuse = useMemo(
    () =>
      new Fuse(hosts, {
        keys: [
          { name: 'label', weight: 0.4 },
          { name: 'hostname', weight: 0.4 },
          { name: 'username', weight: 0.1 },
          { name: 'tags', weight: 0.05 },
          { name: 'group', weight: 0.05 },
        ],
        threshold: 0.4,
        includeScore: true,
      }),
    [hosts],
  );

  const groupFuse = useMemo(
    () =>
      new Fuse(groups, {
        keys: [{ name: 'name', weight: 1 }],
        threshold: 0.4,
        includeScore: true,
      }),
    [groups],
  );

  const commandFuse = useMemo(
    () =>
      new Fuse(allCommands, {
        keys: [
          { name: 'label', weight: 0.6 },
          { name: 'subtitle', weight: 0.4 },
        ],
        threshold: 0.4,
        includeScore: true,
      }),
    [allCommands],
  );

  // ── Build sections ────────────────────────────────────

  const sections = useMemo((): Section[] => {
    const q = query.trim();
    const isSlash = q.startsWith('/') || q.startsWith(':');

    if (isSlash) {
      // Detect `/connect <rest>` or `:connect <rest>` — host fuzzy-match mode
      const withoutPrefix = q.slice(1); // strip leading `/` or `:`
      const connectMatch = withoutPrefix.match(/^connect(?:\s+(.*))?$/i);
      if (connectMatch !== null) {
        const hostQuery = (connectMatch[1] ?? '').trim();
        const matchedHosts: PaletteItem[] = (
          hostQuery ? hostFuse.search(hostQuery).map((r) => r.item) : hosts
        )
          .slice(0, MAX_PER_SECTION)
          .map((h) => ({
            id: `connect-${h.id}`,
            kind: 'host' as const,
            label: h.label.trim() || h.hostname,
            subtitle: `${h.username}@${h.hostname}:${h.port}${h.group ? ` · ${h.group}` : ''}`,
            icon: <RowIcon><ConnectIcon {...iconProps()} /></RowIcon>,
            kbd: '↵',
            host: h,
          }));
        return matchedHosts.length > 0
          ? [{ title: 'Connect to host', items: matchedHosts }]
          : [];
      }

      // General slash-command mode: strip the prefix for matching
      const commandQuery = withoutPrefix;
      const matched = commandQuery
        ? commandFuse.search(commandQuery).map((r) => r.item)
        : allCommands;
      return matched.length > 0
        ? [{ title: 'Commands', items: matched.slice(0, MAX_PER_SECTION) }]
        : [];
    }

    if (!q) {
      // Empty query: show Recent + a Commands hint
      const recentIds = getRecentHostIds();
      const recentItems: PaletteItem[] = recentIds
        .map((id) => hosts.find((h) => h.id === id))
        .filter((h): h is HostSummary => !!h)
        .slice(0, MAX_PER_SECTION)
        .map((h) => ({
          id: `recent-${h.id}`,
          kind: 'recent' as const,
          label: h.label.trim() || h.hostname,
          subtitle: `${h.username}@${h.hostname}:${h.port}`,
          icon: <RowIcon><ConnectIcon {...iconProps()} /></RowIcon>,
          kbd: '↵',
          host: h,
        }));

      const commandHints = allCommands.slice(0, 5).map((c) => c);

      const result: Section[] = [];
      if (recentItems.length > 0) result.push({ title: 'Recent', items: recentItems });
      result.push({ title: 'Commands', items: commandHints });
      return result;
    }

    // Generic query — fuzzy rank all sections
    const matchedHosts: PaletteItem[] = hostFuse.search(q).slice(0, MAX_PER_SECTION).map((r) => ({
      id: `host-${r.item.id}`,
      kind: 'host' as const,
      label: r.item.label.trim() || r.item.hostname,
      subtitle: `${r.item.username}@${r.item.hostname}:${r.item.port}${r.item.group ? ` · ${r.item.group}` : ''}`,
      icon: <RowIcon><TerminalIcon {...iconProps()} /></RowIcon>,
      kbd: '↵',
      host: r.item,
    }));

    const matchedGroups: PaletteItem[] = groupFuse.search(q).slice(0, MAX_PER_SECTION).map((r) => ({
      id: `group-${r.item.id}`,
      kind: 'group' as const,
      label: r.item.name,
      subtitle: 'Group',
      icon: <RowIcon><FolderIcon {...iconProps()} /></RowIcon>,
      kbd: '↵',
      group: r.item,
    }));

    const matchedCommands: PaletteItem[] = commandFuse.search(q).slice(0, MAX_PER_SECTION).map((r) => r.item);

    const result: Section[] = [];
    if (matchedHosts.length > 0) result.push({ title: 'Hosts', items: matchedHosts });
    if (matchedGroups.length > 0) result.push({ title: 'Groups', items: matchedGroups });
    if (matchedCommands.length > 0) result.push({ title: 'Commands', items: matchedCommands });
    return result;
  }, [query, hosts, allCommands, hostFuse, groupFuse, commandFuse]);

  // Flat list of all visible items for keyboard navigation
  const flatItems = useMemo(
    () => sections.flatMap((s) => s.items),
    [sections],
  );

  const clampedIdx = Math.min(selectedIdx, Math.max(0, flatItems.length - 1));

  // ── Item activation ───────────────────────────────────

  const activate = (item: PaletteItem) => {
    if (item.kind === 'host' && item.host) {
      onClose();
      onSelectHost(item.host);
      return;
    }
    if (item.kind === 'recent' && item.host) {
      onClose();
      onSelectHost(item.host);
      return;
    }
    if (item.kind === 'group') {
      // Scroll to group in sidebar via custom event; hosts page listens.
      onClose();
      window.dispatchEvent(
        new CustomEvent('sshthing:scroll-to-group', { detail: { name: item.group?.name } }),
      );
      return;
    }
    if (item.kind === 'command' && item.action) {
      void item.action();
      return;
    }
  };

  // ── Keyboard handling ─────────────────────────────────

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIdx((i) => Math.min(i + 1, flatItems.length - 1));
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIdx((i) => Math.max(i - 1, 0));
        return;
      }
      if (e.key === 'Enter') {
        const item = flatItems[clampedIdx];
        if (item) activate(item);
        return;
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, flatItems, clampedIdx, onClose]);

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const items = listRef.current.querySelectorAll<HTMLElement>('[data-palette-item]');
    items[clampedIdx]?.scrollIntoView({ block: 'nearest' });
  }, [clampedIdx]);

  if (!open) return null;

  // ── Render ────────────────────────────────────────────

  // Build a flat counter so keyboard selection works across sections
  let itemCounter = 0;

  return createPortal(
    <div
      className="palette-overlay"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="palette-panel" role="dialog" aria-label="Command palette">
        <div className="palette-input-row">
          <span className="palette-input-icon"><SearchIcon width={14} height={14} /></span>
          <input
            ref={inputRef}
            className="palette-input"
            type="text"
            placeholder="Search hosts, groups, or type / for commands…"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedIdx(0);
            }}
            autoComplete="off"
            spellCheck={false}
          />
          <kbd className="palette-input-kbd">Esc</kbd>
        </div>
        <div className="palette-list" ref={listRef}>
          {flatItems.length === 0 ? (
            <div className="palette-empty">
              {query ? `No results for "${query}"` : 'No hosts found'}
            </div>
          ) : (
            sections.map((section) => (
              <div key={section.title} className="palette-section">
                <div className="palette-section-header">{section.title}</div>
                {section.items.map((item) => {
                  const idx = itemCounter++;
                  const isSelected = idx === clampedIdx;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      data-palette-item="true"
                      className={`palette-item${isSelected ? ' palette-item--selected' : ''}`}
                      onMouseEnter={() => setSelectedIdx(idx)}
                      onClick={() => activate(item)}
                    >
                      {item.icon}
                      <div className="palette-item__content">
                        <span className="palette-item__label">{item.label}</span>
                        <span className="palette-item__meta">{item.subtitle}</span>
                      </div>
                      {item.kbd && <kbd className="palette-item__kbd">{item.kbd}</kbd>}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}
