import { SignUp } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../../../lib/env";

export default function SignupPage() {
  if (!hasBrowserTeamsEnv()) {
    return (
      <main className="shell">
        <section className="card stack">
          <h1>Signup setup required</h1>
          <p className="noticeDanger">Set the Clerk and Convex environment variables before using the signup page.</p>
        </section>
      </main>
    );
  }

  return (
    <main className="shell">
      <section className="card stack">
        <h1>Create your SSHThing Teams account</h1>
        <p className="muted">New users sign up here before using a personal or shared SSHThing workspace.</p>
        <SignUp />
      </section>
    </main>
  );
}
