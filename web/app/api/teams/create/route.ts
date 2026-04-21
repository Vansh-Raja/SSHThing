import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { name?: string };
  if (!body.name?.trim()) {
    return NextResponse.json({ error: "missing_team_name" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const team = await convexMutation<Record<string, unknown>>(convexApi.teams.create, {
      clerkUserId: actor.clerkUserId,
      userEmail: actor.email,
      displayName: actor.displayName,
      name: body.name,
    });
    return NextResponse.json(team);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "create_team_failed" },
      { status: 400 },
    );
  }
}
