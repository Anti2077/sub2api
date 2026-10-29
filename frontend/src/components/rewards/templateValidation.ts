import type { CardTemplate } from '@/api/rewards'
// Older card snapshots contain only model; a nonempty models list takes precedence.
export function templateModels(t: CardTemplate): string[] { return t.models?.length ? t.models : t.model ? [t.model] : [] }
export function validModelIDs(models: string[]): boolean {
  return models.length >= 1 && models.length <= 100 && models.every(id => !!id && new TextEncoder().encode(id).length <= 200 && !/[\s*?\p{Cc}]/u.test(id))
}
