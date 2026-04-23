import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ hostId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { hostId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const roster = await convexQuery<Array<Record<string, unknown>>>(convexApi.teamHosts.listCredentialRoster, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(roster);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_credential_roster_failed" },
      { status: 400 },
    );
  }
}
