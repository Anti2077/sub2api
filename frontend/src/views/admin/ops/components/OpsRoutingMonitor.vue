<template>
  <section class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.routingMonitor.title') }}</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.routingMonitor.description') }}</p>
      </div>
      <div class="flex items-center gap-2 text-xs">
        <span class="inline-flex items-center gap-1.5 rounded-full bg-gray-100 px-2.5 py-1 font-medium text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          <span class="h-2 w-2 rounded-full" :class="statusDotClass" />
          {{ statusLabel }}
        </span>
        <button type="button" class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-800" @click="loadSnapshot">
          {{ t('admin.ops.routingMonitor.refresh') }}
        </button>
      </div>
    </div>

    <div v-if="connectionStatus === 'closed' && fatalDisabled" class="border-b border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
      {{ t('admin.ops.routingMonitor.disabled') }}
    </div>
    <div v-else-if="loadError" class="border-b border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200">
      {{ t('admin.ops.routingMonitor.loadFailed') }}
    </div>

    <div class="grid grid-cols-1 gap-3 border-b border-gray-200 p-4 md:grid-cols-4 dark:border-dark-700">
      <label class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.routingMonitor.filters.platform') }}
        <input v-model="filters.platform" class="mt-1 w-full rounded-lg border-gray-300 bg-white text-sm dark:border-dark-600 dark:bg-dark-800 dark:text-white" :placeholder="t('admin.ops.routingMonitor.filters.all')" />
      </label>
      <label class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.routingMonitor.filters.user') }}
        <input v-model="filters.user" class="mt-1 w-full rounded-lg border-gray-300 bg-white text-sm dark:border-dark-600 dark:bg-dark-800 dark:text-white" />
      </label>
      <label class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.routingMonitor.filters.requestedModel') }}
        <input v-model="filters.requestedModel" class="mt-1 w-full rounded-lg border-gray-300 bg-white text-sm dark:border-dark-600 dark:bg-dark-800 dark:text-white" />
      </label>
      <label class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.routingMonitor.filters.account') }}
        <input v-model="filters.account" class="mt-1 w-full rounded-lg border-gray-300 bg-white text-sm dark:border-dark-600 dark:bg-dark-800 dark:text-white" />
      </label>
    </div>

    <div v-if="!hasEvents" class="flex min-h-[360px] flex-col items-center justify-center px-6 text-center text-sm text-gray-500 dark:text-gray-400">
      <span class="mb-3 text-3xl text-gray-300 dark:text-dark-500">&#8594;</span>
      <p>{{ connectionStatus === 'connecting' || connectionStatus === 'reconnecting' ? t('admin.ops.routingMonitor.waiting') : t('admin.ops.routingMonitor.empty') }}</p>
    </div>
    <template v-else>
      <div class="overflow-x-auto p-4">
        <svg viewBox="0 0 1000 460" class="min-w-[760px] w-full" role="img" :aria-label="t('admin.ops.routingMonitor.graphLabel')">
          <text x="36" y="24" class="fill-gray-500 text-[13px]">{{ t('admin.ops.routingMonitor.columns.user') }}</text>
          <text x="405" y="24" class="fill-gray-500 text-[13px]">{{ t('admin.ops.routingMonitor.columns.requestedModel') }}</text>
          <text x="755" y="24" class="fill-gray-500 text-[13px]">{{ t('admin.ops.routingMonitor.columns.account') }}</text>
          <g v-for="edge in graphEdges" :key="edge.key" class="cursor-pointer" @click="selectRoute(edge.route)">
            <path :d="edge.path" fill="none" :stroke="edge.route.failed ? '#ef4444' : edge.route.active ? '#0ea5e9' : '#94a3b8'" :stroke-width="edge.width" stroke-linecap="round" stroke-opacity="0.65" />
          </g>
          <g v-for="node in graphNodes" :key="node.key" class="cursor-pointer" @click="selectNode(node)">
            <rect :x="node.x" :y="node.y" :width="node.width" height="38" rx="7" :class="nodeClass(node)" />
            <text :x="node.x + 12" :y="node.y + 24" class="pointer-events-none fill-gray-800 text-[12px] dark:fill-gray-100">{{ truncate(node.label, node.type === 'user' ? 25 : 27) }}</text>
            <text v-if="node.count > 1" :x="node.x + node.width - 12" :y="node.y + 24" text-anchor="end" class="pointer-events-none fill-gray-500 text-[11px]">{{ node.count }}</text>
          </g>
        </svg>
      </div>
      <div class="flex items-center justify-between border-t border-gray-200 px-4 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
        <span>{{ t('admin.ops.routingMonitor.routeCount', { count: graphRoutes.length }) }}</span>
        <button v-if="graphRoutes.length > 10" type="button" class="font-semibold text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="showAll = !showAll">
          {{ showAll ? t('admin.ops.routingMonitor.top10') : t('admin.ops.routingMonitor.viewAll') }}
        </button>
      </div>
      <div v-if="showAll" class="max-h-[420px] overflow-y-auto border-t border-gray-200 dark:border-dark-700">
        <button v-for="route in graphRoutes" :key="`list-${route.key}`" type="button" class="flex w-full items-center justify-between border-b border-gray-100 px-4 py-3 text-left text-xs hover:bg-gray-50 dark:border-dark-800 dark:hover:bg-dark-800" @click="selectRoute(route)">
          <span class="min-w-0 truncate text-gray-700 dark:text-gray-200">{{ route.user }} → {{ route.requestedModel }} → {{ route.accountPath }}</span>
          <span class="ml-3 shrink-0 text-gray-500">{{ route.count }}</span>
        </button>
      </div>
    </template>

    <div v-if="selectedRoute" class="border-t border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-950/50">
      <div class="mb-3 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.routingMonitor.details') }}</h3>
        <button type="button" class="text-gray-400 hover:text-gray-700 dark:hover:text-white" :aria-label="t('admin.ops.routingMonitor.close')" @click="selectedRoute = null">×</button>
      </div>
      <div class="grid grid-cols-1 gap-2 text-xs text-gray-600 sm:grid-cols-2 dark:text-gray-300">
        <div><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.status') }}:</span> {{ selectedRoute.status }}</div>
        <div><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.duration') }}:</span> {{ formatDuration(selectedRoute.durationMs) }}</div>
        <div><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.time') }}:</span> {{ formatTime(selectedRoute.occurredAt) }}</div>
        <div class="truncate"><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.requestId') }}:</span> {{ selectedRoute.requestId || selectedRoute.clientRequestId || '—' }}</div>
        <div><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.requestedModel') }}:</span> {{ selectedRoute.requestedModel || '—' }}</div>
        <div><span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.upstreamModel') }}:</span> {{ selectedRoute.upstreamModel || '—' }}</div>
      </div>
      <div class="mt-3 text-xs text-gray-600 dark:text-gray-300">
        <span class="text-gray-400">{{ t('admin.ops.routingMonitor.detail.hops') }}:</span>
        {{ selectedRoute.hops.map((hop) => hop.account_name || `#${hop.account_id}`).join(' → ') || '—' }}
      </div>
      <p v-if="selectedRoute.errorSummary" class="mt-2 text-xs text-red-600 dark:text-red-300">{{ selectedRoute.errorSummary }}</p>
      <div class="mt-4">
        <div class="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.ops.routingMonitor.detail.requestList') }}</div>
        <div class="max-h-48 overflow-y-auto rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
          <div v-for="request in selectedRoute.requests" :key="`${request.route_key}-${request.event_id}`" class="grid grid-cols-1 gap-1 border-b border-gray-100 px-3 py-2 text-xs last:border-b-0 sm:grid-cols-[1fr_auto_auto] sm:items-center dark:border-dark-800">
            <span class="truncate text-gray-700 dark:text-gray-200">{{ request.request_id || request.client_request_id || '—' }}</span>
            <span :class="request.event_type === 'failed' ? 'text-red-600 dark:text-red-300' : request.status === 'active' ? 'text-sky-600 dark:text-sky-300' : 'text-emerald-600 dark:text-emerald-300'">{{ request.status }}</span>
            <span class="text-gray-500 dark:text-gray-400">{{ formatDuration(request.duration_ms || 0) }} · {{ formatTime(request.occurred_at) }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type OpsRoutingEvent, type OpsRoutingSnapshot, type OpsWSStatus } from '@/api/admin/ops'
import { reduceRoutingEvent, reduceRoutingSnapshot } from '../utils/routingMonitor'

const { t } = useI18n()
const connectionStatus = ref<OpsWSStatus>('connecting')
const fatalDisabled = ref(false)
const loadError = ref(false)
const showAll = ref(false)
const selectedRoute = ref<RouteSummary | null>(null)
const filters = reactive({ platform: '', user: '', requestedModel: '', account: '' })
const events = ref(new Map<string, OpsRoutingEvent>())
let unsubscribe: (() => void) | null = null

const statusLabel = computed(() => t(`admin.ops.routingMonitor.status.${connectionStatus.value}`))
const statusDotClass = computed(() => ({
  'bg-emerald-500': connectionStatus.value === 'connected',
  'bg-amber-500': connectionStatus.value === 'connecting' || connectionStatus.value === 'reconnecting',
  'bg-gray-400': connectionStatus.value === 'closed' || connectionStatus.value === 'offline'
}))

interface RouteSummary {
  key: string
  user: string
  requestedModel: string
  account: string
  accountPath: string
  platform: string
  count: number
  active: boolean
  failed: boolean
  status: string
  durationMs: number
  occurredAt: string
  requestId?: string
  clientRequestId?: string
  upstreamModel?: string
  errorSummary?: string
  hops: OpsRoutingEvent['hops']
  requests: OpsRoutingEvent[]
}

function applyEvent(event: OpsRoutingEvent) {
  events.value = reduceRoutingEvent(events.value, event)
}

function applySnapshot(snapshot: OpsRoutingSnapshot) {
  events.value = reduceRoutingSnapshot(snapshot)
}

function onMessage(message: any) {
  if (message?.type === 'routing_snapshot') applySnapshot(message.data)
  if (message?.type === 'routing_event') applyEvent(message.data)
  loadError.value = false
}

async function loadSnapshot() {
  try {
    applySnapshot(await opsAPI.getRoutingMonitorSnapshot())
    loadError.value = false
  } catch (error) {
    console.warn('[OpsRoutingMonitor] snapshot failed', error)
    loadError.value = true
  }
}

const filteredEvents = computed(() => Array.from(events.value.values()).filter((event) => {
  const match = (value: string | undefined, query: string) => !query || String(value || '').toLowerCase().includes(query.toLowerCase())
  const accountValues = [event.account_name, ...(event.hops || []).map((hop) => hop.account_name)].filter(Boolean).join(' ')
  return match(event.platform || event.account_platform, filters.platform) && match(event.user_label, filters.user) && match(event.requested_model, filters.requestedModel) && match(accountValues, filters.account)
}))

const graphRoutes = computed(() => {
  const grouped = new Map<string, RouteSummary>()
  for (const event of filteredEvents.value) {
    const accountPath = (event.hops || []).map((hop) => hop.account_name || `#${hop.account_id}`).join(' → ') || event.account_name || `#${event.account_id || 0}`
    const key = `${event.user_label || '—'}|${event.requested_model || '—'}|${accountPath}`
    const current = grouped.get(key)
    const summary: RouteSummary = current || {
      key, user: event.user_label || '—', requestedModel: event.requested_model || '—', account: event.account_name || `#${event.account_id || 0}`,
      accountPath,
      platform: event.platform || event.account_platform || '—', count: 0, active: false, failed: false, status: event.status || '—', durationMs: event.duration_ms || 0,
      occurredAt: event.occurred_at, requestId: event.request_id, clientRequestId: event.client_request_id, upstreamModel: event.upstream_model, errorSummary: event.error_summary, hops: event.hops || [], requests: [event]
    }
    if (current) summary.requests.push(event)
    summary.count += 1
    summary.active ||= event.status === 'active'
    summary.failed ||= event.event_type === 'failed' || event.status === 'failed'
    if (new Date(event.occurred_at).getTime() >= new Date(summary.occurredAt).getTime()) Object.assign(summary, {
      status: event.status,
      durationMs: event.duration_ms || 0,
      occurredAt: event.occurred_at,
      requestId: event.request_id,
      clientRequestId: event.client_request_id,
      upstreamModel: event.upstream_model,
      errorSummary: event.error_summary,
      hops: event.hops || [],
      accountPath
    })
    grouped.set(key, summary)
  }
  return Array.from(grouped.values()).sort((a, b) => b.count - a.count || b.occurredAt.localeCompare(a.occurredAt))
})

const visibleRoutes = computed(() => showAll.value ? graphRoutes.value : graphRoutes.value.slice(0, 10))
const hasEvents = computed(() => visibleRoutes.value.length > 0)

interface GraphNode { key: string; type: 'user' | 'model' | 'account'; label: string; x: number; y: number; width: number; count: number }
interface GraphEdge { key: string; path: string; width: number; route: RouteSummary }

const graphNodes = computed<GraphNode[]>(() => {
  const columns: Array<{ type: GraphNode['type']; x: number; width: number }> = [{ type: 'user', x: 24, width: 250 }, { type: 'model', x: 370, width: 250 }, { type: 'account', x: 720, width: 250 }]
  const maps = columns.map((column) => {
    const labels = new Map<string, number>()
    for (const route of visibleRoutes.value) {
      const label = column.type === 'user' ? route.user : column.type === 'model' ? route.requestedModel : route.accountPath
      labels.set(label, (labels.get(label) || 0) + route.count)
    }
    return { column, labels }
  })
  const nodes: GraphNode[] = []
  for (const { column, labels } of maps) Array.from(labels.entries()).forEach(([label, count], index) => nodes.push({ key: `${column.type}-${label}`, type: column.type, label, count, x: column.x, y: 48 + index * 50, width: column.width }))
  return nodes
})

const graphEdges = computed<GraphEdge[]>(() => visibleRoutes.value.map((route, index) => {
  const userIndex = graphNodes.value.findIndex((node) => node.type === 'user' && node.label === route.user)
  const modelIndex = graphNodes.value.findIndex((node) => node.type === 'model' && node.label === route.requestedModel)
  const accountIndex = graphNodes.value.findIndex((node) => node.type === 'account' && node.label === route.accountPath)
  const y1 = 67 + Math.max(userIndex, 0) * 50
  const y2 = 67 + Math.max(modelIndex, 0) * 50
  const y3 = 67 + Math.max(accountIndex, 0) * 50
  return { key: `${route.key}-${index}`, path: `M 274 ${y1} C 320 ${y1}, 335 ${y2}, 370 ${y2} M 620 ${y2} C 665 ${y2}, 680 ${y3}, 720 ${y3}`, width: Math.min(10, 2 + route.count), route }
}))

function selectRoute(route: RouteSummary) { selectedRoute.value = route }
function selectNode(node: GraphNode) {
  const matches = graphRoutes.value.filter((route) => (node.type === 'user' ? route.user : node.type === 'model' ? route.requestedModel : route.accountPath) === node.label)
  if (matches.length <= 1) {
    selectedRoute.value = matches[0] || null
    return
  }
  const latest = [...matches].sort((a, b) => b.occurredAt.localeCompare(a.occurredAt))[0]
  selectedRoute.value = {
    ...latest,
    key: `node-${node.type}-${node.label}`,
    user: node.type === 'user' ? node.label : '—',
    requestedModel: node.type === 'model' ? node.label : '—',
    account: node.type === 'account' ? node.label : '—',
    accountPath: node.type === 'account' ? node.label : '—',
    count: matches.reduce((total, route) => total + route.count, 0),
    active: matches.some((route) => route.active),
    failed: matches.some((route) => route.failed),
    requests: matches.flatMap((route) => route.requests).sort((a, b) => b.occurred_at.localeCompare(a.occurred_at))
  }
}
function nodeClass(node: GraphNode) {
  return node.type === 'user' ? 'fill-sky-50 stroke-sky-300 dark:fill-sky-950/40 dark:stroke-sky-700' : node.type === 'model' ? 'fill-violet-50 stroke-violet-300 dark:fill-violet-950/40 dark:stroke-violet-700' : 'fill-emerald-50 stroke-emerald-300 dark:fill-emerald-950/40 dark:stroke-emerald-700'
}
function truncate(value: string, max: number) { return value.length > max ? `${value.slice(0, max - 1)}…` : value }
function formatDuration(ms: number) { return ms > 0 ? `${ms} ms` : '—' }
function formatTime(value: string) { return value ? new Date(value).toLocaleTimeString() : '—' }

onMounted(async () => {
  await loadSnapshot()
  unsubscribe = opsAPI.subscribeRouting(onMessage, {
    onStatusChange: (status) => { connectionStatus.value = status },
    onFatalClose: () => { fatalDisabled.value = true }
  })
})
onUnmounted(() => { unsubscribe?.(); unsubscribe = null })
</script>
