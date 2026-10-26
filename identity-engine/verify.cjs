'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { createHash } = require('node:crypto');
const COMMIT = '4dcb4216151cbd5066d515297720130a9fd3b198';
function verify(root = path.join(__dirname, 'vendor/lm-detector')) {
  const manifest = JSON.parse(fs.readFileSync(path.join(root, 'manifest.json')));
  if (manifest.commit !== COMMIT || manifest.repository !== 'https://github.com/Ikaleio/lm-detector') throw new Error('Fingerpoint source version mismatch');
  for (const [name, hash] of Object.entries(manifest.files)) {
    if (name.includes('..') || path.isAbsolute(name) || createHash('sha256').update(fs.readFileSync(path.join(root, name))).digest('hex') !== hash) throw new Error(`Fingerpoint artifact checksum mismatch: ${name}`);
  }
  return manifest;
}
module.exports = { COMMIT, verify };
