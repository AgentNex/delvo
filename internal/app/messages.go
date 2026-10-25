package app

import (
	"github.com/AgentNex/delvo/internal/filesystem"
	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/AgentNex/delvo/internal/search"
	"github.com/AgentNex/delvo/internal/watcher"
)

// ScanProgressMsg is received periodically while a scan is running.
type ScanProgressMsg struct {
	Progress scanner.ScanProgress
}

// ScanCompleteMsg signals that the initial or requested scan has completed.
type ScanCompleteMsg struct {
	Result *scanner.ScanResult
	Err    error
}

// EntriesLoadedMsg delivers entries loaded from the database for the active view.
type EntriesLoadedMsg struct {
	ParentPath string
	Entries    []scanner.Entry
}

// SearchResultsMsg carries ranked results from a search query.
type SearchResultsMsg struct {
	Query   string
	Results []search.SearchResult
}

// WatcherChangeMsg delivers a filesystem notification from the ChangeSource.
type WatcherChangeMsg struct {
	Event watcher.ChangeEvent
}

// OpCompleteMsg notifies that file operations (delete, trash, move) finished.
type OpCompleteMsg struct {
	Results []filesystem.DeleteResult
}

// StatusMsg updates the temporary status banner.
type StatusMsg struct {
	Message string
}

// ErrMsg reports non-fatal operational errors to the UI.
type ErrMsg struct {
	Err error
}
