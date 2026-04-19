import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { getWorkspaceContextFromBearer } from "@/lib/teams";

type RouteProps = {
  params: Promise<{ resourceId: string }>;
};

export async function POST(request: Request, { params }: RouteProps) {
  try {
    const { session, workspace } = await getWorkspaceContextFromBearer(request.headers.get("authorization"));
    const { resourceId } = await params;

    const resource = await convexQuery<Record<string, unknown> | null>(convexApi.resources.getResourceById, {
      resourceId,
    });
    if (!resource) {
      return NextResponse.json({ error: "resource_not_found" }, { status: 404 });
    }

    const vaultId = String(resource.vaultId ?? "");
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

    return NextResponse.json({
      resourceId,
      vaultId,
      hostname: resource.hostname ?? "",
      username: resource.username ?? "",
      port: resource.port ?? 22,
      shareMode: resource.shareMode ?? "host_only",
      allowed: true,
      revealAllowed: false,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "connect_failed" },
      { status: 401 },
    );
  }
}
