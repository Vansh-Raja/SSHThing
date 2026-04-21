"use client";

import { useEffect } from "react";
import Link from "next/link";

type ErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function RuntimeError({ error, reset }: ErrorProps) {
  useEffect(() => {
    // Surface to console so dev mode shows stack + digest.
    // eslint-disable-next-line no-console
    console.error("[sshthing] runtime error:", error);
  }, [error]);

  return (
    <main className="shell" style={{ padding: "clamp(40px, 8vw, 96px) 0" }}>
      <div
        className="stack"
        style={{ maxWidth: 720, margin: "0 auto", gap: 24 }}
      >
        <span
          className="hero__eyebrow"
          style={{ color: "var(--danger)" }}
        >
          500 · runtime error
        </span>

        <h1
          className="fw-800"
          style={{
            fontSize: "clamp(2.5rem, 8vw, 5.5rem)",
            lineHeight: 0.95,
            letterSpacing: "-0.035em",
          }}
        >
          something broke
          <br />
          <span style={{ color: "var(--danger)" }}>on our end.</span>
        </h1>

        <div className="block block--flush">
          <div className="term">
            <div className="term__bar">~ sshthing · stderr</div>
            <div className="term__body">
              <span
                className="term__line"
                style={{ color: "var(--danger)" }}
              >
                ✗ {error.message || "unknown error"}
              </span>
              {error.digest ? (
                <span className="term__line muted">
                  → digest: {error.digest}
                </span>
              ) : null}
              <span className="term__line muted">
                → try again, or head back to the home route
              </span>
              <span className="term__line">
                <span className="term__prompt">$</span>
                <span className="term__cursor" aria-hidden="true" />
              </span>
            </div>
          </div>
        </div>

        <div className="row">
          <button
            type="button"
            onClick={reset}
            className="cta cta--primary cta--arrow"
          >
            Retry
          </button>
          <Link className="cta" href="/">
            Back home
          </Link>
        </div>
      </div>
    </main>
  );
}
