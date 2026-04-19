import { auth, clerkClient, currentUser } from "@clerk/nextjs/server";

import { convexApi, convexMutation, convexQuery } from "./convex";
import { createAccessToken, createDeviceCode, createPollSecret, createRefreshToken, hashToken } from "./tokens";
import { getBrowserBaseURL } from "./env";

type ClerkOrg = {
  id: string;
  name: string;
  slug: string | null;
};

type TuiSessionRecord = {
  _id: string;
  clerkUserId: string;
  workspaceId?: string | null;
  teamId?: string | null;
  deviceName: string;
  accessExpiresAt: number;
  refreshExpiresAt: number;
  revokedAt?: number | null;
};

function normalizeSlug(name: string, fallback: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
  return slug || fallback.toLowerCase();
}

function personalWorkspaceName(displayName: string, email: string, userId: string): string {
  if (displayName) {
    return `${displayName}'s Workspace`;
  }
  if (email) {
    return `${email}'s Workspace`;
  }
  return `Workspace ${userId.slice(0, 8)}`;
}

export async function requireBrowserIdentity() {
  const { isAuthenticated, userId, orgId, orgRole } = await auth();
  if (!isAuthenticated || !userId) {
    throw new Error("not_authenticated");
  }

  const client = await clerkClient();
  const user = await currentUser();
  if (!user) {
    throw new Error("missing_user");
  }

  let organization: ClerkOrg | null = null;
  if (orgId) {
    const org = await client.organizations.getOrganization({ organizationId: orgId });
    organization = {
      id: org.id,
      name: org.name,
      slug: org.slug ?? null,
    };
  }

  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";

  return {
    userId,
    orgId,
    orgRole: orgRole ?? null,
    organization,
    email: primaryEmail,
    displayName: [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || userId,
  };
}

export async function buildCliAuthStart(deviceName: string) {
  const started = await convexMutation<{ sessionId: string; deviceCode: string; pollSecret: string; expiresAt: number }>(
    convexApi.sessions.startCliAuth,
    {
      deviceName,
      deviceCode: createDeviceCode(),
      pollSecret: createPollSecret(),
      ttlSeconds: 600,
    },
  );

  const authUrl = new URL("/cli-auth/complete", getBrowserBaseURL());
  authUrl.searchParams.set("session", started.sessionId);
  authUrl.searchParams.set("code", started.deviceCode);

  return {
    authUrl: authUrl.toString(),
    deviceCode: started.deviceCode,
    sessionId: started.sessionId,
    pollSecret: started.pollSecret,
    pollIntervalSeconds: 2,
    expiresAt: started.expiresAt,
  };
}

export async function completeCliAuth(sessionId: string, deviceCode?: string | null) {
  const identity = await requireBrowserIdentity();

  return convexMutation<{ ok: boolean }>(convexApi.sessions.completeCliAuth, {
    sessionId,
    deviceCode: deviceCode ?? "",
    clerkUserId: identity.userId,
  });
}

export async function pollCliAuth(sessionId: string, pollSecret: string) {
  const record = await convexQuery<{
    status: string;
    clerkUserId?: string | null;
    deviceName?: string | null;
    expiresAt: number;
    completedAt?: number | null;
  }>(convexApi.sessions.getCliAuthStatus, {
    sessionId,
    pollSecret,
  });

  if (record.status !== "completed" || !record.clerkUserId) {
    return {
      status: record.status,
      expiresAt: record.expiresAt,
    };
  }

  const client = await clerkClient();
  const user = await client.users.getUser(record.clerkUserId);
  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";
  const displayName = [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || user.id;

  const accessToken = createAccessToken();
  const refreshToken = createRefreshToken();
  const session = await convexMutation<{
    sessionId: string;
    accessExpiresAt: number;
    refreshExpiresAt: number;
  }>(convexApi.sessions.createTuiSession, {
    clerkUserId: record.clerkUserId,
    deviceName: record.deviceName ?? "SSHThing TUI",
    accessTokenHash: hashToken(accessToken),
    refreshTokenHash: hashToken(refreshToken),
    accessTtlSeconds: 900,
    refreshTtlSeconds: 86400 * 30,
  });

  return {
    status: "completed",
    accessToken,
    refreshToken,
    expiresAt: session.accessExpiresAt,
    user: {
      id: record.clerkUserId,
      name: displayName,
      email: primaryEmail,
    },
  };
}

export async function refreshTuiAccess(refreshToken: string) {
  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByRefreshHash, {
    refreshTokenHash: hashToken(refreshToken),
  });

  if (!record || record.revokedAt || record.refreshExpiresAt <= Date.now()) {
    throw new Error("invalid_refresh_token");
  }

  const accessToken = createAccessToken();
  const updated = await convexMutation<{ accessExpiresAt: number }>(convexApi.sessions.rotateAccessToken, {
    sessionId: record._id,
    accessTokenHash: hashToken(accessToken),
    accessTtlSeconds: 900,
  });

  return {
    accessToken,
    expiresAt: updated.accessExpiresAt,
  };
}

export async function revokeTuiSession(refreshToken: string) {
  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByRefreshHash, {
    refreshTokenHash: hashToken(refreshToken),
  });
  if (!record) {
    return { ok: true };
  }
  await convexMutation<{ ok: boolean }>(convexApi.sessions.revokeTuiSession, {
    sessionId: record._id,
  });
  return { ok: true };
}

export async function getTuiSessionFromBearer(authHeader: string | null) {
  const value = authHeader?.trim() ?? "";
  if (!value.startsWith("Bearer ")) {
    throw new Error("missing_bearer_token");
  }
  const accessToken = value.slice("Bearer ".length).trim();
  if (!accessToken) {
    throw new Error("missing_bearer_token");
  }

  const record = await convexQuery<TuiSessionRecord | null>(convexApi.sessions.getTuiSessionByAccessHash, {
    accessTokenHash: hashToken(accessToken),
  });
  if (!record || record.revokedAt || record.accessExpiresAt <= Date.now()) {
    throw new Error("invalid_access_token");
  }

  await convexMutation<{ ok: boolean }>(convexApi.sessions.markTuiSessionSeen, {
    sessionId: record._id,
  });

  return record;
}

export async function getSessionContextFromBearer(authHeader: string | null) {
  const session = await getTuiSessionFromBearer(authHeader);
  return {
    session,
  };
}

export async function ensureWorkspaceForCurrentOrg() {
  const identity = await requireBrowserIdentity();

  const organizationId = identity.organization?.id ?? `personal:${identity.userId}`;
  const organizationName =
    identity.organization?.name ?? personalWorkspaceName(identity.displayName, identity.email, identity.userId);
  const organizationSlug =
    identity.organization?.slug ?? normalizeSlug(identity.displayName || identity.email || identity.userId, identity.userId);
  const clerkRole = identity.organization ? (identity.orgRole ?? "org:member") : "owner";

  return convexMutation<{ workspaceId: string; defaultVaultId: string }>(
    convexApi.workspaces.bootstrapForClerkOrganization,
    {
      clerkOrganizationId: organizationId,
      organizationName,
      organizationSlug,
      clerkUserId: identity.userId,
      userEmail: identity.email,
      displayName: identity.displayName,
      clerkRole,
    },
  );
}

export async function getWorkspaceContextFromBearer(authHeader: string | null) {
  const { session } = await getSessionContextFromBearer(authHeader);
  if (!session.workspaceId) {
    throw new Error("workspace_context_unavailable");
  }
  const workspace = await convexQuery<{
    id: string;
    name: string;
    slug: string;
    clerkOrganizationId: string;
  }>(convexApi.workspaces.getWorkspaceSummary, {
    workspaceId: session.workspaceId,
  });
  return {
    session,
    workspace,
  };
}

export async function listTeamsFromBearer(authHeader: string | null) {
  const { session } = await getSessionContextFromBearer(authHeader);
  return convexQuery<Array<{
    id: string;
    name: string;
    slug: string;
    displayOrder: number;
  }>>(convexApi.teams.listForUser, {
    clerkUserId: session.clerkUserId,
  });
}

export async function createTeamFromBearer(authHeader: string | null, name: string) {
  const { session } = await getSessionContextFromBearer(authHeader);
  const client = await clerkClient();
  const user = await client.users.getUser(session.clerkUserId);
  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";
  const displayName = [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || user.id;

  return convexMutation<{
    id: string;
    name: string;
    slug: string;
    displayOrder: number;
  }>(convexApi.teams.create, {
    clerkUserId: session.clerkUserId,
    userEmail: primaryEmail,
    displayName,
    name,
  });
}

export async function renameTeamFromBearer(authHeader: string | null, teamId: string, name: string) {
  const { session } = await getSessionContextFromBearer(authHeader);
  return convexMutation<{
    id: string;
    name: string;
    slug: string;
    displayOrder: number;
  }>(convexApi.teams.rename, {
    teamId,
    clerkUserId: session.clerkUserId,
    name,
  });
}

export async function deleteTeamFromBearer(authHeader: string | null, teamId: string) {
  const { session } = await getSessionContextFromBearer(authHeader);
  return convexMutation<{ ok: boolean }>(convexApi.teams.remove, {
    teamId,
    clerkUserId: session.clerkUserId,
  });
}

export async function reorderTeamsFromBearer(authHeader: string | null, teamIds: string[]) {
  const { session } = await getSessionContextFromBearer(authHeader);
  return convexMutation<{ ok: boolean }>(convexApi.teams.reorder, {
    clerkUserId: session.clerkUserId,
    teamIds,
  });
}

export async function listTeamHostsFromBearer(authHeader: string | null, teamId: string) {
  const { session } = await getSessionContextFromBearer(authHeader);
  return convexQuery<Array<{
    id: string;
    teamId: string;
    label: string;
    hostname: string;
    username: string;
    port: number;
    group: string;
    tags: string[];
    authMode: string;
    lastConnectedAt: number | null;
  }>>(convexApi.teams.listHosts, {
    teamId,
    clerkUserId: session.clerkUserId,
  });
}
