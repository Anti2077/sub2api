import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import ModelIdentityView from '../ModelIdentityView.vue'
import IdentityRecentRuns from '@/components/admin/account/IdentityRecentRuns.vue'
import type { IdentityRun } from '@/api/admin/modelIdentity'

const mocks = vi.hoisted(() => ({ accounts: vi.fn(), account: vi.fn(), groups: vi.fn(), config: vi.fn(), plans: vi.fn(), history: vi.fn(), run: vi.fn(), cancel: vi.fn(), runStatus: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list: mocks.accounts, getById: mocks.account }, groups: { getAll: mocks.groups } } }))
vi.mock('@/api/admin/modelIdentity', () => mocks)
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }) }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('@/components/admin/account/ModelIdentityPanel.vue', () => ({ default: { name: 'ModelIdentityPanel', props: ['show', 'account', 'groups'], template: '<div />' } }))
enableAutoUnmount(afterEach)
const account = { id: 18, name: 'Fixture account', platform: 'openai', group_ids: [7] }
const configuration = { account_id: 18, user_id: 5, group_id: 7, api_key_id: 9, key_name: '模型身份测试专用｜Fixture group｜Fixture account (#18)' }
const plan = { id: 1, account_id: 18, request_model: 'gpt-6-astra', expected_model: 'openai/gpt-6-astra', enabled: true, interval_minutes: 120, next_run_at: '2026-10-10T12:00:00Z' }
const runs: IdentityRun[] = Array.from({ length: 6 }, (_, i) => ({ id: i + 1, plan_id: 1, status: 'completed', created_at: `2026-10-10T0${i}:00:00Z`, request_model: 'gpt-6-astra', expected_model: 'openai/gpt-6-astra', report: { verdict: i === 5 ? 'mismatched' : 'matched', detected_model: i === 5 ? 'openai/gpt-5.5' : 'openai/gpt-6-astra' } }))
function open() { return mount(ModelIdentityView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, BaseDialog: { props: ['show'], template: '<section v-if="show"><slot /></section>' }, ModelIdentityPanel: true, Icon: true, Pagination: true } } }) }
describe('model identity account matrix', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    mocks.accounts.mockResolvedValue({ items: [account], total: 1 }); mocks.account.mockResolvedValue(account)
    mocks.groups.mockResolvedValue([{ id: 7, name: 'Fixture group', platform: 'openai' }])
    mocks.config.mockResolvedValue(configuration); mocks.plans.mockResolvedValue([plan]); mocks.history.mockResolvedValue(runs)
    mocks.runStatus.mockImplementation(async (id: number) => runs.find(run => run.id === id))
  })
  it('shows the latest result and five timestamped recent runs, newest first', async () => {
    const w = open(); await flushPromises()
    const row = w.get('table tr[data-account-id="18"]')
    expect(row.text()).toContain('Fixture group'); expect(row.text()).toContain('openai/gpt-5.5')
    const recent = row.findComponent(IdentityRecentRuns)
    expect(recent.props('runs').map((run: IdentityRun) => run.id)).toEqual([6, 5, 4, 3, 2])
    expect(recent.findAll('time')).toHaveLength(5)
    await recent.find('button').trigger('click'); await flushPromises()
    expect(mocks.runStatus).toHaveBeenCalledWith(6)
    expect(w.text()).toContain('openai/gpt-5.5')
  })
  it('keeps expanded history immediately below its account and opens real configuration', async () => {
    const w = open(); await flushPromises()
    const row = w.get('table tr[data-account-id="18"]')
    await row.findAll('button').find(button => button.text() === 'admin.modelIdentity.history')!.trigger('click')
    expect(row.element.nextElementSibling?.id).toBe('identity-history-18')
    await row.findAll('button').find(button => button.text() === 'admin.modelIdentity.manageAccount')!.trigger('click'); await flushPromises()
    expect(mocks.account).toHaveBeenCalledWith(18)
    expect(w.findComponent({ name: 'ModelIdentityPanel' }).props('account')?.id).toBe(18)
  })
  it('never substitutes demo accounts when the account API fails', async () => {
    mocks.accounts.mockRejectedValue(new Error('API unavailable'))
    const w = open(); await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('API unavailable')
    expect(w.findAll('[data-account-id]')).toHaveLength(0)
    expect(w.text()).not.toContain('OpenAI 主账号')
  })
  it('distinguishes missing configuration from history or service failures', async () => {
    mocks.config.mockRejectedValue({ status: 404 })
    const w = open(); await flushPromises()
    expect(w.get('table tr[data-account-id="18"]').text()).toContain('admin.modelIdentity.notConfigured')
    expect(mocks.history).not.toHaveBeenCalled()
    mocks.config.mockResolvedValue(configuration); mocks.history.mockRejectedValue(new Error('history failed'))
    await w.findAll('button').find(button => button.text() === 'common.refresh')!.trigger('click'); await flushPromises()
    expect(w.get('table tr[data-account-id="18"]').text()).toContain('history failed')
    expect(w.get('table tr[data-account-id="18"]').text()).toContain('admin.modelIdentity.service_error')
  })
  it('preserves request failure status and sends a manual run to the selected plan', async () => {
    mocks.history.mockResolvedValue([{ ...runs[5], status: 'request_error', report: { verdict: 'inconclusive' } }])
    const w = open(); await flushPromises(); const row = w.get('table tr[data-account-id="18"]')
    expect(row.text()).toContain('admin.modelIdentity.request_error')
    await row.findAll('button').find(button => button.text() === 'admin.modelIdentity.history')!.trigger('click')
    await w.get('#identity-history-18').findAll('button').find(button => button.text().includes('admin.modelIdentity.run'))!.trigger('click'); await flushPromises()
    expect(mocks.run).toHaveBeenCalledWith(1)
  })
})
