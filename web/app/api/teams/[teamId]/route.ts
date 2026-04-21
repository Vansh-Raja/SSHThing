import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function PATCH(request: Request, { params }: Params) {
  const { teamId } = await params;
  const body = (await request.json().catch(() => ({}))) as { name?: string };
  if (!body.name?.trim()) {
    return NextResponse.json({ error: "missing_team_name" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const team = await convexMutation<Record<string, unknown>>(convexApi.teams.rename, {
      teamId,
      clerkUserId: actor.clerkUserId,
      name: body.name,
    });
    return NextResponse.json(team);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "rename_team_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teams.remove, {
      teamId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_team_failed" },
      { status: 400 },
    );
  }
}
