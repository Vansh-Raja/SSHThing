import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getWorkspaceContextFromBearer } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const { workspace } = await getWorkspaceContextFromBearer(request.headers.get("authorization"));
    const vaults = await convexQuery<Array<Record<string, unknown>>>(convexApi.vaults.listForWorkspace, {
      workspaceId: workspace.id,
    });
    return NextResponse.json(vaults);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "unauthorized" },
      { status: 401 },
    );
  }
}
