import { NextResponse } from "next/server";
import { clerkClient } from "@clerk/nextjs/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { requireBrowserIdentity, ensureWorkspaceForCurrentOrg } from "@/lib/teams";

export async function POST(request: Request) {
  try {
    const identity = await requireBrowserIdentity();
    if (!identity.organization) {
      return NextResponse.json({ error: "missing_active_organization" }, { status: 400 });
    }

    const body = (await request.json().catch(() => ({}))) as { email?: string; role?: string; vaultId?: string };
    const email = body.email?.trim().toLowerCase();
    const role = body.role?.trim() || "org:member";
    const vaultId = body.vaultId?.trim();
    if (!email || !vaultId) {
      return NextResponse.json({ error: "missing_email_or_vault" }, { status: 400 });
    }

    const workspace = await ensureWorkspaceForCurrentOrg();
    const client = await clerkClient();
    const invitation = await client.organizations.createOrganizationInvitation({
      organizationId: identity.organization.id,
      emailAddress: email,
      role,
    });

    await convexMutation<{ ok: boolean }>(convexApi.memberships.upsertPendingInvitation, {
      workspaceId: workspace.workspaceId,
      vaultId,
      invitationId: invitation.id,
      email,
      invitedBy: identity.userId,
      vaultRole: role === "org:admin" ? "vault_admin" : "viewer",
    });
    await convexMutation<{ ok: boolean }>(convexApi.audit.logEvent, {
      workspaceId: workspace.workspaceId,
      actorUserId: identity.userId,
      eventType: "invite.sent",
      targetType: "organization_invitation",
      targetId: invitation.id,
      metadata: {
        email,
        vaultId,
        clerkRole: role,
      },
    });

    return NextResponse.json({
      id: invitation.id,
      email,
      status: invitation.status,
      role,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "invite_failed" },
      { status: 400 },
    );
  }
}
