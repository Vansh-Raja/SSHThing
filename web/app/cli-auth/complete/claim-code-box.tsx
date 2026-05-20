"use client";

import { useState } from "react";

/**
 * Headless sign-in: shows the one-time claim code with a copy button.
 * The user pastes this code back into the TUI to finish signing in.
 */
export function ClaimCodeBox({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API unavailable (e.g. insecure context) — the user
      // can still select the code manually.
    }
  };

  return (
    <div className="stack" style={{ gap: 10 }}>
      <code
        style={{
          display: "block",
          padding: "14px 16px",
          fontSize: 18,
          fontWeight: 700,
          letterSpacing: "0.04em",
          wordBreak: "break-all",
          userSelect: "all",
          borderRadius: 8,
          border: "1px solid var(--line, #333)",
          background: "var(--surface, #111)",
        }}
      >
        {code}
      </code>
      <div className="row">
        <button className="btn btn--primary" type="button" onClick={() => void copy()}>
          {copied ? "Copied" : "Copy code"}
        </button>
      </div>
    </div>
  );
}
