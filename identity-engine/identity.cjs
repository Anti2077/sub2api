'use strict';
const engine = require('./vendor/bazaarlink/dist/index.js');
const { classifyIdentityV2B } = require('./vendor/bazaarlink/dist/identity-classifier-v2.js');
const { fuseFamily } = require('./vendor/bazaarlink/dist/identity-family-fusion.js');
const { hasUsableLingData } = require('./vendor/bazaarlink/dist/identity-phase-gate.js');
const { hasV3HSeparator } = require('./vendor/bazaarlink/dist/sub-model-v3g-bias-fingerprint.js');
const { modelIdsMatch, canonicalFamily } = require('./vendor/bazaarlink/dist/model-id-normalize.js');
const snapshot = require('./vendor/bazaarlink/src/baselines-v3e-snapshot.json');
const COMMIT = '5c41136741ca52b5637879cca7bd0cae07404646';
const baselines = engine.loadV3EBaselinesFromSnapshot();
const models = [...new Map([...engine.V3_BASELINES, ...baselines, ...engine.BIAS_BASELINES].map(b => [b.modelId, {
  id: b.modelId, name: b.displayName || b.modelId, family: canonicalFamily(b.family || b.modelId.split('/')[0]),
}])).values()];

function limiter(max) {
  let active = 0;
  const waiting = [];
  return async fn => {
    if (active >= max) await new Promise(resolve => waiting.push(resolve));
    active++;
    try { return await fn(); } finally { active--; waiting.shift()?.(); }
  };
}

// Product verdicts require a specific model. Family matches and weak evidence abstain.
function productVerdict({ expected, detected, verdict, coverage, freshness, conflict }) {
  if (!detected || conflict || !coverage.total || verdict.confidence === 'low' || coverage.errors / Math.max(1, coverage.total) > 0.15 ||
      freshness.missing || freshness.expired || !['clean_match', 'clean_match_submodel_mismatch', 'plain_mismatch'].includes(verdict.status)) {
    return 'inconclusive';
  }
  return modelIdsMatch(expected, detected.modelId) ? 'matched' : 'mismatched';
}

function baselineFreshness(expected, detected, distribution, partition, now) {
  const ids = [...new Set([expected, detected?.modelId].filter(Boolean))];
  // Both ends of the comparison need a dated reference. A strong V3H result
  // can supply it only for models included in its validated candidate pool.
  const references = ids.map(id => {
    const baseline = baselines.find(b => b.modelId === id);
    if (baseline && Number.isFinite(Date.parse(baseline.updatedAt)) && baseline.sampleSize > 0) {
      return { id, missing: false, expired: now - Date.parse(baseline.updatedAt) > 180 * 86400000 };
    }
    const bias = [...partition.fresh, ...partition.expired.map(b => b.baseline)].find(b => b.modelId === id);
    const validated = distribution?.result.strongPass && Object.hasOwn(distribution.result.scores, id);
    return { id, missing: !bias || !validated, expired: !!bias && now - Date.parse(bias.capturedAt) > 180 * 86400000 };
  });
  return { missing: references.some(b => b.missing), expired: references.some(b => b.expired) ||
    !!distribution?.result.usedExpiredBaselines?.length, references, dropped: partition.dropped };
}

async function runIdentity(options, callProbe) {
  if (!models.some(m => m.id === options.expected_model)) throw new Error('unsupported expected model');
  const limited = limiter(4);
  const responses = {}, linguistic = {}, items = [];
  let total = 0, errors = 0, biasSequence = 0;
  const call = (id, prompt, maxTokens = 1024) => limited(async () => {
    if (options.signal?.aborted) throw new Error('cancelled');
    total++;
    try {
      const result = await callProbe({ probe_id: id, prompt, max_tokens: Math.min(maxTokens, 4096) });
      const choice = result.choices?.[0];
      const text = typeof choice?.message?.content === 'string' ? choice.message.content :
        ['refusal'].includes(choice?.finish_reason || choice?.native_finish_reason) ? '' : null;
      if (typeof text !== 'string') throw new Error('missing text response');
      items.push({ probe_id: id, response_text: text, finish_reason: choice?.native_finish_reason || choice?.finish_reason, response_model: result.model || null, usage: result.usage || null });
      return text;
    } catch (err) {
      errors++;
      items.push({ probe_id: id, error: err.message });
      return null;
    }
  });
  const probes = engine.PROBE_SUITE.filter(p => ['identity', 'submodel'].includes(p.group) && !p.multimodalContent &&
    !/^(ikp_|cap_|verb_|perf_|tok_edge_)/.test(p.id));
  await Promise.all(probes.map(async p => {
    const answers = await Promise.all(Array.from({ length: p.repeatCount || 1 }, (_, i) => call(`${p.id}:${i}`, p.prompt, p.maxTokens)));
    const valid = answers.filter(a => a !== null);
    if (valid.length) responses[p.id] = valid[0];
    if (p.repeatCount || p.id.startsWith('ling_')) linguistic[p.id] = answers.map(a => a === null ? 'ERR' : a);
  }));
  const features = engine.extractFingerprint(responses, linguistic);
  const behavioralFeatures = { ...features, linguisticFingerprint: Object.fromEntries(
    Object.entries(features.linguisticFingerprint).filter(([key]) => !key.startsWith('meta_creator_')),
  ) };
  const v2 = classifyIdentityV2B({ fingerprintFeatures: behavioralFeatures, observedResponses: responses, baselines: [], referenceSubModels: [] });
  // Self-declarations are retained in evidence only; use the engine's attack variant.
  const familyScores = v2.attackFamilyScores;
  const familyTop = Object.entries(familyScores).sort((a, b) => b[1] - a[1])[0];
  const global = engine.classifySubmodelV3(responses);
  const family = fuseFamily({ v2Family: hasUsableLingData(linguistic, {}) ? familyTop?.[0] : null, v2Scores: familyScores, v3FamilyImplied: global.familyImplied });
  const scoped = family.confirmedFamily ? engine.classifySubmodelV3(responses, { predictedFamily: family.confirmedFamily }) : null;
  const v3e = engine.classifySubmodelV3E(responses, baselines, { predictedFamily: family.confirmedFamily || undefined });
  const v3f = engine.classifySubmodelV3F(responses, baselines, { predictedFamily: family.confirmedFamily || undefined });
  const adapt = output => output ? { subModelMatch: output.top, candidates: output.candidates, abstained: output.abstained, familyImplied: output.familyImplied } : null;
  // The public engine does not ship the IKP raw corpus; it therefore abstains.
  const emptyRefusal = engine.detectNativeEmptyRefusal(items.map(i => ({probeId:i.probe_id.replace(/:\d+$/, ''),response:i.response_text,status:i.error?'error':'done'})));
  const fused = engine.fuseToV4(adapt(scoped), adapt(global), null, undefined, v3f.top,
    { confirmed: emptyRefusal.confirmed, isEmptyRefusalModel: id => engine.V3_BASELINES.some(b => b.modelId === id && b.nativeEmptyRefusal) }, v3e.top,
    family.confirmedFamily ? { family: family.confirmedFamily, confidence: family.confidence } : null);
  let detected = fused.subModelMatch;
  const coverage = { total, errors };
  const expectedFamily = models.find(m => m.id === options.expected_model).family;
  const computeVerdict = () => engine.computeVerdict({ claimedFamily: expectedFamily, claimedModel: options.expected_model,
    surface: null, behavior: family.confirmedFamily ? { family: family.confirmedFamily, score: family.confidence } : null,
    v3: detected, v3f: v3f.top, coverage,
    subModelSeparable: !detected || hasV3HSeparator(options.expected_model, detected.modelId),
  });
  let verdict = computeVerdict();
  const partition = engine.filterFreshBiasBaselines(engine.BIAS_BASELINES, { now: options.now || Date.now() });
  const candidates = family.confirmedFamily ? engine.candidateSiblingsForConfirmedFamily(family.confirmedFamily, options.expected_model, [...partition.fresh, ...partition.expired.map(b => b.baseline)]) : [];
  let distribution = null;
  if (family.confirmedFamily && family.confidence >= 0.65 && candidates.length >= 2) {
    distribution = await engine.sampleV3HDistributionFingerprint(prompt => call(`bias:${biasSequence++}`, prompt, 128), engine.BIAS_PROBES, candidates);
    if (distribution) {
      distribution.result.usedExpiredBaselines = partition.expired.filter(b => candidates.some(c => c.modelId === b.modelId)).map(b => b.modelId);
      const h = distribution.result;
      if (engine.shouldPromoteSubModelFromV3H(h, family.confirmedFamily, detected, options.expected_model)) {
        detected = { modelId: h.topModel, family: family.confirmedFamily, displayName: h.topModel, score: h.confidence };
        // The public engine's verdict patch only handles a clean-family
        // verdict. Recompute with the newly available submodel signal first.
        verdict = computeVerdict();
        verdict = engine.recomputeVerdictAfterV3HOverride(verdict, h.topModel, h.topModel, options.expected_model);
      }
      verdict = engine.resolveV3HVerdictPatch(verdict, null, h, family.confirmedFamily, options.expected_model, h.topModel || '', {
        familyConfidence: family.confidence, divergingBehaviorScore: family.confidence,
      }) || verdict;
    }
  }
  coverage.total = total; coverage.errors = errors;
  const freshness = baselineFreshness(options.expected_model, detected, distribution, partition, options.now || Date.now());
  const conflict = !family.confirmedFamily || (detected && canonicalFamily(detected.family) !== canonicalFamily(family.confirmedFamily)) ||
    (distribution && distribution.result.topModel && detected && distribution.result.topModel !== detected.modelId);
  return { engine_commit: COMMIT, engine_version: '0.11.0', baseline_version: snapshot.generatedAt,
    expected_model: options.expected_model, detected_model: detected?.modelId || null,
    verdict: productVerdict({ expected: options.expected_model, detected, verdict, coverage, freshness, conflict }),
    evidence: { family, v2, scoped, global, v3e, v3f, fused, emptyRefusal, distribution, engine_verdict: verdict, coverage, freshness }, probes: items,
  };
}
module.exports = { COMMIT, models, limiter, productVerdict, baselineFreshness, runIdentity };
