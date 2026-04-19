import { NextResponse } from "next/server";
import { clerkClient } from "@clerk/nextjs/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { requireBrowserIdentity, ensureWorkspaceForCurrentOrg } from "@/lib/teams";

type RouteProps = {
  params: Promise<{ memberId: string }>;
};

export async function DELETE(_request: Request, { params }: RouteProps) {
  try {
    const identity = await requireBrowserIdentity();
    if (!identity.organization) {
      return NextResponse.json({ error: "missing_active_organization" }, { status: 400 });
    }

    const { memberId } = await params;
    const workspace = await ensureWorkspaceForCurrentOrg();
    const member = await convexQuery<{ clerkUserId: string } | null>(convexApi.memberships.getWorkspaceMember, {
      workspaceId: workspace.workspaceId,
      memberId,
    });
    if (!member) {
      return NextResponse.json({ error: "member_not_found" }, { status: 404 });
    }

    const client = await clerkClient();
    await client.organizations.deleteOrganizationMembership({
      organizationId: identity.organization.id,
      userId: member.clerkUserId,
    });

    await convexMutation<{ ok: boolean }>(convexApi.memberships.deactivateWorkspaceMember, {
      workspaceId: workspace.workspaceId,
      clerkUserId: member.clerkUserId,
    });
    await convexMutation<{ ok: boolean }>(convexApi.audit.logEvent, {
      workspaceId: workspace.workspaceId,
      actorUserId: identity.userId,
      eventType: "membership.removed",
      targetType: "workspace_member",
      targetId: memberId,
      metadata: {
        clerkUserId: member.clerkUserId,
      },
    });

    return NextResponse.json({ ok: true });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_failed" },
      { status: 400 },
    );
  }
}
