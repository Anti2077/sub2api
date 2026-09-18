import { describe, expect, it } from 'vitest'
import { siteOperatingDays } from '../siteOperatingDays'

describe('site operating days', () => {
  it.each(['', 'invalid', '2026-02-30', '2026-9-1'])('hides invalid date %s', (date) => {
    expect(siteOperatingDays(date, new Date('2026-09-19T00:00:00Z'))).toBeNull()
  })
  it('counts completed calendar days, including leap days', () => {
    expect(siteOperatingDays('2024-02-28', new Date('2024-03-01T00:00:00Z'))).toBe(2)
  })
  it('uses UTC consistently across client time zones and midnight', () => {
    expect(siteOperatingDays('2026-09-18', new Date('2026-09-19T07:59:59+08:00'))).toBe(0)
    expect(siteOperatingDays('2026-09-18', new Date('2026-09-19T08:00:00+08:00'))).toBe(1)
  })
  it('never renders negative days for future dates or clock skew', () => {
    expect(siteOperatingDays('2026-09-20', new Date('2026-09-19T00:00:00Z'))).toBe(0)
  })
})
