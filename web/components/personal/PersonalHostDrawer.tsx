"use client";

import { useEffect, useState } from "react";

import type { PersonalGroup, PersonalHost } from "./types";

type Props = {
  host?: PersonalHost | null;
  groups: PersonalGroup[];
  onClose: () => void;
  onSave: (host: PersonalHost) => void;
};

function newSyncId(): string {
  return crypto.randomUUID();
}

function nowISO(): string {
  return new Date().toISOString();
}

export default function PersonalHostDrawer({
  host,
  groups,
  onClose,
  onSave,
}: Props) {
  const [form, setForm] = useState<PersonalHost>(() =>
    host ?? {
      sync_id: newSyncId(),
      label: "",
      group_name: "",
      tags: [],
      hostname: "",
      username: "",
      port: 22,
      key_type: "password",
      secret: "",
      created_at: nowISO(),
      updated_at: nowISO(),
    },
  );
  const [tags, setTags] = useState((host?.tags ?? []).join(", "));

  useEffect(() => {
    setForm(
      host ?? {
        sync_id: newSyncId(),
        label: "",
        group_name: "",
        tags: [],
        hostname: "",
        username: "",
        port: 22,
        key_type: "password",
        secret: "",
        created_at: nowISO(),
        updated_at: nowISO(),
      },
    );
    setTags((host?.tags ?? []).join(", "));
  }, [host]);

  return (
    <div className="drawer-backdrop" role="presentation">
      <aside className="drawer" aria-label="Personal host editor">
        <div className="drawer__header">
          <div>
            <span className="eyebrow">Personal library</span>
            <h2>{host ? "Edit host" : "Add host"}</h2>
          </div>
          <button className="btn btn--ghost" type="button" onClick={onClose}>
            Close
          </button>
        </div>

        <form
          className="drawer__body stack"
          onSubmit={(event) => {
            event.preventDefault();
            onSave({
              ...form,
              port: Number(form.port) || 22,
              tags: tags
                .split(",")
                .map((tag) => tag.trim())
                .filter(Boolean),
              updated_at: nowISO(),
            });
          }}
        >
          <label className="field">
            <span className="field__label">Label</span>
            <input
              className="field__input"
              value={form.label}
              onChange={(event) =>
                setForm((cur) => ({ ...cur, label: event.target.value }))
              }
            />
          </label>
          <div className="grid-2">
            <label className="field">
              <span className="field__label">Hostname</span>
              <input
                className="field__input"
                required
                value={form.hostname}
                onChange={(event) =>
                  setForm((cur) => ({ ...cur, hostname: event.target.value }))
                }
              />
            </label>
            <label className="field">
              <span className="field__label">Username</span>
              <input
                className="field__input"
                required
                value={form.username}
                onChange={(event) =>
                  setForm((cur) => ({ ...cur, username: event.target.value }))
                }
              />
            </label>
          </div>
          <div className="grid-2">
            <label className="field">
              <span className="field__label">Port</span>
              <input
                className="field__input"
                type="number"
                value={form.port}
                onChange={(event) =>
                  setForm((cur) => ({ ...cur, port: Number(event.target.value) }))
                }
              />
            </label>
            <label className="field">
              <span className="field__label">Group</span>
              <input
                className="field__input"
                list="personal-groups"
                value={form.group_name ?? ""}
                onChange={(event) =>
                  setForm((cur) => ({ ...cur, group_name: event.target.value }))
                }
              />
              <datalist id="personal-groups">
                {groups.map((group) => (
                  <option key={group.sync_id} value={group.name} />
                ))}
              </datalist>
            </label>
          </div>
          <label className="field">
            <span className="field__label">Tags</span>
            <input
              className="field__input"
              value={tags}
              onChange={(event) => setTags(event.target.value)}
              placeholder="gpu, prod"
            />
          </label>
          <label className="field">
            <span className="field__label">Credential type</span>
            <select
              className="field__input"
              value={form.key_type || "password"}
              onChange={(event) =>
                setForm((cur) => ({ ...cur, key_type: event.target.value }))
              }
            >
              <option value="password">password</option>
              <option value="pasted">private key</option>
              <option value="">agent / no stored secret</option>
            </select>
          </label>
          <label className="field">
            <span className="field__label">Credential</span>
            <textarea
              className="field__input field__textarea"
              value={form.secret ?? ""}
              onChange={(event) =>
                setForm((cur) => ({ ...cur, secret: event.target.value }))
              }
              placeholder={
                host
                  ? "Leave blank to keep existing encrypted credential"
                  : "Paste password or private key"
              }
            />
          </label>
          <div className="row">
            <button className="btn btn--primary" type="submit">
              Save host
            </button>
            <button className="btn" type="button" onClick={onClose}>
              Cancel
            </button>
          </div>
        </form>
      </aside>
    </div>
  );
}
