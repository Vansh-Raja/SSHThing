import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ hostId: string; memberId: string }>;
};

export async function DELETE(request: Request, { params }: Params) {
  const { hostId, memberId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const result = await convexMutation<{ ok: boolean }>(convexApi.teamHosts.deleteMemberCredentialAsAdmin, {
      hostId,
      memberClerkUserId: memberId,
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_member_credential_failed" },
      { status: 400 },
    );
  }
}
