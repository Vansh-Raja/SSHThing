import { defineSchema, defineTable } from "convex/server";
import { v } from "convex/values";

export default defineSchema({
  teams: defineTable({
    ownerClerkUserId: v.string(),
    name: v.string(),
    slug: v.string(),
    displayOrder: v.number(),
    status: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_owner_and_display_order", ["ownerClerkUserId", "displayOrder"])
    .index("by_owner_and_slug", ["ownerClerkUserId", "slug"]),

  teamMembers: defineTable({
    teamId: v.id("teams"),
    clerkUserId: v.string(),
    email: v.string(),
    displayName: v.string(),
    role: v.string(),
    status: v.string(),
    joinedAt: v.optional(v.number()),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_team", ["teamId"])
    .index("by_team_and_user", ["teamId", "clerkUserId"])
    .index("by_user", ["clerkUserId"]),

  teamHosts: defineTable({
    teamId: v.id("teams"),
    label: v.string(),
    hostname: v.string(),
    username: v.string(),
    port: v.number(),
    group: v.string(),
    tags: v.array(v.string()),
    notes: v.optional(v.string()),
    authMode: v.optional(v.string()),
    credentialMode: v.string(),
    credentialType: v.string(),
    secretVisibility: v.string(),
    createdByClerkUserId: v.string(),
    updatedByClerkUserId: v.string(),
    lastConnectedAt: v.optional(v.number()),
    createdAt: v.number(),
    updatedAt: v.number(),
  }).index("by_team", ["teamId"]),

  teamInvites: defineTable({
    teamId: v.id("teams"),
    emailLower: v.string(),
    role: v.string(),
    invitedByClerkUserId: v.string(),
    status: v.string(),
    tokenHash: v.string(),
    tokenCiphertext: v.string(),
    expiresAt: v.number(),
    acceptedAt: v.optional(v.number()),
    acceptedByClerkUserId: v.optional(v.string()),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_team", ["teamId"])
    .index("by_email_lower_and_status", ["emailLower", "status"])
    .index("by_token_hash", ["tokenHash"])
    .index("by_invited_by_and_status", ["invitedByClerkUserId", "status"]),

  teamHostSharedCredentials: defineTable({
    hostId: v.id("teamHosts"),
    credentialType: v.string(),
    ciphertext: v.string(),
    updatedByClerkUserId: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  }).index("by_host", ["hostId"]),

  teamHostPersonalCredentials: defineTable({
    hostId: v.id("teamHosts"),
    clerkUserId: v.string(),
    username: v.optional(v.string()),
    credentialType: v.string(),
    ciphertext: v.string(),
    createdAt: v.number(),
    updatedAt: v.number(),
  })
    .index("by_host_and_user", ["hostId", "clerkUserId"])
    .index("by_user", ["clerkUserId"]),

  teamAuditEvents: defineTable({
    teamId: v.id("teams"),
    actorClerkUserId: v.string(),
    actorDisplayName: v.string(),
    entityType: v.string(),
    entityId: v.string(),
    eventType: v.string(),
    targetClerkUserId: v.optional(v.string()),
    targetDisplayName: v.optional(v.string()),
    summary: v.string(),
    metadata: v.optional(
      v.object({
        hostLabel: v.optional(v.string()),
        credentialMode: v.optional(v.string()),
        credentialType: v.optional(v.string()),
      }),
    ),
    createdAt: v.number(),
  })
    .index("by_team_and_created_at", ["teamId", "createdAt"])
    .index("by_entity_and_created_at", ["entityId", "createdAt"]),

  cliAuthSessions: defineTable({
    deviceName: v.string(),
    deviceCode: v.string(),
    pollSecret: v.string(),
    status: v.string(),
    requestedAt: v.number(),
    completedAt: v.optional(v.number()),
    clerkUserId: v.optional(v.string()),
    expiresAt: v.number(),
  }).index("by_device_code", ["deviceCode"]),

  tuiSessions: defineTable({
    clerkUserId: v.string(),
    accessTokenHash: v.string(),
    refreshTokenHash: v.string(),
    deviceName: v.string(),
    accessExpiresAt: v.number(),
    refreshExpiresAt: v.number(),
    lastSeenAt: v.number(),
    revokedAt: v.optional(v.number()),
    createdAt: v.number(),
  })
    .index("by_access_hash", ["accessTokenHash"])
    .index("by_refresh_hash", ["refreshTokenHash"]),
});
