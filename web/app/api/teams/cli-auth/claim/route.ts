import { NextResponse } from "next/server";

import { claimCliAuth } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as {
    sessionId?: string;
    pollSecret?: string;
    claimCode?: string;
  };
  if (!body.sessionId || !body.pollSecret || !body.claimCode) {
    return NextResponse.json(
      { error: "missing_session_secret_or_code" },
      { status: 400 },
    );
  }

  try {
    const result = await claimCliAuth(body.sessionId, body.pollSecret, body.claimCode);
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "claim_failed" },
      { status: 400 },
    );
  }
}
