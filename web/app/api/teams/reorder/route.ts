import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { teamIds?: string[] };
  if (!Array.isArray(body.teamIds)) {
    return NextResponse.json({ error: "missing_team_ids" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teams.reorder, {
      clerkUserId: actor.clerkUserId,
      teamIds: body.teamIds,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "reorder_teams_failed" },
      { status: 400 },
    );
  }
}
