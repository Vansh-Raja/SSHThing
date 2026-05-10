/**
 * ExecTab — long-running remote command runner. Replaces ExecModal so
 * batch output stays visible while the user switches to other tabs.
 * Phase 3 stub; will be filled in alongside the host-editor work.
 */
import type { TabContentProps } from '../registry';

export default function ExecTab(_props: TabContentProps) {
  return (
    <div style={{ padding: 24, color: 'var(--muted)' }}>
      Exec runner — implementation pending Phase 3 of the tab migration.
    </div>
  );
}
