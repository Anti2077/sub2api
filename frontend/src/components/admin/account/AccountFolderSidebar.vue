<template>
  <!-- Primary: saved scopes and counts; secondary: rule management; tertiary:
       explanation. Existing primary/gray/dark tokens and rounded-xl surfaces. -->
  <aside class="flex min-w-0 flex-col rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800 lg:max-h-80" data-testid="account-folder-panel">
    <div class="flex shrink-0 items-center justify-between gap-2">
      <button type="button" class="flex items-center gap-2 text-sm font-semibold text-gray-800 dark:text-gray-100 lg:pointer-events-none" :aria-expanded="mobileOpen" @click="mobileOpen = !mobileOpen">
        <Icon name="folder" size="sm" />{{ t('admin.accounts.folders.title') }}
        <Icon name="chevronDown" size="xs" class="lg:hidden" />
      </button>
      <button type="button" class="rounded p-2 text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-dark-700" :aria-label="t('admin.accounts.folders.create')" @click="edit()"><Icon name="plus" size="sm" /></button>
    </div>
    <p class="mt-2 text-xs leading-relaxed text-gray-500 dark:text-dark-300">{{ t('admin.accounts.folders.autoHint') }}</p>
    <div :class="[mobileOpen ? 'block' : 'hidden', 'mt-3 min-h-0 max-h-48 space-y-1 overflow-y-auto lg:block lg:max-h-none lg:flex-1']" :aria-busy="loading">
      <button type="button" :class="rowClass('')" :aria-current="modelValue === '' ? 'true' : undefined" @click="emit('select', '')"><span>{{ t('admin.accounts.folders.all') }}</span><span class="text-xs tabular-nums">{{ overview.total }}</span></button>
      <button type="button" :class="rowClass('unfiled')" :aria-current="modelValue === 'unfiled' ? 'true' : undefined" @click="emit('select', 'unfiled')"><span>{{ t('admin.accounts.folders.unfiled') }}</span><span class="text-xs tabular-nums">{{ overview.unfiled }}</span></button>
      <p v-if="error" class="px-2 py-2 text-xs text-red-600" role="alert">{{ error }} <button type="button" class="underline" @click="refresh">{{ t('common.tryAgain') }}</button></p>
      <div class="my-2 border-t border-gray-100 dark:border-dark-700" />
      <div v-for="root in roots" :key="root.id">
        <div class="flex items-center rounded-lg" :class="modelValue === String(root.id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''">
          <button v-if="children(root.id).length" type="button" class="rounded p-1.5 text-gray-500" :aria-label="root.name" :aria-expanded="!collapsed.has(root.id)" @click="toggle(root.id)"><Icon :name="collapsed.has(root.id) ? 'chevronRight' : 'chevronDown'" size="xs" /></button>
          <Icon v-else name="folder" size="xs" class="ml-2 shrink-0 text-gray-400" />
          <button type="button" class="flex min-w-0 flex-1 items-center gap-2 px-2 py-2.5 text-left text-sm text-gray-700 dark:text-gray-200" :aria-current="modelValue === String(root.id) ? 'true' : undefined" @click="emit('select', String(root.id))"><span class="flex-1 truncate" :title="ruleSummary(root)">{{ root.name }}</span><span v-if="root.error_count" class="h-1.5 w-1.5 rounded-full bg-red-500" :title="t('admin.accounts.folders.errors', { count: root.error_count })" /><span class="text-xs tabular-nums text-gray-500">{{ root.account_count }}</span></button>
          <button type="button" class="rounded p-1.5 text-gray-400 hover:text-primary-600" :aria-label="`${t('admin.accounts.folders.manage')}: ${root.name}`" @click="edit(root)"><Icon name="more" size="sm" /></button>
        </div>
        <div v-if="!collapsed.has(root.id)" class="ml-4 border-l border-gray-100 pl-1 dark:border-dark-700">
          <div v-for="child in children(root.id)" :key="child.id" class="flex items-center rounded-lg" :class="modelValue === String(child.id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''">
            <button type="button" class="flex min-w-0 flex-1 items-center gap-2 px-2 py-2.5 text-left text-sm text-gray-700 dark:text-gray-200" :aria-current="modelValue === String(child.id) ? 'true' : undefined" @click="emit('select', String(child.id))"><Icon name="folder" size="xs" class="shrink-0 text-gray-400" /><span class="flex-1 truncate" :title="ruleSummary(child)">{{ child.name }}</span><span v-if="child.error_count" class="h-1.5 w-1.5 rounded-full bg-red-500" :title="t('admin.accounts.folders.errors', { count: child.error_count })" /><span class="text-xs tabular-nums text-gray-500">{{ child.account_count }}</span></button>
            <button type="button" class="rounded p-1.5 text-gray-400 hover:text-primary-600" :aria-label="`${t('admin.accounts.folders.manage')}: ${child.name}`" @click="edit(child)"><Icon name="more" size="sm" /></button>
          </div>
        </div>
      </div>
      <p v-if="!loading && !roots.length && !error" class="px-2 py-3 text-xs leading-relaxed text-gray-500">{{ t('admin.accounts.folders.empty') }}</p>
    </div>
    <BaseDialog :show="showEdit" :title="editing ? t('admin.accounts.folders.manage') : t('admin.accounts.folders.create')" width="normal" @close="!busy && (showEdit = false)">
      <form id="account-folder-form" class="space-y-4" @submit.prevent="save">
        <fieldset :disabled="busy" class="space-y-4">
        <label class="block text-sm text-gray-700 dark:text-gray-200">{{ t('admin.accounts.folders.name') }}<input v-model="name" required maxlength="80" class="input mt-2 w-full" /></label>
        <label class="block text-sm text-gray-700 dark:text-gray-200">{{ t('admin.accounts.folders.parent') }}<select :aria-label="t('admin.accounts.folders.parent')" v-model="parentID" class="input mt-2 w-full" :disabled="!!editing && children(editing.id).length > 0"><option :value="null">{{ t('admin.accounts.folders.root') }}</option><option v-for="root in roots.filter(f => f.id !== editing?.id)" :key="root.id" :value="root.id">{{ root.name }}</option></select></label>
        <SmartFolderRulesEditor v-model="rules" />
        <p v-if="!hasRules" class="text-xs text-gray-500 dark:text-dark-300">{{ t('admin.accounts.folders.requireRule') }}</p>
        </fieldset>
        <p v-if="formError" class="text-sm text-red-600" role="alert">{{ formError }}</p>
      </form>
      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex gap-2"><button v-if="editing" type="button" class="btn btn-danger btn-sm" :disabled="busy" @click="showDelete = true">{{ t('common.delete') }}</button><button v-if="editing && !editing.parent_id" type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="edit(undefined, editing.id)">{{ t('admin.accounts.folders.createChild') }}</button></div>
          <button type="submit" form="account-folder-form" class="btn btn-primary btn-sm" :disabled="busy || !name.trim() || !hasRules">{{ busy ? t('common.saving') : t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>
    <ConfirmDialog :show="showDelete" :title="t('admin.accounts.folders.delete')" :message="t('admin.accounts.folders.deleteHint')" :confirm-text="t('common.delete')" @confirm="remove" @cancel="showDelete = false" />
    <BaseDialog :show="showTags" :title="t('admin.accounts.tags.bulkTitle')" width="normal" @close="!busy && (showTags = false)">
      <form id="account-tags-form" class="space-y-4" @submit.prevent="updateTags">
        <p class="text-sm text-gray-500 dark:text-dark-300">{{ t('admin.accounts.selectedCount', { count: tagIDs.length }) }}</p>
        <label class="block"><span class="input-label">{{ t('admin.accounts.tags.operation') }}</span>
          <select :aria-label="t('admin.accounts.tags.operation')" v-model="tagOperation" class="input" :disabled="busy"><option value="add">{{ t('admin.accounts.tags.add') }}</option><option value="remove">{{ t('admin.accounts.tags.remove') }}</option></select>
        </label>
        <AccountTagsInput v-model="tags" :disabled="busy" />
        <p class="input-hint">{{ t('admin.accounts.tags.bulkHint') }}</p>
        <p v-if="formError" class="text-sm text-red-600" role="alert">{{ formError }}</p>
      </form>
      <template #footer><button type="submit" form="account-tags-form" class="btn btn-primary" :disabled="busy || !tags.length">{{ busy ? t('common.saving') : t('common.save') }}</button></template>
    </BaseDialog>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountTagsInput from '@/components/account/AccountTagsInput.vue'
import SmartFolderRulesEditor from './SmartFolderRulesEditor.vue'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { accountFoldersAPI, type AccountFolder, type AccountFolderOverview, type AccountFolderRules } from '@/api/admin/accountFolders'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ select: [folder: string]; tagged: []; changed: [] }>()
const { t } = useI18n()
const overview = ref<AccountFolderOverview>({ folders: [], total: 0, unfiled: 0 })
const loading = ref(false), busy = ref(false), error = ref(''), formError = ref('')
const mobileOpen = ref(false), showEdit = ref(false), showDelete = ref(false), showTags = ref(false)
const editing = ref<AccountFolder>(), name = ref(''), parentID = ref<number | null>(null)
const rules = ref<AccountFolderRules>({})
const hasRules = computed(() => !!(rules.value.platforms?.length || rules.value.tags?.length || rules.value.types?.length || rules.value.statuses?.length || rules.value.name?.trim() || rules.value.tag_state || rules.value.expiry || rules.value.schedulable !== undefined))
const tags = ref<string[]>([]), tagIDs = ref<number[]>([]), tagOperation = ref<'add' | 'remove'>('add')
const collapsed = ref(new Set<number>())
try {
  const saved: unknown = JSON.parse(localStorage.getItem('account-folders-collapsed') || '[]')
  if (Array.isArray(saved)) collapsed.value = new Set(saved.filter((id): id is number => Number.isInteger(id) && id > 0))
} catch { /* Storage is optional. */ }
const roots = computed(() => overview.value.folders.filter(f => !f.parent_id))
const children = (id: number) => overview.value.folders.filter(f => f.parent_id === id)
function ruleSummary(folder: AccountFolder) {
  const r = folder.rules || {}
  const parts = [folder.name, ...(r.platforms || []).map(value => CONCRETE_PLATFORM_OPTIONS.find(p => p.value === value)?.label || value), ...(r.tags || []).map(tag => `#${tag}`)]
  if (r.name) parts.push(`${t('admin.accounts.folders.nameContains')}: ${r.name}`)
  return parts.join(' · ')
}
const rowClass = (key: string) => ['flex w-full items-center justify-between rounded-lg px-2 py-2.5 text-left text-sm', props.modelValue === key ? 'bg-primary-50 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' : 'text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-dark-700']
function toggle(id: number) {
  if (collapsed.value.has(id)) collapsed.value.delete(id); else collapsed.value.add(id)
  try { localStorage.setItem('account-folders-collapsed', JSON.stringify([...collapsed.value])) } catch { /* Storage is optional. */ }
}
async function refresh() {
  loading.value = true
  try {
    overview.value = await accountFoldersAPI.overview(); error.value = ''
    if (props.modelValue && props.modelValue !== 'unfiled' && !overview.value.folders.some(f => String(f.id) === props.modelValue)) emit('select', '')
  } catch { error.value = t('admin.accounts.folders.loadFailed') }
  finally { loading.value = false }
}
function edit(folder?: AccountFolder, parent: number | null = null) {
  rules.value = folder?.rules ? JSON.parse(JSON.stringify(folder.rules)) : {}; editing.value = folder; name.value = folder?.name || ''; parentID.value = folder?.parent_id ?? parent; formError.value = ''; showEdit.value = true
}
async function save() {
  if (busy.value || !hasRules.value) return
  busy.value = true
  try { await accountFoldersAPI.save({ id: editing.value?.id, name: name.value.trim(), parent_id: parentID.value, rules: rules.value }); showEdit.value = false; await refresh(); emit('changed') }
  catch (err: any) { formError.value = err?.response?.data?.message || err?.message || t('common.error') }
  finally { busy.value = false }
}
async function remove() {
  if (!editing.value || busy.value) return
  busy.value = true
  try { await accountFoldersAPI.delete(editing.value.id); showDelete.value = false; showEdit.value = false; await refresh(); emit('changed') }
  catch { showDelete.value = false; formError.value = t('common.error') }
  finally { busy.value = false }
}
function openTags(ids: number[]) { tagIDs.value = [...ids]; tags.value = []; tagOperation.value = 'add'; formError.value = ''; showTags.value = true }
async function updateTags() {
  if (busy.value || !tags.value.length) return
  busy.value = true
  try { await accountFoldersAPI.updateTags(tagIDs.value, tags.value, tagOperation.value); showTags.value = false; await refresh(); emit('tagged') }
  catch (err: any) { formError.value = err?.response?.data?.message || err?.message || t('common.error') }
  finally { busy.value = false }
}
onMounted(refresh)
defineExpose({ refresh, openTags, overview })
</script>
