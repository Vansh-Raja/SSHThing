/**
 * HostDrawer — create/edit a host. Slide-out form panel.
 * Handles: key paste, key generate (ed25519/rsa/ecdsa), password auth.
 * Drag-drop a .pem file onto the form to import.
 * Reveal-on-hold: credential fields reveal only while holding mouse/spacebar.
 * Tags: chip input with backspace to remove last tag.
 * Fullscreen key editor: opens Modal for long key editing (Cmd+E or "v" shortcut).
 */
import {
  useState,
  useCallback,
  useEffect,
  useRef,
  type ReactNode,
  type CSSProperties,
  type ChangeEvent,
  type DragEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
} from 'react';
import Drawer from '../ui/Drawer';
import TextField from '../ui/TextField';
import Select from '../ui/Select';
import Button from '../ui/Button';
import Tag from '../ui/Tag';
import Modal from '../ui/Modal';
import { toast } from '../ui/toast';

type AuthMethod = 'key' | 'password' | 'none';
type KeyTab = 'paste' | 'generate';

type HostForm = {
  label: string;
  hostname: string;
  username: string;
  port: string;
  group: string;
  tagsInput: string;
  tags: string[];
  authMode: AuthMethod;
  keyTab: KeyTab;
  keyPem: string;
  password: string;
  keyType: string;
  keyComment: string;
};

const DEFAULT_FORM: HostForm = {
  label: '',
  hostname: '',
  username: '',
  port: '22',
  group: '',
  tagsInput: '',
  tags: [],
  authMode: 'key',
  keyTab: 'paste',
  keyPem: '',
  password: '',
  keyType: 'ed25519',
  keyComment: '',
};

type HostDrawerProps = {
  open: boolean;
  onClose: () => void;
  groups: GroupSummary[];
  host?: HostSummary | null;
  onSaved: () => void;
};

type GeneratedKey = {
  publicKey: string;
  privateKey: string;
};

/** Validate that the text looks like a PEM / OpenSSH key. */
function isValidKeyContent(text: string): boolean {
  const trimmed = text.trim();
  return trimmed.startsWith('-----BEGIN ') && trimmed.includes('-----END ');
}

/** HoldToReveal — reveals a password / PEM field only while the button is held.
 *  On mousedown + Space keydown: reveal. On mouseup / mouseleave / Space keyup: mask.
 *  When `rows` is set, renders a textarea (for private keys).
 *  Pass `extraAction` to render an extra button in the label row (e.g. "EXPAND"). */
function HoldToReveal({
  value,
  label,
  placeholder,
  onChange,
  rows,
  style,
  extraAction,
}: {
  value: string;
  label: string;
  placeholder?: string;
  onChange: (v: string) => void;
  rows?: number;
  style?: CSSProperties;
  extraAction?: ReactNode;
}) {
  const [revealed, setRevealed] = useState(false);
  const isHeld = useRef(false);

  const reveal = () => { isHeld.current = true; setRevealed(true); };
  const mask  = () => { isHeld.current = false; setRevealed(false); };

  // Space-key hold on the button
  const handleBtnKeyDown = (e: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (e.key === ' ' || e.code === 'Space') { e.preventDefault(); reveal(); }
  };
  const handleBtnKeyUp = (e: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (e.key === ' ' || e.code === 'Space') mask();
  };

  return (
    <div className="field">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
        <span className="field__label">{label}</span>
        <div style={{ display: 'flex', gap: 4 }}>
          {extraAction}
          <button
            type="button"
            onMouseDown={(e: ReactMouseEvent<HTMLButtonElement>) => { e.preventDefault(); reveal(); }}
            onMouseUp={mask}
            onMouseLeave={mask}
            onKeyDown={handleBtnKeyDown}
            onKeyUp={handleBtnKeyUp}
            aria-label="Hold to reveal"
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: revealed ? 'var(--accent)' : 'var(--muted)',
              fontSize: 11,
              fontFamily: 'var(--font-mono)',
              padding: '2px 4px',
              userSelect: 'none',
            }}
          >
            {revealed ? 'HIDE' : 'HOLD'}
          </button>
        </div>
      </div>
      {rows ? (
        <textarea
          className="field__input"
          rows={rows}
          placeholder={placeholder}
          value={value}
          onChange={(e: ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
          style={{
            resize: 'vertical',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            ...(revealed ? {} : { WebkitTextSecurity: 'disc' as const }),
            ...style,
          }}
        />
      ) : (
        <input
          className="field__input"
          type={revealed ? 'text' : 'password'}
          placeholder={placeholder}
          value={value}
          onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
          style={style}
          autoComplete="off"
        />
      )}
    </div>
  );
}

export default function HostDrawer({
  open,
  onClose,
  groups,
  host,
  onSaved,
}: HostDrawerProps) {
  const isEdit = !!host;

  const [form, setForm] = useState<HostForm>(() => {
    if (host) {
      return {
        ...DEFAULT_FORM,
        label: host.label,
        hostname: host.hostname,
        username: host.username,
        port: String(host.port),
        group: host.group,
        tags: host.tags,
        authMode: host.authMode,
      };
    }
    return DEFAULT_FORM;
  });

  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Partial<Record<keyof HostForm, string>>>({});
  const [generatedKey, setGeneratedKey] = useState<GeneratedKey | null>(null);
  const [genLoading, setGenLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [keyEditorOpen, setKeyEditorOpen] = useState(false);
  const [keyEditorValue, setKeyEditorValue] = useState('');
  // Edit mode only: tracks whether the user has clicked "View existing" to
  // fetch the stored credential into the form. Once revealed the field is
  // pre-populated and the user can change-then-save (or leave-as-is and save
  // → handleSave routes through updateHostWithKey).
  const [viewLoading, setViewLoading] = useState(false);

  const tagsInputRef = useRef<HTMLInputElement>(null);

  // Reset form whenever the drawer opens, including when switching between
  // "add new" and "edit existing" host to prevent stale state from leaking.
  useEffect(() => {
    if (!open) return;
    if (host) {
      setForm({
        ...DEFAULT_FORM,
        label: host.label,
        hostname: host.hostname,
        username: host.username,
        port: String(host.port),
        group: host.group,
        tags: host.tags,
        authMode: host.authMode,
      });
    } else {
      setForm(DEFAULT_FORM);
    }
    setGeneratedKey(null);
    setErrors({});
  }, [open, host]);

  const set = <K extends keyof HostForm>(key: K, value: HostForm[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => ({ ...e, [key]: undefined }));
  };

  const validate = (): boolean => {
    const errs: Partial<Record<keyof HostForm, string>> = {};
    if (!form.hostname.trim()) errs.hostname = 'Required';
    if (!form.username.trim()) errs.username = 'Required';
    const portNum = parseInt(form.port, 10);
    if (isNaN(portNum) || portNum < 1 || portNum > 65535) errs.port = '1–65535';
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSave = async () => {
    if (!validate()) return;
    setLoading(true);
    try {
      const port = parseInt(form.port, 10);
      // Pull the new credential out of the form once so both edit + create
      // paths can reuse the same value (key paste OR generated-key OR
      // password input — whichever matches authMode).
      const newCredential =
        form.authMode === 'key'
          ? (generatedKey ? generatedKey.privateKey : form.keyPem).trim()
          : form.authMode === 'password'
            ? form.password
            : '';

      if (isEdit && host) {
        // PREVIOUS BUG: the edit branch called updateHost() WITHOUT a
        // plainKey, so any credential the user typed/generated/dropped
        // was silently discarded. Now we route through updateHostWithKey
        // when the credential field is non-empty (treating that as
        // "user wants to replace the existing secret"). An empty field
        // means "leave existing credential alone" and we use plain
        // updateHost. The daemon's UpdateHostWithKey covers both keys
        // and passwords because the host model's KeyType field carries
        // the auth mode and the secret is stored verbatim.
        const baseUpdate = {
          id: host.id,
          label: form.label || undefined,
          hostname: form.hostname,
          username: form.username,
          port,
          group: form.group || undefined,
          tags: form.tags,
          authMode: form.authMode,
        };
        if (newCredential) {
          await window.sshthing.updateHostWithKey({
            ...baseUpdate,
            plainKey: newCredential,
          });
        } else {
          await window.sshthing.updateHost(baseUpdate);
        }
        toast.success('Host updated');
      } else {
        const payload: HostCreate = {
          label: form.label || undefined,
          hostname: form.hostname,
          username: form.username,
          port,
          group: form.group || undefined,
          tags: form.tags,
          authMode: form.authMode,
        };
        if (form.authMode === 'key') {
          if (newCredential) payload.plainKey = newCredential;
        } else if (form.authMode === 'password') {
          if (newCredential) payload.plainPassword = newCredential;
        }
        await window.sshthing.createHost(payload);
        toast.success('Host created');
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      const e = err as Error & { code?: number };
      // Gracefully handle "method not found" from not-yet-implemented daemon RPCs
      if (e.code === -32601) {
        toast.error('This feature requires a newer daemon version.');
      } else {
        toast.error(e.message ?? 'Failed to save host');
      }
    } finally {
      setLoading(false);
    }
  };

  // Edit mode "View existing credential" — reveal the stored secret and
  // drop it into the form so the user can see what's there, then either
  // (a) leave it alone and save (no-op) or (b) edit and save (replaces
  // via updateHostWithKey). Audit-logged on the daemon side.
  const handleViewExisting = useCallback(async () => {
    if (!host) return;
    setViewLoading(true);
    try {
      const res = await window.sshthing.revealCredential(host.id);
      if (res.authMode === 'key') {
        set('keyPem', res.credential);
        set('keyTab', 'paste');
      } else if (res.authMode === 'password') {
        set('password', res.credential);
      }
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Could not reveal existing credential');
    } finally {
      setViewLoading(false);
    }
  }, [host]);

  const handleGenerateKey = async () => {
    setGenLoading(true);
    setGeneratedKey(null);
    try {
      const comment = form.keyComment || `${form.username}@sshthing`;
      const result = await window.sshthing.generateKey(form.keyType, comment);
      setGeneratedKey({ publicKey: result.publicKey, privateKey: result.privateKey });
      toast.success('Key pair generated');
    } catch (err: unknown) {
      const e = err as Error & { code?: number };
      if (e.code === -32601) {
        toast.error('Key generation requires a newer daemon version.');
      } else {
        toast.error(e.message ?? 'Key generation failed');
      }
    } finally {
      setGenLoading(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text).then(
      () => toast.success(`${label} copied`),
      () => toast.error('Failed to copy'),
    );
  };

  // Drag-drop key import with content validation
  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => setDragOver(false);

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setDragOver(false);
      const file = e.dataTransfer.files[0];
      if (!file) return;
      const ext = file.name.split('.').pop()?.toLowerCase();
      if (ext !== 'pem' && ext !== 'key' && !file.name.startsWith('id_')) {
        toast.error('Drop a .pem or OpenSSH private key file');
        return;
      }
      const text = await file.text();
      if (!isValidKeyContent(text)) {
        toast.error('File does not look like a valid PEM / OpenSSH key');
        return;
      }
      set('keyPem', text);
      set('keyTab', 'paste');
      toast.success(`Loaded key from ${file.name}`);
    },
    [],
  );

  // Tags chip input — backspace on empty removes last chip
  const addTag = () => {
    const t = form.tagsInput.trim().toLowerCase();
    if (!t || form.tags.includes(t)) { set('tagsInput', ''); return; }
    set('tags', [...form.tags, t]);
    set('tagsInput', '');
  };

  const removeTag = (tag: string) => set('tags', form.tags.filter((t) => t !== tag));

  const handleTagsKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      addTag();
    } else if (e.key === 'Backspace' && form.tagsInput === '' && form.tags.length > 0) {
      e.preventDefault();
      const lastTag = form.tags[form.tags.length - 1]!;
      removeTag(lastTag);
    }
  };

  // Fullscreen key editor
  const openKeyEditor = () => {
    setKeyEditorValue(form.keyPem);
    setKeyEditorOpen(true);
  };

  const saveKeyEditor = () => {
    set('keyPem', keyEditorValue);
    setKeyEditorOpen(false);
  };

  // Bind `v` to open key editor when paste-key field is focused (if textarea is not active)
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName;
      if (tag === 'TEXTAREA' || tag === 'INPUT') return; // don't intercept text entry
      if (e.key === 'v' && !e.metaKey && !e.ctrlKey && form.keyTab === 'paste') {
        e.preventDefault();
        openKeyEditor();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, form.keyTab, form.keyPem]); // eslint-disable-line react-hooks/exhaustive-deps

  const groupOptions = [
    { value: '', label: '— None —' },
    ...groups.map((g) => ({ value: g.name, label: g.name })),
  ];

  const keyTypeOptions = [
    { value: 'ed25519', label: 'Ed25519 (recommended)' },
    { value: 'rsa', label: 'RSA 4096' },
    { value: 'ecdsa', label: 'ECDSA P-256' },
  ];

  return (
    <>
      <Drawer
        open={open}
        onClose={onClose}
        title={isEdit ? 'Edit Host' : 'Add Host'}
        footer={
          <>
            <div className="drawer__footer-left" />
            <Button variant="ghost" onClick={onClose} disabled={loading}>
              Cancel
            </Button>
            <Button variant="primary" onClick={handleSave} loading={loading}>
              {isEdit ? 'Save Changes' : 'Add Host'}
            </Button>
          </>
        }
      >
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 14,
            outline: dragOver ? '2px dashed var(--accent)' : undefined,
            borderRadius: 2,
            padding: dragOver ? 4 : 0,
            transition: 'outline 100ms ease',
          }}
        >
          {dragOver && (
            <div
              style={{
                textAlign: 'center',
                color: 'var(--accent)',
                fontSize: 12,
                fontWeight: 700,
                padding: '12px 0',
              }}
            >
              Drop key file to import
            </div>
          )}

          <TextField
            label="Label (optional)"
            placeholder="My Server"
            value={form.label}
            onChange={(e: ChangeEvent<HTMLInputElement>) => set('label', e.target.value)}
          />

          <TextField
            label="Hostname / IP"
            placeholder="192.168.1.1"
            value={form.hostname}
            onChange={(e: ChangeEvent<HTMLInputElement>) => set('hostname', e.target.value)}
            error={errors.hostname}
            autoComplete="off"
          />

          <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 10 }}>
            <TextField
              label="Username"
              placeholder="ubuntu"
              value={form.username}
              onChange={(e: ChangeEvent<HTMLInputElement>) => set('username', e.target.value)}
              error={errors.username}
              autoComplete="off"
            />
            <TextField
              label="Port"
              placeholder="22"
              value={form.port}
              onChange={(e: ChangeEvent<HTMLInputElement>) => set('port', e.target.value)}
              error={errors.port}
              style={{ width: 80 }}
            />
          </div>

          <Select
            label="Group"
            options={groupOptions}
            value={form.group}
            onChange={(e) => set('group', e.target.value)}
          />

          {/* Tags chip input */}
          <div className="field">
            <label className="field__label">Tags</label>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 5,
                alignItems: 'center',
                padding: '5px 8px',
                border: '1px solid var(--line-2)',
                background: 'var(--paper-3)',
                borderRadius: 'var(--radius)',
                cursor: 'text',
                minHeight: 34,
              }}
              onClick={() => tagsInputRef.current?.focus()}
            >
              {form.tags.map((tag) => (
                <Tag key={tag} onRemove={() => removeTag(tag)}>{tag}</Tag>
              ))}
              <input
                ref={tagsInputRef}
                placeholder={form.tags.length === 0 ? 'Add tags…' : ''}
                value={form.tagsInput}
                onChange={(e: ChangeEvent<HTMLInputElement>) => set('tagsInput', e.target.value)}
                onKeyDown={handleTagsKeyDown}
                style={{
                  background: 'transparent',
                  border: 'none',
                  outline: 'none',
                  fontSize: 12,
                  color: 'var(--ink)',
                  minWidth: 80,
                  flex: 1,
                  padding: '1px 2px',
                  height: 22,
                }}
              />
            </div>
            <span style={{ color: 'var(--muted)', fontSize: 11 }}>
              Enter or comma to add · Backspace to remove
            </span>
          </div>

          {/* Auth method */}
          <div className="field">
            <span className="field__label">Auth method</span>
            <div className="segmented">
              {(['key', 'password', 'none'] as AuthMethod[]).map((m) => (
                <button
                  key={m}
                  type="button"
                  className="segmented__item"
                  aria-selected={form.authMode === m}
                  onClick={() => set('authMode', m)}
                >
                  {m === 'key' ? 'SSH Key' : m === 'password' ? 'Password' : 'None'}
                </button>
              ))}
            </div>
          </div>

          {form.authMode === 'key' && (
            <>
              <div className="field">
                <span className="field__label">Key source</span>
                <div className="segmented">
                  {(['paste', 'generate'] as KeyTab[]).map((t) => (
                    <button
                      key={t}
                      type="button"
                      className="segmented__item"
                      aria-selected={form.keyTab === t}
                      onClick={() => set('keyTab', t)}
                    >
                      {t === 'paste' ? 'Paste / Drop' : 'Generate'}
                    </button>
                  ))}
                </div>
              </div>

              {form.keyTab === 'paste' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <HoldToReveal
                    label="Private key (PEM / OpenSSH)"
                    value={form.keyPem}
                    placeholder={'-----BEGIN OPENSSH PRIVATE KEY-----\n...'}
                    onChange={(v) => set('keyPem', v)}
                    rows={6}
                    style={{ resize: 'vertical' }}
                    extraAction={
                      <>
                        {isEdit && (
                          <button
                            type="button"
                            onClick={handleViewExisting}
                            disabled={viewLoading}
                            title="Load the currently-stored key into the form"
                            style={{
                              background: 'none',
                              border: 'none',
                              cursor: viewLoading ? 'progress' : 'pointer',
                              color: 'var(--muted)',
                              fontSize: 11,
                              fontFamily: 'var(--font-mono)',
                              padding: '2px 4px',
                            }}
                          >
                            {viewLoading ? 'LOADING…' : 'VIEW EXISTING'}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={openKeyEditor}
                          title="Open fullscreen editor (v)"
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'var(--muted)',
                            fontSize: 11,
                            fontFamily: 'var(--font-mono)',
                            padding: '2px 4px',
                          }}
                        >
                          EXPAND
                        </button>
                      </>
                    }
                  />
                  <span style={{ color: 'var(--muted)', fontSize: 11 }}>
                    Or drag-and-drop a .pem file anywhere in this form.
                  </span>
                </div>
              )}

              {form.keyTab === 'generate' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <Select
                    label="Key type"
                    options={keyTypeOptions}
                    value={form.keyType}
                    onChange={(e) => set('keyType', e.target.value)}
                  />
                  <TextField
                    label="Comment (optional)"
                    placeholder="user@host"
                    value={form.keyComment}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => set('keyComment', e.target.value)}
                  />
                  <Button
                    variant="secondary"
                    onClick={handleGenerateKey}
                    loading={genLoading}
                  >
                    Generate Key Pair
                  </Button>

                  {generatedKey && (
                    <div
                      style={{
                        border: '1.5px solid var(--line)',
                        borderRadius: 2,
                        padding: 12,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 8,
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <span className="field__label">Public key</span>
                        <Button
                          variant="ghost"
                          onClick={() => copyToClipboard(generatedKey.publicKey, 'Public key')}
                          style={{ fontSize: 10 }}
                        >
                          Copy
                        </Button>
                      </div>
                      <pre
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--muted)',
                          margin: 0,
                          overflowX: 'auto',
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-all',
                          background: 'var(--paper-2)',
                          padding: 8,
                          borderRadius: 2,
                        }}
                      >
                        {generatedKey.publicKey}
                      </pre>
                      <p style={{ color: 'var(--muted)', fontSize: 11, margin: 0 }}>
                        Add this public key to <code>~/.ssh/authorized_keys</code> on the server.
                        The private key will be stored encrypted in your vault.
                      </p>
                    </div>
                  )}
                </div>
              )}
            </>
          )}

          {form.authMode === 'password' && (
            <HoldToReveal
              label="Password"
              placeholder="SSH password"
              value={form.password}
              onChange={(v) => set('password', v)}
              extraAction={
                isEdit ? (
                  <button
                    type="button"
                    onClick={handleViewExisting}
                    disabled={viewLoading}
                    title="Load the currently-stored password into the form"
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: viewLoading ? 'progress' : 'pointer',
                      color: 'var(--muted)',
                      fontSize: 11,
                      fontFamily: 'var(--font-mono)',
                      padding: '2px 4px',
                    }}
                  >
                    {viewLoading ? 'LOADING…' : 'VIEW EXISTING'}
                  </button>
                ) : null
              }
            />
          )}
        </div>
      </Drawer>

      {/* Fullscreen key editor modal */}
      <Modal
        open={keyEditorOpen}
        onClose={() => setKeyEditorOpen(false)}
        title="Edit SSH private key"
        maxWidth={680}
        footer={
          <div className="modal__actions">
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => setKeyEditorOpen(false)}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={saveKeyEditor}
            >
              Save
            </button>
          </div>
        }
      >
        <textarea
          className="field__input"
          rows={20}
          value={keyEditorValue}
          onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setKeyEditorValue(e.target.value)}
          autoFocus
          spellCheck={false}
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            resize: 'vertical',
            minHeight: 300,
            lineHeight: 1.5,
          }}
          placeholder={'-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----'}
        />
      </Modal>
    </>
  );
}
