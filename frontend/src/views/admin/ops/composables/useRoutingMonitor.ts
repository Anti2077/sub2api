import { onMounted, onUnmounted, ref } from 'vue'
import type { OpsRoutingEvent, OpsRoutingSnapshot, OpsWSStatus, SubscribeRoutingOptions } from '@/api/admin/ops'
import { pruneRoutingEvents, reduceRoutingEvent, reduceRoutingSnapshot } from '../utils/routingMonitor'

export interface RoutingSource {
  getRoutingMonitorSnapshot: () => Promise<OpsRoutingSnapshot>
  subscribeRouting: (onMessage: (data: { type: string; data?: OpsRoutingEvent | OpsRoutingSnapshot }) => void, options: SubscribeRoutingOptions) => () => void
}

export type RoutingMonitorStatus = OpsWSStatus | 'polling'

export function useRoutingMonitor(source: RoutingSource) {
  const events = ref(new Map<string, OpsRoutingEvent>())
  const status = ref<RoutingMonitorStatus>('connecting')
  const disabled = ref(false)
  const loadError = ref(false)
  const refreshing = ref(false)
  const updatedAt = ref<number | null>(null)
  let disposed = false
  let revision = 0
  let websocketSnapshotRevision = 0
  let eventRevisions = new Map<string, number>()
  let clockOffset = 0
  let unsubscribe: (() => void) | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  let refreshTimer: ReturnType<typeof setInterval> | undefined
  let hasHealthySnapshot = false

  function snapshot(data: OpsRoutingSnapshot, preserveAfter?: number) {
    const serverTime = Date.parse(data.generated_at)
    if (Number.isFinite(serverTime)) clockOffset = serverTime - Date.now()
    let next = reduceRoutingSnapshot(data)
    if (preserveAfter !== undefined) {
      for (const [key, event] of events.value) {
        if ((eventRevisions.get(key) ?? 0) > preserveAfter) next = reduceRoutingEvent(next, event)
      }
    }
    events.value = pruneRoutingEvents(next, Date.now() + clockOffset)
    eventRevisions = new Map([...eventRevisions].filter(([key]) => events.value.has(key)))
    updatedAt.value = Date.now()
    loadError.value = false
    hasHealthySnapshot = true
    if (!disabled.value && status.value !== 'connected') status.value = 'polling'
  }
  function onMessage(message: { type: string; data?: OpsRoutingEvent | OpsRoutingSnapshot }) {
    if (disposed) return
    if (message.type === 'routing_snapshot' && message.data) {
      revision++
      websocketSnapshotRevision = revision
      eventRevisions.clear()
      snapshot(message.data as OpsRoutingSnapshot)
    } else if (message.type === 'routing_event' && message.data && 'route_key' in message.data) {
      revision++
      eventRevisions.set(message.data.route_key, revision)
      events.value = pruneRoutingEvents(reduceRoutingEvent(events.value, message.data), Date.now() + clockOffset)
      updatedAt.value = Date.now()
      loadError.value = false
    }
  }
  async function refresh() {
    if (disposed || refreshing.value || disabled.value) return
    const version = revision
    refreshing.value = true
    try {
      const data = await source.getRoutingMonitorSnapshot()
      // Reconcile missing/expired requests even under continuous traffic, but
      // preserve events received while this HTTP request was in flight. A newer
      // WebSocket snapshot is already authoritative and supersedes this response.
      if (!disposed && websocketSnapshotRevision <= version) { revision++; snapshot(data, version) }
    } catch {
      if (!disposed && version === revision) loadError.value = true
    } finally {
      if (!disposed) refreshing.value = false
    }
  }
  function onVisible() { if (!document.hidden) void refresh() }
  onMounted(() => {
    unsubscribe = source.subscribeRouting(onMessage, {
      onStatusChange: value => {
        if (disposed) return
        // Keep the HTTP snapshot fallback usable when a reverse proxy cannot
        // keep the admin WebSocket open. A transient WS reconnect should not
        // make an otherwise live monitor appear unavailable.
        status.value = value === 'connected' || !hasHealthySnapshot ? value : 'polling'
      },
      onFatalClose: () => { if (!disposed) disabled.value = true }
    })
    void refresh()
    timer = setInterval(() => { events.value = pruneRoutingEvents(events.value, Date.now() + clockOffset) }, 1000)
    refreshTimer = setInterval(() => { if (!document.hidden) void refresh() }, 15_000)
    document.addEventListener('visibilitychange', onVisible)
  })
  onUnmounted(() => {
    disposed = true
    unsubscribe?.()
    clearInterval(timer)
    clearInterval(refreshTimer)
    document.removeEventListener('visibilitychange', onVisible)
  })
  return { events, status, disabled, loadError, refreshing, updatedAt, refresh }
}
