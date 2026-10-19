import { parseNumbers } from './fingerprint-core.js'

/** Replies whose smallest valid integer reaches this value have an abnormal distribution. */
export const ANOMALOUS_MINIMUM = 200

/** Zero-based indexes of replies whose valid integers are all at least `ANOMALOUS_MINIMUM`. */
export function anomalousSamples(texts: readonly string[]): number[] {
  return texts.flatMap((text, index) => {
    const numbers = parseNumbers(text) as number[]
    return numbers.length && Math.min(...numbers) >= ANOMALOUS_MINIMUM ? [index] : []
  })
}
