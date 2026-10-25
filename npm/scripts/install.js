#!/usr/bin/env node

/**
 * Delvo npm postinstall script
 * Downloads the pre-compiled Go binary from GitHub Releases,
 * verifies its SHA-256 checksum, and unpacks it into the bin directory.
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const crypto = require('crypto');
const zlib = require('zlib');
const { execSync, execFileSync } = require('child_process');

const REPO = 'AgentNex/delvo';
const pkg = require('../package.json');
const VERSION = 'v' + pkg.version;

const binDir = path.join(__dirname, '..', 'bin');
const isWindows = process.platform === 'win32';
const binaryName = isWindows ? 'dvo.exe' : 'dvo';
const destBinaryPath = path.join(binDir, binaryName);

// 1. Detect Platform & Architecture
function detectTarget() {
  const isTermux = Boolean(
    process.platform === 'android' ||
    (process.env.PREFIX && process.env.PREFIX.includes('termux')) ||
    fs.existsSync('/data/data/com.termux')
  );

  let os = '';
  if (isTermux) {
    os = 'android';
  } else if (process.platform === 'linux') {
    os = 'linux';
  } else if (process.platform === 'darwin') {
    os = 'darwin';
  } else if (process.platform === 'win32') {
    os = 'windows';
  } else {
    return null;
  }

  let arch = '';
  if (process.arch === 'x64') {
    arch = 'amd64';
  } else if (process.arch === 'arm64') {
    arch = 'arm64';
  } else {
    return null;
  }

  const supported = [
    'android-arm64',
    'linux-amd64',
    'linux-arm64',
    'darwin-amd64',
    'darwin-arm64',
    'windows-amd64'
  ];

  const target = `${os}-${arch}`;
  if (!supported.includes(target)) {
    return null;
  }

  const format = os === 'windows' ? 'zip' : 'tar.gz';
  return { os, arch, target, format, filename: `delvo-${target}.${format}` };
}

// 2. HTTP Helper with redirect following
function fetchBuffer(url, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'delvo-npm-installer' } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        if (maxRedirects <= 0) {
          return reject(new Error('Too many HTTP redirects following: ' + url));
        }
        return resolve(fetchBuffer(res.headers.location, maxRedirects - 1));
      }

      if (res.statusCode !== 200) {
        return reject(new Error(`Failed to download ${url}: HTTP ${res.statusCode} ${res.statusMessage}`));
      }

      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
      res.on('error', reject);
    }).on('error', reject);
  });
}

// 3. Simple pure JS tar extraction fallback
function extractTar(tarBuffer, targetFilename, destinationPath) {
  let offset = 0;
  while (offset + 512 <= tarBuffer.length) {
    const header = tarBuffer.slice(offset, offset + 512);
    // An empty header block signifies end of archive
    if (header.every((b) => b === 0)) break;

    const name = header.slice(0, 100).toString('utf8').replace(/\0.*$/, '').trim();
    const sizeStr = header.slice(124, 136).toString('utf8').replace(/\0.*$/, '').trim();
    const size = parseInt(sizeStr, 8) || 0;

    offset += 512;
    if (name === targetFilename || name.endsWith('/' + targetFilename)) {
      const fileContent = tarBuffer.slice(offset, offset + size);
      fs.writeFileSync(destinationPath, fileContent);
      return true;
    }

    // Align to 512 byte block boundary
    offset += Math.ceil(size / 512) * 512;
  }
  return false;
}

async function main() {
  const targetInfo = detectTarget();

  if (!targetInfo) {
    console.warn(`\x1b[33m[WARN] Delvo does not offer pre-built binaries for ${process.platform}/${process.arch}.\x1b[0m`);
    console.warn('Fallback: You can install Delvo directly with Go:');
    console.warn('  go install github.com/' + REPO + '/cmd/dvo@latest\n');
    return;
  }

  console.log(`\x1b[34m[INFO] Installing Delvo ${VERSION} for ${targetInfo.target}...\x1b[0m`);

  const archiveUrl = `https://github.com/${REPO}/releases/download/${VERSION}/${targetInfo.filename}`;
  const checksumsUrl = `https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt`;

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  try {
    console.log(`[INFO] Downloading archive: ${archiveUrl}`);
    const archiveBuf = await fetchBuffer(archiveUrl);

    console.log(`[INFO] Downloading checksums: ${checksumsUrl}`);
    let expectedHash = null;
    try {
      const checksumsBuf = await fetchBuffer(checksumsUrl);
      const lines = checksumsBuf.toString('utf8').split('\n');
      for (const line of lines) {
        if (line.includes(targetInfo.filename)) {
          expectedHash = line.trim().split(/\s+/)[0];
          break;
        }
      }
    } catch (csErr) {
      console.warn(`\x1b[33m[WARN] Could not fetch checksums.txt: ${csErr.message}. Skipping checksum validation.\x1b[0m`);
    }

    if (expectedHash) {
      const actualHash = crypto.createHash('sha256').update(archiveBuf).digest('hex');
      if (actualHash.toLowerCase() !== expectedHash.toLowerCase()) {
        throw new Error(
          `Checksum mismatch for ${targetInfo.filename}!\nExpected: ${expectedHash}\nActual:   ${actualHash}\nAborting due to potential file corruption or tampering.`
        );
      }
      console.log(`\x1b[32m[SUCCESS] SHA-256 checksum verified: ${actualHash}\x1b[0m`);
    }

    // Extract binary
    const tempFile = path.join(binDir, targetInfo.filename);
    fs.writeFileSync(tempFile, archiveBuf);

    let extracted = false;
    if (targetInfo.format === 'tar.gz') {
      try {
        // Try system tar first
        execSync(`tar -xzf "${tempFile}" -C "${binDir}" "${binaryName}"`, { stdio: 'ignore' });
        extracted = fs.existsSync(destBinaryPath);
      } catch (_) {}

      // Fallback: gunzip + pure JS ustar extract
      if (!extracted) {
        const uncompressed = zlib.gunzipSync(archiveBuf);
        extracted = extractTar(uncompressed, binaryName, destBinaryPath);
      }
    } else if (targetInfo.format === 'zip') {
      try {
        // On Windows PowerShell
        execSync(`powershell -Command "Expand-Archive -Path '${tempFile}' -DestinationPath '${binDir}' -Force"`, { stdio: 'ignore' });
        extracted = fs.existsSync(destBinaryPath);
      } catch (_) {}
    }

    // Cleanup archive file
    try { fs.unlinkSync(tempFile); } catch (_) {}

    if (!extracted || !fs.existsSync(destBinaryPath)) {
      throw new Error(`Failed to extract '${binaryName}' from ${targetInfo.filename}`);
    }

    if (!isWindows) {
      fs.chmodSync(destBinaryPath, 0o755);
    }

    // Verify binary execution
    try {
      const verOutput = execFileSync(destBinaryPath, ['--version'], { encoding: 'utf8' }).trim();
      console.log(`\x1b[32m[SUCCESS] Delvo installed successfully: ${verOutput}\x1b[0m`);
    } catch (e) {
      console.log(`\x1b[32m[SUCCESS] Delvo binary placed at ${destBinaryPath}\x1b[0m`);
    }
  } catch (err) {
    console.error(`\x1b[31m[ERROR] Delvo postinstall failed: ${err.message}\x1b[0m`);
    console.error('You can download binaries manually from: https://github.com/' + REPO + '/releases');
    // Don't kill npm install process completely on build-from-source environments
  }
}

main().catch((err) => {
  console.error('\x1b[31m[ERROR]\x1b[0m', err);
});
