import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import VersionBadge from '../VersionBadge.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const zhMessages = {
  version: {
    updateAvailable: () => '有新版本可用！',
    currentVersion: () => '当前版本',
    refresh: () => '刷新',
    latestVersion: () => '最新版本',
    containerUpdateAvailable: () => '有新的个人镜像可用',
    containerOperatorHint: () => '容器更新由部署端执行',
    copied: () => '已复制',
    copyForOps: () => '复制给运维',
    containerDataSafeHint: () => '数据库、Redis 和配置卷会保留',
    viewSourceCommit: () => '查看源代码提交',
  },
}

describe('VersionBadge custom container updates', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows operator-managed GHCR commands instead of in-place update controls', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 1,
      username: 'admin',
      email: 'admin@example.com',
      role: 'admin',
      balance: 0,
      concurrency: 1,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: false,
      balance_notify_threshold: null,
    }

    const appStore = useAppStore()
    appStore.currentVersion = 'custom-abcdef0'
    appStore.latestVersion = 'custom-1234567'
    appStore.hasUpdate = true
    appStore.buildType = 'container'
    appStore.updateMode = 'container'
    appStore.dockerImage = 'ghcr.io/anti2077/sub2api:custom'
    appStore.releaseInfo = {
      name: 'custom-1234567',
      body: '',
      published_at: '',
      html_url: 'https://github.com/Anti2077/sub2api/commit/1234567',
    }
    appStore.fetchVersion = vi.fn().mockResolvedValue(null)

    const i18n = createI18n({
      legacy: false,
      locale: 'zh',
      messages: { zh: zhMessages },
    })
    const wrapper = mount(VersionBadge, {
      global: { plugins: [i18n] },
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('有新的个人镜像可用')
    expect(wrapper.text()).toContain('ghcr.io/anti2077/sub2api:custom')
    expect(wrapper.text()).toContain('docker compose pull sub2api')
    expect(wrapper.text()).toContain('docker compose up -d --no-deps sub2api')
    expect(wrapper.text()).not.toContain('立即更新')
    expect(wrapper.text()).not.toContain('版本回退')
  })
})
