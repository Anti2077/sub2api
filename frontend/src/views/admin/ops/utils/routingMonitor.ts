import type { OpsRoutingEvent } from '@/api/admin/ops'

export type RoutingEventMap = Map<string, OpsRoutingEvent>

function timestamp(event: OpsRoutingEvent): number {
  const value = Date.parse(event.occurred_at)
  return Number.isFinite(value) ? value : 0
}

export function reduceRoutingEvent(current: RoutingEventMap, event: OpsRoutingEvent | null | undefined): RoutingEventMap {
  if (!event?.route_key) return current
  const previous = current.get(event.route_key)
  if (previous && timestamp(event) < timestamp(previous)) return current
  const terminal = (item: OpsRoutingEvent) => item.event_type === 'completed' || item.event_type === 'failed'
  if (previous && timestamp(event) === timestamp(previous) && terminal(previous) && !terminal(event)) return current
  const next = new Map(current)
  next.set(event.route_key, event)
  return next
}

export function reduceRoutingSnapshot(snapshot: { active?: OpsRoutingEvent[]; recent?: OpsRoutingEvent[] } | null | undefined): RoutingEventMap {
  let next: RoutingEventMap = new Map()
  for (const event of [...(snapshot?.active || []), ...(snapshot?.recent || [])]) {
    next = reduceRoutingEvent(next, event)
  }
  return next
}

// Active requests are reconciled by authoritative snapshots. Only terminal
// events age out locally; a long running request must not become a completion.
export function pruneRoutingEvents(current: RoutingEventMap, now: number, windowMs = 30_000): RoutingEventMap {
  const next = new Map([...current].filter(([, event]) =>
    (event.event_type !== 'completed' && event.event_type !== 'failed') || now - timestamp(event) <= windowMs
  ))
  return next.size === current.size ? current : next
}
