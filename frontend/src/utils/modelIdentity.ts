import type { IdentityRun } from '@/api/admin/modelIdentity'

export type IdentityStatus = 'matched' | 'mismatched' | 'inconclusive' | 'configuration_error' | 'queued' | 'running' | 'cancelling' | 'cancelled' | 'timed_out' | 'service_error' | 'request_error' | 'notConfigured' | 'untested'

export interface IdentityModelVote { model: string; votes: number }

function record(value: unknown): Record<string, unknown> | undefined {
  return value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : undefined
}

// Match the hosted report's per-probe voting table, including abstained runs.
// Scores/posteriors and family weights are not votes. Never sum V3H and V3G:
// they may describe the same probes; V3G is only a legacy fallback.
export function identityModelVotes(report?: Record<string, unknown>): IdentityModelVote[] {
  const assessment = record(report?.identityAssessment)
  for (const classifier of [assessment?.v3h, assessment?.v3g]) {
    const perProbe = record(classifier)?.perProbe
    if (!Array.isArray(perProbe)) continue
    const counts = new Map<string, number>()
    for (const probe of perProbe) {
      const topModel = record(probe)?.topModel
      if (typeof topModel !== 'string' || !topModel.trim()) continue
      const model = topModel.trim()
      counts.set(model, (counts.get(model) || 0) + 1)
    }
    if (counts.size) {
      return [...counts].map(([model, votes]) => ({ model, votes }))
        .sort((a, b) => b.votes - a.votes || a.model.localeCompare(b.model))
    }
  }
  return []
}

export function identityRunStatus(run: IdentityRun): IdentityStatus {
  if (run.status === 'completed') {
    return run.report?.verdict === 'matched' ? 'matched' : run.report?.verdict === 'mismatched' ? 'mismatched' : 'inconclusive'
  }
  return ['queued', 'running', 'cancelling', 'cancelled', 'timed_out', 'configuration_error', 'service_error', 'request_error'].includes(run.status) ? run.status as IdentityStatus : 'service_error'
}
export function isActive(run?: IdentityRun | null) { return Boolean(run && ['queued', 'running', 'cancelling'].includes(run.status)) }
export function identityStatusClass(status: IdentityStatus) {
  if (status === 'matched') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (['mismatched', 'configuration_error', 'service_error', 'request_error', 'timed_out'].includes(status)) return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (['queued', 'running', 'cancelling'].includes(status)) return 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  if (status === 'inconclusive') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}
