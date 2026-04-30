import React, { memo, useMemo } from 'react'
import { useStore } from '../store/useStore'
import { type Scenario } from '../store/types'
import { ArrowUp, ArrowDown, Minus, Info, AlertCircle } from 'lucide-react'

const ScenarioComparisonViewInner = () => {
  const scenarios = useStore(s => s.scenarios)
  const selectedScenarioIds = useStore(s => s.selectedScenarioIds)
  const setSelectedScenarioIds = useStore(s => s.setSelectedScenarioIds)
  const scenarioSelectorRef = React.useRef<HTMLDivElement>(null)

  const selectedScenarios = useMemo(() => 
    scenarios.filter(s => selectedScenarioIds.includes(s.id)),
    [scenarios, selectedScenarioIds]
  )

  const handleToggleScenario = (id: string) => {
    const current = new Set(selectedScenarioIds)
    if (current.has(id)) {
      current.delete(id)
    } else {
      if (current.size >= 3) return
      current.add(id)
    }
    setSelectedScenarioIds(Array.from(current))
  }

  if (scenarios.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8 text-center">
        <div className="w-12 h-12 rounded-full bg-surface-alt flex items-center justify-center mb-4">
          <Info size={24} className="text-textDim" />
        </div>
        <h3 className="text-text font-mono text-sm mb-2">No Scenarios Found</h3>
        <p className="text-textDim text-xs font-mono max-w-xs">
          Run a prediction or analysis to generate scenarios for comparison.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full bg-background overflow-hidden">
      {/* Scenario Selector */}
      <div
        ref={scenarioSelectorRef}
        role="radiogroup"
        aria-label="Select scenarios to compare"
        className="p-4 border-b border-border bg-surface flex flex-wrap gap-2"
        onKeyDown={(e) => {
          const buttons = scenarioSelectorRef.current?.querySelectorAll('button') ?? [];
          const btnArray = Array.from(buttons);
          const currentIdx = btnArray.indexOf(document.activeElement as HTMLButtonElement);
          if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
            e.preventDefault();
            const nextIdx = (currentIdx + 1) % btnArray.length;
            (btnArray[nextIdx] as HTMLButtonElement)?.focus();
          } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
            e.preventDefault();
            const prevIdx = (currentIdx - 1 + btnArray.length) % btnArray.length;
            (btnArray[prevIdx] as HTMLButtonElement)?.focus();
          }
        }}
      >
        <span className="text-textDim text-[10px] uppercase tracking-wider font-bold mr-2 self-center">Compare:</span>
        {scenarios.map(s => (
          <button 
            key={s.id}
            onClick={() => handleToggleScenario(s.id)}
            role="radio"
            aria-checked={selectedScenarioIds.includes(s.id)}
            className={`px-2 py-1 text-xs font-mono rounded border transition-all focus:ring-2 focus:ring-primary ${
              selectedScenarioIds.includes(s.id) 
                ? 'bg-primary/20 border-primary text-primary shadow-[0_0_8px_rgba(var(--primary-rgb),0.3)]' 
                : 'bg-surface border-border text-textDim hover:border-textMuted'
            }`}
          >
            {s.name}
          </button>
        ))}
      </div>

      {/* Comparison Grid */}
      <div className="flex-1 overflow-x-auto overflow-y-auto p-4 gap-4 flex">
        {selectedScenarios.length === 0 ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center">
            <AlertCircle size={32} className="text-textDim mb-2" />
            <p className="text-textDim text-xs font-mono">Select 2 or 3 scenarios to begin comparison</p>
          </div>
        ) : (
          selectedScenarios.map((scenario, idx) => (
            <div key={scenario.id} className="flex-1 min-w-[300px] max-w-md bg-surface border border-border rounded-lg flex flex-col overflow-hidden shadow-sm">
              {/* Header */}
              <div className="p-4 border-b border-border bg-surface-alt flex items-center justify-between">
                <h3 className="font-bold text-sm font-mono text-text truncate">{scenario.name}</h3>
                <div className="flex items-center gap-1 text-[10px] font-mono">
                  <span className="text-textDim uppercase">Conf:</span>
                  <span className="text-primary">{(scenario.confidence * 100).toFixed(1)}%</span>
                </div>
              </div>

              <div className="flex-1 p-4 space-y-6 overflow-y-auto no-scrollbar">
                {/* Confidence Bar */}
                <div className="space-y-2">
                  <div className="flex justify-between text-[10px] font-mono uppercase text-textDim">
                    <span>Confidence Score</span>
                    <span>{scenario.confidence}</span>
                  </div>
                  <div className="h-1.5 w-full bg-background rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-primary transition-all duration-500" 
                      style={{ width: `${scenario.confidence * 100}%` }}
                    />
                  </div>
                </div>

                {/* Trend */}
                <div className="flex items-center justify-between p-3 bg-background/50 border border-border rounded">
                  <span className="text-xs font-mono text-textDim uppercase">Trend Direction</span>
                  <div className={`flex items-center gap-1 font-bold text-xs font-mono ${
                    scenario.trend === 'up' ? 'text-success' : scenario.trend === 'down' ? 'text-danger' : 'text-textDim'
                  }`}>
                    {scenario.trend === 'up' && <ArrowUp size={14} />}
                    {scenario.trend === 'down' && <ArrowDown size={14} />}
                    {scenario.trend === 'neutral' && <Minus size={14} />}
                    <span className="uppercase">{scenario.trend}</span>
                  </div>
                </div>

                {/* Key Signals */}
                <div className="space-y-3">
                  <h4 className="text-textDim text-[10px] uppercase tracking-wider font-bold">Key Signals</h4>
                  <div className="space-y-2">
                    {scenario.signals.map(sig => (
                      <div key={sig.id} className="flex items-center justify-between p-2 bg-surface-alt border border-border rounded text-xs font-mono">
                        <span className="text-text truncate">{sig.name}</span>
                        <div className="flex items-center gap-2">
                          <div className="w-12 h-1 bg-background rounded-full overflow-hidden">
                            <div 
                              className="h-full bg-primary/60" 
                              style={{ width: `${sig.strength * 100}%` }}
                            />
                          </div>
                          <span className="text-[10px] text-textDim">{sig.strength}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Assumptions */}
                <div className="space-y-3">
                  <h4 className="text-textDim text-[10px] uppercase tracking-wider font-bold">Core Assumptions</h4>
                  <ul className="space-y-2">
                    {scenario.assumptions.map((asm, i) => (
                      <li key={i} className="text-xs font-mono text-textDim p-2 border-l-2 border-primary/30 bg-background/30">
                        {asm}
                      </li>
                    ))}
                  </ul>
                </div>

                {/* Description */}
                {scenario.description && (
                  <div className="space-y-3">
                    <h4 className="text-textDim text-[10px] uppercase tracking-wider font-bold">Scenario Detail</h4>
                    <p className="text-xs font-mono text-textDim leading-relaxed italic">
                      "{scenario.description}"
                    </p>
                  </div>
                )}
              </div>

              {/* Footer / Probability */}
              <div className="p-4 border-t border-border bg-surface-alt flex items-center justify-between">
                <span className="text-textDim text-[10px] uppercase font-bold">Probability</span>
                <span className="text-text font-mono text-sm font-bold">
                  {(scenario.probability * 100).toFixed(1)}%
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export const ScenarioComparisonView = memo(ScenarioComparisonViewInner)
