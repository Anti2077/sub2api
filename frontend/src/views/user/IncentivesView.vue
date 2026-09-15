<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('incentives.title') }}</h1>
        <p class="mt-2 text-gray-600 dark:text-gray-300">{{ t('incentives.description') }}</p>
      </header>

      <p v-if="error" role="alert" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
        {{ error }}
        <button type="button" class="btn btn-secondary ml-2" @click="load()">{{ t('incentives.refresh') }}</button>
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

            <div
              v-if="drawing"
              data-testid="incentive-draw-animation"
              class="incentive-draw-stage"
              role="status"
              aria-live="polite"
              aria-busy="true"
            >
              <div class="incentive-draw-stage__pulse" aria-hidden="true"><span /></div>
              <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">{{ t('incentives.drawing') }}</p>
                <p class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">{{ activePrize?.name || t('incentives.drawing') }}</p>
              </div>
              <div class="incentive-draw-stage__dots" aria-hidden="true">
                <span
                  v-for="prize in enabledPrizes"
                  :key="prize.id"
                  :class="{ 'is-active': activePrize?.id === prize.id }"
                />
              </div>
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
                <li
                  v-for="prize in item.prizes.filter((entry) => entry.enabled)"
                  :key="prize.id"
                  class="incentive-prize-row flex justify-between gap-3 text-sm text-gray-600 dark:text-dark-300"
                  :class="{
                    'incentive-prize-row--active': drawing && activePrize?.id === prize.id,
                    'incentive-prize-row--winning': !drawing && result?.prize.id === prize.id
                  }"
                >
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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
const result = ref<IncentiveDrawResult>()
const activePrizeIndex = ref(-1)
let requestKey: string | null = null
let prizeAnimationTimer: number | undefined

const drawAnimationDuration = 1350

const lottery = computed(() => items.value.find((item) => item.kind === 'lottery'))
const enabledPrizes = computed(() => lottery.value?.prizes.filter((prize) => prize.enabled) ?? [])
const activePrize = computed(() => enabledPrizes.value[activePrizeIndex.value] ?? null)

const date = (value: string) => new Date(value).toLocaleString()
const formatRate = (value: number) => (Number.isFinite(value) ? value.toFixed(4) : '0.0000')
const progressPercent = (item: IncentiveStatus) => {
  if (item.threshold <= 0) return 0
  const progress = item.threshold - item.next_threshold
  return Math.min(100, Math.max(0, (progress / item.threshold) * 100))
}
const isItemActive = (item: IncentiveStatus) => item.enabled || (item.kind === 'lottery' && item.check_in_enabled)

type IncentiveDrawResult = {
  prize: DailyLotteryPrize
  reward_amount: number
  source?: 'checkin' | 'consumption'
}

type LoadOptions = {
  silent?: boolean
}

function errorMessage(value: unknown): string {
  if (value instanceof Error && value.message) return value.message
  if (typeof value === 'object' && value !== null && 'message' in value) {
    const message = String((value as { message?: unknown }).message || '')
    if (message) return message
  }
  if (typeof value === 'object' && value !== null && 'response' in value) {
    const response = (value as { response?: { data?: { message?: unknown } } }).response
    const message = String(response?.data?.message || '')
    if (message) return message
  }
  return t('incentives.error')
}

async function load(options: LoadOptions = {}) {
  const { silent = false } = options
  if (!silent) {
    loading.value = true
    error.value = ''
  }

  const [statusResult, historyResult] = await Promise.allSettled([
    incentivesAPI.status(),
    incentivesAPI.history()
  ])

  if (statusResult.status === 'fulfilled') {
    items.value = statusResult.value
  }
  if (historyResult.status === 'fulfilled') {
    history.value = historyResult.value
  }

  if (!silent) {
    const failed = statusResult.status === 'rejected' ? statusResult.reason : historyResult.status === 'rejected' ? historyResult.reason : null
    if (failed) error.value = errorMessage(failed)
    loading.value = false
  }
}

async function checkIn() {
  if (checkInPending.value || !lottery.value?.check_in_enabled || lottery.value.checked_in_today) return
  checkInPending.value = true
  error.value = ''
  try {
    const checkInResult = await incentivesAPI.checkIn()
    const current = lottery.value
    if (current) {
      current.checked_in_today = true
      if (checkInResult.chance_awarded) {
        current.check_in_chance_awarded = true
        current.earned += 1
        current.available += 1
      }
    }
    await load({ silent: true })
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    checkInPending.value = false
  }
}

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function stopPrizeAnimation() {
  if (prizeAnimationTimer !== undefined) {
    window.clearTimeout(prizeAnimationTimer)
    prizeAnimationTimer = undefined
  }
}

function startPrizeAnimation() {
  stopPrizeAnimation()
  activePrizeIndex.value = enabledPrizes.value.length > 0 ? Math.floor(Math.random() * enabledPrizes.value.length) : -1
  if (prefersReducedMotion() || enabledPrizes.value.length <= 1) return

  const tick = () => {
    if (!drawing.value || enabledPrizes.value.length <= 1) return
    const current = activePrizeIndex.value
    let next = Math.floor(Math.random() * enabledPrizes.value.length)
    while (next === current) next = Math.floor(Math.random() * enabledPrizes.value.length)
    activePrizeIndex.value = next
    prizeAnimationTimer = window.setTimeout(tick, 75 + Math.floor(Math.random() * 70))
  }

  prizeAnimationTimer = window.setTimeout(tick, 90)
}

function applyDrawResult(drawResult: IncentiveDrawResult) {
  const current = lottery.value
  if (!current) return
  current.available = Math.max(0, current.available - 1)
  current.used += 1
  if (drawResult.source === 'checkin') {
    current.checked_in_today = true
    current.check_in_chance_awarded = true
  }
}

function waitForAnimation(startedAt: number) {
  if (prefersReducedMotion()) return Promise.resolve()
  const remaining = Math.max(0, drawAnimationDuration - (Date.now() - startedAt))
  return remaining > 0 ? new Promise<void>((resolve) => window.setTimeout(resolve, remaining)) : Promise.resolve()
}

async function draw() {
  if (drawing.value || !lottery.value || lottery.value.available < 1) return
  drawing.value = true
  error.value = ''
  result.value = undefined
  const startedAt = Date.now()
  startPrizeAnimation()
  requestKey ??= crypto.randomUUID()

  try {
    const drawResult = await incentivesAPI.draw(requestKey)
    await waitForAnimation(startedAt)
    stopPrizeAnimation()
    activePrizeIndex.value = enabledPrizes.value.findIndex((prize) => prize.id === drawResult.prize.id)
    result.value = drawResult
    applyDrawResult(drawResult)
    requestKey = null
    // The draw is already committed. A refresh failure must not turn a successful draw into a blocking page error.
    await load({ silent: true })
  } catch (e) {
    stopPrizeAnimation()
    error.value = errorMessage(e)
  } finally {
    drawing.value = false
  }
}

onMounted(load)
onBeforeUnmount(stopPrizeAnimation)
</script>

<style scoped>
.incentive-draw-stage {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 74px;
  padding: 14px 16px;
  border: 1px solid rgb(129 140 248 / 0.35);
  border-radius: 14px;
  background: linear-gradient(135deg, rgb(238 242 255 / 0.9), rgb(240 253 250 / 0.9));
}

.incentive-draw-stage__pulse {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: rgb(99 102 241 / 0.14);
  animation: incentive-draw-pulse 900ms ease-in-out infinite;
}

.incentive-draw-stage__pulse span {
  width: 11px;
  height: 11px;
  border-radius: 9999px;
  background: rgb(79 70 229);
  box-shadow: 0 0 0 7px rgb(99 102 241 / 0.12);
}

.incentive-draw-stage__dots {
  display: flex;
  flex-shrink: 0;
  gap: 5px;
  margin-left: auto;
}

.incentive-draw-stage__dots span {
  width: 7px;
  height: 7px;
  border-radius: 9999px;
  background: rgb(148 163 184 / 0.5);
  transition: transform 120ms ease, background-color 120ms ease;
}

.incentive-draw-stage__dots span.is-active {
  transform: scale(1.45);
  background: rgb(79 70 229);
}

.incentive-prize-row {
  padding: 8px 10px;
  margin: 0 -10px;
  border-radius: 9px;
  transition: color 160ms ease, background-color 160ms ease, transform 160ms ease;
}

.incentive-prize-row--active {
  color: rgb(67 56 202);
  background: rgb(224 231 255 / 0.78);
  transform: translateX(3px);
}

.incentive-prize-row--winning {
  color: rgb(4 120 87);
  background: rgb(209 250 229 / 0.75);
}

@keyframes incentive-draw-pulse {
  0%, 100% { transform: scale(0.92); opacity: 0.7; }
  50% { transform: scale(1.08); opacity: 1; }
}

:global(.dark .incentive-draw-stage) {
  border-color: rgb(129 140 248 / 0.38);
  background: linear-gradient(135deg, rgb(30 41 99 / 0.45), rgb(6 78 59 / 0.3));
}

:global(.dark .incentive-draw-stage__pulse) {
  background: rgb(129 140 248 / 0.2);
}

:global(.dark .incentive-draw-stage__pulse span) {
  background: rgb(165 180 252);
  box-shadow: 0 0 0 7px rgb(129 140 248 / 0.15);
}

:global(.dark .incentive-draw-stage__dots span) {
  background: rgb(148 163 184 / 0.4);
}

:global(.dark .incentive-draw-stage__dots span.is-active) {
  background: rgb(165 180 252);
}

:global(.dark .incentive-prize-row--active) {
  color: rgb(199 210 254);
  background: rgb(49 46 129 / 0.35);
}

:global(.dark .incentive-prize-row--winning) {
  color: rgb(167 243 208);
  background: rgb(6 78 59 / 0.35);
}

@media (prefers-reduced-motion: reduce) {
  .incentive-draw-stage__pulse,
  .incentive-draw-stage__dots span,
  .incentive-prize-row {
    animation: none;
    transition: none;
  }
}
</style>
