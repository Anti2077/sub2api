import { describe, expect, it } from 'vitest'
import { identityDetectedModel, identityFlagshipClass, identityRunStatus } from '../modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
import fixture from './fixtures/identity-flagship-954c.json'

const run = (): IdentityRun => ({ id: 1, plan_id: 1, created_at: '', request_model: 'gpt-6.1-sol', expected_model: 'openai/gpt-6.1-sol', ...structuredClone(fixture) })

describe('hosted flagship range presentation', () => {
  it('adapts the sanitized official 954c report as a range, preserving exact-model abstention', () => {
    const r = run()
    expect(identityFlagshipClass(r)).toBe(true)
    expect(identityRunStatus(r)).toBe('flagshipClass')
    expect(identityDetectedModel(r)).toBe('GPT-6 Astra / GPT-6.1 Sol')
    expect(r.report?.verdict).toBe('inconclusive')
    expect(r.report?.detected_model).toBe('')
  })

  it('keeps a confirmed model and failed or unfinished runs separate from a range', () => {
    for (const status of ['running', 'queued', 'request_error', 'service_error', 'cancelled', 'timed_out', 'configuration_error']) {
      const r = { ...run(), status }
      expect(identityFlagshipClass(r)).toBe(false)
      expect(identityRunStatus(r)).toBe(status)
    }
    const r = run()
    r.report!.verdict = 'matched'; r.report!.detected_model = 'GPT 6.1 Sol'
    expect(identityRunStatus(r)).toBe('matched')
    expect(identityDetectedModel(r)).toBe('GPT 6.1 Sol')
    r.report!.detected_model = ''
    expect(identityDetectedModel(r)).toBe('—')
  })

  it('does not infer a flagship range from the claim, votes, tied scores or a failed classifier', () => {
    for (const v3h of [
      {}, { topModel: 'openai/gpt-6-astra' },
      { perProbe: [{ topModel: 'openai/gpt-6-astra' }] },
      { abstained: true, posteriors: fixture.report.identityAssessment.v3h.posteriors },
      { posteriors: [{ modelId: 'openai/gpt-6-astra', score: 0.51 }, { modelId: 'openai/gpt-6.1-sol', score: 0.49 }] },
      { posteriors: [{ modelId: 'openai/gpt-6-luna', score: 0.99 }] },
      { posteriors: [{ modelId: 'openai/gpt-6-astra', score: '0.99' }] },
    ]) {
      const r = run()
      ;(r.report!.identityAssessment as Record<string, unknown>).v3h = v3h
      expect(identityRunStatus(r)).toBe('inconclusive')
    }
  })

  it('honors official flags, confirmed pairs, other families and insufficient-data verdicts', () => {
    const r = run(), assessment = r.report!.identityAssessment as Record<string, unknown>
    const verdict = assessment.verdict as Record<string, unknown>
    assessment.v3h = {}; verdict.flagshipClass = true
    expect(identityFlagshipClass(r)).toBe(true)
    verdict.flagshipPairResolved = true
    expect(identityFlagshipClass(r)).toBe(false)
    verdict.flagshipPairResolved = false; verdict.trueFamily = 'anthropic'
    expect(identityFlagshipClass(r)).toBe(false)
    verdict.trueFamily = 'openai'; verdict.status = 'insufficient_data'
    expect(identityFlagshipClass(r)).toBe(false)
    verdict.status = 'clean_match_family_only'; assessment.resolvedIdentity = { strongPass: true }
    expect(identityFlagshipClass(r)).toBe(false)
  })

  it('applies the same request-error coverage guard to full reports and compact history', () => {
    const r = run()
    r.report!.identity_probe_coverage = { total: 10, errors: 4 }
    expect(identityFlagshipClass(r)).toBe(false)
    delete r.report!.identity_probe_coverage
    r.report!.items = [{ group: 'identity', status: 'done' }, { group: 'submodel', status: 'error' }]
    expect(identityFlagshipClass(r)).toBe(false)
    r.report!.items = [{ group: 'identity', status: 'done' }, { group: 'security', status: 'error' }]
    expect(identityFlagshipClass(r)).toBe(true)
  })
})


describe('Fingerpoint confirmation threshold', () => {
  const fingerprint = (probability: unknown, detected = 'gpt-6-astra', extra: Record<string, unknown> = {}): IdentityRun => ({
    id: 2, plan_id: 1, status: 'completed', created_at: '', request_model: 'gpt-6.1-sol', expected_model: 'openai/gpt-6.1-sol',
    report: { engine: 'lm-fingerpoint-detector', detected_model: detected, confidence: probability,
      analysis: { probability, probability_status: 'reference_calibrated', used_outputs: 3, ...extra } },
  })
  it('confirms at 90 percent, compares shared classes and flags a different candidate', () => {
    expect(identityRunStatus(fingerprint(0.9))).toBe('matched')
    expect(identityRunStatus(fingerprint(0.9999, 'gpt-5.6-luna'))).toBe('mismatched')
    expect(identityRunStatus(fingerprint(0.89999))).toBe('ranked')
    const summary = fingerprint(0.9)
    delete (summary.report!.analysis as Record<string, unknown>).probability
    expect(identityRunStatus(summary)).toBe('matched')
  })
  it('does not confirm partial, uncalibrated or invalid confidence', () => {
    expect(identityRunStatus(fingerprint(0.99, 'gpt-6-astra', { decision: 'partial' }))).toBe('partial')
    expect(identityRunStatus(fingerprint(0.99, 'gpt-6-astra', { probability_status: 'unavailable' }))).toBe('ranked')
    for (const probability of [null, '0.99', NaN, Infinity, 1.1]) expect(identityRunStatus(fingerprint(probability))).toBe('ranked')
  })
})
