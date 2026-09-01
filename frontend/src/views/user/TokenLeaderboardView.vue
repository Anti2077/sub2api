<template>
  <AppLayout>
    <div class="mx-auto max-w-[1480px] space-y-6">
      <section class="flex flex-col gap-5 border-b border-primary-500/20 pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div class="flex min-w-0 items-start gap-4">
          <div class="flex h-12 w-12 flex-none items-center justify-center rounded-xl bg-accent-200 text-primary-700 shadow-sm dark:bg-primary-900/50 dark:text-primary-300">
            <Icon name="trophy" size="lg" />
          </div>
          <div class="min-w-0">
            <h1 class="text-2xl font-bold text-gray-950 dark:text-white sm:text-3xl">
              {{ t('leaderboard.heading') }}
            </h1>
            <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">
              {{ t('leaderboard.subtitle') }}
            </p>
          </div>
        </div>

        <div
          class="grid h-12 w-full grid-cols-4 rounded-lg bg-primary-950/5 p-1 dark:bg-white/5 sm:w-[440px]"
          role="tablist"
          :aria-label="t('leaderboard.heading')"
        >
          <button
            v-for="option in periodOptions"
            :key="option.value"
            type="button"
            role="tab"
            class="min-w-0 rounded-md px-3 text-sm font-semibold transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-900"
            :class="period === option.value
              ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
              : 'text-gray-500 hover:text-primary-700 dark:text-dark-300 dark:hover:text-primary-300'"
            :aria-selected="period === option.value"
            @click="selectPeriod(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </section>

      <section class="overflow-hidden rounded-lg border border-primary-500/15 bg-white shadow-card dark:border-dark-700 dark:bg-dark-900">
        <header class="relative border-b border-gray-200 px-5 py-6 text-center dark:border-dark-700">
          <h2 class="text-xl font-bold text-gray-950 dark:text-white">
            {{ activePeriodLabel }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
            {{ dateRangeLabel }}
          </p>
          <button
            type="button"
            class="absolute right-3 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-primary-50 hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-primary-300"
            :aria-label="t('leaderboard.refreshLabel')"
            :title="t('leaderboard.refreshLabel')"
            :disabled="loading"
            @click="loadLeaderboard"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </header>

        <div v-if="loading && ranking.length === 0" class="space-y-px bg-gray-100 dark:bg-dark-700" aria-live="polite">
          <div v-for="index in 8" :key="index" class="grid min-h-[72px] grid-cols-[56px_1fr_100px] items-center gap-3 bg-white px-5 dark:bg-dark-900">
            <div class="h-9 w-9 animate-pulse rounded-full bg-gray-200 dark:bg-dark-700"></div>
            <div class="h-4 w-36 max-w-full animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="ml-auto h-4 w-20 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
        </div>

        <div v-else-if="error" class="flex min-h-[320px] flex-col items-center justify-center px-6 text-center">
          <Icon name="exclamationCircle" size="xl" class="text-red-500" />
          <p class="mt-3 text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('leaderboard.loadFailed') }}</p>
          <button type="button" class="btn btn-secondary mt-4" @click="loadLeaderboard">
            <Icon name="refresh" size="sm" />
            {{ t('leaderboard.retry') }}
          </button>
        </div>

        <div v-else-if="ranking.length === 0" class="flex min-h-[320px] flex-col items-center justify-center px-6 text-center">
          <Icon name="trophy" size="xl" class="text-primary-300" />
          <p class="mt-3 text-sm text-gray-500 dark:text-dark-300">{{ t('leaderboard.empty') }}</p>
        </div>

        <template v-else>
          <div class="hidden overflow-x-auto md:block">
            <table class="w-full min-w-[860px] table-fixed text-sm">
              <thead class="bg-primary-950/[0.035] text-gray-500 dark:bg-white/[0.035] dark:text-dark-300">
                <tr>
                  <th class="w-[7%] px-5 py-4 text-left font-medium">{{ t('leaderboard.rank') }}</th>
                  <th class="w-[25%] px-5 py-4 text-left font-medium">{{ t('leaderboard.user') }}</th>
                  <th class="w-[12%] px-4 py-4 text-right font-medium">{{ t('leaderboard.requests') }}</th>
                  <th class="w-[12%] px-4 py-4 text-right font-medium">{{ t('leaderboard.input') }}</th>
                  <th class="w-[12%] px-4 py-4 text-right font-medium">{{ t('leaderboard.output') }}</th>
                  <th class="w-[12%] px-4 py-4 text-right font-medium">{{ t('leaderboard.cache') }}</th>
                  <th class="w-[20%] px-5 py-4 text-right font-medium">{{ t('leaderboard.totalTokens') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                <tr
                  v-for="item in ranking"
                  :key="item.rank"
                  class="h-[72px] transition-colors hover:bg-primary-50/70 dark:hover:bg-dark-800/70"
                  :class="item.is_current_user ? 'bg-accent-100/70 dark:bg-primary-950/30' : ''"
                >
                  <td class="px-5 py-3"><RankBadge :rank="item.rank" /></td>
                  <td class="px-5 py-3 font-semibold text-gray-800 dark:text-gray-100">
                    <span>{{ item.masked_email }}</span>
                    <span v-if="item.is_current_user" class="ml-2 rounded bg-primary-100 px-1.5 py-0.5 text-xs text-primary-700 dark:bg-primary-900/60 dark:text-primary-300">
                      {{ t('leaderboard.currentUser') }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-right text-gray-600 dark:text-dark-200" :title="formatExact(item.requests)">{{ formatCompact(item.requests) }}</td>
                  <td class="px-4 py-3 text-right text-gray-600 dark:text-dark-200" :title="formatExact(item.input_tokens)">{{ formatCompact(item.input_tokens) }}</td>
                  <td class="px-4 py-3 text-right text-gray-600 dark:text-dark-200" :title="formatExact(item.output_tokens)">{{ formatCompact(item.output_tokens) }}</td>
                  <td class="px-4 py-3 text-right text-gray-600 dark:text-dark-200" :title="formatExact(item.cache_tokens)">{{ formatCompact(item.cache_tokens) }}</td>
                  <td class="px-5 py-3 text-right text-base font-bold text-gray-950 dark:text-white" :title="formatExact(item.total_tokens)">{{ formatCompact(item.total_tokens) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="divide-y divide-gray-200 md:hidden dark:divide-dark-700">
            <div
              v-for="item in ranking"
              :key="item.rank"
              class="px-4 py-4"
              :class="item.is_current_user ? 'bg-accent-100/70 dark:bg-primary-950/30' : ''"
            >
              <div class="flex items-center gap-3">
                <RankBadge :rank="item.rank" />
                <div class="min-w-0 flex-1">
                  <p class="truncate font-semibold text-gray-900 dark:text-white">{{ item.masked_email }}</p>
                  <p v-if="item.is_current_user" class="mt-0.5 text-xs font-medium text-primary-600 dark:text-primary-300">{{ t('leaderboard.currentUser') }}</p>
                </div>
                <div class="text-right">
                  <p class="text-base font-bold text-gray-950 dark:text-white">{{ formatCompact(item.total_tokens) }}</p>
                  <p class="text-xs text-gray-500 dark:text-dark-300">{{ t('leaderboard.totalTokens') }}</p>
                </div>
              </div>
              <dl class="mt-4 grid grid-cols-4 gap-2 text-center">
                <div v-for="metric in mobileMetrics(item)" :key="metric.label" class="min-w-0">
                  <dt class="truncate text-[11px] text-gray-500 dark:text-dark-300">{{ metric.label }}</dt>
                  <dd class="mt-1 truncate text-xs font-semibold text-gray-700 dark:text-gray-200" :title="formatExact(metric.value)">{{ formatCompact(metric.value) }}</dd>
                </div>
              </dl>
            </div>
          </div>
        </template>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  usageAPI,
  type LeaderboardPeriod,
  type PublicTokenRankingItem
} from '@/api/usage'

const { t, locale } = useI18n()
const period = ref<LeaderboardPeriod>('day')
const ranking = ref<PublicTokenRankingItem[]>([])
const startDate = ref('')
const endDate = ref('')
const loading = ref(false)
const error = ref(false)
let requestSequence = 0

const periodOptions = computed(() => ([
  { value: 'day' as const, label: t('leaderboard.day') },
  { value: 'week' as const, label: t('leaderboard.week') },
  { value: 'month' as const, label: t('leaderboard.month') },
  { value: 'year' as const, label: t('leaderboard.year') }
]))

const activePeriodLabel = computed(() => periodOptions.value.find((item) => item.value === period.value)?.label ?? '')
const dateRangeLabel = computed(() => startDate.value && endDate.value
  ? t('leaderboard.dateRange', { start: startDate.value, end: endDate.value })
  : '')

const compactFormatter = computed(() => new Intl.NumberFormat(locale.value, {
  notation: 'compact',
  maximumFractionDigits: 2
}))
const exactFormatter = computed(() => new Intl.NumberFormat(locale.value))
const formatCompact = (value: number) => compactFormatter.value.format(value || 0)
const formatExact = (value: number) => exactFormatter.value.format(value || 0)

const RankBadge = defineComponent({
  props: { rank: { type: Number, required: true } },
  setup(props) {
    return () => h('span', {
      class: [
        'inline-flex h-9 w-9 items-center justify-center rounded-full text-sm font-bold tabular-nums',
        props.rank === 1 ? 'bg-amber-400/25 text-amber-700 dark:bg-amber-400/20 dark:text-amber-300' :
          props.rank === 2 ? 'bg-slate-200 text-slate-600 dark:bg-slate-600/40 dark:text-slate-200' :
            props.rank === 3 ? 'bg-orange-700/20 text-orange-700 dark:bg-orange-700/30 dark:text-orange-300' :
              'bg-primary-950/5 text-gray-500 dark:bg-white/10 dark:text-dark-200'
      ]
    }, String(props.rank))
  }
})

function mobileMetrics(item: PublicTokenRankingItem) {
  return [
    { label: t('leaderboard.requests'), value: item.requests },
    { label: t('leaderboard.input'), value: item.input_tokens },
    { label: t('leaderboard.output'), value: item.output_tokens },
    { label: t('leaderboard.cache'), value: item.cache_tokens }
  ]
}

async function loadLeaderboard() {
  const sequence = ++requestSequence
  loading.value = true
  error.value = false
  try {
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
    const response = await usageAPI.getPublicTokenLeaderboard(period.value, timezone)
    if (sequence !== requestSequence) return
    ranking.value = response.ranking ?? []
    startDate.value = response.start_date
    endDate.value = response.end_date
  } catch (loadError) {
    if (sequence !== requestSequence) return
    console.error('Failed to load token leaderboard:', loadError)
    error.value = true
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

function selectPeriod(nextPeriod: LeaderboardPeriod) {
  if (period.value === nextPeriod) return
  period.value = nextPeriod
  loadLeaderboard()
}

onMounted(loadLeaderboard)
</script>
