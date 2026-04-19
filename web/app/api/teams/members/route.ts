import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getWorkspaceContextFromBearer } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const { session, workspace } = await getWorkspaceContextFromBearer(request.headers.get("authorization"));
    const url = new URL(request.url);
    const vaultId = url.searchParams.get("vaultId");
    if (!vaultId) {
      return NextResponse.json({ error: "missing_vault_id" }, { status: 400 });
    }

    const access = await convexQuery<{ vaultRole: string; workspaceRole: string } | null>(
      convexApi.memberships.getAccessContext,
      {
        workspaceId: workspace.id,
        vaultId,
        clerkUserId: session.clerkUserId,
      },
    );
    if (!access) {
      return NextResponse.json({ error: "forbidden" }, { status: 403 });
    }

    const members = await convexQuery<Array<Record<string, unknown>>>(convexApi.memberships.listMembersForVault, {
      workspaceId: workspace.id,
      vaultId,
    });
    return NextResponse.json(members);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "unauthorized" },
      { status: 401 },
    );
  }
}
