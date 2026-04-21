import { NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { decryptTeamSecret, encryptTeamSecret } from "@/lib/teamSecrets";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ hostId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { hostId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const host = await convexQuery<Record<string, unknown>>(convexApi.teamHosts.getHost, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });
    const sharedCredential = typeof host.sharedCredential === "string" && host.sharedCredential
      ? decryptTeamSecret(host.sharedCredential)
      : null;
    return NextResponse.json({
      ...host,
      sharedCredential,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "get_host_failed" },
      { status: 400 },
    );
  }
}

export async function PATCH(request: Request, { params }: Params) {
  const { hostId } = await params;
  const body = (await request.json().catch(() => ({}))) as {
    label?: string;
    hostname?: string;
    username?: string;
    port?: number;
    group?: string;
    tags?: string[];
    credentialMode?: string;
    credentialType?: string;
    secretVisibility?: string;
    sharedCredential?: string | null;
  };

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<Record<string, unknown>>(convexApi.teamHosts.update, {
      hostId,
      clerkUserId: actor.clerkUserId,
      label: body.label?.trim() || body.hostname?.trim() || "",
      hostname: body.hostname?.trim() || "",
      username: body.username?.trim() || "",
      port: body.port ?? 22,
      group: body.group?.trim() || "",
      tags: Array.isArray(body.tags) ? body.tags : [],
      credentialMode: body.credentialMode ?? "shared",
      credentialType: body.credentialType ?? "none",
      secretVisibility: body.secretVisibility ?? "revealed_to_access_holders",
      sharedCredentialCiphertext:
        body.sharedCredential === null
          ? null
          : body.sharedCredential
            ? encryptTeamSecret(body.sharedCredential)
            : undefined,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "update_host_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { hostId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamHosts.remove, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_host_failed" },
      { status: 400 },
    );
  }
}
