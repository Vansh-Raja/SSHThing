import { NextResponse } from "next/server";

import { refreshTuiAccess } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { refreshToken?: string };
  if (!body.refreshToken) {
    return NextResponse.json({ error: "missing_refresh_token" }, { status: 400 });
  }

  try {
    const result = await refreshTuiAccess(body.refreshToken);
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "refresh_failed" },
      { status: 401 },
    );
  }
}
