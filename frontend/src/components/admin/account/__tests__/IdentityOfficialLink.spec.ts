import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import IdentityOfficialLink from '../IdentityOfficialLink.vue'
import IdentityHistoryTable from '../IdentityHistoryTable.vue'
import { identityOfficialURL } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const run: IdentityRun = { id: 42, plan_id: 1, status: 'running', request_model: 'gpt-6', expected_model: 'openai/gpt-6-astra', created_at: '', report: { remote_run_id: 'official-task-abc' } }

describe('hosted identity task links', () => {
  it('supports live, completed and failed runs with a fixed official origin and no local ID substitution', async () => {
    const w = mount(IdentityOfficialLink, { props: { run } })
    const link = () => w.get('a')
    expect(link().attributes('href')).toBe('https://bazaarlink.ai/probe?runId=official-task-abc')
    expect(link().text()).toContain('officialProgress')
    expect(link().attributes('rel')).toBe('noopener noreferrer')
    expect(link().attributes('referrerpolicy')).toBe('no-referrer')
    for (const status of ['completed', 'request_error', 'cancelled', 'timed_out']) {
      await w.setProps({ run: { ...run, status } })
      expect(link().text()).toContain('officialReport')
      expect(link().attributes('href')).toContain('official-task-abc')
    }
    w.unmount()
  })

  it('supports old stored reports but never navigates to reportUrl or malformed task IDs', () => {
    expect(identityOfficialURL({ ...run, report: { runId: 'old-task', reportUrl: 'https://evil.test/' } })).toBe('https://bazaarlink.ai/probe?runId=old-task')
    expect(identityOfficialURL({ ...run, report: { run_id: 'legacy-task' } })).toContain('legacy-task')
    for (const report of [{}, { reportUrl: 'https://evil.test/' }, { remote_run_id: 42 }, { runId: 'https://evil.test/' }, { runId: 'x\nsecret' }, { runId: 'x'.repeat(201) }]) {
      expect(identityOfficialURL({ ...run, report })).toBeUndefined()
    }
    expect(identityOfficialURL()).toBeUndefined()
  })

  it('only shows the link once the hosted run is created, including in history', async () => {
    const w = mount(IdentityHistoryTable, { props: { runs: [{ ...run, report: {} }], busy: false } })
    expect(w.find('a').exists()).toBe(false)
    await w.setProps({ runs: [run, { ...run, id: 43, status: 'completed', report: { runId: 'old-result' } }] })
    expect(w.findAll('a').map(link => link.attributes('href'))).toEqual(['https://bazaarlink.ai/probe?runId=official-task-abc', 'https://bazaarlink.ai/probe?runId=old-result'])
    w.unmount()
  })
})
