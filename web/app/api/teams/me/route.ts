import { NextResponse } from "next/server";

import { getSessionContextFromBearer } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const context = await getSessionContextFromBearer(request.headers.get("authorization"));
    return NextResponse.json({
      auth: {
        authenticated: true,
        hasWorkspace: false,
        userId: context.session.clerkUserId,
      },
    });
  } catch (error) {
    return NextResponse.json(
      {
        auth: {
          authenticated: false,
          hasWorkspace: false,
        },
        error: error instanceof Error ? error.message : "unauthorized",
      },
      { status: 401 },
    );
  }
}
