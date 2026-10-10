<template>
  <p v-if="!runs.length" class="text-sm text-gray-500">{{ t('admin.modelIdentity.noHistory') }}</p>
  <div v-else class="overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
    <table class="w-full text-left text-xs"><thead class="text-gray-500"><tr><th class="p-3">{{ t('admin.modelIdentity.runTime') }}</th><th class="p-3">{{ t('admin.modelIdentity.requestModel') }}</th><th class="p-3">{{ t('admin.modelIdentity.expectedModel') }}</th><th class="p-3">{{ t('admin.modelIdentity.detectedModel') }}</th><th class="p-3">{{ t('admin.modelIdentity.lastResult') }}</th><th class="p-3">{{ t('common.actions') }}</th></tr></thead><tbody><tr v-for="run in runs" :key="run.id" class="border-t border-gray-100 dark:border-dark-700"><td class="whitespace-nowrap p-3">{{ new Date(run.created_at).toLocaleString() }}</td><td class="p-3">{{ run.request_model }}</td><td class="p-3">{{ run.expected_model }}</td><td class="p-3">{{ run.report?.detected_model || '—' }}</td><td class="p-3"><span class="rounded-full px-2 py-1" :class="identityStatusClass(identityRunStatus(run))">{{ t(`admin.modelIdentity.${identityRunStatus(run)}`) }}</span></td><td class="p-3"><button type="button" class="btn btn-ghost min-h-11 text-xs" @click="emit('report', run)">{{ t('admin.modelIdentity.report') }}</button><button v-if="isActive(run)" type="button" class="btn btn-ghost min-h-11 text-xs" :disabled="busy" @click="emit('cancel', run)">{{ t('common.cancel') }}</button></td></tr></tbody></table>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { identityRunStatus, identityStatusClass, isActive } from '@/utils/modelIdentity'
import type { IdentityRun } from '@/api/admin/modelIdentity'
defineProps<{ runs: IdentityRun[]; busy: boolean }>()
const emit = defineEmits<{ report: [run: IdentityRun]; cancel: [run: IdentityRun] }>()
const { t } = useI18n()
</script>
