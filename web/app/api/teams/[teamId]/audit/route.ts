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
    const events = await convexQuery<Array<Record<string, unknown>>>(convexApi.teamAudit.listForTeam, {
      teamId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(events);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_audit_events_failed" },
      { status: 400 },
    );
  }
}
