package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/AgentNex/delvo/internal/filesystem"
	"github.com/AgentNex/delvo/internal/index"
	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/AgentNex/delvo/internal/search"
	"github.com/AgentNex/delvo/internal/watcher"
)

// StartScanCmd initiates parallel scanning in a background goroutine.
func StartScanCmd(s *scanner.Scanner, idx *index.Index) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		res, err := s.Scan(ctx, nil, func(batch []scanner.Entry) error {
			return idx.BatchUpsert(ctx, batch)
		})
		if err == nil && res != nil {
			_ = idx.UpdateDirSizes(ctx, res.DirSizes)
		}
		return ScanCompleteMsg{Result: res, Err: err}
	}
}

// LoadDirEntriesCmd queries the local SQLite index for child entries of a folder.
func LoadDirEntriesCmd(idx *index.Index, parentPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		entries, err := idx.GetEntriesInDir(ctx, parentPath)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return EntriesLoadedMsg{ParentPath: parentPath, Entries: entries}
	}
}

// SearchCmd runs the fuzzy search algorithm in background.
func SearchCmd(allEntries []scanner.Entry, query string, curDir string, scopeAll bool, caseSensitive bool) tea.Cmd {
	return func() tea.Msg {
		opts := search.SearchOptions{
			Query:         query,
			CaseSensitive: caseSensitive,
			ScopeAllRoots: scopeAll,
			CurrentDir:    curDir,
		}
		results := search.Search(allEntries, opts)
		return SearchResultsMsg{Query: query, Results: results}
	}
}

// DeleteCmd performs confirmed deletions or trash moves and cleans up the index.
func DeleteCmd(paths []string, roots []string, opts filesystem.DeleteOptions, idx *index.Index) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		results := filesystem.DeleteItems(paths, roots, opts)
		for _, res := range results {
			if res.Success {
				_ = idx.DeleteEntry(ctx, res.Path)
			}
		}

		return OpCompleteMsg{Results: results}
	}
}

// ListenWatcherCmd awaits the next debounced filesystem change event.
func ListenWatcherCmd(eventChan <-chan watcher.ChangeEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-eventChan
		if !ok {
			return nil
		}
		return WatcherChangeMsg{Event: ev}
	}
}
