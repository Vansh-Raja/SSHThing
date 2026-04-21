import { auth, clerkClient, currentUser } from "@clerk/nextjs/server";

import { convexApi, convexMutation, convexQuery } from "./convex";
import { getBrowserBaseURL } from "./env";
import { createAccessToken, createDeviceCode, createPollSecret, createRefreshToken, hashToken } from "./tokens";

type TuiSessionRecord = {
  _id: string;
  clerkUserId: string;
  deviceName: string;
  accessExpiresAt: number;
  refreshExpiresAt: number;
  revokedAt?: number | null;
};

export type BrowserIdentity = {
  userId: string;
  email: string;
  displayName: string;
};

export type RequestActor = {
  clerkUserId: string;
  email: string;
  displayName: string;
  source: "browser" | "bearer";
};

export async function requireBrowserIdentity(): Promise<BrowserIdentity> {
  const { isAuthenticated, userId } = await auth();
  if (!isAuthenticated || !userId) {
    throw new Error("not_authenticated");
  }

  const user = await currentUser();
  if (!user) {
    throw new Error("missing_user");
  }

  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";

  return {
    userId,
    email: primaryEmail.toLowerCase(),
    displayName: [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || userId,
  };
}

async function getClerkUserSummary(clerkUserId: string): Promise<BrowserIdentity> {
  const client = await clerkClient();
  const user = await client.users.getUser(clerkUserId);
  const primaryEmail = user.emailAddresses.find((email) => email.id === user.primaryEmailAddressId)?.emailAddress
    ?? user.emailAddresses[0]?.emailAddress
    ?? "";

  return {
    userId: user.id,
    email: primaryEmail.toLowerCase(),
    displayName: [user.firstName, user.lastName].filter(Boolean).join(" ") || user.username || primaryEmail || user.id,
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

  const user = await getClerkUserSummary(record.clerkUserId);
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
      name: user.displayName,
      email: user.email,
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
  return { session };
}

export async function getActorFromRequest(authHeader: string | null): Promise<RequestActor> {
  const trimmed = authHeader?.trim() ?? "";
  if (trimmed.startsWith("Bearer ")) {
    const session = await getTuiSessionFromBearer(authHeader);
    const user = await getClerkUserSummary(session.clerkUserId);
    return {
      clerkUserId: user.userId,
      email: user.email,
      displayName: user.displayName,
      source: "bearer",
    };
  }

  const identity = await requireBrowserIdentity();
  return {
    clerkUserId: identity.userId,
    email: identity.email,
    displayName: identity.displayName,
    source: "browser",
  };
}

export function buildInviteLink(inviteId: string, token: string): string {
  const inviteUrl = new URL(`/teams/invites/${inviteId}`, getBrowserBaseURL());
  inviteUrl.searchParams.set("token", token);
  return inviteUrl.toString();
}
