<template>
  <section class="card overflow-hidden" :aria-label="t('admin.dashboard.todaySpend.title')">
    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.todaySpend.title') }}</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.todaySpend.window') }}</p>
      </div>
      <span v-if="stale" class="rounded-full bg-amber-50 px-3 py-1 text-xs text-amber-800 dark:bg-amber-950/40 dark:text-amber-200">{{ t('admin.dashboard.todaySpend.updating') }}</span>
      <span v-else class="text-xs font-medium text-gray-500 dark:text-gray-400">USD</span>
    </div>
    <dl class="grid gap-5 p-5 sm:grid-cols-2 sm:gap-8">
      <div data-testid="today-actual-spend">
        <dt class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.dashboard.todaySpend.actual') }}</dt>
        <dd class="mt-2 break-all text-3xl font-semibold tabular-nums tracking-tight text-emerald-700 dark:text-emerald-300">{{ formatAmount(actualCost) }}</dd>
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.todaySpend.actualHelp') }}</p>
      </div>
      <div data-testid="today-account-spend" class="border-t border-gray-100 pt-5 sm:border-l sm:border-t-0 sm:pl-8 sm:pt-0 dark:border-dark-700">
        <dt class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.dashboard.todaySpend.cost') }}</dt>
        <dd class="mt-2 break-all text-3xl font-semibold tabular-nums tracking-tight text-gray-900 dark:text-white">{{ formatAmount(accountCost) }}</dd>
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.todaySpend.costHelp') }}</p>
      </div>
    </dl>
  </section>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
defineProps<{ actualCost?: number | null; accountCost?: number | null; stale?: boolean }>()
const currency = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 4 })
function formatAmount(value: number | null | undefined) {
  return typeof value === 'number' && Number.isFinite(value) ? currency.format(value) : '—'
}
</script>
