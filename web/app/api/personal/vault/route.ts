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
    const summary = await convexQuery(convexApi.personalVaults.getVaultSummary, {
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(summary);
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_vault_failed" },
      { status: 400 },
    );
  }
}
