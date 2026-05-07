/**
 * UploadModal — prompts for the remote destination path after a drag-drop.
 * Triggered when the user drops a local file onto a host row.
 */
import { useEffect, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import TextField from '../ui/TextField';
import Spinner from '../ui/Spinner';

export interface UploadOptions {
  recursive: boolean;
  preserve: boolean;
}

type UploadModalProps = {
  open: boolean;
  host: HostSummary | null;
  localPath: string;
  onClose: () => void;
  /** Called with (localPath, remotePath, options) once confirmed. */
  onConfirm: (localPath: string, remotePath: string, options: UploadOptions) => void;
};

export default function UploadModal({ open, host, localPath, onClose, onConfirm }: UploadModalProps) {
  const [remotePath, setRemotePath] = useState('');
  const [recursive, setRecursive] = useState(false);
  const [preserve, setPreserve] = useState(false);
  const [loading, setLoading] = useState(false);

  // Seed the remote path with the filename from localPath on open.
  useEffect(() => {
    if (open && localPath) {
      const filename = localPath.replace(/\\/g, '/').split('/').pop() ?? '';
      setRemotePath(`~/${filename}`);
    }
    if (!open) {
      setRecursive(false);
      setPreserve(false);
    }
  }, [open, localPath]);

  const handleConfirm = () => {
    if (!remotePath.trim()) return;
    setLoading(true);
    onConfirm(localPath, remotePath.trim(), { recursive, preserve });
    setLoading(false);
    onClose();
  };

  const displayName = host ? (host.label.trim() || host.hostname) : '';

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`Upload to ${displayName}`}
      maxWidth={400}
      footer={
        <div className="modal__actions">
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleConfirm}
            disabled={loading || !remotePath.trim()}
          >
            {loading ? <Spinner size={14} /> : null}
            Upload
          </Button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div>
          <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
            Local file
          </span>
          <p
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--ink)',
              margin: '4px 0 0',
              wordBreak: 'break-all',
            }}
          >
            {localPath}
          </p>
        </div>

        <TextField
          label="Remote destination"
          value={remotePath}
          onChange={(e) => setRemotePath(e.target.value)}
          placeholder="~/filename.txt"
          disabled={loading}
        />

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
