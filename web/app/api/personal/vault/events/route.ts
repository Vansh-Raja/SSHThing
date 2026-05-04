import { NextRequest, NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

export async function GET(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const events = await convexQuery(convexApi.personalVaults.listEvents, {
      clerkUserId: actor.clerkUserId,
    });
    return NextResponse.json(events);
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_events_failed" },
      { status: 400 },
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const body = (await request.json()) as {
      deviceId?: string;
      source?: string;
      action?: string;
      itemType?: string;
      itemCount?: number;
    };
    await convexMutation(convexApi.personalVaults.recordSyncEvent, {
      clerkUserId: actor.clerkUserId,
      deviceId: body.deviceId,
      source: body.source ?? "web",
      action: body.action ?? "edit",
      itemType: body.itemType,
      itemCount: body.itemCount,
    });
    return NextResponse.json({ ok: true });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_event_save_failed" },
      { status: 400 },
    );
  }
}
