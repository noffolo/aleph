import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Binary } from 'lucide-react';
import { useStore } from '../store/useStore';
import { t } from '../i18n';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class AlephErrorBoundary extends Component<Props, State> {
  public state: State = { hasError: false };

  public static getDerivedStateFromError(_: Error): State {
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    if (import.meta.env.DEV) {
      console.error("Uncaught error:", error, errorInfo);
    }
  }

  private handleRetry = () => {
    try {
      const store = useStore.getState();
      store.setData(null);
      store.setPredictions([]);
      store.setLastError(null);
      store.setCurrentView('copilot');
    } catch (e) {
      console.error('AlephErrorBoundary.handleRetry failed:', e)
    }
    this.setState({ hasError: false });
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full bg-surface-alt p-8">
          <div className="flex items-center justify-center w-20 h-20 rounded-2xl bg-primary text-white mb-8 shadow-lg">
            <Binary size={40} />
          </div>
<h2 className="text-2xl font-bold text-text mb-2">{t('errors.rawMode')}</h2>
           <p className="text-textMuted text-center max-w-md mb-8">{t('errors.rawModeDesc')}</p>
<button
             onClick={this.handleRetry}
             className="px-8 py-3 bg-primary text-white rounded-xl font-bold hover:bg-primary/90 transition-all shadow-lg"
           >
             {t('toast.retry')}
           </button>
        </div>
      );
    }

    return this.props.children;
  }
}
