import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  error: Error | null;
}

export class NeuralCanvasErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Neural Canvas renderer failed', error, info);
  }

  componentDidUpdate(previousProps: Props) {
    if (this.state.error && previousProps.children !== this.props.children) {
      this.setState({ error: null });
    }
  }

  render() {
    if (!this.state.error) return this.props.children;
    if (this.props.fallback) return this.props.fallback;

    return (
      <div
        role="alert"
        className="rounded border border-red-500/30 bg-red-950/30 p-4 text-sm text-red-200"
      >
        This artifact could not be rendered safely.
      </div>
    );
  }
}
