import { NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { createTeamAutomationTokenMaterial } from "@/lib/teamAutomationTokens";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const tokens = await convexQuery<Array<Record<string, unknown>>>(
      convexApi.teamAutomationTokens.listForTeam,
      {
        teamId,
        clerkUserId: actor.clerkUserId,
      },
    );
    return NextResponse.json(tokens);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_team_tokens_failed" },
      { status: 400 },
    );
  }
}

export async function POST(request: Request, { params }: Params) {
  const { teamId } = await params;
  const body = (await request.json().catch(() => ({}))) as {
    name?: string;
    hostIds?: string[];
    expiresAt?: number | null;
    maxUses?: number | null;
  };

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const material = createTeamAutomationTokenMaterial();
    const created = await convexMutation<Record<string, unknown>>(
      convexApi.teamAutomationTokens.create,
      {
        teamId,
        clerkUserId: actor.clerkUserId,
        actorDisplayName: actor.displayName,
        name: body.name?.trim() || "team automation token",
        tokenId: material.tokenId,
        tokenHash: material.tokenHash,
        hostIds: Array.isArray(body.hostIds) ? body.hostIds : [],
        expiresAt: body.expiresAt ?? null,
        maxUses: body.maxUses ?? null,
      },
    );
    return NextResponse.json({
      ...created,
      rawToken: material.rawToken,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "create_team_token_failed" },
      { status: 400 },
    );
  }
}
