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
