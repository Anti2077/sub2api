<template>
  <p v-if="!runs.length" class="text-xs text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p>
  <div v-else :class="compact ? 'grid grid-cols-5 gap-1' : 'flex flex-wrap gap-2'">
    <button v-for="run in runs" :key="run.id" type="button" class="min-h-11 rounded-lg border border-gray-200 py-1.5 text-xs hover:border-primary-400 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-600" :class="compact ? 'px-1 text-center' : 'px-2 text-left'" :title="`${run.request_model} → ${run.expected_model} · ${run.report?.detected_model || '—'}`" @click="emit('report', run)">
      <span class="block font-medium" :class="identityStatusClass(identityRunStatus(run))">{{ t(`admin.modelIdentity.${identityRunStatus(run)}`) }}</span>
      <time class="mt-1 block whitespace-nowrap text-gray-500 dark:text-gray-400" :datetime="run.created_at"><template v-if="compact"><span class="block">{{ new Date(run.created_at).toLocaleDateString(locale === 'zh' ? 'zh-CN' : 'en-US', { month: '2-digit', day: '2-digit' }) }}</span><span class="block">{{ new Date(run.created_at).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit', hour12: false }) }}</span></template><template v-else>{{ new Date(run.created_at).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }) }}</template></time>
    </button>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { identityRunStatus, identityStatusClass } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
defineProps<{ runs: IdentityRun[]; compact?: boolean }>()
const emit = defineEmits<{ report: [run: IdentityRun] }>()
const { t, locale } = useI18n()
</script>
