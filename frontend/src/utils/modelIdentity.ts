import type { IdentityRun } from '@/api/admin/modelIdentity'

export type IdentityStatus = 'matched' | 'mismatched' | 'inconclusive' | 'flagshipClass' | 'configuration_error' | 'queued' | 'running' | 'cancelling' | 'cancelled' | 'timed_out' | 'service_error' | 'request_error' | 'notConfigured' | 'untested'

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

const flagshipModels = ['openai/gpt-6-astra', 'openai/gpt-6.1-sol'] as const
function isFlagshipModel(value: unknown): boolean {
  return typeof value === 'string' && flagshipModels.includes(value.toLowerCase() as typeof flagshipModels[number])
}

// The hosted page presents this as a model range, not an exact match. Infer
// only from its final family verdict and non-tied classifier evidence; a claim,
// vote count, or topModel alone cannot establish the range. The history API
// supplies the same small evidence subset for reports saved before this UI.
export function identityFlagshipClass(run?: IdentityRun | null): boolean {
  if (!run || run.status !== 'completed' || run.report?.verdict !== 'inconclusive') return false
  const report = run.report, assessment = record(report.identityAssessment), verdict = record(assessment?.verdict)
  if (verdict?.status !== 'clean_match_family_only' || verdict.flagshipPairResolved === true || record(assessment?.resolvedIdentity)?.strongPass === true) return false
  const coverage = record(report.identity_probe_coverage)
  const probes = Array.isArray(report.items) ? report.items.map(record).filter(probe => probe?.group === 'identity' || probe?.group === 'submodel') : []
  const total = typeof coverage?.total === 'number' ? coverage.total : probes.length
  const errors = typeof coverage?.errors === 'number' ? coverage.errors : probes.filter(probe => probe?.status === 'error').length
  if (total > 0 && errors / total > 0.3) return false
  const family = verdict.trueFamily || record(assessment?.familyFusion)?.confirmedFamily || assessment?.predictedFamily
  if (family !== 'openai') return false
  if (verdict.flagshipClass === true) return true
  if (!isFlagshipModel(assessment?.claimedModel)) return false
  const v3h = record(assessment?.v3h)
  if (v3h?.abstained === true) return false
  const posteriors = Array.isArray(v3h?.posteriors) && v3h.posteriors.length ? v3h.posteriors : record(assessment?.v3)?.candidates
  if (!Array.isArray(posteriors)) return false
  const ranked = posteriors.map(record).filter((candidate): candidate is Record<string, unknown> => Boolean(candidate && typeof candidate.modelId === 'string' && candidate.modelId.startsWith('openai/') && typeof candidate.score === 'number' && Number.isFinite(candidate.score) && candidate.score >= 0 && candidate.score <= 1))
    .sort((a, b) => Number(b.score) - Number(a.score))
  return Boolean(ranked[0] && isFlagshipModel(ranked[0].modelId) && (!ranked[1] || Number(ranked[0].score) - Number(ranked[1].score) >= 0.05))
}

export function identityDetectedModel(run?: IdentityRun | null): string {
  if (identityFlagshipClass(run)) return 'GPT-6 Astra / GPT-6.1 Sol'
  const model = run?.report?.detected_model
  return typeof model === 'string' && model.trim() ? model : '—'
}

export function identityRunStatus(run: IdentityRun): IdentityStatus {
  if (run.status === 'completed') {
    if (identityFlagshipClass(run)) return 'flagshipClass'
    return run.report?.verdict === 'matched' ? 'matched' : run.report?.verdict === 'mismatched' ? 'mismatched' : 'inconclusive'
  }
  return ['queued', 'running', 'cancelling', 'cancelled', 'timed_out', 'configuration_error', 'service_error', 'request_error'].includes(run.status) ? run.status as IdentityStatus : 'service_error'
}
export function isActive(run?: IdentityRun | null) { return Boolean(run && ['queued', 'running', 'cancelling'].includes(run.status)) }
export function identityStatusClass(status: IdentityStatus) {
  if (status === 'matched') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (['mismatched', 'configuration_error', 'service_error', 'request_error', 'timed_out'].includes(status)) return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (['queued', 'running', 'cancelling'].includes(status)) return 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  if (status === 'inconclusive' || status === 'flagshipClass') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

// Always construct a fixed-origin URL from an opaque ID. Never trust reportUrl
// or the local numeric run ID; old hosted reports may contain runId/run_id.
export function identityOfficialURL(run?: IdentityRun | null): string | undefined {
  const report = run?.report
  for (const id of [report?.remote_run_id, report?.runId, report?.run_id]) {
    if (typeof id === 'string' && /^[A-Za-z0-9_-]{1,200}$/.test(id)) {
      return `https://bazaarlink.ai/probe?runId=${encodeURIComponent(id)}`
    }
  }
  return undefined
}
