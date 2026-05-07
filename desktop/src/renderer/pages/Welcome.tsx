/**
 * Welcome — first-run onboarding screen shown after unlocking when the user
 * has no hosts yet. Offers quick actions to get started.
 */
import { useNavigate } from 'react-router-dom';

export default function Welcome() {
  const navigate = useNavigate();

  const markSeen = () => {
    try {
      localStorage.setItem('sshthing-welcome-shown', 'true');
    } catch {
      // ignore
    }
  };

  const goToHosts = () => {
    markSeen();
    navigate('/hosts');
  };

  const goToTeams = () => {
    markSeen();
    navigate('/teams');
  };

  const goToSettings = () => {
    markSeen();
    navigate('/settings');
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        gap: 24,
        padding: 32,
        textAlign: 'center',
      }}
    >
      <div
        style={{
          width: 56,
          height: 56,
          borderRadius: 12,
          background: 'var(--accent)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 28,
        }}
      >
        ⬡
      </div>

      <div>
        <h1
          style={{
            fontSize: 24,
            fontWeight: 800,
            letterSpacing: '-0.02em',
            color: 'var(--ink)',
            margin: '0 0 6px',
          }}
        >
          Welcome to SSHThing
        </h1>
        <p style={{ fontSize: 13, color: 'var(--muted)', margin: 0, maxWidth: 360 }}>
          Your vault is ready. Let&apos;s get you connected to your first server.
        </p>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 10, width: '100%', maxWidth: 280 }}>
        <button type="button" className="btn btn--primary" onClick={goToHosts}>
          + Add your first host
        </button>
        <button type="button" className="btn btn--ghost" onClick={goToTeams}>
          Create or join a team
        </button>
        <button type="button" className="btn btn--ghost" onClick={goToSettings}>
          Open settings
        </button>
      </div>

      <button
        type="button"
        style={{
          background: 'none',
          border: 'none',
          color: 'var(--muted)',
          fontSize: 12,
          cursor: 'pointer',
          padding: '4px 0',
        }}
        onClick={goToHosts}
      >
        Skip for now
      </button>
    </div>
  );
}
