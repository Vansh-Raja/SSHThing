import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const teams = await convexQuery<Array<Record<string, unknown>>>(convexApi.teams.listForUser, {
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(teams);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_teams_failed" },
      { status: 401 },
    );
  }
}
