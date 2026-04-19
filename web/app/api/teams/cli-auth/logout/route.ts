import { NextResponse } from "next/server";

import { revokeTuiSession } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { refreshToken?: string };
  if (!body.refreshToken) {
    return NextResponse.json({ ok: true });
  }

  await revokeTuiSession(body.refreshToken);
  return NextResponse.json({ ok: true });
}
