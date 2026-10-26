'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../..');
const { COMMIT, verify } = require('../../identity-engine/verify.cjs');

test('Go acceptance pin matches the verified packaged engine and reference data', () => {
  assert.equal(verify().commit, COMMIT);
  const source = fs.readFileSync(path.join(root, 'backend/internal/service/model_identity_port.go'), 'utf8');
  const pin = source.match(/const IdentityEngineCommit = "([a-f0-9]{40})"/);
  assert.ok(pin, 'Go must declare the engine version it accepts');
  assert.equal(pin[1], COMMIT, 'updating the worker without Go would break catalog and run acceptance');
});
