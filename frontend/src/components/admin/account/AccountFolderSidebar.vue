<template>
  <aside class="flex min-w-0 flex-col rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800 lg:h-48" data-testid="account-folder-panel">
    <div class="flex shrink-0 items-center justify-between gap-2">
      <button type="button" class="flex items-center gap-2 text-sm font-semibold text-gray-800 dark:text-gray-100 lg:pointer-events-none" :aria-expanded="mobileOpen" @click="mobileOpen = !mobileOpen">
        <Icon name="folder" size="sm" />{{ t('admin.accounts.folders.title') }}
        <Icon name="chevronDown" size="xs" class="lg:hidden" />
      </button>
      <button type="button" class="rounded p-2 text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-dark-700" :aria-label="t('admin.accounts.folders.create')" @click="edit()"><Icon name="plus" size="sm" /></button>
    </div>
    <div :class="[mobileOpen ? 'block' : 'hidden', 'mt-3 min-h-0 max-h-48 space-y-1 overflow-y-auto lg:block lg:max-h-none lg:flex-1']" :aria-busy="loading">
      <button type="button" :class="rowClass('')" @click="emit('select', '')"><span>{{ t('admin.accounts.folders.all') }}</span><span class="text-xs tabular-nums">{{ overview.total }}</span></button>
      <button type="button" :class="rowClass('unfiled')" @click="emit('select', 'unfiled')"><span>{{ t('admin.accounts.folders.unfiled') }}</span><span class="text-xs tabular-nums">{{ overview.unfiled }}</span></button>
      <p v-if="error" class="px-2 py-2 text-xs text-red-600" role="alert">{{ error }} <button type="button" class="underline" @click="refresh">{{ t('common.tryAgain') }}</button></p>
      <div class="my-2 border-t border-gray-100 dark:border-dark-700" />
      <div v-for="root in roots" :key="root.id">
        <div class="flex items-center rounded-lg" :class="modelValue === String(root.id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''">
          <button v-if="children(root.id).length" type="button" class="rounded p-1.5 text-gray-500" :aria-label="root.name" :aria-expanded="!collapsed.has(root.id)" @click="toggle(root.id)"><Icon :name="collapsed.has(root.id) ? 'chevronRight' : 'chevronDown'" size="xs" /></button>
          <Icon v-else name="folder" size="xs" class="ml-2 shrink-0 text-gray-400" />
          <button type="button" class="flex min-w-0 flex-1 items-center gap-2 px-2 py-2.5 text-left text-sm text-gray-700 dark:text-gray-200" @click="emit('select', String(root.id))"><span class="flex-1 truncate" :title="root.name">{{ root.name }}</span><span v-if="root.error_count" class="h-1.5 w-1.5 rounded-full bg-red-500" :title="t('admin.accounts.folders.errors', { count: root.error_count })" /><span class="text-xs tabular-nums text-gray-500">{{ root.account_count }}</span></button>
          <button type="button" class="rounded p-1.5 text-gray-400 hover:text-primary-600" :aria-label="`${t('admin.accounts.folders.manage')}: ${root.name}`" @click="edit(root)"><Icon name="more" size="sm" /></button>
        </div>
        <div v-if="!collapsed.has(root.id)" class="ml-4 border-l border-gray-100 pl-1 dark:border-dark-700">
          <div v-for="child in children(root.id)" :key="child.id" class="flex items-center rounded-lg" :class="modelValue === String(child.id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''">
            <button type="button" class="flex min-w-0 flex-1 items-center gap-2 px-2 py-2.5 text-left text-sm text-gray-700 dark:text-gray-200" @click="emit('select', String(child.id))"><Icon name="folder" size="xs" class="shrink-0 text-gray-400" /><span class="flex-1 truncate" :title="child.name">{{ child.name }}</span><span v-if="child.error_count" class="h-1.5 w-1.5 rounded-full bg-red-500" :title="t('admin.accounts.folders.errors', { count: child.error_count })" /><span class="text-xs tabular-nums text-gray-500">{{ child.account_count }}</span></button>
            <button type="button" class="rounded p-1.5 text-gray-400 hover:text-primary-600" :aria-label="`${t('admin.accounts.folders.manage')}: ${child.name}`" @click="edit(child)"><Icon name="more" size="sm" /></button>
          </div>
        </div>
      </div>
      <p v-if="!loading && !roots.length && !error" class="px-2 py-3 text-xs leading-relaxed text-gray-500">{{ t('admin.accounts.folders.empty') }}</p>
    </div>
    <BaseDialog :show="showEdit" :title="editing ? t('admin.accounts.folders.manage') : t('admin.accounts.folders.create')" width="narrow" @close="showEdit = false">
      <form id="account-folder-form" class="space-y-4" @submit.prevent="save">
        <label class="block text-sm text-gray-700 dark:text-gray-200">{{ t('admin.accounts.folders.name') }}<input v-model="name" required maxlength="80" class="input mt-2 w-full" /></label>
        <label class="block text-sm text-gray-700 dark:text-gray-200">{{ t('admin.accounts.folders.parent') }}<select v-model="parentID" class="input mt-2 w-full" :disabled="!!editing && children(editing.id).length > 0"><option :value="null">{{ t('admin.accounts.folders.root') }}</option><option v-for="root in roots.filter(f => f.id !== editing?.id)" :key="root.id" :value="root.id">{{ root.name }}</option></select></label>
        <p v-if="formError" class="text-sm text-red-600" role="alert">{{ formError }}</p>
      </form>
      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex gap-2"><button v-if="editing" type="button" class="btn btn-danger btn-sm" :disabled="busy" @click="showDelete = true">{{ t('common.delete') }}</button><button v-if="editing && !editing.parent_id" type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="edit(undefined, editing.id)">{{ t('admin.accounts.folders.createChild') }}</button></div>
          <button type="submit" form="account-folder-form" class="btn btn-primary btn-sm" :disabled="busy || !name.trim()">{{ t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>
    <ConfirmDialog :show="showDelete" :title="t('admin.accounts.folders.delete')" :message="t('admin.accounts.folders.deleteHint')" :confirm-text="t('common.delete')" @confirm="remove" @cancel="showDelete = false" />
    <BaseDialog :show="showMove" :title="t('admin.accounts.folders.move')" width="narrow" @close="showMove = false">
      <p class="mb-3 text-sm text-gray-500">{{ t('admin.accounts.selectedCount', { count: moveIDs.length }) }}</p>
      <label class="block text-sm text-gray-700 dark:text-gray-200">{{ t('admin.accounts.folders.destination') }}<select v-model="destination" class="input mt-2 w-full"><option :value="null">{{ t('admin.accounts.folders.unfiled') }}</option><option v-for="folder in destinations" :key="folder.id" :value="folder.id">{{ folder.label }}</option></select></label>
      <p v-if="formError" class="mt-3 text-sm text-red-600" role="alert">{{ formError }}</p>
      <template #footer><button type="button" class="btn btn-primary" :disabled="busy" @click="move">{{ t('admin.accounts.folders.move') }}</button></template>
    </BaseDialog>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { accountFoldersAPI, type AccountFolder, type AccountFolderOverview } from '@/api/admin/accountFolders'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ select: [folder: string]; moved: []; changed: [] }>()
const { t } = useI18n()
const overview = ref<AccountFolderOverview>({ folders: [], total: 0, unfiled: 0 })
const loading = ref(false), busy = ref(false), error = ref(''), formError = ref('')
const mobileOpen = ref(false), showEdit = ref(false), showDelete = ref(false), showMove = ref(false)
const editing = ref<AccountFolder>(), name = ref(''), parentID = ref<number | null>(null)
const destination = ref<number | null>(null), moveIDs = ref<number[]>([])
const collapsed = ref(new Set<number>())
try {
  const saved: unknown = JSON.parse(localStorage.getItem('account-folders-collapsed') || '[]')
  if (Array.isArray(saved)) collapsed.value = new Set(saved.filter((id): id is number => Number.isInteger(id) && id > 0))
} catch { /* Storage is optional. */ }
const roots = computed(() => overview.value.folders.filter(f => !f.parent_id))
const children = (id: number) => overview.value.folders.filter(f => f.parent_id === id)
const destinations = computed(() => roots.value.flatMap(root => [{ id: root.id, label: root.name }, ...children(root.id).map(child => ({ id: child.id, label: `${root.name} / ${child.name}` }))]))
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
  editing.value = folder; name.value = folder?.name || ''; parentID.value = folder?.parent_id ?? parent; formError.value = ''; showEdit.value = true
}
async function save() {
  if (busy.value) return
  busy.value = true
  try { await accountFoldersAPI.save({ id: editing.value?.id, name: name.value.trim(), parent_id: parentID.value }); showEdit.value = false; await refresh(); emit('changed') }
  catch (err: any) { formError.value = err?.response?.data?.message || t('common.error') }
  finally { busy.value = false }
}
async function remove() {
  if (!editing.value || busy.value) return
  busy.value = true
  try { await accountFoldersAPI.delete(editing.value.id); showDelete.value = false; showEdit.value = false; await refresh(); emit('changed') }
  catch { showDelete.value = false; formError.value = t('common.error') }
  finally { busy.value = false }
}
function openMove(ids: number[]) { moveIDs.value = [...ids]; destination.value = null; formError.value = ''; showMove.value = true }
async function move() {
  if (busy.value) return
  busy.value = true
  try { await accountFoldersAPI.move(moveIDs.value, destination.value); showMove.value = false; await refresh(); emit('moved') }
  catch (err: any) { formError.value = err?.response?.data?.message || t('common.error') }
  finally { busy.value = false }
}
onMounted(refresh)
defineExpose({ refresh, openMove, overview })
</script>
