import { SignIn } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../../../lib/env";

export default function LoginPage() {
  if (!hasBrowserTeamsEnv()) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div className="block stack" style={{ maxWidth: 640, margin: "0 auto" }}>
          <span className="eyebrow">Setup required</span>
          <h1 className="text-xl fw-800">Configure Clerk + Convex first.</h1>
          <p className="muted text-sm">
            Set the required environment variables before using login.
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
          <span className="eyebrow">Step 01 · authenticate</span>
          <h1 className="text-xl fw-800" style={{ lineHeight: 1.15 }}>
            Log in to continue.
          </h1>
          <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
            After you sign in the TUI completes its device handshake and gets a
            scoped token. No other action needed here.
          </p>
          <hr className="hr" />
          <div className="term" style={{ fontSize: 12 }}>
            <span className="term__line">
              <span className="term__prompt">$</span> sshthing login
            </span>
            <span className="term__line muted">→ waiting for browser…</span>
          </div>
        </aside>

        <div className="block" style={{ padding: "24px 20px" }}>
          <SignIn
            routing="path"
            path="/login"
            signUpUrl="/signup"
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
