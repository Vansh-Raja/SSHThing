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

function getPlatformHint(platform?: string): string {
  const p = (platform ?? '').toLowerCase();
  const nav = typeof navigator !== 'undefined' ? navigator.platform.toLowerCase() : '';
  const isMac = p.includes('darwin') || p.includes('mac') || nav.includes('mac');
  const isLinux = p.includes('linux') || nav.includes('linux');
  const isWin = p.includes('win') || nav.includes('win');

  if (isMac) {
    return 'Install FUSE-T and SSHFS: brew install --cask fuse-t && brew install --cask fuse-t-sshfs';
  }
  if (isLinux) {
    return 'Install SSHFS: sudo apt install sshfs (Debian/Ubuntu) or sudo dnf install fuse-sshfs (Fedora)';
  }
  if (isWin) {
    return 'Mount is not supported on Windows yet.';
  }
  return '';
}

export default function MountDrawer({ open, host, onClose, onMounted }: MountDrawerProps) {
  const [remotePath, setRemotePath] = useState('/');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [prereqError, setPrereqError] = useState('');
  const [checkingPrereqs, setCheckingPrereqs] = useState(false);

  // Reset form when opened.
  useEffect(() => {
    if (open) {
      setRemotePath('/');
      setError('');
      setPrereqError('');
      setCheckingPrereqs(true);
      window.sshthing
        .mountCheckPrereqs()
        .then((res) => {
          if (!res.ok || res.missing.length > 0) {
            const missingText = res.missing.length > 0 ? `Missing: ${res.missing.join(', ')}. ` : '';
            const hint = getPlatformHint(res.platform);
            setPrereqError(`${missingText}${hint}`);
          }
        })
        .catch((err: unknown) => {
          const e = err as Error;
          const hint = getPlatformHint();
          setPrereqError(`${e.message ?? 'Could not verify mount prerequisites.'} ${hint}`.trim());
        })
        .finally(() => setCheckingPrereqs(false));
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
  const canMount = !loading && !checkingPrereqs && !prereqError && !!host;

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
          <Button variant="primary" onClick={() => void handleMount()} disabled={!canMount}>
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

        {checkingPrereqs && (
          <p style={{ fontSize: 12, color: 'var(--muted)', margin: 0 }}>
            Checking prerequisites…
          </p>
        )}

        {prereqError && (
          <p style={{ fontSize: 12, color: 'var(--danger)', margin: 0, whiteSpace: 'pre-wrap' }}>
            {prereqError}
          </p>
        )}

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
