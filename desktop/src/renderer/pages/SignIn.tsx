/**
 * SignIn — Convex device-code sign-in flow.
 *
 * Flow:
 *  1. User clicks "Sign in" → daemon calls StartCLIAuth → returns URL + poll params.
 *  2. Daemon opens the browser to the auth URL.
 *  3. Renderer polls every pollIntervalSeconds seconds.
 *  4. On "completed" → daemon has saved session → navigate to /hosts.
 *  5. On "expired" → show retry button.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../ui/Button';
import { toast } from '../ui/toast';

type SignInState = 'idle' | 'starting' | 'polling' | 'expired' | 'error';

export default function SignIn() {
  const navigate = useNavigate();
  const [state, setState] = useState<SignInState>('idle');
  const [errorMsg, setErrorMsg] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollParamsRef = useRef<{ sessionId: string; pollSecret: string } | null>(null);
  // Cache the browser URL so the user can re-open it without restarting the flow.
  const browserUrlRef = useRef<string | null>(null);

  const stopPoll = useCallback(() => {
    if (pollRef.current !== null) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const startPoll = useCallback(
    (sessionId: string, pollSecret: string, intervalMs: number) => {
      stopPoll();
      pollParamsRef.current = { sessionId, pollSecret };
      pollRef.current = setInterval(async () => {
        try {
          const res = await window.sshthing.authPollSignIn(sessionId, pollSecret);
          if (res.status === 'completed') {
            stopPoll();
            navigate('/hosts', { replace: true });
          } else if (res.status === 'expired') {
            stopPoll();
            setState('expired');
          }
          // 'pending' → keep polling
        } catch (err: unknown) {
          stopPoll();
          const e = err as Error;
          setErrorMsg(e.message ?? 'Sign-in check failed');
          setState('error');
        }
      }, intervalMs);
    },
    [navigate, stopPoll]
  );

  const handleSignIn = useCallback(async () => {
    setState('starting');
    setErrorMsg('');
    try {
      const started = await window.sshthing.authStartSignIn();
      // Cache the URL so the user can re-open the browser without restarting.
      browserUrlRef.current = started.url;
      // Daemon opens the browser.
      await window.sshthing.authOpenBrowser(started.url);
      setState('polling');
      const intervalMs = (started.pollIntervalSeconds > 0 ? started.pollIntervalSeconds : 2) * 1000;
      startPoll(started.sessionId, started.pollSecret, intervalMs);
    } catch (err: unknown) {
      const e = err as Error;
      const msg = e.message ?? 'Failed to start sign-in';
      setErrorMsg(msg);
      setState('error');
      toast.error(msg);
    }
  }, [startPoll]);

  const handleCancel = useCallback(() => {
    stopPoll();
    browserUrlRef.current = null;
    // Best-effort sign-out to clear any pending daemon auth state.
    window.sshthing.authSignOut().catch(() => {/* ignore — may have nothing pending */});
    setState('idle');
  }, [stopPoll]);

  const handleReopenBrowser = useCallback(() => {
    const url = browserUrlRef.current;
    if (!url) return;
    window.sshthing.authOpenBrowser(url).catch(() => {/* ignore */});
  }, []);

  // Clean up on unmount.
  useEffect(() => {
    return () => stopPoll();
  }, [stopPoll]);

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        gap: 20,
        padding: 32,
      }}
    >
      <div style={{ textAlign: 'center', marginBottom: 8 }}>
        <h1
          style={{
            fontSize: 28,
            fontWeight: 800,
            letterSpacing: '-0.03em',
            color: 'var(--ink)',
            marginBottom: 6,
          }}
        >
          Sign in to SSHThing
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: 13 }}>
          Connect your SSHThing Cloud account for cross-device sync and team features.
        </p>
      </div>

      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
          width: '100%',
          maxWidth: 340,
          alignItems: 'center',
        }}
      >
        {state === 'idle' && (
          <Button variant="primary" onClick={handleSignIn} style={{ width: '100%' }}>
            Sign in with browser
          </Button>
        )}

        {state === 'starting' && (
          <Button variant="primary" loading style={{ width: '100%' }}>
            Opening browser…
          </Button>
        )}

        {state === 'polling' && (
          <>
            <div
              style={{
                background: 'var(--paper-2)',
                border: '1.5px solid var(--line)',
                borderRadius: 6,
                padding: '16px 20px',
                textAlign: 'center',
                width: '100%',
              }}
            >
              <p style={{ fontSize: 13, color: 'var(--ink)', margin: 0, marginBottom: 4 }}>
                Waiting for browser sign-in…
              </p>
              <p style={{ fontSize: 12, color: 'var(--muted)', margin: 0 }}>
                Complete sign-in in your browser, then return here.
              </p>
            </div>
            <Button variant="ghost" onClick={handleReopenBrowser} style={{ width: '100%' }}>
              Re-open browser
            </Button>
            <Button variant="ghost" onClick={handleCancel} style={{ width: '100%' }}>
              Cancel
            </Button>
          </>
        )}

        {state === 'expired' && (
          <>
            <p style={{ color: 'var(--danger)', fontSize: 13, textAlign: 'center', margin: 0 }}>
              Sign-in session expired. Please try again.
            </p>
            <Button variant="primary" onClick={handleSignIn} style={{ width: '100%' }}>
              Try again
            </Button>
          </>
        )}

        {state === 'error' && (
          <>
            {errorMsg && (
              <p style={{ color: 'var(--danger)', fontSize: 13, textAlign: 'center', margin: 0 }}>
                {errorMsg}
              </p>
            )}
            <Button variant="primary" onClick={handleSignIn} style={{ width: '100%' }}>
              Try again
            </Button>
          </>
        )}

        <button
          type="button"
          onClick={() => navigate(-1)}
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--muted)',
            cursor: 'pointer',
            fontSize: 12,
            marginTop: 4,
          }}
        >
          Back
        </button>
      </div>
    </div>
  );
}
