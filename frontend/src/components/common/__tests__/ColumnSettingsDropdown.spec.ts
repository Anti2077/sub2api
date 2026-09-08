import { describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'

import ColumnSettingsDropdown from '../ColumnSettingsDropdown.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const VueDraggableStub = defineComponent({
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<div><slot /></div>',
})

const columns = [
  { key: 'user', label: 'User' },
  { key: 'model', label: 'Model' },
  { key: 'created_at', label: 'Created' },
]

describe('ColumnSettingsDropdown', () => {
  it('renders all columns in the persisted order and toggles visibility', async () => {
    const wrapper = mount(ColumnSettingsDropdown, {
      props: {
        columns,
        order: ['model', 'user', 'created_at'],
        hiddenKeys: ['model'],
        alwaysVisibleKeys: ['created_at'],
        title: 'Columns',
      },
      global: { stubs: { VueDraggable: VueDraggableStub, Icon: true } },
    })

    await wrapper.get('button[title="Columns"]').trigger('click')

    expect(wrapper.findAll('[data-testid^="usage-column-toggle-"]').map((node) => node.text())).toEqual([
      'Model',
      'User',
      'Created',
    ])
    await wrapper.get('[data-testid="usage-column-toggle-model"]').trigger('click')
    expect(wrapper.emitted('toggle-column')).toEqual([['model']])
    expect(wrapper.get('input[aria-label="Created"]').attributes('disabled')).toBeDefined()
  })

  it('emits a normalized order when the draggable list changes', async () => {
    const wrapper = mount(ColumnSettingsDropdown, {
      props: {
        columns,
        order: ['user', 'model', 'created_at'],
        hiddenKeys: [],
        title: 'Columns',
      },
      global: { stubs: { VueDraggable: VueDraggableStub, Icon: true } },
    })

    await wrapper.get('button[title="Columns"]').trigger('click')
    wrapper.findComponent(VueDraggableStub).vm.$emit('update:modelValue', ['created_at', 'model', 'user'])
    await nextTick()

    expect(wrapper.emitted('update:order')).toEqual([[['created_at', 'model', 'user']]])
  })
})
