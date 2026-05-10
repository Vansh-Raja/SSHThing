/**
 * HostsTab — the always-open base tab. Wraps the existing Hosts page.
 *
 * After Phase 1 of the tab migration, this will render ONLY the
 * sidebar+detail view (no internal terminal-tab strip) — clicking
 * "connect" on a host dispatches `tabActions.open('terminal', …)` to
 * the top-level manager so the SSH session opens as a sibling tab. For
 * now we re-export the existing Hosts page unchanged so the rest of
 * the tab scaffold compiles before we refactor that page.
 */
import Hosts from '../../pages/Hosts';
import type { TabContentProps } from '../registry';

export default function HostsTab(_props: TabContentProps) {
  return <Hosts />;
}
