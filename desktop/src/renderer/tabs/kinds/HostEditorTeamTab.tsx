/**
 * HostEditorTeamTab — full-tab create/edit form for a team host.
 *
 * Replaces `TeamHostDrawer` for both flows. Phase 3 stub.
 *
 * Phase 3 will fix the existing-shared-credential reveal bug: the old
 * drawer always blanked `sharedCredential` on edit, so admins had no
 * way to see what was stored before changing it.
 */
import type { TabContentProps } from '../registry';

export default function HostEditorTeamTab(_props: TabContentProps) {
  return (
    <div style={{ padding: 24, color: 'var(--muted)' }}>
      Team host editor — implementation pending Phase 3 of the tab migration.
    </div>
  );
}
