package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FSNotifyWatcher implements ChangeSource using github.com/fsnotify/fsnotify.
type FSNotifyWatcher struct {
	watcher   *fsnotify.Watcher
	watched   map[string]bool
	mu        sync.Mutex
	coalescer *Coalescer
	outEvents chan ChangeEvent
	outErrors chan error
}

// NewFSNotifyWatcher constructs a new FSNotifyWatcher with debounce.
func NewFSNotifyWatcher(debounce time.Duration) (*FSNotifyWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	return &FSNotifyWatcher{
		watcher:   w,
		watched:   make(map[string]bool),
		coalescer: NewCoalescer(debounce),
		outEvents: make(chan ChangeEvent, 100),
		outErrors: make(chan error, 50),
	}, nil
}

// Start launches background goroutines translating fsnotify events into ChangeEvents.
func (f *FSNotifyWatcher) Start(ctx context.Context) (<-chan ChangeEvent, <-chan error, error) {
	debouncedChan := f.coalescer.Run(ctx)

	// Ingestion goroutine: read from fsnotify.Watcher and push to coalescer
	go func() {
		defer close(f.outErrors)

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-f.watcher.Errors:
				if !ok {
					return
				}
				select {
				case f.outErrors <- err:
				default:
				}
			case ev, ok := <-f.watcher.Events:
				if !ok {
					return
				}

				var changeType ChangeType
				switch {
				case ev.Has(fsnotify.Create):
					changeType = ChangeCreate
					// If new directory was created, try to add watch dynamically
					if fi, err := os.Lstat(ev.Name); err == nil && fi.IsDir() {
						_ = f.AddWatch(ev.Name)
					}
				case ev.Has(fsnotify.Write):
					changeType = ChangeWrite
				case ev.Has(fsnotify.Remove):
					changeType = ChangeRemove
					_ = f.RemoveWatch(ev.Name)
				case ev.Has(fsnotify.Rename):
					changeType = ChangeRename
					_ = f.RemoveWatch(ev.Name)
				default:
					continue
				}

				f.coalescer.Push(ChangeEvent{
					Path:      filepath.Clean(ev.Name),
					Type:      changeType,
					Timestamp: time.Now(),
				})
			}
		}
	}()

	// Output goroutine
	go func() {
		defer close(f.outEvents)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-debouncedChan:
				if !ok {
					return
				}
				select {
				case f.outEvents <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return f.outEvents, f.outErrors, nil
}

// AddWatch registers a path with the underlying fsnotify watcher.
func (f *FSNotifyWatcher) AddWatch(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	clean := filepath.Clean(path)
	if f.watched[clean] {
		return nil
	}

	if err := f.watcher.Add(clean); err != nil {
		return fmt.Errorf("fsnotify add watch %s: %w", clean, err)
	}

	f.watched[clean] = true
	return nil
}

// RemoveWatch unregisters a path.
func (f *FSNotifyWatcher) RemoveWatch(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	clean := filepath.Clean(path)
	if !f.watched[clean] {
		return nil
	}

	_ = f.watcher.Remove(clean)
	delete(f.watched, clean)
	return nil
}

// Close terminates the watcher and frees system resources.
func (f *FSNotifyWatcher) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watcher.Close()
}
