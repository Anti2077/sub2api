import { describe, expect, it } from 'vitest'
import type { OpsRoutingEvent } from '@/api/admin/ops'
import { buildRoutingGraph, groupRoutingEvents } from '../routingGraph'

const event = (i: number, overrides: Partial<OpsRoutingEvent> = {}): OpsRoutingEvent => ({
  event_id: `e-${i}`, route_key: `r-${i}`, occurred_at: '2026-09-08T00:00:00Z', event_type: 'started', status: 'active',
  user_id: i, user_label: `User ${i}`, requested_model: 'shared-model', platform: 'openai', account_id: i + 100, account_name: `Account ${i}`, attempt_count: 1, hops: [], ...overrides
})

describe('routing graph geometry and identity', () => {
  it('connects actual node boundaries in all three columns, including the tenth row', () => {
    const graph = buildRoutingGraph(groupRoutingEvents(Array.from({length:10}, (_, i) => event(i + 1))))
    for (const edge of graph.edges) {
      expect(edge.path.startsWith(`M ${edge.source.x + edge.source.width} ${edge.source.y + edge.source.height / 2} C`)).toBe(true)
      expect(edge.path.endsWith(`${edge.target.x} ${edge.target.y + edge.target.height / 2}`)).toBe(true)
    }
    expect(graph.nodes.every(node => node.y + node.height <= graph.height)).toBe(true)
    expect(graph.height).toBeGreaterThan(460)
  })

  it('centers a shorter model column within the full height of the surrounding nodes', () => {
    const graph = buildRoutingGraph(groupRoutingEvents(Array.from({length:10}, (_, i) => event(i + 1, {requested_model:`model-${i % 3}`}))))
    const center = (type: 'user' | 'model' | 'account') => {
      const nodes = graph.nodes.filter(node => node.type === type)
      return (Math.min(...nodes.map(node => node.y)) + Math.max(...nodes.map(node => node.y + node.height))) / 2
    }
    expect(center('model')).toBe(center('user'))
    expect(center('model')).toBe(center('account'))
    const models = graph.nodes.filter(node => node.type === 'model')
    expect(models[0].y).toBeGreaterThan(64)
    expect(models[1].y - models[0].y).toBe(68)
    for (const edge of graph.edges) {
      expect(edge.path.endsWith(`${edge.target.x} ${edge.target.y + edge.target.height / 2}`)).toBe(true)
    }
  })

  it('aggregates shared connections without duplicating or overstating requests', () => {
    const routes = groupRoutingEvents([event(1), event(2), event(3, {user_id:1, user_label:'User 1', account_id:101, account_name:'Account 1'})])
    const graph = buildRoutingGraph(routes)
    expect(routes).toHaveLength(2)
    expect(graph.nodes.filter(n => n.type === 'model')).toHaveLength(1)
    expect(graph.edges.filter(e => e.source.type === 'user').reduce((sum, edge) => sum + edge.count, 0)).toBe(3)
    expect(graph.edges.filter(e => e.source.type === 'model').reduce((sum, edge) => sum + edge.count, 0)).toBe(3)
  })

  it('keeps users and accounts with the same display names separate by ID', () => {
    const routes = groupRoutingEvents([event(1, {user_label:'same',account_name:'same'}), event(2, {user_label:'same',account_name:'same'})])
    const graph = buildRoutingGraph(routes)
    expect(routes).toHaveLength(2)
    expect(graph.nodes.filter(n => n.type === 'user')).toHaveLength(2)
    expect(graph.nodes.filter(n => n.type === 'account')).toHaveLength(2)
  })

  it('uses the final account as the endpoint and retains ordered failover evidence per request', () => {
    const hops = [{ account_id:7, account_name:'first', occurred_at:'2026-09-08T00:00:00Z' }, { account_id:101, account_name:'last', occurred_at:'2026-09-08T00:00:01Z' }]
    const [route] = groupRoutingEvents([event(1, { account_name:'last', hops, attempt_count:2 })])
    expect(route.account).toBe('last · #101')
    expect(route.requests[0].hops).toEqual(hops)
    expect(buildRoutingGraph([route]).nodes.filter(n => n.type === 'account')).toHaveLength(1)
  })

  it('keeps node positions stable when request activity changes route ranking', () => {
    const data = [event(1), event(2), event(3)]
    const positions = (entries: OpsRoutingEvent[]) => buildRoutingGraph(groupRoutingEvents(entries)).nodes.map(n => [n.key,n.x,n.y])
    expect(positions(data)).toEqual(positions([...data].reverse()))
    expect(positions(data)).toEqual(positions([...data,event(4,{user_id:3,user_label:'User 3',account_id:103,account_name:'Account 3'})]))
  })

  it('uses event types for outcomes even when the server status is an HTTP reason phrase', () => {
    const [route] = groupRoutingEvents([event(1,{event_type:'completed',status:'OK'}),event(2,{user_id:1,user_label:'User 1',account_id:101,account_name:'Account 1',event_type:'failed',status:'Bad Gateway'})])
    expect([route.active,route.completed,route.failed]).toEqual([0,1,1])
  })
})
