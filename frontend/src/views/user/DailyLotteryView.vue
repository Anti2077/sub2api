<template>
  <AppLayout>
    <div class="daily-lottery-view mx-auto w-full max-w-6xl space-y-6">
      <section v-if="loading" class="card flex min-h-72 items-center justify-center p-8">
        <div class="text-center text-gray-500 dark:text-dark-300">
          <Icon name="refresh" size="lg" class="mx-auto mb-3 animate-spin" />
          <p>{{ t('dailyLottery.loading') }}</p>
        </div>
      </section>

      <section v-else-if="loadError" class="card flex min-h-72 items-center justify-center p-8">
        <div class="max-w-md text-center">
          <Icon name="exclamationTriangle" size="xl" class="mx-auto mb-3 text-amber-500" />
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.loadFailed') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ loadError }}</p>
          <button type="button" class="btn btn-primary mt-5" @click="loadPage">
            <Icon name="refresh" size="sm" />
            {{ t('dailyLottery.retry') }}
          </button>
        </div>
      </section>

      <section v-else-if="status && !status.enabled" class="card flex min-h-72 items-center justify-center p-8">
        <div class="max-w-lg text-center">
          <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
            <Icon name="gift" size="xl" />
          </div>
          <h2 class="mt-5 text-xl font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.disabledTitle') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-300">{{ t('dailyLottery.disabledDescription') }}</p>
        </div>
      </section>

      <template v-else-if="status">
        <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px]">
          <section class="card overflow-hidden">
            <header class="border-b border-gray-100 px-5 py-5 dark:border-dark-700 sm:px-6">
              <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <p class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-300">{{ t('dailyLottery.today') }} · {{ status.today }}</p>
                  <h2 class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.heading') }}</h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('dailyLottery.subtitle') }}</p>
                </div>
                <span :class="statusBadgeClass" class="inline-flex w-fit items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-semibold">
                  <Icon :name="status.already_drawn ? 'checkCircle' : status.checked_in ? 'sparkles' : 'calendar'" size="sm" />
                  {{ statusLabel }}
                </span>
              </div>
            </header>

            <div class="p-5 sm:p-6">
              <div class="mb-3 flex items-center justify-between gap-4">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.prizesTitle') }}</h3>
                <span class="text-xs text-gray-500 dark:text-dark-300">{{ t('dailyLottery.serverTimezone', { timezone: status.timezone }) }}</span>
              </div>

              <div class="prize-grid" :style="{ '--prize-count': Math.min(status.prizes.length, 4) }">
                <div
                  v-for="(prize, index) in status.prizes"
                  :key="prize.id"
                  class="prize-cell"
                  :class="{
                    'prize-cell-active': drawing && activePrizeIndex === index,
                    'prize-cell-winning': displayedEntry?.prize_id === prize.id && !drawing
                  }"
                >
                  <Icon name="gift" size="md" class="text-primary-500 dark:text-primary-300" />
                  <strong class="mt-2 block truncate text-sm text-gray-900 dark:text-white">{{ prize.name }}</strong>
                  <span class="mt-1 block text-sm font-semibold text-primary-600 dark:text-primary-300">{{ formatCurrency(prize.reward_amount) }}</span>
                  <span class="mt-1 block text-xs text-gray-500 dark:text-dark-300">{{ t('dailyLottery.probability', { value: formatProbability(prize.probability) }) }}</span>
                </div>
              </div>

              <div v-if="displayedEntry?.drawn_at" class="result-panel mt-6" aria-live="polite">
                <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300">
                  <Icon name="trophy" size="lg" />
                </div>
                <div class="min-w-0">
                  <p class="text-xs font-semibold uppercase text-emerald-600 dark:text-emerald-300">{{ t('dailyLottery.resultTitle') }}</p>
                  <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.won', { prize: displayedEntry.prize_name || '-' }) }}</p>
                  <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
                    {{ displayedEntry.reward_amount > 0 ? t('dailyLottery.rewardAdded', { amount: formatCurrency(displayedEntry.reward_amount) }) : t('dailyLottery.noBalanceReward') }}
                  </p>
                  <p v-if="drawResult" class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('dailyLottery.newBalance', { amount: formatCurrency(drawResult.new_balance) }) }}</p>
                </div>
              </div>

              <div class="mt-6 flex flex-col gap-3 sm:flex-row">
                <button
                  type="button"
                  data-testid="daily-lottery-check-in"
                  class="btn btn-secondary min-h-11 flex-1"
                  :disabled="status.checked_in || checkingIn || drawing"
                  @click="handleCheckIn"
                >
                  <Icon :name="status.checked_in ? 'checkCircle' : 'calendar'" size="md" />
                  {{ checkingIn ? t('dailyLottery.checkingIn') : status.checked_in ? t('dailyLottery.checkedIn') : t('dailyLottery.checkIn') }}
                </button>
                <button
                  type="button"
                  data-testid="daily-lottery-draw"
                  class="btn btn-primary min-h-11 flex-1"
                  :disabled="!status.can_draw || checkingIn || drawing"
                  @click="handleDraw"
                >
                  <Icon name="sparkles" size="md" :class="drawing ? 'animate-pulse' : ''" />
                  {{ drawing ? t('dailyLottery.drawing') : t('dailyLottery.draw') }}
                </button>
              </div>
            </div>
          </section>

          <aside class="card h-fit p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <span class="text-sm font-medium text-gray-500 dark:text-dark-300">{{ t('dailyLottery.today') }}</span>
              <span class="h-2.5 w-2.5 rounded-full bg-emerald-500 shadow-[0_0_0_4px_rgba(16,185,129,0.12)]"></span>
            </div>
            <p class="mt-3 text-2xl font-semibold text-gray-900 dark:text-white">{{ statusLabel }}</p>
            <div class="my-5 h-px bg-gray-100 dark:bg-dark-700"></div>
            <dl class="space-y-4 text-sm">
              <div>
                <dt class="text-gray-500 dark:text-dark-300">{{ t('dailyLottery.resetAt', { time: formatResetTime(status.next_reset_at) }) }}</dt>
              </div>
              <div>
                <dt class="text-gray-500 dark:text-dark-300">{{ t('dailyLottery.serverTimezone', { timezone: status.timezone }) }}</dt>
              </div>
            </dl>
          </aside>
        </div>

        <section class="card overflow-hidden">
          <header class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.historyTitle') }}</h2>
            <button type="button" class="btn btn-ghost btn-sm" :title="t('common.refresh')" @click="loadHistory">
              <Icon name="refresh" size="sm" :class="historyLoading ? 'animate-spin' : ''" />
            </button>
          </header>
          <div v-if="historyLoading && history.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-dark-300">{{ t('common.loading') }}</div>
          <div v-else-if="history.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-dark-300">{{ t('dailyLottery.historyEmpty') }}</div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <article v-for="entry in history" :key="entry.id" class="grid gap-3 px-5 py-4 sm:grid-cols-[120px_minmax(0,1fr)_140px] sm:items-center sm:px-6">
              <time class="text-sm font-medium text-gray-900 dark:text-white">{{ entry.checkin_date }}</time>
              <div class="min-w-0">
                <p class="truncate text-sm text-gray-700 dark:text-gray-200">{{ entry.drawn_at ? entry.prize_name : t('dailyLottery.pendingDraw') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ formatDateTimeToMinute(entry.drawn_at || entry.checked_in_at) }}</p>
              </div>
              <span class="text-sm font-semibold text-primary-600 dark:text-primary-300 sm:text-right">{{ entry.drawn_at ? formatCurrency(entry.reward_amount) : '—' }}</span>
            </article>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import dailyLotteryAPI, {
  type DailyLotteryDrawResult,
  type DailyLotteryEntry,
  type DailyLotteryStatus
} from '@/api/dailyLottery'
import { useAppStore } from '@/stores'
import { formatCurrency, formatDateTimeToMinute } from '@/utils/format'

const { t, locale } = useI18n()
const appStore = useAppStore()

const status = ref<DailyLotteryStatus | null>(null)
const history = ref<DailyLotteryEntry[]>([])
const drawResult = ref<DailyLotteryDrawResult | null>(null)
const loading = ref(true)
const historyLoading = ref(false)
const checkingIn = ref(false)
const drawing = ref(false)
const loadError = ref('')
const activePrizeIndex = ref(-1)
let drawTicker: number | undefined

const displayedEntry = computed(() => drawResult.value?.entry || status.value?.entry)
const statusLabel = computed(() => {
  if (!status.value) return ''
  if (status.value.already_drawn) return t('dailyLottery.chanceUsed')
  if (status.value.can_draw) return t('dailyLottery.chanceAvailable')
  return status.value.checked_in ? t('dailyLottery.checkedIn') : t('dailyLottery.notCheckedIn')
})
const statusBadgeClass = computed(() => {
  if (status.value?.already_drawn) return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-200'
  if (status.value?.can_draw) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200'
  return 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
})

function errorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const message = String((error as { message?: unknown }).message || '')
    if (message) return message
  }
  return fallback
}

function formatProbability(value: number): string {
  return new Intl.NumberFormat(locale.value, { style: 'percent', maximumFractionDigits: 2 }).format(value)
}

function formatResetTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

async function loadHistory() {
  historyLoading.value = true
  try {
    history.value = await dailyLotteryAPI.getHistory(30)
  } catch (error) {
    appStore.showError(errorMessage(error, t('dailyLottery.loadFailed')))
  } finally {
    historyLoading.value = false
  }
}

async function loadPage() {
  loading.value = true
  loadError.value = ''
  try {
    const [nextStatus, nextHistory] = await Promise.all([
      dailyLotteryAPI.getStatus(),
      dailyLotteryAPI.getHistory(30)
    ])
    status.value = nextStatus
    history.value = nextHistory
  } catch (error) {
    loadError.value = errorMessage(error, t('dailyLottery.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleCheckIn() {
  if (!status.value || status.value.checked_in || checkingIn.value) return
  checkingIn.value = true
  try {
    status.value = await dailyLotteryAPI.checkIn()
    appStore.showSuccess(t('dailyLottery.checkInSuccess'))
    await loadHistory()
  } catch (error) {
    appStore.showError(errorMessage(error, t('dailyLottery.checkInFailed')))
  } finally {
    checkingIn.value = false
  }
}

function stopTicker() {
  if (drawTicker !== undefined) {
    window.clearInterval(drawTicker)
    drawTicker = undefined
  }
}

async function handleDraw() {
  if (!status.value?.can_draw || drawing.value) return
  drawing.value = true
  drawResult.value = null
  activePrizeIndex.value = 0
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (!reduceMotion && status.value.prizes.length > 1) {
    drawTicker = window.setInterval(() => {
      activePrizeIndex.value = (activePrizeIndex.value + 1) % status.value!.prizes.length
    }, 110)
  }

  try {
    const result = await dailyLotteryAPI.draw()
    if (!reduceMotion) {
      await new Promise((resolve) => window.setTimeout(resolve, 720))
    }
    stopTicker()
    activePrizeIndex.value = status.value.prizes.findIndex((prize) => prize.id === result.entry.prize_id)
    drawResult.value = result
    status.value = {
      ...status.value,
      can_draw: false,
      already_drawn: true,
      entry: result.entry
    }
    await loadHistory()
  } catch (error) {
    appStore.showError(errorMessage(error, t('dailyLottery.drawFailed')))
    try {
      status.value = await dailyLotteryAPI.getStatus()
    } catch {
      // Preserve the last known state; a subsequent refresh can recover it.
    }
  } finally {
    stopTicker()
    drawing.value = false
  }
}

onMounted(loadPage)
onBeforeUnmount(stopTicker)
</script>

<style scoped>
.prize-grid {
  display: grid;
  grid-template-columns: repeat(var(--prize-count), minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid rgb(229 231 235);
  border-radius: 8px;
  gap: 1px;
  background: rgb(229 231 235);
}

.prize-cell {
  min-width: 0;
  min-height: 132px;
  padding: 18px 12px;
  text-align: center;
  background: rgb(255 255 255 / 0.45);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}

.prize-cell-active {
  background: rgb(232 241 245 / 0.95);
  box-shadow: inset 0 0 0 2px rgb(67 120 159 / 0.7);
}

.prize-cell-winning {
  background: rgb(209 250 229 / 0.72);
  box-shadow: inset 0 0 0 2px rgb(16 185 129 / 0.65);
}

.result-panel {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  padding: 18px;
  border: 1px solid rgb(167 243 208);
  border-radius: 8px;
  background: rgb(236 253 245 / 0.72);
}

:global(.dark .daily-lottery-view .prize-grid) {
  border-color: rgb(96 129 143 / 0.58);
  background: rgb(96 129 143 / 0.58);
}

:global(.dark .daily-lottery-view .prize-cell) {
  background: rgb(63 100 119 / 0.72);
}

:global(.dark .daily-lottery-view .prize-cell-active) {
  background: rgb(51 104 160 / 0.5);
}

:global(.dark .daily-lottery-view .prize-cell-winning) {
  background: rgb(6 78 59 / 0.46);
}

:global(.dark .daily-lottery-view .result-panel) {
  border-color: rgb(16 185 129 / 0.42);
  background: rgb(6 78 59 / 0.34);
}

@media (max-width: 767px) {
  .prize-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

}

@media (prefers-reduced-motion: reduce) {
  .prize-cell {
    transition: none;
  }
}
</style>
