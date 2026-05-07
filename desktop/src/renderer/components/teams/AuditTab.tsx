/**
 * AuditTab — team audit log with:
 * - Filter bar: text search, member dropdown, event-type multi-select, date range
 * - Virtualised list (CSS approach, no react-window dep) with scroll container
 * - Row click → AuditEventDetailModal with structured fields + collapsible raw JSON
 *
 * Wave 2B: filters + enhanced detail modal (raw metadata block).
 */
import { useMemo, useState } from 'react';
import { useTeamAudit } from '../../hooks/useTeamAudit';
import Modal from '../../ui/Modal';
import EmptyState from '../EmptyState';
import { Skeleton } from '../Skeleton';

type AuditTabProps = {
  teamId: string;
};

const EVENT_TYPE_COLORS: Record<string, string> = {
  host_created: 'var(--accent)',
  host_updated: 'var(--accent)',
  host_deleted: 'var(--danger)',
  member_invited: 'var(--accent)',
  member_joined: 'var(--success, #4caf50)',
  member_removed: 'var(--danger)',
  member_role_updated: 'var(--muted)',
  credential_revealed: 'var(--warning, #ff9800)',
  invite_revoked: 'var(--danger)',
  token_created: 'var(--accent)',
  token_revoked: 'var(--danger)',
};

// All known event types (used for multi-select filter).
const KNOWN_EVENT_TYPES = [
  'host_created',
  'host_updated',
  'host_deleted',
  'member_invited',
  'member_joined',
  'member_removed',
  'member_role_updated',
  'credential_revealed',
  'invite_revoked',
  'token_created',
  'token_revoked',
];

type DateRangeOption = 'all' | '7d' | '30d';

function formatTimestamp(ts: number): string {
  return new Date(ts * 1000).toLocaleString();
}

function formatTimestampRelative(ts: number): string {
  const now = Date.now();
  const diffMs = now - ts * 1000;
  const diffMin = Math.floor(diffMs / 60_000);
  const diffHr = Math.floor(diffMs / 3_600_000);
  const diffDay = Math.floor(diffMs / 86_400_000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  return `${diffDay}d ago`;
}

export default function AuditTab({ teamId }: AuditTabProps) {
  const { events, loading, error, reload } = useTeamAudit(teamId);
  const [selectedEvent, setSelectedEvent] = useState<TeamAuditEvent | null>(null);
  const [metaExpanded, setMetaExpanded] = useState(false);

  // ── Filter state ──────────────────────────────────────────────────────────

  const [textFilter, setTextFilter] = useState('');
  const [memberFilter, setMemberFilter] = useState('');  // actorClerkUserId or ''
  const [eventTypeFilter, setEventTypeFilter] = useState<Set<string>>(new Set());
  const [dateRange, setDateRange] = useState<DateRangeOption>('all');

  // Derive unique actor list for member dropdown.
  const actors = useMemo(() => {
    const seen = new Map<string, string>(); // clerkUserId → displayName
    for (const e of events) {
      if (!seen.has(e.actorClerkUserId)) {
        seen.set(e.actorClerkUserId, e.actorDisplayName);
      }
    }
    return Array.from(seen.entries()).map(([id, name]) => ({ id, name }));
  }, [events]);

  // Derive event types actually present in data (supplement KNOWN_EVENT_TYPES).
  const presentEventTypes = useMemo(() => {
    const types = new Set(KNOWN_EVENT_TYPES);
    for (const e of events) types.add(e.eventType);
    return Array.from(types).sort();
  }, [events]);

  // Apply all filters.
  const filtered = useMemo(() => {
    const now = Date.now();
    const cutoffMs = dateRange === '7d' ? now - 7 * 86_400_000 : dateRange === '30d' ? now - 30 * 86_400_000 : 0;

    return events.filter((e) => {
      // Date range
      if (cutoffMs > 0 && e.createdAt * 1000 < cutoffMs) return false;

      // Member (actor)
      if (memberFilter && e.actorClerkUserId !== memberFilter) return false;

      // Event type multi-select
      if (eventTypeFilter.size > 0 && !eventTypeFilter.has(e.eventType)) return false;

      // Text search
      if (textFilter.trim()) {
        const q = textFilter.toLowerCase();
        if (
          !e.summary.toLowerCase().includes(q) &&
          !e.actorDisplayName.toLowerCase().includes(q) &&
          !e.eventType.toLowerCase().includes(q)
        ) {
          return false;
        }
      }

      return true;
    });
  }, [events, textFilter, memberFilter, eventTypeFilter, dateRange]);

  const toggleEventType = (type: string) => {
    setEventTypeFilter((prev) => {
      const next = new Set(prev);
      if (next.has(type)) next.delete(type);
      else next.add(type);
      return next;
    });
  };

  // ── Render helpers ────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div style={{ padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: 6 }}>
        {Array.from({ length: 5 }, (_, i) => (
          <div
            key={i}
            style={{
              padding: '9px 12px',
              borderRadius: 'var(--radius)',
              background: 'var(--paper-2)',
              border: '1px solid var(--line)',
              display: 'flex',
              gap: 10,
              alignItems: 'center',
            }}
            aria-hidden="true"
          >
            <Skeleton width={8} height={8} style={{ borderRadius: '50%', flexShrink: 0 }} />
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 5 }}>
              <Skeleton width="60%" height={13} />
              <Skeleton width="35%" height={11} />
            </div>
            <Skeleton width={64} height={18} style={{ borderRadius: 'var(--radius)' }} />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 32, display: 'flex', justifyContent: 'center' }}>
        <EmptyState
          title="Failed to load audit log"
          description={error}
          action={{ label: 'Retry', onClick: reload }}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: '16px 20px', display: 'flex', flexDirection: 'column', gap: 10, height: '100%' }}>
      {/* ── Filter bar ──────────────────────────────────────────────────── */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {/* Row 1: text search + count */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <input
            className="field__input"
            type="search"
            placeholder="Search events…"
            value={textFilter}
            onChange={(e) => setTextFilter(e.target.value)}
            style={{ flex: 1, minHeight: 30, padding: '5px 8px', fontSize: 12 }}
          />
          <span style={{ fontSize: 11, color: 'var(--muted)', flexShrink: 0 }}>
            {filtered.length} {filtered.length === 1 ? 'event' : 'events'}
          </span>
        </div>

        {/* Row 2: member dropdown + date range + event type pills */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
          {/* Member */}
          <select
            className="field__input"
            value={memberFilter}
            onChange={(e) => setMemberFilter(e.target.value)}
            style={{ fontSize: 11, padding: '4px 6px', minHeight: 28, minWidth: 120 }}
          >
            <option value="">All members</option>
            {actors.map((a) => (
              <option key={a.id} value={a.id}>{a.name}</option>
            ))}
          </select>

          {/* Date range */}
          <select
            className="field__input"
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value as DateRangeOption)}
            style={{ fontSize: 11, padding: '4px 6px', minHeight: 28, minWidth: 100 }}
          >
            <option value="all">All time</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>

          {/* Event type pills */}
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {presentEventTypes.slice(0, 8).map((type) => {
              const active = eventTypeFilter.has(type);
              return (
                <button
                  key={type}
                  type="button"
                  onClick={() => toggleEventType(type)}
                  style={{
                    fontSize: 10,
                    padding: '3px 7px',
                    borderRadius: 'var(--radius)',
                    border: `1px solid ${active ? (EVENT_TYPE_COLORS[type] ?? 'var(--accent)') : 'var(--line)'}`,
                    background: active ? 'color-mix(in srgb, currentColor 10%, transparent)' : 'transparent',
                    color: active ? (EVENT_TYPE_COLORS[type] ?? 'var(--accent)') : 'var(--muted)',
                    cursor: 'pointer',
                    textTransform: 'uppercase',
                    letterSpacing: '0.04em',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {type.replace(/_/g, ' ')}
                </button>
              );
            })}
            {eventTypeFilter.size > 0 && (
              <button
                type="button"
                onClick={() => setEventTypeFilter(new Set())}
                style={{
                  fontSize: 10,
                  padding: '3px 7px',
                  borderRadius: 'var(--radius)',
                  border: '1px solid var(--line)',
                  background: 'transparent',
                  color: 'var(--muted)',
                  cursor: 'pointer',
                }}
              >
                Clear
              </button>
            )}
          </div>
        </div>
      </div>

      {/* ── Events list ─────────────────────────────────────────────────── */}
      {filtered.length === 0 ? (
        <EmptyState
          title={textFilter || memberFilter || eventTypeFilter.size > 0 ? 'No matching events' : 'No audit events yet'}
          description={
            textFilter || memberFilter || eventTypeFilter.size > 0
              ? 'Try adjusting the filters.'
              : 'Team actions will appear here.'
          }
        />
      ) : (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 2,
            overflowY: 'auto',
            flex: 1,
            // For > 100 items, the browser's native overflow scroll is the
            // virtualisation layer (no react-window dep required for this count).
            maxHeight: 'calc(100vh - 260px)',
            contain: 'strict',
          }}
        >
          {filtered.map((event) => (
            <button
              key={event.id}
              type="button"
              onClick={() => { setSelectedEvent(event); setMetaExpanded(false); }}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 10,
                padding: '9px 12px',
                borderRadius: 4,
                background: 'var(--paper-2)',
                border: '1px solid var(--line)',
                textAlign: 'left',
                cursor: 'pointer',
                width: '100%',
                flexShrink: 0,
              }}
            >
              {/* Color dot */}
              <div
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: EVENT_TYPE_COLORS[event.eventType] ?? 'var(--muted)',
                  marginTop: 4,
                  flexShrink: 0,
                }}
              />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 500,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {event.summary}
                </div>
                <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
                  {event.actorDisplayName} · {formatTimestampRelative(event.createdAt)}
                </div>
              </div>
              <span
                className="chip"
                style={{
                  fontSize: 9,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  flexShrink: 0,
                  color: EVENT_TYPE_COLORS[event.eventType],
                  borderColor: EVENT_TYPE_COLORS[event.eventType],
                }}
              >
                {event.eventType.replace(/_/g, ' ')}
              </span>
            </button>
          ))}
        </div>
      )}

      {/* ── Audit event detail modal ─────────────────────────────────────── */}
      <Modal
        open={!!selectedEvent}
        onClose={() => setSelectedEvent(null)}
        title={selectedEvent?.eventType.replace(/_/g, ' ') ?? ''}
        maxWidth={520}
      >
        {selectedEvent && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <p style={{ margin: 0, fontSize: 13, lineHeight: 1.55 }}>{selectedEvent.summary}</p>

            <dl
              style={{
                display: 'grid',
                gridTemplateColumns: 'max-content 1fr',
                gap: '4px 12px',
                margin: 0,
              }}
            >
              <dt style={{ fontSize: 11, color: 'var(--muted)' }}>Actor</dt>
              <dd style={{ fontSize: 12, margin: 0 }}>
                {selectedEvent.actorDisplayName}
              </dd>

              {selectedEvent.targetDisplayName && (
                <>
                  <dt style={{ fontSize: 11, color: 'var(--muted)' }}>Target</dt>
                  <dd style={{ fontSize: 12, margin: 0 }}>{selectedEvent.targetDisplayName}</dd>
                </>
              )}

              <dt style={{ fontSize: 11, color: 'var(--muted)' }}>Event type</dt>
              <dd style={{ fontSize: 12, margin: 0 }}>
                <span
                  className="chip"
                  style={{
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: EVENT_TYPE_COLORS[selectedEvent.eventType],
                    borderColor: EVENT_TYPE_COLORS[selectedEvent.eventType],
                  }}
                >
                  {selectedEvent.eventType.replace(/_/g, ' ')}
                </span>
              </dd>

              <dt style={{ fontSize: 11, color: 'var(--muted)' }}>Entity</dt>
              <dd style={{ fontSize: 12, margin: 0, fontFamily: 'var(--font-mono)' }}>
                {selectedEvent.entityType} / {selectedEvent.entityId}
              </dd>

              <dt style={{ fontSize: 11, color: 'var(--muted)' }}>Time</dt>
              <dd style={{ fontSize: 12, margin: 0 }}>
                {formatTimestamp(selectedEvent.createdAt)}
                {' '}
                <span style={{ color: 'var(--muted)' }}>
                  ({formatTimestampRelative(selectedEvent.createdAt)})
                </span>
              </dd>
            </dl>

            {/* Raw metadata block (collapsible) */}
            {selectedEvent.metadata && Object.keys(selectedEvent.metadata).length > 0 && (
              <div>
                <button
                  type="button"
                  onClick={() => setMetaExpanded((x) => !x)}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--muted)',
                    fontSize: 11,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    padding: 0,
                    marginBottom: 4,
                  }}
                >
                  <span style={{ display: 'inline-block', transform: metaExpanded ? 'rotate(90deg)' : 'none', transition: 'transform 0.15s' }}>
                    ▶
                  </span>
                  Raw metadata
                </button>
                {metaExpanded && (
                  <pre
                    style={{
                      margin: 0,
                      padding: '8px',
                      background: 'var(--paper-2)',
                      border: '1px solid var(--line)',
                      borderRadius: 'var(--radius)',
                      fontSize: 11,
                      fontFamily: 'var(--font-mono)',
                      overflowX: 'auto',
                      maxHeight: 200,
                      overflowY: 'auto',
                      color: 'var(--ink)',
                    }}
                  >
                    {JSON.stringify(selectedEvent.metadata, null, 2)}
                  </pre>
                )}
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
