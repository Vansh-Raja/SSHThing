import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";
import { hashToken } from "@/lib/tokens";

type Params = {
  params: Promise<{ inviteId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { inviteId } = await params;
  const body = (await request.json().catch(() => ({}))) as { token?: string };

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<Record<string, unknown>>(convexApi.teamInvites.accept, {
      inviteId,
      tokenHash: body.token ? hashToken(body.token) : null,
      clerkUserId: actor.clerkUserId,
      emailLower: actor.email,
      displayName: actor.displayName,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "accept_invite_failed" },
      { status: 400 },
    );
  }
}
