import { NextRequest, NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function GET(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    await convexMutation(convexApi.personalVaults.getOrCreateVault, {
      clerkUserId: actor.clerkUserId,
      name: "Personal Library",
    });
    const [vault, items] = await Promise.all([
      convexQuery(convexApi.personalVaults.getVaultSummary, {
        clerkUserId: actor.clerkUserId,
      }),
      convexQuery<{ revision: string; items: unknown[] }>(convexApi.personalVaults.listItems, {
        clerkUserId: actor.clerkUserId,
      }),
    ]);
    return NextResponse.json({ vault, ...items });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_sync_failed" },
      { status: 400 },
    );
  }
}
