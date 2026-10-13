import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n, type MessageContext } from 'vue-i18n'
import IdentityPerformance from '../IdentityPerformance.vue'
import { identityPerformance } from '@/utils/modelIdentityPerformance'
import zh from '@/i18n/locales/zh/admin/modelIdentity'
import en from '@/i18n/locales/en/admin/modelIdentity'

const report = {
  wallClockMs: 8000,
  items: [
    { probeId: 'probe-a', label: 'Text probe', status: 'done', ttftMs: 0, tps: 20, durationMs: 1500, outputTokens: 30, retries: [{ ttftMs: 9000, tps: 1 }] },
    { probeId: 'probe-b', status: 'done', ttftMs: 1000, tps: 40, durationMs: 2500, outputTokens: 100 },
    { probeId: 'probe-c', status: 'done', ttftMs: null, tps: 60 },
    { probeId: 'probe-error', status: 'error', ttftMs: 9000, tps: 200 },
  ],
}
function open(value: Record<string, unknown> = report, locale = 'zh') {
  const messages = (source: typeof zh) => ({ admin: { modelIdentity: Object.fromEntries(
    Object.entries(source.modelIdentity).map(([key, text]) => [key, ({ named }: MessageContext) => text.replace(/\{(\w+)\}/g, (_, name: string) => String(named(name)))]),
  ) } })
  return mount(IdentityPerformance, {
    props: { report: value },
    global: { plugins: [createI18n({ legacy: false, locale, messages: { zh: messages(zh), en: messages(en) } })] },
  })
}

describe('identity report performance', () => {
  it('averages only valid successful samples independently and does not count retry history twice', () => {
    const performance = identityPerformance(report)
    expect(performance.ttft).toEqual({ count: 2, value: 500 })
    expect(performance.tps).toEqual({ count: 3, value: 40 })
    expect(performance.items).toHaveLength(4)
    expect(performance.wallClockMs).toBe(8000)
  })

  it('rejects malformed, nonfinite, negative and string values without inventing TPS from usage', () => {
    const performance = identityPerformance({ wallClockMs: -1, items: [null, [], 'invalid',
      { status: 'done', ttftMs: -1, tps: 0, durationMs: Infinity, outputTokens: 1.5 },
      { status: 'done', ttftMs: '500', tps: NaN, durationMs: -5, outputTokens: -1 },
      { status: 'done', ttftMs: Infinity, tps: '50', durationMs: 1000, outputTokens: 100 },
      { status: 'running', ttftMs: 0, tps: 1 },
      { status: 'mystery', ttftMs: 0, tps: 1 },
    ] })
    expect(performance.ttft).toEqual({ count: 0, value: null })
    expect(performance.tps).toEqual({ count: 0, value: null })
    expect(performance.wallClockMs).toBeNull()
    expect(performance.items[0]).toMatchObject({ ttftMs: null, tps: null, durationMs: null, outputTokens: null })
    expect(performance.items[4].status).toBe('unknown')
    for (const value of [undefined, {}, { items: null }, { items: {} }]) expect(identityPerformance(value).items).toEqual([])
  })

  it('shows Chinese summaries and per-question timing, including missing values and failure status', () => {
    const w = open()
    expect(w.get('[data-testid="averageTtft"]').text()).toContain('500 ms')
    expect(w.get('[data-testid="averageTtft"]').text()).toContain('2 个有效成功样本')
    expect(w.get('[data-testid="averageTps"]').text()).toContain('40 token/s')
    expect(w.get('[data-testid="averageTps"]').text()).toContain('3 个有效成功样本')
    expect(w.get('summary').text()).toBe('逐题性能（4 题）')
    const rows = w.findAll('li')
    expect(rows[0].text()).toContain('Text probe')
    expect(rows[0].text()).toContain('0 ms')
    expect(rows[2].text()).toContain('—')
    expect(rows[3].text()).toContain('失败')
    expect(w.text()).toContain('不是整次检测吞吐量')
    w.unmount()
  })

  it('supports old and failed reports without converting null to zero, and reactive English updates', async () => {
    const w = open({}, 'en')
    expect(w.text()).toContain('No valid successful TTFT or TPS samples')
    expect(w.get('[data-testid="averageTtft"]').text()).toContain('—')
    expect(w.find('details').exists()).toBe(false)
    await w.setProps({ report })
    expect(w.get('[data-testid="averageTps"]').text()).toContain('3 valid successful samples')
    expect(w.get('summary').text()).toContain('4 questions')
    await w.setProps({ report: { items: [{ status: 'error', ttftMs: null, tps: null }] } })
    expect(w.get('[data-testid="averageTps"]').text()).toContain('—')
    expect(w.get('li').text()).toContain('Failed')
    w.unmount()
  })
})
