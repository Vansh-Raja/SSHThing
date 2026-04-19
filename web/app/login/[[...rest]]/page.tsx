import { SignIn } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../../../lib/env";

export default function LoginPage() {
  if (!hasBrowserTeamsEnv()) {
    return (
      <main className="shell">
        <section className="card stack">
          <h1>Login setup required</h1>
          <p className="noticeDanger">Set the Clerk and Convex environment variables before using the login page.</p>
        </section>
      </main>
    );
  }

  return (
    <main className="shell">
      <section className="card stack">
        <h1>Sign in to SSHThing Teams</h1>
        <p className="muted">Use your Clerk account to authenticate, then return to the TUI.</p>
        <SignIn />
      </section>
    </main>
  );
}
