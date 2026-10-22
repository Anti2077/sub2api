import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountTagsInput from '../AccountTagsInput.vue'

enableAutoUnmount(afterEach)
const api = vi.hoisted(() => ({ tags: vi.fn() }))
vi.mock('@/api/admin/accountFolders', () => ({ accountFoldersAPI: api }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const create = (tags: string[] = []) => {
  const wrapper = mount(AccountTagsInput, { props: { modelValue: tags, 'onUpdate:modelValue': value => wrapper.setProps({ modelValue: value }) } })
  return wrapper
}
beforeEach(() => { api.tags.mockResolvedValue([{ name: '已有 标签', account_count: 2 }]) })

describe('AccountTagsInput', () => {
  it('commits a trimmed tag on Enter, deduplicates and removes selected labels', async () => {
    const wrapper = create(['one']); const input = wrapper.get('input')
    await input.setValue(' 中文 标签 '); await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.props('modelValue')).toEqual(['one', '中文 标签'])
    await input.setValue('one'); await input.trigger('blur')
    expect(wrapper.props('modelValue')).toEqual(['one', '中文 标签'])
    await wrapper.get('button[aria-label="admin.accounts.tags.remove: one"]').trigger('click')
    expect(wrapper.props('modelValue')).toEqual(['中文 标签'])
  })
  it('does not commit during IME composition', async () => {
    const wrapper = create(); const input = wrapper.get('input')
    await input.setValue('中文'); await input.trigger('keydown', { key: 'Enter', isComposing: true })
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.props('modelValue')).toEqual(['中文'])
  })
  it('offers existing tags with spaces and blocks invalid drafts', async () => {
    const wrapper = create(); await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('已有 标签'))!.trigger('click')
    expect(wrapper.props('modelValue')).toEqual(['已有 标签'])
    const input = wrapper.get('input')
    await input.setValue('长'.repeat(129)); await input.trigger('keydown', { key: ',' })
    expect(input.attributes('aria-invalid')).toBe('true')
    expect((input.element as HTMLInputElement).validity.valid).toBe(false)
    expect(wrapper.props('modelValue')).toEqual(['已有 标签'])
  })
  it('keeps the form usable when suggestions cannot be loaded', async () => {
    api.tags.mockRejectedValueOnce(new Error('offline'))
    const wrapper = create(); await flushPromises()
    await wrapper.get('input').setValue('new'); await wrapper.get('input').trigger('blur')
    expect(wrapper.props('modelValue')).toEqual(['new'])
  })
})
