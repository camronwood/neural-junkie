import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    this.setState({
      error,
      errorInfo,
    });

    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="min-h-screen bg-slack-bg flex flex-col justify-center py-12 px-4">
          <div className="mx-auto w-full max-w-md">
            <div className="rounded-xl border border-slack-border bg-slack-bgHover shadow-2xl px-6 py-8">
              <div className="text-center">
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-500/15 ring-1 ring-red-500/40">
                  <svg
                    className="h-6 w-6 text-red-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    aria-hidden
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
                    />
                  </svg>
                </div>
                <h2 className="mt-5 text-center text-xl font-semibold text-slack-text">
                  Something went wrong
                </h2>
                <p className="mt-2 text-center text-sm text-slack-textMuted leading-relaxed">
                  An unexpected error occurred. Try again or reload the app.
                </p>
              </div>

              <div className="mt-6 rounded-lg border border-red-500/30 bg-red-950/40 p-4">
                <div className="flex gap-3">
                  <div className="flex-shrink-0 pt-0.5">
                    <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20" aria-hidden>
                      <path
                        fillRule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                  <div className="min-w-0 flex-1">
                    <h3 className="text-sm font-medium text-red-200">Error</h3>
                    <p className="mt-1 font-mono text-xs text-red-100/90 break-all leading-relaxed">
                      {this.state.error?.message || 'Unknown error'}
                    </p>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex gap-3">
                <button
                  type="button"
                  onClick={this.handleRetry}
                  className="flex-1 rounded-md bg-slack-accent py-2.5 px-4 text-sm font-medium text-white shadow hover:bg-slack-accentHover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
                >
                  Try again
                </button>
                <button
                  type="button"
                  onClick={this.handleReload}
                  className="flex-1 rounded-md border border-slack-border bg-slack-bg py-2.5 px-4 text-sm font-medium text-slack-text hover:bg-slack-bgHover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
                >
                  Reload
                </button>
              </div>

              {import.meta.env.DEV && this.state.errorInfo && (
                <details className="mt-6 rounded-lg border border-slack-border bg-slack-bg p-3">
                  <summary className="cursor-pointer text-xs font-medium text-slack-textMuted">
                    Stack trace (development)
                  </summary>
                  <pre className="mt-2 max-h-40 overflow-auto rounded bg-slack-bg text-[11px] text-slack-textMuted">
                    {this.state.errorInfo.componentStack}
                  </pre>
                </details>
              )}
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
