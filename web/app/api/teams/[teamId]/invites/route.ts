import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { encryptTeamSecret } from "@/lib/teamSecrets";
import { buildInviteLink, getActorFromRequest } from "@/lib/teams";
import { createRefreshToken, hashToken } from "@/lib/tokens";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { teamId } = await params;
  const body = (await request.json().catch(() => ({}))) as { email?: string; role?: string };
  const email = body.email?.trim().toLowerCase();
  if (!email) {
    return NextResponse.json({ error: "missing_email" }, { status: 400 });
  }

  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const rawToken = createRefreshToken();
    const invite = await convexMutation<Record<string, unknown>>(convexApi.teamInvites.create, {
      teamId,
      clerkUserId: actor.clerkUserId,
      emailLower: email,
      role: body.role ?? "member",
      tokenHash: hashToken(rawToken),
      tokenCiphertext: encryptTeamSecret(rawToken),
      expiresAt: Date.now() + 1000 * 60 * 60 * 24 * 7,
    });

    return NextResponse.json({
      ...invite,
      shareUrl: buildInviteLink(String(invite.id), rawToken),
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "create_invite_failed" },
      { status: 400 },
    );
  }
}
