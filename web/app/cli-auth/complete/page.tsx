import Link from "next/link";
import { SignInButton } from "@clerk/nextjs";

import { completeCliAuth, requireBrowserIdentity } from "../../../lib/teams";

type PageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function CliAuthCompletePage({ searchParams }: PageProps) {
  const params = (await searchParams) ?? {};
  const session = Array.isArray(params.session) ? params.session[0] : params.session;
  const code = Array.isArray(params.code) ? params.code[0] : params.code;

  const identity = await requireBrowserIdentity().catch(() => null);
  if (!identity) {
    return (
      <main className="shell">
        <section className="card stack">
          <h1>Sign in to complete SSHThing CLI auth</h1>
          <p className="muted">The CLI handoff can only be completed from an authenticated browser session.</p>
          <SignInButton mode="modal">
            <button className="buttonLink buttonPrimary" type="button">
              Sign in
            </button>
          </SignInButton>
        </section>
      </main>
    );
  }

  if (!session) {
    return (
      <main className="shell">
        <section className="card stack">
          <h1>Missing CLI auth session</h1>
          <p className="muted">Open this page from a fresh device-flow link created by the SSHThing TUI.</p>
        </section>
      </main>
    );
  }

  let status = "SSHThing sign-in completed. You can return to the terminal.";
  let isError = false;
  try {
    await completeCliAuth(session, code ?? null);
  } catch (error) {
    isError = true;
    status = error instanceof Error ? error.message : "Failed to complete SSHThing CLI login.";
  }

  return (
    <main className="shell">
      <section className="card stack">
        <h1>SSHThing CLI login</h1>
        <p className={isError ? "noticeDanger" : "noticeSuccess"}>{status}</p>
        <p className="muted">
          {isError
            ? "If the terminal is still polling, restart the login flow from the TUI and try again."
            : "The terminal should finish polling within a few seconds."}
        </p>
        <div className="pillRow">
          <span className="pill">Device code: {code ?? "n/a"}</span>
        </div>
      </section>
    </main>
  );
}
