'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { bank, detector, models, COMMIT, runFingerprint, analyzeSharedOutputs } = require('./engine.cjs');
const fixtures = require('./fixtures/reference.json');
const addedModelFixtures = require('./fixtures/added-models.json');
const { verify } = require('./verify.cjs');
const { calibrateRanking } = require('./dist/shared/shared-detector.js');

test('pinned source and every packaged artifact verify; official merged class remains intact', () => {
  assert.equal(verify().commit, COMMIT);
  assert.equal(models.length, 63);
  assert.equal(models.find(m => m.id === 'gpt-6-astra').name, 'gpt-6-astra/6.1-sol');
  for (const id of ['kimi-k3.1-pro', 'gemini-3.5-flash-lite', 'minimax-m3.1-flash']) {
    assert.ok(models.some(model => model.id === id), `${id} is present in the calibrated reference bank`);
  }
});
for (const fixture of [...fixtures, ...addedModelFixtures]) test(`official reference replay: ${fixture.model.id}`, async () => {
  assert.equal(fixture.detector_reference_sha256, bank.reference_sha256);
  const actual = analyzeSharedOutputs(fixture.outputs, bank, detector, { allowPartial: true });
  assert.equal(actual.prediction, fixture.expected.prediction);
  assert.ok(Math.abs(actual.probability - fixture.expected.probability) < 1e-10);
  assert.deepEqual(actual.results.slice(0, 8).map(x => x.model), fixture.expected.top8);
  assert.equal(actual.evidence.insufficient, true, 'the upstream evidence flag is not a veto');
  let index = 0;
  const report = await runFingerprint({ expected_model: 'unregistered-expected-label', request_model: 'configured-route', options: { stream: false, parallel: false, relaxed: false }, signal: new AbortController().signal, emit() {}, challengesOverride: fixture.outputs.map((o,i) => ({ id: String(i), prompt: 'fixed safe challenge', expected_count: o.expected_count })), transportOverride: async (_url, config) => {
    assert.equal(config.model, 'configured-route');
    return Response.json({ choices: [{ message: { content: fixture.outputs[index++].text }, finish_reason: 'stop' }], usage: { completion_tokens: 900 } });
  }});
  assert.deepEqual(report.analysis.results, actual.results);
  assert.equal(report.analysis.probability, actual.probability);
  assert.ok(!Object.hasOwn(report, 'verdict'));
  assert.equal(report.samples.length, 3);
  assert.ok(report.samples.every(s => s.duration_ms >= 0 && s.ttft_ms === null && s.tps === null));
});
test('partial samples rank without a fabricated probability; bad calibration retains official order', () => {
  const outputs = fixtures[0].outputs;
  for (const count of [1,2]) {
    const a = analyzeSharedOutputs(outputs.slice(0,count), bank, detector, { allowPartial:true });
    assert.equal(a.decision,'partial'); assert.equal(a.used_outputs,count); assert.equal(a.results.length,63); assert.equal(a.probability,null);
    assert.ok(a.results.every(r => r.probability === null));
  }
  const mismatched = { ...detector, calibration: { ...detector.calibration, binding:{ ...detector.calibration.binding, base_sha256:'wrong' } } };
  const a=analyzeSharedOutputs(outputs,bank,mismatched,{allowPartial:true});
  assert.equal(a.probability,null);assert.deepEqual(a.results.map(x=>x.model),analyzeSharedOutputs(outputs,bank,detector).results.map(x=>x.model));
  assert.equal(calibrateRanking(Array(60).fill(0),mismatched),null);
});
test('programmatic sequences and insufficient numbers produce no ranking', () => {
  for (const text of ['1 2 3', Array.from({length:300},(_,i)=>i+1).join(',')]) {
    const a=analyzeSharedOutputs(Array.from({length:3},()=>({text,expected_count:300})),bank,detector,{allowPartial:true});
    assert.equal(a.results.length,0);assert.equal(a.probability,null);
    assert.ok(['not_model','unscorable'].includes(a.decision));
  }
});
test('one failed sample retains partial ranking and never retries in Node', async () => {
  const fixture=fixtures[0];let calls=0;
  const report=await runFingerprint({expected_model:fixture.model.id,request_model:'configured',options:{stream:false,parallel:false},signal:new AbortController().signal,emit(){},challengesOverride:fixture.outputs.map((o,i)=>({id:String(i),prompt:'safe',expected_count:o.expected_count})),transportOverride:async()=>{
    const i=calls++;if(i===1)return Response.json({error:{message:'temporary upstream failure'}},{status:503});
    return Response.json({choices:[{message:{content:fixture.outputs[i].text},finish_reason:'stop'}]});
  }});
  assert.equal(calls,3);assert.equal(report.analysis.decision,'partial');assert.equal(report.confidence,null);assert.ok(report.samples[1].error);
});
