<template>
  <div class="space-y-2">
    <div class="grid gap-2 sm:grid-cols-2"><label class="text-sm">{{ t('rewards.kind') }}<select v-model="kind" class="input mt-1" :disabled="disabled" :aria-label="t('rewards.kind')"><option value="">{{ t('rewards.all') }}</option><option v-for="k in kinds" :key="k" :value="k">{{ t(`rewards.${k}`) }}</option></select></label><label class="text-sm">{{ t('rewards.searchTemplates') }}<input v-model="search" class="input mt-1" type="search" :disabled="disabled" :aria-label="t('rewards.searchTemplates')"></label></div>
    <label class="block text-sm">{{ t('rewards.template') }}<select :value="modelValue || ''" class="input mt-1" :aria-label="t('rewards.template')" :disabled="disabled" required @change="emit('update:modelValue',Number(($event.target as HTMLSelectElement).value))"><option value="" disabled>{{ t('rewards.manageChooseTemplate') }}</option><option v-for="v in options" :key="v.id" :value="v.id">#{{ v.id }} · {{ v.name }} · {{ t(`rewards.${v.kind}`) }}{{ v.published ? '' : ` · ${t('rewards.draft')}` }}</option></select></label>
    <div v-if="chosen" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-700"><p class="break-words font-semibold">#{{ chosen.id }} · {{ chosen.name }} · v{{ chosen.version }}</p><p class="mt-1 break-all">{{ chosen.kind==='group_rate' ? `${chosen.rate}× · 24h` : chosen.model }}{{ chosen.kind==='model_tokens' ? ` · ${chosen.tokens.toLocaleString()} Token · ${chosen.hours}h` : chosen.kind==='model_day' ? ' · 24h' : '' }}</p><p class="mt-1 break-words">{{ t('rewards.groups') }}: {{ chosen.group_ids.map(id=>groups.find(g=>g.id===id)?.name || `#${id}`).join(', ') }}</p><p v-if="!chosen.published" role="alert" class="mt-1 text-red-700 dark:text-red-300">{{ t('rewards.manageUnpublished') }}</p></div>
  </div>
</template>
<script setup lang="ts">
import {computed,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import type {CardTemplate} from '@/api/rewards'
const props=withDefaults(defineProps<{modelValue:number;templates:CardTemplate[];groups?:{id:number;name:string}[];disabled?:boolean}>(),{groups:()=>[],disabled:false})
const emit=defineEmits<{'update:modelValue':[value:number]}>(),{t}=useI18n()
const search=ref(''),kind=ref(''),kinds=['model_day','group_rate','model_tokens'] as const
const chosen=computed(()=>props.templates.find(v=>v.id===props.modelValue))
const options=computed(()=>props.templates.filter(v=>v.id===props.modelValue || (v.published && (!kind.value || v.kind===kind.value) && `${v.id} ${v.name} ${v.model} ${v.group_ids.map(id=>props.groups.find(g=>g.id===id)?.name||id).join(' ')}`.toLowerCase().includes(search.value.trim().toLowerCase()))))
</script>
