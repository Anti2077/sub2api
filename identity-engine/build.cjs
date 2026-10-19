'use strict';
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');
require('./verify.cjs').verify();
const src = path.join(__dirname, 'vendor/lm-detector/shared'), out = path.join(__dirname, 'dist/shared');
fs.mkdirSync(out, { recursive: true });
for (const name of fs.readdirSync(src).filter(n => /\.(ts|js)$/.test(n))) {
  const source = fs.readFileSync(path.join(src, name), 'utf8');
  const result = ts.transpileModule(source, { fileName: name, compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS, esModuleInterop: true } });
  fs.writeFileSync(path.join(out, name.replace(/\.ts$/, '.js')), result.outputText);
}
