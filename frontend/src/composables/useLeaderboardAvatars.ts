import { onScopeDispose, ref, watch, type Ref } from 'vue'
import { getLeaderboardAvatar, type PublicTokenRankingItem } from '@/api/usage'

const avatarKey = (item: PublicTokenRankingItem) => JSON.stringify([item.username, item.avatar_version])

// Keep binary requests out of the ranking's loading state. Both responsive
// layouts share each blob; only three images can compete for bandwidth at once.
export function useLeaderboardAvatars(ranking: Ref<PublicTokenRankingItem[]>) {
  const urls = ref(new Map<string, string>())
  const failed = ref(new Set<string>())
  let controller: AbortController | undefined

  watch(ranking, (items) => {
    controller?.abort()
    const activeController = new AbortController()
    controller = activeController
    const pending = items.filter((item) => !item.is_anonymous && item.avatar_version)
    const wanted = new Set(pending.map(avatarKey))
    for (const [key, url] of urls.value) {
      if (!wanted.has(key)) {
        URL.revokeObjectURL(url)
        urls.value.delete(key)
      }
    }
    failed.value = new Set()
    const queue = pending.filter((item) => !urls.value.has(avatarKey(item)))
    async function worker() {
      while (queue.length && !activeController.signal.aborted) {
        const item = queue.shift()!
        const key = avatarKey(item)
        try {
          const blob = await getLeaderboardAvatar(item.username, item.avatar_version!, activeController.signal)
          if (activeController.signal.aborted) return
          urls.value.set(key, URL.createObjectURL(blob))
        } catch {
          // An image failure must never replace a successful ranking with an error.
          if (!activeController.signal.aborted) failed.value.add(key)
        }
      }
    }
    const workerCount = Math.min(3, queue.length)
    for (let i = 0; i < workerCount; i++) void worker()
  })

  onScopeDispose(() => {
    controller?.abort()
    for (const url of urls.value.values()) URL.revokeObjectURL(url)
    urls.value.clear()
  })

  function avatarURL(item: PublicTokenRankingItem): string {
    if (item.is_anonymous) return ''
    if (item.avatar_version) {
      const key = avatarKey(item)
      return failed.value.has(key) ? '' : urls.value.get(key) || ''
    }
    const url = item.avatar_url?.trim() || ''
    return failed.value.has(url) ? '' : url
  }

  function markAvatarFailed(item: PublicTokenRankingItem) {
    failed.value.add(item.avatar_version ? avatarKey(item) : item.avatar_url?.trim() || '')
  }

  return { avatarURL, markAvatarFailed }
}
