<template>
  <section class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700" data-testid="identity-performance" :aria-label="t('admin.modelIdentity.performance')">
    <h3 class="text-sm font-semibold">{{ t('admin.modelIdentity.performance') }}</h3>
    <dl class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
      <div v-for="metric in summary" :key="metric.key" class="rounded-md bg-gray-50 p-3 dark:bg-dark-900" :data-testid="metric.key">
        <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.modelIdentity.${metric.key}`) }}</dt>
        <dd class="mt-1 text-lg font-semibold tabular-nums">{{ metric.value }}</dd>
        <dd v-if="metric.count !== undefined" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.performanceSamples', { count: metric.count }) }}</dd>
      </div>
    </dl>
    <p v-if="!performance.ttft.count && !performance.tps.count" class="mt-3 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.performanceUnavailable') }}</p>
    <p class="mt-3 text-xs leading-relaxed text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.performanceHint') }}</p>
    <details v-if="performance.items.length" class="mt-2">
      <summary class="min-h-11 cursor-pointer py-3 text-sm">{{ t('admin.modelIdentity.performanceDetails', { count: performance.items.length }) }}</summary>
      <ol class="space-y-2">
        <li v-for="(item, index) in performance.items" :key="index" class="min-w-0 rounded-md bg-gray-50 p-3 dark:bg-dark-900">
          <div class="flex items-start justify-between gap-2 text-xs">
            <span class="min-w-0 break-words font-medium">{{ item.label || item.probeId || t('admin.modelIdentity.performanceProbe', { number: index + 1 }) }}</span>
            <span class="shrink-0 text-gray-500 dark:text-gray-400">{{ t(`admin.modelIdentity.performanceStatus_${item.status}`) }}</span>
          </div>
          <p v-if="item.label && item.probeId" class="mt-1 break-all text-xs text-gray-500 dark:text-gray-400">{{ item.probeId }}</p>
          <dl class="mt-2 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.ttft') }}</dt><dd class="mt-1 tabular-nums">{{ milliseconds(item.ttftMs) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.tps') }}</dt><dd class="mt-1 tabular-nums">{{ speed(item.tps) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.probeDuration') }}</dt><dd class="mt-1 tabular-nums">{{ milliseconds(item.durationMs) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-gray-400">{{ t('admin.modelIdentity.outputTokens') }}</dt><dd class="mt-1 tabular-nums">{{ number(item.outputTokens) }}</dd></div>
          </dl>
        </li>
      </ol>
    </details>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { identityPerformance } from '@/utils/modelIdentityPerformance'

const props = defineProps<{ report?: Record<string, unknown> }>()
const { t, locale } = useI18n()
const performance = computed(() => identityPerformance(props.report))
const number = (value: number | null) => value === null ? '—' : new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }).format(value)
const milliseconds = (value: number | null) => value === null ? '—' : `${number(value)} ms`
const speed = (value: number | null) => value === null ? '—' : `${number(value)} token/s`
const summary = computed(() => [
  { key: 'averageTtft', value: milliseconds(performance.value.ttft.value), count: performance.value.ttft.count },
  { key: 'averageTps', value: speed(performance.value.tps.value), count: performance.value.tps.count },
  { key: 'detectionDuration', value: milliseconds(performance.value.wallClockMs) },
])
</script>
