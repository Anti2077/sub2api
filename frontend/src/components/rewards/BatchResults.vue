<template>
  <div v-if="results.length || refreshError" class="card space-y-3 p-4 text-sm">
    <p role="status">{{ t('rewards.bulkTemplateResult',{success:results.filter(v=>!v.error).length,failed:results.filter(v=>v.error).length}) }}</p>
    <p v-if="refreshError" role="alert" class="break-words text-red-700 dark:text-red-300">{{ t('rewards.manageRefreshFailed') }} {{ refreshError }}</p>
    <details v-if="results.length" open><summary class="cursor-pointer">{{ t('rewards.bulkTemplateDetails') }}</summary><ul class="mt-3 space-y-2"><li v-for="item in results" :key="item.id" class="break-words" :class="item.error ? 'text-red-700 dark:text-red-300' : 'text-emerald-800 dark:text-emerald-200'">{{ item.name }} — {{ item.error || t('rewards.bulkTemplateSaved') }}</li></ul></details>
  </div>
</template>
<script setup lang="ts">
import {useI18n} from 'vue-i18n'
defineProps<{results:{id:number|string;name:string;error?:string}[];refreshError?:string}>()
const {t}=useI18n()
</script>
