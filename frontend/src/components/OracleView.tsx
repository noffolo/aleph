import React, { useEffect, useRef } from 'react';
import { useStore } from '../store/useStore';
import { nlpClient } from '../api/factory';
import { Zap, TrendingUp, AlertTriangle, Clock, ThumbsUp, ThumbsDown, MessageSquareText } from 'lucide-react';
import { t } from '../i18n';

interface Prediction {
  entityId: string;
  probability: number;
  predictedState: string;
  explanation: string;
}

export const OracleView: React.FC<{inline?:boolean}> = ({inline=false}) => {
  const { projectID, predictions, setPredictions } = useStore();
  const [isLoading, setIsLoading] = React.useState(false);
  const [sentimentText, setSentimentText] = React.useState('');
  const [sentimentResult, setSentimentResult] = React.useState<{ score: number; label: string } | null>(null);
  const [sentimentLoading, setSentimentLoading] = React.useState(false);
  const [feedbackGiven, setFeedbackGiven] = React.useState<Record<string, boolean>>({});
  const abortRef = useRef<AbortController | null>(null);
  const predsRef = useRef<Prediction[]>([]);

  useEffect(() => {
    if (!projectID) return;

    abortRef.current?.abort();
    const ac = new AbortController();
    abortRef.current = ac;

    const fetchPredictions = async () => {
      setIsLoading(true);
      predsRef.current = [];
      setPredictions([]);
      try {
        for await (const res of nlpClient.streamPredictions({ contextId: projectID, ontologyQuery: "*" }, { signal: ac.signal })) {
          if (ac.signal.aborted) break;
          const p: Prediction = {
            entityId: (res as any).entityId || '',
            probability: (res as any).probability || 0,
            predictedState: (res as any).predictedState || '',
            explanation: (res as any).explanation || ''
          };
          predsRef.current = [...predsRef.current, p];
          setPredictions(predsRef.current);
        }
      } catch (err: any) {
        if (err?.code !== 'CANCELLED') {
          console.error(t('errors.default'), err);
        }
      } finally {
        if (!ac.signal.aborted) setIsLoading(false);
      }
    };

    fetchPredictions();
    return () => ac.abort();
  }, [projectID]);

  const handleFeedback = (pred: Prediction, isCorrect: boolean) => {
    setFeedbackGiven(prev => ({ ...prev, [pred.entityId]: isCorrect }));
    nlpClient.recordFeedback({
      entityId: pred.entityId,
      isCorrect,
      feedbackType: 'prediction',
    }).catch((err: any) => {
      setFeedbackGiven(prev => {
        const next = { ...prev };
        delete next[pred.entityId];
        return next;
      });
      useStore.getState().setLastError(`Feedback non inviato: ${err?.message || 'errore di rete'}`)
      setTimeout(() => useStore.getState().setLastError(null), 4000)
    });
  };

  const handleAnalyzeSentiment = async () => {
    if (!sentimentText.trim()) return;
    setSentimentLoading(true);
    try {
      const res = await nlpClient.analyzeSentiment({ text: sentimentText }) as any;
      setSentimentResult({ score: res.score || 0, label: (res.label || 'neutral').toLowerCase() });
    } catch (err: any) {
      setSentimentResult({ score: 0, label: 'error' });
      useStore.getState().setLastError(`Analisi sentiment fallita: ${err?.message || 'errore'}`)
      setTimeout(() => useStore.getState().setLastError(null), 4000)
    } finally {
      setSentimentLoading(false);
    }
  };

  return (
    <div className={(inline ? '' : 'max-w-6xl mx-auto ') + 'space-y-12 pb-24 animate-in fade-in duration-700'}>
      <header className="flex flex-col space-y-4">
        <div className="flex items-center space-x-3 text-primary">
          <Zap size={32} className="fill-current" />
          <h2 className="text-4xl font-black tracking-tighter uppercase italic">{t('oracle.title')}</h2>
        </div>
        <p className="text-textMuted font-medium max-w-2xl">
          Analisi degli scenari probabilistici generati dal sistema.
          Le predizioni sono calibrate in tempo reale con i segnali disponibili.
        </p>
      </header>

      {isLoading && predictions.length === 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {[1, 2].map(i => (
            <div key={i} className="h-80 bg-surface-alt rounded-lg animate-pulse"></div>
          ))}
        </div>
      )}

      {predictions.length === 0 && !isLoading && (
        <div className="py-20 text-center">
          <Zap size={48} className="mx-auto text-textDim mb-4" />
          <p className="text-textMuted font-bold text-sm">{t('oracle.empty')}</p>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {predictions.map((pred: any, i: number) => (
          <div key={i} className="bg-surface rounded-lg p-10 border border-border shadow-lg shadow-primary/5 flex flex-col justify-between group hover:border-primary/30 transition-all duration-500 animate-in slide-in-from-bottom-4">
            <div className="space-y-6">
              <div className="flex justify-between items-start">
                <div className={`p-4 rounded-lg ${pred.predictedState === 'ACTION_REQUIRED' ? 'bg-warning/10 text-warning' : 'bg-primary/10 text-primary'}`}>
                  {pred.predictedState === 'ACTION_REQUIRED' ? <AlertTriangle size={24} /> : <TrendingUp size={24} />}
                </div>
                <div className="flex flex-col items-end">
                  <span className="text-[10px] font-black text-textMuted uppercase tracking-widest mb-1">Indice di Affidabilità</span>
                  <span className="text-4xl font-black text-text">{(pred.probability * 100).toFixed(0)}%</span>
                  <span className={`text-[9px] font-medium ${
                    pred.probability > 0.8 ? 'text-success' :
                    pred.probability >= 0.5 ? 'text-yellow-500' :
                    'text-danger'
                  }`}>
                    {pred.probability > 0.8 ? 'Alta affidabilità' :
                     pred.probability >= 0.5 ? 'Affidabilità media' :
                     'Bassa affidabilità'}
                  </span>
                </div>
              </div>

              <div>
                <h3 className="text-2xl font-black text-text tracking-tight leading-tight mb-2 uppercase italic">{pred.entityId.replace(/_/g, ' ')}</h3>
                <p className="text-textMuted font-medium leading-relaxed">{pred.explanation}</p>
              </div>
            </div>

            <div className="mt-10 pt-8 border-t border-border flex items-center justify-between">
               <div className="flex items-center space-x-2 text-textMuted">
                  <Clock size={14} />
                  <span className="text-[10px] font-black uppercase tracking-widest">Calcolato ora</span>
               </div>
                <div className="flex items-center space-x-1">
                   <button
                     onClick={() => handleFeedback(pred, true)}
                     disabled={feedbackGiven[pred.entityId] === true}
                     className={`p-2 rounded-xl transition-all ${feedbackGiven[pred.entityId] === true ? 'bg-success/10 text-success' : 'hover:bg-success/10 text-textDim hover:text-success'}`}
                      title={t('oracle.correctPrediction')}
                   >
                     <ThumbsUp size={16} />
                   </button>
                   <button
                     onClick={() => handleFeedback(pred, false)}
                     disabled={feedbackGiven[pred.entityId] === false}
                     className={`p-2 rounded-xl transition-all ${feedbackGiven[pred.entityId] === false ? 'bg-danger/10 text-danger' : 'hover:bg-danger/10 text-textDim hover:text-danger'}`}
                      title={t('oracle.wrongPrediction')}
                   >
                     <ThumbsDown size={16} />
                   </button>
                </div>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-12 bg-surface rounded-lg p-10 border border-border shadow-lg">
        <div className="flex items-center space-x-3 mb-6">
          <div className="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center text-primary">
            <MessageSquareText size={20} />
          </div>
          <h3 className="text-2xl font-black tracking-tight uppercase italic">{t('oracle.sentimentTitle')} <span className="text-sm font-normal text-textMuted align-middle">(beta)</span></h3>
        </div>
        <p className="text-textMuted text-sm mb-6">{t('oracle.sentimentSubtitle')}</p>
        <div className="flex space-x-3">
          <textarea
            value={sentimentText}
            onChange={(e) => setSentimentText(e.target.value)}
            placeholder={t('oracle.sentimentPlaceholder')}
            className="flex-1 h-24 p-4 bg-surface-alt rounded-lg border border-border text-sm resize-none focus:outline-none focus:border-primary/50 focus:ring-2 focus:ring-primary/10 transition-all"
          />
          <button
            onClick={handleAnalyzeSentiment}
            disabled={sentimentLoading || !sentimentText.trim()}
            className="px-6 bg-primary text-white rounded-lg font-bold text-sm hover:bg-primary/90 transition-all disabled:opacity-50 disabled:cursor-not-allowed self-end"
          >
            {sentimentLoading ? 'Analisi...' : 'Analizza'}
          </button>
        </div>
        {sentimentResult && (
          <div className="mt-6 flex items-center space-x-6">
            {sentimentResult.score < 0 ? (
              <div className="p-4 rounded-lg bg-surface-alt text-textMuted">
                <span className="text-3xl font-black">N/D</span>
                <div className="text-[9px] text-textMuted mt-1">Analisi non disponibile</div>
              </div>
            ) : (
            <div className={`p-4 rounded-lg ${sentimentResult.label === 'positive' ? 'bg-success/10 text-success' : sentimentResult.label === 'negative' ? 'bg-danger/10 text-danger' : 'bg-surface-alt text-textMuted'}`}>
              <span className="text-3xl font-black">{(sentimentResult.score * 100).toFixed(0)}% ±8%</span>
              <div className="text-[9px] text-textMuted mt-1">Intervallo di confidenza approssimativo</div>
            </div>
            )}
            {sentimentResult.score >= 0 && (
            <div>
              <div className="text-[10px] font-black uppercase tracking-widest text-textMuted mb-1">Sentimento</div>
              <div className={`text-xl font-black capitalize ${sentimentResult.label === 'positive' ? 'text-success' : sentimentResult.label === 'negative' ? 'text-danger' : 'text-textMuted'}`}>
                {sentimentResult.label === 'error' ? 'Errore di analisi' : sentimentResult.label === 'positive' ? 'Positivo' : sentimentResult.label === 'negative' ? 'Negativo' : 'Neutrale'}
              </div>
              <div className={`text-[9px] mt-1 font-medium ${
                sentimentResult.score >= 0.7 || sentimentResult.score <= 0.3 ? 'text-success' :
                sentimentResult.score > 0.55 || sentimentResult.score < 0.45 ? 'text-yellow-500' :
                'text-danger'
              }`}>
                {sentimentResult.score >= 0.7 || sentimentResult.score <= 0.3 ? 'Alta confidenza' :
                 sentimentResult.score > 0.55 || sentimentResult.score < 0.45 ? 'Confidenza media' :
                 'Bassa confidenza'}
              </div>
            </div>
            )}
            {sentimentResult.score >= 0 && (
            <div className="ml-auto">
              <div className="text-[10px] font-black uppercase tracking-widest text-textMuted mb-1">Punteggio</div>
              <div className="text-sm font-mono text-textMuted">{sentimentResult.score.toFixed(4)}</div>
            </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
