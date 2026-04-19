import { NextResponse } from "next/server";

import { buildCliAuthStart } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { deviceName?: string };
  const deviceName = body.deviceName?.trim() || "SSHThing TUI";
  const started = await buildCliAuthStart(deviceName);
  return NextResponse.json(started);
}
