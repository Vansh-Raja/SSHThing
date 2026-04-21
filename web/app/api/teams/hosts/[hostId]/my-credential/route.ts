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
    const credential = await convexQuery<Record<string, unknown>>(convexApi.teamHosts.getMyCredential, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });

    return NextResponse.json({
      ...credential,
      secret:
        typeof credential.ciphertext === "string" && credential.ciphertext
          ? decryptTeamSecret(credential.ciphertext)
          : "",
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "get_credential_failed" },
      { status: 400 },
    );
  }
}

export async function PUT(request: Request, { params }: Params) {
  const { hostId } = await params;
  const body = (await request.json().catch(() => ({}))) as {
    username?: string;
    credentialType?: string;
    secret?: string;
  };
  if (!body.secret?.trim()) {
    return NextResponse.json({ error: "missing_secret" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamHosts.upsertMyCredential, {
      hostId,
      clerkUserId: actor.clerkUserId,
      username: body.username?.trim() || undefined,
      credentialType: body.credentialType ?? "password",
      ciphertext: encryptTeamSecret(body.secret),
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "save_credential_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { hostId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamHosts.deleteMyCredential, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_credential_failed" },
      { status: 400 },
    );
  }
}
