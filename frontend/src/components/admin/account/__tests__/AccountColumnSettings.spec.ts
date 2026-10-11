import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import AccountColumnSettings from '../AccountColumnSettings.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const draggable = defineComponent({ props: ['modelValue'], emits: ['update:modelValue'], template: '<div><slot /></div>' })
describe('AccountColumnSettings', () => {
  it('allows keyboard reordering of hidden columns and keeps visibility independent', async () => {
    const wrapper = mount(AccountColumnSettings, {
      props: { columns: [{ key: 'id', label: 'ID' }, { key: 'proxy', label: 'Proxy' }], hidden: new Set(['proxy']) },
      global: { stubs: { Icon: true, VueDraggable: draggable } }
    })
    await wrapper.findAll('.account-column-handle')[1].trigger('keydown', { key: 'ArrowUp' })
    expect(wrapper.emitted('order')).toEqual([[['proxy', 'id']]])
    expect(wrapper.findAll('input')[0].element.checked).toBe(false)
    await wrapper.findAll('input')[0].setValue(true)
    expect(wrapper.emitted('toggle')).toEqual([['proxy']])
    await wrapper.findAll('button')[0].trigger('click')
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })
})
