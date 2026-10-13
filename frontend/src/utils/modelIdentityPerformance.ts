export interface IdentityPerformanceItem {
  probeId: string
  label: string
  status: 'pending' | 'running' | 'done' | 'error' | 'unknown'
  ttftMs: number | null
  tps: number | null
  durationMs: number | null
  outputTokens: number | null
}

function measurement(value: unknown, positive = false): number | null {
  return typeof value === 'number' && Number.isFinite(value) && (positive ? value > 0 : value >= 0) ? value : null
}

// Use hosted items independently of local gateway probes: repeats and retries
// mean their counts/order need not agree. Never derive TPS from wall-clock time.
export function identityPerformance(report?: Record<string, unknown>) {
  const items: IdentityPerformanceItem[] = []
  if (Array.isArray(report?.items)) {
    for (const value of report.items) {
      if (!value || typeof value !== 'object' || Array.isArray(value)) continue
      const item = value as Record<string, unknown>
      const text = (value: unknown) => typeof value === 'string' ? value.trim() : ''
      items.push({
        probeId: text(item.probeId), label: text(item.label),
        status: ['pending', 'running', 'done', 'error'].includes(String(item.status)) ? item.status as IdentityPerformanceItem['status'] : 'unknown',
        ttftMs: measurement(item.ttftMs), tps: measurement(item.tps, true),
        durationMs: measurement(item.durationMs),
        outputTokens: Number.isInteger(item.outputTokens) ? measurement(item.outputTokens) : null,
      })
    }
  }
  const average = (key: 'ttftMs' | 'tps') => {
    const values = items.filter(item => item.status === 'done' && item[key] !== null).map(item => item[key] as number)
    return { count: values.length, value: values.length ? values.reduce((mean, value, index) => mean + (value - mean) / (index + 1), 0) : null }
  }
  return { items, ttft: average('ttftMs'), tps: average('tps'), wallClockMs: measurement(report?.wallClockMs) }
}
