/** Completed UTC calendar days since opening; invalid/unset dates stay hidden. */
export function siteOperatingDays(startedOn: string, now: Date): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(startedOn)) return null
  const start = new Date(`${startedOn}T00:00:00Z`)
  if (!Number.isFinite(start.getTime()) || start.toISOString().slice(0, 10) !== startedOn) return null
  const today = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())
  return Math.max(0, Math.floor((today - start.getTime()) / 86_400_000))
}
