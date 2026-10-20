// Keep uncertain admin requests in this tab, scoped to the signed-in account.
// Only operation IDs and input are saved; authentication tokens are never copied.
function storageKey(kind: string) {
  const user = JSON.parse(localStorage.getItem('auth_user') || '{}') as {id?: number}
  return `rewards-management:${user.id || 'session'}:${kind}`
}
export function readManagementSession<T>(kind: string, valid: (value: unknown) => value is T): T | undefined {
  try {
    const value: unknown = JSON.parse(sessionStorage.getItem(storageKey(kind)) || 'null')
    return valid(value) ? value : undefined
  } catch { return undefined }
}
export function writeManagementSession(kind: string, value?: unknown): void {
  const key = storageKey(kind)
  if (value === undefined) sessionStorage.removeItem(key)
  else sessionStorage.setItem(key, JSON.stringify(value))
}
