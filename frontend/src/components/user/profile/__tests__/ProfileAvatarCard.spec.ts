import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileAvatarCard from '@/components/user/profile/ProfileAvatarCard.vue'
import type { User } from '@/types'

const {
  updateProfileMock,
  showSuccessMock,
  showErrorMock,
  authStoreState
} = vi.hoisted(() => ({
  updateProfileMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  authStoreState: {
    user: null as User | null
  }
}))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: updateProfileMock
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock
  })
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (error: unknown) => (error as Error).message || 'request failed'
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        if (key === 'profile.avatar.title') return 'Profile avatar'
        if (key === 'profile.avatar.description') return 'Upload and manage your avatar'
        if (key === 'profile.avatar.uploadAction') return 'Upload image'
        if (key === 'profile.avatar.uploadHint') return 'Uploads up to 10MB are compressed to 1MB when saved'
        if (key === 'profile.avatar.saveSuccess') return 'Avatar updated'
        if (key === 'profile.avatar.deleteSuccess') return 'Avatar removed'
        if (key === 'profile.avatar.invalidType') return 'Please choose an image file'
        if (key === 'profile.avatar.uploadTooLarge') return 'Avatar uploads must be 10MB or smaller'
        if (key === 'profile.avatar.readFailed') return 'Failed to read the selected image'
        if (key === 'common.save') return 'Save'
        if (key === 'common.delete') return 'Delete'
        return key
      }
    })
  }
})

function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 5,
    username: 'alice',
    email: 'alice@example.com',
    avatar_url: null,
    role: 'user',
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-04-20T00:00:00Z',
    updated_at: '2026-04-20T00:00:00Z',
    ...overrides
  }
}

const originalFileReader = globalThis.FileReader

function installAvatarReadMock() {
  class MockFileReader {
    result: string | null = null
    onload: (() => void) | null = null

    readAsDataURL(blob: Blob) {
      this.result = `data:${blob.type};base64,${Buffer.from('original-avatar').toString('base64')}`
      this.onload?.()
    }
  }
  globalThis.FileReader = MockFileReader as unknown as typeof FileReader
}

async function selectFile(wrapper: ReturnType<typeof mount>, file: File) {
  const input = wrapper.get('[data-testid="profile-avatar-file-input"]')
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  await input.trigger('change')
  await flushPromises()
}

describe('ProfileAvatarCard', () => {
  beforeEach(() => {
    updateProfileMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    authStoreState.user = null
  })

  afterEach(() => {
    globalThis.FileReader = originalFileReader
    vi.restoreAllMocks()
  })

  it('does not render a manual avatar input field', () => {
    authStoreState.user = createUser()

    const wrapper = mount(ProfileAvatarCard, {
      props: {
        user: authStoreState.user
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.find('[data-testid="profile-avatar-input"]').exists()).toBe(false)
  })

  it('uploads the original image above 1MB for server compression', async () => {
    installAvatarReadMock()
    const updatedUser = createUser({ avatar_url: 'data:image/png;base64,b3JpZ2luYWwtYXZhdGFy' })
    updateProfileMock.mockResolvedValue(updatedUser)
    authStoreState.user = createUser()

    const wrapper = mount(ProfileAvatarCard, {
      props: {
        user: authStoreState.user
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const fileInput = wrapper.get('[data-testid="profile-avatar-file-input"]')
    Object.defineProperty(fileInput.element, 'files', {
      value: [new File([new Uint8Array(2 * 1024 * 1024)], 'avatar.png', { type: 'image/png' })],
      configurable: true
    })

    await fileInput.trigger('change')
    await flushPromises()
    await wrapper.get('[data-testid="profile-avatar-save"]').trigger('click')

    expect(updateProfileMock).toHaveBeenCalledWith({
      avatar_url: 'data:image/png;base64,b3JpZ2luYWwtYXZhdGFy'
    })
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('shows a preview after selecting an avatar in embedded mode', async () => {
    installAvatarReadMock()
    authStoreState.user = createUser()

    const wrapper = mount(ProfileAvatarCard, {
      props: {
        user: authStoreState.user,
        embedded: true
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const fileInput = wrapper.get('[data-testid="profile-avatar-file-input"]')
    Object.defineProperty(fileInput.element, 'files', {
      value: [new File([new Uint8Array(2 * 1024 * 1024)], 'avatar.png', { type: 'image/png' })],
      configurable: true
    })

    await fileInput.trigger('change')
    await flushPromises()

    const preview = wrapper.get('[data-testid="profile-avatar-preview"]')
    expect(preview.attributes('src')).toBe('data:image/png;base64,b3JpZ2luYWwtYXZhdGFy')
  })

  it('uploads an animated GIF above 1MB unchanged and uses the saved GIF', async () => {
    installAvatarReadMock()
    const saved = 'data:image/gif;base64,c2F2ZWQtZ2lm'
    updateProfileMock.mockResolvedValue(createUser({ avatar_url: saved }))
    const wrapper = mount(ProfileAvatarCard, { props: { user: createUser() } })
    await selectFile(wrapper, new File([new Uint8Array(2 * 1024 * 1024)], 'avatar.gif', { type: 'image/gif' }))
    expect(wrapper.get('[data-testid="profile-avatar-preview"]').attributes('src')).toBe('data:image/gif;base64,b3JpZ2luYWwtYXZhdGFy')
    await wrapper.get('[data-testid="profile-avatar-save"]').trigger('click')
    await flushPromises()
    expect(updateProfileMock).toHaveBeenCalledWith({ avatar_url: 'data:image/gif;base64,b3JpZ2luYWwtYXZhdGFy' })
    expect(authStoreState.user?.avatar_url).toBe(saved)
    expect(wrapper.get('[data-testid="profile-avatar-save"]').attributes('disabled')).toBeDefined()
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('accepts exactly 10MB and rejects larger files before reading', async () => {
    installAvatarReadMock()
    const wrapper = mount(ProfileAvatarCard, { props: { user: createUser() } })
    await selectFile(wrapper, new File([new Uint8Array(10 * 1024 * 1024)], 'avatar.gif', { type: 'image/gif' }))
    expect(wrapper.get('[data-testid="profile-avatar-save"]').attributes('disabled')).toBeUndefined()
    await selectFile(wrapper, new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'avatar.gif', { type: 'image/gif' }))
    expect(showErrorMock).toHaveBeenCalledWith('Avatar uploads must be 10MB or smaller')
    expect(wrapper.get('[data-testid="profile-avatar-save"]').attributes('disabled')).toBeDefined()
    expect(updateProfileMock).not.toHaveBeenCalled()
  })

  it('rejects unsupported files', async () => {
    const wrapper = mount(ProfileAvatarCard, { props: { user: createUser() } })
    await selectFile(wrapper, new File(['invalid'], 'avatar.txt', { type: 'text/plain' }))
    expect(showErrorMock).toHaveBeenCalledWith('Please choose an image file')
    expect(updateProfileMock).not.toHaveBeenCalled()
  })

  it('disables controls while saving and retains the draft on failure', async () => {
    installAvatarReadMock()
    let rejectSave!: (error: Error) => void
    updateProfileMock.mockImplementation(() => new Promise((_resolve, reject) => { rejectSave = reject }))
    const wrapper = mount(ProfileAvatarCard, { props: { user: createUser() } })
    await selectFile(wrapper, new File(['gif'], 'avatar.gif', { type: 'image/gif' }))
    await wrapper.get('[data-testid="profile-avatar-save"]').trigger('click')
    expect(wrapper.get('[data-testid="profile-avatar-file-input"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="profile-avatar-delete"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[role="status"]').text()).toBe('profile.avatar.saving')
    rejectSave(new Error('compression failed'))
    await flushPromises()
    expect(showErrorMock).toHaveBeenCalledWith('compression failed')
    expect(wrapper.get('[data-testid="profile-avatar-save"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-testid="profile-avatar-preview"]').attributes('src')).toBe('data:image/gif;base64,b3JpZ2luYWwtYXZhdGFy')
  })

  it('deletes the current avatar', async () => {
    const updatedUser = createUser({ avatar_url: null })
    updateProfileMock.mockResolvedValue(updatedUser)
    authStoreState.user = createUser({ avatar_url: 'https://cdn.example.com/old.png' })

    const wrapper = mount(ProfileAvatarCard, {
      props: {
        user: authStoreState.user
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.get('[data-testid="profile-avatar-delete"]').trigger('click')

    expect(updateProfileMock).toHaveBeenCalledWith({ avatar_url: '' })
    expect(authStoreState.user?.avatar_url).toBeNull()
    expect(showSuccessMock).toHaveBeenCalledWith('Avatar removed')
  })
})
