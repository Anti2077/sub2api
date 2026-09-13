<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('incentives.title') }}</h1>
        <p class="mt-2 text-gray-600 dark:text-gray-300">{{ t('incentives.description') }}</p>
      </header>

      <p v-if="error" role="alert" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
        {{ error }}
        <button type="button" class="btn btn-secondary ml-2" @click="load">{{ t('incentives.refresh') }}</button>
      </p>
      <p v-if="loading" class="text-gray-500 dark:text-dark-300">{{ t('incentives.loading') }}</p>

      <div v-if="items.length" class="grid gap-6 lg:grid-cols-2">
        <section
          v-for="item in items"
          :id="item.kind"
          :key="item.kind"
          class="card space-y-5 p-5 sm:p-6"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t(`incentives.${item.kind}`) }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
                {{ date(item.starts_at) }} — {{ date(item.ends_at) }} · {{ item.timezone }}
              </p>
            </div>
            <span
              class="rounded-full px-3 py-1 text-xs font-semibold"
              :class="isItemActive(item) ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'"
            >
              {{ isItemActive(item) ? t('incentives.enabled') : t('incentives.disabled') }}
            </span>
          </div>

          <template v-if="item.kind === 'global_rate'">
            <div v-if="item.enabled" class="space-y-4">
              <div class="flex items-end justify-between gap-3">
                <div>
                  <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ t('incentives.progress') }}</p>
                  <p class="mt-1 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">${{ item.spend.toFixed(2) }}</p>
                </div>
                <p class="text-right text-sm text-gray-500 dark:text-dark-300">
                  {{ t('incentives.next') }}<br />
                  <strong class="text-primary-600 dark:text-primary-300">${{ item.next_threshold.toFixed(2) }}</strong>
                </p>
              </div>

              <div
                class="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700"
                role="progressbar"
                :aria-valuenow="progressPercent(item)"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-label="t('incentives.progress')"
              >
                <div class="h-full rounded-full bg-gradient-to-r from-primary-500 to-emerald-500 transition-[width] duration-300" :style="{ width: `${progressPercent(item)}%` }" />
              </div>

              <p v-if="!item.eligible" role="status" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200">
                {{ t('incentives.excludedNotice') }}
              </p>

              <div v-if="item.groups.length" class="grid gap-2 sm:grid-cols-2">
                <div v-for="group in item.groups" :key="group.group_id" class="rounded-lg border border-gray-200/80 bg-white/60 px-3 py-3 dark:border-dark-700 dark:bg-dark-900/30">
                  <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ group.name }}</p>
                  <div class="mt-1 flex items-baseline justify-between gap-2">
                    <span class="text-sm tabular-nums text-gray-500 line-through dark:text-dark-500">{{ formatRate(group.base_rate) }}x</span>
                    <strong class="text-base tabular-nums text-primary-700 dark:text-primary-300">{{ formatRate(group.current_rate) }}x</strong>
                  </div>
                </div>
              </div>
              <p class="text-sm text-gray-500 dark:text-dark-300">{{ t('incentives.rateNote') }}</p>
            </div>
            <p v-else class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-300">
              {{ t('incentives.disabled') }}
            </p>
          </template>

          <template v-else>
            <div v-if="item.enabled" class="space-y-4">
              <div class="flex items-end justify-between gap-3">
                <div>
                  <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ t('incentives.progress') }}</p>
                  <p class="mt-1 text-3xl font-semibold tabular-nums text-gray-900 dark:text-white">${{ item.personal_spend.toFixed(2) }}</p>
                </div>
                <p class="text-right text-sm text-gray-500 dark:text-dark-300">
                  {{ t('incentives.next') }}<br />
                  <strong class="text-primary-600 dark:text-primary-300">${{ item.next_threshold.toFixed(2) }}</strong>
                </p>
              </div>
              <div
                class="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700"
                role="progressbar"
                :aria-valuenow="progressPercent(item)"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-label="t('incentives.progress')"
              >
                <div class="h-full rounded-full bg-gradient-to-r from-primary-500 to-violet-500 transition-[width] duration-300" :style="{ width: `${progressPercent(item)}%` }" />
              </div>
            </div>
            <p v-else class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-300">
              {{ t('incentives.consumptionDisabled') }}
            </p>

            <div class="rounded-xl border border-primary-200 bg-primary-50/70 p-4 dark:border-primary-800/60 dark:bg-primary-900/20">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('incentives.checkInTitle') }}</h3>
                  <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ t('incentives.checkInDescription') }}</p>
                </div>
                <span v-if="item.checked_in_today" class="rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                  {{ t('incentives.checkedInToday') }}
                </span>
              </div>
              <button
                type="button"
                data-testid="incentive-check-in"
                class="btn btn-secondary mt-4 min-h-11 w-full sm:w-auto"
                :disabled="checkInPending || item.checked_in_today || !item.check_in_enabled"
                @click="checkIn"
              >
                {{ checkInPending ? t('incentives.checkingIn') : item.checked_in_today ? t('incentives.checkedInToday') : item.check_in_enabled ? t('incentives.checkIn') : t('incentives.checkInDisabled') }}
              </button>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3 text-sm text-gray-600 dark:text-dark-300">
              <p>
                {{ t('incentives.available') }}: <strong class="text-gray-900 dark:text-white">{{ item.available }}</strong>
                · {{ t('incentives.earned') }}: {{ item.earned }} · {{ t('incentives.used') }}: {{ item.used }}
              </p>
              <p>{{ t('incentives.expiry') }}</p>
            </div>

            <button
              type="button"
              data-testid="incentive-draw"
              class="btn btn-primary min-h-11 w-full"
              :disabled="drawing || item.available < 1 || (!item.enabled && !item.check_in_enabled)"
              @click="draw"
            >
              {{ t(drawing ? 'incentives.drawing' : 'incentives.draw') }}
            </button>
            <p v-if="result" role="status" class="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              {{ result.prize.name }} · ${{ result.reward_amount.toFixed(2) }}
            </p>

            <div v-if="item.prizes.filter((prize) => prize.enabled).length" class="space-y-2">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('incentives.prizes') }}</p>
              <ul class="space-y-2">
                <li v-for="prize in item.prizes.filter((entry) => entry.enabled)" :key="prize.id" class="flex justify-between gap-3 text-sm text-gray-600 dark:text-dark-300">
                  <span>{{ prize.name }}</span>
                  <span>${{ prize.reward_amount.toFixed(2) }} · {{ (prize.probability * 100).toFixed(2) }}%</span>
                </li>
              </ul>
            </div>
          </template>
        </section>
      </div>

      <section v-if="history" class="card p-5 sm:p-6">
        <h2 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('incentives.history') }}</h2>
        <p v-if="!history.rewards.length" class="text-sm text-gray-500 dark:text-dark-300">{{ t('incentives.empty') }}</p>
        <ul v-else class="space-y-3">
          <li v-for="reward in history.rewards" :key="reward.id" class="flex flex-wrap justify-between gap-2 border-b border-gray-100 pb-3 text-sm last:border-b-0 last:pb-0 dark:border-dark-700">
            <time class="text-gray-500 dark:text-dark-300">{{ date(reward.created_at) }}</time>
            <span class="font-medium text-gray-900 dark:text-white">{{ reward.prize.name }} · ${{ reward.reward_amount.toFixed(2) }}</span>
          </li>
        </ul>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { incentivesAPI, type IncentiveHistory, type IncentiveStatus } from '@/api/incentives'
import type { DailyLotteryPrize } from '@/api/dailyLottery'

const { t } = useI18n()
const items = ref<IncentiveStatus[]>([])
const history = ref<IncentiveHistory>()
const loading = ref(false)
const drawing = ref(false)
const checkInPending = ref(false)
const error = ref('')
const result = ref<{ prize: DailyLotteryPrize; reward_amount: number }>()
let requestKey: string | null = null

const lottery = computed(() => items.value.find((item) => item.kind === 'lottery'))

const date = (value: string) => new Date(value).toLocaleString()
const formatRate = (value: number) => (Number.isFinite(value) ? value.toFixed(4) : '0.0000')
const progressPercent = (item: IncentiveStatus) => {
  if (item.threshold <= 0) return 0
  const progress = item.threshold - item.next_threshold
  return Math.min(100, Math.max(0, (progress / item.threshold) * 100))
}
const isItemActive = (item: IncentiveStatus) => item.enabled || (item.kind === 'lottery' && item.check_in_enabled)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [status, activityHistory] = await Promise.all([incentivesAPI.status(), incentivesAPI.history()])
    items.value = status
    history.value = activityHistory
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('incentives.error')
  } finally {
    loading.value = false
  }
}

async function checkIn() {
  if (checkInPending.value || !lottery.value?.check_in_enabled || lottery.value.checked_in_today) return
  checkInPending.value = true
  error.value = ''
  try {
    await incentivesAPI.checkIn()
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('incentives.error')
  } finally {
    checkInPending.value = false
  }
}

async function draw() {
  if (drawing.value || !lottery.value || lottery.value.available < 1) return
  drawing.value = true
  error.value = ''
  requestKey ??= crypto.randomUUID()
  try {
    result.value = await incentivesAPI.draw(requestKey)
    requestKey = null
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('incentives.error')
  } finally {
    drawing.value = false
  }
}

onMounted(load)
</script>
