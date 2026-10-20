import type { TokenizerBank } from './tokenizer-bank'

/** One answered request. `probe` is null for the baseline, which sends only the wrapper. */
export interface TokenizerObservation { probe: string | null; tokens: number; responseModel?: string }

/**
 * Bayesian open-set identification over three kinds of hypotheses:
 * - known class c: the upstream uses c's tokenizer, and a probe deviates only through measurement noise;
 * - relative of c: an unlisted tokenizer close to c that deviates on a sizable share of probes;
 * - alien: an unlisted tokenizer unrelated to every class; each delta follows the smoothed spread of all classes.
 * Per-probe residuals follow a two-component discrete Laplace mixture with a small floor, so one bad probe
 * costs a bounded amount of evidence. The contamination rate is marginalized over a session-level grid,
 * and the hidden request overhead over a window around the baseline, which observes it directly.
 */
export const TOKENIZER_MODEL = {
  prior: { known: 0.8, relative: 0.1, alien: 0.1 },
  /** Contamination rate grid as [rate, weight]. */
  knownRates: [[0.02, 0.4], [0.06, 0.4], [0.15, 0.2]],
  relativeRates: [[0.2, 0.3], [0.4, 0.4], [0.7, 0.3]],
  /** Two-sided geometric parameters of the narrow and wide residual components. */
  narrow: 0.02,
  wide: 0.85,
  floor: 1e-6,
  /** The overhead may differ from the first baseline by this many tokens. */
  offsetWindow: 3,
  /** Alien spread as a share of the probe's median count. */
  alienSpread: 0.15,
  minimumProbes: 4,
  maximumProbes: 20,
  stopAt: 0.99,
  /** Stop when the best batch would reduce the pairwise error bound by less than this. */
  minimumGain: 1e-4,
  /** Candidate classes kept when planning the next batch. */
  planned: 24,
} as const

type Grid = readonly (readonly [number, number])[]

export interface ClassScore {
  id: string
  /** Posterior that the upstream uses exactly this class's tokenizer. */
  exact: number
  /** Posterior that it uses an unlisted relative of this class. */
  related: number
  /** Probes whose count equals the class's count after removing the fitted overhead. */
  matched: number
  compared: number
  overhead: number
}
export interface LabScore { lab: string; lab_name: string; total: number }
export interface TokenizerPosterior {
  classes: ClassScore[]
  labs: LabScore[]
  /** Sum of every relative and the alien hypothesis. */
  unknown: number
  alien: number
  answered: number
}

const logSumExp = (values: number[]) => {
  const top = Math.max(...values)
  return top === -Infinity ? top : top + Math.log(values.reduce((sum, value) => sum + Math.exp(value - top), 0))
}
const logGeometric = (distance: number, p: number) => Math.log((1 - p) / (1 + p)) + Math.abs(distance) * Math.log(p)
const logResidual = (distance: number, rate: number) => logSumExp([
  Math.log(1 - rate) + logGeometric(distance, TOKENIZER_MODEL.narrow),
  Math.log(rate) + logGeometric(distance, TOKENIZER_MODEL.wide),
  Math.log(TOKENIZER_MODEL.floor),
])
const nearest = (delta: number, allowed: number[]) => allowed.reduce((best, value) => Math.abs(delta - value) < Math.abs(best) ? delta - value : best, Infinity)

interface Prepared { index: Map<string, number>; values: number[][][]; alienScale: number[] }
const prepared = new WeakMap<TokenizerBank, Prepared>()
function prepare(bank: TokenizerBank): Prepared {
  let cached = prepared.get(bank)
  if (!cached) {
    const index = new Map(bank.probes.map((probe, j) => [probe.id, j]))
    const values = bank.classes.map(item => item.counts.map((value, j) => item.alternatives?.[j] ?? [value]))
    const alienScale = bank.probes.map((_, j) => {
      const sorted = bank.classes.map(item => item.counts[j]).sort((a, b) => a - b)
      return Math.exp(-1 / Math.max(1.5, TOKENIZER_MODEL.alienSpread * sorted[Math.floor(sorted.length / 2)]))
    })
    cached = { index, values, alienScale }
    prepared.set(bank, cached)
  }
  return cached
}

interface Datum { j: number; tokens: number }
function data(bank: TokenizerBank, observations: TokenizerObservation[]): Datum[] {
  const { index } = prepare(bank)
  return observations.map(observation => {
    if (observation.probe === null) return { j: -1, tokens: observation.tokens }
    const j = index.get(observation.probe)
    if (j === undefined) throw new Error(`Probe text ${observation.probe} is not in the tokenizer bank.`)
    return { j, tokens: observation.tokens }
  })
}

const offsets = (baseline: number) => Array.from({ length: 2 * TOKENIZER_MODEL.offsetWindow + 1 }, (_, i) => baseline - TOKENIZER_MODEL.offsetWindow + i)
const allowedOf = (prepared: Prepared, c: number, j: number) => j === -1 ? [0] : prepared.values[c][j]

/**
 * A baseline carries no probe text, so its residual only measures overhead noise. It is scored with the same rate
 * under every hypothesis; otherwise a drifting overhead would count as evidence for an unlisted tokenizer.
 */
const BASELINE_RATE = TOKENIZER_MODEL.knownRates[0][0]

/** Log marginal likelihood of class c's counts under a rate grid, with the best offset for reporting. */
function classEvidence(prepared: Prepared, rows: Datum[], c: number, grid: Grid, baseline: number) {
  const terms: number[] = []
  let best = { log: -Infinity, offset: baseline }
  const prior = -Math.log(2 * TOKENIZER_MODEL.offsetWindow + 1)
  for (const offset of offsets(baseline)) {
    const residuals = rows.map(row => nearest(row.tokens - offset, allowedOf(prepared, c, row.j)))
    for (const [rate, weight] of grid) {
      const log = residuals.reduce((sum, value, i) => sum + logResidual(value, rows[i].j === -1 ? BASELINE_RATE : rate), 0)
      terms.push(prior + Math.log(weight) + log)
      if (log > best.log) best = { log, offset }
    }
  }
  return { log: logSumExp(terms), offset: best.offset }
}

function alienEvidence(prepared: Prepared, rows: Datum[], size: number, baseline: number) {
  const prior = -Math.log(2 * TOKENIZER_MODEL.offsetWindow + 1)
  return logSumExp(offsets(baseline).map(offset => prior + rows.reduce((sum, row) => {
    if (row.j === -1) return sum + logResidual(row.tokens - offset, BASELINE_RATE)
    const density = prepared.values.reduce((total, _, c) => total + Math.exp(logGeometric(nearest(row.tokens - offset, prepared.values[c][row.j]), prepared.alienScale[row.j])), 0) / size
    return sum + Math.log(density + TOKENIZER_MODEL.floor)
  }, 0)))
}

/** Returns null until the first baseline is answered, since it anchors the overhead. */
export function tokenizerPosterior(bank: TokenizerBank, observations: TokenizerObservation[]): TokenizerPosterior | null {
  const rows = data(bank, observations)
  const first = rows.find(row => row.j === -1)
  if (!first) return null
  const state = prepare(bank), size = bank.classes.length, prior = TOKENIZER_MODEL.prior
  const known = bank.classes.map((_, c) => classEvidence(state, rows, c, TOKENIZER_MODEL.knownRates, first.tokens))
  const relative = bank.classes.map((_, c) => classEvidence(state, rows, c, TOKENIZER_MODEL.relativeRates, first.tokens))
  const logs = {
    known: known.map(item => Math.log(prior.known / size) + item.log),
    relative: relative.map(item => Math.log(prior.relative / size) + item.log),
    alien: Math.log(prior.alien) + alienEvidence(state, rows, size, first.tokens),
  }
  const evidence = logSumExp([...logs.known, ...logs.relative, logs.alien])
  const probes = rows.filter(row => row.j !== -1)
  const classes = bank.classes.map((item, c): ClassScore => ({
    id: item.id,
    exact: Math.exp(logs.known[c] - evidence),
    related: Math.exp(logs.relative[c] - evidence),
    matched: probes.filter(row => nearest(row.tokens - known[c].offset, state.values[c][row.j]) === 0).length,
    compared: probes.length,
    overhead: known[c].offset,
  })).sort((a, b) => b.exact + b.related - a.exact - a.related)
  const labs = new Map<string, LabScore>()
  for (const [c, item] of bank.classes.entries()) {
    const lab = labs.get(item.lab) ?? { lab: item.lab, lab_name: item.lab_name, total: 0 }
    lab.total += Math.exp(logs.known[c] - evidence) + Math.exp(logs.relative[c] - evidence)
    labs.set(item.lab, lab)
  }
  const alien = Math.exp(logs.alien - evidence)
  return {
    classes, alien, answered: probes.length,
    unknown: alien + classes.reduce((sum, score) => sum + score.related, 0),
    labs: [...labs.values()].sort((a, b) => b.total - a.total),
  }
}

/** Planning plugs in the middle rate of each grid. */
const PLAN_RATE = { known: 0.06, relative: 0.4 } as const
/** Outcomes considered around the spread of class values when planning. */
const PLAN_WINDOW = 24
type AtomKind = keyof typeof PLAN_RATE | 'alien'
interface Atom { group: string; weight: number; kind: AtomKind; c: number }

const densityTables = new Map<number, Float64Array>()
/** exp(logResidual(d, rate)) for |d| < 2048; larger distances only meet the floor. */
function densityOf(distance: number, rate: number) {
  let table = densityTables.get(rate)
  if (!table) {
    table = Float64Array.from({ length: 2048 }, (_, d) => Math.exp(logResidual(d, rate)))
    densityTables.set(rate, table)
  }
  return table[Math.min(Math.abs(distance), 2047)]
}

interface Plan { low: number; size: number; roots: Map<string, Float64Array> }
const plans = new WeakMap<TokenizerBank, Plan[]>()
/**
 * Square roots of each atom's normalized predictive distribution over the outcome window of probe j, cached per
 * bank. The Bhattacharyya coefficient of two atoms is then the dot product of their root vectors.
 */
function rootsOf(bank: TokenizerBank, state: Prepared, atom: Atom, j: number) {
  let tables = plans.get(bank)
  if (!tables) {
    tables = bank.probes.map((_, p) => {
      const values = state.values.map(item => item[p][0])
      const low = Math.min(...values) - PLAN_WINDOW
      return { low, size: Math.max(...values) + PLAN_WINDOW - low + 1, roots: new Map() }
    })
    plans.set(bank, tables)
  }
  const plan = tables[j], key = `${atom.kind}:${atom.c}`
  let roots = plan.roots.get(key)
  if (!roots) {
    const raw = Float64Array.from({ length: plan.size }, (_, i) => {
      const y = plan.low + i
      if (atom.kind !== 'alien') return densityOf(y - state.values[atom.c][j][0], PLAN_RATE[atom.kind])
      return state.values.reduce((sum, item) => sum + Math.exp(logGeometric(y - item[j][0], state.alienScale[j])), 0)
    })
    const norm = raw.reduce((sum, value) => sum + value, 0)
    roots = raw.map(value => Math.sqrt(value / norm))
    plan.roots.set(key, roots)
  }
  return roots
}
const dot = (a: Float64Array, b: Float64Array) => { let sum = 0; for (let i = 0; i < a.length; i++) sum += a[i] * b[i]; return sum }
const initialPlans = new WeakMap<TokenizerBank, Map<number, string[]>>()

/**
 * Plans the next batch with the pairwise Bhattacharyya surrogate
 *   F(S) = sum over pairs a<b in different groups of sqrt(w_a w_b) * (1 - prod_{j in S} BC_j(a, b)),
 * with w normalized over the atoms of the `planned` leading classes,
 * which bounds the MAP error from above and is monotone submodular, so the greedy batch is within 1 - 1/e of the
 * best batch. With exact counts BC is 0 or 1 and F reduces to equivalence-class edge cutting (EC2). Each class
 * contributes its known atom and its relative atom (grouped with the alien as "unknown"), so batches also test the
 * leader against an unlisted tokenizer. Returns no probes when no candidate reduces the bound noticeably.
 */
export function nextProbes(bank: TokenizerBank, posterior: TokenizerPosterior | null, asked: Set<string>, count: number): string[] {
  // The plan before the first answer depends only on the bank, so it is computed once.
  const initial = !posterior && asked.size === 0 ? initialPlans.get(bank)?.get(count) : undefined
  if (initial) return [...initial]
  const state = prepare(bank)
  const position = new Map(bank.classes.map((item, c) => [item.id, c]))
  // Before the first answer every class is equally likely, so all of them take part.
  const scores = posterior?.classes.slice(0, TOKENIZER_MODEL.planned) ?? bank.classes.map(item => ({ id: item.id, exact: 1, related: 0 }))
  const atoms: Atom[] = []
  for (const score of scores) {
    const c = position.get(score.id) as number
    atoms.push({ group: score.id, weight: score.exact, kind: 'known', c })
    if (score.related > 0) atoms.push({ group: 'unknown', weight: score.related, kind: 'relative', c })
  }
  if (posterior && posterior.alien > 0) atoms.push({ group: 'unknown', weight: posterior.alien, kind: 'alien', c: -1 })
  const total = atoms.reduce((sum, atom) => sum + atom.weight, 0) || 1
  const pairs: { a: number; b: number; weight: number }[] = []
  for (let a = 0; a < atoms.length; a++) {
    for (let b = a + 1; b < atoms.length; b++) {
      if (atoms[a].group !== atoms[b].group) pairs.push({ a, b, weight: Math.sqrt(atoms[a].weight * atoms[b].weight) / total })
    }
  }
  const candidates = bank.probes.map((probe, j) => ({ id: probe.id, j })).filter(candidate => !asked.has(candidate.id))
  const coefficients = new Map(candidates.map(({ id, j }) => {
    const roots = atoms.map(atom => rootsOf(bank, state, atom, j))
    return [id, pairs.map(({ a, b }) => dot(roots[a], roots[b]))] as const
  }))
  const remaining = pairs.map(() => 1)
  const chosen: string[] = []
  while (chosen.length < count) {
    let best: { id: string; gain: number } | undefined
    for (const { id } of candidates) {
      if (chosen.includes(id)) continue
      const bc = coefficients.get(id) as readonly number[]
      const gain = pairs.reduce((sum, pair, t) => sum + pair.weight * remaining[t] * (1 - bc[t]), 0)
      if (!best || gain > best.gain) best = { id, gain }
    }
    if (!best || best.gain < TOKENIZER_MODEL.minimumGain) break
    chosen.push(best.id)
    const bc = coefficients.get(best.id) as readonly number[]
    bc.forEach((value, t) => { remaining[t] *= value })
  }
  if (!posterior && asked.size === 0) {
    const cache = initialPlans.get(bank) ?? new Map<number, string[]>()
    cache.set(count, [...chosen])
    initialPlans.set(bank, cache)
  }
  return chosen
}

export type TokenizerVerdictKind = 'exact' | 'related' | 'unknown'
export interface ClaimCheck {
  /** Classes the claimed model id is served with; empty when the vendor has not published its tokenizer. */
  expected: string[]
  vendor: string
  /**
   * The claim comes from a first-party API model id, whose vendor publishes the exact tokenizer, so an unlisted relative
   * does not support it. Alias claims of open-weight lineages also accept a relative, such as a retrained variant.
   */
  firstParty: boolean
  /** Posterior probability that the observed tokenizer is compatible with the claim. */
  probability: number
  status: 'consistent' | 'inconsistent' | 'uncertain'
}
export interface TokenizerVerdict {
  kind: TokenizerVerdictKind
  /** exact: the matching class; related: the class the unlisted tokenizer is closest to; unknown: the closest class. */
  top: ClassScore
  /** exact: P(exactly that class); related: P(an unlisted relative of that class); unknown: P(any unlisted tokenizer). */
  confidence: number
  claim: ClaimCheck | null
}

const normalizeModel = (model: string) => model.trim().toLowerCase().replace(/^.*\//, '').replace(/:.*$/, '')

/** Maps a model id to the classes it is served with. Provider prefixes and `:tag` suffixes are ignored. */
export function expectedClasses(bank: TokenizerBank, model: string): Omit<ClaimCheck, 'probability' | 'status'> | null {
  const id = normalizeModel(model)
  if (!id) return null
  const vendor = bank.api_models.find(entry => new RegExp(entry.pattern, 'i').test(id))
  if (vendor) return { expected: vendor.class ? [vendor.class] : [], vendor: vendor.vendor, firstParty: true }
  const classes = bank.classes.filter(item => item.aliases.some(alias => new RegExp(alias, 'i').test(id)))
  return classes.length ? { expected: classes.map(item => item.id), vendor: classes[0].lab_name, firstParty: false } : null
}

/** Posterior that the upstream uses one of `classes`, counting their unlisted relatives only when `relatives` is set. */
export const classPosterior = (posterior: TokenizerPosterior, classes: string[], relatives: boolean) => posterior.classes
  .filter(score => classes.includes(score.id)).reduce((sum, score) => sum + score.exact + (relatives ? score.related : 0), 0)

export function tokenizerVerdict(bank: TokenizerBank, posterior: TokenizerPosterior, model = ''): TokenizerVerdict {
  const byExact = posterior.classes.reduce((best, score) => score.exact > best.exact ? score : best)
  const byRelated = posterior.classes.reduce((best, score) => score.related > best.related ? score : best)
  // An unlisted tokenizer is named after its closest class only when that relative outweighs the alien hypothesis.
  const [kind, top, confidence]: [TokenizerVerdictKind, ClassScore, number] = byExact.exact >= posterior.unknown
    ? ['exact', byExact, byExact.exact]
    : byRelated.related >= posterior.alien ? ['related', byRelated, byRelated.related] : ['unknown', posterior.classes[0], posterior.unknown]
  const expected = model ? expectedClasses(bank, model) : null
  let claim: ClaimCheck | null = null
  if (expected) {
    const probability = expected.expected.length ? classPosterior(posterior, expected.expected, !expected.firstParty) : posterior.unknown
    claim = { ...expected, probability, status: probability >= 0.9 ? 'consistent' : probability <= 0.1 ? 'inconsistent' : 'uncertain' }
  }
  return { kind, top, confidence, claim }
}

/** Stops once one class or the unknown hypothesis is settled, or when the probe budget runs out. */
export function probingDone(posterior: TokenizerPosterior, maximum: number = TOKENIZER_MODEL.maximumProbes) {
  if (posterior.answered >= maximum) return true
  if (posterior.answered < TOKENIZER_MODEL.minimumProbes) return false
  const top = posterior.classes[0]
  return Math.max(top.exact, posterior.unknown) >= TOKENIZER_MODEL.stopAt
}
