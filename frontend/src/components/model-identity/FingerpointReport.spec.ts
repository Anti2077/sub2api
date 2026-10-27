import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import FingerpointReport from './FingerpointReport.vue'
import type { IdentityRun } from '@/api/admin/modelIdentity'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function fixture(probability: number | null, decision = 'ranked'): IdentityRun {
  return {
    id: 1, plan_id: 1, status: 'completed', created_at: '2026-10-19T00:00:00Z', request_model: 'configured-route', expected_model: 'gpt-6.1-sol',
    report: {
      engine: 'lm-fingerpoint-detector', detected_model: 'gpt-6-astra',
      analysis: { prediction: 'gpt-6-astra', prediction_name: 'gpt-6-astra/6.1-sol', probability, probability_status: probability === null ? 'unavailable' : 'reference_calibrated', decision, used_outputs: decision === 'partial' ? 2 : 3,
        results: Array.from({ length: 12 }, (_, i) => ({ model: i ? `candidate-${i}` : 'gpt-6-astra', display_name: `Model ${i}`, probability: probability === null ? null : probability / (i + 1), family_name: 'Family' })) },
      samples: Array.from({ length: 3 }, (_, i) => ({ id: String(i), prompt: `Challenge ${i}`, text: '123, 567', state: 'done', valid_numbers: 2, expected_count: 300, duration_ms: 1200, ttft_ms: 120, tps: 42 })),
    },
  }
}

describe('Fingerpoint evidence report', () => {
  it('links each stored report to its actual engine source revision', async () => {
    const run = fixture(0.99)
    run.report!.engine_commit = '4dcb4216151cbd5066d515297720130a9fd3b198'
    const wrapper = mount(FingerpointReport, { props: { run } })
    expect(wrapper.get('a').attributes('href')).toBe(`https://github.com/Ikaleio/lm-detector/tree/${run.report!.engine_commit}`)
    const historical = fixture(0.99)
    historical.report!.engine_commit = '654c8334bb53951aaf04cbd61252d163649e1369'
    await wrapper.setProps({ run: historical })
    expect(wrapper.get('a').attributes('href')).toBe(`https://github.com/Ikaleio/lm-detector/tree/${historical.report!.engine_commit}`)
    await wrapper.setProps({ run: fixture(0.99) })
    expect(wrapper.get('a').attributes('href')).toBe('https://github.com/Ikaleio/lm-detector')
  })

  it('shows partial evidence without manufacturing confidence percentages', () => {
    const wrapper = mount(FingerpointReport, { props: { run: fixture(null, 'partial') } })
    expect(wrapper.findAll('[data-sample]')).toHaveLength(3)
    expect(wrapper.text()).toContain('admin.modelIdentity.partialWarning')
    expect(wrapper.text()).toContain('admin.modelIdentity.calibrationWarning')
    expect(wrapper.text()).not.toContain('%')
    expect(wrapper.text()).toContain('gpt-6-astra/6.1-sol')
    expect(wrapper.text()).toContain('admin.modelIdentity.astraSharedHint')
    expect(wrapper.text()).not.toContain('admin.modelIdentity.candidateDiffers')
  })

  it('flags a candidate different from the expected model even with a high reference probability', () => {
    const run = fixture(0.9999)
    const analysis = run.report!.analysis as { prediction: string; prediction_name: string }
    run.report!.detected_model = 'gpt-5.6-luna'
    analysis.prediction = 'gpt-5.6-luna'
    analysis.prediction_name = 'gpt-5.6-luna'
    const wrapper = mount(FingerpointReport, { props: { run } })
    expect(wrapper.text()).toContain('admin.modelIdentity.candidateDiffers')
    expect(wrapper.text()).toContain('admin.modelIdentity.rankingCaution')
    expect(wrapper.text()).toContain('admin.modelIdentity.mismatched')
  })

  it('expands all candidates and resets expansion when a different run opens', async () => {
    const wrapper = mount(FingerpointReport, { props: { run: fixture(0.9) } })
    expect(wrapper.findAll('ol li')).toHaveLength(8)
    expect(wrapper.text()).toContain('90.0%')
    const toggle = wrapper.get('button[aria-expanded]')
    await toggle.trigger('click')
    expect(wrapper.findAll('ol li')).toHaveLength(12)
    await wrapper.setProps({ run: { ...fixture(null), id: 2 } })
    expect(wrapper.findAll('ol li')).toHaveLength(8)
    expect(wrapper.get('button[aria-expanded]').attributes('aria-expanded')).toBe('false')
    expect(wrapper.text()).not.toContain('%')
  })
})
