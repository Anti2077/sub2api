import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountFolderSidebar from '../AccountFolderSidebar.vue'

enableAutoUnmount(afterEach)
const api = vi.hoisted(() => ({ overview: vi.fn(), save: vi.fn(), delete: vi.fn(), tags: vi.fn(), updateTags: vi.fn() }))
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
  api.tags.mockResolvedValue([])
  api.updateTags.mockResolvedValue({ updated: 2 })
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
  it('updates only the original selection and retains tags on server failure', async () => {
    const wrapper = create(); await flushPromises()
    const ids = [10, 20]
    wrapper.vm.openTags(ids); ids.push(30); await flushPromises()
    const input = wrapper.get('input[type="text"]')
    await input.setValue('batch A')
    await input.trigger('keydown', { key: 'Enter' })
    api.updateTags.mockRejectedValueOnce(new Error('network'))
    await wrapper.get('#account-tags-form').trigger('submit')
    await flushPromises()
    expect(api.updateTags).toHaveBeenCalledWith([10, 20], ['batch A'], 'add')
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.emitted('tagged')).toBeUndefined()
    await wrapper.get('#account-tags-form').trigger('submit')
    await flushPromises()
    expect(wrapper.emitted('tagged')).toHaveLength(1)
  })
  it('saves conditions and keeps child rules independent from the parent', async () => {
    const wrapper = create(); await flushPromises()
    await wrapper.get('button[aria-label="admin.accounts.folders.create"]').trigger('click')
    const form = wrapper.get('#account-folder-form')
    await form.get('input').setValue('Important')
    expect(wrapper.get('button[form="account-folder-form"]').attributes('disabled')).toBeDefined()
    await form.findAll('input').find(input => input.attributes('maxlength') === '200')!.setValue('prod')
    await form.get('select').setValue('1')
    await form.trigger('submit'); await flushPromises()
    expect(api.save).toHaveBeenCalledWith({ id: undefined, name: 'Important', parent_id: 1, rules: { name: 'prod' } })
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })
  it('clears a saved folder that was deleted by another administrator', async () => {
    const wrapper = create('99'); await flushPromises()
    expect(wrapper.emitted('select')).toEqual([['']])
  })
})
