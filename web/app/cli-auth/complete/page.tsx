import { SignInButton } from "@clerk/nextjs";

import { completeCliAuth, requireBrowserIdentity } from "../../../lib/teams";

type PageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function CliAuthCompletePage({ searchParams }: PageProps) {
  const params = (await searchParams) ?? {};
  const session = Array.isArray(params.session)
    ? params.session[0]
    : params.session;
  const code = Array.isArray(params.code) ? params.code[0] : params.code;

  const identity = await requireBrowserIdentity().catch(() => null);

  // --- State 1: not signed in ---------------------------------------------
  if (!identity) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div
          className="block stack"
          style={{ maxWidth: 640, margin: "0 auto" }}
        >
          <span className="eyebrow">Device flow · step 02</span>
          <h1 className="text-xl fw-800" style={{ lineHeight: 1.15 }}>
            Log in to finish the handshake.
          </h1>
          <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
            The terminal opened this page and is waiting. Sign in and the Teams access token
            is issued automatically.
          </p>
          <div className="row">
            <SignInButton mode="modal">
              <button className="btn btn--primary" type="button">
                Log in
              </button>
            </SignInButton>
          </div>
        </div>
      </main>
    );
  }

  // --- State 2: missing session id ----------------------------------------
  if (!session) {
    return (
      <main className="shell" style={{ padding: "48px 0" }}>
        <div
          className="block stack"
          style={{ maxWidth: 640, margin: "0 auto" }}
        >
          <span className="eyebrow">Missing session</span>
          <h1 className="text-xl fw-800">No device code in this link.</h1>
          <p className="muted text-sm">
            Open this page from a fresh <code>sshthing login</code> run — the
            link expires after ten minutes.
          </p>
        </div>
      </main>
    );
  }

  // --- State 3: attempt completion ----------------------------------------
  let ok = true;
  let errorMsg: string | null = null;
  try {
    await completeCliAuth(session, code ?? null);
  } catch (error) {
    ok = false;
    errorMsg =
      error instanceof Error ? error.message : "Failed to complete CLI login.";
  }

  return (
    <main className="shell" style={{ padding: "48px 0" }}>
      <div
        className="stack"
        style={{ maxWidth: 720, margin: "0 auto", gap: 20 }}
      >
        <div
          className={ok ? "block block--accent" : "block"}
          style={ok ? undefined : { borderColor: "var(--danger)" }}
        >
          <span
            className="eyebrow"
            style={ok ? { color: "inherit", opacity: 0.75 } : { color: "var(--danger)" }}
          >
            {ok ? "Handshake complete" : "Handshake failed"}
          </span>
          <h1 className="text-xl fw-800" style={{ lineHeight: 1.2, marginTop: 6 }}>
            {ok ? "You're signed in — return to the terminal." : "Something went wrong."}
          </h1>
          <p
            className="text-sm"
            style={{
              marginTop: 10,
              lineHeight: 1.6,
              opacity: ok ? 0.85 : 1,
              color: ok ? "inherit" : "var(--danger)",
            }}
          >
            {ok
              ? "The TUI should pick up your Teams session within a couple of seconds. You can close this tab."
              : errorMsg}
          </p>
        </div>

        <div className="block block--flush">
          <div className="term">
            <div className="term__bar">~ sshthing · device flow</div>
            <div className="term__body">
              <span className="term__line">
                <span className="term__prompt">$</span> sshthing login
              </span>
              <span className="term__line muted">
                → device code:{" "}
                <strong>{code ?? "—"}</strong>
              </span>
              <span className="term__line muted">→ browser authenticated…</span>
              {ok ? (
                <>
                  <span
                    className="term__line"
                    style={{ color: "var(--accent)" }}
                  >
                    ✓ token issued — close this tab
                  </span>
                  <span className="term__line">
                    <span className="term__prompt">$</span>
                    <span className="term__cursor" aria-hidden="true" />
                  </span>
                </>
              ) : (
                <>
                  <span
                    className="term__line"
                    style={{ color: "var(--danger)" }}
                  >
                    ✗ {errorMsg}
                  </span>
                  <span className="term__line muted">
                    → restart the flow from the TUI
                  </span>
                </>
              )}
            </div>
          </div>
        </div>

        <p className="eyebrow" style={{ textAlign: "center" }}>
          Signed in as {identity.displayName}
        </p>
      </div>
    </main>
  );
}
