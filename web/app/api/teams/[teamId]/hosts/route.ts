import { NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { decryptTeamSecret, encryptTeamSecret } from "@/lib/teamSecrets";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const hosts = await convexQuery<Array<Record<string, unknown>>>(convexApi.teamHosts.listForTeam, {
      teamId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(hosts);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_team_hosts_failed" },
      { status: 400 },
    );
  }
}

export async function POST(request: Request, { params }: Params) {
  const { teamId } = await params;
  const body = (await request.json().catch(() => ({}))) as {
    label?: string;
    hostname?: string;
    username?: string;
    port?: number;
    group?: string;
    tags?: string[];
    notes?: string;
    credentialMode?: string;
    credentialType?: string;
    secretVisibility?: string;
    sharedCredential?: string;
  };

  if (!body.hostname?.trim()) {
    return NextResponse.json({ error: "missing_hostname" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const host = await convexMutation<Record<string, unknown>>(convexApi.teamHosts.create, {
      teamId,
      clerkUserId: actor.clerkUserId,
      label: body.label?.trim() || body.hostname.trim(),
      hostname: body.hostname.trim(),
      username: body.username?.trim() || "",
      port: body.port ?? 22,
      group: body.group?.trim() || "",
      tags: Array.isArray(body.tags) ? body.tags : [],
      notes: body.notes?.trim() || "",
      credentialMode: body.credentialMode ?? "shared",
      credentialType: body.credentialType ?? "none",
      secretVisibility: body.secretVisibility ?? "revealed_to_access_holders",
      sharedCredentialCiphertext:
        body.sharedCredential && body.credentialMode !== "per_member"
          ? encryptTeamSecret(body.sharedCredential)
          : undefined,
    });
    return NextResponse.json(host);
  } catch (error) {
    const message = error instanceof Error ? error.message : "create_host_failed";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
