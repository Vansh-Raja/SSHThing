/**
 * DownloadModal — prompts for a remote source path and local destination
 * directory before starting an SFTP download.
 */
import { useEffect, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import TextField from '../ui/TextField';
import Spinner from '../ui/Spinner';

export interface DownloadOptions {
  recursive: boolean;
  preserve: boolean;
}

type DownloadModalProps = {
  open: boolean;
  host: HostSummary | null;
  onClose: () => void;
  /** Called with (hostId, remotePath, localPath, options) once confirmed. */
  onConfirm: (hostId: string, remotePath: string, localPath: string, options: DownloadOptions) => void;
};

export default function DownloadModal({ open, host, onClose, onConfirm }: DownloadModalProps) {
  const [remotePath, setRemotePath] = useState('');
  const [localPath, setLocalPath] = useState('');
  const [recursive, setRecursive] = useState(false);
  const [preserve, setPreserve] = useState(false);
  const [loading, setLoading] = useState(false);
  const [choosing, setChoosing] = useState(false);

  // Reset on open.
  useEffect(() => {
    if (!open) {
      setRemotePath('');
      setLocalPath('');
      setRecursive(false);
      setPreserve(false);
    }
  }, [open]);

  const handleChooseDirectory = async () => {
    setChoosing(true);
    try {
      const result = await window.sshthing.chooseDirectory();
      if (!result.canceled && result.path) {
        setLocalPath(result.path);
      }
    } catch {
      // Ignore — user may have dismissed or IPC error
    } finally {
      setChoosing(false);
    }
  };

  const handleConfirm = () => {
    if (!host || !remotePath.trim() || !localPath.trim()) return;
    setLoading(true);
    onConfirm(host.id, remotePath.trim(), localPath.trim(), { recursive, preserve });
    setLoading(false);
    onClose();
  };

  const displayName = host ? (host.label.trim() || host.hostname) : '';
  const canConfirm = !loading && !!remotePath.trim() && !!localPath.trim();

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Download from ${displayName}`}
      maxWidth={420}
      footer={
        <div className="modal__actions">
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleConfirm}
            disabled={!canConfirm}
          >
            {loading ? <Spinner size={14} /> : null}
            Download
          </Button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <TextField
          label="Remote path"
          value={remotePath}
          onChange={(e) => setRemotePath(e.target.value)}
          placeholder="~/remote/file.txt"
          disabled={loading}
        />

        {/* Local directory picker */}
        <div>
          <span style={{ display: 'block', fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700, marginBottom: 4 }}>
            Local destination
          </span>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <div style={{ flex: 1 }}>
              <TextField
                label=""
                value={localPath}
                onChange={(e) => setLocalPath(e.target.value)}
                placeholder="/Users/you/Downloads"
                disabled={loading}
              />
            </div>
            <Button
              variant="ghost"
              onClick={() => { void handleChooseDirectory(); }}
              disabled={loading || choosing}
              style={{ flexShrink: 0, whiteSpace: 'nowrap' }}
            >
              {choosing ? <Spinner size={12} /> : 'Choose folder…'}
            </Button>
          </div>
        </div>

        {/* Transfer options */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
            Options
          </span>
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={recursive}
              onChange={(e) => setRecursive(e.target.checked)}
              disabled={loading}
            />
            Recursive (copy directories)
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={preserve}
              onChange={(e) => setPreserve(e.target.checked)}
              disabled={loading}
            />
            Preserve timestamps &amp; permissions
          </label>
        </div>
      </div>
    </Modal>
  );
}
