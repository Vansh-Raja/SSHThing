import { Suspense } from "react";
import { auth } from "@clerk/nextjs/server";
import { SignInButton, SignUpButton } from "@clerk/nextjs";

import PersonalDashboard from "../../components/personal/PersonalDashboard";
import { hasServerTeamsEnv } from "../../lib/env";

function PersonalFallback() {
  return <main className="shell" style={{ padding: "48px 0" }}>Loading personal library…</main>;
}

export default async function PersonalPage() {
  if (!hasServerTeamsEnv()) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div className="block stack" style={{ maxWidth: 640 }}>
          <span className="eyebrow">Setup required</span>
          <h1 className="text-xl fw-800">Configure Clerk + Convex first.</h1>
          <p className="muted text-sm">
            Personal cloud sync requires the same account backend used by Teams.
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
          <h1 className="text-xl fw-800">Log in to manage your personal library.</h1>
          <p className="muted text-sm">
            Your browser will still need your sync password before any personal
            hosts can be decrypted or edited.
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
    <Suspense fallback={<PersonalFallback />}>
      <PersonalDashboard />
    </Suspense>
  );
}
