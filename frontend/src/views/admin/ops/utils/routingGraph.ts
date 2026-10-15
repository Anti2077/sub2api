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
  apiKeyKey: string
  modelKey: string
  accountKey: string
  user: string
  apiKey: string
  apiKeyExpanded: boolean
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
  type: 'user' | 'apiKey' | 'model' | 'account'
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

export function routingAPIKeyLabel(event: OpsRoutingEvent): string {
  const name = event.api_key_name?.trim()
  const id = event.api_key_id
  return name ? `${name}${id ? ` · #${id}` : ''}` : id ? `#${id}` : '—'
}

export function groupRoutingEvents(events: OpsRoutingEvent[], expandAllAPIKeys = false, expandedUsers: ReadonlySet<string> = new Set()): RoutingRoute[] {
  const routes = new Map<string, RoutingRoute>()
  for (const event of events) {
    const platform = event.platform || event.account_platform || '—'
    const user = event.user_id ? `${event.user_label || '—'} · #${event.user_id}` : event.user_label || '—'
    const model = event.requested_model || '—'
    const account = event.account_name ? `${event.account_name}${event.account_id ? ` · #${event.account_id}` : ''}` : event.account_id ? `#${event.account_id}` : '—'
    // Display names (including masked emails) are not entity identities.
    const userKey = JSON.stringify(['user', event.user_id ?? user])
    const apiKey = routingAPIKeyLabel(event)
    const apiKeyKey = JSON.stringify(['apiKey', userKey, event.api_key_id ?? apiKey])
    const modelKey = JSON.stringify(['model', platform, model])
    const accountKey = JSON.stringify(['account', event.account_platform || platform, event.account_id ?? account])
    const apiKeyExpanded = expandAllAPIKeys || expandedUsers.has(userKey)
    const key = JSON.stringify(apiKeyExpanded ? [userKey, apiKeyKey, modelKey, accountKey] : [userKey, modelKey, accountKey])
    let route = routes.get(key)
    if (!route) {
      route = { key, userKey, apiKeyKey, modelKey, accountKey, user, apiKey, apiKeyExpanded, model, account, platform, requests: [], active: 0, completed: 0, failed: 0 }
      routes.set(key, route)
    }
    route.requests.push(event)
    route[routingState(event)]++
  }
  for (const route of routes.values()) route.requests.sort((a, b) => b.occurred_at.localeCompare(a.occurred_at))
  return [...routes.values()].sort((a, b) => b.active - a.active || b.requests.length - a.requests.length || a.key.localeCompare(b.key))
}

export function buildRoutingGraph(routes: RoutingRoute[]) {
  const showAPIKeys = routes.some(route => route.apiKeyExpanded)
  const nodes: RoutingNode[] = []
  const nodeMap = new Map<string, RoutingNode>()
  const columns = [
    { type: 'user' as const, key: 'userKey' as const, label: 'user' as const, heading: 'user' },
    ...(showAPIKeys ? [{ type: 'apiKey' as const, key: 'apiKeyKey' as const, label: 'apiKey' as const, heading: 'apiKey' }] : []),
    { type: 'model' as const, key: 'modelKey' as const, label: 'model' as const, heading: 'requestedModel' },
    { type: 'account' as const, key: 'accountKey' as const, label: 'account' as const, heading: 'account' }
  ].map((column, index) => ({ ...column, x: 24 + index * 386 }))
  let rows = 0
  for (const column of columns) {
    const entries = new Map<string, RoutingNode>()
    for (const route of routes) {
      if (column.type === 'apiKey' && !route.apiKeyExpanded) continue
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
  // Keep each user's keys in their own vertical band, leaving collapsed users
  // a clear lane through the key column.
  let userRows = 0
  if (showAPIKeys) {
    for (const user of nodes.filter(node => node.type === 'user')) {
      const keys = new Set(routes.filter(route => route.userKey === user.key && route.apiKeyExpanded).map(route => route.apiKeyKey))
      const count = Math.max(1, keys.size)
      user.y = 64 + (userRows + (count - 1) / 2) * 68
      nodes.filter(node => keys.has(node.key)).forEach((node, index) => { node.y = 64 + (userRows + index) * 68 })
      userRows += count
    }
    rows = Math.max(rows, userRows)
  }
  // Center shorter columns as a group within the tallest node column.
  // Apply this before constructing paths so all connection endpoints follow.
  for (const column of columns) {
    const columnNodes = nodes.filter(node => node.type === column.type)
    const count = showAPIKeys && (column.type === 'user' || column.type === 'apiKey') ? userRows : columnNodes.length
    const offset = (rows - count) * 68 / 2
    for (const node of columnNodes) node.y += offset
  }
  const edges = new Map<string, RoutingEdge>()
  for (const route of routes) {
    const keys = columns.filter(column => column.type !== 'apiKey' || route.apiKeyExpanded).map(column => route[column.key])
    for (let index = 0; index < keys.length - 1; index++) {
      const from = keys[index]
      const to = keys[index + 1]
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
        const skipsKeys = showAPIKeys && source.type === 'user' && target.type === 'model'
        const start = skipsKeys ? columns[1].x + source.width + 12 : x1
        const bend = (x2 - start) / 2
        const path = `M ${x1} ${y1}${skipsKeys ? ` L ${start} ${y1}` : ''} C ${start + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`
        edge = { key, source, target, path, count: 0, active: 0, failed: 0, routes: [] }
        edges.set(key, edge)
      }
      edge.count += route.requests.length
      edge.active += route.active
      edge.failed += route.failed
      edge.routes.push(route.key)
    }
  }
  return { columns, nodes, edges: [...edges.values()], width: 1100 + (showAPIKeys ? 386 : 0), height: Math.max(240, 64 + rows * 68 + 16) }
}
