import React, { Component, ErrorInfo, ReactNode } from 'react';

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
    console.error("Uncaught error:", error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="p-4 bg-red-900 text-white rounded">
          <h2 className="font-bold">Modalità Raw (Servizio Degradato)</h2>
          <p>Il sistema predittivo è temporaneamente offline. Visualizzazione dati grezzi attiva.</p>
        </div>
      );
    }

    return this.props.children;
  }
}
