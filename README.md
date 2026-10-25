# Delvo

> High-performance terminal file manager and disk-usage explorer tailored for Termux on Android and modern Linux terminals. Built end-to-end in Go.

```text
  ██████╗ ███████╗██╗     ██╗   ██╗ ██████╗ 
  ██╔══██╗██╔════╝██║     ██║   ██║██╔═══██╗
  ██║  ██║█████╗  ██║     ██║   ██║██║   ██║
  ██║  ██║██╔══╝  ██║     ╚██╗ ██╔╝██║   ██║
  ██████╔╝███████╗███████╗ ╚████╔╝ ╚██████╔╝
  ╚═════╝ ╚══════╝╚══════╝  ╚═══╝   ╚═════╝ 
```

**Delvo** combines interactive `gdu`/`ncdu`-style visual disk-usage exploration, lightning-fast fuzzy search, metadata filtering and sorting, multi-selection, and safe bulk file operations into a polished, responsive terminal interface.

---

## Key Features

- **Terminal UI Tailored for Phones & Scalable to Desktops:** Built with a dark navy (`#0b0f19`), bright cyan (`#00f0ff`), and mint/teal theme. Automatically adapts from narrow portrait Termux screens to wide multi-pane desktop layouts.
- **Bounded Parallel Scanner:** Recursive filesystem traversal using a worker pool that stays responsive during scans and gracefully handles permission boundaries without crashing.
- **Bottom-Up Disk Usage Exploration:** Computes recursive directory totals bottom-up with visual disk-usage bars (`[██████░░░░░░] 52%`).
- **Local SQLite Metadata Index:** Backed by `modernc.org/sqlite` (pure Go, CGo-free). Fast persistent caching, transactional batch updates, and zero server or cloud dependencies.
- **Sub-Millisecond Fuzzy & Glob Search:** Powered by `github.com/sahilm/fuzzy` with ranking bonuses for exact name matches, prefixes, and extensions.
- **Multi-Selection & Safety Protections:**
  - Multi-select with aggregate size counters.
  - Confirmation modals before deletion.
  - Overlapping path resolution (avoids double deletion if parent and child are both selected).
  - Strict symlink protection (removing a symlink never follows or deletes target contents).
  - Safe reversible trash workflow (`~/.local/share/delvo/trash/`).
- **Filesystem Change Monitoring:** Integrates `github.com/fsnotify/fsnotify` with event debouncing and fallback polling for directory reconciliation.

---

## Architecture & Dependencies

Delvo adheres strictly to a Go-only stack with minimal, well-justified direct dependencies:

| Dependency | Purpose |
|------------|---------|
| `charm.land/bubbletea/v2` | Event loop and model-update-view TUI architecture |
| `charm.land/lipgloss/v2` | Monospace layouts, color styling, borders, and visual theme |
| `github.com/sahilm/fuzzy` | Initial fuzzy matching algorithm for filenames and paths |
| `github.com/fsnotify/fsnotify` | Filesystem mutation monitoring on supported targets |
| `modernc.org/sqlite` | Pure Go embedded SQLite engine without CGo |

---

## Installation & Distribution

Delvo offers multiple verified distribution channels tailored for Unix, Termux (Android), macOS, and Windows.

### Platform Support Matrix

| Platform | Architecture | Binary Archive | Verified Channels |
| :--- | :--- | :--- | :--- |
| **Android / Termux** | `arm64` (`aarch64`) | `delvo-android-arm64.tar.gz` | curl, npm, GitHub Releases, Source |
| **Linux** | `amd64` (`x86_64`) | `delvo-linux-amd64.tar.gz` | curl, npm, GitHub Releases, Source |
| **Linux** | `arm64` (`aarch64`) | `delvo-linux-arm64.tar.gz` | curl, npm, GitHub Releases, Source |
| **macOS** | Apple Silicon (`arm64`) | `delvo-darwin-arm64.tar.gz` | curl, npm, GitHub Releases, Source |
| **macOS** | Intel (`amd64`) | `delvo-darwin-amd64.tar.gz` | curl, npm, GitHub Releases, Source |
| **Windows** | `x64` (`amd64`) | `delvo-windows-amd64.zip` | npm, GitHub Releases, Source |

---

### Channel 1: Automated `curl` Installer (Recommended for Termux & Linux/macOS)

The standalone shell installer automatically detects your operating system and CPU architecture, downloads the matching verified release asset from GitHub, validates its SHA-256 checksum against `checksums.txt`, and installs `dvo` directly into `$PREFIX/bin` (on Termux) or `/usr/local/bin` / `~/.local/bin` (on Linux/macOS).

```bash
# Automated install via curl
curl -fsSL https://raw.githubusercontent.com/AgentNex/delvo/main/install.sh | bash
```

#### Custom Installation Directory or Version
```bash
# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/AgentNex/delvo/main/install.sh | bash -s -- --install-dir ~/.local/bin

# Install specific version
curl -fsSL https://raw.githubusercontent.com/AgentNex/delvo/main/install.sh | bash -s -- --version v0.1.0
```

#### Uninstalling via curl installer
```bash
# Clean uninstallation
curl -fsSL https://raw.githubusercontent.com/AgentNex/delvo/main/install.sh | bash -s -- --uninstall
```

---

### Channel 2: npm Package Manager

Delvo provides a minimal npm wrapper ([`@agentnex/delvo`](https://www.npmjs.com/package/@agentnex/delvo)) that downloads the native Go binary during post-installation and registers the `dvo` command. Delvo itself remains 100% Go and runs natively without Node.js execution overhead.

**Prerequisites:** Node.js (>= 16.0.0) and npm.

```bash
# Install globally
npm install -g @agentnex/delvo

# Or run ephemerally with npx
npx @agentnex/delvo
```

#### Uninstalling via npm
```bash
npm uninstall -g @agentnex/delvo
```

---

### Channel 3: GitHub Releases (Manual Download)

Pre-compiled standalone binaries with cryptographic SHA-256 checksums are published on the [Delvo Releases page](https://github.com/AgentNex/delvo/releases).

1. Download the archive for your OS/architecture and `checksums.txt`.
2. Verify SHA-256 checksum:
   ```bash
   sha256sum --check --ignore-missing checksums.txt
   ```
3. Extract and place in your `$PATH`:
   ```bash
   tar -xzf delvo-linux-amd64.tar.gz
   chmod +x dvo
   mv dvo ~/.local/bin/
   ```

---

### Channel 4: Build from Source (Fallback for Any Go-supported Platform)

If your architecture or OS is not covered by precompiled binaries, you can install directly via Go:

**Prerequisites:** Go 1.22+ and Git.

```bash
# Direct install via Go module toolchain
go install github.com/AgentNex/delvo/cmd/dvo@latest

# Or clone and compile locally
git clone https://github.com/AgentNex/delvo.git
cd delvo
go build -ldflags "-s -w" -o dvo ./cmd/dvo
mv dvo $PREFIX/bin/   # On Termux
# or: sudo mv dvo /usr/local/bin/  # On Linux/macOS
```

---

## Command Line Usage

```bash
# Launch interactive TUI
dvo

# Run a non-interactive parallel disk scan on current directory
dvo scan .

# Run a non-interactive scan on specific folders with 8 workers
dvo scan --workers 8 ~/storage/shared

# Display configured roots and SQLite database path
dvo roots

# Rebuild the local SQLite cache index
dvo rebuild

# Display version
dvo version

# Display help
dvo help
```

---

## Interactive Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` | Open directory / Navigate |
| `Backspace` / `h` | Go up to parent directory |
| `Space` | Toggle selection on highlighted item |
| `a` | Select all / Invert selection |
| `c` | Clear all selections |
| `/` | Activate fuzzy search |
| `1` | Switch to **Browse** tab |
| `2` | Switch to **Search** tab |
| `3` | Switch to **Disk Usage** tab (sorted by size with usage bars) |
| `4` | Switch to **Filters** tab (toggle hidden files) |
| `s` | Cycle sort attribute (Name → Size → Date → Ext) |
| `i` | Toggle details inspection panel |
| `d` | Safe delete / Trash modal |
| `r` | Rescan directory |
| `q` / `Esc` | Quit Delvo |

---

## Configuration

Configuration is loaded from `~/.config/delvo/config.json`:

```json
{
  "roots": [
    "/data/data/com.termux/files/home",
    "/data/data/com.termux/files/home/storage/shared"
  ],
  "db_path": "/data/data/com.termux/files/home/.local/share/delvo/delvo.db",
  "trash_dir": "/data/data/com.termux/files/home/.local/share/delvo/trash",
  "workers": 8,
  "debounce_ms": 80,
  "watch_enabled": true,
  "show_hidden": false,
  "case_sensitive": false,
  "theme": "dark-navy"
}
```

---

## Safety & Destructive Operations

1. **Explicit Confirmation**: Pressing `d` opens a modal showing affected item count, total known size, and preview list.
2. **Reversible Trash by Default**: Delvo offers `[t]` to safely move files into `~/.local/share/delvo/trash/` with timestamped preservation, alongside `[y]` for permanent removal.
3. **Overlapping Path Resolution**: If `/foo` and `/foo/bar.txt` are both selected, Delvo prunes `/foo/bar.txt` to eliminate duplicate operations and errors.
4. **Symlink Boundary Protection**: Deleting a symlink deletes only the link itself, never traversing or deleting files in the target directory.
5. **No Shell Execution**: File operations never shell out to external commands like `rm -rf`.

---

## Known Platform Limitations

- **Android App-Private Storage**: Delvo cannot access other Android apps' private storage in `/data/data/<other-app>` or restricted Android 11+ system directories.
- **Filesystem Notifications on Android**: While `fsnotify` inotify events operate properly in user-writable paths, Android shared storage mounts (e.g. SDCard/FUSE) may throttle or miss inotify signals. Delvo provides targeted rescan (`r`) and fallback reconciliation.
- **Access Times**: Access timestamps (`atime`) are filesystem and mount dependent (often disabled via `noatime` on Android flash storage).
- **Go Race Detector**: The `-race` flag is currently unsupported upstream by Go on `android/arm64`.

---

## Uninstall

```bash
# Remove binary
rm -f $PREFIX/bin/dvo

# Remove cache and config (optional)
rm -rf ~/.local/share/delvo ~/.config/delvo
```

---

## License

MIT License © 2026 AgentNex
