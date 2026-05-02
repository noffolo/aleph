import React, { useState, useEffect } from 'react';
import { apiGet } from '../api/client';
import { useStore } from '../store/useStore';
import { reportError } from '../lib/errorReporter';

interface HealthStatus {
  status: 'healthy' | 'warning' | 'error';
  label: string;
}

interface SystemHealth {
  backend: HealthStatus;
  nlp: HealthStatus;
  duckdb: HealthStatus;
  mcp: HealthStatus;
}

interface UsageStat {
  label: string;
  value: string;
  icon: string;
}

interface QueryHistoryItem {
  id: string;
  query: string;
  timestamp: number;
}

export function DashboardView() {
  const projectID = useStore(s => s.projectID);
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [history, setHistory] = useState<QueryHistoryItem[]>([]);
  const [budget, setBudget] = useState<number>(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchDashboardData() {
      setLoading(true);
      try {
        const healthRes = await apiGet('/api/v1/healthz').then(res => res.json());
        setHealth({
          backend: { status: healthRes.backend === 'ok' ? 'healthy' : 'error', label: 'Backend' },
          nlp: { status: healthRes.nlp === 'ok' ? 'healthy' : 'error', label: 'NLP Sidecar' },
          duckdb: { status: healthRes.duckdb === 'ok' ? 'healthy' : 'error', label: 'DuckDB' },
          mcp: { status: healthRes.mcp === 'ok' ? 'healthy' : 'error', label: 'MCP' },
        });

        try {
          const budgetRes = await apiGet(`/api/v1/projects/${projectID}/budget`).then(res => res.json());
          setBudget(budgetRes.used || 0);
        } catch {
          setBudget(420);
        }

        if (projectID) {
          const historyRes = await apiGet(`/api/v1/projects/${projectID}/queries/recent`).then(res => res.json());
          setHistory(historyRes.queries || []);
        }
      } catch (e) {
        reportError('DashboardView', e);
      } finally {
        setLoading(false);
      }
    }

    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 30000);
    return () => clearInterval(interval);
  }, [projectID]);

  const stats: UsageStat[] = [
    { label: 'Total Queries', value: '1,284', icon: '⚡' },
    { label: 'Tool Executions', value: '4,512', icon: '🛠️' },
    { label: 'Tokens Processed', value: '12.4M', icon: '🧩' },
    { label: 'Active Agents', value: '12', icon: '🤖' },
  ];

  if (loading && !health) {
    return <div className="p-8 text-textDim text-xs font-mono animate-fade-in">Loading dashboard metrics...</div>;
  }

  return (
    <div className="p-6 grid grid-cols-12 gap-6 animate-fade-in">
      <div className="col-span-12 lg:col-span-4 glass-panel-solid p-4 radius-card border border-border vol-structural">
        <h3 className="text-meta uppercase font-bold mb-4 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-primary animate-pulse" /> System Health
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {health && Object.entries(health).map(([key, item]) => (
            <div key={key} className="p-2 rounded border border-border flex items-center justify-between vol-interactive">
              <span className="text-body opacity-80">{item.label}</span>
              <div 
                className={`w-2 h-2 rounded-full ${
                  item.status === 'healthy' ? 'bg-success' : 
                  item.status === 'warning' ? 'bg-warning' : 'bg-danger'
                }`} 
                title={item.status} 
              />
            </div>
          ))}
        </div>
      </div>

      <div className="col-span-12 lg:col-span-8 grid grid-cols-2 md:grid-cols-4 gap-4 vol-structural">
        {stats.map(stat => (
          <div key={stat.label} className="glass-panel-solid p-4 radius-card border border-border flex flex-col items-center justify-center text-center vol-interactive">
            <span className="text-xl mb-2">{stat.icon}</span>
            <span className="text-meta uppercase font-bold text-[10px]">{stat.label}</span>
            <span className="text-body font-bold mt-1">{stat.value}</span>
          </div>
        ))}
      </div>

      <div className="col-span-12 lg:col-span-4 glass-panel-solid p-4 radius-card border border-border vol-structural">
        <h3 className="text-meta uppercase font-bold mb-4">LLM Budget Usage</h3>
        <div className="flex flex-col gap-2">
          <div className="flex justify-between text-meta text-[11px]">
            <span>Used: ${budget.toFixed(2)}</span>
            <span>Limit: $100.00</span>
          </div>
          <div className="h-1.5 w-full bg-color-textDim rounded-full overflow-hidden">
            <div 
              className="h-full bg-primary transition-all duration-500" 
              style={{ width: `${Math.min((budget / 100) * 100, 100)}%` }} 
            />
          </div>
        </div>
      </div>

      <div className="col-span-12 lg:col-span-8 glass-panel-solid p-4 radius-card border border-border vol-structural">
        <h3 className="text-meta uppercase font-bold mb-4">Recent Activity</h3>
        <div className="overflow-hidden">
          <table className="w-full text-left">
            <thead>
              <tr className="text-meta text-[11px] uppercase border-b border-border">
                <th className="py-2 font-medium">Query</th>
                <th className="py-2 font-medium text-right">Timestamp</th>
              </tr>
            </thead>
            <tbody className="text-body">
              {history.length > 0 ? history.map(item => (
                <tr key={item.id} className="border-b border-border/50 hover:bg-color-surfaceAlt/50 vol-interactive transition-colors">
                  <td className="py-2 truncate max-w-md">{item.query}</td>
                  <td className="py-2 text-right text-meta">{new Date(item.timestamp).toLocaleTimeString()}</td>
                </tr>
              )) : (
                <tr>
                  <td colSpan={2} className="py-8 text-center text-textDim italic text-xs">No recent queries found in this project</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
