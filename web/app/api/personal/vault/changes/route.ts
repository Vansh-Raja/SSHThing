import { NextRequest, NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

function parseRevision(value: string | null): number | undefined {
  if (!value) return undefined;
  const n = Number(value);
  return Number.isFinite(n) && n >= 0 ? n : undefined;
}

export async function GET(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const { searchParams } = new URL(request.url);
    const result = await convexQuery(convexApi.personalVaults.listChanges, {
      clerkUserId: actor.clerkUserId,
      sinceRevision: parseRevision(searchParams.get("sinceRevision")),
    });
    return NextResponse.json(result);
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_changes_failed" },
      { status: 400 },
    );
  }
}
