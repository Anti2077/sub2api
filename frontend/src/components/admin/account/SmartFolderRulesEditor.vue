<template>
  <!-- Primary: matching conditions. Secondary: combination semantics.
       Uses existing input styles, gap-4/5 spacing, gray/dark text and surfaces. -->
  <div class="space-y-5">
    <p class="rounded-lg bg-gray-50 p-3 text-sm leading-relaxed text-gray-600 dark:bg-dark-900 dark:text-dark-200">{{ t('admin.accounts.folders.rulesHint') }}</p>
    <SearchMultiSelect :model-value="modelValue.platforms || []" :options="CONCRETE_PLATFORM_OPTIONS"
      :label="t('admin.accounts.platform')" :placeholder="t('admin.accounts.allPlatforms')"
      :search-placeholder="t('common.search')" :empty-text="t('common.noData')" :remove-label="t('common.delete')"
      @update:model-value="update('platforms', $event.map(String))" />
    <label class="block"><span class="input-label">{{ t('admin.accounts.folders.nameContains') }}</span>
      <input :value="modelValue.name || ''" maxlength="200" class="input" type="text" @input="update('name', ($event.target as HTMLInputElement).value)" />
    </label>
    <AccountTagsInput :model-value="modelValue.tags || []" :disabled="modelValue.tag_state === 'untagged'" @update:model-value="update('tags', $event)" />
    <div class="grid gap-4 sm:grid-cols-2">
      <label class="block"><span class="input-label">{{ t('admin.accounts.folders.tagMatch') }}</span>
        <select :aria-label="t('admin.accounts.folders.tagMatch')" :value="modelValue.tag_match || 'all'" class="input" @change="update('tag_match', ($event.target as HTMLSelectElement).value)">
          <option value="all">{{ t('admin.accounts.folders.tagsAll') }}</option><option value="any">{{ t('admin.accounts.folders.tagsAny') }}</option>
        </select>
      </label>
      <label class="block"><span class="input-label">{{ t('admin.accounts.folders.tagState') }}</span>
        <select :aria-label="t('admin.accounts.folders.tagState')" :value="modelValue.tag_state || ''" class="input" @change="update('tag_state', ($event.target as HTMLSelectElement).value)">
          <option value="">{{ t('admin.accounts.folders.any') }}</option><option value="tagged">{{ t('admin.accounts.folders.tagged') }}</option><option value="untagged">{{ t('admin.accounts.folders.untagged') }}</option>
        </select>
      </label>
    </div>
    <details class="rounded-lg border border-gray-200 p-3 dark:border-dark-600" :open="hasAdvanced || undefined">
      <summary class="cursor-pointer text-sm font-medium text-gray-700 dark:text-dark-100">{{ t('admin.accounts.folders.advanced') }}</summary>
      <div class="mt-4 space-y-4">
        <SearchMultiSelect :model-value="modelValue.types || []" :options="typeOptions" :label="t('admin.accounts.folders.accountType')"
          :placeholder="t('admin.accounts.allTypes')" :search-placeholder="t('common.search')" :empty-text="t('common.noData')" :remove-label="t('common.delete')"
          @update:model-value="update('types', $event.map(String))" />
        <SearchMultiSelect :model-value="modelValue.statuses || []" :options="statusOptions" :label="t('admin.accounts.folders.storedStatus')"
          :placeholder="t('admin.accounts.allStatus')" :search-placeholder="t('common.search')" :empty-text="t('common.noData')" :remove-label="t('common.delete')"
          @update:model-value="update('statuses', $event.map(String))" />
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block"><span class="input-label">{{ t('admin.accounts.folders.scheduling') }}</span>
            <select :aria-label="t('admin.accounts.folders.scheduling')" :value="modelValue.schedulable === undefined ? '' : String(modelValue.schedulable)" class="input" @change="update('schedulable', scheduleValue($event))">
              <option value="">{{ t('admin.accounts.folders.any') }}</option><option value="true">{{ t('admin.accounts.folders.enabled') }}</option><option value="false">{{ t('admin.accounts.folders.disabled') }}</option>
            </select>
          </label>
          <label class="block"><span class="input-label">{{ t('admin.accounts.folders.expiry') }}</span>
            <select :aria-label="t('admin.accounts.folders.expiry')" :value="modelValue.expiry || ''" class="input" @change="update('expiry', ($event.target as HTMLSelectElement).value)">
              <option value="">{{ t('admin.accounts.folders.any') }}</option><option value="expired">{{ t('admin.accounts.folders.expired') }}</option><option value="expiring_7d">{{ t('admin.accounts.folders.expiringSoon') }}</option><option value="never">{{ t('admin.accounts.folders.neverExpires') }}</option>
            </select>
          </label>
        </div>
      </div>
    </details>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountTagsInput from '@/components/account/AccountTagsInput.vue'
import SearchMultiSelect from '@/components/common/SearchMultiSelect.vue'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
import type { AccountFolderRules } from '@/api/admin/accountFolders'
const props = defineProps<{ modelValue: AccountFolderRules }>()
const emit = defineEmits<{ 'update:modelValue': [rules: AccountFolderRules] }>()
const { t } = useI18n()
const hasAdvanced = computed(() => !!(props.modelValue.types?.length || props.modelValue.statuses?.length || props.modelValue.schedulable !== undefined || props.modelValue.expiry))
const typeOptions = computed(() => [
  { value: 'oauth', label: 'OAuth' }, { value: 'setup-token', label: 'Setup Token' }, { value: 'apikey', label: 'API Key' },
  { value: 'upstream', label: t('admin.accounts.folders.upstream') }, { value: 'bedrock', label: 'AWS Bedrock' }, { value: 'service_account', label: 'Service Account' }
])
const statusOptions = computed(() => ['active', 'inactive', 'error'].map(value => ({ value, label: t(`admin.accounts.status.${value}`) })))
function scheduleValue(event: Event) { const value = (event.target as HTMLSelectElement).value; return value === '' ? undefined : value === 'true' }
function update(key: keyof AccountFolderRules, value: unknown) {
  const rules = { ...props.modelValue, [key]: value }
  if (key === 'tag_state' && value === 'untagged') rules.tags = []
  emit('update:modelValue', rules)
}
</script>
