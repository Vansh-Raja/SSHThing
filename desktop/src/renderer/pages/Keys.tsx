/**
 * Keys — overview of SSH keys in use. We don't have a standalone key store
 * yet (keys live inside hosts), so this page surfaces:
 *   - hosts that authenticate by key
 *   - a "Generate new key" action that returns a fresh keypair
 *
 * The generated key is shown once; the user copies the public key out
 * and the private key is held only in memory until they decide what to
 * do with it (typically attach to a host via the host drawer).
 */
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import Select from '../ui/Select';
import TextField from '../ui/TextField';
import { toast } from '../ui/toast';
import { CopyIcon, KeyIcon, PlusIcon } from '../components/icons';

const KEY_TYPES: { value: string; label: string }[] = [
  { value: 'ed25519', label: 'Ed25519 (recommended)' },
  { value: 'rsa', label: 'RSA 4096' },
  { value: 'ecdsa', label: 'ECDSA p-256' },
];

function copyToClipboard(text: string, label: string) {
  navigator.clipboard
    .writeText(text)
    .then(() => toast.success(`${label} copied`))
    .catch(() => toast.error('Could not copy'));
}

export default function Keys() {
  const navigate = useNavigate();
  const [hosts, setHosts] = useState<HostSummary[]>([]);
  const [loading, setLoading] = useState(true);

  // Generate modal
  const [genOpen, setGenOpen] = useState(false);
  const [genType, setGenType] = useState('ed25519');
  const [genComment, setGenComment] = useState('');
  const [genLoading, setGenLoading] = useState(false);
  const [generated, setGenerated] = useState<GenerateKeyResult | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await window.sshthing.listHosts();
      setHosts(r.hosts);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { void load(); }, [load]);

  const keyHosts = hosts.filter((h) => h.authMode === 'key');
  const passwordHosts = hosts.filter((h) => h.authMode === 'password');
  const noAuthHosts = hosts.filter((h) => h.authMode === 'none');

  const handleGenerate = useCallback(async () => {
    setGenLoading(true);
    try {
      const r = await window.sshthing.generateKey(genType, genComment.trim() || 'sshthing');
      setGenerated(r);
    } catch (err) {
      toast.error((err as Error).message ?? 'Generation failed');
    } finally {
      setGenLoading(false);
    }
  }, [genType, genComment]);

  const closeGen = useCallback(() => {
    setGenOpen(false);
    setGenerated(null);
    setGenComment('');
  }, []);

  return (
    <div className="page-scroll" style={{ width: '100%' }}>
      <div
        style={{
          padding: '32px 36px',
          maxWidth: 760,
          margin: '0 auto',
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          <h1 style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em', margin: 0, flex: 1 }}>
            SSH Keys
          </h1>
          <Button variant="primary" onClick={() => setGenOpen(true)}>
            <PlusIcon /> Generate key
          </Button>
        </div>

        <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
          SSHThing stores private keys encrypted in your vault, scoped per host. Generate a new pair below
          or attach an existing key to a host from the host drawer.
        </p>

        {loading ? (
          <p style={{ color: 'var(--muted)', fontSize: 13 }}>Loading hosts…</p>
        ) : (
          <>
            {/* Key auth hosts */}
            <section className="settings-section">
              <div className="settings-section__title">
                Hosts using key auth · {keyHosts.length}
              </div>
              <div className="settings-section__body">
                {keyHosts.length === 0 ? (
                  <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
                    No hosts use SSH-key authentication yet.
                  </p>
                ) : (
                  keyHosts.map((h) => (
                    <div key={h.id} className="settings-row">
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <span style={{ color: 'var(--accent)' }}><KeyIcon /></span>
                        <div>
                          <div className="settings-row__label">{h.label.trim() || h.hostname}</div>
                          <div className="settings-row__hint">{h.username}@{h.hostname}{h.port && h.port !== 22 ? `:${h.port}` : ''}</div>
                        </div>
                      </div>
                      <Button variant="ghost" onClick={() => navigate(`/hosts?host=${encodeURIComponent(h.id)}`)}>
                        Open host
                      </Button>
                    </div>
                  ))
                )}
              </div>
            </section>

            {/* Other auth modes (informational) */}
            {(passwordHosts.length > 0 || noAuthHosts.length > 0) && (
              <section className="settings-section">
                <div className="settings-section__title">Other auth modes</div>
                <div className="settings-section__body">
                  {passwordHosts.length > 0 && (
                    <div className="settings-row">
                      <div>
                        <div className="settings-row__label">Password</div>
                        <div className="settings-row__hint">{passwordHosts.length} host(s)</div>
                      </div>
                    </div>
                  )}
                  {noAuthHosts.length > 0 && (
                    <div className="settings-row">
                      <div>
                        <div className="settings-row__label">Agent / no stored credential</div>
                        <div className="settings-row__hint">{noAuthHosts.length} host(s)</div>
                      </div>
                    </div>
                  )}
                </div>
              </section>
            )}
          </>
        )}
      </div>

      {/* Generate modal */}
      <Modal open={genOpen} onClose={closeGen} title={generated ? 'Key generated' : 'Generate SSH key'}>
        {!generated ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <Select
              label="Key type"
              options={KEY_TYPES}
              value={genType}
              onChange={(e) => setGenType(e.target.value)}
            />
            <TextField
              label="Comment (optional)"
              value={genComment}
              onChange={(e) => setGenComment(e.target.value)}
              placeholder="user@machine"
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 8 }}>
              <Button variant="ghost" onClick={closeGen} disabled={genLoading}>Cancel</Button>
              <Button variant="primary" onClick={handleGenerate} loading={genLoading}>
                Generate
              </Button>
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
              Your new <strong style={{ color: 'var(--ink)' }}>{generated.keyType}</strong> keypair was generated.
              Copy the public key out before closing — the private half can be attached to a host from the host drawer.
            </p>

            <div className="field">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span className="field__label">Public key</span>
                <Button
                  variant="ghost"
                  onClick={() => copyToClipboard(generated.publicKey, 'Public key')}
                  style={{ height: 26, padding: '0 8px', fontSize: 12 }}
                >
                  <CopyIcon /> Copy
                </Button>
              </div>
              <textarea
                className="field__input"
                readOnly
                value={generated.publicKey}
                rows={3}
                style={{ minHeight: 70 }}
              />
            </div>

            <div className="field">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span className="field__label">Private key</span>
                <Button
                  variant="ghost"
                  onClick={() => copyToClipboard(generated.privateKey, 'Private key')}
                  style={{ height: 26, padding: '0 8px', fontSize: 12 }}
                >
                  <CopyIcon /> Copy
                </Button>
              </div>
              <textarea
                className="field__input"
                readOnly
                value={generated.privateKey}
                rows={6}
                style={{ minHeight: 140, fontSize: 11 }}
              />
              <p className="field__label" style={{ color: 'var(--danger)', textTransform: 'none', fontWeight: 400 }}>
                Treat this private key as a secret. SSHThing won't show it again.
              </p>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 4 }}>
              <Button variant="primary" onClick={closeGen}>Done</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
