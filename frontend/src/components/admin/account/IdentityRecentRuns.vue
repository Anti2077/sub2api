<template>
  <div v-if="dense" class="flex shrink-0 items-center" :aria-label="t('admin.modelIdentity.history')">
    <template v-for="(run, index) in slots" :key="run?.id ?? `empty-${index}`">
      <button
        v-if="run"
        type="button"
        class="flex h-11 w-5 shrink-0 sm:w-6 items-center justify-center rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500"
        :aria-label="runDescription(run)"
        :aria-describedby="tooltipRun?.id === run.id ? tooltipID : undefined"
        @mouseenter="showTooltip(run, $event)"
        @mouseleave="hideUnlessFocused($event)"
        @focus="showTooltip(run, $event)"
        @blur="tooltipRun = null"
        @keydown.esc="tooltipRun = null"
        @click="emit('report', run)"
      >
        <span class="h-3 w-3 rounded-sm !bg-current" :class="identityStatusClass(identityRunStatus(run))" aria-hidden="true" />
        <time class="sr-only" :datetime="run.created_at">{{ formatTime(run.created_at) }}</time>
      </button>
      <span v-else class="flex h-11 w-5 shrink-0 sm:w-6 items-center justify-center" :title="t('admin.modelIdentity.noHistory')">
        <span class="h-3 w-3 rounded-sm bg-gray-200 dark:bg-dark-600" aria-hidden="true" />
        <span class="sr-only">{{ t('admin.modelIdentity.noHistory') }}</span>
      </span>
    </template>
    <Teleport to="body">
      <div v-if="tooltipRun" :id="tooltipID" role="tooltip" class="pointer-events-none fixed z-[100] w-72 max-w-[calc(100vw-24px)] rounded-lg border border-gray-200 bg-white p-3 text-xs text-gray-700 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200" :style="tooltipStyle">
        <p class="font-medium" :class="identityStatusClass(identityRunStatus(tooltipRun))">{{ t(`admin.modelIdentity.${identityRunStatus(tooltipRun)}`) }}</p>
        <p class="mt-1 text-gray-500 dark:text-gray-400">{{ formatTime(tooltipRun.created_at) }}</p>
        <p class="mt-2 break-words font-mono">{{ tooltipRun.request_model }} → {{ tooltipRun.expected_model }}</p>
        <p class="mt-1 break-words">{{ identityDetectedModel(tooltipRun) }}</p>
      </div>
    </Teleport>
  </div>
  <p v-else-if="!runs.length" class="text-xs text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p>
  <div v-else :class="compact ? 'grid grid-cols-5 gap-1' : 'flex flex-wrap gap-2'">
    <button v-for="run in runs" :key="run.id" type="button" class="min-h-11 rounded-lg border border-gray-200 py-1.5 text-xs hover:border-primary-400 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-600" :class="compact ? 'px-1 text-center' : 'px-2 text-left'" :title="`${new Date(run.created_at).toLocaleString()} · ${t(`admin.modelIdentity.${identityRunStatus(run)}`)} · ${run.request_model} → ${run.expected_model} · ${identityDetectedModel(run)}`" @click="emit('report', run)">
      <span class="contents">
      <span class="block font-medium" :class="identityStatusClass(identityRunStatus(run))">{{ t(`admin.modelIdentity.${identityRunStatus(run)}`) }}</span>
      <time class="mt-1 block whitespace-nowrap text-gray-500 dark:text-gray-400" :datetime="run.created_at"><template v-if="compact"><span class="block">{{ new Date(run.created_at).toLocaleDateString(locale === 'zh' ? 'zh-CN' : 'en-US', { month: '2-digit', day: '2-digit' }) }}</span><span class="block">{{ new Date(run.created_at).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit', hour12: false }) }}</span></template><template v-else>{{ new Date(run.created_at).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }) }}</template></time>
      </span>
    </button>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { identityDetectedModel, identityRunStatus, identityStatusClass } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
const props = defineProps<{ runs: IdentityRun[]; compact?: boolean; dense?: boolean }>()
const emit = defineEmits<{ report: [run: IdentityRun] }>()
const { t, locale } = useI18n()

const slots = computed(() => Array.from({ length: 5 }, (_, index) => props.runs[index]))
const tooltipRun = ref<IdentityRun | null>(null)
const tooltipID = `identity-recent-${useId()}`
const tooltipStyle = ref<Record<string, string>>({})
const formatTime = (value: string) => new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
const runDescription = (run: IdentityRun) => `${formatTime(run.created_at)} · ${t(`admin.modelIdentity.${identityRunStatus(run)}`)} · ${run.request_model} → ${run.expected_model} · ${identityDetectedModel(run)}`

function showTooltip(run: IdentityRun, event: Event) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const width = Math.min(288, window.innerWidth - 24)
  const above = rect.top >= 180
  tooltipStyle.value = {
    left: `${Math.max(12, Math.min(rect.left + rect.width / 2 - width / 2, window.innerWidth - width - 12))}px`,
    ...(above ? { bottom: `${window.innerHeight - rect.top + 6}px` } : { top: `${rect.bottom + 6}px` }),
  }
  tooltipRun.value = run
}
function hideUnlessFocused(event: Event) {
  if (event.currentTarget !== document.activeElement) tooltipRun.value = null
}
function hideTooltip() { tooltipRun.value = null }
onMounted(() => {
  window.addEventListener('scroll', hideTooltip, true)
  window.addEventListener('resize', hideTooltip)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', hideTooltip, true)
  window.removeEventListener('resize', hideTooltip)
})
</script>
