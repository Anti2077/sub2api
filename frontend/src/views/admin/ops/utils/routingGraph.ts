import type { OpsRoutingEvent } from '@/api/admin/ops'

export type RouteState = 'active' | 'completed' | 'failed'
export function routingState(event: OpsRoutingEvent): RouteState {
  if (event.event_type === 'failed') return 'failed'
  if (event.event_type === 'completed') return 'completed'
  return 'active'
}
export interface RoutingRoute {
  key: string
  userKey: string
  modelKey: string
  accountKey: string
  user: string
  model: string
  account: string
  platform: string
  requests: OpsRoutingEvent[]
  active: number
  completed: number
  failed: number
}
export interface RoutingNode {
  key: string
  type: 'user' | 'model' | 'account'
  label: string
  x: number
  y: number
  width: number
  height: number
  count: number
  routes: string[]
}
export interface RoutingEdge {
  key: string
  source: RoutingNode
  target: RoutingNode
  path: string
  count: number
  active: number
  failed: number
  routes: string[]
}

export function groupRoutingEvents(events: OpsRoutingEvent[]): RoutingRoute[] {
  const routes = new Map<string, RoutingRoute>()
  for (const event of events) {
    const platform = event.platform || event.account_platform || '—'
    const user = event.user_id ? `${event.user_label || '—'} · #${event.user_id}` : event.user_label || '—'
    const model = event.requested_model || '—'
    const account = event.account_name ? `${event.account_name}${event.account_id ? ` · #${event.account_id}` : ''}` : event.account_id ? `#${event.account_id}` : '—'
    // Display names (including masked emails) are not entity identities.
    const userKey = JSON.stringify(['user', event.user_id ?? user])
    const modelKey = JSON.stringify(['model', platform, model])
    const accountKey = JSON.stringify(['account', event.account_platform || platform, event.account_id ?? account])
    const key = JSON.stringify([userKey, modelKey, accountKey])
    let route = routes.get(key)
    if (!route) {
      route = { key, userKey, modelKey, accountKey, user, model, account, platform, requests: [], active: 0, completed: 0, failed: 0 }
      routes.set(key, route)
    }
    route.requests.push(event)
    route[routingState(event)]++
  }
  for (const route of routes.values()) route.requests.sort((a, b) => b.occurred_at.localeCompare(a.occurred_at))
  return [...routes.values()].sort((a, b) => b.active - a.active || b.requests.length - a.requests.length || a.key.localeCompare(b.key))
}

export function buildRoutingGraph(routes: RoutingRoute[]) {
  const nodes: RoutingNode[] = []
  const nodeMap = new Map<string, RoutingNode>()
  const columns = [
    { type: 'user' as const, x: 24, key: 'userKey' as const, label: 'user' as const },
    { type: 'model' as const, x: 410, key: 'modelKey' as const, label: 'model' as const },
    { type: 'account' as const, x: 796, key: 'accountKey' as const, label: 'account' as const }
  ]
  let rows = 0
  for (const column of columns) {
    const entries = new Map<string, RoutingNode>()
    for (const route of routes) {
      const key = route[column.key]
      let node = entries.get(key)
      if (!node) {
        node = { key, type: column.type, label: route[column.label], x: column.x, y: 64 + entries.size * 68, width: 280, height: 48, count: 0, routes: [] }
        entries.set(key, node)
      }
      node.count += route.requests.length
      node.routes.push(route.key)
    }
    rows = Math.max(rows, entries.size)
    const ordered = [...entries.values()].sort((a, b) => a.key.localeCompare(b.key))
    ordered.forEach((node, index) => { node.y = 64 + index * 68; nodes.push(node); nodeMap.set(node.key, node) })
  }
  // Center shorter columns as a group within the tallest node column.
  // Apply this before constructing paths so all connection endpoints follow.
  for (const column of columns) {
    const columnNodes = nodes.filter(node => node.type === column.type)
    const offset = (rows - columnNodes.length) * 68 / 2
    for (const node of columnNodes) node.y += offset
  }
  const edges = new Map<string, RoutingEdge>()
  for (const route of routes) {
    for (const [from, to] of [[route.userKey, route.modelKey], [route.modelKey, route.accountKey]]) {
      const source = nodeMap.get(from)!
      const target = nodeMap.get(to)!
      const key = JSON.stringify([from, to])
      let edge = edges.get(key)
      if (!edge) {
        // Both endpoints are derived from the actual node geometry, never a global array index.
        const x1 = source.x + source.width
        const y1 = source.y + source.height / 2
        const x2 = target.x
        const y2 = target.y + target.height / 2
        const bend = (x2 - x1) / 2
        edge = { key, source, target, path: `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`, count: 0, active: 0, failed: 0, routes: [] }
        edges.set(key, edge)
      }
      edge.count += route.requests.length
      edge.active += route.active
      edge.failed += route.failed
      edge.routes.push(route.key)
    }
  }
  return { nodes, edges: [...edges.values()], width: 1100, height: Math.max(240, 64 + rows * 68 + 16) }
}
