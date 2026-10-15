import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import OpsRoutingMonitor from '../OpsRoutingMonitor.vue'
import { useRoutingMonitor, type RoutingSource } from '../../composables/useRoutingMonitor'
import type { OpsRoutingEvent, OpsRoutingSnapshot } from '@/api/admin/ops'
import zh from '@/i18n/locales/zh/admin/ops'

function event(overrides: Partial<OpsRoutingEvent> = {}): OpsRoutingEvent {
  return {event_id:'one',route_key:'request-1',occurred_at:new Date().toISOString(),event_type:'started',status:'active',user_id:1,user_label:'Ada',requested_model:'model',account_id:2,account_name:'upstream',attempt_count:1,hops:[],...overrides}
}
function makeSource(initial: OpsRoutingEvent[] = []) {
  let receive: Parameters<RoutingSource['subscribeRouting']>[0] = () => {}
  const stop = vi.fn()
  const data = (): OpsRoutingSnapshot => ({generated_at:new Date().toISOString(),active:initial,recent:[]})
  const source: RoutingSource = {
    getRoutingMonitorSnapshot: vi.fn(async () => data()),
    subscribeRouting: (cb, options) => { receive = cb; options.onStatusChange?.('connected'); return stop }
  }
  return {source,stop,send:(item:OpsRoutingEvent)=>receive({type:'routing_event',data:item})}
}
const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => { for (const wrapper of wrappers.splice(0)) wrapper.unmount(); localStorage.clear(); vi.restoreAllMocks(); vi.useRealTimers() })
function panel(source: RoutingSource) {
  const wrapper = mount(OpsRoutingMonitor,{props:{source},global:{stubs:{TransitionGroup:true},plugins:[createI18n({legacy:false,locale:'zh',messageCompiler: message => ctx => String(message).replace(/\{(\w+)\}/g, (_, key: string) => String(ctx.named(key))),messages:{zh:{admin:zh}}})]}})
  wrappers.push(wrapper)
  return wrapper
}

describe('routing monitor interactions', () => {
  it('toggles key names across the graph, routes and details, clears selection, and remembers the choice', async () => {
    const mock = makeSource([event({ api_key_id: 10, api_key_name: '开发密钥' }), event({ route_key: 'request-2', api_key_id: 11, api_key_name: '开发密钥' })])
    const wrapper = panel(mock.source); await flushPromises()
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(0)
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('开发密钥')
    const toggle = wrapper.get('input[type="checkbox"]')
    await toggle.setValue(true)
    expect(wrapper.findAll('svg > text').map(node => node.text())).toEqual(['用户', 'API 密钥名称', '请求模型', '实际账号 / 上游'])
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(2)
    expect(wrapper.findAll('.routing-row')).toHaveLength(2)
    await wrapper.find('.routing-node-apiKey').trigger('keydown', { key: 'Enter' })
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    await wrapper.find('.routing-route-details').trigger('click')
    expect(wrapper.get('dl').text()).toContain('开发密钥 · #10')
    await toggle.setValue(false)
    expect(wrapper.find('dl').exists()).toBe(false)
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('开发密钥')
    await toggle.setValue(true)
    await wrapper.get('select.input.w-auto').setValue('25')
    const restored = panel(mock.source); await flushPromises()
    expect((restored.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
    expect(restored.findAll('.routing-node-apiKey')).toHaveLength(2)
    expect((restored.get('select.input.w-auto').element as HTMLSelectElement).value).toBe('25')
    expect(restored.get('input[type="checkbox"]').element.parentElement?.parentElement).toBe(restored.get('select.input.w-auto').element.parentElement?.parentElement)
  })

  it('expands only the clicked user without filtering other users, and collapses from the list', async () => {
    const wrapper = panel(makeSource([
      event({ api_key_id: 10, api_key_name: 'Ada key A' }),
      event({ route_key: 'request-2', api_key_id: 11, api_key_name: 'Ada key B' }),
      event({ route_key: 'request-3', user_id: 2, user_label: 'Grace', api_key_id: 12, api_key_name: 'Grace key' })
    ]).source)
    await flushPromises()
    expect(wrapper.findAll('.routing-row')).toHaveLength(2)
    const ada = () => wrapper.findAll('.routing-node-user').find(node => node.text().includes('Ada'))!
    await ada().trigger('keydown', { key: 'Enter' })
    expect(ada().attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(2)
    expect(wrapper.findAll('.routing-node-user')).toHaveLength(2)
    expect(wrapper.findAll('.routing-row')).toHaveLength(3)
    expect(wrapper.text()).not.toContain('Grace key')
    await wrapper.find('.routing-route-details').trigger('click')
    expect(wrapper.get('dl').text()).toContain('Ada key')
    await wrapper.findAll('.routing-row').find(row => row.text().includes('Ada'))!.get('button').trigger('click')
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(0)
    expect(wrapper.findAll('.routing-row')).toHaveLength(2)
    expect(wrapper.find('dl').exists()).toBe(false)
    await ada().trigger('keydown', { key: ' ' })
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(2)
    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(3)
    await wrapper.get('input[type="checkbox"]').setValue(false)
    expect(wrapper.findAll('.routing-node-apiKey')).toHaveLength(0)
  })

  it('ignores unsupported stored graph limits', async () => {
    localStorage.setItem('ops-routing-monitor-graph-limit', '1000')
    const wrapper = panel(makeSource([event()]).source); await flushPromises()
    expect((wrapper.get('select.input.w-auto').element as HTMLSelectElement).value).toBe('10')
  })

  it('keeps the toggle usable when browser storage is unavailable', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked') })
    const wrapper = panel(makeSource([event({ api_key_name: 'test-key' })]).source); await flushPromises()
    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.find('.routing-node-apiKey').text()).toContain('test-key')
  })

  it('always exposes routes and request-specific failover details, including keyboard selection', async () => {
    const mock=makeSource([event({hops:[{account_id:1,account_name:'first',occurred_at:new Date().toISOString()},{account_id:2,account_name:'last',occurred_at:new Date().toISOString()}],attempt_count:2})])
    const wrapper=panel(mock.source); await flushPromises()
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    await wrapper.find('.routing-node-model').trigger('keydown',{key:'Enter'})
    expect(wrapper.find('.routing-node-model').attributes('aria-pressed')).toBe('true')
    await wrapper.find('.routing-route-details').trigger('click')
    expect(wrapper.text()).toContain('first · #1')
    expect(wrapper.text()).toContain('last · #2')
    expect(wrapper.findAll('.routing-edge.is-highlighted')).toHaveLength(2)
  })
  it('lets the standalone page own the heading while keeping live controls visible', async () => {
    const mock=makeSource([event()]); const wrapper=panel(mock.source); await flushPromises()
    expect(wrapper.find('h2').exists()).toBe(true)
    await wrapper.setProps({showHeading:false})
    expect(wrapper.find('h2').exists()).toBe(false)
    expect(wrapper.text()).toContain('已连接')
    expect(wrapper.findAll('button').some(button => button.text() === '暂停画面')).toBe(true)
  })

  it('freezes the view while events keep arriving and resumes without losing requests', async () => {
    const mock=makeSource([event()]); const wrapper=panel(mock.source); await flushPromises()
    const pause = () => wrapper.findAll('button').find(b => ['暂停画面','恢复实时'].includes(b.text()))!
    await pause().trigger('click')
    mock.send(event({route_key:'request-2',user_id:3,user_label:'Grace'})); await nextTick()
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    await pause().trigger('click')
    expect(wrapper.findAll('.routing-row')).toHaveLength(2)
  })
  it('expires terminal events and selected details while idle', async () => {
    vi.useFakeTimers(); vi.setSystemTime(new Date('2026-09-08T00:00:00Z'))
    const mock=makeSource([event({event_type:'completed',status:'OK'})]); const wrapper=panel(mock.source); await flushPromises()
    await wrapper.find('.routing-route-details').trigger('click')
    await vi.advanceTimersByTimeAsync(61_000)
    expect(wrapper.findAll('.routing-row')).toHaveLength(0)
    expect(wrapper.text()).toContain('最近 1 分钟暂无路由事件')
  })

  it('offers dynamic user, model, and account dropdown filters without a platform filter', async () => {
    const mock = makeSource([event({ user_id: 7, user_label: 'Grace', requested_model: 'gpt-5.6-sol', account_id: 9, account_name: 'primary' })])
    const wrapper = panel(mock.source)
    await flushPromises()
    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(4) // three routing filters plus graph limit
    expect(selects[0].text()).toContain('Grace · #7')
    expect(selects[1].text()).toContain('gpt-5.6-sol')
    expect(selects[2].text()).toContain('primary · #9')
    expect(wrapper.text()).not.toContain('平台')
  })
})

describe('routing subscription lifecycle', () => {
  it('does not overwrite a newer completion with a late HTTP snapshot', async () => {
    const mock=makeSource(); let resolve!: (data:OpsRoutingSnapshot)=>void
    mock.source.getRoutingMonitorSnapshot=()=>new Promise(r=>{resolve=r})
    let state!: ReturnType<typeof useRoutingMonitor>
    const wrapper=mount(defineComponent({setup(){state=useRoutingMonitor(mock.source);return()=>h('div')}})); wrappers.push(wrapper)
    const completedAt = Date.now()
    const staleStarted = event({ occurred_at: new Date(completedAt - 1000).toISOString() })
    mock.send(event({event_type:'completed',status:'OK',occurred_at:new Date(completedAt).toISOString()}))
    resolve({generated_at:new Date(completedAt).toISOString(),active:[staleStarted],recent:[]}); await flushPromises()
    expect(state.events.value.get('request-1')?.event_type).toBe('completed')
  })
  it('reconciles orphaned active requests even when live traffic arrives during refresh', async () => {
    const mock=makeSource([event()]); let state!: ReturnType<typeof useRoutingMonitor>
    const wrapper=mount(defineComponent({setup(){state=useRoutingMonitor(mock.source);return()=>h('div')}})); wrappers.push(wrapper)
    await flushPromises()
    let resolve!: (data:OpsRoutingSnapshot)=>void
    mock.source.getRoutingMonitorSnapshot=()=>new Promise(r=>{resolve=r})
    const refresh=state.refresh()
    mock.send(event({route_key:'new-request'}))
    resolve({generated_at:new Date().toISOString(),active:[],recent:[]})
    await refresh
    expect([...state.events.value.keys()]).toEqual(['new-request'])
  })

  it('shows snapshot fallback instead of staying in reconnecting after HTTP succeeds', async () => {
    const mock = makeSource([event()])
    mock.source.subscribeRouting = (_cb, options) => {
      options.onStatusChange?.('reconnecting')
      return mock.stop
    }
    let state!: ReturnType<typeof useRoutingMonitor>
    const wrapper = mount(defineComponent({ setup() { state = useRoutingMonitor(mock.source); return () => h('div') } }))
    wrappers.push(wrapper)
    await flushPromises()
    expect(state.status.value).toBe('polling')
  })

  it('unsubscribes even when the initial request has not returned on unmount', async () => {
    const mock=makeSource(); let resolve!: (data:OpsRoutingSnapshot)=>void
    mock.source.getRoutingMonitorSnapshot=()=>new Promise(r=>{resolve=r})
    const wrapper=panel(mock.source); wrapper.unmount(); wrappers.pop()
    expect(mock.stop).toHaveBeenCalledOnce()
    resolve({generated_at:new Date().toISOString(),active:[],recent:[]}); await flushPromises()
    expect(mock.stop).toHaveBeenCalledOnce()
  })
})
