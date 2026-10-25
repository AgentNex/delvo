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

## Installation

### In Termux (Android)

```bash
# 1. Ensure Go and Git are installed
pkg update && pkg install golang git

# 2. Grant shared storage permissions (recommended)
termux-setup-storage

# 3. Clone and build Delvo
git clone https://github.com/AgentNex/delvo.git
cd delvo
go build -o $PREFIX/bin/dvo ./cmd/dvo

# 4. Launch Delvo
dvo
```

### From Source (Any Linux / macOS / BSD)

```bash
git clone https://github.com/AgentNex/delvo.git
cd delvo
go build -o dvo ./cmd/dvo
sudo mv dvo /usr/local/bin/
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
