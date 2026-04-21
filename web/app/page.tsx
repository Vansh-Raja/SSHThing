import Link from "next/link";
import { Show, SignInButton, SignUpButton } from "@clerk/nextjs";

import { hasBrowserTeamsEnv } from "../lib/env";

export default function HomePage() {
  const hasEnv = hasBrowserTeamsEnv();

  return (
    <main className="shell">
      <section className="hero">
        <div className="hero__inner">
          <span className="hero__eyebrow">
            {hasEnv ? "Teams · device flow · clerk + convex" : "Setup required"}
          </span>

          <h1 className="hero__wordmark" aria-label="SSHThing">
            sshthing<span>.</span>
          </h1>

          {hasEnv ? (
            <>
              <div className="hero__taglineRow">
                <div className="block hero__tagline">
                  Cloud-backed SSH access
                  <br />
                  for your team.
                </div>
                <div className="block hero__price">
                  <span className="hero__price-value">FREE</span>
                  <span className="hero__price-label">while in beta</span>
                </div>
              </div>

              <div className="hero__actions">
                <Show when="signed-out">
                  <SignUpButton mode="modal">
                    <button
                      className="cta cta--primary cta--arrow"
                      type="button"
                    >
                      Start free · get access
                    </button>
                  </SignUpButton>
                  <SignInButton mode="modal">
                    <button className="cta" type="button">
                      Log in
                    </button>
                  </SignInButton>
                </Show>
                <Show when="signed-in">
                  <Link className="cta cta--primary cta--arrow" href="/teams">
                    Open teams
                  </Link>
                  <Link className="cta" href="/">
                    Home
                  </Link>
                </Show>
              </div>
            </>
          ) : (
            <div className="block" style={{ maxWidth: 640 }}>
              <span className="eyebrow">Configuration required</span>
              <p
                style={{
                  marginTop: 10,
                  fontSize: 16,
                  fontWeight: 600,
                }}
              >
                Missing Clerk or Convex environment variables.
              </p>
              <p className="muted" style={{ marginTop: 6, fontSize: 13 }}>
                See <code>web/.env.example</code> and restart the dev server.
              </p>
            </div>
          )}
        </div>
      </section>

      {hasEnv ? (
        <section
          className="stack"
          style={{ paddingBottom: "clamp(28px, 4vw, 56px)" }}
        >
          <div className="grid-2">
            <div className="block stack">
              <span className="eyebrow">What it is</span>
              <h2 className="text-xl fw-800" style={{ lineHeight: 1.15 }}>
                The browser half of SSHThing Teams.
              </h2>
              <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
                You sign in here. The terminal app opens this site for the
                device-code handshake, then gets a scoped token. That&apos;s
                it. Team setup, invites, and shared host management live here;
                the TUI stays focused on browsing and connecting.
              </p>
              <div className="row">
                <Link className="btn" href="/login">
                  Log in
                </Link>
                <Link className="btn btn--primary" href="/signup">
                  Create account
                </Link>
              </div>
            </div>

            <div className="block block--flush">
              <div className="term">
                <div className="term__bar">~ sshthing · login</div>
                <div className="term__body">
                  <span className="term__line">
                    <span className="term__prompt">$</span> sshthing login
                  </span>
                  <span className="term__line muted">
                    → opening browser for sign in…
                  </span>
                  <span className="term__line">
                    <span className="term__prompt">$</span> device code:{" "}
                    <strong>A9-7F-2C</strong>
                  </span>
                  <span className="term__line muted">
                    → polling for completion…
                  </span>
                  <span
                    className="term__line"
                    style={{ color: "var(--accent)" }}
                  >
                    ✓ signed in · token stored locally
                  </span>
                  <span className="term__line">
                    <span className="term__prompt">$</span>
                    <span className="term__cursor" aria-hidden="true" />
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>
      ) : null}
    </main>
  );
}
