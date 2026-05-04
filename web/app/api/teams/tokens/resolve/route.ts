import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { decryptTeamSecret } from "@/lib/teamSecrets";
import { parseTeamAutomationToken } from "@/lib/teamAutomationTokens";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as {
    token?: string;
    teamId?: string;
    target?: string;
    targetId?: string;
    command?: string;
    clientDevice?: string;
  };

  try {
    const parsed = parseTeamAutomationToken(body.token ?? "");
    const resolved = await convexMutation<{
      executionId: string;
      host: Record<string, unknown> & { secret?: string };
    }>(convexApi.teamAutomationTokens.resolveForExecution, {
      tokenId: parsed.tokenId,
      tokenHash: parsed.tokenHash,
      teamId: body.teamId || null,
      target: body.target ?? "",
      targetId: body.targetId || null,
      command: body.command ?? "",
      clientDevice: body.clientDevice ?? "",
    });

    return NextResponse.json({
      ...resolved,
      host: {
        ...resolved.host,
        secret:
          typeof resolved.host.secret === "string" && resolved.host.secret
            ? decryptTeamSecret(resolved.host.secret)
            : "",
      },
    });
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "resolve_team_token_failed" },
      { status: 400 },
    );
  }
}
