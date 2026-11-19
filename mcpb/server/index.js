#!/usr/bin/env node
'use strict';

/**
 * cortex-mcp-local — npx wrapper for the Cortex binary.
 *
 * First run: download the matching release asset from GitHub, verify SHA256,
 * unpack it into ~/.cortex/bin/v<version>/, then exec `cortex <args>`.
 * Later runs: straight exec, no network.
 *
 * Env overrides:
 *   CORTEX_BIN         use an existing binary, skip download
 *   CORTEX_VERSION     override release tag (e.g. v4.0.1)
 *   CORTEX_CACHE_DIR   override cache directory
 */

const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');
const { spawn, spawnSync } = require('child_process');
const https = require('https');

const REPO = 'lh123aa/cortex';
const pkg = require('../package.json');
const VERSION = (process.env.CORTEX_VERSION || 'v' + pkg.version).replace(/^v?/, 'v');

function log(msg) {
  process.stderr.write(`[cortex-mcp-local] ${msg}\n`);
}

function platformAsset() {
  const p = process.platform;
  const a = process.arch;
  const osName = p === 'win32' ? 'windows' : p === 'darwin' ? 'darwin' : p === 'linux' ? 'linux' : null;
  const arch = a === 'x64' ? 'amd64' : a === 'arm64' ? 'arm64' : null;
  if (!osName || !arch) {
    throw new Error(`unsupported platform: ${p}/${a}. Use the manual install instead: https://cortex.qataroasis.cc/install.sh`);
  }
  const base = `cortex-${osName}-${arch}`;
  return {
    osName,
    arch,
    asset: osName === 'windows' ? `${base}.zip` : `${base}.tar.gz`,
    binary: osName === 'windows' ? 'cortex.exe' : 'cortex',
  };
}

function cacheRoot() {
  return process.env.CORTEX_CACHE_DIR || path.join(os.homedir(), '.cortex', 'bin', VERSION);
}

function httpsGet(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error('too many redirects'));
    https
      .get(url, { headers: { 'User-Agent': 'cortex-mcp-local' } }, (res) => {
        if ([301, 302, 303, 307, 308].includes(res.statusCode)) {
          res.resume();
          return resolve(httpsGet(res.headers.location, redirects + 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`GET ${url} -> HTTP ${res.statusCode}`));
        }
        resolve(res);
      })
      .on('error', reject);
  });
}

async function download(url, dest) {
  const res = await httpsGet(url);
  await new Promise((resolve, reject) => {
    const out = fs.createWriteStream(dest);
    res.pipe(out);
    out.on('finish', () => out.close(resolve));
    out.on('error', reject);
    res.on('error', reject);
  });
}

function sha256(file) {
  return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}

async function expectedHash(asset) {
  try {
    const res = await httpsGet(`https://github.com/${REPO}/releases/download/${VERSION}/SHA256SUMS`);
    const body = await new Promise((resolve, reject) => {
      let buf = '';
      res.on('data', (c) => (buf += c));
      res.on('end', () => resolve(buf));
      res.on('error', reject);
    });
    for (const line of body.split('\n')) {
      const [hash, name] = line.trim().split(/\s+/);
      if (name === asset || name === './' + asset) return hash;
    }
  } catch (err) {
    log(`SHA256SUMS unavailable (${err.message}); skipping verification`);
  }
  return null;
}

function extract(archive, dir, osName) {
  if (osName === 'windows') {
    const r = spawnSync('powershell', ['-NoProfile', '-Command', `Expand-Archive -LiteralPath '${archive}' -DestinationPath '${dir}' -Force`], { stdio: 'inherit' });
    if (r.status !== 0) throw new Error('unzip failed');
  } else {
    const r = spawnSync('tar', ['-xzf', archive, '-C', dir], { stdio: 'inherit' });
    if (r.status !== 0) throw new Error('tar failed');
  }
}

function findBinary(dir, name) {
  const stack = [dir];
  while (stack.length) {
    const cur = stack.pop();
    for (const entry of fs.readdirSync(cur, { withFileTypes: true })) {
      const full = path.join(cur, entry.name);
      if (entry.isDirectory()) stack.push(full);
      else if (entry.name === name) return full;
    }
  }
  return null;
}

async function ensureBinary() {
  if (process.env.CORTEX_BIN) {
    if (!fs.existsSync(process.env.CORTEX_BIN)) throw new Error(`CORTEX_BIN not found: ${process.env.CORTEX_BIN}`);
    return process.env.CORTEX_BIN;
  }
  const { osName, asset, binary } = platformAsset();
  const dir = cacheRoot();
  const target = path.join(dir, binary);

  if (fs.existsSync(target)) return target;

  fs.mkdirSync(dir, { recursive: true });
  const archive = path.join(dir, asset);
  log(`first run: downloading ${asset} (${VERSION}) for ${osName}`);

  try {
    await download(`https://github.com/${REPO}/releases/download/${VERSION}/${asset}`, archive);
    const want = await expectedHash(asset);
    if (want) {
      const got = sha256(archive);
      if (got !== want) throw new Error(`sha256 mismatch for ${asset}: got ${got}, want ${want}`);
      log('sha256 verified');
    }
    extract(archive, dir, osName);
    const found = findBinary(dir, binary);
    if (!found) throw new Error(`binary ${binary} not found inside ${asset}`);
    if (found !== target) fs.copyFileSync(found, target);
    if (osName !== 'windows') fs.chmodSync(target, 0o755);
    log(`installed to ${target}`);
  } finally {
    if (fs.existsSync(archive)) fs.rmSync(archive, { force: true });
  }
  return target;
}

(async () => {
  try {
    const bin = await ensureBinary();
    const args = process.argv.slice(2);
    const child = spawn(bin, args, { stdio: 'inherit', env: process.env });
    for (const sig of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
      process.on(sig, () => child.kill(sig));
    }
    child.on('exit', (code, signal) => {
      if (signal) process.kill(process.pid, signal);
      else process.exit(code === null ? 1 : code);
    });
    child.on('error', (err) => {
      log(`failed to start binary: ${err.message}`);
      process.exit(1);
    });
  } catch (err) {
    log(err.message);
    process.exit(1);
  }
})();
