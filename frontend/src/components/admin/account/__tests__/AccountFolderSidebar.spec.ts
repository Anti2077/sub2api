import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountFolderSidebar from '../AccountFolderSidebar.vue'

enableAutoUnmount(afterEach)
const api = vi.hoisted(() => ({ overview: vi.fn(), save: vi.fn(), delete: vi.fn(), move: vi.fn() }))
vi.mock('@/api/admin/accountFolders', () => ({ accountFoldersAPI: api }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const folders = [
  { id: 1, name: 'Main', parent_id: null, account_count: 3, error_count: 1 },
  { id: 2, name: 'OpenAI', parent_id: 1, account_count: 2, error_count: 0 }
]
const create = (modelValue = '') => mount(AccountFolderSidebar, {
  props: { modelValue },
  global: { stubs: {
    Icon: true,
    BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
    ConfirmDialog: true
  } }
})
beforeEach(() => {
  vi.clearAllMocks(); localStorage.clear()
  api.overview.mockResolvedValue({ folders, total: 4, unfiled: 1 })
  api.move.mockResolvedValue({ moved: 2 })
})
describe('AccountFolderSidebar', () => {
  it('restores collapse state and selects a child without changing its parent scope', async () => {
    localStorage.setItem('account-folders-collapsed', '[1]')
    const wrapper = create(); await flushPromises()
    expect(wrapper.text()).not.toContain('OpenAI')
    await wrapper.get('button[aria-label="Main"]').trigger('click')
    await wrapper.findAll('button').find(b => b.text().includes('OpenAI'))!.trigger('click')
    expect(wrapper.emitted('select')).toEqual([['2']])
    expect(localStorage.getItem('account-folders-collapsed')).toBe('[]')
  })
  it('moves the original selection and retains the dialog on server failure', async () => {
    const wrapper = create(); await flushPromises()
    const ids = [10, 20]
    wrapper.vm.openMove(ids); ids.push(30); await flushPromises()
    const select = wrapper.get('select')
    await select.setValue('2')
    api.move.mockRejectedValueOnce(new Error('network'))
    await wrapper.findAll('button').find(b => b.text() === 'admin.accounts.folders.move')!.trigger('click')
    await flushPromises()
    expect(api.move).toHaveBeenCalledWith([10, 20], 2)
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.emitted('moved')).toBeUndefined()
    await wrapper.findAll('button').find(b => b.text() === 'admin.accounts.folders.move')!.trigger('click')
    await flushPromises()
    expect(wrapper.emitted('moved')).toHaveLength(1)
  })
  it('clears a saved folder that was deleted by another administrator', async () => {
    const wrapper = create('99'); await flushPromises()
    expect(wrapper.emitted('select')).toEqual([['']])
  })
})
