"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { apiRequest, errorMessage } from "../teams/api";
import { toast } from "../ui/toast";
import { decryptPersonalItem, derivePersonalVaultKey, encryptPersonalItem } from "../../lib/personalCrypto";
import PersonalHostDrawer from "./PersonalHostDrawer";
import PersonalUnlock from "./PersonalUnlock";
import type {
  PersonalActivityEvent,
  PersonalGroup,
  PersonalHost,
  PersonalVaultItem,
  PersonalVaultSummary,
} from "./types";

type Tab = "hosts" | "groups" | "settings" | "activity";

type ItemsResponse = {
  revision: string;
  items: PersonalVaultItem[];
};

function nowISO(): string {
  return new Date().toISOString();
}

function newGroup(name: string): PersonalGroup {
  const now = nowISO();
  return { sync_id: crypto.randomUUID(), name, created_at: now, updated_at: now };
}

export default function PersonalDashboard() {
  const [vault, setVault] = useState<PersonalVaultSummary | null>(null);
  const [cryptoKey, setCryptoKey] = useState<CryptoKey | null>(null);
  const [unlockError, setUnlockError] = useState("");
  const [hosts, setHosts] = useState<PersonalHost[]>([]);
  const [groups, setGroups] = useState<PersonalGroup[]>([]);
  const [events, setEvents] = useState<PersonalActivityEvent[]>([]);
  const [tab, setTab] = useState<Tab>("hosts");
  const [drawerHost, setDrawerHost] = useState<PersonalHost | null | undefined>(undefined);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const next = await apiRequest<PersonalVaultSummary>("/api/personal/vault");
        if (!cancelled) setVault(next);
      } catch (err) {
        toast.error(errorMessage(err, "personal_vault_load_failed"));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const loadEncryptedItems = useCallback(
    async (key: CryptoKey) => {
      const response = await apiRequest<ItemsResponse>("/api/personal/vault/items");
      const nextHosts: PersonalHost[] = [];
      const nextGroups: PersonalGroup[] = [];
      for (const item of response.items) {
        if (item.deletedAt) continue;
        if (item.itemType === "host") {
          nextHosts.push(await decryptPersonalItem<PersonalHost>(key, item.ciphertext));
        } else if (item.itemType === "group") {
          nextGroups.push(await decryptPersonalItem<PersonalGroup>(key, item.ciphertext));
        }
      }
      setHosts(nextHosts.sort((a, b) => (a.label || a.hostname).localeCompare(b.label || b.hostname)));
      setGroups(nextGroups.sort((a, b) => a.name.localeCompare(b.name)));
    },
    [],
  );

  const loadEvents = useCallback(async () => {
    try {
      setEvents(await apiRequest<PersonalActivityEvent[]>("/api/personal/vault/events"));
    } catch (err) {
      toast.error(errorMessage(err, "personal_events_failed"));
    }
  }, []);

  async function unlock(password: string) {
    if (!vault) return;
    try {
      setUnlockError("");
      const key = await derivePersonalVaultKey(password, vault.kdf.salt);
      await loadEncryptedItems(key);
      setCryptoKey(key);
      void loadEvents();
    } catch {
      setUnlockError("Could not decrypt this vault. Check the sync password.");
    }
  }

  async function uploadItems(items: PersonalVaultItem[]) {
    await apiRequest("/api/personal/vault/items", {
      method: "POST",
      body: JSON.stringify({
        deviceId: "browser",
        items,
      }),
    });
    void apiRequest("/api/personal/vault/events", {
      method: "POST",
      body: JSON.stringify({ source: "web", action: "edit", itemCount: items.length }),
    });
  }

  async function saveHost(host: PersonalHost) {
    if (!cryptoKey) return;
    const normalized: PersonalHost = {
      ...host,
      sync_id: host.sync_id || crypto.randomUUID(),
      label: host.label.trim(),
      hostname: host.hostname.trim(),
      username: host.username.trim(),
      group_name: host.group_name?.trim() ?? "",
      key_type: host.key_type,
      updated_at: nowISO(),
    };
    if (normalized.secret?.trim()) {
      normalized.key_data = "";
    } else {
      delete normalized.secret;
    }
    const ciphertext = await encryptPersonalItem(cryptoKey, normalized);
    const updatedAt = Date.parse(normalized.updated_at);
    await uploadItems([
      {
        itemType: "host",
        syncId: normalized.sync_id,
        ciphertext,
        nonce: "",
        updatedAt,
        schemaVersion: 5,
      },
    ]);
    setHosts((cur) => {
      const rest = cur.filter((h) => h.sync_id !== normalized.sync_id);
      return [...rest, normalized].sort((a, b) => (a.label || a.hostname).localeCompare(b.label || b.hostname));
    });
    setDrawerHost(undefined);
    toast.success("Personal host saved.");
  }

  async function deleteHost(host: PersonalHost) {
    if (!cryptoKey) return;
    const deletedAt = Date.now();
    await uploadItems([
      {
        itemType: "host",
        syncId: host.sync_id,
        ciphertext: await encryptPersonalItem(cryptoKey, { ...host, updated_at: new Date(deletedAt).toISOString() }),
        nonce: "",
        updatedAt: deletedAt,
        deletedAt,
        schemaVersion: 5,
      },
    ]);
    setHosts((cur) => cur.filter((h) => h.sync_id !== host.sync_id));
    toast.success("Personal host deleted.");
  }

  async function addGroup() {
    if (!cryptoKey) return;
    const name = window.prompt("Group name");
    if (!name?.trim()) return;
    const group = newGroup(name.trim());
    await uploadItems([
      {
        itemType: "group",
        syncId: group.sync_id,
        ciphertext: await encryptPersonalItem(cryptoKey, group),
        nonce: "",
        updatedAt: Date.parse(group.updated_at),
        schemaVersion: 5,
      },
    ]);
    setGroups((cur) => [...cur, group].sort((a, b) => a.name.localeCompare(b.name)));
  }

  const groupedHosts = useMemo(() => {
    const byGroup = new Map<string, PersonalHost[]>();
    for (const host of hosts) {
      const group = host.group_name?.trim() || "Ungrouped";
      byGroup.set(group, [...(byGroup.get(group) ?? []), host]);
    }
    return Array.from(byGroup.entries());
  }, [hosts]);

  if (loading) {
    return <main className="shell" style={{ padding: "48px 0" }}>Loading personal vault…</main>;
  }

  if (!vault) {
    return <main className="shell" style={{ padding: "48px 0" }}>Personal vault unavailable.</main>;
  }

  if (!cryptoKey) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <PersonalUnlock onUnlock={unlock} error={unlockError} />
      </main>
    );
  }

  return (
    <main className="teams-page">
      <div className="team-bar">
        <div className="team-bar__row">
          <div className="team-bar__switcher">Personal Library</div>
          <div className="team-tabs" role="tablist">
            {(["hosts", "groups", "settings", "activity"] as Tab[]).map((next) => (
              <button
                key={next}
                className={`team-tab ${tab === next ? "team-tab--active" : ""}`}
                type="button"
                onClick={() => {
                  setTab(next);
                  if (next === "activity") void loadEvents();
                }}
              >
                {next}
              </button>
            ))}
          </div>
        </div>
      </div>

      <section className="shell stack" style={{ padding: "28px 0 64px" }}>
        {tab === "hosts" ? (
          <div className="stack">
            <div className="row row--between">
              <div>
                <span className="eyebrow">Encrypted personal hosts</span>
                <h1 className="text-xl fw-800">{hosts.length} hosts</h1>
              </div>
              <button className="btn btn--primary" type="button" onClick={() => setDrawerHost(null)}>
                Add host
              </button>
            </div>
            {groupedHosts.map(([group, entries]) => (
              <div className="block stack" key={group}>
                <span className="eyebrow">{group}</span>
                {entries.map((host) => (
                  <div className="data-row" key={host.sync_id}>
                    <div className="data-row__primary">
                      <span className="data-row__title">{host.label || host.hostname}</span>
                      <span className="muted text-sm">
                        {host.username}@{host.hostname}:{host.port}
                      </span>
                    </div>
                    <div className="row">
                      <button className="btn" type="button" onClick={() => setDrawerHost(host)}>
                        Edit
                      </button>
                      <button className="btn btn--danger" type="button" onClick={() => void deleteHost(host)}>
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : null}

        {tab === "groups" ? (
          <div className="block stack">
            <div className="row row--between">
              <h1 className="text-xl fw-800">Groups</h1>
              <button className="btn btn--primary" type="button" onClick={() => void addGroup()}>
                Add group
              </button>
            </div>
            {groups.map((group) => (
              <div className="data-row" key={group.sync_id}>
                <span className="data-row__title">{group.name}</span>
              </div>
            ))}
          </div>
        ) : null}

        {tab === "settings" ? (
          <div className="block stack">
            <span className="eyebrow">Vault</span>
            <p className="text-sm">Encryption: {vault.encryptionVersion}</p>
            <p className="text-sm">KDF: {vault.kdf.name} · {vault.kdf.iterations.toLocaleString()} iterations</p>
            <p className="muted text-sm">This browser keeps the derived key in memory only. Refreshing requires unlock again.</p>
          </div>
        ) : null}

        {tab === "activity" ? (
          <div className="block stack">
            <h1 className="text-xl fw-800">Activity</h1>
            {events.map((event, idx) => (
              <div className="data-row" key={`${event.createdAt}-${idx}`}>
                <span>
                  {event.source} · {event.action} · {event.itemCount ?? 0} items
                </span>
                <span className="muted text-sm">{new Date(event.createdAt).toLocaleString()}</span>
              </div>
            ))}
          </div>
        ) : null}
      </section>

      {drawerHost !== undefined ? (
        <PersonalHostDrawer
          host={drawerHost}
          groups={groups}
          onClose={() => setDrawerHost(undefined)}
          onSave={(host) => void saveHost(host)}
        />
      ) : null}
    </main>
  );
}
