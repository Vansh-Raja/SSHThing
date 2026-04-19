import { NextResponse } from "next/server";

import { createTeamFromBearer } from "@/lib/teams";

export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as { name?: string };
  if (!body.name?.trim()) {
    return NextResponse.json({ error: "missing_team_name" }, { status: 400 });
  }

  try {
    const team = await createTeamFromBearer(request.headers.get("authorization"), body.name);
    return NextResponse.json(team);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "create_team_failed" },
      { status: 400 },
    );
  }
}
