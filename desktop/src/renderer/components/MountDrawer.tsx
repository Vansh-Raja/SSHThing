/**
 * MountDrawer — lets the user start an SSHFS mount for a host.
 * The local path is derived from daemon config, so only the remote path is editable.
 */
import { useEffect, useState } from 'react';
import Drawer from '../ui/Drawer';
import Button from '../ui/Button';
import TextField from '../ui/TextField';
import Spinner from '../ui/Spinner';

type MountDrawerProps = {
  open: boolean;
  host: HostSummary | null;
  onClose: () => void;
  onMounted: (summary: MountSummary) => void;
};

export default function MountDrawer({ open, host, onClose, onMounted }: MountDrawerProps) {
  const [remotePath, setRemotePath] = useState('/');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Reset form when opened.
  useEffect(() => {
    if (open) {
      setRemotePath('/');
      setError('');
    }
  }, [open]);

  const handleMount = async () => {
    if (!host) return;
    setLoading(true);
    setError('');
    try {
      const summary = await window.sshthing.mountStart(host.id, remotePath);
      onMounted(summary);
      onClose();
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message ?? 'Mount failed');
    } finally {
      setLoading(false);
    }
  };

  const displayName = host ? (host.label.trim() || host.hostname) : '';

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={`Mount — ${displayName}`}
      width={400}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button variant="primary" onClick={() => void handleMount()} disabled={loading || !host}>
            {loading ? <Spinner size={14} /> : null}
            {loading ? 'Mounting…' : 'Mount'}
          </Button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <p style={{ fontSize: 12, color: 'var(--muted)', margin: 0 }}>
          Mounts the remote filesystem via SSHFS. The local mount point is derived from
          your daemon configuration.
        </p>

        <TextField
          label="Remote path"
          value={remotePath}
          onChange={(e) => setRemotePath(e.target.value)}
          placeholder="/"
          disabled={loading}
        />

        {error && (
          <p style={{ fontSize: 12, color: 'var(--danger)', margin: 0 }}>{error}</p>
        )}

        {loading && (
          <p style={{ fontSize: 12, color: 'var(--muted)', margin: 0 }}>
            Starting SSHFS — this may take a few seconds…
          </p>
        )}
      </div>
    </Drawer>
  );
}
