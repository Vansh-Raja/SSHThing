import Link from "next/link";
import {
  Show,
  SignInButton,
  SignUpButton,
  UserButton,
} from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../../lib/env";
import { requireBrowserIdentity } from "../../lib/teams";

async function TeamsSignedIn() {
  const identity = await requireBrowserIdentity().catch(() => null);

  if (!identity) {
    return (
      <section className="card stack">
        <h1>Teams</h1>
        <p className="muted">Your browser session is not ready yet. Reload this page after signing in.</p>
      </section>
    );
  }

  return (
    <section className="stack">
      <section className="card stack">
        <div className="actionRow" style={{ justifyContent: "space-between", alignItems: "center" }}>
          <div className="stack" style={{ gap: 8 }}>
            <h1>Teams</h1>
            <p className="muted">Use your signed-in account to manage a workspace and complete CLI auth handoff.</p>
          </div>
          <div className="actionRow">
            <UserButton />
          </div>
        </div>
        <div className="pillRow">
          <span className="pill">User: {identity.displayName}</span>
          {identity.organization?.name ? <span className="pill">Clerk organization: {identity.organization.name}</span> : null}
        </div>
        <p className="muted">Browser sign-in is ready. Teams are created explicitly from SSHThing after the terminal signs in.</p>
      </section>

      <section className="gridTwo">
        <div className="card stack">
          <h2>Next steps</h2>
          <p className="muted">
            Start the device flow from the TUI, then finish it here when you land on the CLI auth completion page.
          </p>
          <div className="code">
            In the TUI, open Profile and sign in. Then press Shift+T to enter Teams mode and create your first team.
          </div>
        </div>
        <div className="card stack">
          <h2>Teams mode</h2>
          <p className="muted">
            Clerk is only used for browser auth right now. Teams themselves are managed inside SSHThing.
          </p>
          <Link className="buttonLink buttonSecondary" href="/">
            Back home
          </Link>
        </div>
      </section>
    </section>
  );
}

export default async function TeamsPage() {
  if (!hasBrowserTeamsEnv()) {
    return (
      <main className="shell">
        <section className="card stack">
          <h1>Teams setup required</h1>
          <p className="noticeDanger">Set the Clerk and Convex environment variables before using browser auth.</p>
        </section>
      </main>
    );
  }

  return (
    <main className="shell">
      <Show
        when="signed-in"
        fallback={
          <section className="card stack">
            <h1>Sign in to access Teams</h1>
            <p className="muted">Browser auth happens here before the TUI receives a device session.</p>
            <div className="actionRow">
              <SignInButton mode="modal">
                <button className="buttonLink buttonPrimary" type="button">
                  Sign in
                </button>
              </SignInButton>
              <SignUpButton mode="modal">
                <button className="buttonLink buttonSecondary" type="button">
                  Sign up
                </button>
              </SignUpButton>
            </div>
          </section>
        }
      >
        <TeamsSignedIn />
      </Show>
    </main>
  );
}
