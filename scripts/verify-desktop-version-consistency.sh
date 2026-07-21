#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

node - <<'NODE'
const fs = require('fs');
const conf = JSON.parse(fs.readFileSync('desktop/src-tauri/tauri.conf.json', 'utf8'));
const pkg = JSON.parse(fs.readFileSync('desktop/package.json', 'utf8'));
const cargo = fs.readFileSync('desktop/src-tauri/Cargo.toml', 'utf8');
const cargoVersion = cargo.match(/^version = "([^"]+)"/m)?.[1];
const versions = {
  'tauri.conf.json': conf.version,
  'package.json': pkg.version,
  'Cargo.toml': cargoVersion,
};
const unique = new Set(Object.values(versions));
if (unique.size !== 1 || [...unique].some((value) => typeof value !== 'string' || !value)) {
  console.error(`Desktop version mismatch: ${JSON.stringify(versions)}`);
  process.exit(1);
}
console.log(`Desktop versions agree: ${conf.version}`);
NODE
