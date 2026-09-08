// Development-only entry point, excluded from the production Vite build.
// All identities and routes below are synthetic; no production API is used.
import { createApp, h, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import OpsRoutingMonitor from '../src/views/admin/ops/components/OpsRoutingMonitor.vue'
import zh from '../src/i18n/locales/zh/admin/ops'
import en from '../src/i18n/locales/en/admin/ops'
import type { OpsRoutingEvent } from '../src/api/admin/ops'
import type { RoutingSource } from '../src/views/admin/ops/composables/useRoutingMonitor'
import '../src/style.css'

let receive: Parameters<RoutingSource['subscribeRouting']>[0] | undefined
let connection: Parameters<RoutingSource['subscribeRouting']>[1] | undefined
let fixtures: OpsRoutingEvent[] = []
function scenario(count: number) {
  const now = new Date().toISOString()
  fixtures = Array.from({ length: count }, (_, i): OpsRoutingEvent => ({
    route_key: `demo-${i}`, event_id: `demo-event-${i}`, request_id: `req-local-${String(i + 1).padStart(3, '0')}`,
    event_type: i % 5 === 0 ? 'failed' : i % 3 === 0 ? 'completed' : 'started', status: i % 5 === 0 ? 'Bad Gateway' : i % 3 === 0 ? 'OK' : 'active',
    user_id: i + 1, user_label: i < 2 ? '同名用户' : `开发团队 ${i + 1}`,
    requested_model: ['gpt-5.6-sol', 'claude-sonnet-5', 'gemini-3-pro'][i % 3], upstream_model: `upstream-model-${i % 3}`,
    platform: ['openai', 'anthropic', 'gemini'][i % 3], account_id: i + 100,
    account_name: i < 2 ? '同名上游账号' : `主力渠道 / ${i + 1} · 长名称显示测试`,
    occurred_at: now, duration_ms: i * 1200, attempt_count: i % 4 === 0 ? 2 : 1,
    error_summary: i % 5 === 0 ? '上游返回 502（本地模拟）' : undefined,
    hops: [...(i % 4 === 0 ? [{ account_id: 9, account_name: '备用账号（第一次尝试）', occurred_at: now }] : []), { account_id: i + 100, account_name: `当前账号 ${i + 100}`, occurred_at: now }]
  }))
  receive?.({ type: 'routing_snapshot', data: snapshot() })
}
function snapshot() { return { generated_at: new Date().toISOString(), active: fixtures.filter(e => e.event_type === 'started'), recent: fixtures.filter(e => e.event_type !== 'started') } }
const source: RoutingSource = {
  getRoutingMonitorSnapshot: async () => snapshot(),
  subscribeRouting: (callback, options) => { receive = callback; connection = options; options.onStatusChange?.('connected'); return () => { receive = undefined } }
}
scenario(12)
createApp({
  setup() {
    const dark = ref(true)
    const live = ref(false)
    document.documentElement.classList.add('dark')
    let stream: ReturnType<typeof setInterval> | undefined
    const control = (label: string, action: () => void) => h('button', { class: 'rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900', onClick: action }, label)
    return () => h('main', { class: 'min-h-screen bg-gray-50 p-4 dark:bg-dark-950 md:p-8' }, [
      h('div', { class: 'mx-auto mb-5 flex max-w-7xl flex-wrap items-center gap-2' }, [
        h('strong', { class: 'mr-3 text-sm text-gray-800 dark:text-gray-100' }, '本地模拟 · 不连接生产 API'),
        control('12 条链路', () => scenario(12)), control('35 条链路', () => scenario(35)), control('空数据', () => scenario(0)),
        control('失败后恢复', () => { connection?.onStatusChange?.('reconnecting'); setTimeout(() => { connection?.onStatusChange?.('connected'); scenario(12) }, 2500) }),
        control(live.value ? '停止事件' : '持续事件', () => { live.value = !live.value; if (!live.value) clearInterval(stream); else stream = setInterval(() => { const e = fixtures[2]; if (e) { const next = { ...e, event_id: `tick-${Date.now()}`, occurred_at: new Date().toISOString() }; receive?.({type:'routing_event', data:next}) } }, 2000) }),
        control(dark.value ? '切换浅色' : '切换深色', () => { dark.value = !dark.value; document.documentElement.classList.toggle('dark', dark.value) })
      ]), h('div', { class: 'mx-auto max-w-7xl' }, [h(OpsRoutingMonitor, { source })])
    ])
  }
}).use(createI18n({legacy:false,locale:'zh',messages:{zh:{admin:zh},en:{admin:en}}})).mount('#app')
