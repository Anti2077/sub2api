<template>
  <AppLayout>
    <div class="relative mx-auto max-w-3xl space-y-6">
      <RedeemCeremony ref="ceremony" @active="ceremonyActive = $event" @progress="updateCeremonyProgress" />
      <!-- Current Balance Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
          >
            <Icon name="creditCard" size="xl" class="text-white" />
          </div>
          <p class="text-sm font-medium text-primary-100">{{ t('redeem.currentBalance') }}</p>
          <p class="relative mt-2 text-4xl font-bold tabular-nums text-white">
            <span class="relative inline-block">
            <span ref="balanceTarget" :class="{ 'redeem-arrived': ceremonyActive && ceremonyProgress >= 1 }">${{ displayedBalance.toFixed(2) }}</span>
            <span v-if="ceremonyActive && ceremonyProgress > 0 && animationResult?.type === 'balance'" class="redeem-gain text-primary-100" aria-hidden="true">+${{ animationResult.value.toFixed(2) }}</span>
            </span>
          </p>
          <p class="mt-6 text-sm text-primary-100 sm:mt-2">
            {{ t('redeem.concurrency') }}: <span ref="concurrencyTarget">{{ displayedConcurrency }} {{ t('redeem.requests') }}</span>
          </p>
        </div>
      </div>

      <RedeemTicket ref="ticket" v-model="redeemCode" :locked="submitting" :invalid="invalidCode === redeemCode.trim()" @redeem="handleRedeem" />

      <!-- Success Message -->
      <transition name="fade">
        <div
          v-if="redeemResult"
          ref="resultTarget"
          role="status"
          class="card border-emerald-200 bg-emerald-50 dark:border-emerald-800/50 dark:bg-emerald-900/20"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30"
              >
                <Icon name="checkCircle" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-emerald-800 dark:text-emerald-300">
                  {{ t('redeem.redeemSuccess') }}
                </h3>
                <div class="mt-2 text-sm text-emerald-700 dark:text-emerald-400">
                  <p>{{ redeemResult.message }}</p>
                  <div class="mt-3 space-y-1">
                    <p v-if="redeemResult.type === 'balance'" class="font-medium">
                      {{ t('redeem.added') }}: ${{ redeemResult.value.toFixed(2) }}
                    </p>
                    <p v-else-if="redeemResult.type === 'concurrency'" class="font-medium">
                      {{ t('redeem.added') }}: {{ redeemResult.value }}
                      {{ t('redeem.concurrentRequests') }}
                    </p>
                    <p v-else-if="redeemResult.type === 'subscription'" class="font-medium">
                      {{ t('redeem.subscriptionAssigned') }}
                      <span v-if="redeemResult.group_name"> - {{ redeemResult.group_name }}</span>
                      <span v-if="redeemResult.validity_days">
                        ({{
                          t('redeem.subscriptionDays', { days: redeemResult.validity_days })
                        }})</span
                      >
                    </p>
                    <p v-if="redeemResult.new_balance !== undefined">
                      {{ t('redeem.newBalance') }}:
                      <span class="font-semibold">${{ redeemResult.new_balance.toFixed(2) }}</span>
                    </p>
                    <p v-if="redeemResult.new_concurrency !== undefined">
                      {{ t('redeem.newConcurrency') }}:
                      <span class="font-semibold"
                        >{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</span
                      >
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Error Message -->
      <transition name="fade">
        <div
          v-if="errorMessage"
          class="card border-red-200 bg-red-50 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-red-100 dark:bg-red-900/30"
              >
                <Icon
                  name="exclamationCircle"
                  size="md"
                  class="text-red-600 dark:text-red-400"
                />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-red-800 dark:text-red-300">
                  {{ t('redeem.redeemFailed') }}
                </h3>
                <p class="mt-2 text-sm text-red-700 dark:text-red-400">
                  {{ errorMessage }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Information Card -->
      <div
        class="card border-primary-200 bg-primary-50 dark:border-primary-800/50 dark:bg-primary-900/20"
      >
        <div class="p-6">
          <div class="flex items-start gap-4">
            <div
              class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 dark:bg-primary-900/30"
            >
              <Icon name="infoCircle" size="md" class="text-primary-600 dark:text-primary-400" />
            </div>
            <div class="flex-1">
              <h3 class="text-sm font-semibold text-primary-800 dark:text-primary-300">
                {{ t('redeem.aboutCodes') }}
              </h3>
              <ul
                class="mt-2 list-inside list-disc space-y-1 text-sm text-primary-700 dark:text-primary-400"
              >
                <li>{{ t('redeem.codeRule1') }}</li>
                <li>{{ t('redeem.codeRule2') }}</li>
                <li>
                  {{ t('redeem.codeRule3') }}
                  <span
                    v-if="contactInfo"
                    class="ml-1.5 inline-flex items-center rounded-md bg-primary-200/50 px-2 py-0.5 text-xs font-medium text-primary-800 dark:bg-primary-800/40 dark:text-primary-200"
                  >
                    {{ contactInfo }}
                  </span>
                </li>
                <li>{{ t('redeem.codeRule4') }}</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Activity -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('redeem.recentActivity') }}
          </h2>
        </div>
        <div class="p-6">
          <!-- Loading State -->
          <div v-if="loadingHistory" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>

          <!-- History List -->
          <div v-else-if="history.length > 0" class="space-y-3">
            <div
              v-for="item in history"
              :key="item.id"
              class="flex items-center justify-between rounded-xl bg-gray-50 p-4 dark:bg-dark-800"
            >
              <div class="flex items-center gap-4">
                <div
                  :class="[
                    'flex h-10 w-10 items-center justify-center rounded-xl',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'bg-emerald-100 dark:bg-emerald-900/30'
                        : 'bg-red-100 dark:bg-red-900/30'
                      : isSubscriptionType(item.type)
                        ? 'bg-purple-100 dark:bg-purple-900/30'
                        : item.value >= 0
                          ? 'bg-blue-100 dark:bg-blue-900/30'
                          : 'bg-orange-100 dark:bg-orange-900/30'
                  ]"
                >
                  <!-- 余额类型图标 -->
                  <Icon
                    v-if="isBalanceType(item.type)"
                    name="dollar"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : 'text-red-600 dark:text-red-400'
                    "
                  />
                  <!-- 订阅类型图标 -->
                  <Icon
                    v-else-if="isSubscriptionType(item.type)"
                    name="badge"
                    size="md"
                    class="text-purple-600 dark:text-purple-400"
                  />
                  <!-- 并发类型图标 -->
                  <Icon
                    v-else
                    name="bolt"
                    size="md"
                    :class="
                      item.value >= 0
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-orange-600 dark:text-orange-400'
                    "
                  />
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getHistoryItemTitle(item) }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(item.used_at) }}
                  </p>
                </div>
              </div>
              <div class="text-right">
                <p
                  :class="[
                    'text-sm font-semibold',
                    isBalanceType(item.type)
                      ? item.value >= 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : 'text-red-600 dark:text-red-400'
                      : isSubscriptionType(item.type)
                        ? 'text-purple-600 dark:text-purple-400'
                        : item.value >= 0
                          ? 'text-blue-600 dark:text-blue-400'
                          : 'text-orange-600 dark:text-orange-400'
                  ]"
                >
                  {{ formatHistoryValue(item) }}
                </p>
                <p
                  v-if="!isAdminAdjustment(item.type)"
                  class="font-mono text-xs text-gray-400 dark:text-dark-500"
                >
                  {{ item.code.slice(0, 8) }}...
                </p>
                <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('redeem.adminAdjustment') }}
                </p>
                <!-- Display notes for admin adjustments -->
                <p
                  v-if="item.notes"
                  class="mt-1 text-xs text-gray-500 dark:text-dark-400 italic max-w-[200px] truncate"
                  :title="item.notes"
                >
                  {{ item.notes }}
                </p>
              </div>
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('redeem.historyWillAppear') }}
            </p>
          </div>
          <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm">
            <span>{{ t('common.total') }}: {{ historyTotal }} {{ t('pagination.results') }}</span>
            <label>
              {{ t('pagination.perPage') }}
              <select
                v-model="historyPageSize"
                class="input w-20"
                :disabled="loadingHistory || submitting"
                @change="fetchHistory(1)"
              >
                <option v-for="size in [20, 50, 100]" :key="size" :value="size">{{ size }}</option>
              </select>
            </label>
            <button
              class="btn btn-secondary"
              :disabled="loadingHistory || submitting || historyPage <= 1"
              @click="fetchHistory(historyPage - 1)"
            >{{ t('pagination.previous') }}</button>
            <button
              class="btn btn-secondary"
              :disabled="loadingHistory || submitting || historyPage * historyPageSize >= historyTotal"
              @click="fetchHistory(historyPage + 1)"
            >{{ t('pagination.next') }}</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import RedeemCeremony from '@/components/redeem/RedeemCeremony.vue'
import RedeemTicket from '@/components/redeem/RedeemTicket.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

const redeemCode = ref('')
const submitting = ref(false)
type RedeemResult = {
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
}
const redeemResult = ref<RedeemResult | null>(null)
const errorMessage = ref('')
const invalidCode = ref<string | null>(null)
watch(redeemCode, () => { invalidCode.value = null; errorMessage.value = '' })

const ceremony = ref<InstanceType<typeof RedeemCeremony>>()
const ticket = ref<InstanceType<typeof RedeemTicket>>()
const balanceTarget = ref<HTMLElement | null>(null)
const concurrencyTarget = ref<HTMLElement | null>(null)
const resultTarget = ref<HTMLElement | null>(null)
const ceremonyActive = ref(false)
const ceremonyProgress = ref(0)
const balanceOverride = ref<number | null>(null)
const concurrencyOverride = ref<number | null>(null)
const displayedBalance = computed(() => balanceOverride.value ?? user.value?.balance ?? 0)
const displayedConcurrency = computed(() => concurrencyOverride.value ?? user.value?.concurrency ?? 0)
const animationResult = ref<typeof redeemResult.value>(null)
let animationFrom = 0
let animationTo = 0
let disposed = false
// Release the confirmed fallback when a later profile update arrives.
watch(() => user.value?.balance, () => { if (!submitting.value) balanceOverride.value = null })
watch(() => user.value?.concurrency, () => { if (!submitting.value) concurrencyOverride.value = null })
const updateCeremonyProgress = (progress: number) => {
  ceremonyProgress.value = progress
  const value = animationFrom + (animationTo - animationFrom) * (1 - (1 - progress) ** 3)
  if (animationResult.value?.type === 'balance') balanceOverride.value = value
  if (animationResult.value?.type === 'concurrency') concurrencyOverride.value = Math.round(value)
}
onBeforeUnmount(() => { disposed = true; ceremony.value?.stop(); historyRequest++ })

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
let historyRequest = 0
let loadedHistoryPageSize = 20
const contactInfo = ref('')

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async (page = 1) => {
  const request = ++historyRequest
  const pageSize = historyPageSize.value
  loadingHistory.value = true
  try {
    const result = await redeemAPI.getHistory(page, pageSize)
    if (request !== historyRequest) return
    history.value = result.items
    historyTotal.value = result.total
    historyPage.value = page
    historyPageSize.value = pageSize
    loadedHistoryPageSize = pageSize
  } catch (error) {
    if (request !== historyRequest) return
    historyPageSize.value = loadedHistoryPageSize
    appStore.showError(t('redeem.historyLoadFailed'))
    console.error('Failed to fetch history:', error)
  } finally {
    if (request === historyRequest) loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (submitting.value) return
  const code = redeemCode.value.trim()
  if (!code) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }
  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null
  ceremonyProgress.value = 0
  const beforeBalance = displayedBalance.value
  const beforeConcurrency = displayedConcurrency.value
  // The request is the only operation that can fail redemption itself.
  let result: RedeemResult
  try {
    result = await redeemAPI.redeem(code)
  } catch (error: any) {
    if (!disposed) {
      const reason = error.reason || error.response?.data?.reason
      if (['REDEEM_CODE_NOT_FOUND', 'REDEEM_CODE_USED', 'REDEEM_CODE_EXPIRED', 'REDEEM_CODE_INVALID'].includes(reason)) invalidCode.value = code
      errorMessage.value = error.response?.data?.detail || error.message || t('redeem.failedToRedeem')
      appStore.showError(t('redeem.redeemFailed'))
      submitting.value = false
      ticket.value?.rollback()
    }
    return
  }
  if (disposed) return
  balanceOverride.value = beforeBalance
  concurrencyOverride.value = beforeConcurrency
  animationResult.value = result
  // Start visual delivery promptly from authoritative response values while refreshing in parallel.
  const refresh = (async () => {
    try { await authStore.refreshUser() }
    catch (error) {
      console.error('Failed to refresh user after redeem:', error)
      if (!disposed) appStore.showWarning(t('redeem.userRefreshFailed'))
    }
    if (result.type === 'subscription') {
      try { await subscriptionStore.fetchActiveSubscriptions(true) }
      catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        if (!disposed) appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }
  })()
  try {
    // Older servers may omit the new totals: wait for profile refresh rather than inventing a balance.
    if ((result.type === 'balance' && result.new_balance === undefined) ||
        (result.type === 'concurrency' && result.new_concurrency === undefined)) await refresh
    if (disposed) return
    animationFrom = result.type === 'concurrency' ? beforeConcurrency : beforeBalance
    animationTo = result.type === 'concurrency'
      ? result.new_concurrency ?? user.value?.concurrency ?? beforeConcurrency
      : result.new_balance ?? user.value?.balance ?? beforeBalance
    // Subscription delivery targets its own result, never the monetary balance.
    if (result.type === 'subscription') redeemResult.value = result
    await nextTick()
    try {
      await Promise.all([ticket.value?.accept(), result.value > 0 ? ceremony.value?.play() : Promise.resolve()])
    } catch { /* Visual feedback cannot invalidate an already successful redemption. */ }
    if (disposed) return
    updateCeremonyProgress(1)
    redeemResult.value = result
    redeemCode.value = ''
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
    await Promise.all([refresh, fetchHistory()])
  } finally {
    if (!disposed) {
      // Preserve authoritative totals even if a profile refresh failed or returned stale data.
      balanceOverride.value = result.type === 'balance'
        ? result.new_balance ?? user.value?.balance ?? beforeBalance : null
      concurrencyOverride.value = result.type === 'concurrency'
        ? result.new_concurrency ?? user.value?.concurrency ?? beforeConcurrency : null
      submitting.value = false
    }
  }
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
  } catch (error) {
    console.error('Failed to load contact info:', error)
  }
})
</script>

<style scoped>
.redeem-arrived { display: inline-block; animation: redeem-arrival .5s ease-out both; }
.redeem-gain { position: absolute; left: 100%; top: 50%; font-size: .875rem; white-space: nowrap; animation: redeem-gain .4s ease-out both; }
@keyframes redeem-arrival { 50% { transform: scale(1.045); } }
@keyframes redeem-gain { from { opacity: 0; transform: translate(8px, calc(-50% + 6px)); } to { opacity: 1; transform: translate(8px, -50%); } }
@media (prefers-reduced-motion: reduce) { .redeem-arrived, .redeem-gain { animation: none; } }

.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
@media (max-width: 550px) {
  .redeem-gain { left: 50%; top: 100%; animation-name: redeem-gain-mobile; }
  @keyframes redeem-gain-mobile {
    from { opacity: 0; transform: translate(-50%, 6px); }
    to { opacity: 1; transform: translate(-50%, 0); }
  }
}
</style>
