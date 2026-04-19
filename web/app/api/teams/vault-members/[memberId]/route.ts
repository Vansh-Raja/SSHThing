import { NextResponse } from "next/server";
import { clerkClient } from "@clerk/nextjs/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { requireBrowserIdentity, ensureWorkspaceForCurrentOrg } from "@/lib/teams";

type RouteProps = {
  params: Promise<{ memberId: string }>;
};

function vaultRoleToClerkRole(vaultRole: string): string {
  return vaultRole === "vault_admin" ? "org:admin" : "org:member";
}

export async function PATCH(request: Request, { params }: RouteProps) {
  try {
    const identity = await requireBrowserIdentity();
    if (!identity.organization) {
      return NextResponse.json({ error: "missing_active_organization" }, { status: 400 });
    }

    const { memberId } = await params;
    const body = (await request.json().catch(() => ({}))) as { vaultId?: string; role?: string };
    if (!body.vaultId || !body.role) {
      return NextResponse.json({ error: "missing_vault_or_role" }, { status: 400 });
    }

    const workspace = await ensureWorkspaceForCurrentOrg();
    const member = await convexQuery<{ clerkUserId: string } | null>(convexApi.memberships.getWorkspaceMember, {
      workspaceId: workspace.workspaceId,
      memberId,
    });
    if (!member) {
      return NextResponse.json({ error: "member_not_found" }, { status: 404 });
    }

    const client = await clerkClient();
    await client.organizations.updateOrganizationMembership({
      organizationId: identity.organization.id,
      userId: member.clerkUserId,
      role: vaultRoleToClerkRole(body.role),
    });

    const updated = await convexMutation<Record<string, unknown>>(convexApi.memberships.setVaultRole, {
      workspaceId: workspace.workspaceId,
      vaultId: body.vaultId,
      clerkUserId: member.clerkUserId,
      vaultRole: body.role,
    });
    await convexMutation<{ ok: boolean }>(convexApi.audit.logEvent, {
      workspaceId: workspace.workspaceId,
      actorUserId: identity.userId,
      eventType: "membership.role_changed",
      targetType: "workspace_member",
      targetId: memberId,
      metadata: {
        vaultId: body.vaultId,
        vaultRole: body.role,
      },
    });

    return NextResponse.json(updated);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "update_failed" },
      { status: 400 },
    );
  }
}
