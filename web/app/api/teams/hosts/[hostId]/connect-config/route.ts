import { NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { decryptTeamSecret } from "@/lib/teamSecrets";
import { getActorFromRequest } from "@/lib/teams";

type Params = {
  params: Promise<{ hostId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { hostId } = await params;
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const config = await convexQuery<Record<string, unknown>>(convexApi.teamHosts.getConnectConfig, {
      hostId,
      clerkUserId: actor.clerkUserId,
    });
    await convexMutation<{ ok: boolean }>(convexApi.teamHosts.markConnected, {
      hostId,
    });

    return NextResponse.json({
      ...config,
      secret: typeof config.secret === "string" && config.secret ? decryptTeamSecret(config.secret) : "",
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "connect_config_failed" },
      { status: 400 },
    );
  }
}
