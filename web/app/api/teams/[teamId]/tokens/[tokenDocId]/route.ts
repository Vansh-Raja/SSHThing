import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string; tokenDocId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { teamId, tokenDocId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(
      convexApi.teamAutomationTokens.revoke,
      {
        teamId,
        tokenDocId,
        clerkUserId: actor.clerkUserId,
        actorDisplayName: actor.displayName,
      },
    );
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "revoke_team_token_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { teamId, tokenDocId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(
      convexApi.teamAutomationTokens.deleteRevoked,
      {
        teamId,
        tokenDocId,
        clerkUserId: actor.clerkUserId,
      },
    );
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_team_token_failed" },
      { status: 400 },
    );
  }
}
