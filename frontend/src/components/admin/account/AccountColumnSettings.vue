<template>
  <div>
    <div class="flex items-center justify-between gap-2 px-2 py-2">
      <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ t('admin.accounts.viewColumns') }}</span>
      <button type="button" class="text-xs text-primary-600 dark:text-primary-400" @click="emit('reset')">{{ t('admin.accounts.resetColumns') }}</button>
    </div>
    <p class="px-2 pb-2 text-xs text-gray-500 dark:text-gray-400">{{ t('common.dragToReorder') }}</p>
    <VueDraggable v-model="draft" handle=".account-column-handle" :animation="150" ghost-class="opacity-40" class="space-y-1">
      <div v-for="(column, index) in ordered" :key="column.key" class="flex items-center gap-2 rounded-md px-2 py-1 hover:bg-gray-50 dark:hover:bg-dark-700">
        <button type="button" class="account-column-handle cursor-grab touch-none rounded p-2 text-gray-400 active:cursor-grabbing" :aria-label="`${t('common.dragToReorder')}: ${column.label}`" @keydown.up.prevent="move(index, -1)" @keydown.down.prevent="move(index, 1)">
          <Icon name="menu" size="sm" />
        </button>
        <label class="flex min-w-0 flex-1 cursor-pointer items-center justify-between gap-3 py-2 text-sm text-gray-700 dark:text-gray-200">
          <span class="truncate">{{ column.label }}</span>
          <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" :checked="!hidden.has(column.key)" @change="emit('toggle', column.key)" />
        </label>
      </div>
    </VueDraggable>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
const props = defineProps<{ columns: Column[]; hidden: Set<string> }>()
const emit = defineEmits<{ order: [keys: string[]]; toggle: [key: string]; reset: [] }>()
const { t } = useI18n()
const draft = ref<string[]>([])
watch(() => props.columns.map(c => c.key), keys => { draft.value = keys }, { immediate: true })
watch(draft, keys => {
  if (keys.join('|') !== props.columns.map(c => c.key).join('|')) emit('order', keys)
}, { deep: true })
const ordered = computed(() => draft.value.map(key => props.columns.find(c => c.key === key)).filter((c): c is Column => !!c))
function move(index: number, offset: number) {
  const next = index + offset
  if (next < 0 || next >= draft.value.length) return
  const keys = [...draft.value]
  ;[keys[index], keys[next]] = [keys[next], keys[index]]
  draft.value = keys
}
</script>
