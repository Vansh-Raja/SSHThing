import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const members = await convexQuery<Array<Record<string, unknown>>>(convexApi.teamMembers.listForTeam, {
      teamId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(members);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_members_failed" },
      { status: 400 },
    );
  }
}
