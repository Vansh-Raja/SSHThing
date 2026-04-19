import { NextResponse } from "next/server";

import { getWorkspaceContextFromBearer } from "@/lib/teams";

export async function GET(request: Request) {
  try {
    const { workspace } = await getWorkspaceContextFromBearer(request.headers.get("authorization"));
    return NextResponse.json(workspace);
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "unauthorized" },
      { status: 401 },
    );
  }
}
