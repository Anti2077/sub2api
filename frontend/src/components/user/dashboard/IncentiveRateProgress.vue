<template>
  <section
    v-if="status?.enabled"
    class="incentive-rate-progress card p-5"
    :aria-labelledby="headingId"
  >
    <div class="flex min-w-0 items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <span class="incentive-rate-progress__icon" aria-hidden="true">
          <Icon name="chart" size="sm" />
        </span>
        <div class="min-w-0">
          <h2 :id="headingId" class="text-base font-semibold">
            {{ t('incentives.rateProgress') }}
          </h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ date(status.starts_at) }} — {{ date(status.ends_at) }} · {{ status.timezone }}
          </p>
        </div>
      </div>
      <RouterLink
        to="/incentives#global_rate"
        class="shrink-0 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('incentives.viewDetails') }} →
      </RouterLink>
    </div>

    <p
      v-if="!status.eligible"
      role="status"
      aria-atomic="true"
      class="incentive-rate-progress__notice mt-3 rounded-lg px-3 py-2 text-xs"
    >
      {{ t('incentives.excludedNotice') }}
    </p>

    <div class="mt-5">
      <div class="flex items-end justify-between gap-3">
        <div class="flex items-baseline gap-2">
          <strong class="text-3xl tabular-nums text-gray-900 dark:text-white">
            {{ formatRate(currentRate) }}x
          </strong>
          <span class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('incentives.currentRate') }}
          </span>
        </div>
        <span class="shrink-0 text-xs tabular-nums text-gray-500 dark:text-dark-400">
          ${{ status.spend.toFixed(2) }} / ${{ status.threshold.toFixed(2) }}
        </span>
      </div>

      <div
        class="incentive-rate-progress__track"
        role="progressbar"
        :aria-valuenow="progressPercent"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="t('incentives.progress')"
      >
        <div class="incentive-rate-progress__fill" :style="{ width: `${progressPercent}%` }" />
        <span class="incentive-rate-progress__marker" :style="{ left: `${progressPercent}%` }" aria-hidden="true" />
      </div>

      <div class="mt-1.5 flex items-center justify-between gap-3 text-xs">
        <span class="text-gray-500 dark:text-dark-400">
          {{ t('incentives.rateDrop') }} {{ formatRate(rateDrop) }}x
        </span>
        <span class="font-medium text-primary-600 dark:text-primary-300">
          {{ t('incentives.next') }} ${{ status.next_threshold.toFixed(2) }}
        </span>
      </div>
    </div>

    <div v-if="status.groups.length" class="mt-5 grid gap-2 sm:grid-cols-2">
      <div
        v-for="group in status.groups"
        :key="group.group_id"
        class="rounded-lg border border-gray-200/80 bg-white/60 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/30"
      >
        <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ group.name }}</div>
        <div class="mt-1 flex items-baseline justify-between gap-2">
          <span class="text-sm tabular-nums text-gray-500 line-through dark:text-dark-500">{{ formatRate(group.base_rate) }}x</span>
          <span class="text-base font-semibold tabular-nums text-primary-700 dark:text-primary-300">{{ formatRate(group.current_rate) }}x</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { incentivesAPI, type IncentiveStatus } from '@/api/incentives'

const { t } = useI18n()
const status = ref<IncentiveStatus | null>(null)
const headingId = `incentive-rate-progress-${Math.random().toString(36).slice(2, 8)}`
let refreshTimer: number | undefined

const progressPercent = computed(() => {
  if (!status.value || status.value.threshold <= 0) return 0
  const progress = status.value.threshold - status.value.next_threshold
  return Math.min(100, Math.max(0, (progress / status.value.threshold) * 100))
})
const currentRate = computed(() => status.value?.groups[0]?.current_rate ?? 0)
const rateDrop = computed(() => {
  const group = status.value?.groups[0]
  return group ? Math.max(0, group.base_rate - group.current_rate) : 0
})

const date = (value: string) => new Date(value).toLocaleString()
const formatRate = (value: number) => Number.isFinite(value) ? value.toFixed(4) : '0.0000'

async function load() {
  try {
    const items = await incentivesAPI.status()
    status.value = items.find((item) => item.kind === 'global_rate') ?? null
  } catch {
    status.value = null
  }
}

onMounted(() => {
  load()
  refreshTimer = window.setInterval(load, 60_000)
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>

<style scoped>
.incentive-rate-progress {
  overflow: hidden;
  border: 1px solid rgb(226 232 240 / 0.9);
  background: linear-gradient(135deg, rgb(239 246 255 / 0.95), rgb(250 245 255 / 0.95));
}

.incentive-rate-progress__notice {
  border: 1px solid rgb(245 158 11 / 0.35);
  background: rgb(255 251 235 / 0.9);
  color: rgb(146 64 14);
}

.incentive-rate-progress__icon {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  background: rgb(16 185 129 / 0.12);
  color: rgb(5 150 105);
  padding: 0.35rem;
}

.incentive-rate-progress__track {
  position: relative;
  height: 0.65rem;
  overflow: visible;
  border-radius: 9999px;
  background: rgb(226 232 240 / 0.9);
}

.incentive-rate-progress__fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, rgb(59 130 246), rgb(16 185 129));
  transition: width 300ms ease;
}

.incentive-rate-progress__marker {
  position: absolute;
  top: 50%;
  height: 0.9rem;
  width: 0.9rem;
  transform: translate(-50%, -50%);
  border: 2px solid white;
  border-radius: 9999px;
  background: rgb(16 185 129);
  box-shadow: 0 1px 4px rgb(15 23 42 / 0.25);
  transition: left 300ms ease;
}

.dark .incentive-rate-progress__notice {
  border-color: rgb(245 158 11 / 0.35);
  background: rgb(120 53 15 / 0.22);
  color: rgb(253 230 138);
}

.dark .incentive-rate-progress {
  border-color: rgb(51 65 85 / 0.9);
  background: linear-gradient(135deg, rgb(23 37 84 / 0.6), rgb(58 35 83 / 0.45));
}

.dark .incentive-rate-progress__track {
  background: rgb(51 65 85 / 0.9);
}

@media (prefers-reduced-motion: reduce) {
  .incentive-rate-progress__fill,
  .incentive-rate-progress__marker {
    transition: none;
  }
}
</style>
