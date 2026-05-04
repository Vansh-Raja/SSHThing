import { NextRequest, NextResponse } from "next/server";

import { convexApi, convexMutation, convexQuery } from "@/lib/convex";
import { getActorFromRequest } from "@/lib/teams";

function parseSince(value: string | null): number | undefined {
  if (!value) return undefined;
  const n = Number(value);
  return Number.isFinite(n) && n > 0 ? n : undefined;
}

export async function GET(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const { searchParams } = new URL(request.url);
    const result = await convexQuery(convexApi.personalVaults.listItems, {
      clerkUserId: actor.clerkUserId,
      since: parseSince(searchParams.get("since")),
    });
    return NextResponse.json(result);
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_items_failed" },
      { status: 400 },
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    const body = (await request.json()) as {
      deviceId?: string;
      force?: boolean;
      items?: unknown[];
    };
    const result = await convexMutation(convexApi.personalVaults.upsertItems, {
      clerkUserId: actor.clerkUserId,
      deviceId: body.deviceId ?? "",
      force: body.force ?? false,
      items: body.items ?? [],
    });
    return NextResponse.json(result);
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "personal_items_save_failed" },
      { status: 400 },
    );
  }
}
