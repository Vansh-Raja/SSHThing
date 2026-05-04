import { NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { parseTeamAutomationToken } from "@/lib/teamAutomationTokens";

type Params = {
  params: Promise<{ executionId: string }>;
};

export async function POST(request: Request, { params }: Params) {
  const { executionId } = await params;
  const body = (await request.json().catch(() => ({}))) as {
    token?: string;
    status?: string;
    exitCode?: number | null;
    error?: string;
  };

  try {
    const parsed = parseTeamAutomationToken(body.token ?? "");
    const result = await convexMutation<{ ok: boolean }>(
      convexApi.teamAutomationTokens.finishExecution,
      {
        tokenId: parsed.tokenId,
        tokenHash: parsed.tokenHash,
        executionId,
        status: body.status ?? "failed",
        exitCode: body.exitCode ?? null,
        error: body.error ?? "",
      },
    );
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "finish_team_token_execution_failed" },
      { status: 400 },
    );
  }
}
