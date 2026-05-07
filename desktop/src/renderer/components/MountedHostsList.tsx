/**
 * MountedHostsList — displays currently active SSHFS mounts.
 * Provides "Show in Finder" and "Unmount" actions.
 */
import { toast } from '../ui/toast';
import Button from '../ui/Button';
import Spinner from '../ui/Spinner';
import EmptyState from './EmptyState';

type MountedHostsListProps = {
  mounts: MountSummary[];
  unmounting: Set<string>;
  onUnmount: (hostId: string) => Promise<void>;
};

export default function MountedHostsList({ mounts, unmounting, onUnmount }: MountedHostsListProps) {
  if (mounts.length === 0) {
    return (
      <EmptyState
        title="No active mounts"
        description='Use the "Mount" action on a host to mount its filesystem here.'
      />
    );
  }

  const handleOpenPath = async (localPath: string) => {
    try {
      const errMsg = await window.sshthing.openPath(localPath);
      if (errMsg) {
        toast.error(`Could not open: ${errMsg}`);
      }
    } catch {
      toast.error('Failed to open path');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {mounts.map((m) => {
        const isUnmounting = unmounting.has(m.hostId);
        return (
          <div
            key={m.hostId}
            style={{
              border: '1.5px solid var(--line)',
              borderRadius: 4,
              padding: '8px 10px',
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: 8,
              background: 'var(--paper-2)',
            }}
          >
            <div style={{ minWidth: 0, flex: 1 }}>
              <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--ink)', marginBottom: 2 }}>
                {m.hostname}
              </div>
              <div
                style={{
                  fontSize: 11,
                  color: 'var(--muted)',
                  fontFamily: 'var(--font-mono)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
                title={`${m.localPath} → ${m.remotePath}`}
              >
                {m.localPath} → {m.remotePath}
              </div>
            </div>
            <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
              <Button
                variant="ghost"
                onClick={() => void handleOpenPath(m.localPath)}
                disabled={isUnmounting}
                style={{ fontSize: 11, minHeight: 26, padding: '0 8px' }}
              >
                Show in Finder
              </Button>
              <Button
                variant="ghost"
                onClick={() => void onUnmount(m.hostId)}
                disabled={isUnmounting}
                style={{ fontSize: 11, minHeight: 26, padding: '0 8px', color: 'var(--danger)' }}
              >
                {isUnmounting ? <Spinner size={12} /> : null}
                {isUnmounting ? 'Unmounting…' : 'Unmount'}
              </Button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
