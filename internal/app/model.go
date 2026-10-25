package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/AgentNex/delvo/internal/config"
	"github.com/AgentNex/delvo/internal/filesystem"
	"github.com/AgentNex/delvo/internal/filter"
	"github.com/AgentNex/delvo/internal/index"
	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/AgentNex/delvo/internal/search"
	"github.com/AgentNex/delvo/internal/ui"
	"github.com/AgentNex/delvo/internal/watcher"
)

// Model is the central Bubble Tea application state.
type Model struct {
	cfg               *config.Config
	idx               *index.Index
	scanner           *scanner.Scanner
	watcher           *watcher.FSNotifyWatcher
	watcherEvents     <-chan watcher.ChangeEvent
	activeTab         int // 0: Browse, 1: Search, 2: DiskUsage, 3: Filters
	currentDir        string
	allEntries        []scanner.Entry
	rawEntries        []scanner.Entry
	displayedEntries  []scanner.Entry
	searchResults     []search.SearchResult
	matchedIndices    map[int][]int
	selected          map[string]struct{}
	cursor            int
	scrollOffset      int
	searchQuery       string
	searchActive      bool
	filter            filter.FilterState
	sort              filter.SortState
	scanProgress      scanner.ScanProgress
	showDeleteConfirm bool
	showDetailsModal  bool
	statusMessage     string
	errorMessage      string
	width             int
	height            int
	theme             ui.Theme
}

// NewModel instantiates the application model.
func NewModel(cfg *config.Config, idx *index.Index) (*Model, error) {
	currentDir := ""
	if len(cfg.Roots) > 0 {
		currentDir = cfg.Roots[0]
	}

	scan := scanner.New(scanner.ScanOptions{
		Roots:          cfg.Roots,
		Workers:        cfg.Workers,
		BatchSize:      250,
		FollowSymlinks: false,
	})

	var w *watcher.FSNotifyWatcher
	var wEvents <-chan watcher.ChangeEvent
	if cfg.WatchEnabled {
		var err error
		w, err = watcher.NewFSNotifyWatcher(time.Duration(cfg.DebounceMs) * time.Millisecond)
		if err == nil {
			for _, r := range cfg.Roots {
				_ = w.AddWatch(r)
			}
			wEvents, _, _ = w.Start(context.Background())
		}
	}

	return &Model{
		cfg:            cfg,
		idx:            idx,
		scanner:        scan,
		watcher:        w,
		watcherEvents:  wEvents,
		activeTab:      0,
		currentDir:     currentDir,
		selected:       make(map[string]struct{}),
		matchedIndices: make(map[int][]int),
		filter:         filter.DefaultFilterState(),
		sort:           filter.DefaultSortState(),
		theme:          ui.NewDarkNavyTheme(),
		width:          80,
		height:         24,
	}, nil
}

// Init sets up initial commands on app launch.
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, StartScanCmd(m.scanner, m.idx))
	if m.currentDir != "" {
		cmds = append(cmds, LoadDirEntriesCmd(m.idx, m.currentDir))
	}
	if m.watcherEvents != nil {
		cmds = append(cmds, ListenWatcherCmd(m.watcherEvents))
	}
	return tea.Batch(cmds...)
}

// Update processes incoming messages and user input.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ScanCompleteMsg:
		if msg.Err == nil && msg.Result != nil {
			m.allEntries = msg.Result.Entries
			m.scanProgress = msg.Result.Progress
			m.statusMessage = fmt.Sprintf("Indexed %d items (%s)", msg.Result.Progress.FilesScanned, scanner.FormatSize(msg.Result.Progress.BytesScanned))
			return m, LoadDirEntriesCmd(m.idx, m.currentDir)
		} else if msg.Err != nil {
			m.errorMessage = fmt.Sprintf("Scan failed: %v", msg.Err)
		}
		return m, nil

	case EntriesLoadedMsg:
		if msg.ParentPath == m.currentDir {
			m.rawEntries = msg.Entries
			m.recomputeDisplayed()
		}
		return m, nil

	case SearchResultsMsg:
		m.searchResults = msg.Results
		m.matchedIndices = make(map[int][]int, len(msg.Results))
		m.displayedEntries = make([]scanner.Entry, len(msg.Results))
		for i, res := range msg.Results {
			m.displayedEntries[i] = res.Entry
			if len(res.MatchedIndices) > 0 {
				m.matchedIndices[i] = res.MatchedIndices
			}
		}
		if m.cursor >= len(m.displayedEntries) {
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case WatcherChangeMsg:
		_ = watcher.ReconcileEvent(context.Background(), m.idx, msg.Event)
		var cmds []tea.Cmd
		if m.watcherEvents != nil {
			cmds = append(cmds, ListenWatcherCmd(m.watcherEvents))
		}
		if filepath.Dir(msg.Event.Path) == m.currentDir || msg.Event.Path == m.currentDir {
			cmds = append(cmds, LoadDirEntriesCmd(m.idx, m.currentDir))
		}
		return m, tea.Batch(cmds...)

	case OpCompleteMsg:
		m.showDeleteConfirm = false
		m.selected = make(map[string]struct{})
		successCount := 0
		for _, r := range msg.Results {
			if r.Success {
				successCount++
			}
		}
		m.statusMessage = fmt.Sprintf("Deleted/moved %d item(s)", successCount)
		return m, LoadDirEntriesCmd(m.idx, m.currentDir)

	case ErrMsg:
		m.errorMessage = msg.Err.Error()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m *Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// 1. Delete Confirmation Modal
	if m.showDeleteConfirm {
		switch keyStr {
		case "t", "T":
			// Move to trash
			paths := m.getSelectedOrCurrentPaths()
			return m, DeleteCmd(paths, m.cfg.Roots, filesystem.DeleteOptions{Permanent: false, TrashDir: m.cfg.TrashDir}, m.idx)
		case "y", "Y":
			// Permanent delete
			paths := m.getSelectedOrCurrentPaths()
			return m, DeleteCmd(paths, m.cfg.Roots, filesystem.DeleteOptions{Permanent: true}, m.idx)
		case "n", "N", "esc", "q":
			m.showDeleteConfirm = false
			return m, nil
		}
		return m, nil
	}

	// 2. Details Modal
	if m.showDetailsModal {
		if keyStr == "esc" || keyStr == "i" || keyStr == "q" {
			m.showDetailsModal = false
			return m, nil
		}
		return m, nil
	}

	// 3. Search Mode Active
	if m.searchActive {
		switch keyStr {
		case "enter":
			m.searchActive = false
			return m, nil
		case "esc":
			m.searchActive = false
			m.searchQuery = ""
			m.recomputeDisplayed()
			return m, nil
		case "backspace":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				return m, SearchCmd(m.allEntries, m.searchQuery, m.currentDir, m.activeTab == 1, m.cfg.CaseSensitive)
			}
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		default:
			// Add rune if printable
			if len(msg.Text) > 0 {
				m.searchQuery += msg.Text
				return m, SearchCmd(m.allEntries, m.searchQuery, m.currentDir, m.activeTab == 1, m.cfg.CaseSensitive)
			}
		}
		return m, nil
	}

	// 4. Normal Browsing Navigation
	switch keyStr {
	case "q", "ctrl+c":
		if m.watcher != nil {
			_ = m.watcher.Close()
		}
		if m.idx != nil {
			_ = m.idx.Close()
		}
		return m, tea.Quit

	case "up", "k":
		m.moveCursor(-1)

	case "down", "j":
		m.moveCursor(1)

	case "enter":
		if len(m.displayedEntries) > 0 && m.cursor < len(m.displayedEntries) {
			target := m.displayedEntries[m.cursor]
			if target.IsDir {
				m.currentDir = target.Path
				m.cursor = 0
				m.scrollOffset = 0
				return m, LoadDirEntriesCmd(m.idx, m.currentDir)
			}
		}

	case "backspace", "h":
		parent := filepath.Dir(m.currentDir)
		if parent != m.currentDir && filesystem.IsInsideRoots(parent, m.cfg.Roots) {
			m.currentDir = parent
			m.cursor = 0
			m.scrollOffset = 0
			return m, LoadDirEntriesCmd(m.idx, m.currentDir)
		}

	case " ":
		// Toggle selection
		if len(m.displayedEntries) > 0 && m.cursor < len(m.displayedEntries) {
			p := m.displayedEntries[m.cursor].Path
			if _, ok := m.selected[p]; ok {
				delete(m.selected, p)
			} else {
				m.selected[p] = struct{}{}
			}
		}

	case "a":
		// Toggle select all visible
		if len(m.selected) == len(m.displayedEntries) {
			m.selected = make(map[string]struct{})
		} else {
			for _, e := range m.displayedEntries {
				m.selected[e.Path] = struct{}{}
			}
		}

	case "c":
		// Clear selection
		m.selected = make(map[string]struct{})

	case "/":
		m.searchActive = true
		m.activeTab = 1
		return m, nil

	case "d":
		// Delete confirmation
		paths := m.getSelectedOrCurrentPaths()
		if len(paths) > 0 {
			m.showDeleteConfirm = true
		}

	case "1":
		m.activeTab = 0
		m.searchActive = false
		m.recomputeDisplayed()

	case "2":
		m.activeTab = 1
		m.searchActive = true

	case "3":
		m.activeTab = 2 // Disk usage tab
		m.sort.Field = filter.SortBySize
		m.sort.Descending = true
		m.recomputeDisplayed()

	case "4":
		m.activeTab = 3 // Filters toggle
		m.filter.ShowHidden = !m.filter.ShowHidden
		m.recomputeDisplayed()

	case "s":
		// Cycle sort
		switch m.sort.Field {
		case filter.SortByName:
			m.sort.Field = filter.SortBySize
			m.sort.Descending = true
		case filter.SortBySize:
			m.sort.Field = filter.SortByModTime
			m.sort.Descending = true
		case filter.SortByModTime:
			m.sort.Field = filter.SortByExt
			m.sort.Descending = false
		default:
			m.sort.Field = filter.SortByName
			m.sort.Descending = false
		}
		m.recomputeDisplayed()

	case "i":
		m.showDetailsModal = !m.showDetailsModal

	case "r":
		m.statusMessage = "Rescanning..."
		return m, StartScanCmd(m.scanner, m.idx)
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.displayedEntries) == 0 {
		m.cursor = 0
		m.scrollOffset = 0
		return
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.displayedEntries) {
		m.cursor = len(m.displayedEntries) - 1
	}

	geom := ui.ComputeLayout(m.width, m.height)
	visibleRows := geom.MainHeight

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visibleRows {
		m.scrollOffset = m.cursor - visibleRows + 1
	}
}

func (m *Model) recomputeDisplayed() {
	filtered := filter.ApplyFilters(m.rawEntries, m.filter)
	filter.SortEntries(filtered, m.sort)
	m.displayedEntries = filtered
	m.matchedIndices = nil
	if m.cursor >= len(m.displayedEntries) {
		m.cursor = 0
		m.scrollOffset = 0
	}
}

func (m *Model) getSelectedOrCurrentPaths() []string {
	if len(m.selected) > 0 {
		paths := make([]string, 0, len(m.selected))
		for p := range m.selected {
			paths = append(paths, p)
		}
		return paths
	}

	if len(m.displayedEntries) > 0 && m.cursor < len(m.displayedEntries) {
		return []string{m.displayedEntries[m.cursor].Path}
	}

	return nil
}

func (m *Model) calculateParentSize() int64 {
	var total int64
	for _, e := range m.displayedEntries {
		total += e.Size
	}
	return total
}

// View constructs the complete interactive terminal UI frame.
func (m *Model) View() tea.View {
	geom := ui.ComputeLayout(m.width, m.height)

	// Top header
	header := ui.RenderHeader(ui.HeaderParams{
		ActiveTab:    m.activeTab,
		SearchQuery:  m.searchQuery,
		SearchActive: m.searchActive,
		MatchCount:   len(m.displayedEntries),
		TotalCount:   len(m.allEntries),
		CurrentPath:  m.currentDir,
		Progress:     m.scanProgress,
		Width:        m.width,
		Theme:        m.theme,
	})

	// Main list
	listStr := ui.RenderFileList(ui.FileListParams{
		Entries:        m.displayedEntries,
		Selected:       m.selected,
		Cursor:         m.cursor,
		Offset:         m.scrollOffset,
		MaxRows:        geom.MainHeight,
		Width:          geom.ListWidth,
		Theme:          m.theme,
		ParentSize:     m.calculateParentSize(),
		MatchedIndices: m.matchedIndices,
		ShowDiskUsage:  m.activeTab == 2,
	})

	var mainArea string
	if !geom.IsNarrow && geom.DetailsWidth > 0 {
		var curEntry *scanner.Entry
		if len(m.displayedEntries) > 0 && m.cursor < len(m.displayedEntries) {
			curEntry = &m.displayedEntries[m.cursor]
		}
		detailsStr := ui.RenderDetailsPanel(ui.DetailsParams{
			Entry:      curEntry,
			ParentSize: m.calculateParentSize(),
			Width:      geom.DetailsWidth,
			Height:     geom.MainHeight,
			Theme:      m.theme,
		})

		// Side by side join
		listLines := strings.Split(listStr, "\n")
		detailLines := strings.Split(detailsStr, "\n")
		var joinedLines []string
		for i := 0; i < geom.MainHeight; i++ {
			ll := ""
			if i < len(listLines) {
				ll = listLines[i]
			}
			dl := ""
			if i < len(detailLines) {
				dl = detailLines[i]
			}
			joinedLines = append(joinedLines, ui.PadRight(ll, geom.ListWidth)+" │ "+dl)
		}
		mainArea = strings.Join(joinedLines, "\n")
	} else {
		mainArea = listStr
	}

	// Status & Footer
	filterBar := ui.RenderFilterStatusBar(ui.FilterPanelParams{
		Filter: m.filter,
		Sort:   m.sort,
		Width:  m.width,
		Theme:  m.theme,
	})

	helpBar := ui.RenderHelpBar(m.width, m.searchActive, m.theme)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteByte('\n')
	sb.WriteString(mainArea)
	sb.WriteByte('\n')
	sb.WriteString(filterBar)
	sb.WriteByte('\n')
	sb.WriteString(helpBar)

	// Overlays / Modals
	rendered := sb.String()
	if m.showDeleteConfirm {
		paths := m.getSelectedOrCurrentPaths()
		var totalSize int64
		for _, e := range m.displayedEntries {
			for _, p := range paths {
				if e.Path == p {
					totalSize += e.Size
				}
			}
		}
		confirmBox := ui.RenderDeleteConfirm(ui.DeleteConfirmParams{
			Paths:     paths,
			TotalSize: totalSize,
			ItemCount: len(paths),
			Width:     m.width,
			Height:    m.height,
			Theme:     m.theme,
		})
		rendered = confirmBox + "\n\n" + rendered
	} else if m.showDetailsModal && len(m.displayedEntries) > 0 && m.cursor < len(m.displayedEntries) {
		modal := ui.RenderDetailsPanel(ui.DetailsParams{
			Entry:      &m.displayedEntries[m.cursor],
			ParentSize: m.calculateParentSize(),
			Width:      m.width - 4,
			Height:     10,
			Theme:      m.theme,
		})
		rendered = modal + "\n\n" + rendered
	}

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}
