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
afterEach(() => { for (const wrapper of wrappers.splice(0)) wrapper.unmount(); vi.useRealTimers() })
function panel(source: RoutingSource) {
  const wrapper = mount(OpsRoutingMonitor,{props:{source},global:{plugins:[createI18n({legacy:false,locale:'zh',messageCompiler: message => ctx => String(message).replace(/\{(\w+)\}/g, (_, key: string) => String(ctx.named(key))),messages:{zh:{admin:zh}}})]}})
  wrappers.push(wrapper)
  return wrapper
}

describe('routing monitor interactions', () => {
  it('always exposes routes and request-specific failover details, including keyboard selection', async () => {
    const mock=makeSource([event({hops:[{account_id:1,account_name:'first',occurred_at:new Date().toISOString()},{account_id:2,account_name:'last',occurred_at:new Date().toISOString()}],attempt_count:2})])
    const wrapper=panel(mock.source); await flushPromises()
    expect(wrapper.findAll('.routing-row')).toHaveLength(1)
    await wrapper.find('.routing-node').trigger('keydown',{key:'Enter'})
    expect(wrapper.find('.routing-node').attributes('aria-pressed')).toBe('true')
    await wrapper.find('.routing-row').trigger('click')
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
    await wrapper.find('.routing-row').trigger('click')
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
    mock.send(event({event_type:'completed',status:'OK'}))
    resolve({generated_at:new Date().toISOString(),active:[event()],recent:[]}); await flushPromises()
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
