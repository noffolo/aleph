import React, { useMemo, useState, useEffect } from 'react';
import { Activity, Users, Shield, Star, AlertTriangle, CheckCircle2, BarChart3 } from 'lucide-react';
import type { ToolIntel, ToolAnomaly } from '../store/types';
import { useStore } from '../store/useStore';
import { apiGet, apiPost, apiPatch } from '../api/client';
import { reportError } from '../lib/errorReporter';

const Card = ({ title, icon: Icon, children, className = "" }: { title: string, icon: React.ElementType, children: React.ReactNode, className?: string }) => (
  <div className={`bg-[#0e0e18] border border-[#2a2a3a] rounded-lg p-4 flex flex-col ${className}`}>
    <div className="flex items-center gap-2 mb-4 text-[#e0e0e0] font-medium border-b border-[#2a2a3a] pb-2">
      <Icon size={16} className="text-[#60a5fa]" />
      <span className="text-xs uppercase tracking-wider">{title}</span>
    </div>
    <div className="flex-1">{children}</div>
  </div>
);

const StatLabel = ({ label, value }: { label: string, value: string | number }) => (
  <div className="flex justify-between text-xs py-1 border-b border-[#1d1d2c] last:border-0">
    <span className="text-[#88889b]">{label}</span>
    <span className="text-[#e0e0e0] font-mono">{value}</span>
  </div>
);

export default function ToolIntelligenceView() {
  const [tools, setTools] = useState<ToolIntel[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchToolIntel() {
      try {
        const data = await apiGet('/api/v1/tools/intelligence?tool_id=all');
        setTools(Array.isArray(data) ? data : [data]);
      } catch (e) {
        reportError('ToolIntelligenceView', e);
      } finally {
        setLoading(false);
      }
    }
    fetchToolIntel();
  }, []);

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center text-[#88889b] text-xs italic">
        Loading intelligence data...
      </div>
    );
  }

  if (tools.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-[#88889b] text-xs italic">
        No tool data available
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col gap-4 p-4 overflow-auto bg-[#080810] text-[#e0e0e0]">
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 h-fit">
        <Card title="CodeFlow Analysis" icon={Activity}>
          <div className="space-y-4">
            <div className="bg-[#141420] rounded border border-[#2a2a3a] p-2 h-48 overflow-hidden relative">
               <svg width="100%" height="100%" viewBox="0 0 200 150">
                  {tools.slice(0, 4).map((t, i) => (
                    <g key={t.id}>
                      <circle cx={(i * 45) + 30} cy={(i % 2 === 0 ? 30 : 100)} r="6" fill="#60a5fa" />
                      <text x={(i * 45) + 30} y={(i % 2 === 0 ? 20 : 115)} textAnchor="middle" fill="#88889b" fontSize="8" className="font-mono">
                        {t.name.slice(0, 8)}
                      </text>
                      {i < 3 && (
                        <line 
                          x1={(i * 45) + 30} y1={(i % 2 === 0 ? 30 : 100)} 
                          x2={((i+1) * 45) + 30} y2={((i+1) % 2 === 0 ? 30 : 100)} 
                          stroke="#2a2a3a" strokeWidth="1" 
                        />
                      )}
                    </g>
                  ))}
                </svg>
            </div>
            <div className="space-y-2">
              {tools.slice(0, 3).map(t => (
                <div key={t.id} className="p-2 bg-[#141420] rounded border border-[#2a2a3a] text-[10px]">
                  <div className="font-bold text-[#60a5fa] mb-1">{t.name}</div>
                  <div className="flex justify-between">
                    <StatLabel label="Execs" value={t.execCount} />
                    <StatLabel label="Avg ms" value={t.avgDuration} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>

        <Card title="Usage Patterns" icon={Users}>
          <div className="space-y-4">
            <div className="space-y-2">
              {tools.slice(0, 4).map(t => (
                <div key={t.id} className="flex items-center gap-2">
                  <div className="text-[10px] w-20 truncate font-mono text-[#88889b]">{t.name}</div>
                  <div className="flex-1 bg-[#1d1d2c] h-2 rounded-full overflow-hidden">
                    <div 
                      className="bg-[#60a5fa] h-full" 
                      style={{ width: `${Math.min(100, (t.execCount / 200) * 100)}%` }} 
                    />
                  </div>
                  <div className="text-[10px] font-mono w-8 text-right">{t.execCount}</div>
                </div>
              ))}
            </div>
            <div className="mt-4 pt-4 border-t border-[#2a2a3a]">
              <div className="text-[10px] uppercase text-[#88889b] mb-2 font-medium">Top Users</div>
              <div className="flex flex-wrap gap-2">
                {Array.from(new Set(tools.flatMap(t => t.topUsers))).map(user => (
                  <span key={user} className="px-2 py-0.5 bg-[#141420] border border-[#2a2a3a] rounded-full text-[10px] text-[#e0e0e0] font-mono">
                    {user}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </Card>

        <Card title="Security Intelligence" icon={Shield}>
          <div className="space-y-4">
            {tools.slice(0, 3).map(t => (
              <div key={t.id} className="p-2 bg-[#141420] rounded border border-[#2a2a3a] space-y-2">
                <div className="flex justify-between items-center text-[10px]">
                  <span className="font-bold text-[#e0e0e0]">{t.name}</span>
                  <span className="font-mono text-[#88889b]">Risk: {t.riskScore}</span>
                </div>
                <div className="h-1.5 w-full bg-[#1d1d2c] rounded-full overflow-hidden">
                  <div 
                    className={`h-full transition-all duration-500 ${
                      t.riskScore <= 30 ? 'bg-green-500' : t.riskScore <= 60 ? 'bg-yellow-500' : 'bg-red-500'
                    }`} 
                    style={{ width: `${t.riskScore}%` }} 
                  />
                </div>
                {t.warnings.length > 0 && (
                  <div className="flex items-start gap-1 text-[10px] text-yellow-500">
                    <AlertTriangle size={10} className="mt-0.5 shrink-0" />
                    <span>{t.warnings[0]}</span>
                  </div>
                )}
              </div>
            ))}
          </div>
        </Card>
      </div>

      <div className="bg-[#0e0e18] border border-[#2a2a3a] rounded-lg p-4 flex-1 overflow-auto">
        <div className="flex items-center gap-2 mb-4 text-[#e0e0e0] font-medium border-b border-[#2a2a3a] pb-2">
          <Star size={16} className="text-[#60a5fa]" />
          <span className="text-xs uppercase tracking-wider">Cross-Context Recommendations</span>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {tools.map(t => (
            <div key={t.id} className="p-3 bg-[#141420] border border-[#2a2a3a] rounded-lg group hover:border-[#60a5fa] transition-colors">
              <div className="flex justify-between items-start mb-2">
                <div className="flex items-center gap-2">
                  <div className="p-1 bg-[#0e0e18] rounded border border-[#2a2a3a]">
                    <BarChart3 size={12} className="text-[#60a5fa]" />
                  </div>
                  <span className="text-xs font-bold">{t.name}</span>
                </div>
                <span className={`text-[10px] px-1.5 py-0.5 rounded font-mono ${
                  t.usageFreq === 'high' ? 'bg-blue-500/20 text-blue-400' : 'bg-gray-500/20 text-gray-400'
                }`}>
                  {t.usageFreq}
                </span>
              </div>
              <div className="space-y-1">
                {t.recommendations.map((rec, idx) => (
                  <div key={idx} className="flex items-start gap-1 text-xs text-[#88889b]">
                    <CheckCircle2 size={12} className="mt-0.5 text-green-500 shrink-0" />
                    <span>{rec}</span>
                  </div>
                ))}
                {t.anomalies.map((anom, idx) => (
                  <div key={idx} className="flex items-start gap-1 text-xs text-yellow-500">
                    <AlertTriangle size={12} className="mt-0.5 shrink-0" />
                    <span>{anom.desc}</span>
                  </div>
                ))}
                {t.relatedTools.length > 0 && (
                  <div className="flex items-center gap-1 text-[10px] text-[#60a5fa] mt-2 italic">
                    <span>Related: {t.relatedTools.join(', ')}</span>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
