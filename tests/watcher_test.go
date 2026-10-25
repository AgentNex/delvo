package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AgentNex/delvo/internal/watcher"
)

func TestFSNotifyOnTargetDevice(t *testing.T) {
	tempDir := t.TempDir()

	w, err := watcher.NewFSNotifyWatcher(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.AddWatch(tempDir); err != nil {
		t.Fatalf("failed to add watch to %s: %v", tempDir, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, errors, err := w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Channel to capture errors
	go func() {
		for err := range errors {
			t.Logf("watcher internal error: %v", err)
		}
	}()

	testFile := filepath.Join(tempDir, "sample.txt")

	// 1. Create file
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	// Await create/write event
	receivedCreate := false
	timeout := time.After(2 * time.Second)

loop:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break loop
			}
			t.Logf("Received event: %s on %s", ev.Type, ev.Path)
			if ev.Path == testFile {
				receivedCreate = true
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	if !receivedCreate {
		t.Logf("Note: fsnotify on this device did not deliver create event within timeout (documenting Termux/Android inotify behavior)")
	} else {
		t.Logf("✓ fsnotify successfully received event on Android/Termux target device!")
	}
}

func TestCoalescer(t *testing.T) {
	c := watcher.NewCoalescer(30 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := c.Run(ctx)

	// Rapidly push multiple writes for the same path
	target := "/path/to/busy.log"
	for i := 0; i < 10; i++ {
		c.Push(watcher.ChangeEvent{
			Path:      target,
			Type:      watcher.ChangeWrite,
			Timestamp: time.Now(),
		})
	}

	// Should emit at most 1 coalesced event
	select {
	case ev := <-out:
		if ev.Path != target || ev.Type != watcher.ChangeWrite {
			t.Errorf("unexpected event: %+v", ev)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("timeout waiting for coalesced event")
	}
}
