<template>
  <!-- Primary: labels and editable value. Secondary: selected tags. Tertiary:
       suggestions/help. Uses the project's input, gray/dark and primary ramps. -->
  <div>
    <label :for="inputID" class="input-label">{{ label || t('admin.accounts.tags.title') }}</label>
    <input :id="inputID" ref="input" v-model="text" class="input" type="text"
      :disabled="disabled" :aria-describedby="`${inputID}-hint`" :aria-invalid="invalid || undefined"
      :placeholder="t('admin.accounts.tags.placeholder')" @input="validateDraft" @keydown="handleKeydown" @blur="commit" />
    <p :id="`${inputID}-hint`" class="input-hint" :class="invalid ? 'text-red-600 dark:text-red-400' : ''">
      {{ invalid ? t('admin.accounts.tags.invalid') : t('admin.accounts.tags.hint') }}
    </p>
    <div v-if="modelValue.length" class="mt-2 flex flex-wrap gap-1.5">
      <button v-for="tag in modelValue" :key="tag" type="button" :disabled="disabled"
        class="inline-flex min-h-9 max-w-full items-center gap-1 rounded-md bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200"
        :aria-label="`${t('admin.accounts.tags.remove')}: ${tag}`" @click="setTags(modelValue.filter(value => value !== tag))">
        <span class="truncate">{{ tag }}</span><span aria-hidden="true">×</span>
      </button>
    </div>
    <div v-if="suggestions.length && !disabled" class="mt-2 flex flex-wrap items-center gap-1.5">
      <span class="text-xs text-gray-500 dark:text-dark-300">{{ t('admin.accounts.tags.existing') }}</span>
      <button v-for="tag in suggestions" :key="tag.name" type="button"
        class="max-w-full truncate rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-dark-200 dark:hover:bg-dark-700"
        @click="setTags([...modelValue, tag.name])">{{ tag.name }} <span class="tabular-nums">{{ tag.account_count }}</span></button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { accountFoldersAPI, type AccountTag } from '@/api/admin/accountFolders'
import { validAccountTags } from '@/utils/accountTags'
const props = defineProps<{ modelValue: string[]; label?: string; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [tags: string[]] }>()
const { t } = useI18n()
const inputID = `account-tags-${useId()}`
const input = ref<HTMLInputElement>()
const text = ref('')
const knownTags = ref<AccountTag[]>([])
const candidates = computed(() => [...new Set([...props.modelValue, ...(text.value.trim() ? [text.value.trim()] : [])])])
const invalid = computed(() => !validAccountTags(candidates.value))
const suggestions = computed(() => knownTags.value.filter(tag => !props.modelValue.includes(tag.name) && tag.name.toLocaleLowerCase().includes(text.value.trim().toLocaleLowerCase())).slice(0, 8))
function validateDraft() {
  input.value?.setCustomValidity(invalid.value ? t('admin.accounts.tags.invalid') : '')
}
function handleKeydown(event: KeyboardEvent) {
  if (!event.isComposing && event.keyCode !== 229 && ['Enter', ',', '，'].includes(event.key)) {
    event.preventDefault()
    commit()
  }
}
function commit() {
  validateDraft()
  if (invalid.value || !text.value.trim()) return
  emit('update:modelValue', candidates.value)
  text.value = ''
}
function setTags(tags: string[]) {
  if (validAccountTags(tags)) emit('update:modelValue', tags)
}
watch(() => props.modelValue, validateDraft)
onMounted(async () => {
  try { knownTags.value = await accountFoldersAPI.tags() } catch { /* Suggestions are optional. */ }
})
</script>
