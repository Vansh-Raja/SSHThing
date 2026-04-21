import { SignUp } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../../../lib/env";

export default function SignupPage() {
  if (!hasBrowserTeamsEnv()) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div className="block stack" style={{ maxWidth: 640, margin: "0 auto" }}>
          <span className="eyebrow">Setup required</span>
          <h1 className="text-xl fw-800">Configure Clerk + Convex first.</h1>
          <p className="muted text-sm">
            Set the required environment variables before using signup.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="shell" style={{ padding: "48px 0 64px" }}>
      <div
        className="grid-2"
        style={{
          alignItems: "stretch",
          maxWidth: 980,
          margin: "0 auto",
        }}
      >
        <aside className="block stack">
          <span className="eyebrow">New account · takes 30 seconds</span>
          <h1 className="text-xl fw-800" style={{ lineHeight: 1.15 }}>
            Create your SSHThing account.
          </h1>
          <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
            Sign in once, create teams in the browser, and keep the terminal
            app focused on SSH connections.
          </p>
          <hr className="hr" />
          <ul
            className="stack"
            style={{ gap: 8, listStyle: "none", padding: 0, margin: 0, fontSize: 13 }}
          >
            <li>
              <span className="term__prompt">›</span> Free during beta
            </li>
            <li>
              <span className="term__prompt">›</span> Encrypted credential
              storage
            </li>
            <li>
              <span className="term__prompt">›</span> Works with any SSH host
            </li>
          </ul>
        </aside>

        <div className="block" style={{ padding: "24px 20px" }}>
          <SignUp
            routing="path"
            path="/signup"
            signInUrl="/login"
            appearance={{
              elements: {
                rootBox: "cl-rootBox",
                card: "cl-card",
              },
            }}
          />
        </div>
      </div>
    </main>
  );
}
