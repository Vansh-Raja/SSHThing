import { NextResponse } from "next/server";

import { buildCliAuthStart } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as {
    deviceName?: string;
    mode?: string;
  };
  const deviceName = body.deviceName?.trim() || "SSHThing TUI";
  const headless = body.mode?.trim().toLowerCase() === "headless";
  const started = await buildCliAuthStart(deviceName, headless);
  return NextResponse.json(started);
}
