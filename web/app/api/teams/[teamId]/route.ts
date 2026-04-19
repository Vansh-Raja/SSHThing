import { NextResponse } from "next/server";

import { deleteTeamFromBearer, renameTeamFromBearer } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function PATCH(request: Request, { params }: Params) {
  const { teamId } = await params;
  const body = (await request.json().catch(() => ({}))) as { name?: string };
  if (!body.name?.trim()) {
    return NextResponse.json({ error: "missing_team_name" }, { status: 400 });
  }

  try {
    const team = await renameTeamFromBearer(request.headers.get("authorization"), teamId, body.name);
    return NextResponse.json(team);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "rename_team_failed" },
      { status: 400 },
    );
  }
}

export async function DELETE(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const result = await deleteTeamFromBearer(request.headers.get("authorization"), teamId);
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "delete_team_failed" },
      { status: 400 },
    );
  }
}
