import { NextResponse } from "next/server";

import { convexApi, convexQuery } from "@/lib/convex";
import { authErrorStatus } from "@/lib/httpErrors";
import { decryptTeamSecret } from "@/lib/teamSecrets";
import { buildInviteLink, getActorFromRequest } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const invites = await convexQuery<{
      incoming: Array<Record<string, unknown>>;
      sent: Array<Record<string, unknown>>;
    }>(convexApi.teamInvites.listForUser, {
      clerkUserId: actor.clerkUserId,
      emailLower: actor.email,
    });

    const sent = invites.sent.map((invite) => {
      const tokenCiphertext = typeof invite.tokenCiphertext === "string" ? invite.tokenCiphertext : "";
      const shareUrl = tokenCiphertext
        ? buildInviteLink(String(invite.id), decryptTeamSecret(tokenCiphertext))
        : null;
      return {
        ...invite,
        tokenCiphertext: undefined,
        shareUrl,
      };
    });

    return NextResponse.json({
      incoming: invites.incoming,
      sent,
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_invites_failed" },
      { status: authErrorStatus(error) },
    );
  }
}
