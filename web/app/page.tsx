import Link from "next/link";
import { Show, SignInButton, SignUpButton, UserButton } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../lib/env";

export default function HomePage() {
  const hasEnv = hasBrowserTeamsEnv();

  if (!hasEnv) {
    return (
      <main className="shell stack">
        <section className="card hero">
          <div className="pillRow">
            <span className="pill">SSHThing Teams</span>
            <span className="pill">Setup required</span>
          </div>
          <h1>Configure Clerk and Convex to enable the browser flow.</h1>
          <p className="noticeDanger">
            Missing one or more Clerk/Convex environment variables. See <code>web/.env.example</code>.
          </p>
        </section>
      </main>
    );
  }

  return (
    <main className="shell stack">
      <section className="card hero">
        <div className="pillRow">
          <span className="pill">SSHThing Teams</span>
          <span className="pill">Browser device flow</span>
          <span className="pill">Clerk + Convex</span>
        </div>
        <h1>Cloud-backed team access for SSHThing.</h1>
        <p className="muted">
          Use this browser app for authentication and CLI auth completion, with a personal workspace available by default.
        </p>
        <div className="actionRow">
          <Show when="signed-out">
            <SignInButton mode="modal">
              <button className="buttonLink buttonPrimary" type="button">
                Sign in
              </button>
            </SignInButton>
            <SignUpButton mode="modal">
              <button className="buttonLink buttonSecondary" type="button">
                Create account
              </button>
            </SignUpButton>
          </Show>
          <Show when="signed-in">
            <Link className="buttonLink buttonPrimary" href="/teams">
              Open Teams
            </Link>
            <div className="buttonLink buttonSecondary">
              <UserButton />
            </div>
          </Show>
        </div>
      </section>

      <section className="gridTwo">
        <div className="card stack">
          <h2>Browser surfaces</h2>
          <p className="muted">
            The TUI never embeds Clerk UI. It opens this web app for sign-in and browser-based auth handoff.
          </p>
          <div className="actionRow">
            <Link className="buttonLink buttonSecondary" href="/login">
              Login
            </Link>
            <Link className="buttonLink buttonSecondary" href="/signup">
              Sign up
            </Link>
            <Link className="buttonLink buttonSecondary" href="/teams">
              Teams
            </Link>
          </div>
        </div>
        <div className="card stack">
          <h2>Environment</h2>
          <p className="noticeSuccess">Required Clerk and Convex environment variables are present.</p>
          <p className="muted">Clerk Organizations are no longer required for the default local flow.</p>
        </div>
      </section>
    </main>
  );
}
