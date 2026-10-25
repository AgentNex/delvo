#!/usr/bin/env node

/**
 * Delvo CLI npm launcher.
 * Dispatches directly to the downloaded native Go binary.
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const isWindows = process.platform === 'win32';
const binName = isWindows ? 'dvo.exe' : 'dvo';
const binaryPath = path.join(__dirname, binName);

if (!fs.existsSync(binaryPath)) {
  console.error('\x1b[31m[ERROR] Delvo native binary not found at:\x1b[0m ' + binaryPath);
  console.error('Attempting on-demand installation...');

  try {
    const installScript = path.join(__dirname, '..', 'scripts', 'install.js');
    require('child_process').execFileSync(process.execPath, [installScript], {
      stdio: 'inherit'
    });
  } catch (err) {
    console.error('\x1b[31m[ERROR] Failed to download Delvo native binary.\x1b[0m');
    console.error('You can install manually from: https://github.com/AgentNex/delvo/releases');
    console.error('Or compile from source with Go: go install github.com/AgentNex/delvo/cmd/dvo@latest');
    process.exit(1);
  }
}

// Ensure binary is executable
if (!isWindows) {
  try {
    fs.chmodSync(binaryPath, 0o755);
  } catch (_) {}
}

// Spawn the native Go binary with stdio inheritance for Bubbletea TUI
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

child.on('error', (err) => {
  console.error('\x1b[31m[ERROR] Failed to start Delvo binary:\x1b[0m', err.message);
  process.exit(1);
});

child.on('close', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 0);
  }
});

// Forward signals to child process
const signals = ['SIGINT', 'SIGTERM', 'SIGHUP', 'SIGQUIT'];
signals.forEach((sig) => {
  process.on(sig, () => {
    if (!child.killed) {
      try {
        child.kill(sig);
      } catch (_) {}
    }
  });
});
