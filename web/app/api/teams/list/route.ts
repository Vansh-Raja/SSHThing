import { NextResponse } from "next/server";

import { listTeamsFromBearer } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const teams = await listTeamsFromBearer(request.headers.get("authorization"));
    return NextResponse.json(teams);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_teams_failed" },
      { status: 401 },
    );
  }
}
