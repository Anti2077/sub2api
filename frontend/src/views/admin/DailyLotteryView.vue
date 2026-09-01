<template>
  <AppLayout>
    <div class="daily-lottery-admin-view mx-auto w-full max-w-6xl space-y-6">
      <section class="card overflow-hidden">
        <header class="flex flex-col gap-4 border-b border-gray-100 px-5 py-5 dark:border-dark-700 sm:flex-row sm:items-start sm:justify-between sm:px-6">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.admin.configTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">{{ t('dailyLottery.admin.configSubtitle') }}</p>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadPage">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </header>

        <div v-if="loading && !config" class="flex min-h-72 items-center justify-center p-8 text-gray-500 dark:text-dark-300">
          <Icon name="refresh" size="lg" class="mr-3 animate-spin" />
          {{ t('common.loading') }}
        </div>

        <div v-else-if="loadError && !config" class="flex min-h-72 items-center justify-center p-8">
          <div class="max-w-md text-center">
            <Icon name="exclamationTriangle" size="xl" class="mx-auto mb-3 text-amber-500" />
            <p class="font-medium text-gray-900 dark:text-white">{{ t('dailyLottery.admin.loadFailed') }}</p>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ loadError }}</p>
            <button type="button" class="btn btn-primary mt-5" @click="loadPage">{{ t('dailyLottery.retry') }}</button>
          </div>
        </div>

        <form v-else-if="config" class="space-y-6 p-5 sm:p-6" @submit.prevent="saveConfig">
          <div class="flex items-start justify-between gap-5 rounded-lg border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/45">
            <div>
              <label for="daily-lottery-enabled" class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.admin.enabled') }}</label>
              <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-300">{{ t('dailyLottery.admin.enabledHint') }}</p>
            </div>
            <Toggle id="daily-lottery-enabled" v-model="config.enabled" :aria-label="t('dailyLottery.admin.enabled')" />
          </div>

          <div>
            <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.admin.prizesTitle') }}</h3>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ t('dailyLottery.admin.prizesHint') }}</p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="config.prizes.length >= 8" @click="addPrize">
                <Icon name="plus" size="sm" />
                {{ t('dailyLottery.admin.addPrize') }}
              </button>
            </div>

            <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="hidden grid-cols-[minmax(180px,1.5fr)_150px_140px_110px_132px] gap-3 bg-gray-50 px-4 py-2.5 text-xs font-semibold text-gray-500 dark:bg-dark-900/55 dark:text-dark-300 lg:grid">
                <span>{{ t('dailyLottery.admin.prizeName') }}</span>
                <span>{{ t('dailyLottery.admin.rewardAmount') }}</span>
                <span>{{ t('dailyLottery.admin.weight') }}</span>
                <span>{{ t('dailyLottery.admin.probability') }}</span>
                <span class="text-right">{{ t('common.actions') }}</span>
              </div>

              <div class="divide-y divide-gray-200 dark:divide-dark-700">
                <div v-for="(prize, index) in config.prizes" :key="prize.id || `new-${index}`" class="grid gap-3 px-4 py-4 lg:grid-cols-[minmax(180px,1.5fr)_150px_140px_110px_132px] lg:items-center">
                  <label class="block">
                    <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-dark-300 lg:hidden">{{ t('dailyLottery.admin.prizeName') }}</span>
                    <input v-model="prize.name" type="text" maxlength="40" class="input" :aria-label="`${t('dailyLottery.admin.prizeName')} ${index + 1}`" />
                  </label>

                  <label class="block">
                    <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-dark-300 lg:hidden">{{ t('dailyLottery.admin.rewardAmount') }}</span>
                    <div class="relative">
                      <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">$</span>
                      <input v-model.number="prize.reward_amount" type="number" min="0" max="1000000" step="0.00000001" class="input pl-7" :aria-label="`${t('dailyLottery.admin.rewardAmount')} ${index + 1}`" />
                    </div>
                  </label>

                  <label class="block">
                    <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-dark-300 lg:hidden">{{ t('dailyLottery.admin.weight') }}</span>
                    <input v-model.number="prize.weight" type="number" min="1" max="1000000000" step="1" class="input" :aria-label="`${t('dailyLottery.admin.weight')} ${index + 1}`" />
                  </label>

                  <div class="flex items-center justify-between gap-3 lg:block">
                    <span class="text-xs font-medium text-gray-500 dark:text-dark-300 lg:hidden">{{ t('dailyLottery.admin.probability') }}</span>
                    <div class="flex items-center gap-2">
                      <Toggle v-model="prize.enabled" :aria-label="`${t('dailyLottery.admin.prizeEnabled')} ${index + 1}`" />
                      <span class="min-w-12 text-right text-sm font-semibold text-gray-700 dark:text-gray-200">{{ formatProbability(prizeProbability(index)) }}</span>
                    </div>
                  </div>

                  <div class="flex items-center justify-end gap-1">
                    <button type="button" class="icon-action" :disabled="index === 0" :title="t('dailyLottery.admin.moveUp')" :aria-label="`${t('dailyLottery.admin.moveUp')} ${index + 1}`" @click="movePrize(index, -1)">
                      <Icon name="chevronUp" size="sm" />
                    </button>
                    <button type="button" class="icon-action" :disabled="index === config.prizes.length - 1" :title="t('dailyLottery.admin.moveDown')" :aria-label="`${t('dailyLottery.admin.moveDown')} ${index + 1}`" @click="movePrize(index, 1)">
                      <Icon name="chevronDown" size="sm" />
                    </button>
                    <button type="button" class="icon-action text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20" :disabled="config.prizes.length <= 2" :title="t('dailyLottery.admin.removePrize')" :aria-label="`${t('dailyLottery.admin.removePrize')} ${index + 1}`" @click="removePrize(index)">
                      <Icon name="trash" size="sm" />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <p class="text-xs text-gray-500 dark:text-dark-300">{{ t('dailyLottery.admin.totalWeight', { value: enabledWeight.toLocaleString() }) }}</p>
              <ul v-if="validationErrors.length" class="space-y-1 text-sm text-red-600 dark:text-red-300" role="alert">
                <li v-for="message in validationErrors" :key="message">{{ message }}</li>
              </ul>
            </div>
          </div>

          <div class="flex justify-end border-t border-gray-100 pt-5 dark:border-dark-700">
            <button type="submit" class="btn btn-primary min-w-40" :disabled="saving || validationErrors.length > 0">
              <Icon name="check" size="sm" />
              {{ saving ? t('dailyLottery.admin.saving') : t('dailyLottery.admin.save') }}
            </button>
          </div>
        </form>
      </section>

      <section class="card overflow-hidden">
        <header class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dailyLottery.admin.historyTitle') }}</h2>
          <button type="button" class="btn btn-ghost btn-sm" :title="t('common.refresh')" @click="loadHistory">
            <Icon name="refresh" size="sm" :class="historyLoading ? 'animate-spin' : ''" />
          </button>
        </header>

        <div v-if="historyLoading && history.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-dark-300">{{ t('common.loading') }}</div>
        <div v-else-if="history.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-dark-300">{{ t('dailyLottery.historyEmpty') }}</div>
        <template v-else>
          <div class="hidden overflow-x-auto md:block">
            <table class="w-full min-w-[760px] text-left text-sm">
              <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-900/55 dark:text-dark-300">
                <tr>
                  <th class="px-5 py-3 font-semibold sm:px-6">{{ t('dailyLottery.admin.user') }}</th>
                  <th class="px-5 py-3 font-semibold">{{ t('dailyLottery.admin.checkedInAt') }}</th>
                  <th class="px-5 py-3 font-semibold">{{ t('dailyLottery.admin.drawnAt') }}</th>
                  <th class="px-5 py-3 font-semibold">{{ t('dailyLottery.historyPrize') }}</th>
                  <th class="px-5 py-3 text-right font-semibold sm:px-6">{{ t('dailyLottery.historyReward') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="entry in history" :key="entry.id">
                  <td class="px-5 py-4 sm:px-6">
                    <p class="font-medium text-gray-900 dark:text-white">{{ entry.username || '-' }}</p>
                    <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-300">{{ entry.user_email }}</p>
                  </td>
                  <td class="px-5 py-4 text-gray-600 dark:text-gray-200">{{ formatDateTimeToMinute(entry.checked_in_at) }}</td>
                  <td class="px-5 py-4 text-gray-600 dark:text-gray-200">{{ entry.drawn_at ? formatDateTimeToMinute(entry.drawn_at) : t('dailyLottery.admin.notDrawn') }}</td>
                  <td class="px-5 py-4 text-gray-600 dark:text-gray-200">{{ entry.prize_name || '—' }}</td>
                  <td class="px-5 py-4 text-right font-semibold text-primary-600 dark:text-primary-300 sm:px-6">{{ entry.drawn_at ? formatCurrency(entry.reward_amount) : '—' }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="divide-y divide-gray-100 dark:divide-dark-700 md:hidden">
            <article v-for="entry in history" :key="entry.id" class="space-y-3 px-5 py-4">
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ entry.username || entry.user_email }}</p>
                  <p class="mt-0.5 truncate text-xs text-gray-500 dark:text-dark-300">{{ entry.user_email }}</p>
                </div>
                <span class="text-sm font-semibold text-primary-600 dark:text-primary-300">{{ entry.drawn_at ? formatCurrency(entry.reward_amount) : '—' }}</span>
              </div>
              <p class="text-sm text-gray-600 dark:text-gray-200">{{ entry.prize_name || t('dailyLottery.admin.notDrawn') }}</p>
              <time class="block text-xs text-gray-500 dark:text-dark-300">{{ formatDateTimeToMinute(entry.checked_in_at) }}</time>
            </article>
          </div>

          <Pagination
            v-if="historyTotal > 0"
            :page="historyPage"
            :page-size="historyPageSize"
            :total="historyTotal"
            :show-page-size-selector="false"
            @update:page="handleHistoryPage"
          />
        </template>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/api/admin'
import type { DailyLotteryAdminEntry, DailyLotteryConfig } from '@/api/admin/dailyLottery'
import type { DailyLotteryPrize } from '@/api/dailyLottery'
import { useAppStore } from '@/stores'
import { formatCurrency, formatDateTimeToMinute } from '@/utils/format'

const { t, locale } = useI18n()
const appStore = useAppStore()

const config = ref<DailyLotteryConfig | null>(null)
const history = ref<DailyLotteryAdminEntry[]>([])
const historyPage = ref(1)
const historyPageSize = 20
const historyTotal = ref(0)
const loading = ref(false)
const saving = ref(false)
const historyLoading = ref(false)
const loadError = ref('')

const enabledWeight = computed(() => config.value?.prizes.reduce((sum, prize) => sum + (prize.enabled ? Number(prize.weight) || 0 : 0), 0) || 0)
const allWeight = computed(() => config.value?.prizes.reduce((sum, prize) => sum + (Number(prize.weight) || 0), 0) || 0)
const validationErrors = computed(() => {
  if (!config.value) return []
  const errors: string[] = []
  if (config.value.prizes.length < 2 || config.value.prizes.length > 8) errors.push(t('dailyLottery.admin.validationPrizeCount'))
  if (config.value.prizes.some((prize) => !prize.name.trim() || [...prize.name.trim()].length > 40)) errors.push(t('dailyLottery.admin.validationName'))
  if (config.value.prizes.some((prize) => !Number.isFinite(Number(prize.reward_amount)) || Number(prize.reward_amount) < 0 || Number(prize.reward_amount) > 1_000_000)) errors.push(t('dailyLottery.admin.validationReward'))
  if (config.value.prizes.some((prize) => !Number.isInteger(Number(prize.weight)) || Number(prize.weight) < 1 || Number(prize.weight) > 1_000_000_000)) errors.push(t('dailyLottery.admin.validationWeight'))
  if (!config.value.prizes.some((prize) => prize.enabled)) errors.push(t('dailyLottery.admin.validationEnabled'))
  if (allWeight.value > 1_000_000_000) errors.push(t('dailyLottery.admin.validationTotalWeight'))
  return errors
})

function errorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const message = String((error as { message?: unknown }).message || '')
    if (message) return message
  }
  return fallback
}

function cloneConfig(value: DailyLotteryConfig): DailyLotteryConfig {
  return {
    enabled: value.enabled,
    prizes: value.prizes.map((prize) => ({ ...prize }))
  }
}

function prizeProbability(index: number): number {
  const prize = config.value?.prizes[index]
  if (!prize?.enabled || enabledWeight.value <= 0) return 0
  return Number(prize.weight) / enabledWeight.value
}

function formatProbability(value: number): string {
  return new Intl.NumberFormat(locale.value, { style: 'percent', maximumFractionDigits: 2 }).format(value)
}

async function loadConfig() {
  loading.value = true
  loadError.value = ''
  try {
    config.value = cloneConfig(await adminAPI.dailyLottery.getConfig())
  } catch (error) {
    loadError.value = errorMessage(error, t('dailyLottery.admin.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadHistory() {
  historyLoading.value = true
  try {
    const response = await adminAPI.dailyLottery.getHistory(historyPage.value, historyPageSize)
    history.value = response.items
    historyTotal.value = response.total
  } catch (error) {
    appStore.showError(errorMessage(error, t('dailyLottery.admin.historyLoadFailed')))
  } finally {
    historyLoading.value = false
  }
}

async function loadPage() {
  await Promise.all([loadConfig(), loadHistory()])
}

function addPrize() {
  if (!config.value || config.value.prizes.length >= 8) return
  const prize: DailyLotteryPrize = {
    id: '',
    name: '',
    reward_amount: 0,
    weight: 1,
    enabled: true
  }
  config.value.prizes.push(prize)
}

function removePrize(index: number) {
  if (!config.value || config.value.prizes.length <= 2) return
  config.value.prizes.splice(index, 1)
}

function movePrize(index: number, offset: -1 | 1) {
  if (!config.value) return
  const nextIndex = index + offset
  if (nextIndex < 0 || nextIndex >= config.value.prizes.length) return
  const [prize] = config.value.prizes.splice(index, 1)
  config.value.prizes.splice(nextIndex, 0, prize)
}

async function saveConfig() {
  if (!config.value || validationErrors.value.length > 0 || saving.value) return
  saving.value = true
  try {
    const payload: DailyLotteryConfig = {
      enabled: config.value.enabled,
      prizes: config.value.prizes.map((prize) => ({
        ...prize,
        name: prize.name.trim(),
        reward_amount: Number(prize.reward_amount),
        weight: Number(prize.weight)
      }))
    }
    config.value = cloneConfig(await adminAPI.dailyLottery.updateConfig(payload))
    appStore.showSuccess(t('dailyLottery.admin.saved'))
  } catch (error) {
    appStore.showError(errorMessage(error, t('dailyLottery.admin.saveFailed')))
  } finally {
    saving.value = false
  }
}

function handleHistoryPage(page: number) {
  historyPage.value = page
  loadHistory()
}

onMounted(loadPage)
</script>

<style scoped>
.icon-action {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: rgb(107 114 128);
  transition: color 150ms ease, background-color 150ms ease;
}

.icon-action:hover:not(:disabled) {
  color: rgb(55 65 81);
  background: rgb(243 244 246);
}

.icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.35;
}

:global(.dark .daily-lottery-admin-view .icon-action) {
  color: rgb(183 203 209);
}

:global(.dark .daily-lottery-admin-view .icon-action:hover:not(:disabled)) {
  color: white;
  background: rgb(74 112 130 / 0.75);
}
</style>
