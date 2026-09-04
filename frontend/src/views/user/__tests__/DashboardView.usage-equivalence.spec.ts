import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewSource = readFileSync(resolve(testDir, '../DashboardView.vue'), 'utf8')
const statsSource = readFileSync(
  resolve(testDir, '../../../components/user/dashboard/UserDashboardStats.vue'),
  'utf8'
)

describe('DashboardView usage equivalence placement', () => {
  it('renders the plan equivalents immediately before the platform breakdown', () => {
    expect(viewSource).toContain('<template v-if="usageEquivalenceEnabled" #before-platform-breakdown>')
    expect(viewSource).toContain('<UserUsageEquivalence />')
    expect(viewSource).toContain('isFeatureFlagEnabled(FeatureFlags.usageEquivalence)')

    const slotIndex = statsSource.indexOf('<slot name="before-platform-breakdown" />')
    const platformIndex = statsSource.indexOf('<!-- Row 3: Per-platform breakdown -->')
    expect(slotIndex).toBeGreaterThan(-1)
    expect(platformIndex).toBeGreaterThan(slotIndex)
  })
})
