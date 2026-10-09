const { test } = require('node:test');
const assert = require('node:assert/strict');
const { limiter, productVerdict, baselineFreshness, models, runIdentity } = require('./identity.cjs');
const engine = require('./vendor/bazaarlink/dist/index.js');
const fixture = (changes = {}) => ({ expected: 'openai/gpt-5.5', detected: { modelId: 'openai/gpt-5.5' },
  verdict: { status: 'clean_match', confidence: 'high' }, coverage: { total: 100, errors: 0 }, freshness: {}, conflict: false, ...changes });
test('specific identity requires exact model and sufficient evidence', () => {
  assert.equal(productVerdict(fixture()), 'matched');
  assert.equal(productVerdict(fixture({ detected: { modelId: 'openai/gpt-5.3-codex' }, verdict: { status: 'clean_match_submodel_mismatch', confidence: 'high' } })), 'mismatched');
  for (const changes of [{ detected: null }, { verdict: { status: 'clean_match_family_only', confidence: 'high' } }, { freshness: { missing: true } },
    { freshness: { expired: true } }, { conflict: true }, { coverage: { total: 0, errors: 0 } }, { coverage: { total: 10, errors: 2 } }, { verdict: { status: 'ambiguous', confidence: 'high' } }]) {
    assert.equal(productVerdict(fixture(changes)), 'inconclusive');
  }
});
test('both expected and detected models require usable dated baselines', () => {
  const now = Date.parse('2026-10-10');
  const partition = engine.filterFreshBiasBaselines(engine.BIAS_BASELINES, { now });
  const detected = { modelId: 'openai/gpt-5.5' };
  assert.equal(baselineFreshness(detected.modelId, detected, null, partition, now).missing, false);
  assert.equal(baselineFreshness('missing/model', detected, null, partition, now).missing, true);
  assert.equal(baselineFreshness(detected.modelId, detected, null, partition, Date.parse('2027-10-10')).expired, true);
});
test('pinned engine verdicts keep same-family and conflicting evidence distinct', () => {
  const input = { claimedFamily: 'openai', claimedModel: 'openai/gpt-5.5', surface: null,
    behavior: { family: 'openai', score: 0.98 }, coverage: { total: 100, errors: 0 }, subModelSeparable: true };
  for (const [id, expected] of [['openai/gpt-5.5', 'matched'], ['openai/gpt-5.3-codex', 'mismatched']]) {
    const detected = { modelId: id, displayName: id, family: 'openai', score: 0.95 };
    const verdict = engine.computeVerdict({ ...input, v3: detected });
    assert.equal(productVerdict(fixture({ detected, verdict })), expected);
  }
  const familyOnly = engine.computeVerdict({ ...input, v3: null });
  assert.equal(productVerdict(fixture({ verdict: familyOnly })), 'inconclusive');
});
test('probe limiter never exceeds four concurrent requests', async () => {
  const limited = limiter(4); let active = 0, peak = 0;
  await Promise.all(Array.from({ length: 20 }, () => limited(async () => { peak = Math.max(peak, ++active);await new Promise(r => setTimeout(r, 5));active--; })));
  assert.equal(peak, 4);
});
test('all expected model choices come from pinned baseline IDs', () => {
  const ids = new Set([...engine.V3_BASELINES, ...engine.loadV3EBaselinesFromSnapshot(), ...engine.BIAS_BASELINES].map(x => x.modelId));
  assert.ok(models.length > 20);assert.ok(models.every(m => ids.has(m.id)));
});
test('engine freshness excludes missing metadata and thin samples', () => {
  const result = engine.filterFreshBiasBaselines([
    { modelId: 'missing', probes: {} }, { modelId: 'thin', capturedAt: '2026-10-01', sampleCount: 1, probes: {} },
    { modelId: 'stale', capturedAt: '2020-01-01', sampleCount: 100, probes: {} },
  ], { now: Date.parse('2026-10-10') });
  assert.equal(result.dropped.length, 2);assert.equal(result.expired.length, 1);
});
test('failed upstream probes do not claim a specific identity', async () => {
  const report = await runIdentity({ expected_model: models[0].id }, async () => { throw new Error('simulation failure'); });
  assert.equal(report.verdict, 'inconclusive');assert.equal(report.evidence.coverage.errors, report.evidence.coverage.total);
});
test('full pinned baseline replay updates the verdict after distribution sampling', async () => {
  const baseline = engine.BIAS_BASELINES.find(b => b.modelId === 'openai/gpt-5.5');
  const responses = {
    identity_style_en: 'Certainly! The most important skill is reasoning.', identity_style_zh_tw: 'Clear thinking.',
    identity_reasoning_shape: 'Let me solve this step by step.', identity_self_knowledge: 'I am Claude by Anthropic.',
    identity_json_discipline: '{"ok":true}', identity_refusal_pattern: "I'm sorry, but I cannot provide that.",
    ling_kr_num: '\uB9C8\uD754\uB458', ling_jp_pm: 'Ishiba', ling_fr_pm: 'Bayrou', ling_ru_pres: 'Vladimir Putin',
    tok_count_num: '4', tok_split_word: 'token|ization', tok_self_knowledge: 'tiktoken', meta_context_len: '128000',
    meta_creator: 'Anthropic', ling_uk_pm: 'Starmer', ling_kr_crisis: 'December 2024 martial law', ling_de_chan: 'Merz',
    submodel_cutoff: '2025-04', submodel_capability: '1. 3\n2. tuesday\n3. 6\n4. 541\n5. etadommocca',
    submodel_refusal: "I'm sorry, but I can't help with that.", submodel_selfspec: 'I do not have access to internal parameters.',
  };
  const call = async probe => {
    const bias = engine.BIAS_PROBES.find(p => p.prompt === probe.prompt);
    const text = bias ? Object.entries(baseline.probes[bias.id]).sort((a,b) => b[1] - a[1])[0][0] :
      responses[probe.probe_id.replace(/:\d+$/, '')] || 'A concise answer.';
    return { model: 'anthropic/claude-opus-4.7', choices: [{ message: { content: text } }] };
  };
  for (const [expected, now, verdict] of [
    ['openai/gpt-5.5', '2026-10-10', 'matched'],
    ['openai/gpt-5.3-codex', '2026-10-10', 'mismatched'],
    ['openai/gpt-5.5', '2027-10-10', 'inconclusive'],
  ]) {
    const report = await runIdentity({ expected_model: expected, now: Date.parse(now) }, call);
    assert.equal(report.detected_model, baseline.modelId);
    assert.equal(report.verdict, verdict);
    assert.equal(report.evidence.distribution.result.strongPass, true);
    assert.ok(report.probes.every(p => !/^(ikp_|cap_|verb_|perf_|tok_edge_)/.test(p.probe_id)));
  }
});
