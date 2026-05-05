import { mutation, query } from "./_generated/server";
import { v } from "convex/values";

function revisionFromTime(updatedAt: number): string {
  return String(Math.trunc(updatedAt));
}

function latestRevision(vault: { latestRevision?: number; updatedAt: number }): number {
  return vault.latestRevision ?? Math.trunc(vault.updatedAt);
}

function randomSaltHex(): string {
  const alphabet = "0123456789abcdef";
  let out = "";
  for (let i = 0; i < 32; i += 1) {
    out += alphabet[Math.floor(Math.random() * alphabet.length)];
  }
  return out;
}

async function getVaultByUser(ctx: any, clerkUserId: string) {
  return await ctx.db
    .query("personalVaults")
    .withIndex("by_user", (q: any) => q.eq("clerkUserId", clerkUserId))
    .first();
}

export const getOrCreateVault = mutation({
  args: {
    clerkUserId: v.string(),
    name: v.optional(v.string()),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    const existing = await getVaultByUser(ctx, args.clerkUserId);
    if (existing) {
      return existing;
    }
    const vaultId = await ctx.db.insert("personalVaults", {
      clerkUserId: args.clerkUserId,
      name: args.name ?? "Personal Library",
      status: "active",
      schemaVersion: 1,
      encryptionVersion: "aes-gcm-pbkdf2-v1",
      kdf: {
        name: "PBKDF2-SHA256",
        iterations: 100000,
        salt: randomSaltHex(),
      },
      createdAt: now,
      updatedAt: now,
      latestRevision: 0,
    });
    const vault = await ctx.db.get(vaultId);
    if (!vault) throw new Error("vault_create_failed");
    return vault;
  },
});

export const getVaultSummary = query({
  args: {
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) return null;
    return {
      vaultId: vault._id,
      schemaVersion: vault.schemaVersion,
      encryptionVersion: vault.encryptionVersion,
      kdf: vault.kdf,
      updatedAt: vault.updatedAt,
      latestRevision: latestRevision(vault),
    };
  },
});

export const listItems = query({
  args: {
    clerkUserId: v.string(),
    since: v.optional(v.number()),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) {
      return { revision: "0", items: [] };
    }
    const since = args.since ?? 0;
    const items = await ctx.db
      .query("personalVaultItems")
      .withIndex("by_vault_and_updated_at", (q) => q.eq("vaultId", vault._id).gte("updatedAt", since))
      .collect();
    return {
      revision: String(latestRevision(vault)),
      items: items.map((item) => ({
        itemType: item.itemType,
        syncId: item.syncId,
        ciphertext: item.ciphertext,
        nonce: item.nonce,
        updatedAt: item.updatedAt,
        deletedAt: item.deletedAt ?? null,
        schemaVersion: item.schemaVersion,
        itemRevision: item.itemRevision ?? Math.trunc(item.updatedAt),
      })),
    };
  },
});

export const listChanges = query({
  args: {
    clerkUserId: v.string(),
    sinceRevision: v.optional(v.number()),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) {
      return { revision: "0", latestRevision: 0, items: [] };
    }
    const sinceRevision = args.sinceRevision ?? 0;
    const items = sinceRevision <= 0
      ? await ctx.db
          .query("personalVaultItems")
          .withIndex("by_vault", (q) => q.eq("vaultId", vault._id))
          .collect()
      : await ctx.db
          .query("personalVaultItems")
          .withIndex("by_vault_and_item_revision", (q) => q.eq("vaultId", vault._id).gt("itemRevision", sinceRevision))
          .collect();
    return {
      revision: String(latestRevision(vault)),
      latestRevision: latestRevision(vault),
      items: items.map((item) => ({
        itemType: item.itemType,
        syncId: item.syncId,
        ciphertext: item.ciphertext,
        nonce: item.nonce,
        updatedAt: item.updatedAt,
        deletedAt: item.deletedAt ?? null,
        schemaVersion: item.schemaVersion,
        itemRevision: item.itemRevision ?? Math.trunc(item.updatedAt),
      })),
    };
  },
});

const vaultItemValidator = v.object({
  itemType: v.string(),
  syncId: v.string(),
  ciphertext: v.string(),
  nonce: v.string(),
  updatedAt: v.number(),
  deletedAt: v.optional(v.union(v.number(), v.null())),
  schemaVersion: v.number(),
  baseRevision: v.optional(v.union(v.string(), v.number(), v.null())),
});

export const upsertItems = mutation({
  args: {
    clerkUserId: v.string(),
    deviceId: v.string(),
    force: v.optional(v.boolean()),
    items: v.array(vaultItemValidator),
  },
  handler: async (ctx, args) => {
    const now = Date.now();
    let vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) {
      const vaultId = await ctx.db.insert("personalVaults", {
        clerkUserId: args.clerkUserId,
        name: "Personal Library",
        status: "active",
        schemaVersion: 1,
        encryptionVersion: "aes-gcm-pbkdf2-v1",
        kdf: { name: "PBKDF2-SHA256", iterations: 100000, salt: randomSaltHex() },
        createdAt: now,
        updatedAt: now,
        latestRevision: 0,
      });
      vault = await ctx.db.get(vaultId);
      if (!vault) throw new Error("vault_create_failed");
    }

    const conflicts = [];
    let nextRevision = latestRevision(vault);
    let accepted = 0;
    for (const item of args.items) {
      const existing = await ctx.db
        .query("personalVaultItems")
        .withIndex("by_vault_and_sync_id", (q) => q.eq("vaultId", vault._id).eq("syncId", item.syncId))
        .first();
      const existingRevision = existing?.itemRevision ?? (existing ? Math.trunc(existing.updatedAt) : 0);
      const rawBaseRevision = item.baseRevision ?? null;
      const baseRevision = rawBaseRevision === null || rawBaseRevision === "" ? null : Number(rawBaseRevision);
      if (existing && baseRevision !== null && Number.isFinite(baseRevision) && existingRevision > baseRevision && !args.force) {
        conflicts.push({
          itemType: item.itemType,
          syncId: item.syncId,
          remoteAt: existing.updatedAt,
          localAt: item.updatedAt,
          remoteRevision: existingRevision,
          localRevision: baseRevision,
        });
        continue;
      }
      if (existing && existing.updatedAt > item.updatedAt && baseRevision === null && !args.force) {
        conflicts.push({
          itemType: item.itemType,
          syncId: item.syncId,
          remoteAt: existing.updatedAt,
          localAt: item.updatedAt,
          remoteRevision: existingRevision,
        });
        continue;
      }
      nextRevision += 1;
      const patch = {
        itemType: item.itemType,
        ciphertext: item.ciphertext,
        nonce: item.nonce,
        updatedAt: item.updatedAt,
        deletedAt: item.deletedAt ?? undefined,
        schemaVersion: item.schemaVersion,
        itemRevision: nextRevision,
        updatedByDevice: args.deviceId.trim() || undefined,
      };
      if (existing) {
        await ctx.db.patch(existing._id, patch);
      } else {
        await ctx.db.insert("personalVaultItems", {
          vaultId: vault._id,
          clerkUserId: args.clerkUserId,
          syncId: item.syncId,
          ...patch,
        });
      }
      accepted += 1;
    }

    const updatedAt = Date.now();
    await ctx.db.patch(vault._id, { updatedAt, latestRevision: nextRevision });
    if (args.deviceId.trim()) {
      const existingDevice = await ctx.db
        .query("personalVaultDevices")
        .withIndex("by_vault_and_device", (q) => q.eq("vaultId", vault._id).eq("deviceId", args.deviceId))
        .first();
      const devicePatch = {
        deviceName: args.deviceId,
        lastSyncAt: updatedAt,
        lastPushedRevision: nextRevision,
        lastSeenAt: updatedAt,
      };
      if (existingDevice) {
        await ctx.db.patch(existingDevice._id, devicePatch);
      } else {
        await ctx.db.insert("personalVaultDevices", {
          vaultId: vault._id,
          clerkUserId: args.clerkUserId,
          deviceId: args.deviceId,
          createdAt: now,
          ...devicePatch,
        });
      }
    }

    return { ok: conflicts.length === 0, revision: String(nextRevision), latestRevision: nextRevision, accepted, conflicts };
  },
});

export const markDeviceSeen = mutation({
  args: {
    clerkUserId: v.string(),
    deviceId: v.string(),
    deviceName: v.optional(v.string()),
    lastPulledRevision: v.optional(v.number()),
    lastPushedRevision: v.optional(v.number()),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault || !args.deviceId.trim()) return { ok: true };
    const now = Date.now();
    const existingDevice = await ctx.db
      .query("personalVaultDevices")
      .withIndex("by_vault_and_device", (q) => q.eq("vaultId", vault._id).eq("deviceId", args.deviceId))
      .first();
    const patch = {
      deviceName: args.deviceName ?? args.deviceId,
      lastSyncAt: now,
      lastPulledRevision: args.lastPulledRevision,
      lastPushedRevision: args.lastPushedRevision,
      lastSeenAt: now,
    };
    if (existingDevice) {
      await ctx.db.patch(existingDevice._id, patch);
    } else {
      await ctx.db.insert("personalVaultDevices", {
        vaultId: vault._id,
        clerkUserId: args.clerkUserId,
        deviceId: args.deviceId,
        createdAt: now,
        ...patch,
      });
    }
    return { ok: true };
  },
});

export const recordSyncEvent = mutation({
  args: {
    clerkUserId: v.string(),
    deviceId: v.optional(v.string()),
    source: v.string(),
    action: v.string(),
    itemType: v.optional(v.string()),
    itemCount: v.optional(v.number()),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) return { ok: true };
    await ctx.db.insert("personalVaultSyncEvents", {
      vaultId: vault._id,
      clerkUserId: args.clerkUserId,
      deviceId: args.deviceId,
      source: args.source,
      action: args.action,
      itemType: args.itemType,
      itemCount: args.itemCount,
      createdAt: Date.now(),
    });
    return { ok: true };
  },
});

export const listEvents = query({
  args: {
    clerkUserId: v.string(),
  },
  handler: async (ctx, args) => {
    const vault = await getVaultByUser(ctx, args.clerkUserId);
    if (!vault) return [];
    const events = await ctx.db
      .query("personalVaultSyncEvents")
      .withIndex("by_vault_and_created_at", (q) => q.eq("vaultId", vault._id))
      .order("desc")
      .take(50);
    return events.map((event) => ({
      source: event.source,
      action: event.action,
      itemType: event.itemType ?? null,
      itemCount: event.itemCount ?? null,
      createdAt: event.createdAt,
    }));
  },
});
