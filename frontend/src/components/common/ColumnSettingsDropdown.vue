<template>
  <div ref="dropdownRef" class="relative">
    <button
      v-bind="$attrs"
      type="button"
      :data-testid="$attrs['data-testid'] || 'column-settings'"
      class="btn btn-secondary px-2 md:px-3"
      :title="title"
      :aria-expanded="isOpen"
      @click="isOpen = !isOpen"
    >
      <Icon name="grid" size="sm" />
      <span v-if="buttonLabel" class="hidden md:inline">{{ buttonLabel }}</span>
    </button>

    <div
      v-if="isOpen"
      class="absolute right-0 top-full z-50 mt-1 max-h-[28rem] w-64 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
    >
      <div class="border-b border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
        {{ resolvedDragHint }}
      </div>
      <VueDraggable
        v-model="draftOrder"
        :animation="150"
        handle=".column-drag-handle"
        ghost-class="column-settings-ghost"
        class="py-1"
      >
        <div
          v-for="column in orderedColumns"
          :key="column.key"
          class="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-200"
        >
          <button
            type="button"
            class="column-drag-handle shrink-0 cursor-grab rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 active:cursor-grabbing dark:hover:bg-dark-700 dark:hover:text-gray-200"
            :title="resolvedDragHint"
            :aria-label="`${resolvedDragHint}: ${column.label}`"
          >
            <Icon name="menu" size="sm" />
          </button>
          <button
            type="button"
            :disabled="isAlwaysVisible(column.key)"
            class="min-w-0 flex-1 truncate text-left hover:text-gray-900 disabled:cursor-not-allowed disabled:hover:text-inherit dark:hover:text-white"
            :data-testid="`usage-column-toggle-${column.key}`"
            @click="emit('toggle-column', column.key)"
          >
            {{ column.label }}
          </button>
          <input
            type="checkbox"
            class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-800"
            :checked="isVisible(column.key)"
            :disabled="isAlwaysVisible(column.key)"
            :title="isAlwaysVisible(column.key) ? resolvedAlwaysVisibleHint : undefined"
            :aria-label="column.label"
            @click.stop
            @change="emit('toggle-column', column.key)"
          />
        </div>
      </VueDraggable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from './types'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  columns: Column[]
  order: string[]
  hiddenKeys: string[]
  alwaysVisibleKeys?: string[]
  title: string
  buttonLabel?: string
  dragHint?: string
  alwaysVisibleHint?: string
}>(), {
  alwaysVisibleKeys: () => [],
  buttonLabel: '',
  dragHint: '',
  alwaysVisibleHint: '',
})

const emit = defineEmits<{
  (e: 'update:order', order: string[]): void
  (e: 'toggle-column', key: string): void
}>()

const { t } = useI18n()
const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const draftOrder = ref<string[]>([])
let skipNextOrderEmit = false

const resolvedDragHint = computed(() => props.dragHint || t('common.dragToReorder'))
const resolvedAlwaysVisibleHint = computed(() => props.alwaysVisibleHint || t('common.columnAlwaysVisible'))

const normalizeOrder = (candidate: string[]) => {
  const available = new Set(props.columns.map((column) => column.key))
  const known = new Set<string>()
  const normalized: string[] = []

  for (const key of candidate) {
    if (available.has(key) && !known.has(key)) {
      known.add(key)
      normalized.push(key)
    }
  }
  for (const column of props.columns) {
    if (!known.has(column.key)) normalized.push(column.key)
  }
  return normalized
}

const sameOrder = (left: string[], right: string[]) =>
  left.length === right.length && left.every((key, index) => key === right[index])

const syncDraftOrder = () => {
  const next = normalizeOrder(props.order)
  if (!sameOrder(draftOrder.value, next)) {
    skipNextOrderEmit = true
    draftOrder.value = next
  }
}

const orderedColumns = computed(() => {
  const columnsByKey = new Map(props.columns.map((column) => [column.key, column]))
  return draftOrder.value.map((key) => columnsByKey.get(key)).filter((column): column is Column => !!column)
})

const isVisible = (key: string) => !props.hiddenKeys.includes(key)
const isAlwaysVisible = (key: string) => props.alwaysVisibleKeys.includes(key)

const handleClickOutside = (event: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) isOpen.value = false
}

watch(draftOrder, (next) => {
  if (skipNextOrderEmit) {
    skipNextOrderEmit = false
    return
  }
  const normalized = normalizeOrder(next)
  if (!sameOrder(normalized, next)) {
    draftOrder.value = normalized
    return
  }
  if (!sameOrder(normalized, props.order)) emit('update:order', normalized)
}, { deep: true })
watch(() => [props.columns, props.order], syncDraftOrder, { immediate: true, deep: true })

onMounted(() => document.addEventListener('click', handleClickOutside))
onUnmounted(() => document.removeEventListener('click', handleClickOutside))

defineExpose({ draftOrder, orderedColumns })
</script>

<style scoped>
.column-settings-ghost {
  background: rgb(239 246 255 / 0.8);
  outline: 1px dashed rgb(59 130 246 / 0.6);
}

:global(.dark) .column-settings-ghost {
  background: rgb(30 58 138 / 0.35);
}
</style>
