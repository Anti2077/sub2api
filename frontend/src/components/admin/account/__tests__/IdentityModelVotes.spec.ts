import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n, type MessageContext } from 'vue-i18n'
import IdentityModelVotes from '../IdentityModelVotes.vue'
import { identityModelVotes, identityRunStatus } from '@/utils/modelIdentity'
import zh from '@/i18n/locales/zh/admin/modelIdentity'
import en from '@/i18n/locales/en/admin/modelIdentity'

const models = ['openai/gpt-6-astra', 'openai/gpt-6.1-sol', 'openai/gpt-5.3-codex', 'openai/gpt-5.6-luna']
const report = {
  verdict: 'inconclusive',
  identityAssessment: {
    v3h: {
      abstained: true, sampleCount: 215,
      perProbe: [0, 1, 0, 1, 0, 1, 0, 2, 3].map((model, index) => ({ probeId: `probe-${index}`, topModel: models[model] })),
    },
  },
}

function open(value: Record<string, unknown> = report, locale = 'zh') {
  // Vitest uses the runtime-only i18n build; supply message functions there.
  const messages = (source: typeof zh) => ({ admin: { modelIdentity: Object.fromEntries(
    Object.entries(source.modelIdentity).map(([key, text]) => [key, ({ named }: MessageContext) => text.replace(/\{(\w+)\}/g, (_, name: string) => String(named(name)))]),
  ) } })
  return mount(IdentityModelVotes, {
    props: { report: value },
    global: { plugins: [createI18n({ legacy: false, locale, messages: { zh: messages(zh), en: messages(en) } })] },
  })
}

describe('identity report model votes', () => {
  it('shows the screenshot distribution even when the engine abstains, without changing the verdict', () => {
    const w = open()
    const rows = w.findAll('dl > div')
    expect(rows.map(row => row.get('dt').text())).toEqual(models)
    expect(rows.map(row => row.get('dd').text())).toEqual(['4 票', '3 票', '1 票', '1 票'])
    expect(w.text()).toContain('共 9 个探针投票')
    expect(w.text()).toContain('不代表置信度或最终身份判定')
    expect(w.text()).not.toContain('215 个探针')
    expect(identityRunStatus({ id: 1, plan_id: 1, status: 'completed', expected_model: models[1], request_model: 'gpt-6.1-sol', created_at: '', report })).toBe('inconclusive')
    w.unmount()
  })

  it('ignores abstentions and malformed entries, sorts ties by model ID, and avoids double-counting V3G', () => {
    expect(identityModelVotes({ identityAssessment: {
      v3h: { perProbe: [null, {}, { topModel: null }, { topModel: 4 }, { topModel: ' ' }, { topModel: 'model-b' }, { topModel: 'model-a' }] },
      v3g: { perProbe: [{ topModel: 'model-c' }] },
    } })).toEqual([{ model: 'model-a', votes: 1 }, { model: 'model-b', votes: 1 }])
  })

  it('supports stored V3G reports and reactive report changes', async () => {
    const w = open({}, 'en')
    expect(w.text()).toContain('does not include per-probe voting data')
    expect(w.find('dl').exists()).toBe(false)
    await w.setProps({ report: { identityAssessment: { v3g: { perProbe: [{ topModel: models[0] }, { topModel: models[0] }] } } } })
    expect(w.get('dt').text()).toBe(models[0])
    expect(w.get('dd').text()).toBe('2 votes')
    w.unmount()
  })

  it('never treats posterior scores, family weights or malformed structures as model votes', () => {
    for (const value of [undefined, {}, { identityAssessment: null }, { identityAssessment: [] },
      { identityAssessment: { v3h: { posteriors: [{ modelId: models[0], score: 0.98 }] }, v3: { familyVote: { votes: { openai: 4 } } } } },
      { identityAssessment: { v3h: { perProbe: {} }, v3g: [] } },
    ]) expect(identityModelVotes(value)).toEqual([])
  })
})
