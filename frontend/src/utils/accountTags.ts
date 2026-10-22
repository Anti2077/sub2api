export const ACCOUNT_TAGS_KEY = 'account_tags'
export const MAX_ACCOUNT_TAGS = 32
export const MAX_ACCOUNT_TAG_LENGTH = 128

export function accountTags(extra?: Record<string, unknown> | null): string[] {
  const value = extra?.[ACCOUNT_TAGS_KEY]
  return Array.isArray(value) ? [...new Set(value.filter((tag): tag is string => typeof tag === 'string' && tag.trim() !== ''))] : []
}

export function validAccountTags(tags: string[]): boolean {
  return tags.length <= MAX_ACCOUNT_TAGS && tags.every(tag => tag.trim() !== '' && Array.from(tag).length <= MAX_ACCOUNT_TAG_LENGTH && !/\p{Cc}/u.test(tag))
}

export function withAccountTags(extra: Record<string, unknown> | undefined, tags: string[]): Record<string, unknown> {
  return { ...extra, [ACCOUNT_TAGS_KEY]: tags }
}
