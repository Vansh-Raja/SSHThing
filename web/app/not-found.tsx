import Link from "next/link";

export default function NotFound() {
  return (
    <main className="shell" style={{ padding: "clamp(40px, 8vw, 96px) 0" }}>
      <div
        className="stack"
        style={{ maxWidth: 720, margin: "0 auto", gap: 24 }}
      >
        <span className="hero__eyebrow">404 · not found</span>

        <h1
          className="fw-800"
          style={{
            fontSize: "clamp(3rem, 10vw, 7rem)",
            lineHeight: 0.9,
            letterSpacing: "-0.04em",
          }}
        >
          route not
          <br />
          <span style={{ color: "var(--accent)" }}>found.</span>
        </h1>

        <div className="block block--flush">
          <div className="term">
            <div className="term__bar">~ sshthing</div>
            <div className="term__body">
              <span className="term__line">
                <span className="term__prompt">$</span> curl localhost:3000
                <span className="muted"> /unknown</span>
              </span>
              <span
                className="term__line"
                style={{ color: "var(--danger)" }}
              >
                ✗ HTTP 404 — no handler registered for this path
              </span>
              <span className="term__line">
                <span className="term__prompt">$</span>
                <span className="term__cursor" aria-hidden="true" />
              </span>
            </div>
          </div>
        </div>

        <div className="row">
          <Link className="cta cta--primary cta--arrow" href="/">
            Back home
          </Link>
          <Link className="cta" href="/teams">
            Open teams
          </Link>
        </div>
      </div>
    </main>
  );
}
