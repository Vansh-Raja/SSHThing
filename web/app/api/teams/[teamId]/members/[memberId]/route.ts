import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string; memberId: string }>;
};

export async function PATCH(request: Request, { params }: Params) {
  const { teamId, memberId } = await params;
  const body = (await request.json().catch(() => ({}))) as { role?: string };
  if (!body.role) {
    return NextResponse.json({ error: "missing_role" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const updated = await convexMutation<Record<string, unknown>>(convexApi.teamMembers.updateRole, {
      teamId,
      memberId,
      clerkUserId: actor.clerkUserId,
      role: body.role,
    });
    return NextResponse.json(updated);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "update_member_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { teamId, memberId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamMembers.remove, {
      teamId,
      memberId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "remove_member_failed" },
      { status: 400 },
    );
  }
}
