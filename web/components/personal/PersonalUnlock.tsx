"use client";

import { useState } from "react";

type Props = {
  onUnlock: (password: string) => void;
  error?: string;
};

export default function PersonalUnlock({ onUnlock, error }: Props) {
  const [password, setPassword] = useState("");
  return (
    <div className="block stack" style={{ maxWidth: 560 }}>
      <span className="eyebrow">End-to-end encrypted</span>
      <h1 className="text-xl fw-800">Unlock your personal library.</h1>
      <p className="muted text-sm" style={{ lineHeight: 1.6 }}>
        Enter your SSHThing sync password. It is used only in this browser tab
        to decrypt and encrypt your personal hosts; it is never sent to the
        server.
      </p>
      <form
        className="stack"
        onSubmit={(event) => {
          event.preventDefault();
          if (password.trim()) onUnlock(password);
        }}
      >
        <label className="field">
          <span className="field__label">Sync password</span>
          <input
            className="field__input"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
          />
        </label>
        {error ? <p className="muted text-sm">{error}</p> : null}
        <button className="btn btn--primary" type="submit">
          Unlock
        </button>
      </form>
    </div>
  );
}
