import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { decryptTeamSecret } from "@/lib/teamSecrets";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ hostId: string; memberId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { hostId, memberId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const revealed = await convexMutation<Record<string, unknown>>(convexApi.teamHosts.revealMemberCredential, {
      hostId,
      memberClerkUserId: memberId,
      clerkUserId: actor.clerkUserId,
    });

    return NextResponse.json({
      ...revealed,
      secret:
        typeof revealed.ciphertext === "string" && revealed.ciphertext
          ? decryptTeamSecret(revealed.ciphertext)
          : "",
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "reveal_member_credential_failed" },
      { status: 400 },
    );
  }
}
