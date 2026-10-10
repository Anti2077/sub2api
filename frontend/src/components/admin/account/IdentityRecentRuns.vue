<template>
  <p v-if="!runs.length" class="text-xs text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p>
  <div v-else class="flex flex-wrap gap-2">
    <button v-for="run in runs" :key="run.id" type="button" class="min-h-11 rounded-lg border border-gray-200 px-2 py-1.5 text-left text-xs hover:border-primary-400 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-dark-600" :title="`${run.request_model} → ${run.expected_model} · ${run.report?.detected_model || '—'}`" @click="emit('report', run)">
      <span class="block font-medium" :class="identityStatusClass(identityRunStatus(run))">{{ t(`admin.modelIdentity.${identityRunStatus(run)}`) }}</span>
      <time class="mt-1 block whitespace-nowrap text-gray-500 dark:text-gray-400" :datetime="run.created_at">{{ new Date(run.created_at).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false }) }}</time>
    </button>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { identityRunStatus, identityStatusClass } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
defineProps<{ runs: IdentityRun[] }>()
const emit = defineEmits<{ report: [run: IdentityRun] }>()
const { t, locale } = useI18n()
</script>
