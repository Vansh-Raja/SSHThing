import { Suspense } from "react";
import { auth } from "@clerk/nextjs/server";
import { SignInButton, SignUpButton } from "@clerk/nextjs";

import TeamsDashboard from "../../components/TeamsDashboard";
import { hasServerTeamsEnv } from "../../lib/env";

function DashboardFallback() {
  return (
    <main className="teams-page">
      <div className="team-bar">
        <div className="team-bar__row">
          <div className="team-bar__switcher" aria-busy="true">
            Loading teams…
          </div>
        </div>
      </div>
    </main>
  );
}

export default async function TeamsPage() {
  if (!hasServerTeamsEnv()) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div className="block stack" style={{ maxWidth: 640 }}>
          <span className="eyebrow">Setup required</span>
          <h1 className="text-xl fw-800">Configure Clerk + Convex first.</h1>
          <p className="muted text-sm">
            Browser auth requires the environment variables listed in <code>web/.env.example</code>.
          </p>
        </div>
      </main>
    );
  }

  const { userId } = await auth();

  if (!userId) {
    return (
      <main className="shell" style={{ padding: "48px 0 64px" }}>
        <div className="block stack" style={{ maxWidth: 640 }}>
          <span className="eyebrow">Sign in required</span>
          <h1 className="text-xl fw-800">Log in to manage your teams.</h1>
          <p className="muted text-sm">
            This dashboard manages teams, members, invites, and shared hosts for the SSHThing terminal app.
          </p>
          <div className="row">
            <SignInButton mode="modal">
              <button className="btn btn--primary" type="button">
                Log in
              </button>
            </SignInButton>
            <SignUpButton mode="modal">
              <button className="btn" type="button">
                Create account
              </button>
            </SignUpButton>
          </div>
        </div>
      </main>
    );
  }

  return (
    <Suspense fallback={<DashboardFallback />}>
      <TeamsDashboard />
    </Suspense>
  );
}
