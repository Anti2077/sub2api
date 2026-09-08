import { describe, expect, it } from 'vitest'
import type { OpsRoutingEvent } from '@/api/admin/ops'
import { reduceRoutingEvent, reduceRoutingSnapshot } from '../routingMonitor'

function event(overrides: Partial<OpsRoutingEvent>): OpsRoutingEvent {
  return {
    event_id: 'event',
    event_type: 'started',
    occurred_at: '2026-09-07T00:00:00.000Z',
    route_key: 'request-1',
    status: 'active',
    attempt_count: 1,
    hops: [],
    ...overrides
  }
}

describe('routing monitor reducer', () => {
  it('keeps the newest event when snapshot active and recent overlap', () => {
    const state = reduceRoutingSnapshot({
      active: [event({ event_id: 'started', occurred_at: '2026-09-07T00:00:01.000Z' })],
      recent: [event({ event_id: 'completed', event_type: 'completed', status: 'completed', occurred_at: '2026-09-07T00:00:02.000Z' })]
    })

    expect(state.get('request-1')?.event_type).toBe('completed')
  })

  it('does not reopen a completion when a delayed start has the same millisecond timestamp', () => {
    const completed = event({event_type:'completed',status:'OK'})
    const current = reduceRoutingEvent(new Map(), completed)
    expect(reduceRoutingEvent(current, event({event_type:'started',status:'active'}))).toBe(current)
  })

  it('ignores an out-of-order event without replacing the current state', () => {
    const current = reduceRoutingSnapshot({ recent: [event({ event_id: 'completed', event_type: 'completed', status: 'completed', occurred_at: '2026-09-07T00:00:03.000Z' })] })
    const next = reduceRoutingEvent(current, event({ event_id: 'started', occurred_at: '2026-09-07T00:00:02.000Z' }))

    expect(next).toBe(current)
    expect(next.get('request-1')?.event_type).toBe('completed')
  })

  it('preserves failover hops on the latest event for a logical request', () => {
    const current = reduceRoutingEvent(new Map(), event({
      event_id: 'switched',
      event_type: 'switched',
      occurred_at: '2026-09-07T00:00:01.000Z',
      attempt_count: 2,
      hops: [
        { account_id: 1, occurred_at: '2026-09-07T00:00:00.000Z' },
        { account_id: 2, occurred_at: '2026-09-07T00:00:01.000Z' }
      ]
    }))
    const next = reduceRoutingEvent(current, event({
      event_id: 'completed',
      event_type: 'completed',
      status: 'completed',
      occurred_at: '2026-09-07T00:00:02.000Z',
      attempt_count: 2,
      hops: current.get('request-1')?.hops
    }))

    expect(next.get('request-1')?.attempt_count).toBe(2)
    expect(next.get('request-1')?.hops).toHaveLength(2)
  })
})

describe('routing event expiry', () => {
  it('expires completed requests without removing long-running active requests', async () => {
    const { pruneRoutingEvents } = await import('../routingMonitor')
    const current = reduceRoutingSnapshot({
      active: [event({route_key:'active',occurred_at:'2026-09-07T00:00:00Z'})],
      recent: [event({route_key:'old',event_type:'completed',status:'OK',occurred_at:'2026-09-07T00:03:00Z'}),event({route_key:'new',event_type:'failed',status:'Bad Gateway',occurred_at:'2026-09-07T00:03:50Z'})]
    })
    expect([...pruneRoutingEvents(current, Date.parse('2026-09-07T00:04:00Z')).keys()]).toEqual(['active','new'])
  })
})
