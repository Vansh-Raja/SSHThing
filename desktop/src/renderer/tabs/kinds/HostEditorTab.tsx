/**
 * HostEditorTab — full-tab create/edit form for a personal host.
 *
 * Replaces the old `HostDrawer` slide-out for both create and edit
 * flows. Built fresh in Phase 3 of the tab migration; this stub will
 * be filled in alongside the bug fixes for:
 *   - Edit mode no longer drops the new credential on save (the old
 *     drawer called `updateHost` without `plainKey`/`plainPassword`).
 *   - Edit mode optionally pre-fills the existing credential (gated
 *     behind a "Show" affordance, audit-logged).
 *   - Dirty tracking + "Discard changes?" prompt on close.
 */
import type { TabContentProps } from '../registry';

export default function HostEditorTab(_props: TabContentProps) {
  return (
    <div style={{ padding: 24, color: 'var(--muted)' }}>
      Host editor — implementation pending Phase 3 of the tab migration.
    </div>
  );
}
