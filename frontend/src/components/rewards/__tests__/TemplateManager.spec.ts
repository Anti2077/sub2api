import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TemplateManager from '../TemplateManager.vue'
import type { CardTemplate } from '@/api/rewards'

const api = vi.hoisted(() => ({ saveTemplate: vi.fn() }))
vi.mock('@/api/rewards', () => ({ rewardsAPI: api, rewardError: (e: Error) => e.message }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (k: string, values?: object) => k + (values ? JSON.stringify(values) : '') }) }))
const templates: CardTemplate[] = Array.from({ length: 15 }, (_, i) => ({
  id: i+1, version: 3, name: `Template ${i+1}`, kind: i===0 ? 'group_rate' : i===1 ? 'model_day' : 'model_tokens',
  model: i===0 ? '' : 'luna', group_ids: [i===0 ? 2 : 1], rate: .25, tokens: 2500, hours: 24, claim_days: 30, published: i!==2
}))
const render = (reload = vi.fn().mockResolvedValue(undefined)) => mount(TemplateManager, {
  props: { templates: structuredClone(templates), groups: [{id:1,name:'Luna group'},{id:2,name:'GPT group'}], busy:false, reload },
  global: { stubs: { ConfirmDialog: { props: ['show','message'], emits: ['confirm','cancel'], template: '<div v-if="show" role="dialog">{{ message }}<button data-confirm @click="$emit(\'confirm\')">Confirm</button><button data-cancel @click="$emit(\'cancel\')">Cancel</button></div>' } } }
})
const button = (w: ReturnType<typeof render>, key: string) => w.findAll('button').find(b => b.text()===`rewards.${key}`)!
const rows = (w: ReturnType<typeof render>) => w.findAll('article')
beforeEach(() => { vi.clearAllMocks(); api.saveTemplate.mockResolvedValue({}) })

describe('TemplateManager', () => {
  it('selects all matching pages and resizes without losing selection, while disabling no-op actions', async()=>{
    const w=render();await w.findAll('button').find(b=>b.text().startsWith('rewards.manageSelectMatching'))!.trigger('click');expect(w.text()).toContain('"count":15');await w.get('select[aria-label="rewards.managePageSize"]').setValue('24');expect(rows(w)).toHaveLength(15);await button(w,'manageInvertPage').trigger('click');expect(button(w,'bulkPublish').attributes('disabled')).toBeDefined()
    await w.get('article input').setValue(true);expect(button(w,'bulkPublish').attributes('disabled')).toBeDefined();expect(button(w,'bulkUnpublish').attributes('disabled')).toBeUndefined();w.unmount()
  })
  it('combines type/status/search and clears selection when the filter changes', async () => {
    const w=render()
    expect(rows(w)).toHaveLength(12)
    await w.get('article input').setValue(true)
    await w.findAll('select')[0].setValue('model_tokens')
    expect(button(w,'bulkUnpublish').attributes('disabled')).toBeDefined()
    await w.findAll('select')[1].setValue('draft')
    expect(rows(w)).toHaveLength(1)
    expect(rows(w)[0].text()).toContain('Template 3')
    await w.get('input[type="search"]').setValue('missing')
    expect(rows(w)).toHaveLength(0)
    w.unmount()
  })
  it('searches models and group names', async () => {
    const w=render()
    await w.get('input[type="search"]').setValue('GPT group')
    expect(rows(w)).toHaveLength(1)
    expect(rows(w)[0].text()).toContain('0.25×')
    await w.get('input[type="search"]').setValue('luna')
    expect(rows(w)).toHaveLength(12)
    w.unmount()
  })
  it('selects only the current page while preserving selection on other pages', async () => {
    const w=render()
    await w.get('input[type="checkbox"]').setValue(true)
    await button(w,'nextPage').trigger('click')
    expect(rows(w)).toHaveLength(3)
    expect((w.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(false)
    await w.get('input[type="checkbox"]').setValue(true)
    await w.get('input[type="checkbox"]').setValue(false)
    await button(w,'previousPage').trigger('click')
    expect((w.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    w.unmount()
  })
  it('requires confirmation and cancel does not save', async () => {
    const w=render()
    await w.get('article input').setValue(true)
    await button(w,'bulkUnpublish').trigger('click')
    expect(api.saveTemplate).not.toHaveBeenCalled()
    await w.get('[data-cancel]').trigger('click')
    expect(w.find('[role="dialog"]').exists()).toBe(false)
    expect(api.saveTemplate).not.toHaveBeenCalled()
    w.unmount()
  })
  it('continues after a version conflict and retries only the failed item with refreshed version', async () => {
    const reload=vi.fn().mockResolvedValue(undefined), w=render(reload)
    await w.findAll('article input')[0].setValue(true)
    await w.findAll('article input')[1].setValue(true)
    api.saveTemplate.mockRejectedValueOnce(new Error('Version conflict')).mockResolvedValueOnce({})
    await button(w,'bulkUnpublish').trigger('click')
    await w.get('[data-confirm]').trigger('click'); await flushPromises()
    expect(api.saveTemplate).toHaveBeenCalledTimes(2)
    expect(api.saveTemplate.mock.calls[0][0]).toMatchObject({id:1,version:3,published:false})
    expect(w.get('[role="alert"]').text()).toContain('#1: Version conflict')
    expect(w.findAll('li').map(v => v.text())).toEqual([
      '#1 · Template 1 — rewards.bulkTemplateFailed',
      '#2 · Template 2 — rewards.bulkTemplateSaved'
    ])
    expect(w.findAll('article input').filter(x => (x.element as HTMLInputElement).checked)).toHaveLength(1)
    await w.setProps({templates: templates.map(v => ({...v,version:4,published:v.id===2 ? false : v.published}))})
    await button(w,'bulkUnpublish').trigger('click')
    await w.get('[data-confirm]').trigger('click'); await flushPromises()
    expect(api.saveTemplate).toHaveBeenCalledTimes(3)
    expect(api.saveTemplate.mock.calls[2][0]).toMatchObject({id:1,version:4,published:false})
    expect(reload).toHaveBeenCalledTimes(2)
    w.unmount()
  })
  it('locks repeated submissions through saving and refresh and skips already-matching templates', async () => {
    let done!: () => void
    const reload=vi.fn(() => new Promise<void>(resolve => { done=resolve })), w=render(reload)
    await w.get('input[type="checkbox"]').setValue(true)
    await button(w,'bulkPublish').trigger('click')
    await w.get('[data-confirm]').trigger('click'); await flushPromises()
    expect(api.saveTemplate).toHaveBeenCalledTimes(1)
    expect(api.saveTemplate.mock.calls[0][0]).toMatchObject({id:3,published:true})
    expect(button(w,'bulkPublish').attributes('disabled')).toBeDefined()
    expect(w.emitted('busy')?.at(-1)).toEqual([true])
    done(); await flushPromises()
    expect(w.emitted('busy')?.at(-1)).toEqual([false])
    w.unmount()
  })
  it('duplicates as a new draft without mutating the source or calling save', async () => {
    const w=render()
    await button(w,'duplicateTemplate').trigger('click')
    const copy=w.emitted('edit')![0][0] as CardTemplate
    expect(copy).toMatchObject({id:0,version:0,published:false,kind:'group_rate',group_ids:[2]})
    copy.group_ids.push(9)
    expect(templates[0].group_ids).toEqual([2])
    expect(api.saveTemplate).not.toHaveBeenCalled()
    w.unmount()
  })
})
