<template>
  <section class="card overflow-hidden" aria-labelledby="usage-equivalence-title">
    <div class="border-b border-gray-100 px-4 py-4 dark:border-dark-700 sm:px-6">
      <div class="flex flex-col gap-4 2xl:flex-row 2xl:items-center 2xl:justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <div class="flex h-9 w-9 flex-none items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30">
              <Icon name="calculator" size="md" class="text-emerald-700 dark:text-emerald-300" />
            </div>
            <div>
              <h2 id="usage-equivalence-title" class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('dashboard.usageEquivalence.title') }}
              </h2>
              <p class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">
                {{ t('dashboard.usageEquivalence.subtitle') }}
              </p>
            </div>
          </div>
        </div>

        <div
          class="grid grid-cols-2 gap-2 rounded-xl bg-gray-100 p-1 sm:grid-cols-3 xl:grid-cols-6 dark:bg-dark-800"
          role="group"
          :aria-label="t('dashboard.usageEquivalence.periodLabel')"
        >
          <button
            v-for="option in periodOptions"
            :key="option.value"
            :data-testid="`usage-equivalence-period-${option.value}`"
            type="button"
            class="inline-flex min-h-11 items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-900"
            :class="period === option.value
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
            :aria-pressed="period === option.value"
            @click="selectPeriod(option.value)"
          >
            <Icon v-if="loading && period === option.value" name="refresh" size="xs" class="animate-spin" />
            {{ option.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="p-4 sm:p-6" aria-live="polite" :aria-busy="loading">
      <div v-if="loading && !data" data-testid="usage-equivalence-loading" class="space-y-5">
        <div class="grid gap-3 sm:grid-cols-3">
          <Skeleton v-for="index in 3" :key="index" height="148px" />
        </div>
        <Skeleton height="88px" />
      </div>

      <div
        v-else-if="error"
        data-testid="usage-equivalence-error"
        class="flex min-h-44 flex-col items-center justify-center rounded-xl border border-dashed border-red-200 bg-red-50/60 px-5 py-8 text-center dark:border-red-900/60 dark:bg-red-950/20"
      >
        <Icon name="exclamationCircle" size="xl" class="text-red-500" />
        <p class="mt-3 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('dashboard.usageEquivalence.loadFailed') }}
        </p>
        <button
          type="button"
          class="mt-4 min-h-11 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
          @click="load"
        >
          {{ t('dashboard.usageEquivalence.retry') }}
        </button>
      </div>

      <template v-else-if="data">
        <div class="overflow-hidden rounded-lg border border-gray-200 md:grid md:grid-cols-3 md:divide-x md:divide-y-0 dark:border-dark-700 dark:divide-dark-700">
          <article
            v-for="plan in data.plans"
            :key="plan.id"
            :data-testid="`usage-equivalence-plan-${plan.id}`"
            class="min-w-0 border-b border-gray-200 p-5 last:border-b-0 md:border-b-0 dark:border-dark-700"
          >
            <div class="flex items-start justify-between gap-3">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ plan.name }}</p>
              <span
                class="whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold"
                :class="planBadgeStyle(plan.id)"
              >
                {{ plan.usage_multiple }}× Plus
              </span>
            </div>
            <dl class="mt-5 grid grid-cols-2 gap-x-4 gap-y-5 md:grid-cols-1 xl:grid-cols-2">
              <div>
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('dashboard.usageEquivalence.sevenDayWindow') }}
                </dt>
                <dd class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">
                  {{ formatEquivalent(plan.equivalent_7d_windows) }}
                  <span class="block text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('dashboard.usageEquivalence.quotaWindows') }}
                  </span>
                </dd>
                <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">
                  {{ t('dashboard.usageEquivalence.configuredReference', { value: formatCurrency(plan.quota_7d_standard_cost) }) }}
                </p>
              </div>
              <div>
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('dashboard.usageEquivalence.thirtyDayWindow') }}
                </dt>
                <dd class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">
                  {{ formatEquivalent(plan.equivalent_30d_windows) }}
                  <span class="block text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('dashboard.usageEquivalence.quotaWindows') }}
                  </span>
                </dd>
                <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">
                  {{ t('dashboard.usageEquivalence.configuredReference', { value: formatCurrency(plan.quota_30d_standard_cost) }) }}
                </p>
              </div>
            </dl>
          </article>
        </div>

        <div class="mt-4 grid gap-4 rounded-2xl border border-gray-100 bg-gray-50/80 p-4 sm:grid-cols-[minmax(0,1.5fr)_repeat(3,minmax(0,1fr))] sm:p-5 dark:border-dark-700 dark:bg-dark-800/40">
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
              {{ t('dashboard.usageEquivalence.standardValue') }}
            </p>
            <p class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">
              {{ formatCurrency(data.standard_cost) }}
            </p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.usageEquivalence.actualCharged') }}</p>
            <p class="mt-1 font-mono text-sm font-semibold text-emerald-700 dark:text-emerald-300">
              {{ formatCurrency(data.actual_cost) }}
            </p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.usageEquivalence.effectiveMultiplier') }}</p>
            <p class="mt-1 font-mono text-sm font-semibold text-gray-900 dark:text-white">
              {{ effectiveMultiplier }}
            </p>
          </div>
          <div>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.usageEquivalence.activity') }}</p>
            <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('dashboard.usageEquivalence.activityValue', {
                requests: formatInteger(data.total_requests),
                tokens: formatCompact(data.total_tokens)
              }) }}
            </p>
          </div>
        </div>

        <div
          v-if="data.standard_cost === 0"
          data-testid="usage-equivalence-zero"
          class="mt-4 rounded-xl border border-dashed border-gray-200 px-4 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400"
        >
          {{ t('dashboard.usageEquivalence.noUsage') }}
        </div>

        <div class="mt-4 flex flex-col gap-2 text-xs leading-5 text-gray-500 dark:text-dark-400 sm:flex-row sm:items-start sm:justify-between">
          <p class="flex max-w-4xl items-start gap-2">
            <Icon name="infoCircle" size="sm" class="mt-0.5 flex-none" />
            <span>{{ t('dashboard.usageEquivalence.disclaimer') }}</span>
          </p>
          <a
            :href="data.reference_source"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex min-h-11 flex-none items-center gap-1.5 rounded-lg px-2 font-medium text-primary-600 hover:text-primary-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
          >
            {{ t('dashboard.usageEquivalence.officialSource') }}
            <Icon name="externalLink" size="xs" />
          </a>
        </div>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usageAPI } from '@/api/usage'
import type {
  UsageEquivalencePeriod,
  UsageEquivalenceResponse
} from '@/api/usage'
import Icon from '@/components/icons/Icon.vue'
import Skeleton from '@/components/common/Skeleton.vue'

const { locale, t } = useI18n()
const period = ref<UsageEquivalencePeriod>('this_month')
const data = ref<UsageEquivalenceResponse | null>(null)
const loading = ref(false)
const error = ref(false)
let requestSequence = 0
let controller: AbortController | null = null

const periodOptions = computed(() => [
  { value: 'last_24h' as const, label: t('dashboard.usageEquivalence.periods.last24Hours') },
  { value: 'last_7d' as const, label: t('dashboard.usageEquivalence.periods.last7Days') },
  { value: 'this_month' as const, label: t('dashboard.usageEquivalence.periods.thisMonth') },
  { value: 'last_30d' as const, label: t('dashboard.usageEquivalence.periods.last30Days') },
  { value: 'last_6m' as const, label: t('dashboard.usageEquivalence.periods.last6Months') },
  { value: 'all_time' as const, label: t('dashboard.usageEquivalence.periods.allTime') }
])

const effectiveMultiplier = computed(() => {
  if (!data.value || data.value.standard_cost <= 0) return '—'
  return `${data.value.effective_rate_multiplier.toFixed(2)}×`
})

const formatterLocale = computed(() => locale.value === 'zh' ? 'zh-CN' : 'en-US')

const formatCurrency = (value: number) => new Intl.NumberFormat(formatterLocale.value, {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2
}).format(value)

const formatInteger = (value: number) => new Intl.NumberFormat(formatterLocale.value, {
  maximumFractionDigits: 0
}).format(value)

const formatCompact = (value: number) => new Intl.NumberFormat(formatterLocale.value, {
  notation: 'compact',
  maximumFractionDigits: 1
}).format(value)

const formatEquivalent = (value: number) => {
  return new Intl.NumberFormat(formatterLocale.value, {
    maximumSignificantDigits: 3
  }).format(value)
}

const planBadgeStyle = (id: UsageEquivalenceResponse['plans'][number]['id']) => ({
  chatgpt_plus: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200',
  chatgpt_pro_5x: 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-200',
  chatgpt_pro_20x: 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-200'
}[id])

const load = async () => {
  const sequence = ++requestSequence
  controller?.abort()
  const requestController = new AbortController()
  controller = requestController
  loading.value = true
  error.value = false

  try {
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
    const response = await usageAPI.getUsageEquivalence(period.value, timezone, {
      signal: requestController.signal
    })
    if (sequence !== requestSequence) return
    data.value = response
  } catch (caught) {
    if (sequence !== requestSequence || requestController.signal.aborted) return
    console.error('Failed to load usage equivalence:', caught)
    error.value = true
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

const selectPeriod = (nextPeriod: UsageEquivalencePeriod) => {
  if (period.value === nextPeriod) return
  period.value = nextPeriod
  void load()
}

onMounted(() => {
  void load()
})

onBeforeUnmount(() => {
  requestSequence += 1
  controller?.abort()
})
</script>
