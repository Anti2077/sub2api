<template>
  <section class="routing-monitor space-y-4" :class="{ 'is-paused': paused }" :aria-label="t(`${prefix}.title`)">
    <div class="routing-panel flex flex-wrap items-start justify-between gap-4 p-5">
      <div class="max-w-2xl">
        <h2 v-if="showHeading !== false" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t(`${prefix}.title`) }}</h2>
        <p v-if="showHeading !== false" class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t(`${prefix}.description`) }}</p>
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t(`${prefix}.lastUpdated`) }} {{ updatedAt ? new Date(updatedAt).toLocaleTimeString() : '—' }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span class="routing-status" role="status">
          <span class="h-2 w-2 rounded-full" :class="status === 'connected' ? 'bg-emerald-500' : 'bg-amber-500'" />
          {{ t(`${prefix}.status.${status}`) }}
        </span>
        <button type="button" class="routing-button" :aria-pressed="paused" @click="togglePause">{{ t(`${prefix}.${paused ? 'resume' : 'pause'}`) }}</button>
        <button type="button" class="routing-button" :disabled="refreshing || disabled" @click="refresh">{{ t(`${prefix}.${refreshing ? 'refreshing' : 'refresh'}`) }}</button>
      </div>
    </div>

    <p v-if="disabled || loadError || paused" class="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200" role="status">
      {{ t(`${prefix}.${disabled ? 'disabled' : loadError ? 'loadFailed' : 'pausedHint'}`) }}
    </p>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div v-for="metric in metrics" :key="metric.key" class="routing-panel px-5 py-4">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">{{ t(`${prefix}.metrics.${metric.key}`) }}</div>
        <div class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ metric.value }}</div>
      </div>
    </div>

    <div class="routing-panel grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-3">
      <label class="text-xs font-medium text-gray-600 dark:text-gray-300">
        {{ t(`${prefix}.filters.user`) }}
        <select v-model="filters.user" class="input mt-1 w-full">
          <option value="">{{ t(`${prefix}.filters.all`) }}</option>
          <option v-for="option in userOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </label>
      <label class="text-xs font-medium text-gray-600 dark:text-gray-300">
        {{ t(`${prefix}.filters.requestedModel`) }}
        <select v-model="filters.requestedModel" class="input mt-1 w-full">
          <option value="">{{ t(`${prefix}.filters.all`) }}</option>
          <option v-for="option in modelOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </label>
      <label class="text-xs font-medium text-gray-600 dark:text-gray-300">
        {{ t(`${prefix}.filters.account`) }}
        <select v-model="filters.account" class="input mt-1 w-full">
          <option value="">{{ t(`${prefix}.filters.all`) }}</option>
          <option v-for="option in accountOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </label>
    </div>

    <div v-if="!routes.length" class="routing-panel px-6 py-16 text-center" role="status">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t(`${prefix}.${hasFilters ? 'noMatches' : status === 'connecting' ? 'waiting' : 'empty'}`) }}</p>
      <button v-if="hasFilters" type="button" class="routing-button mt-4" @click="clearFilters">{{ t(`${prefix}.clearFilters`) }}</button>
    </div>
    <template v-else>
      <div class="routing-panel overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t(`${prefix}.graphTitle`) }}</h3>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ t(`${prefix}.graphHint`) }}</p>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <label class="routing-button inline-flex cursor-pointer items-center gap-2">
              <input v-model="expandAllAPIKeys" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t(`${prefix}.showAPIKeys`) }}
            </label>
            <label class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
              {{ t(`${prefix}.graphLimit`) }}
              <select v-model.number="graphLimit" class="input w-auto py-2"><option :value="10">10</option><option :value="25">25</option><option :value="50">50</option></select>
            </label>
          </div>
        </div>
        <div class="hidden max-h-[620px] overflow-auto p-3 md:block" tabindex="0" :aria-label="t(`${prefix}.${hasExpandedAPIKeys ? 'graphLabelWithAPIKeys' : 'graphLabel'}`)">
          <svg :viewBox="`0 0 ${graph.width} ${graph.height}`" class="block w-full" :class="hasExpandedAPIKeys ? 'min-w-[1200px]' : 'min-w-[1024px]'" role="group" :aria-label="t(`${prefix}.${hasExpandedAPIKeys ? 'graphLabelWithAPIKeys' : 'graphLabel'}`)">
            <text v-for="column in graph.columns" :key="column.key" :x="column.x" y="32" class="fill-gray-600 text-[15px] font-semibold dark:fill-gray-300">{{ t(`${prefix}.columns.${column.heading}`) }}</text>
            <TransitionGroup name="routing-edge-motion" tag="g" move-class="routing-no-move" appear>
              <g v-for="(edge, index) in graph.edges" :key="edge.key" role="button" tabindex="0" class="routing-edge" :class="{ 'is-dimmed': hasSelection && !edgeHighlighted(edge), 'is-highlighted': edgeHighlighted(edge), 'is-active': edge.active > 0 }" :style="{ '--flow-delay': `${-(index % 7) * 0.43}s` }" :aria-label="`${edge.source.label} → ${edge.target.label}: ${edge.count}`" :aria-pressed="focus?.kind === 'edge' && focus.key === edge.key" @click="selectFocus('edge', edge.key)" @keydown.enter.prevent="selectFocus('edge', edge.key)" @keydown.space.prevent="selectFocus('edge', edge.key)">
                <title>{{ edge.source.label }} → {{ edge.target.label }} · {{ edge.count }}</title>
                <path :d="edge.path" class="routing-edge-hit" />
                <path :d="edge.path" class="routing-edge-line" :class="{ 'is-active': edge.active > 0 }" :stroke-width="Math.min(7, 2 + Math.sqrt(edge.count))" />
                <path :d="edge.path" pathLength="100" class="routing-edge-flow routing-edge-flow-glow" aria-hidden="true" />
                <path :d="edge.path" pathLength="100" class="routing-edge-flow" aria-hidden="true" />
                <path :d="`M ${edge.target.x - 7} ${edge.target.y + edge.target.height / 2 - 4} l 7 4 l -7 4`" class="routing-arrow" />
              </g>
            </TransitionGroup>
            <TransitionGroup name="routing-node-motion" tag="g" move-class="routing-no-move" appear>
              <g v-for="node in graph.nodes" :key="node.key" role="button" tabindex="0" class="routing-node" :style="{ transform: `translate(${node.x}px, ${node.y}px)` }" :class="[`routing-node-${node.type}`, { 'is-dimmed': hasSelection && !nodeHighlighted(node), 'is-highlighted': nodeHighlighted(node) }]" :aria-label="`${node.label}: ${node.count}`" :aria-pressed="node.type === 'user' && !expandAllAPIKeys ? undefined : focus?.kind === 'node' && focus.key === node.key" :aria-expanded="node.type === 'user' && !expandAllAPIKeys ? isUserExpanded(node.key) : undefined" @click="selectNode(node)" @keydown.enter.prevent="selectNode(node)" @keydown.space.prevent="selectNode(node)">
                <title>{{ node.label }} · {{ node.count }}</title>
                <rect x="0" y="0" :width="node.width" :height="node.height" rx="10" />
                <foreignObject x="12" y="0" :width="node.width - 24" :height="node.height" class="pointer-events-none">
                  <div class="flex h-full items-center gap-2 text-[15px] text-gray-900 dark:text-gray-100"><span v-if="node.type === 'user' && !expandAllAPIKeys" aria-hidden="true" class="shrink-0">{{ isUserExpanded(node.key) ? '▾' : '▸' }}</span><span class="min-w-0 flex-1 truncate">{{ node.label }}</span><span class="shrink-0 font-semibold tabular-nums">{{ node.count }}</span></div>
                </foreignObject>
              </g>
            </TransitionGroup>
          </svg>
        </div>
        <div class="hidden border-t border-gray-200 px-5 py-3 text-xs text-gray-600 dark:border-dark-700 dark:text-gray-300 md:block">{{ t(`${prefix}.shownRoutes`, { shown: graphRoutes.length, total: routes.length }) }} · {{ t(`${prefix}.edgeHint`) }}</div>
      </div>

      <div class="routing-panel overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <div><h3 class="font-semibold text-gray-900 dark:text-white">{{ t(`${prefix}.routeList`) }}</h3><p class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ t(`${prefix}.routeCount`, { count: listedRoutes.length }) }}</p></div>
          <button v-if="hasSelection" type="button" class="routing-button" @click="clearSelection">{{ t(`${prefix}.clearSelection`) }}</button>
        </div>
        <div class="max-h-[480px] overflow-y-auto">
          <div v-for="route in listedRoutes" :key="route.key" class="routing-row" :class="{ 'is-selected': selectedKey === route.key }">
            <div class="grid min-w-0 flex-1 gap-2 md:grid-cols-[1fr_3.4fr]">
              <button type="button" class="min-w-0 rounded-lg p-1 text-left hover:bg-sky-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:hover:bg-sky-950" :aria-expanded="!expandAllAPIKeys ? isUserExpanded(route.userKey) : undefined" :aria-pressed="expandAllAPIKeys ? focus?.kind === 'node' && focus.key === route.userKey : undefined" @click="selectUser(route.userKey)">
                <span class="routing-row-label">{{ t(`${prefix}.columns.user`) }}</span><span class="flex items-center gap-2"><span v-if="!expandAllAPIKeys" aria-hidden="true">{{ route.apiKeyExpanded ? '▾' : '▸' }}</span><span class="truncate" :title="route.user">{{ route.user }}</span></span>
              </button>
              <button type="button" class="routing-route-details flex min-w-0 items-center gap-4 rounded-lg p-1 text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500" :aria-expanded="selectedKey === route.key" @click="selectedKey = selectedKey === route.key ? null : route.key">
                <span class="grid min-w-0 flex-1 gap-2" :class="route.apiKeyExpanded ? 'md:grid-cols-[1fr_1.2fr_1.2fr]' : 'md:grid-cols-2'">
                  <span v-if="route.apiKeyExpanded" class="min-w-0"><span class="routing-row-label">{{ t(`${prefix}.columns.apiKey`) }}</span><span class="block truncate" :title="route.apiKey">{{ route.apiKey }}</span></span>
                  <span class="min-w-0"><span class="routing-row-label">{{ t(`${prefix}.columns.requestedModel`) }}</span><span class="block truncate" :title="route.model">{{ route.model }}</span></span>
                  <span class="min-w-0"><span class="routing-row-label">{{ t(`${prefix}.columns.account`) }}</span><span class="block truncate" :title="route.account">{{ route.account }}</span></span>
                </span>
                <span class="flex shrink-0 flex-col items-end gap-1 text-xs tabular-nums">
                  <span v-if="route.active" class="font-semibold text-sky-700 dark:text-sky-300">{{ t(`${prefix}.states.active`) }} {{ route.active }}</span>
                  <span v-if="route.failed" class="font-semibold text-red-700 dark:text-red-300">{{ t(`${prefix}.states.failed`) }} {{ route.failed }}</span>
                  <span v-if="route.completed" class="text-emerald-700 dark:text-emerald-300">{{ t(`${prefix}.states.completed`) }} {{ route.completed }}</span>
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <section v-if="selectedRoute" class="routing-panel overflow-hidden" :aria-label="t(`${prefix}.details`)">
        <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-dark-700"><h3 class="font-semibold text-gray-900 dark:text-white">{{ t(`${prefix}.details`) }}</h3><button class="routing-button" type="button" @click="selectedKey = null">{{ t(`${prefix}.close`) }}</button></div>
        <div class="max-h-[540px] overflow-y-auto divide-y divide-gray-200 dark:divide-dark-700">
          <article v-for="request in selectedRoute.requests" :key="request.route_key" class="space-y-3 p-5 text-sm">
            <div class="flex flex-wrap items-center justify-between gap-2"><code class="break-all text-xs text-gray-700 dark:text-gray-200">{{ request.request_id || request.client_request_id || request.route_key }}</code><span class="routing-status">{{ t(`${prefix}.states.${routingState(request)}`) }} · {{ request.duration_ms ? `${request.duration_ms.toLocaleString()} ms` : '—' }}</span></div>
            <dl class="grid gap-2 text-xs text-gray-600 sm:grid-cols-2 dark:text-gray-300">
              <div v-if="selectedRoute.apiKeyExpanded"><dt class="inline">{{ t(`${prefix}.columns.apiKey`) }}: </dt><dd class="inline break-all">{{ routingAPIKeyLabel(request) }}</dd></div>
              <div><dt class="inline">{{ t(`${prefix}.detail.requestedModel`) }}: </dt><dd class="inline break-all">{{ request.requested_model || '—' }}</dd></div>
              <div><dt class="inline">{{ t(`${prefix}.detail.upstreamModel`) }}: </dt><dd class="inline break-all">{{ request.upstream_model || '—' }}</dd></div>
              <div><dt class="inline">{{ t(`${prefix}.detail.time`) }}: </dt><dd class="inline">{{ new Date(request.occurred_at).toLocaleTimeString() }}</dd></div>
              <div><dt class="inline">{{ t(`${prefix}.attempts`) }}: </dt><dd class="inline">{{ request.attempt_count }}</dd></div>
            </dl>
            <div><p class="routing-row-label">{{ t(`${prefix}.detail.hops`) }}</p><ol class="mt-2 flex flex-wrap gap-2"><li v-for="(hop, index) in request.hops" :key="`${index}-${hop.account_id}`" class="rounded-lg border border-gray-200 px-3 py-2 text-xs text-gray-700 dark:border-dark-600 dark:text-gray-200">{{ index + 1 }}. {{ hop.account_name || '—' }} · #{{ hop.account_id }}</li></ol></div>
            <p v-if="request.error_summary" class="break-words rounded-lg bg-red-50 p-3 text-xs text-red-700 dark:bg-red-950/40 dark:text-red-200">{{ request.error_summary }}</p>
          </article>
        </div>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type OpsRoutingEvent } from '@/api/admin/ops'
import { useRoutingMonitor, type RoutingSource } from '../composables/useRoutingMonitor'
import { buildRoutingGraph, groupRoutingEvents, routingAPIKeyLabel, routingState, type RoutingEdge, type RoutingNode } from '../utils/routingGraph'

const props = withDefaults(defineProps<{ source?: RoutingSource; showHeading?: boolean }>(), { showHeading: true })
const { t } = useI18n()
const prefix = 'admin.ops.routingMonitor'
const apiKeyDisplayStorageKey = 'ops-routing-monitor-show-api-keys'
const graphLimitStorageKey = 'ops-routing-monitor-graph-limit'
const expandAllAPIKeys = ref(false)
const graphLimit = ref(10)
const expandedUsers = ref(new Set<string>())
try {
  expandAllAPIKeys.value = localStorage.getItem(apiKeyDisplayStorageKey) === 'true'
  const savedLimit = Number(localStorage.getItem(graphLimitStorageKey))
  if ([10, 25, 50].includes(savedLimit)) graphLimit.value = savedLimit
} catch { /* Storage may be unavailable. */ }
watch(expandAllAPIKeys, value => {
  clearSelection()
  expandedUsers.value = new Set()
  try { localStorage.setItem(apiKeyDisplayStorageKey, String(value)) } catch { /* Keep the toggle usable without storage. */ }
})
watch(graphLimit, value => {
  try { localStorage.setItem(graphLimitStorageKey, String(value)) } catch { /* Keep the limit usable without storage. */ }
})
const { events, status, disabled, loadError, refreshing, updatedAt, refresh } = useRoutingMonitor(props.source ?? opsAPI)
const frozenEvents = ref<OpsRoutingEvent[] | null>(null)
const paused = computed(() => frozenEvents.value !== null)
function togglePause() { frozenEvents.value = paused.value ? null : [...events.value.values()] }
const filters = reactive({ user: '', requestedModel: '', account: '' })
const sourceEvents = computed(() => frozenEvents.value ?? [...events.value.values()])
type FilterOption = { value: string; label: string }
function userFilterValue(event: OpsRoutingEvent) {
  if (event.user_id !== undefined) return `id:${event.user_id}`
  const label = (event.user_label || '').trim()
  return label ? `label:${label}` : ''
}
function accountFilterValue(accountId: number | undefined, accountName: string | undefined) {
  if (accountId !== undefined && accountId > 0) return `id:${accountId}`
  const name = (accountName || '').trim()
  return name ? `name:${name}` : ''
}
function sortedOptions(options: Map<string, string>): FilterOption[] {
  return [...options.entries()]
    .map(([value, label]) => ({ value, label }))
    .sort((left, right) => left.label.localeCompare(right.label))
}
const userOptions = computed(() => {
  const options = new Map<string, string>()
  for (const event of sourceEvents.value) {
    const value = userFilterValue(event)
    if (!value) continue
    options.set(value, event.user_label ? `${event.user_label} · #${event.user_id ?? '-'}` : `#${event.user_id}`)
  }
  return sortedOptions(options)
})
const modelOptions = computed(() => {
  const options = new Map<string, string>()
  for (const event of sourceEvents.value) {
    const model = (event.requested_model || '').trim()
    if (model) options.set(model, model)
  }
  return sortedOptions(options)
})
const accountOptions = computed(() => {
  const options = new Map<string, string>()
  for (const event of sourceEvents.value) {
    const value = accountFilterValue(event.account_id, event.account_name)
    if (!value) continue
    options.set(value, `${event.account_name || '-'} · #${event.account_id ?? '-'}`)
  }
  return sortedOptions(options)
})
const hasFilters = computed(() => Object.values(filters).some(Boolean))
function clearFilters() { filters.user = ''; filters.requestedModel = ''; filters.account = ''; clearSelection() }
const filteredEvents = computed(() => sourceEvents.value.filter(event => {
  return (!filters.user || userFilterValue(event) === filters.user)
    && (!filters.requestedModel || event.requested_model === filters.requestedModel)
    && (!filters.account || accountFilterValue(event.account_id, event.account_name) === filters.account)
}))
const routes = computed(() => groupRoutingEvents(filteredEvents.value, expandAllAPIKeys.value, expandedUsers.value))
const graphRoutes = computed(() => routes.value.slice(0, graphLimit.value))
const graph = computed(() => buildRoutingGraph(graphRoutes.value))
const hasExpandedAPIKeys = computed(() => graph.value.columns.some(column => column.type === 'apiKey'))
const selectedKey = ref<string | null>(null)
const focus = ref<{ kind: 'node' | 'edge'; key: string } | null>(null)
const focusedRoutes = computed(() => {
  if (!focus.value) return null
  const items = focus.value.kind === 'node' ? graph.value.nodes : graph.value.edges
  return items.find(item => item.key === focus.value?.key)?.routes ?? null
})
const selectedRoute = computed(() => routes.value.find(route => route.key === selectedKey.value) ?? null)
const hasSelection = computed(() => selectedRoute.value !== null || focusedRoutes.value !== null)
const highlightedKeys = computed(() => selectedRoute.value ? [selectedRoute.value.key] : focusedRoutes.value ?? [])
const listedRoutes = computed(() => focusedRoutes.value ? routes.value.filter(route => focusedRoutes.value!.includes(route.key)) : routes.value)
function isUserExpanded(key: string) { return expandAllAPIKeys.value || expandedUsers.value.has(key) }
function selectUser(key: string) {
  if (expandAllAPIKeys.value) { selectFocus('node', key); return }
  clearSelection()
  const next = new Set(expandedUsers.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedUsers.value = next
}
function selectNode(node: RoutingNode) {
  if (node.type === 'user') selectUser(node.key)
  else selectFocus('node', node.key)
}
function selectFocus(kind: 'node' | 'edge', key: string) {
  selectedKey.value = null
  focus.value = focus.value?.kind === kind && focus.value.key === key ? null : { kind, key }
}
function clearSelection() { focus.value = null; selectedKey.value = null }
function nodeHighlighted(node: RoutingNode) { return node.routes.some(key => highlightedKeys.value.includes(key)) }
function edgeHighlighted(edge: RoutingEdge) { return edge.routes.some(key => highlightedKeys.value.includes(key)) }
const metrics = computed(() => [
  { key: 'requests', value: filteredEvents.value.length },
  ...(['active', 'completed', 'failed'] as const).map(key => ({ key, value: filteredEvents.value.filter(event => routingState(event) === key).length }))
])
</script>

<style scoped>
.routing-panel { @apply rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900; }
.routing-button { @apply min-h-[44px] rounded-lg border border-gray-300 bg-white px-3 py-2 text-xs font-medium text-gray-700 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700; }
.routing-status { @apply inline-flex items-center gap-2 rounded-full bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-700 dark:bg-dark-800 dark:text-gray-200; }
.routing-row { @apply flex w-full items-center gap-4 border-b border-gray-100 px-5 py-4 text-sm text-gray-800 last:border-b-0 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-primary-500 dark:border-dark-800 dark:text-gray-100 dark:hover:bg-dark-800; }
.routing-row.is-selected { @apply bg-primary-50 dark:bg-primary-950/40; }
.routing-row-label { @apply mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400; }
.routing-edge, .routing-node { cursor: pointer; transition: opacity 200ms ease; }
.routing-node { transition: transform 420ms cubic-bezier(0.22, 1, 0.36, 1), opacity 200ms ease; }
.routing-edge path { transition: d 420ms cubic-bezier(0.22, 1, 0.36, 1), stroke 180ms ease, stroke-width 180ms ease; }
.routing-node rect { transition: fill 180ms ease, stroke 180ms ease, stroke-width 180ms ease; }
.routing-edge-flow {
  @apply stroke-slate-400 dark:stroke-slate-300;
  fill: none;
  stroke-width: 3;
  stroke-linecap: round;
  stroke-dasharray: 2 98;
  pointer-events: none;
  opacity: 0.55;
  animation: routing-flow 5s linear infinite;
  animation-delay: var(--flow-delay, 0s);
}
.routing-edge.is-active .routing-edge-flow { @apply stroke-sky-500 dark:stroke-sky-300; opacity: 0.95; animation-duration: 2.6s; }
.routing-edge-flow.routing-edge-flow-glow { stroke-width: 9; opacity: 0.12; }
.routing-edge.is-active .routing-edge-flow-glow { opacity: 0.2; }
.routing-edge.is-highlighted .routing-edge-flow { @apply stroke-sky-500 dark:stroke-sky-200; }
/* Positions already transition in SVG coordinates; disable TransitionGroup FLIP. */
.routing-no-move { transition: none !important; }
.routing-edge-motion-enter-active, .routing-node-motion-enter-active { transition: opacity 320ms ease; }
.routing-edge-motion-leave-active, .routing-node-motion-leave-active { transition: opacity 140ms ease; pointer-events: none; }
.routing-edge-motion-enter-from, .routing-edge-motion-leave-to, .routing-node-motion-enter-from, .routing-node-motion-leave-to { opacity: 0; }
.routing-monitor.is-paused .routing-edge-flow { animation-play-state: paused; }
@keyframes routing-flow { from { stroke-dashoffset: 0; } to { stroke-dashoffset: -100; } }
.routing-edge-hit { fill: none; stroke: transparent; stroke-width: 18; pointer-events: stroke; }
.routing-edge-line { fill: none; stroke: #94a3b8; pointer-events: none; }
.routing-edge-line.is-active { @apply stroke-sky-600 dark:stroke-sky-400; stroke-dasharray: 6 4; }
.routing-arrow { @apply stroke-slate-500 dark:stroke-slate-300; fill: none; stroke-width: 2; pointer-events: none; }
.routing-edge.is-highlighted .routing-edge-line, .routing-edge:hover .routing-edge-line, .routing-edge:focus .routing-edge-line { @apply stroke-sky-600 dark:stroke-sky-400; stroke-width: 5; }
.routing-edge:focus { outline: none; }
.routing-edge:focus .routing-edge-hit { stroke: #bae6fd; stroke-width: 12; }
.routing-node rect { stroke-width: 1.5; }
.routing-node-user rect { @apply fill-sky-50 stroke-sky-300 dark:fill-sky-950 dark:stroke-sky-700; }
.routing-node-apiKey rect { @apply fill-amber-50 stroke-amber-300 dark:fill-amber-950 dark:stroke-amber-700; }
.routing-node-model rect { @apply fill-violet-50 stroke-violet-300 dark:fill-violet-950 dark:stroke-violet-700; }
.routing-node-account rect { @apply fill-emerald-50 stroke-emerald-300 dark:fill-emerald-950 dark:stroke-emerald-700; }
.routing-node.is-highlighted rect, .routing-node:hover rect, .routing-node:focus rect { @apply stroke-sky-600 dark:stroke-sky-400; stroke-width: 3; }
.routing-node:focus { outline: none; }
.is-dimmed { opacity: 0.22; }
@media (prefers-reduced-motion: reduce) {
  .routing-monitor *, .routing-monitor *::before, .routing-monitor *::after { animation: none !important; transition: none !important; }
  .routing-edge-flow { display: none; }
}
</style>
