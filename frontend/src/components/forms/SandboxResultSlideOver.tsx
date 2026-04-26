import type { SandboxResult } from '../../store/types'

interface SandboxResultSlideOverProps {
  result?: SandboxResult
}

export function SandboxResultSlideOver({ result }: SandboxResultSlideOverProps) {
  return (
    <div className="space-y-4">
      <div className={`text-sm font-bold ${result?.exitCode === 0 ? 'text-success' : 'text-danger'}`}>
        Exit Code: {result?.exitCode ?? 'N/A'}
      </div>
      {result?.stdout && (
        <div className="bg-background p-4 rounded-lg border border-border">
          <div className="text-[10px] font-bold text-textDim uppercase mb-2">Stdout</div>
          <pre className="text-xs font-mono text-textMuted whitespace-pre-wrap">{result.stdout}</pre>
        </div>
      )}
      {result?.stderr && (
        <div className="bg-danger/10 p-4 rounded-lg border border-danger/20">
          <div className="text-[10px] font-bold text-danger uppercase mb-2">Stderr</div>
          <pre className="text-xs font-mono text-danger whitespace-pre-wrap">{result.stderr}</pre>
        </div>
      )}
      {result?.metricsJson && (
        <div className="text-[10px] text-textDim font-mono">Metrics: {result.metricsJson}</div>
      )}
    </div>
  )
}
