import { NextResponse } from "next/server";

import { pollCliAuth } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { sessionId?: string; pollSecret?: string };
  if (!body.sessionId || !body.pollSecret) {
    return NextResponse.json({ error: "missing_session_or_secret" }, { status: 400 });
  }

  try {
    const result = await pollCliAuth(body.sessionId, body.pollSecret);
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "poll_failed" },
      { status: 400 },
    );
  }
}
