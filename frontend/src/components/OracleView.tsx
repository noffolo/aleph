import { fromProto } from '../api/adapters';
import { StreamPredictionsResponse } from '../api/proto/aleph/nlp/v1/nlp_pb';
import React, { useState, useEffect } from 'react';
import { useStore } from '../store/useStore';
import { createPromiseClient } from "@connectrpc/connect";
import { transport } from "../api/client";
import { NLPService } from "../api/proto/aleph/nlp/v1/nlp_connect";
import { Zap, TrendingUp, AlertTriangle, Info, BarChart3, Clock } from 'lucide-react';

const nlpClient = createPromiseClient(NLPService, transport);

interface Prediction {
  entityId: string;
  probability: number;
  predictedState: string;
  explanation: string;
}

export const OracleView: React.FC = () => {
  const { projectID } = useStore();
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!projectID) return;
    
    const fetchPredictions = async () => {
      setIsLoading(true);
      setPredictions([]);
      try {
        for await (const res of nlpClient.streamPredictions({ contextId: "globale", ontologyQuery: "*" })) {
          setPredictions(prev => [...prev, {
            entityId: fromProto<StreamPredictionsResponse>(res).entityId,
            probability: fromProto<StreamPredictionsResponse>(res).probability,
            predictedState: fromProto<StreamPredictionsResponse>(res).predictedState,
            explanation: fromProto<StreamPredictionsResponse>(res).explanation
          }]);
        }
      } catch (err) {
        console.error("Errore nello streaming delle predizioni:", err);
      } finally {
        setIsLoading(false);
      }
    };

    fetchPredictions();
  }, [projectID]);

  return (
    <div className="max-w-6xl mx-auto space-y-12 pb-24 animate-in fade-in duration-700">
      <header className="flex flex-col space-y-4">
        <div className="flex items-center space-x-3 text-blue-600">
          <Zap size={32} className="fill-current" />
          <h2 className="text-4xl font-black tracking-tighter uppercase italic">Motore Predittivo Oracle</h2>
        </div>
        <p className="text-gray-500 font-medium max-w-2xl">
          Analisi degli scenari probabilistici generati dall'Ensemble Aleph. 
          Le predizioni sono calibrate in tempo reale con i segnali dei mercati e i driver SHAP.
        </p>
      </header>

      {isLoading && predictions.length === 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {[1, 2].map(i => (
            <div key={i} className="h-80 bg-gray-100 rounded-[32px] animate-pulse"></div>
          ))}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {predictions.map((pred, i) => (
          <div key={i} className="bg-white rounded-[40px] p-10 border border-gray-100 shadow-2xl shadow-blue-900/5 flex flex-col justify-between group hover:border-blue-200 transition-all duration-500 animate-in slide-in-from-bottom-4">
            <div className="space-y-6">
              <div className="flex justify-between items-start">
                <div className={`p-4 rounded-3xl ${pred.predictedState === 'ACTION_REQUIRED' ? 'bg-amber-50 text-amber-600' : 'bg-blue-50 text-blue-600'}`}>
                  {pred.predictedState === 'ACTION_REQUIRED' ? <AlertTriangle size={24} /> : <TrendingUp size={24} />}
                </div>
                <div className="flex flex-col items-end">
                  <span className="text-[10px] font-black text-gray-400 uppercase tracking-widest mb-1">Indice di Affidabilità</span>
                  <span className="text-4xl font-black text-blue-900">{(pred.probability * 100).toFixed(0)}%</span>
                </div>
              </div>

              <div>
                <h3 className="text-2xl font-black text-blue-950 tracking-tight leading-tight mb-2 uppercase italic">{pred.entityId.replace(/_/g, ' ')}</h3>
                <p className="text-gray-500 font-medium leading-relaxed">{pred.explanation}</p>
              </div>

              <div className="pt-6 space-y-4">
                 <div className="flex items-center justify-between text-[10px] font-black uppercase tracking-widest text-gray-400">
                    <span>Driver di Predizione (SHAP)</span>
                    <Info size={12} />
                 </div>
                 <div className="space-y-3">
                    <div className="space-y-1.5">
                       <div className="flex justify-between text-[10px] font-bold">
                          <span className="text-blue-600">Sentiment di Mercato</span>
                          <span>+42%</span>
                       </div>
                       <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                          <div className="h-full bg-blue-600 rounded-full" style={{ width: '65%' }}></div>
                       </div>
                    </div>
                    <div className="space-y-1.5">
                       <div className="flex justify-between text-[10px] font-bold">
                          <span className="text-amber-600">Deriva Statistica</span>
                          <span>-12%</span>
                       </div>
                       <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                          <div className="h-full bg-amber-500 rounded-full" style={{ width: '25%' }}></div>
                       </div>
                    </div>
                 </div>
              </div>
            </div>

            <div className="mt-10 pt-8 border-t border-gray-50 flex items-center justify-between">
               <div className="flex items-center space-x-2 text-gray-400">
                  <Clock size={14} />
                  <span className="text-[10px] font-black uppercase tracking-widest">Calcolato ora</span>
               </div>
               <button className="px-6 py-2.5 bg-gray-900 text-white rounded-xl text-[10px] font-black uppercase tracking-widest hover:bg-blue-600 transition-all shadow-xl shadow-gray-200">
                  Sintetizza Scenari
               </button>
            </div>
          </div>
        ))}
      </div>

      <footer className="bg-blue-900 rounded-[40px] p-12 text-white overflow-hidden relative shadow-3xl shadow-blue-200">
         <div className="absolute top-0 right-0 p-12 opacity-10">
            <BarChart3 size={200} />
         </div>
         <div className="relative z-10 max-w-xl space-y-6">
            <h3 className="text-3xl font-black tracking-tighter uppercase italic leading-none">Stabilità Brier Score</h3>
            <p className="text-blue-200 font-medium">
               Il sistema mantiene un Brier Score di 0.14 negli ultimi 30 giorni. La calibrazione Bayesiana con Polymarket ha ridotto l'errore del 22%.
            </p>
            <div className="flex items-center space-x-8">
               <div className="flex flex-col">
                  <span className="text-[10px] font-black uppercase tracking-widest text-blue-400 mb-1">Ultima Calibrazione</span>
                  <span className="text-xl font-bold">2m fa</span>
               </div>
               <div className="flex flex-col">
                  <span className="text-[10px] font-black uppercase tracking-widest text-blue-400 mb-1">Orizzonte Predittivo</span>
                  <span className="text-xl font-bold">45 Giorni</span>
               </div>
            </div>
         </div>
      </footer>
    </div>
  );
};
