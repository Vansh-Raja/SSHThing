import { NextResponse } from "next/server";

import { getActorFromRequest } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const actor = await getActorFromRequest(request.headers.get("authorization"));
    return NextResponse.json({
      auth: {
        authenticated: true,
        userId: actor.clerkUserId,
      },
    });
  } catch (error) {
    return NextResponse.json(
      {
        auth: {
          authenticated: false,
        },
        error: error instanceof Error ? error.message : "unauthorized",
      },
      { status: 401 },
    );
  }
}
