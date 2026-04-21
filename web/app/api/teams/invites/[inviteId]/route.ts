import { NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";
import { hashToken } from "@/lib/tokens";

type Params = {
  params: Promise<{ inviteId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { inviteId } = await params;
  const token = new URL(request.url).searchParams.get("token");

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const invite = await convexQuery<Record<string, unknown> | null>(convexApi.teamInvites.getForToken, {
      inviteId,
      tokenHash: token ? hashToken(token) : null,
      clerkUserId: actor.clerkUserId,
      emailLower: actor.email,
    });
    if (!invite) {
      return NextResponse.json({ error: "invite_not_found" }, { status: 404 });
    }
    return NextResponse.json(invite);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "get_invite_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { inviteId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamInvites.revoke, {
      inviteId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "revoke_invite_failed" },
      { status: 400 },
    );
  }
}
