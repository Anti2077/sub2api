<template>
  <div class="space-y-2">
    <label v-if="label" class="input-label">{{ label }}</label>
    <div class="relative" ref="rootRef">
      <button type="button" class="flex min-h-11 w-full flex-wrap items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-left text-sm transition-colors hover:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800" :aria-expanded="open" aria-haspopup="true" @click="open = !open">
        <span v-if="selectedOptions.length === 0" class="text-gray-400 dark:text-dark-400">{{ placeholder }}</span>
        <span v-for="option in selectedOptions" :key="String(option.value)" class="inline-flex max-w-full items-center gap-1 rounded-md bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
          <span class="max-w-[15rem] truncate">{{ option.label }}</span>
          <button type="button" class="rounded p-0.5 hover:bg-primary-100 dark:hover:bg-primary-800" :aria-label="removeLabel" @click.stop="toggle(option.value)">×</button>
        </span>
        <span class="ml-auto text-gray-400">⌄</span>
      </button>
      <div v-if="open" class="absolute z-30 mt-1 w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-dark-600 dark:bg-dark-800">
        <div class="border-b border-gray-200 px-3 py-2 dark:border-dark-600"><input ref="searchRef" v-model="query" type="search" class="w-full bg-transparent text-sm outline-none" :placeholder="searchPlaceholder" :aria-label="searchPlaceholder" @keydown.esc="open = false" /></div>
        <div class="max-h-64 overflow-y-auto p-1" role="listbox" aria-multiselectable="true">
          <button v-for="option in filteredOptions" :key="String(option.value)" type="button" class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700" @click="toggle(option.value)">
            <span class="flex h-4 w-4 shrink-0 items-center justify-center rounded border" :class="selectedValues.includes(option.value) ? 'border-primary-500 bg-primary-500 text-white' : 'border-gray-300 dark:border-dark-500'">{{ selectedValues.includes(option.value) ? '✓' : '' }}</span><span class="min-w-0 truncate">{{ option.label }}</span>
          </button>
          <p v-if="filteredOptions.length === 0" class="px-3 py-4 text-center text-sm text-gray-500">{{ emptyText }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
export interface SearchMultiSelectOption { value: string | number; label: string }
const props = withDefaults(defineProps<{ modelValue: Array<string | number>; options: SearchMultiSelectOption[]; label?: string; placeholder?: string; searchPlaceholder?: string; emptyText?: string; removeLabel?: string }>(), { label: '', placeholder: 'Select options', searchPlaceholder: 'Search', emptyText: 'No options found', removeLabel: 'Remove option' })
const emit = defineEmits<{ 'update:modelValue': [value: Array<string | number>] }>()
const open = ref(false), query = ref(''), rootRef = ref<HTMLElement | null>(null), searchRef = ref<HTMLInputElement | null>(null)
const selectedValues = computed(() => props.modelValue)
const selectedOptions = computed(() => props.modelValue.map(value => props.options.find(option => option.value === value) ?? { value, label: String(value) }))
const filteredOptions = computed(() => { const q = query.value.trim().toLowerCase(); return q ? props.options.filter(option => option.label.toLowerCase().includes(q)) : props.options })
function toggle(value: string | number) { emit('update:modelValue', props.modelValue.includes(value) ? props.modelValue.filter(item => item !== value) : [...props.modelValue, value]) }
function onOutside(event: MouseEvent) { if (open.value && rootRef.value && !rootRef.value.contains(event.target as Node)) open.value = false }
watch(open, value => { if (value) nextTick(() => searchRef.value?.focus()) })
document.addEventListener('mousedown', onOutside)
onBeforeUnmount(() => document.removeEventListener('mousedown', onOutside))
</script>
