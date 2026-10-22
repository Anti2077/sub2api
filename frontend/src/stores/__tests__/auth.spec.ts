import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

// Mock authAPI
const mockLogin = vi.fn()
const mockLogin2FA = vi.fn()
const mockLogout = vi.fn()
const mockGetCurrentUser = vi.fn()
const mockRegister = vi.fn()
const mockRefreshToken = vi.fn()

vi.mock('@/api', () => ({
  authAPI: {
    login: (...args: any[]) => mockLogin(...args),
    login2FA: (...args: any[]) => mockLogin2FA(...args),
    logout: (...args: any[]) => mockLogout(...args),
    getCurrentUser: (...args: any[]) => mockGetCurrentUser(...args),
    register: (...args: any[]) => mockRegister(...args),
    refreshToken: (...args: any[]) => mockRefreshToken(...args),
  },
  isTotp2FARequired: (response: any) => response?.requires_2fa === true,
}))

const fakeUser = {
  id: 1,
  username: 'testuser',
  email: 'test@example.com',
  role: 'user' as const,
  balance: 100,
  concurrency: 5,
  status: 'active' as const,
  allowed_groups: null,
  created_at: '2024-01-01',
  updated_at: '2024-01-01',
}

const fakeAdminUser = {
  ...fakeUser,
  id: 2,
  username: 'admin',
  email: 'admin@example.com',
  role: 'admin' as const,
}

const fakeAuthResponse = {
  access_token: 'test-token-123',
  refresh_token: 'refresh-token-456',
  expires_in: 3600,
  token_type: 'Bearer',
  user: { ...fakeUser },
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  // --- login ---

  describe('login', () => {
    it('成功登录后设置 token 和 user', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.token).toBe('test-token-123')
      expect(store.user).toEqual(fakeUser)
      expect(store.isAuthenticated).toBe(true)
      expect(localStorage.getItem('auth_token')).toBe('test-token-123')
      expect(localStorage.getItem('auth_user')).toBe(JSON.stringify(fakeUser))
    })

    it('登录失败时清除状态并抛出错误', async () => {
      mockLogin.mockRejectedValue(new Error('Invalid credentials'))
      const store = useAuthStore()

      await expect(store.login({ email: 'test@example.com', password: 'wrong' })).rejects.toThrow(
        'Invalid credentials'
      )

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('需要 2FA 时返回响应但不设置认证状态', async () => {
      const twoFAResponse = { requires_2fa: true, temp_token: 'temp-123' }
      mockLogin.mockResolvedValue(twoFAResponse)
      const store = useAuthStore()

      const result = await store.login({ email: 'test@example.com', password: '123456' })

      expect(result).toEqual(twoFAResponse)
      expect(store.token).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  // --- login2FA ---

  describe('login2FA', () => {
    it('2FA 验证成功后设置认证状态', async () => {
      mockLogin2FA.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      const user = await store.login2FA('temp-123', '654321')

      expect(store.token).toBe('test-token-123')
      expect(store.user).toEqual(fakeUser)
      expect(user).toEqual(fakeUser)
      expect(mockLogin2FA).toHaveBeenCalledWith({
        temp_token: 'temp-123',
        totp_code: '654321',
      })
    })

    it('2FA 验证失败时清除状态并抛出错误', async () => {
      mockLogin2FA.mockRejectedValue(new Error('Invalid TOTP'))
      const store = useAuthStore()

      await expect(store.login2FA('temp-123', '000000')).rejects.toThrow('Invalid TOTP')
      expect(store.token).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  // --- logout ---

  describe('logout', () => {
    it('注销后清除所有状态和 localStorage', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      mockLogout.mockResolvedValue(undefined)
      const store = useAuthStore()

      // 先登录
      await store.login({ email: 'test@example.com', password: '123456' })
      expect(store.isAuthenticated).toBe(true)

      // 注销
      await store.logout()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('auth_token')).toBeNull()
      expect(localStorage.getItem('auth_user')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
      expect(localStorage.getItem('token_expires_at')).toBeNull()
    })
  })

  // --- checkAuth ---

  describe('checkAuth', () => {
    it('从 localStorage 恢复持久化状态', () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))

      // Mock refreshUser (getCurrentUser) 防止后台刷新报错
      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })

      const store = useAuthStore()
      store.checkAuth()

      expect(store.token).toBe('saved-token')
      expect(store.user).toEqual(fakeUser)
      expect(store.isAuthenticated).toBe(true)
    })

    it('localStorage 无数据时保持未认证状态', () => {
      const store = useAuthStore()
      store.checkAuth()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('localStorage 中用户数据损坏时清除状态', () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', 'invalid-json{{{')

      const store = useAuthStore()
      store.checkAuth()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorage.getItem('auth_token')).toBeNull()
    })

    it('恢复 refresh token 和过期时间', () => {
      const futureTs = String(Date.now() + 3600_000)
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeUser))
      localStorage.setItem('refresh_token', 'saved-refresh')
      localStorage.setItem('token_expires_at', futureTs)

      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })

      const store = useAuthStore()
      store.checkAuth()

      expect(store.isAuthenticated).toBe(true)
    })

    it('恢复持久化 pending auth session', () => {
      localStorage.setItem(
        'pending_auth_session',
        JSON.stringify({
          token: 'pending-token',
          token_field: 'pending_auth_token',
          provider: 'wechat',
          redirect: '/profile',
        })
      )

      const store = useAuthStore()
      store.checkAuth()

      expect(store.hasPendingAuthSession).toBe(true)
      expect(store.pendingAuthSession).toEqual({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })
    })
  })

  describe('pending auth session', () => {
    it('persists and clears pending auth session state', () => {
      const store = useAuthStore()

      store.setPendingAuthSession({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })

      expect(store.hasPendingAuthSession).toBe(true)
      expect(JSON.parse(localStorage.getItem('pending_auth_session') || 'null')).toEqual({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'wechat',
        redirect: '/profile',
      })

      store.clearPendingAuthSession()

      expect(store.hasPendingAuthSession).toBe(false)
      expect(localStorage.getItem('pending_auth_session')).toBeNull()
    })

    it('restores a persisted pending oauth session without requiring a token value', () => {
      const firstStore = useAuthStore()

      firstStore.setPendingAuthSession({
        token: '',
        token_field: 'pending_oauth_token',
        provider: 'oidc',
        redirect: '/welcome',
        adoption_required: true,
        suggested_display_name: 'OIDC Nick'
      })

      setActivePinia(createPinia())
      const restoredStore = useAuthStore()
      restoredStore.checkAuth()

      expect(restoredStore.isAuthenticated).toBe(false)
      expect(restoredStore.hasPendingAuthSession).toBe(true)
      expect(restoredStore.pendingAuthSession).toEqual({
        token: '',
        token_field: 'pending_oauth_token',
        provider: 'oidc',
        redirect: '/welcome',
        adoption_required: true,
        suggested_display_name: 'OIDC Nick',
        suggested_avatar_url: undefined
      })
    })

    it('preserves pending auth session when registration fails', async () => {
      const store = useAuthStore()
      store.setPendingAuthSession({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'oidc',
        redirect: '/register',
      })
      mockRegister.mockRejectedValue(new Error('Register failed'))

      await expect(
        store.register({ email: 'user@example.com', username: 'user', password: 'secret-123' })
      ).rejects.toThrow('Register failed')

      expect(store.hasPendingAuthSession).toBe(true)
      expect(store.pendingAuthSession).toEqual({
        token: 'pending-token',
        token_field: 'pending_auth_token',
        provider: 'oidc',
        redirect: '/register',
      })
    })
  })

  // --- isAdmin ---

  describe('isAdmin', () => {
    it('管理员用户返回 true', async () => {
      const adminResponse = { ...fakeAuthResponse, user: { ...fakeAdminUser } }
      mockLogin.mockResolvedValue(adminResponse)
      const store = useAuthStore()

      await store.login({ email: 'admin@example.com', password: '123456' })

      expect(store.isAdmin).toBe(true)
    })

    it('普通用户返回 false', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.isAdmin).toBe(false)
    })

    it('未登录时返回 false', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })
  })

  // --- refreshUser ---

  describe('refreshUser', () => {
    it('刷新用户数据并更新 localStorage', async () => {
      mockLogin.mockResolvedValue(fakeAuthResponse)
      const store = useAuthStore()
      await store.login({ email: 'test@example.com', password: '123456' })

      const updatedUser = { ...fakeUser, username: 'updated-name' }
      mockGetCurrentUser.mockResolvedValue({ data: updatedUser })

      const result = await store.refreshUser()

      expect(result).toEqual(updatedUser)
      expect(store.user).toEqual(updatedUser)
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(updatedUser)
    })

    it('未认证时抛出错误', async () => {
      const store = useAuthStore()
      await expect(store.refreshUser()).rejects.toThrow('Not authenticated')
    })

    const cachedAvatar = {
      ...fakeUser,
      avatar_url: 'data:image/gif;base64,' + 'AAAA'.repeat(200 * 1024),
      avatar_version: 'v1-avatar-a'
    }

    async function loginWithAvatar(user: typeof fakeUser & {
      avatar_url?: string | null
      avatar_version?: string
    } = cachedAvatar) {
      mockLogin.mockResolvedValue({ ...fakeAuthResponse, user })
      const store = useAuthStore()
      await store.login({ email: user.email, password: '123456' })
      return store
    }

    function deferredUserResponse() {
      let resolve!: (response: { data: typeof fakeUser & {
        avatar_url?: string | null
        avatar_version?: string
      } }) => void
      let reject!: (reason: unknown) => void
      const promise = new Promise<Parameters<typeof resolve>[0]>((res, rej) => {
        resolve = res
        reject = rej
      })
      return { promise, resolve, reject }
    }

    it('fetches a full avatar initially, then reuses it while refreshing live fields every 60s', async () => {
      const store = await loginWithAvatar({ ...cachedAvatar, avatar_url: undefined, avatar_version: undefined })
      mockGetCurrentUser.mockResolvedValueOnce({ data: cachedAvatar })
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith(undefined)

      mockGetCurrentUser.mockResolvedValue({ data: {
        ...fakeUser, balance: 42, role: 'admin', run_mode: 'simple', avatar_version: 'v1-avatar-a'
      } })
      await vi.advanceTimersByTimeAsync(60_000)

      expect(mockGetCurrentUser).toHaveBeenLastCalledWith('v1-avatar-a')
      expect(store.user).toMatchObject({
        avatar_url: cachedAvatar.avatar_url, avatar_version: 'v1-avatar-a', balance: 42, role: 'admin'
      })
      expect(store.isSimpleMode).toBe(true)
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(store.user)
      await vi.advanceTimersByTimeAsync(60_000)
      expect(mockGetCurrentUser).toHaveBeenCalledTimes(3)
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith('v1-avatar-a')
    })

    it('negotiates the persisted avatar/version pair during startup', async () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', JSON.stringify(cachedAvatar))
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeUser, avatar_version: 'v1-avatar-a' } })
      const store = useAuthStore()
      await store.checkAuth()
      expect(mockGetCurrentUser).toHaveBeenCalledWith('v1-avatar-a')
      expect(store.user?.avatar_url).toBe(cachedAvatar.avatar_url)
    })

    it('fetches an incomplete persisted avatar cache without a validator', async () => {
      localStorage.setItem('auth_token', 'saved-token')
      localStorage.setItem('auth_user', JSON.stringify({ ...fakeUser, avatar_version: 'v1-avatar-a' }))
      mockGetCurrentUser.mockResolvedValue({ data: cachedAvatar })
      const store = useAuthStore()
      await store.checkAuth()
      expect(mockGetCurrentUser).toHaveBeenCalledWith(undefined)
      expect(store.user?.avatar_url).toBe(cachedAvatar.avatar_url)
    })

    it('replaces an avatar changed on another device and refreshes its source metadata', async () => {
      const store = await loginWithAvatar()
      const changed = {
        ...fakeUser, avatar_url: 'https://cdn.example.com/new.png', avatar_version: 'v1-avatar-b',
        avatar_source: { provider: 'oidc' }
      }
      mockGetCurrentUser.mockResolvedValueOnce({ data: changed })
      await store.refreshUser()
      expect(store.user).toEqual(changed)

      const { avatar_url: _avatar, ...unchanged } = changed
      mockGetCurrentUser.mockResolvedValueOnce({ data: unchanged })
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith('v1-avatar-b')
      expect(store.user).toEqual(changed)
    })

    it('clears a deleted avatar and persists the confirmed absence for future polls', async () => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeUser, avatar_version: 'v1-empty' } })
      await store.refreshUser()
      expect(store.user?.avatar_url).toBeNull()
      expect(JSON.parse(localStorage.getItem('auth_user')!).avatar_url).toBeNull()
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith('v1-empty')
      expect(store.user?.avatar_url).toBeNull()
    })

    it('handles an account that never had an avatar', async () => {
      const store = await loginWithAvatar({ ...cachedAvatar, avatar_url: undefined, avatar_version: undefined })
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeUser, avatar_version: 'v1-empty' } })
      await store.refreshUser()
      expect(store.user?.avatar_url).toBeNull()
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith('v1-empty')
    })

    it.each([null, ''])('honors an explicit cleared avatar value %s', async (avatar_url) => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeUser, avatar_version: 'v1-empty', avatar_url } })
      await store.refreshUser()
      expect(store.user?.avatar_url).toBe(avatar_url)
    })

    it('uses full replacement semantics with an older server that has no avatar_version', async () => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockResolvedValue({ data: fakeUser })
      await store.refreshUser()
      expect(store.user).toEqual(fakeUser)
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith(undefined)
    })

    it('never reuses an avatar for a different response owner', async () => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeAdminUser, avatar_version: 'v1-avatar-a' } })
      await store.refreshUser()
      expect(store.user?.id).toBe(fakeAdminUser.id)
      expect(store.user?.avatar_url).toBeNull()
    })

    it('does not resurrect an in-flight avatar response after logout', async () => {
      const store = await loginWithAvatar()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const refresh = store.refreshUser()
      await store.logout()
      pending.resolve({ data: cachedAvatar })
      await expect(refresh).rejects.toThrow('Authentication session changed')
      expect(store.user).toBeNull()
      expect(localStorage.getItem('auth_user')).toBeNull()
    })

    it('fetches a switched account without the previous avatar validator', async () => {
      const store = await loginWithAvatar()
      await store.logout()
      mockLogin.mockResolvedValue({ ...fakeAuthResponse, user: fakeAdminUser })
      await store.login({ email: fakeAdminUser.email, password: '123456' })
      mockGetCurrentUser.mockResolvedValue({ data: { ...fakeAdminUser, avatar_version: 'v1-other-empty' } })
      await store.refreshUser()
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith(undefined)
      expect(store.user?.avatar_url).toBeNull()
    })

    it.each(['success', '401'] as const)('ignores an old request %s after switching accounts', async (outcome) => {
      const store = await loginWithAvatar()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const refresh = store.refreshUser()
      await store.logout()
      mockLogin.mockResolvedValue({ ...fakeAuthResponse, user: fakeAdminUser })
      await store.login({ email: fakeAdminUser.email, password: '123456' })
      if (outcome === 'success') pending.resolve({ data: cachedAvatar })
      else pending.reject({ status: 401 })
      await expect(refresh).rejects.toBeDefined()
      expect(store.user).toEqual(fakeAdminUser)
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(fakeAdminUser)
      expect(store.isAuthenticated).toBe(true)
    })

    it.each(['success', '401'] as const)('ignores an old request %s after an account switch in another tab', async (outcome) => {
      const store = await loginWithAvatar()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const refresh = store.refreshUser()
      localStorage.setItem('auth_token', 'other-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeAdminUser))
      if (outcome === 'success') pending.resolve({ data: cachedAvatar })
      else pending.reject({ status: 401 })
      await expect(refresh).rejects.toBeDefined()
      expect(localStorage.getItem('auth_token')).toBe('other-token')
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(fakeAdminUser)
    })

    it('allows token rotation for the same account while polling', async () => {
      const store = await loginWithAvatar()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const refresh = store.refreshUser()
      localStorage.setItem('auth_token', 'rotated-token')
      pending.resolve({ data: { ...fakeUser, balance: 42, avatar_version: 'v1-avatar-a' } })
      await refresh
      expect(store.user?.balance).toBe(42)
      expect(store.user?.avatar_url).toBe(cachedAvatar.avatar_url)
    })

    it('fetches a full avatar during OAuth token initialization', async () => {
      const store = useAuthStore()
      mockGetCurrentUser.mockResolvedValueOnce({ data: cachedAvatar })
      await store.setToken('oauth-token')
      expect(mockGetCurrentUser).toHaveBeenLastCalledWith(undefined)
      expect(store.user?.avatar_url).toBe(cachedAvatar.avatar_url)
    })

    it.each(['success', '401'] as const)('preserves another tab account during OAuth initialization after a late %s', async (outcome) => {
      const store = useAuthStore()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const initialization = store.setToken('oauth-token')
      localStorage.setItem('auth_token', 'other-token')
      localStorage.setItem('auth_user', JSON.stringify(fakeAdminUser))
      if (outcome === 'success') pending.resolve({ data: cachedAvatar })
      else pending.reject({ status: 401 })
      await expect(initialization).rejects.toBeDefined()
      expect(localStorage.getItem('auth_token')).toBe('other-token')
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(fakeAdminUser)
      expect(store.user).toBeNull()
    })

    it('preserves a local replacement session during OAuth initialization', async () => {
      const store = useAuthStore()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const initialization = store.setToken('oauth-token')
      await store.logout()
      mockLogin.mockResolvedValue({ ...fakeAuthResponse, user: fakeAdminUser })
      await store.login({ email: fakeAdminUser.email, password: '123456' })
      pending.resolve({ data: cachedAvatar })
      await expect(initialization).rejects.toThrow('Authentication session changed')
      expect(store.user).toEqual(fakeAdminUser)
      expect(JSON.parse(localStorage.getItem('auth_user')!)).toEqual(fakeAdminUser)
    })

    it('does not overwrite a profile update completed while a poll was in flight', async () => {
      const store = await loginWithAvatar()
      const pending = deferredUserResponse()
      mockGetCurrentUser.mockReturnValueOnce(pending.promise)
      const refresh = store.refreshUser()
      const updated = { ...cachedAvatar, avatar_url: 'new-avatar', avatar_version: 'v1-new' }
      store.user = { ...store.user!, ...updated }
      localStorage.setItem('auth_user', JSON.stringify(updated))
      pending.resolve({ data: { ...fakeUser, avatar_version: 'v1-avatar-a' } })
      await refresh
      expect(store.user?.avatar_url).toBe('new-avatar')
      expect(store.user?.avatar_version).toBe('v1-new')
    })

    it.each([false, true])('clears an expired session (interceptor cleared storage: %s)', async (storageCleared) => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockImplementationOnce(async () => {
        if (storageCleared) localStorage.clear()
        throw { status: 401 }
      })
      await expect(store.refreshUser()).rejects.toEqual({ status: 401 })
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('does not clear a replacement session reported by the token interceptor', async () => {
      const store = await loginWithAvatar()
      mockGetCurrentUser.mockRejectedValueOnce({ status: 401, code: 'AUTH_SESSION_CHANGED' })
      await expect(store.refreshUser()).rejects.toMatchObject({ code: 'AUTH_SESSION_CHANGED' })
      expect(store.isAuthenticated).toBe(true)
    })
  })

  // --- isSimpleMode ---

  describe('isSimpleMode', () => {
    it('run_mode 为 simple 时返回 true', async () => {
      const simpleResponse = {
        ...fakeAuthResponse,
        user: { ...fakeUser, run_mode: 'simple' as const },
      }
      mockLogin.mockResolvedValue(simpleResponse)
      const store = useAuthStore()

      await store.login({ email: 'test@example.com', password: '123456' })

      expect(store.isSimpleMode).toBe(true)
    })

    it('默认为 standard 模式', () => {
      const store = useAuthStore()
      expect(store.isSimpleMode).toBe(false)
    })
  })
})
