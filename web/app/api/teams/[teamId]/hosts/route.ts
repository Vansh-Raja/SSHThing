import { NextResponse } from "next/server";

import { listTeamHostsFromBearer } from "@/lib/teams";

type Params = {
  params: Promise<{ teamId: string }>;
};

export async function GET(request: Request, { params }: Params) {
  const { teamId } = await params;
  try {
    const hosts = await listTeamHostsFromBearer(request.headers.get("authorization"), teamId);
    return NextResponse.json(hosts);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "list_team_hosts_failed" },
      { status: 400 },
    );
  }
}
