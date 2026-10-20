/** One probe string. The request sends `wrapper.prefix + text + wrapper.suffix` as the only user message. */
export interface TokenizerProbe { id: string; category: string; text: string }

/**
 * One tokenizer class: open tokenizers of one lineage whose counts agree on the probes.
 * `counts[j]` is `count(prefix + probes[j].text + suffix) - count(prefix + suffix)`.
 * `alternatives[j]` lists every value members disagree on for probe j.
 */
export interface TokenizerClass {
  id: string
  lab: string
  lab_name: string
  series: string
  series_zh: string
  members: string[]
  aliases: string[]
  counts: number[]
  alternatives?: Record<string, number[]>
  note?: string
}

/** A first-party API model id pattern. `class` is null when the vendor has not published its tokenizer. */
export interface TokenizerApiModel { pattern: string; vendor: string; class: string | null }

export interface TokenizerBank {
  schema: 'tokenizer-bank-v1'
  generated_at: string
  /** `excluded` lists archived repositories left out because their files do not reflect the model's tokenizer. */
  source: { tool: string; archive_manifest_sha256: string; tokenizers: number; candidates: number; excluded?: { key: string; reason: string }[] }
  wrapper: { prefix: string; suffix: string }
  probes: TokenizerProbe[]
  classes: TokenizerClass[]
  api_models: TokenizerApiModel[]
}

/** Throws unless the bank has the expected shape, so a stale or truncated file fails before any request is sent. */
export function assertTokenizerBank(value: unknown): asserts value is TokenizerBank {
  const bank = value as TokenizerBank
  if (bank?.schema !== 'tokenizer-bank-v1') throw new Error('The tokenizer bank schema must be tokenizer-bank-v1.')
  if (!Array.isArray(bank.probes) || !bank.probes.length || !Array.isArray(bank.classes) || bank.classes.length < 2) {
    throw new Error('The tokenizer bank needs a nonempty probes array and a classes array with two or more classes.')
  }
  if (typeof bank.wrapper?.prefix !== 'string' || typeof bank.wrapper?.suffix !== 'string') {
    throw new Error('The tokenizer bank needs wrapper.prefix and wrapper.suffix strings.')
  }
  const ids = new Set<string>()
  for (const item of bank.classes) {
    if (ids.has(item.id)) throw new Error(`Tokenizer class ${item.id} occurs more than one time in the tokenizer bank.`)
    ids.add(item.id)
    if (item.counts.length !== bank.probes.length || !item.counts.every(Number.isSafeInteger)) {
      throw new Error(`Tokenizer class ${item.id} needs one integer count for each probe text.`)
    }
  }
  if (!Array.isArray(bank.api_models)) throw new Error('The tokenizer bank needs an api_models array.')
  for (const model of bank.api_models) {
    if (model.class !== null && !ids.has(model.class)) throw new Error(`API model pattern ${model.pattern} points to class ${model.class}, which is not in the tokenizer bank.`)
  }
  // Compile the patterns now so a broken bank fails before any request is sent and billed.
  const patterns = [...bank.api_models.map(model => model.pattern), ...bank.classes.flatMap(item => Array.isArray(item.aliases) ? item.aliases : [null])]
  for (const pattern of patterns) {
    if (typeof pattern !== 'string') throw new Error('Each tokenizer class needs an aliases array.')
    try { new RegExp(pattern, 'i') } catch (error) { throw new Error(`Pattern ${pattern} in the tokenizer bank is not a valid regular expression: ${(error as Error).message}`) }
  }
}

export const wrapProbe = (bank: TokenizerBank, text: string) => bank.wrapper.prefix + text + bank.wrapper.suffix
