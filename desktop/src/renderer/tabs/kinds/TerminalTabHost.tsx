/**
 * TerminalTabHost — top-level wrapper for SSH terminal sessions in the
 * workspace tab system. Bridges the tab manager's `Tab` shape (with
 * `state: { sessionId, hostId, hostLabel }`) to the existing
 * `TerminalTab` component which already does the xterm.js + daemon
 * session lifecycle.
 *
 * Title: starts as the host label; updated to whatever OSC 1/2 the
 * remote shell emits via the `onTitleChange` callback.
 *
 * Exit handling: when the daemon emits `session.exit`, we mark the
 * tab non-dirty and let the user close it. Auto-close on clean exit
 * (the previous in-Hosts behaviour) is preserved.
 */
import { useCallback } from 'react';
import TerminalTab, { type TerminalTabData } from '../../components/TerminalTab';
import { useTabActions } from '../../contexts/TabsContext';
import type { TabContentProps } from '../registry';

export default function TerminalTabHost({ tab, isActive }: TabContentProps) {
  const { rename, close } = useTabActions();
  if (tab.state.kind !== 'terminal') return null;
  const { sessionId, hostId, hostLabel } = tab.state;

  const data: TerminalTabData = {
    id: tab.id,
    hostId,
    hostLabel,
    sessionId,
    title: tab.title,
  };

  const onTitleChange = useCallback(
    (_tabId: string, title: string) => rename(tab.id, title),
    [rename, tab.id],
  );
  const onExit = useCallback(
    (_tabId: string, exitCode: number) => {
      if (exitCode === 0) {
        // Mirror the previous behaviour: brief delay so the "exited"
        // line is readable, then close the tab.
        window.setTimeout(() => close(tab.id), 600);
      }
    },
    [close, tab.id],
  );

  return (
    <TerminalTab
      data={data}
      active={isActive}
      onTitleChange={onTitleChange}
      onExit={onExit}
    />
  );
}
