// Development-only fixture using the production components and a local adapter.
// Synthetic accounts only. Every API request is intercepted before transport.
import { createApp, h, ref, computed } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import AccountFolderSidebar from '../src/components/admin/account/AccountFolderSidebar.vue'
import CreateAccountModal from '../src/components/account/CreateAccountModal.vue'
import ImportDataModal from '../src/components/admin/account/ImportDataModal.vue'
import { apiClient } from '../src/api/client'
import { accountTags, withAccountTags } from '../src/utils/accountTags'
import type { AccountFolder, AccountFolderRules } from '../src/api/admin/accountFolders'
import zh from '../src/i18n/locales/zh'
import en from '../src/i18n/locales/en'
import '../src/style.css'

const accounts = ref([
  { id: 1, name: 'OpenAI · 主力渠道', platform: 'openai', type: 'apikey', status: 'active', schedulable: true, extra: { account_tags: ['生产', '北区'] } },
  { id: 2, name: 'Grok · 图片账号', platform: 'grok', type: 'oauth', status: 'error', schedulable: true, extra: { account_tags: ['生产', '图像'] } },
  { id: 3, name: 'OpenAI · 测试账号', platform: 'openai', type: 'oauth', status: 'active', schedulable: false, extra: { account_tags: ['测试'] } },
  { id: 4, name: 'Gemini · 待整理', platform: 'gemini', type: 'apikey', status: 'active', schedulable: true, extra: { account_tags: [] } }
])
const folders = ref<AccountFolder[]>([
  { id: 1, name: '生产账号', parent_id: null, rules: { tags: ['生产'], tag_match: 'all' }, account_count: 0, error_count: 0 },
  { id: 2, name: 'OpenAI 主力', parent_id: 1, rules: { platforms: ['openai'], tags: ['生产'] }, account_count: 0, error_count: 0 },
  { id: 3, name: '待处理异常', parent_id: null, rules: { statuses: ['error'] }, account_count: 0, error_count: 0 }
])
function matches(account: typeof accounts.value[number], rules: AccountFolderRules) {
  const tags = accountTags(account.extra)
  return (!rules.platforms?.length || rules.platforms.includes(account.platform)) &&
    (!rules.types?.length || rules.types.includes(account.type)) &&
    (!rules.statuses?.length || rules.statuses.includes(account.status)) &&
    (!rules.name || account.name.toLowerCase().includes(rules.name.toLowerCase())) &&
    (rules.schedulable === undefined || account.schedulable === rules.schedulable) &&
    (rules.tag_state !== 'untagged' || !tags.length) && (rules.tag_state !== 'tagged' || tags.length > 0) &&
    (!rules.tags?.length || (rules.tag_match === 'any' ? rules.tags.some(tag => tags.includes(tag)) : rules.tags.every(tag => tags.includes(tag))))
}
function inFolder(account: typeof accounts.value[number], id: number) {
  return folders.value.some(folder => (folder.id === id || folder.parent_id === id) && matches(account, folder.rules))
}
const requests = ref<string[]>([])
apiClient.defaults.adapter = async config => {
  const body = typeof config.data === 'string' ? JSON.parse(config.data) : config.data
  const url = config.url || ''
  requests.value.push(`${config.method?.toUpperCase()} ${url}`)
  let data: unknown = {}
  if (url === '/admin/accounts/folders') {
    if (config.method === 'post') {
      const folder = { ...body, id: body.id || Math.max(0, ...folders.value.map(folder => folder.id)) + 1, account_count: 0, error_count: 0 }
      folders.value = [...folders.value.filter(value => value.id !== folder.id), folder]
      data = folder
    } else data = {
      folders: folders.value.map(folder => ({ ...folder, account_count: accounts.value.filter(account => inFolder(account, folder.id)).length, error_count: accounts.value.filter(account => inFolder(account, folder.id) && account.status === 'error').length })),
      total: accounts.value.length, unfiled: accounts.value.filter(account => !folders.value.some(folder => matches(account, folder.rules))).length
    }
  } else if (url.startsWith('/admin/accounts/folders/') && config.method === 'delete') {
    const id = Number(url.split('/').pop())
    folders.value = folders.value.filter(folder => folder.id !== id && folder.parent_id !== id)
  } else if (url === '/admin/accounts/tags') {
    if (config.method === 'post') {
      accounts.value = accounts.value.map(account => {
        if (!body.account_ids.includes(account.id)) return account
        const previous = accountTags(account.extra)
        const tags = body.operation === 'add' ? [...new Set([...previous, ...body.tags])] : previous.filter(tag => !body.tags.includes(tag))
        return { ...account, extra: withAccountTags(account.extra, tags) as typeof account.extra }
      })
      data = { updated: body.account_ids.length }
    } else data = [...new Set(accounts.value.flatMap(account => accountTags(account.extra)))].map(name => ({ name, account_count: accounts.value.filter(account => accountTags(account.extra).includes(name)).length }))
  } else if (url === '/admin/accounts' && config.method === 'post') {
    const account = { ...body, id: accounts.value.length + 1, status: 'active', schedulable: true }
    accounts.value.push(account); data = account
  } else if (url === '/admin/accounts/data') {
    for (const account of body.data.accounts) accounts.value.push({ ...account, id: accounts.value.length + 1, status: 'active', schedulable: true })
    data = { account_created: body.data.accounts.length, account_failed: 0, proxy_created: 0, proxy_failed: 0, proxy_reused: 0 }
  } else if (url.includes('tls-fingerprint') || url.includes('default-model-mapping')) data = []
  else if (url.includes('web-search')) data = { enabled: false, providers: [] }
  return { data, status: 200, statusText: 'OK', headers: {}, config }
}

createApp({
  setup() {
    const sidebar = ref<InstanceType<typeof AccountFolderSidebar>>()
    const selected = ref(''), checked = ref<number[]>([]), dark = ref(false), create = ref(false), importing = ref(false)
    const visible = computed(() => accounts.value.filter(account => !selected.value || (selected.value === 'unfiled' ? !folders.value.some(folder => matches(account, folder.rules)) : inFolder(account, Number(selected.value)))))
    const refresh = () => { checked.value = []; void sidebar.value?.refresh() }
    const toggle = (id: number) => { checked.value = checked.value.includes(id) ? checked.value.filter(value => value !== id) : [...checked.value, id] }
    return () => h('main', { class: 'min-h-screen bg-gray-50 p-4 text-gray-900 dark:bg-dark-950 dark:text-dark-100 md:p-8' }, [
      h('header', { class: 'mx-auto mb-5 flex max-w-6xl flex-wrap items-center justify-between gap-3' }, [
        h('div', [h('h1', { class: 'text-xl font-semibold' }, '账号管理'), h('p', { class: 'mt-1 text-sm text-gray-500 dark:text-dark-300' }, '本地模拟 · 智能文件夹与账号标签')]),
        h('div', { class: 'flex flex-wrap gap-2' }, [
          h('button', { class: 'btn btn-secondary', onClick: () => { dark.value = !dark.value; document.documentElement.classList.toggle('dark', dark.value) } }, '切换主题'),
          h('button', { class: 'btn btn-secondary', onClick: () => { importing.value = true } }, '导入账号'),
          h('button', { class: 'btn btn-primary', onClick: () => { create.value = true } }, '添加账号')
        ])
      ]),
      h('div', { class: 'mx-auto grid max-w-6xl items-start gap-4 lg:grid-cols-[14rem_minmax(0,1fr)]' }, [
        h(AccountFolderSidebar, { ref: sidebar, modelValue: selected.value, onSelect: value => { selected.value = value; checked.value = [] }, onChanged: refresh, onTagged: refresh }),
        h('section', { class: 'min-w-0 rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800' }, [
          h('div', { class: 'mb-4 flex flex-wrap items-center justify-between gap-2' }, [h('h2', { class: 'text-sm font-semibold' }, `${selected.value ? '筛选结果' : '全部账号'} · ${visible.value.length}`), h('button', { class: 'btn btn-secondary btn-sm', disabled: !checked.value.length, onClick: () => sidebar.value?.openTags(checked.value) }, `管理标签 (${checked.value.length})`)]),
          ...visible.value.map(account => h('label', { class: 'flex min-w-0 cursor-pointer items-start gap-3 border-t border-gray-100 py-4 dark:border-dark-700', 'data-account-id': account.id }, [
            h('input', { type: 'checkbox', class: 'mt-1', checked: checked.value.includes(account.id), onChange: () => toggle(account.id) }),
            h('div', { class: 'min-w-0 flex-1' }, [h('div', { class: 'font-medium' }, account.name), h('p', { class: 'mt-1 text-xs text-gray-500 dark:text-dark-300' }, `${account.platform} · ${account.type} · ${account.status}`), h('div', { class: 'mt-2 flex flex-wrap gap-1.5' }, accountTags(account.extra).map(tag => h('span', { class: 'max-w-40 truncate rounded bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200' }, tag)))])
          ]))
        ])
      ]),
      h(CreateAccountModal, { show: create.value, proxies: [], groups: [], onClose: () => { create.value = false }, onCreated: () => { create.value = false; refresh() } }),
      h(ImportDataModal, { show: importing.value, onClose: () => { importing.value = false }, onImported: () => { importing.value = false; refresh() } })
    ])
  }
}).use(createPinia()).use(createI18n({ legacy: false, locale: 'zh', messages: { zh, en } })).mount('#app')
