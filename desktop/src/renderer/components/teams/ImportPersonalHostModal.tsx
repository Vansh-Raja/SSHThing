/**
 * ImportPersonalHostModal — lets an admin import a personal vault host into
 * the current team. Handles conflict resolution (Update / Duplicate / Cancel)
 * mirroring the TUI's importPersonalHostToCurrentTeam flow.
 */
import { useCallback, useEffect, useState } from 'react';
import Modal from '../../ui/Modal';
import { toast } from '../../ui/toast';

type ImportPersonalHostModalProps = {
  open: boolean;
  onClose: () => void;
  teamId: string;
  onImported: () => void;
};

type Phase = 'select' | 'preview' | 'conflict' | 'importing';

export default function ImportPersonalHostModal({
  open,
  onClose,
  teamId,
  onImported,
}: ImportPersonalHostModalProps) {
  const [phase, setPhase] = useState<Phase>('select');
  const [personalHosts, setPersonalHosts] = useState<HostSummary[]>([]);
  const [hostsLoading, setHostsLoading] = useState(false);
  const [hostsError, setHostsError] = useState<string | null>(null);

  const [selectedHostId, setSelectedHostId] = useState<string | null>(null);
  const [preview, setPreview] = useState<ImportPersonalHostPreviewResult | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [importing, setImporting] = useState(false);

  // Load personal hosts when modal opens.
  useEffect(() => {
    if (!open) {
      setPhase('select');
      setSelectedHostId(null);
      setPreview(null);
      return;
    }
    setHostsLoading(true);
    setHostsError(null);
    window.sshthing
      .listHosts()
      .then((result) => {
        setPersonalHosts(result.hosts ?? []);
      })
      .catch((err: unknown) => {
        setHostsError((err as Error).message ?? 'Failed to load personal hosts');
      })
      .finally(() => setHostsLoading(false));
  }, [open]);

  const handlePreview = useCallback(async () => {
    if (!selectedHostId) return;
    setPreviewing(true);
    try {
      const result = await window.sshthing.teamsHostsImportPersonalPreview(selectedHostId, teamId);
      setPreview(result);
      if (result.hasConflict && result.isIdentical) {
        toast.info('This host has already been imported into the team.');
        setPhase('select');
        return;
      }
      if (result.hasConflict) {
        setPhase('conflict');
      } else {
        setPhase('preview');
      }
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to check for conflicts');
    } finally {
      setPreviewing(false);
    }
  }, [selectedHostId, teamId]);

  const handleCommit = useCallback(async (action: ImportPersonalHostAction) => {
    if (!selectedHostId || !preview) return;
    setImporting(true);
    try {
      await window.sshthing.teamsHostsImportPersonalCommit({
        personalHostId: selectedHostId,
        teamId,
        action,
        existingHostId: action === 'update' ? (preview.existingHostId ?? '') : undefined,
      });
      const selectedHost = personalHosts.find((h) => h.id === selectedHostId);
      const label = selectedHost?.label || selectedHost?.hostname || selectedHostId;
      toast.success(`Imported "${label}" into team`);
      onImported();
      onClose();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Import failed');
    } finally {
      setImporting(false);
    }
  }, [selectedHostId, preview, teamId, personalHosts, onImported, onClose]);

  const selectedHost = personalHosts.find((h) => h.id === selectedHostId);

  // ── Render ──

  if (!open) return null;

  if (phase === 'conflict' && preview) {
    const existingLabel = preview.existingLabel ?? preview.existingHostId ?? 'existing host';
    const proposedLabel = preview.proposed.label || preview.proposed.hostname;

    return (
      <Modal
        open={open}
        onClose={onClose}
        title="Conflict detected"
        maxWidth={500}
        footer={
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
            <button type="button" className="btn btn--ghost" onClick={onClose} disabled={importing}>
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => void handleCommit('duplicate')}
              disabled={importing}
            >
              {importing ? <span className="spinner" /> : 'Create duplicate'}
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleCommit('update')}
              disabled={importing}
            >
              {importing ? <span className="spinner" /> : 'Update existing'}
            </button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <p style={{ margin: 0, fontSize: 13, lineHeight: 1.55 }}>
            A similar host already exists in the team.
          </p>
          <dl
            style={{
              display: 'grid',
              gridTemplateColumns: 'max-content 1fr',
              gap: '4px 12px',
              margin: 0,
              fontSize: 12,
            }}
          >
            <dt style={{ color: 'var(--muted)' }}>Existing</dt>
            <dd style={{ margin: 0 }}>{existingLabel}</dd>
            <dt style={{ color: 'var(--muted)' }}>Proposed</dt>
            <dd style={{ margin: 0 }}>{proposedLabel}</dd>
          </dl>
          <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12, color: 'var(--muted)', lineHeight: 1.7 }}>
            <li><strong>Update existing</strong> — overwrite the team host with this host's data.</li>
            <li><strong>Create duplicate</strong> — add a new team host alongside the existing one.</li>
          </ul>
        </div>
      </Modal>
    );
  }

  if (phase === 'preview' && preview) {
    const p = preview.proposed;
    return (
      <Modal
        open={open}
        onClose={onClose}
        title="Import host"
        maxWidth={500}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn btn--ghost" onClick={() => setPhase('select')} disabled={importing}>
              Back
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleCommit('create')}
              disabled={importing}
            >
              {importing ? <span className="spinner" /> : 'Import'}
            </button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <p style={{ margin: 0, fontSize: 13, lineHeight: 1.55 }}>
            The following host will be imported as a shared-credential team host.
          </p>
          <dl
            style={{
              display: 'grid',
              gridTemplateColumns: 'max-content 1fr',
              gap: '4px 12px',
              margin: 0,
              fontSize: 12,
            }}
          >
            <dt style={{ color: 'var(--muted)' }}>Label</dt>
            <dd style={{ margin: 0 }}>{p.label || '—'}</dd>
            <dt style={{ color: 'var(--muted)' }}>Host</dt>
            <dd style={{ margin: 0, fontFamily: 'var(--font-mono)' }}>
              {p.username}@{p.hostname}:{p.port}
            </dd>
            <dt style={{ color: 'var(--muted)' }}>Credential</dt>
            <dd style={{ margin: 0 }}>
              {p.credentialType !== 'none' ? `${p.credentialType} (shared)` : 'None'}
            </dd>
            {p.group && (
              <>
                <dt style={{ color: 'var(--muted)' }}>Group</dt>
                <dd style={{ margin: 0 }}>{p.group}</dd>
              </>
            )}
          </dl>
          {p.credentialType !== 'none' && (
            <div
              style={{
                padding: '8px 12px',
                borderRadius: 'var(--radius)',
                background: 'color-mix(in srgb, var(--warning, #ff9800) 12%, transparent)',
                border: '1px solid color-mix(in srgb, var(--warning, #ff9800) 30%, transparent)',
                fontSize: 12,
                lineHeight: 1.55,
              }}
            >
              The personal credential will be copied to the team as a shared credential.
            </div>
          )}
        </div>
      </Modal>
    );
  }

  // Default: select phase
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Import from personal vault"
      maxWidth={480}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <button type="button" className="btn btn--ghost" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="btn btn--primary"
            onClick={() => void handlePreview()}
            disabled={!selectedHostId || previewing}
          >
            {previewing ? <span className="spinner" /> : 'Next'}
          </button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {hostsLoading && (
          <div style={{ fontSize: 12, color: 'var(--muted)' }}>Loading personal hosts…</div>
        )}
        {hostsError && (
          <div style={{ fontSize: 12, color: 'var(--danger)' }}>{hostsError}</div>
        )}
        {!hostsLoading && !hostsError && personalHosts.length === 0 && (
          <div style={{ fontSize: 12, color: 'var(--muted)' }}>No personal hosts in your vault.</div>
        )}
        {!hostsLoading && personalHosts.length > 0 && (
          <>
            <p style={{ margin: 0, fontSize: 12, color: 'var(--muted)', lineHeight: 1.55 }}>
              Select a personal host to import into this team. Its credential will
              become the shared team credential.
            </p>
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
                maxHeight: 280,
                overflowY: 'auto',
              }}
            >
              {personalHosts.map((h) => {
                const isSelected = h.id === selectedHostId;
                return (
                  <button
                    key={h.id}
                    type="button"
                    onClick={() => setSelectedHostId(h.id)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 10,
                      padding: '8px 12px',
                      borderRadius: 'var(--radius)',
                      background: isSelected ? 'color-mix(in srgb, var(--accent) 12%, var(--paper-2))' : 'var(--paper-2)',
                      border: `1px solid ${isSelected ? 'var(--accent)' : 'var(--line)'}`,
                      textAlign: 'left',
                      cursor: 'pointer',
                      width: '100%',
                    }}
                  >
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 13, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {h.label || h.hostname}
                      </div>
                      <div style={{ fontSize: 11, color: 'var(--muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {h.username}@{h.hostname}:{h.port}
                      </div>
                    </div>
                    <span className="chip" style={{ fontSize: 10, textTransform: 'uppercase', flexShrink: 0 }}>
                      {h.authMode}
                    </span>
                  </button>
                );
              })}
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}
