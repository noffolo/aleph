import { Component, type ErrorInfo, type ReactNode } from 'react';
import { t } from '../i18n';

interface Props {
  children: ReactNode;
  label?: string;
}

interface State {
  hasError: boolean;
}

export class InlineErrorBoundary extends Component<Props, State> {
  public state: State = { hasError: false };

  public static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error(`InlineErrorBoundary (${this.props.label || 'unknown'}):`, error, errorInfo);
  }

  private handleRetry = () => {
    this.setState({ hasError: false });
  };

  public render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full bg-surface border border-danger/30 rounded-lg p-6 font-mono text-sm text-danger">
          <span className="text-[10px] tracking-widest uppercase leading-snug mb-2">Errore nel pannello</span>
          <p className="text-textMuted text-xs mb-4">Si è verificato un errore in questo componente.</p>
          <button
            onClick={this.handleRetry}
            className="px-4 py-2 bg-danger/10 text-danger rounded border border-danger/30 hover:bg-danger/20 transition-all"
          >
            Riprova
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}