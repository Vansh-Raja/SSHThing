/**
 * ErrorBoundary — catches render-time errors in the wrapped sub-tree and
 * shows a user-friendly "Something broke" fallback with a Reload button.
 *
 * Must be a class component because React hooks cannot catch render errors.
 */
import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  message: string;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, message: '' };
  }

  static getDerivedStateFromError(error: unknown): State {
    const message =
      error instanceof Error ? error.message : String(error);
    return { hasError: true, message };
  }

  componentDidCatch(error: unknown, info: ErrorInfo): void {
    // Log to console; in a real app we'd send to a crash-reporting service.
    console.error('[ErrorBoundary] caught error:', error);
    console.error('[ErrorBoundary] component stack:', info.componentStack);
  }

  handleReload = (): void => {
    window.location.reload();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            gap: 16,
            padding: 32,
            textAlign: 'center',
          }}
        >
          <div style={{ fontSize: 32, lineHeight: 1 }}>⚠</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--ink)' }}>
            Something broke
          </div>
          {this.state.message && (
            <div
              style={{
                fontSize: 12,
                color: 'var(--muted)',
                fontFamily: 'var(--font-mono)',
                background: 'var(--paper-2)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius)',
                padding: '8px 12px',
                maxWidth: 480,
                wordBreak: 'break-word',
              }}
            >
              {this.state.message}
            </div>
          )}
          <button
            type="button"
            className="btn btn--primary"
            onClick={this.handleReload}
          >
            Reload
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
