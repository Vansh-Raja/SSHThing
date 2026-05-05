import { NextRequest, NextResponse } from "next/server";

import { convexApi, convexMutation } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function POST(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const body = (await request.json()) as {
      deviceId?: string;
      deviceName?: string;
      lastPulledRevision?: number;
      lastPushedRevision?: number;
    };
    await convexMutation(convexApi.personalVaults.markDeviceSeen, {
      clerkUserId: actor.clerkUserId,
      deviceId: body.deviceId ?? "",
      deviceName: body.deviceName,
      lastPulledRevision: body.lastPulledRevision,
      lastPushedRevision: body.lastPushedRevision,
    });
    return NextResponse.json({ ok: true });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_device_seen_failed" },
      { status: 400 },
    );
  }
}
